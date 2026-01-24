package pageindex

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// TOCDetector handles detection and extraction of table of contents from documents.
type TOCDetector struct {
	llm    LLMProvider
	config *Config
}

// NewTOCDetector creates a new TOC detector with the given LLM provider.
func NewTOCDetector(llm LLMProvider, config *Config) *TOCDetector {
	if config == nil {
		config = DefaultConfig()
	}
	return &TOCDetector{llm: llm, config: config}
}

// DetectTOC scans the first N pages of a document to find the table of contents.
// Returns the TOC content, list of page indices containing TOC, and whether page numbers are given.
func (d *TOCDetector) DetectTOC(ctx context.Context, pages []PageContent) (*TOCCheckResult, error) {
	checkLimit := d.config.TOCCheckPageNum
	if checkLimit > len(pages) {
		checkLimit = len(pages)
	}

	var tocPages []int
	var tocContent strings.Builder

	// Scan pages for TOC content
	for i := 0; i < checkLimit; i++ {
		isTOC, err := d.isPageTOC(ctx, pages[i].Text)
		if err != nil {
			return nil, fmt.Errorf("checking page %d for TOC: %w", i, err)
		}

		if isTOC {
			tocPages = append(tocPages, i)
			if tocContent.Len() > 0 {
				tocContent.WriteString("\n")
			}
			tocContent.WriteString(pages[i].Text)
		} else if len(tocPages) > 0 {
			// Stop when we hit a non-TOC page after finding TOC
			break
		}
	}

	if len(tocPages) == 0 {
		return &TOCCheckResult{
			TOCContent:        "",
			TOCPageList:       nil,
			PageIndexGivenTOC: false,
		}, nil
	}

	// Check if page numbers are given in TOC
	pageIndexGiven, err := d.hasPageNumbers(ctx, tocContent.String())
	if err != nil {
		return nil, fmt.Errorf("checking for page numbers: %w", err)
	}

	return &TOCCheckResult{
		TOCContent:        tocContent.String(),
		TOCPageList:       tocPages,
		PageIndexGivenTOC: pageIndexGiven,
	}, nil
}

// isPageTOC uses the LLM to determine if a page contains TOC content.
func (d *TOCDetector) isPageTOC(ctx context.Context, pageText string) (bool, error) {
	prompt := fmt.Sprintf(TOCDetectorPrompt, truncateForPrompt(pageText, 3000))

	response, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return false, err
	}

	result, err := ExtractJSON[TOCDetectorResponse](response)
	if err != nil {
		return false, err
	}

	return strings.ToLower(result.TOCDetected) == "yes", nil
}

// hasPageNumbers uses the LLM to determine if the TOC includes page numbers.
func (d *TOCDetector) hasPageNumbers(ctx context.Context, tocContent string) (bool, error) {
	prompt := fmt.Sprintf(PageIndexGivenPrompt, truncateForPrompt(tocContent, 4000))

	response, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return false, err
	}

	result, err := ExtractJSON[PageIndexResponse](response)
	if err != nil {
		return false, err
	}

	return strings.ToLower(result.PageIndexGivenTOC) == "yes", nil
}

// TransformTOC converts raw TOC text into structured TOC items.
func (d *TOCDetector) TransformTOC(ctx context.Context, tocContent string) ([]TOCItem, error) {
	prompt := fmt.Sprintf(TOCTransformPrompt, tocContent)

	// Use continuation for potentially long responses
	fullResponse, err := d.completeWithContinuation(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("transforming TOC: %w", err)
	}

	result, err := ExtractJSON[TOCTransformResponse](fullResponse)
	if err != nil {
		return nil, err
	}

	return result.TableOfContents, nil
}

// completeWithContinuation handles long responses that may be truncated.
func (d *TOCDetector) completeWithContinuation(ctx context.Context, prompt string) (string, error) {
	provider, ok := d.llm.(*OpenAIProvider)
	if !ok {
		// Fallback to simple completion for other providers
		return d.llm.Complete(ctx, prompt)
	}

	var fullResponse strings.Builder
	history := []Message{{Role: "user", Content: prompt}}

	for i := 0; i < 10; i++ { // Max 10 continuations
		var response, finishReason string
		var err error

		if i == 0 {
			response, finishReason, err = provider.CompleteWithFinishReason(ctx, prompt, nil)
		} else {
			response, finishReason, err = provider.CompleteWithFinishReason(ctx, TOCTransformContinuePrompt, history)
		}

		if err != nil {
			return "", err
		}

		fullResponse.WriteString(response)
		history = append(history, Message{Role: "assistant", Content: response})

		if finishReason == "finished" {
			break
		}

		// Check if response looks complete
		isComplete, err := d.isResponseComplete(ctx, fullResponse.String(), prompt)
		if err != nil {
			// Continue anyway on error
			continue
		}
		if isComplete {
			break
		}

		history = append(history, Message{Role: "user", Content: TOCTransformContinuePrompt})
	}

	return fullResponse.String(), nil
}

// isResponseComplete checks if a partial response appears complete.
func (d *TOCDetector) isResponseComplete(ctx context.Context, response, original string) (bool, error) {
	prompt := fmt.Sprintf(CompletionCheckPrompt, truncateForPrompt(response, 2000), truncateForPrompt(original, 2000))

	resp, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return false, err
	}

	result, err := ExtractJSON[CompletionCheckResponse](resp)
	if err != nil {
		return false, err
	}

	return strings.ToLower(result.Completed) == "yes", nil
}

// GenerateTOC creates a TOC when the document doesn't have one.
func (d *TOCDetector) GenerateTOC(ctx context.Context, pages []PageContent) ([]TOCItem, error) {
	// Combine page texts for analysis
	var combined strings.Builder
	for i, page := range pages {
		combined.WriteString(fmt.Sprintf("\n=== Page %d ===\n", i+1))
		combined.WriteString(truncateForPrompt(page.Text, 1000))
	}

	prompt := fmt.Sprintf(GenerateTOCPrompt, truncateForPrompt(combined.String(), 8000))

	response, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generating TOC: %w", err)
	}

	result, err := ExtractJSON[TOCTransformResponse](response)
	if err != nil {
		return nil, err
	}

	return result.TableOfContents, nil
}

// MapPageNumbersToPhysical converts logical page numbers to physical page indices.
// This handles documents where logical page 1 may not be physical page 1.
func MapPageNumbersToPhysical(items []TOCItem, tocPageIndices []int, totalPages int) []TOCItem {
	if len(items) == 0 {
		return items
	}

	// Find the offset between logical and physical pages
	// Assume first content page after TOC is where numbering starts
	firstContentPage := 0
	if len(tocPageIndices) > 0 {
		firstContentPage = tocPageIndices[len(tocPageIndices)-1] + 1
	}

	// Find the smallest page number in TOC to determine offset
	minPage := -1
	for _, item := range items {
		if item.Page != nil {
			if minPage == -1 || *item.Page < minPage {
				minPage = *item.Page
			}
		}
	}

	if minPage == -1 {
		// No page numbers in TOC
		return items
	}

	// Calculate offset: physical_index = page_number + offset
	offset := firstContentPage - minPage

	result := make([]TOCItem, len(items))
	for i, item := range items {
		result[i] = item
		if item.Page != nil {
			physIdx := *item.Page + offset
			if physIdx >= 0 && physIdx < totalPages {
				result[i].PhysicalIndex = &physIdx
			}
		}
	}

	return result
}

// CheckSectionStart determines if a section starts at the beginning of its page.
func (d *TOCDetector) CheckSectionStart(ctx context.Context, title string, pageText string) (bool, error) {
	// Get first 500 characters of page
	pageStart := truncateForPrompt(pageText, 500)

	prompt := fmt.Sprintf(StartCheckPrompt, title, pageStart)

	response, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return false, err
	}

	result, err := ExtractJSON[StartCheckResponse](response)
	if err != nil {
		return false, err
	}

	return strings.ToLower(result.StartBegin) == "yes", nil
}

// CheckSectionStartsConcurrently checks multiple sections in parallel.
func (d *TOCDetector) CheckSectionStartsConcurrently(ctx context.Context, items []TOCItem, pages []PageContent, maxWorkers int) ([]TOCItem, error) {
	if maxWorkers <= 0 {
		maxWorkers = 5
	}

	result := make([]TOCItem, len(items))
	copy(result, items)

	type job struct {
		index int
		item  TOCItem
		page  PageContent
	}

	type jobResult struct {
		index      int
		startsHere bool
		err        error
	}

	jobs := make(chan job, len(items))
	results := make(chan jobResult, len(items))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				startsHere, err := d.CheckSectionStart(ctx, j.item.Title, j.page.Text)
				results <- jobResult{index: j.index, startsHere: startsHere, err: err}
			}
		}()
	}

	// Queue jobs
	for i, item := range items {
		if item.PhysicalIndex != nil && *item.PhysicalIndex < len(pages) {
			jobs <- job{index: i, item: item, page: pages[*item.PhysicalIndex]}
		}
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for r := range results {
		if r.err != nil {
			continue // Skip errors, leave as default
		}
		if r.startsHere {
			result[r.index].AppearStart = "yes"
		} else {
			result[r.index].AppearStart = "no"
		}
	}

	return result, nil
}

// truncateForPrompt shortens text to fit in prompts.
func truncateForPrompt(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...[truncated]"
}
