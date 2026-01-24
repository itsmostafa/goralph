// Package pageindex implements a vectorless, reasoning-based RAG (Retrieval
// Augmented Generation) framework for building tree-structured indexes from
// documents.
//
// This is a Go port of the PageIndex Python framework:
// https://github.com/VectifyAI/PageIndex
//
// # Overview
//
// PageIndex uses LLM reasoning to traverse document structures instead of
// vector similarity search. It builds hierarchical tree indexes from PDFs
// and Markdown files, enabling efficient document navigation and retrieval.
//
// # Key Concepts
//
//   - Tree-structured indexes: Documents are parsed into hierarchical trees
//     where each node represents a section with title, content, and children.
//
//   - TOC detection: For PDFs, the system detects table of contents pages and
//     uses them to build the document structure.
//
//   - Structure indices: Nodes are assigned hierarchical structure codes like
//     "1.2.3" that define parent-child relationships.
//
//   - LLM verification: Section boundaries are verified using LLM calls to
//     check if titles appear on specified pages.
//
// # Usage
//
// For Markdown files:
//
//	cfg := pageindex.DefaultConfig()
//	doc, err := pageindex.MDToTree(ctx, "document.md", cfg, llmProvider)
//
// For PDFs (to be implemented):
//
//	doc, err := pageindex.PDFToTree(ctx, "document.pdf", cfg, llmProvider)
//
// # Architecture
//
// The package is organized into several components:
//
//   - types.go: Core data types (TreeNode, TOCItem, Config)
//   - llm.go: LLM provider interface and OpenAI implementation
//   - tree.go: Tree building from flat TOC lists
//   - markdown.go: Markdown file processing
//   - tokens.go: Token counting utilities
//
// # Python Reference
//
// This implementation is based on the PageIndex Python framework. Key mappings:
//
//   - page_index.py:tree_parser → tree.PostProcessTOC
//   - page_index.py:meta_processor → (planned for PDF support)
//   - page_index_md.py:md_to_tree → markdown.MDToTree
//   - utils.py:list_to_tree → tree.ListToTree
//   - utils.py:build_tree_from_nodes → tree.BuildTreeFromNodes
package pageindex
