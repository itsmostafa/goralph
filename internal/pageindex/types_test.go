package pageindex

import (
	"encoding/json"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.Model != "gpt-4o" {
		t.Errorf("expected Model='gpt-4o', got %q", cfg.Model)
	}

	if cfg.TOCCheckPageNum != 20 {
		t.Errorf("expected TOCCheckPageNum=20, got %d", cfg.TOCCheckPageNum)
	}

	if cfg.MaxPageNumEachNode != 10 {
		t.Errorf("expected MaxPageNumEachNode=10, got %d", cfg.MaxPageNumEachNode)
	}

	if cfg.MaxTokenNumEachNode != 5000 {
		t.Errorf("expected MaxTokenNumEachNode=5000, got %d", cfg.MaxTokenNumEachNode)
	}

	if !cfg.IfAddNodeID {
		t.Error("expected IfAddNodeID=true")
	}

	if cfg.IfAddNodeText {
		t.Error("expected IfAddNodeText=false")
	}

	if cfg.IfAddNodeSummary {
		t.Error("expected IfAddNodeSummary=false")
	}
}

func TestTreeNodeString(t *testing.T) {
	node := &TreeNode{
		Title:    "Test Node",
		NodeID:   "0001",
		StartIdx: 1,
		EndIdx:   10,
	}

	result := node.String()
	if result == "" {
		t.Error("expected non-empty string representation")
	}

	// Should be valid JSON
	var parsed TreeNode
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("String() should return valid JSON: %v", err)
	}

	if parsed.Title != "Test Node" {
		t.Errorf("expected Title='Test Node', got %q", parsed.Title)
	}
}

func TestDocumentString(t *testing.T) {
	doc := &Document{
		Name:        "Test Document",
		Description: "A test document",
		Structure: []*TreeNode{
			{Title: "Chapter 1"},
		},
	}

	result := doc.String()
	if result == "" {
		t.Error("expected non-empty string representation")
	}

	// Should be valid JSON
	var parsed Document
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("String() should return valid JSON: %v", err)
	}

	if parsed.Name != "Test Document" {
		t.Errorf("expected Name='Test Document', got %q", parsed.Name)
	}
}

func TestTreeNodeClone(t *testing.T) {
	t.Run("nil node", func(t *testing.T) {
		var node *TreeNode
		clone := node.Clone()
		if clone != nil {
			t.Error("expected nil clone for nil node")
		}
	})

	t.Run("simple node", func(t *testing.T) {
		node := &TreeNode{
			Title:    "Original",
			NodeID:   "0001",
			StartIdx: 1,
			EndIdx:   10,
			Text:     "Some text",
		}
		clone := node.Clone()

		if clone == node {
			t.Error("clone should be a different object")
		}
		if clone.Title != node.Title {
			t.Errorf("expected Title=%q, got %q", node.Title, clone.Title)
		}

		// Modify clone and ensure original is unchanged
		clone.Title = "Modified"
		if node.Title != "Original" {
			t.Error("modifying clone should not affect original")
		}
	})

	t.Run("node with children", func(t *testing.T) {
		node := &TreeNode{
			Title: "Parent",
			Children: []*TreeNode{
				{Title: "Child 1"},
				{Title: "Child 2"},
			},
		}
		clone := node.Clone()

		if len(clone.Children) != 2 {
			t.Errorf("expected 2 children, got %d", len(clone.Children))
		}

		// Modify clone's children
		clone.Children[0].Title = "Modified Child"
		if node.Children[0].Title != "Child 1" {
			t.Error("modifying clone's children should not affect original")
		}
	})
}

func TestTreeNodeWalk(t *testing.T) {
	t.Run("nil node", func(t *testing.T) {
		var node *TreeNode
		count := 0
		node.Walk(func(n *TreeNode) { count++ })
		if count != 0 {
			t.Error("expected no walks for nil node")
		}
	})

	t.Run("tree traversal", func(t *testing.T) {
		node := &TreeNode{
			Title: "Root",
			Children: []*TreeNode{
				{Title: "Child 1", Children: []*TreeNode{
					{Title: "Grandchild"},
				}},
				{Title: "Child 2"},
			},
		}

		var visited []string
		node.Walk(func(n *TreeNode) {
			visited = append(visited, n.Title)
		})

		if len(visited) != 4 {
			t.Errorf("expected 4 nodes visited, got %d", len(visited))
		}

		// Depth-first order
		expected := []string{"Root", "Child 1", "Grandchild", "Child 2"}
		for i, title := range expected {
			if visited[i] != title {
				t.Errorf("expected visited[%d]=%q, got %q", i, title, visited[i])
			}
		}
	})
}

func TestTreeNodeLeafNodes(t *testing.T) {
	node := &TreeNode{
		Title: "Root",
		Children: []*TreeNode{
			{Title: "Child 1"},
			{Title: "Child 2", Children: []*TreeNode{
				{Title: "Grandchild"},
			}},
		},
	}

	leaves := node.LeafNodes()
	if len(leaves) != 2 {
		t.Errorf("expected 2 leaf nodes, got %d", len(leaves))
	}

	titles := map[string]bool{}
	for _, leaf := range leaves {
		titles[leaf.Title] = true
	}

	if !titles["Child 1"] || !titles["Grandchild"] {
		t.Error("expected leaves to be 'Child 1' and 'Grandchild'")
	}
}

func TestTreeNodeAllNodes(t *testing.T) {
	node := &TreeNode{
		Title: "Root",
		Children: []*TreeNode{
			{Title: "Child 1"},
			{Title: "Child 2"},
		},
	}

	all := node.AllNodes()
	if len(all) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(all))
	}
}

func TestWriteNodeIDs(t *testing.T) {
	nodes := []*TreeNode{
		{
			Title: "Chapter 1",
			Children: []*TreeNode{
				{Title: "Section 1.1"},
				{Title: "Section 1.2"},
			},
		},
		{
			Title: "Chapter 2",
		},
	}

	count := WriteNodeIDs(nodes)
	if count != 4 {
		t.Errorf("expected 4 nodes assigned IDs, got %d", count)
	}

	// Check IDs are sequential
	if nodes[0].NodeID != "0000" {
		t.Errorf("expected first node ID='0000', got %q", nodes[0].NodeID)
	}
	if nodes[0].Children[0].NodeID != "0001" {
		t.Errorf("expected second node ID='0001', got %q", nodes[0].Children[0].NodeID)
	}
	if nodes[0].Children[1].NodeID != "0002" {
		t.Errorf("expected third node ID='0002', got %q", nodes[0].Children[1].NodeID)
	}
	if nodes[1].NodeID != "0003" {
		t.Errorf("expected fourth node ID='0003', got %q", nodes[1].NodeID)
	}
}

func TestTOCItemJSON(t *testing.T) {
	page := 5
	item := TOCItem{
		Structure: "1.2.3",
		Title:     "Test Section",
		Page:      &page,
		Level:     3,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal TOCItem: %v", err)
	}

	var parsed TOCItem
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal TOCItem: %v", err)
	}

	if parsed.Structure != "1.2.3" {
		t.Errorf("expected Structure='1.2.3', got %q", parsed.Structure)
	}
	if parsed.Title != "Test Section" {
		t.Errorf("expected Title='Test Section', got %q", parsed.Title)
	}
	if parsed.Page == nil || *parsed.Page != 5 {
		t.Error("expected Page=5")
	}
}
