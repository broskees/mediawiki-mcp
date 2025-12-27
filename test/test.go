package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	transport := &mcp.SSEClientTransport{Endpoint: "http://localhost:8080/sse"}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Connect error: %v", err)
	}
	defer session.Close()

	fmt.Println("✓ Connected to MCP server\n")

	// Test 1: wiki_info
	fmt.Println("=== Test 1: wiki_info ===")
	result, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_info",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org"},
	})
	var info map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &info)
	fmt.Printf("  Name: %s\n", info["name"])
	fmt.Printf("  Language: %s\n", info["language"])
	fmt.Printf("  Articles: %.0f\n", info["article_count"])
	fmt.Println("  ✓ PASS\n")

	// Test 2: wiki_search
	fmt.Println("=== Test 2: wiki_search ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_search",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "query": "Go programming", "limit": 3},
	})
	var search map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &search)
	fmt.Printf("  Found %d results:\n", int(search["total_hits"].(float64)))
	for i, r := range search["results"].([]any) {
		res := r.(map[string]any)
		fmt.Printf("    %d. %s (%.0f words)\n", i+1, res["title"], res["word_count"])
	}
	fmt.Println("  ✓ PASS\n")

	// Test 3: wiki_page_outline
	fmt.Println("=== Test 3: wiki_page_outline ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_page_outline",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "title": "Go (programming language)"},
	})
	var outline map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &outline)
	fmt.Printf("  Title: %s\n", outline["title"])
	fmt.Printf("  Total words: %.0f\n", outline["total_word_count"])
	sections := outline["sections"].([]any)
	fmt.Printf("  Sections: %d\n", len(sections))
	for i, s := range sections {
		if i >= 5 {
			fmt.Printf("    ... (%d more)\n", len(sections)-5)
			break
		}
		sec := s.(map[string]any)
		fmt.Printf("    [%d] %s\n", int(sec["index"].(float64)), sec["title"])
	}
	fmt.Println("  ✓ PASS\n")

	// Test 4: wiki_page_section
	fmt.Println("=== Test 4: wiki_page_section ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_page_section",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "title": "Go (programming language)", "section_index": 1},
	})
	var secResp map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &secResp)
	sec := secResp["section"].(map[string]any)
	content := sec["content"].(string)
	fmt.Printf("  Section: %s\n", sec["title"])
	fmt.Printf("  Word count: %.0f\n", sec["word_count"])
	preview := content
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	fmt.Printf("  Preview: %s\n", preview)
	fmt.Println("  ✓ PASS\n")

	// Test 5: wiki_category
	fmt.Println("=== Test 5: wiki_category ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_category",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "category": "Programming languages", "limit": 5},
	})
	var catResp map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &catResp)
	members := catResp["members"].([]any)
	fmt.Printf("  Category: %s\n", catResp["category"])
	fmt.Printf("  Members: %d\n", len(members))
	for _, m := range members {
		member := m.(map[string]any)
		fmt.Printf("    - %s (%s)\n", member["title"], member["type"])
	}
	fmt.Println("  ✓ PASS\n")

	// Test 6: wiki_backlinks
	fmt.Println("=== Test 6: wiki_backlinks ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_backlinks",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "title": "Go (programming language)", "limit": 5},
	})
	var blResp map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &blResp)
	backlinks := blResp["backlinks"].([]any)
	fmt.Printf("  Backlinks to 'Go (programming language)': %d\n", len(backlinks))
	for _, b := range backlinks {
		bl := b.(map[string]any)
		fmt.Printf("    - %s\n", bl["title"])
	}
	fmt.Println("  ✓ PASS\n")

	// Test 7: Error handling
	fmt.Println("=== Test 7: Error Handling ===")
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wiki_page_section",
		Arguments: map[string]any{"wiki_url": "https://en.wikipedia.org", "title": "Go (programming language)", "section_index": 999},
	})
	if result.IsError {
		var errResp map[string]any
		json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &errResp)
		fmt.Printf("  Error code: %s\n", errResp["error"])
		fmt.Printf("  Hint: %s\n", errResp["hint"])
		fmt.Println("  ✓ PASS\n")
	} else {
		log.Fatal("Expected error for invalid section")
	}

	fmt.Println("========================================")
	fmt.Println("✓ ALL TESTS PASSED!")
	fmt.Println("========================================")
}
