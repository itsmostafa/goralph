package pageindex

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MarkdownNode represents a node extracted from markdown headers.
type MarkdownNode struct {
	Title    string
	Level    int
	LineNum  int
	Text     string
	TokenCnt int
}

var (
	headerPattern    = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	codeBlockPattern = regexp.MustCompile(`^\x60\x60\x60`)
)

// ExtractNodesFromMarkdown parses a markdown file and extracts header nodes.
func ExtractNodesFromMarkdown(content string) ([]MarkdownNode, []string) {
	lines := strings.Split(content, "\n")
	var nodes []MarkdownNode
	inCodeBlock := false

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code block delimiters
		if codeBlockPattern.MatchString(trimmed) {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Skip if in code block or empty
		if inCodeBlock || trimmed == "" {
			continue
		}

		// Check for header
		if matches := headerPattern.FindStringSubmatch(trimmed); matches != nil {
			nodes = append(nodes, MarkdownNode{
				Title:   strings.TrimSpace(matches[2]),
				Level:   len(matches[1]), // Number of # characters
				LineNum: lineNum + 1,     // 1-indexed
			})
		}
	}

	return nodes, lines
}

// ExtractNodeTextContent populates the Text field for each node.
// Text includes everything from the header line until the next header.
func ExtractNodeTextContent(nodes []MarkdownNode, lines []string) []MarkdownNode {
	result := make([]MarkdownNode, len(nodes))
	copy(result, nodes)

	for i := range result {
		startLine := result[i].LineNum - 1 // 0-indexed

		var endLine int
		if i+1 < len(result) {
			endLine = result[i+1].LineNum - 1
		} else {
			endLine = len(lines)
		}

		// Join lines for this section
		result[i].Text = strings.TrimSpace(strings.Join(lines[startLine:endLine], "\n"))
	}

	return result
}

// TreeThinning merges nodes that are below the minimum token threshold.
// Small children are absorbed into their parent.
func TreeThinning(nodes []MarkdownNode, minTokenThreshold int, tokenCounter func(string) int) []MarkdownNode {
	// First, calculate cumulative token counts (including children)
	nodes = updateNodeTokenCounts(nodes, tokenCounter)

	// Set of indices to remove (absorbed into parent)
	toRemove := make(map[int]bool)

	// Process from end to start so children are processed before parents
	for i := len(nodes) - 1; i >= 0; i-- {
		if toRemove[i] {
			continue
		}

		node := nodes[i]
		if node.TokenCnt < minTokenThreshold {
			// Find all children of this node
			childIndices := findAllChildren(nodes, i)

			// Collect and merge children text
			var childTexts []string
			for _, idx := range childIndices {
				if !toRemove[idx] && strings.TrimSpace(nodes[idx].Text) != "" {
					childTexts = append(childTexts, nodes[idx].Text)
					toRemove[idx] = true
				}
			}

			// Merge into parent
			if len(childTexts) > 0 {
				merged := nodes[i].Text
				for _, ct := range childTexts {
					if merged != "" && !strings.HasSuffix(merged, "\n") {
						merged += "\n\n"
					}
					merged += ct
				}
				nodes[i].Text = merged
				nodes[i].TokenCnt = tokenCounter(merged)
			}
		}
	}

	// Build result excluding removed nodes
	var result []MarkdownNode
	for i, node := range nodes {
		if !toRemove[i] {
			result = append(result, node)
		}
	}

	return result
}

// updateNodeTokenCounts calculates token counts including children.
func updateNodeTokenCounts(nodes []MarkdownNode, tokenCounter func(string) int) []MarkdownNode {
	result := make([]MarkdownNode, len(nodes))
	copy(result, nodes)

	// Process from end to start
	for i := len(result) - 1; i >= 0; i-- {
		childIndices := findAllChildren(nodes, i)

		// Combine own text with children
		totalText := result[i].Text
		for _, idx := range childIndices {
			if result[idx].Text != "" {
				totalText += "\n" + result[idx].Text
			}
		}

		result[i].TokenCnt = tokenCounter(totalText)
	}

	return result
}

// findAllChildren finds all descendant nodes of the given parent index.
func findAllChildren(nodes []MarkdownNode, parentIdx int) []int {
	if parentIdx >= len(nodes) {
		return nil
	}

	parentLevel := nodes[parentIdx].Level
	var children []int

	for i := parentIdx + 1; i < len(nodes); i++ {
		if nodes[i].Level <= parentLevel {
			break // Hit a node at same or higher level
		}
		children = append(children, i)
	}

	return children
}

// BuildMarkdownTree builds a tree from markdown nodes.
func BuildMarkdownTree(nodes []MarkdownNode) []*TreeNode {
	if len(nodes) == 0 {
		return nil
	}

	type stackEntry struct {
		node  *TreeNode
		level int
	}

	var stack []stackEntry
	var rootNodes []*TreeNode
	nodeCounter := 1

	for _, n := range nodes {
		treeNode := &TreeNode{
			Title:   n.Title,
			NodeID:  padNodeID(nodeCounter),
			Text:    n.Text,
			LineNum: n.LineNum,
		}
		nodeCounter++

		// Pop stack until we find parent
		for len(stack) > 0 && stack[len(stack)-1].level >= n.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			rootNodes = append(rootNodes, treeNode)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, treeNode)
		}

		stack = append(stack, stackEntry{node: treeNode, level: n.Level})
	}

	return rootNodes
}

// MDToTree processes a markdown file and returns a tree structure.
func MDToTree(ctx context.Context, mdPath string, cfg *Config, llm LLMProvider) (*Document, error) {
	// Read file
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, err
	}

	// Extract nodes
	nodes, lines := ExtractNodesFromMarkdown(string(content))
	nodes = ExtractNodeTextContent(nodes, lines)

	// Optional tree thinning
	if cfg.MinNodeToken > 0 {
		tokenCounter := func(s string) int {
			// Simple word-based approximation
			return len(strings.Fields(s)) * 4 / 3 // ~1.33 tokens per word average
		}
		nodes = TreeThinning(nodes, cfg.MinNodeToken, tokenCounter)
	}

	// Build tree
	tree := BuildMarkdownTree(nodes)

	// Assign node IDs
	if cfg.IfAddNodeID {
		WriteNodeIDs(tree)
	}

	// Generate summaries if requested
	if cfg.IfAddNodeSummary && llm != nil {
		if err := generateSummariesForTree(ctx, tree, cfg, llm); err != nil {
			return nil, err
		}
	}

	// Generate document description if requested
	var description string
	if cfg.IfAddDocDescription && llm != nil {
		var err error
		description, err = generateDocDescription(ctx, tree, llm)
		if err != nil {
			return nil, err
		}
	}

	// Remove text if not requested
	if !cfg.IfAddNodeText {
		removeTextFromTree(tree)
	}

	// Build document
	docName := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath))

	return &Document{
		Name:        docName,
		Description: description,
		Structure:   tree,
	}, nil
}

// generateSummariesForTree generates summaries for all nodes in the tree.
func generateSummariesForTree(ctx context.Context, nodes []*TreeNode, cfg *Config, llm LLMProvider) error {
	allNodes := FlattenTree(nodes)

	for _, node := range allNodes {
		if node.Text == "" {
			continue
		}

		// Simple token approximation
		tokenCount := len(strings.Fields(node.Text)) * 4 / 3

		if tokenCount < cfg.SummaryTokenThreshold {
			// Use text as-is for small sections
			if len(node.Children) == 0 {
				node.Summary = node.Text
			} else {
				node.PrefixSum = node.Text
			}
		} else {
			// Generate summary via LLM
			summary, err := generateNodeSummary(ctx, node, llm)
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

// generateNodeSummary generates a summary for a single node.
func generateNodeSummary(ctx context.Context, node *TreeNode, llm LLMProvider) (string, error) {
	prompt := `You are given a part of a document, your task is to generate a description of the partial document about what are main points covered in the partial document.

Partial Document Text: ` + node.Text + `

Directly return the description, do not include any other text.`

	return llm.Complete(ctx, prompt)
}

// generateDocDescription generates a description for the entire document.
func generateDocDescription(ctx context.Context, tree []*TreeNode, llm LLMProvider) (string, error) {
	// Create clean structure for description (without text)
	cleanTree := createCleanStructureForDescription(tree)

	prompt := `Your are an expert in generating descriptions for a document.
You are given a structure of a document. Your task is to generate a one-sentence description for the document, which makes it easy to distinguish the document from other documents.

Document Structure: ` + structureToString(cleanTree) + `

Directly return the description, do not include any other text.`

	return llm.Complete(ctx, prompt)
}

// createCleanStructureForDescription creates a minimal structure for doc description.
func createCleanStructureForDescription(nodes []*TreeNode) []*TreeNode {
	var clean []*TreeNode
	for _, node := range nodes {
		cleanNode := &TreeNode{
			Title:     node.Title,
			NodeID:    node.NodeID,
			Summary:   node.Summary,
			PrefixSum: node.PrefixSum,
		}
		if len(node.Children) > 0 {
			cleanNode.Children = createCleanStructureForDescription(node.Children)
		}
		clean = append(clean, cleanNode)
	}
	return clean
}

// structureToString converts a tree to a string representation.
func structureToString(nodes []*TreeNode) string {
	var sb strings.Builder
	var write func([]*TreeNode, int)
	write = func(children []*TreeNode, indent int) {
		for _, node := range children {
			sb.WriteString(strings.Repeat("  ", indent))
			sb.WriteString("- ")
			sb.WriteString(node.Title)
			if node.Summary != "" {
				sb.WriteString(": ")
				sb.WriteString(truncate(node.Summary, 100))
			}
			sb.WriteString("\n")
			if len(node.Children) > 0 {
				write(node.Children, indent+1)
			}
		}
	}
	write(nodes, 0)
	return sb.String()
}

// removeTextFromTree removes Text field from all nodes.
func removeTextFromTree(nodes []*TreeNode) {
	for _, node := range nodes {
		node.Text = ""
		if len(node.Children) > 0 {
			removeTextFromTree(node.Children)
		}
	}
}

// ReadMarkdownFile reads a markdown file and returns its content.
func ReadMarkdownFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// PrintTOC prints a tree structure as an indented table of contents.
func PrintTOC(nodes []*TreeNode, indent int) string {
	var sb strings.Builder
	for _, node := range nodes {
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString(node.Title)
		sb.WriteString("\n")
		if len(node.Children) > 0 {
			sb.WriteString(PrintTOC(node.Children, indent+1))
		}
	}
	return sb.String()
}
