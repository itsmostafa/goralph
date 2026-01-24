// Package pageindex implements a vectorless, reasoning-based RAG framework
// for building tree-structured indexes from documents.
//
// This is a Go port of the PageIndex Python framework:
// https://github.com/VectifyAI/PageIndex
package pageindex

import (
	"encoding/json"
	"fmt"
)

// TreeNode represents a node in the document structure tree.
// Each node corresponds to a section in the document with its title,
// position, and optional content.
type TreeNode struct {
	Title     string      `json:"title"`
	NodeID    string      `json:"node_id,omitempty"`
	StartIdx  int         `json:"start_index,omitempty"`
	EndIdx    int         `json:"end_index,omitempty"`
	LineNum   int         `json:"line_num,omitempty"`
	Text      string      `json:"text,omitempty"`
	Summary   string      `json:"summary,omitempty"`
	PrefixSum string      `json:"prefix_summary,omitempty"`
	Children  []*TreeNode `json:"nodes,omitempty"`
}

// TOCItem represents a flat table-of-contents entry before tree construction.
// It contains the hierarchical structure index (e.g., "1.2.3") and page references.
type TOCItem struct {
	Structure     string `json:"structure,omitempty"`      // Hierarchical index like "1.2.3"
	Title         string `json:"title"`                    // Section title
	Page          *int   `json:"page,omitempty"`           // Logical page number from TOC
	PhysicalIndex *int   `json:"physical_index,omitempty"` // Actual PDF page index
	StartIndex    int    `json:"start_index,omitempty"`    // Start page after processing
	EndIndex      int    `json:"end_index,omitempty"`      // End page after processing
	AppearStart   string `json:"appear_start,omitempty"`   // "yes" if section starts at page beginning
	Level         int    `json:"level,omitempty"`          // Header level for markdown
}

// Config holds configuration options for PageIndex operations.
type Config struct {
	// Model specifies the LLM model to use (e.g., "gpt-4o", "claude-sonnet-4-20250514")
	Model string

	// TOCCheckPageNum is the maximum number of pages to scan for TOC detection
	TOCCheckPageNum int

	// MaxPageNumEachNode is the threshold for splitting large nodes
	MaxPageNumEachNode int

	// MaxTokenNumEachNode is the token threshold for splitting large nodes
	MaxTokenNumEachNode int

	// IfAddNodeID controls whether to assign unique IDs to nodes
	IfAddNodeID bool

	// IfAddNodeText controls whether to include full text in nodes
	IfAddNodeText bool

	// IfAddNodeSummary controls whether to generate summaries for nodes
	IfAddNodeSummary bool

	// IfAddDocDescription controls whether to generate a document description
	IfAddDocDescription bool

	// SummaryTokenThreshold is the threshold below which text is used as-is for summary
	SummaryTokenThreshold int

	// MinNodeToken is the minimum token count for tree thinning
	MinNodeToken int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model:                 "gpt-4o",
		TOCCheckPageNum:       20,
		MaxPageNumEachNode:    10,
		MaxTokenNumEachNode:   5000,
		IfAddNodeID:           true,
		IfAddNodeText:         false,
		IfAddNodeSummary:      false,
		IfAddDocDescription:   false,
		SummaryTokenThreshold: 200,
		MinNodeToken:          500,
	}
}

// PageContent represents extracted content from a single page.
type PageContent struct {
	Text       string // Extracted text content
	TokenCount int    // Token count for the page
}

// Document represents a processed document with its metadata and structure.
type Document struct {
	Name        string      `json:"doc_name"`
	Description string      `json:"doc_description,omitempty"`
	Structure   []*TreeNode `json:"structure"`
}

// TOCCheckResult contains the result of checking for a table of contents.
type TOCCheckResult struct {
	TOCContent        string // Raw TOC content
	TOCPageList       []int  // List of page indices containing TOC
	PageIndexGivenTOC bool   // Whether page numbers are given in TOC
}

// VerifyResult contains the result of verifying a TOC entry.
type VerifyResult struct {
	ListIndex     int    `json:"list_index"`
	Answer        string `json:"answer"` // "yes" or "no"
	Title         string `json:"title"`
	PageNumber    *int   `json:"page_number,omitempty"`
	PhysicalIndex *int   `json:"physical_index,omitempty"`
}

// LLMResponse represents a generic JSON response from the LLM.
type LLMResponse struct {
	Thinking string `json:"thinking,omitempty"`
	Answer   string `json:"answer,omitempty"`
}

// TOCDetectorResponse represents the response from TOC detection.
type TOCDetectorResponse struct {
	Thinking    string `json:"thinking,omitempty"`
	TOCDetected string `json:"toc_detected"` // "yes" or "no"
}

// PageIndexResponse represents the response for page index detection.
type PageIndexResponse struct {
	Thinking          string `json:"thinking,omitempty"`
	PageIndexGivenTOC string `json:"page_index_given_in_toc"` // "yes" or "no"
}

// CompletionCheckResponse represents the response for checking completeness.
type CompletionCheckResponse struct {
	Thinking  string `json:"thinking,omitempty"`
	Completed string `json:"completed"` // "yes" or "no"
}

// StartCheckResponse represents the response for checking section start.
type StartCheckResponse struct {
	Thinking   string `json:"thinking,omitempty"`
	StartBegin string `json:"start_begin"` // "yes" or "no"
}

// TOCTransformResponse represents the response from TOC transformation.
type TOCTransformResponse struct {
	TableOfContents []TOCItem `json:"table_of_contents"`
}

// String returns a JSON representation of the TreeNode for debugging.
func (n *TreeNode) String() string {
	b, _ := json.MarshalIndent(n, "", "  ")
	return string(b)
}

// String returns a JSON representation of the Document.
func (d *Document) String() string {
	b, _ := json.MarshalIndent(d, "", "  ")
	return string(b)
}

// Clone creates a deep copy of the TreeNode.
func (n *TreeNode) Clone() *TreeNode {
	if n == nil {
		return nil
	}
	clone := &TreeNode{
		Title:     n.Title,
		NodeID:    n.NodeID,
		StartIdx:  n.StartIdx,
		EndIdx:    n.EndIdx,
		LineNum:   n.LineNum,
		Text:      n.Text,
		Summary:   n.Summary,
		PrefixSum: n.PrefixSum,
	}
	if n.Children != nil {
		clone.Children = make([]*TreeNode, len(n.Children))
		for i, child := range n.Children {
			clone.Children[i] = child.Clone()
		}
	}
	return clone
}

// Walk traverses the tree in depth-first order, calling fn for each node.
func (n *TreeNode) Walk(fn func(*TreeNode)) {
	if n == nil {
		return
	}
	fn(n)
	for _, child := range n.Children {
		child.Walk(fn)
	}
}

// LeafNodes returns all leaf nodes (nodes without children).
func (n *TreeNode) LeafNodes() []*TreeNode {
	var leaves []*TreeNode
	n.Walk(func(node *TreeNode) {
		if len(node.Children) == 0 {
			leaves = append(leaves, node)
		}
	})
	return leaves
}

// AllNodes returns all nodes in the tree as a flat slice.
func (n *TreeNode) AllNodes() []*TreeNode {
	var nodes []*TreeNode
	n.Walk(func(node *TreeNode) {
		nodes = append(nodes, node)
	})
	return nodes
}

// WriteNodeIDs assigns sequential zero-padded IDs to all nodes in the tree.
func WriteNodeIDs(nodes []*TreeNode) int {
	counter := 0
	var assign func([]*TreeNode)
	assign = func(children []*TreeNode) {
		for _, node := range children {
			node.NodeID = fmt.Sprintf("%04d", counter)
			counter++
			if node.Children != nil {
				assign(node.Children)
			}
		}
	}
	assign(nodes)
	return counter
}
