package pageindex

import (
	"strings"
)

// ListToTree converts a flat list of TOC items into a hierarchical tree structure.
// It uses the structure field (e.g., "1.2.3") to determine parent-child relationships.
func ListToTree(items []TOCItem) []*TreeNode {
	if len(items) == 0 {
		return nil
	}

	// Map to track nodes by their structure code
	nodeMap := make(map[string]*TreeNode)
	var rootNodes []*TreeNode

	for _, item := range items {
		node := &TreeNode{
			Title:    item.Title,
			StartIdx: item.StartIndex,
			EndIdx:   item.EndIndex,
		}

		if item.Structure == "" || item.Structure == "0" {
			// Root node without structure
			rootNodes = append(rootNodes, node)
			continue
		}

		nodeMap[item.Structure] = node

		// Find parent structure
		parentStruct := getParentStructure(item.Structure)
		if parentStruct == "" {
			// Top-level node
			rootNodes = append(rootNodes, node)
		} else if parent, ok := nodeMap[parentStruct]; ok {
			// Add as child to existing parent
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not found, treat as root
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes
}

// getParentStructure returns the parent structure code.
// For example, "1.2.3" returns "1.2", and "1" returns "".
func getParentStructure(structure string) string {
	parts := strings.Split(structure, ".")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

// BuildTreeFromNodes builds a tree from a flat list of nodes with level information.
// This is used for Markdown processing where nodes have explicit levels.
func BuildTreeFromNodes(nodes []TOCItem) []*TreeNode {
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
		currentLevel := n.Level
		if currentLevel == 0 {
			currentLevel = 1 // Default to level 1 if not specified
		}

		treeNode := &TreeNode{
			Title:    n.Title,
			NodeID:   padNodeID(nodeCounter),
			Text:     "", // Will be filled in later
			LineNum:  n.Level, // Temporarily store level, will be updated
			StartIdx: n.StartIndex,
			EndIdx:   n.EndIndex,
		}
		nodeCounter++

		// Pop nodes from stack until we find parent level
		for len(stack) > 0 && stack[len(stack)-1].level >= currentLevel {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			rootNodes = append(rootNodes, treeNode)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, treeNode)
		}

		stack = append(stack, stackEntry{node: treeNode, level: currentLevel})
	}

	return rootNodes
}

// PostProcessTOC converts flat TOC items to tree structure with proper indices.
// It calculates end indices based on the next item's start and handles "appear_start".
func PostProcessTOC(items []TOCItem, endPhysicalIndex int) []*TreeNode {
	if len(items) == 0 {
		return nil
	}

	// Calculate start and end indices
	for i := range items {
		if items[i].PhysicalIndex != nil {
			items[i].StartIndex = *items[i].PhysicalIndex
		}

		if i < len(items)-1 {
			if items[i+1].PhysicalIndex != nil {
				nextIdx := *items[i+1].PhysicalIndex
				// If next section starts at beginning of page, previous ends at page before
				if items[i+1].AppearStart == "yes" {
					items[i].EndIndex = nextIdx - 1
				} else {
					items[i].EndIndex = nextIdx
				}
			}
		} else {
			items[i].EndIndex = endPhysicalIndex
		}
	}

	// Convert to tree
	tree := ListToTree(items)
	if len(tree) == 0 {
		// If tree conversion failed, return flat structure
		var nodes []*TreeNode
		for _, item := range items {
			nodes = append(nodes, &TreeNode{
				Title:    item.Title,
				StartIdx: item.StartIndex,
				EndIdx:   item.EndIndex,
			})
		}
		return nodes
	}

	return tree
}

// AddPrefaceIfNeeded adds a "Preface" node if the first item doesn't start at page 1.
func AddPrefaceIfNeeded(items []TOCItem) []TOCItem {
	if len(items) == 0 {
		return items
	}

	firstIdx := items[0].PhysicalIndex
	if firstIdx != nil && *firstIdx > 1 {
		one := 1
		preface := TOCItem{
			Structure:     "0",
			Title:         "Preface",
			PhysicalIndex: &one,
		}
		return append([]TOCItem{preface}, items...)
	}

	return items
}

// ValidateAndTruncatePhysicalIndices removes items with invalid physical indices.
func ValidateAndTruncatePhysicalIndices(items []TOCItem, pageCount, startIndex int) []TOCItem {
	maxAllowedPage := pageCount + startIndex - 1

	var valid []TOCItem
	for _, item := range items {
		if item.PhysicalIndex != nil {
			if *item.PhysicalIndex <= maxAllowedPage {
				valid = append(valid, item)
			}
			// Skip items with index beyond document length
		} else {
			valid = append(valid, item)
		}
	}

	return valid
}

// padNodeID pads an integer ID to 4 digits.
func padNodeID(id int) string {
	return strings.Repeat("0", max(0, 4-len(itoa(id)))) + itoa(id)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b strings.Builder
	for i > 0 {
		b.WriteByte(byte('0' + i%10))
		i /= 10
	}
	s := b.String()
	// Reverse
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CleanTree removes empty Children slices and internal fields for output.
func CleanTree(nodes []*TreeNode) []*TreeNode {
	var cleaned []*TreeNode

	for _, node := range nodes {
		cleanNode := &TreeNode{
			Title:    node.Title,
			NodeID:   node.NodeID,
			Text:     node.Text,
			Summary:  node.Summary,
			StartIdx: node.StartIdx,
			EndIdx:   node.EndIdx,
			LineNum:  node.LineNum,
		}

		if len(node.Children) > 0 {
			cleanNode.Children = CleanTree(node.Children)
		}

		cleaned = append(cleaned, cleanNode)
	}

	return cleaned
}

// FlattenTree returns all nodes in the tree as a flat slice.
func FlattenTree(nodes []*TreeNode) []*TreeNode {
	var result []*TreeNode

	var walk func([]*TreeNode)
	walk = func(children []*TreeNode) {
		for _, node := range children {
			result = append(result, node)
			if node.Children != nil {
				walk(node.Children)
			}
		}
	}

	walk(nodes)
	return result
}

// GetLeafNodes returns only leaf nodes (nodes without children).
func GetLeafNodes(nodes []*TreeNode) []*TreeNode {
	var leaves []*TreeNode

	var walk func([]*TreeNode)
	walk = func(children []*TreeNode) {
		for _, node := range children {
			if len(node.Children) == 0 {
				leaves = append(leaves, node)
			} else {
				walk(node.Children)
			}
		}
	}

	walk(nodes)
	return leaves
}
