package pageindex

import (
	"testing"
)

func TestGetParentStructure(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single digit", "1", ""},
		{"two levels", "1.2", "1"},
		{"three levels", "1.2.3", "1.2"},
		{"four levels", "1.2.3.4", "1.2.3"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParentStructure(tt.input)
			if result != tt.expected {
				t.Errorf("getParentStructure(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestListToTree(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		result := ListToTree(nil)
		if result != nil {
			t.Error("expected nil for empty input")
		}
	})

	t.Run("flat list", func(t *testing.T) {
		items := []TOCItem{
			{Structure: "1", Title: "Chapter 1", StartIndex: 1, EndIndex: 10},
			{Structure: "2", Title: "Chapter 2", StartIndex: 11, EndIndex: 20},
		}
		result := ListToTree(items)
		if len(result) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(result))
		}
		if result[0].Title != "Chapter 1" {
			t.Errorf("expected first node title 'Chapter 1', got %q", result[0].Title)
		}
	})

	t.Run("nested structure", func(t *testing.T) {
		items := []TOCItem{
			{Structure: "1", Title: "Chapter 1", StartIndex: 1, EndIndex: 20},
			{Structure: "1.1", Title: "Section 1.1", StartIndex: 1, EndIndex: 10},
			{Structure: "1.2", Title: "Section 1.2", StartIndex: 11, EndIndex: 20},
			{Structure: "2", Title: "Chapter 2", StartIndex: 21, EndIndex: 30},
		}
		result := ListToTree(items)
		if len(result) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(result))
		}
		if len(result[0].Children) != 2 {
			t.Errorf("expected Chapter 1 to have 2 children, got %d", len(result[0].Children))
		}
		if result[0].Children[0].Title != "Section 1.1" {
			t.Errorf("expected first child 'Section 1.1', got %q", result[0].Children[0].Title)
		}
	})

	t.Run("deep nesting", func(t *testing.T) {
		items := []TOCItem{
			{Structure: "1", Title: "Chapter 1"},
			{Structure: "1.1", Title: "Section 1.1"},
			{Structure: "1.1.1", Title: "Subsection 1.1.1"},
			{Structure: "1.1.1.1", Title: "Para 1.1.1.1"},
		}
		result := ListToTree(items)
		if len(result) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(result))
		}
		// Navigate down the tree
		node := result[0]
		if len(node.Children) != 1 || node.Children[0].Title != "Section 1.1" {
			t.Error("expected Section 1.1 as child")
		}
		node = node.Children[0]
		if len(node.Children) != 1 || node.Children[0].Title != "Subsection 1.1.1" {
			t.Error("expected Subsection 1.1.1 as child")
		}
		node = node.Children[0]
		if len(node.Children) != 1 || node.Children[0].Title != "Para 1.1.1.1" {
			t.Error("expected Para 1.1.1.1 as child")
		}
	})
}

func TestBuildTreeFromNodes(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := BuildTreeFromNodes(nil)
		if result != nil {
			t.Error("expected nil for empty input")
		}
	})

	t.Run("flat nodes same level", func(t *testing.T) {
		items := []TOCItem{
			{Title: "Section A", Level: 1},
			{Title: "Section B", Level: 1},
		}
		result := BuildTreeFromNodes(items)
		if len(result) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(result))
		}
	})

	t.Run("nested by level", func(t *testing.T) {
		items := []TOCItem{
			{Title: "Chapter 1", Level: 1},
			{Title: "Section 1.1", Level: 2},
			{Title: "Section 1.2", Level: 2},
			{Title: "Chapter 2", Level: 1},
		}
		result := BuildTreeFromNodes(items)
		if len(result) != 2 {
			t.Errorf("expected 2 root nodes, got %d", len(result))
		}
		if len(result[0].Children) != 2 {
			t.Errorf("expected Chapter 1 to have 2 children, got %d", len(result[0].Children))
		}
	})
}

func TestPostProcessTOC(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := PostProcessTOC(nil, 100)
		if result != nil {
			t.Error("expected nil for empty input")
		}
	})

	t.Run("calculates end indices", func(t *testing.T) {
		idx1, idx2 := 1, 11
		items := []TOCItem{
			{Structure: "1", Title: "Chapter 1", PhysicalIndex: &idx1},
			{Structure: "2", Title: "Chapter 2", PhysicalIndex: &idx2, AppearStart: "yes"},
		}
		result := PostProcessTOC(items, 20)
		if len(result) != 2 {
			t.Fatalf("expected 2 nodes, got %d", len(result))
		}
		// First item should end at 10 (11-1) because next starts at beginning
		if result[0].EndIdx != 10 {
			t.Errorf("expected first node EndIdx=10, got %d", result[0].EndIdx)
		}
		// Second item should end at document end
		if result[1].EndIdx != 20 {
			t.Errorf("expected second node EndIdx=20, got %d", result[1].EndIdx)
		}
	})
}

func TestAddPrefaceIfNeeded(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := AddPrefaceIfNeeded(nil)
		if result != nil {
			t.Error("expected nil for empty input")
		}
	})

	t.Run("first item at page 1", func(t *testing.T) {
		idx := 1
		items := []TOCItem{{Title: "Chapter 1", PhysicalIndex: &idx}}
		result := AddPrefaceIfNeeded(items)
		if len(result) != 1 {
			t.Error("expected no preface added when first page is 1")
		}
	})

	t.Run("first item at page 5", func(t *testing.T) {
		idx := 5
		items := []TOCItem{{Title: "Chapter 1", PhysicalIndex: &idx}}
		result := AddPrefaceIfNeeded(items)
		if len(result) != 2 {
			t.Fatalf("expected preface added, got %d items", len(result))
		}
		if result[0].Title != "Preface" {
			t.Errorf("expected first item to be 'Preface', got %q", result[0].Title)
		}
	})
}

func TestValidateAndTruncatePhysicalIndices(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		idx1, idx2 := 5, 10
		items := []TOCItem{
			{Title: "A", PhysicalIndex: &idx1},
			{Title: "B", PhysicalIndex: &idx2},
		}
		result := ValidateAndTruncatePhysicalIndices(items, 20, 1)
		if len(result) != 2 {
			t.Error("expected all items to be valid")
		}
	})

	t.Run("remove out of range", func(t *testing.T) {
		idx1, idx2 := 5, 50
		items := []TOCItem{
			{Title: "A", PhysicalIndex: &idx1},
			{Title: "B", PhysicalIndex: &idx2},
		}
		result := ValidateAndTruncatePhysicalIndices(items, 20, 1)
		if len(result) != 1 {
			t.Errorf("expected 1 valid item, got %d", len(result))
		}
		if result[0].Title != "A" {
			t.Errorf("expected remaining item to be 'A', got %q", result[0].Title)
		}
	})
}

func TestPadNodeID(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0000"},
		{1, "0001"},
		{12, "0012"},
		{123, "0123"},
		{1234, "1234"},
		{12345, "12345"},
	}

	for _, tt := range tests {
		result := padNodeID(tt.input)
		if result != tt.expected {
			t.Errorf("padNodeID(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFlattenTree(t *testing.T) {
	root := &TreeNode{
		Title: "Root",
		Children: []*TreeNode{
			{Title: "Child 1"},
			{Title: "Child 2", Children: []*TreeNode{
				{Title: "Grandchild"},
			}},
		},
	}
	result := FlattenTree([]*TreeNode{root})
	if len(result) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(result))
	}
}

func TestGetLeafNodes(t *testing.T) {
	root := &TreeNode{
		Title: "Root",
		Children: []*TreeNode{
			{Title: "Child 1"},
			{Title: "Child 2", Children: []*TreeNode{
				{Title: "Grandchild"},
			}},
		},
	}
	leaves := GetLeafNodes([]*TreeNode{root})
	if len(leaves) != 2 {
		t.Errorf("expected 2 leaf nodes, got %d", len(leaves))
	}
	titles := map[string]bool{}
	for _, l := range leaves {
		titles[l.Title] = true
	}
	if !titles["Child 1"] || !titles["Grandchild"] {
		t.Error("expected leaf nodes to be 'Child 1' and 'Grandchild'")
	}
}

func TestCleanTree(t *testing.T) {
	nodes := []*TreeNode{
		{
			Title:  "Chapter 1",
			NodeID: "0001",
			Children: []*TreeNode{
				{Title: "Section 1.1", NodeID: "0002"},
			},
		},
	}
	result := CleanTree(nodes)
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}
	if result[0].Title != "Chapter 1" {
		t.Errorf("expected title 'Chapter 1', got %q", result[0].Title)
	}
	if len(result[0].Children) != 1 {
		t.Error("expected children to be preserved")
	}
}
