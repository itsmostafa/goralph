package pageindex

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PDFProcessor handles PDF document processing using PageIndex algorithms.
type PDFProcessor struct {
	llm      LLMProvider
	config   *Config
	detector *TOCDetector
	verifier *Verifier
}

// NewPDFProcessor creates a new PDF processor with the given LLM provider.
func NewPDFProcessor(llm LLMProvider, config *Config) *PDFProcessor {
	if config == nil {
		config = DefaultConfig()
	}
	return &PDFProcessor{
		llm:      llm,
		config:   config,
		detector: NewTOCDetector(llm, config),
		verifier: NewVerifier(llm, config),
	}
}

// ProcessPDF processes a PDF file and returns a structured Document.
// This is the main entry point for PDF processing.
func (p *PDFProcessor) ProcessPDF(ctx context.Context, pdfPath string) (*Document, error) {
	// Extract pages from PDF
	pages, err := p.extractPages(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("extracting pages: %w", err)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages extracted from PDF")
	}

	// Detect TOC
	tocResult, err := p.detector.DetectTOC(ctx, pages)
	if err != nil {
		return nil, fmt.Errorf("detecting TOC: %w", err)
	}

	// Determine processing mode
	mode := DetermineProcessingMode(tocResult)

	var tree []*TreeNode
	switch mode {
	case "toc_with_pages":
		tree, err = p.processTOCWithPages(ctx, tocResult, pages)
	case "toc_without_pages":
		tree, err = p.processTOCWithoutPages(ctx, tocResult, pages)
	case "no_toc":
		tree, err = p.processNoTOC(ctx, pages)
	default:
		return nil, fmt.Errorf("unknown processing mode: %s", mode)
	}

	if err != nil {
		return nil, fmt.Errorf("processing with mode %s: %w", mode, err)
	}

	// Post-process tree
	if p.config.IfAddNodeID {
		WriteNodeIDs(tree)
	}

	// Add text to nodes if requested
	if p.config.IfAddNodeText {
		p.addNodeTexts(tree, pages)
	}

	// Generate summaries if requested
	if p.config.IfAddNodeSummary {
		if err := p.generateSummaries(ctx, tree); err != nil {
			return nil, fmt.Errorf("generating summaries: %w", err)
		}
	}

	// Generate document description if requested
	var description string
	if p.config.IfAddDocDescription {
		description, err = generateDocDescription(ctx, tree, p.llm)
		if err != nil {
			return nil, fmt.Errorf("generating description: %w", err)
		}
	}

	// Build document
	docName := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))

	return &Document{
		Name:        docName,
		Description: description,
		Structure:   tree,
	}, nil
}

// processTOCWithPages processes PDFs that have a TOC with page numbers.
func (p *PDFProcessor) processTOCWithPages(ctx context.Context, tocResult *TOCCheckResult, pages []PageContent) ([]*TreeNode, error) {
	// Transform TOC to structured format
	items, err := p.detector.TransformTOC(ctx, tocResult.TOCContent)
	if err != nil {
		return nil, err
	}

	// Map logical page numbers to physical indices
	items = MapPageNumbersToPhysical(items, tocResult.TOCPageList, len(pages))

	// Validate and truncate physical indices
	items = ValidateAndTruncatePhysicalIndices(items, len(pages), 1)

	// Add preface if document content starts before first TOC entry
	items = AddPrefaceIfNeeded(items)

	// Verify TOC entries against actual pages
	report, err := p.verifier.VerifyTOCEntries(ctx, items, pages, 10)
	if err != nil {
		return nil, err
	}

	// Determine action based on verification
	action := EvaluateVerification(report, nil)

	switch action {
	case "fix":
		// Attempt to fix incorrect entries
		items, _, err = p.verifier.FixIncorrectTOCWithRetries(ctx, items, pages, 3)
		if err != nil {
			return nil, err
		}
	case "fallback":
		// TOC is too inaccurate, generate from scratch
		return p.processNoTOC(ctx, pages)
	}

	// Check section starts
	items, err = p.detector.CheckSectionStartsConcurrently(ctx, items, pages, 5)
	if err != nil {
		return nil, err
	}

	// Convert to tree
	return PostProcessTOC(items, len(pages)-1), nil
}

// processTOCWithoutPages processes PDFs that have a TOC but no page numbers.
func (p *PDFProcessor) processTOCWithoutPages(ctx context.Context, tocResult *TOCCheckResult, pages []PageContent) ([]*TreeNode, error) {
	// Transform TOC to structured format
	items, err := p.detector.TransformTOC(ctx, tocResult.TOCContent)
	if err != nil {
		return nil, err
	}

	// Find each section in the document
	items, err = p.findSectionPages(ctx, items, pages)
	if err != nil {
		return nil, err
	}

	// Check section starts
	items, err = p.detector.CheckSectionStartsConcurrently(ctx, items, pages, 5)
	if err != nil {
		return nil, err
	}

	// Convert to tree
	return PostProcessTOC(items, len(pages)-1), nil
}

// processNoTOC processes PDFs that don't have an explicit TOC.
func (p *PDFProcessor) processNoTOC(ctx context.Context, pages []PageContent) ([]*TreeNode, error) {
	// Generate TOC from document structure
	items, err := p.detector.GenerateTOC(ctx, pages)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		// Fallback: create single node for entire document
		return []*TreeNode{{
			Title:    "Document",
			StartIdx: 0,
			EndIdx:   len(pages) - 1,
		}}, nil
	}

	// Check section starts
	items, err = p.detector.CheckSectionStartsConcurrently(ctx, items, pages, 5)
	if err != nil {
		return nil, err
	}

	// Convert to tree
	return PostProcessTOC(items, len(pages)-1), nil
}

// findSectionPages searches for each section title in the document pages.
func (p *PDFProcessor) findSectionPages(ctx context.Context, items []TOCItem, pages []PageContent) ([]TOCItem, error) {
	result := make([]TOCItem, len(items))
	copy(result, items)

	// Search sequentially to maintain order
	currentPage := 0
	for i := range result {
		title := result[i].Title

		// Search from current position forward
		found := false
		for j := currentPage; j < len(pages) && !found; j++ {
			if containsFuzzy(pages[j].Text, title) {
				result[i].PhysicalIndex = &j
				currentPage = j
				found = true
			}
		}

		if !found {
			// Use LLM to find the section
			pageNum, err := p.verifier.searchForEntry(ctx, title, pages, currentPage, 10)
			if err == nil && pageNum >= 0 {
				result[i].PhysicalIndex = &pageNum
				if pageNum > currentPage {
					currentPage = pageNum
				}
			}
		}
	}

	return result, nil
}

// addNodeTexts populates the Text field for all nodes based on page ranges.
func (p *PDFProcessor) addNodeTexts(nodes []*TreeNode, pages []PageContent) {
	for _, node := range FlattenTree(nodes) {
		if node.StartIdx < 0 || node.EndIdx < node.StartIdx {
			continue
		}

		start := node.StartIdx
		end := node.EndIdx
		if start >= len(pages) {
			continue
		}
		if end >= len(pages) {
			end = len(pages) - 1
		}

		var text strings.Builder
		for i := start; i <= end; i++ {
			if text.Len() > 0 {
				text.WriteString("\n\n")
			}
			text.WriteString(pages[i].Text)
		}

		node.Text = text.String()
	}
}

// generateSummaries generates summaries for all nodes using the LLM.
func (p *PDFProcessor) generateSummaries(ctx context.Context, nodes []*TreeNode) error {
	allNodes := FlattenTree(nodes)

	for _, node := range allNodes {
		if node.Text == "" {
			continue
		}

		tokenCount := CountTokens(node.Text)

		if tokenCount < p.config.SummaryTokenThreshold {
			// Use text as-is for small sections
			if len(node.Children) == 0 {
				node.Summary = node.Text
			} else {
				node.PrefixSum = node.Text
			}
		} else {
			// Generate summary via LLM
			prompt := fmt.Sprintf(SummaryPrompt, node.Title, truncateForPrompt(node.Text, 4000))
			summary, err := p.llm.Complete(ctx, prompt)
			if err != nil {
				return err
			}
			if len(node.Children) == 0 {
				node.Summary = summary
			} else {
				node.PrefixSum = summary
			}
		}
	}

	return nil
}

// extractPages extracts text content from each page of a PDF.
// Uses pdftotext (from poppler-utils) for extraction.
func (p *PDFProcessor) extractPages(pdfPath string) ([]PageContent, error) {
	// Check if pdftotext is available
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return nil, fmt.Errorf("pdftotext not found: install poppler-utils (brew install poppler on macOS)")
	}

	// Get page count
	pageCount, err := p.getPageCount(pdfPath)
	if err != nil {
		return nil, err
	}

	// Extract each page
	pages := make([]PageContent, pageCount)
	for i := 0; i < pageCount; i++ {
		text, err := p.extractPage(pdfPath, i+1)
		if err != nil {
			return nil, fmt.Errorf("extracting page %d: %w", i+1, err)
		}

		pages[i] = PageContent{
			Text:       text,
			TokenCount: CountTokens(text),
		}
	}

	return pages, nil
}

// getPageCount returns the number of pages in a PDF.
func (p *PDFProcessor) getPageCount(pdfPath string) (int, error) {
	// Use pdfinfo to get page count
	cmd := exec.Command("pdfinfo", pdfPath)
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try pdftotext with a high page number
		// This is less efficient but works if pdfinfo isn't available
		return p.getPageCountFallback(pdfPath)
	}

	// Parse "Pages: N" from output
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				count, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}
				return count, nil
			}
		}
	}

	return 0, fmt.Errorf("could not determine page count from pdfinfo")
}

// getPageCountFallback counts pages by extracting until failure.
func (p *PDFProcessor) getPageCountFallback(pdfPath string) (int, error) {
	// Binary search for page count
	low, high := 1, 10000

	for low < high {
		mid := (low + high + 1) / 2

		cmd := exec.Command("pdftotext", "-f", strconv.Itoa(mid), "-l", strconv.Itoa(mid), pdfPath, "-")
		if err := cmd.Run(); err != nil {
			high = mid - 1
		} else {
			low = mid
		}
	}

	if low == 0 {
		return 0, fmt.Errorf("could not determine page count")
	}

	return low, nil
}

// extractPage extracts text from a single page of a PDF.
func (p *PDFProcessor) extractPage(pdfPath string, pageNum int) (string, error) {
	cmd := exec.Command("pdftotext", "-f", strconv.Itoa(pageNum), "-l", strconv.Itoa(pageNum), "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// SplitLargeNodes recursively splits nodes that exceed size thresholds.
func (p *PDFProcessor) SplitLargeNodes(ctx context.Context, nodes []*TreeNode, pages []PageContent) ([]*TreeNode, error) {
	var result []*TreeNode

	for _, node := range nodes {
		// First process children
		if len(node.Children) > 0 {
			children, err := p.SplitLargeNodes(ctx, node.Children, pages)
			if err != nil {
				return nil, err
			}
			node.Children = children
		}

		// Check if node needs splitting
		pageSpan := node.EndIdx - node.StartIdx + 1
		tokenCount := 0
		if node.Text != "" {
			tokenCount = CountTokens(node.Text)
		} else if node.StartIdx >= 0 && node.EndIdx >= node.StartIdx {
			for i := node.StartIdx; i <= node.EndIdx && i < len(pages); i++ {
				tokenCount += pages[i].TokenCount
			}
		}

		needsSplit := pageSpan > p.config.MaxPageNumEachNode || tokenCount > p.config.MaxTokenNumEachNode

		if needsSplit && len(node.Children) == 0 {
			// Split this leaf node
			children, err := p.splitNode(ctx, node, pages)
			if err != nil {
				// On error, keep the node as-is
				result = append(result, node)
			} else if len(children) > 0 {
				node.Children = children
				result = append(result, node)
			} else {
				result = append(result, node)
			}
		} else {
			result = append(result, node)
		}
	}

	return result, nil
}

// splitNode splits a large node into smaller child nodes.
func (p *PDFProcessor) splitNode(ctx context.Context, node *TreeNode, pages []PageContent) ([]*TreeNode, error) {
	// Gather text for this node
	var text strings.Builder
	for i := node.StartIdx; i <= node.EndIdx && i < len(pages); i++ {
		text.WriteString(pages[i].Text)
	}

	if text.Len() == 0 {
		return nil, nil
	}

	// Use LLM to suggest subsections
	prompt := fmt.Sprintf(SplitNodePrompt, node.Title, truncateForPrompt(text.String(), 4000))

	response, err := p.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	type splitResult struct {
		Subsections []struct {
			Title         string `json:"title"`
			StartPosition int    `json:"start_position"`
			EndPosition   int    `json:"end_position"`
		} `json:"subsections"`
	}

	result, err := ExtractJSON[splitResult](response)
	if err != nil {
		// Fallback: split evenly by pages
		return p.splitNodeByPages(node, pages), nil
	}

	if len(result.Subsections) == 0 {
		return nil, nil
	}

	// Create child nodes
	var children []*TreeNode
	totalPages := node.EndIdx - node.StartIdx + 1

	for i, sub := range result.Subsections {
		// Map character positions to page numbers (approximate)
		startRatio := float64(sub.StartPosition) / float64(text.Len())
		endRatio := float64(sub.EndPosition) / float64(text.Len())

		startPage := node.StartIdx + int(startRatio*float64(totalPages))
		endPage := node.StartIdx + int(endRatio*float64(totalPages))

		if startPage < node.StartIdx {
			startPage = node.StartIdx
		}
		if endPage > node.EndIdx {
			endPage = node.EndIdx
		}
		if i < len(result.Subsections)-1 && endPage >= result.Subsections[i+1].StartPosition {
			// Avoid overlap
			endPage = node.StartIdx + int(float64(result.Subsections[i+1].StartPosition)/float64(text.Len())*float64(totalPages)) - 1
		}

		children = append(children, &TreeNode{
			Title:    sub.Title,
			StartIdx: startPage,
			EndIdx:   endPage,
		})
	}

	return children, nil
}

// splitNodeByPages splits a node evenly by page count.
func (p *PDFProcessor) splitNodeByPages(node *TreeNode, _ []PageContent) []*TreeNode {
	totalPages := node.EndIdx - node.StartIdx + 1
	if totalPages <= p.config.MaxPageNumEachNode {
		return nil
	}

	numParts := (totalPages + p.config.MaxPageNumEachNode - 1) / p.config.MaxPageNumEachNode
	pagesPerPart := totalPages / numParts

	var children []*TreeNode
	for i := 0; i < numParts; i++ {
		startPage := node.StartIdx + i*pagesPerPart
		endPage := startPage + pagesPerPart - 1
		if i == numParts-1 {
			endPage = node.EndIdx
		}

		children = append(children, &TreeNode{
			Title:    fmt.Sprintf("%s (Part %d)", node.Title, i+1),
			StartIdx: startPage,
			EndIdx:   endPage,
		})
	}

	return children
}
