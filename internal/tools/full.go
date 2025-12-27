package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetPageFull retrieves the entire content of a page
func GetPageFull(ctx context.Context, client *wiki.Client, wikiURL, title string) (*wiki.PageFull, error) {
	// Check cache
	cacheKey := wiki.PageCacheKey(wikiURL, title)
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.PageFull), nil
	}

	// Build API request
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", title)
	params.Set("prop", "text|links")
	params.Set("disableeditsection", "1")
	params.Set("disabletoc", "1")

	// Make request
	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get page full: %w", err)
	}

	if resp.Parse == nil {
		return nil, fmt.Errorf("empty parse response")
	}

	// Convert HTML to Markdown
	markdown, err := wiki.HTMLToMarkdown(resp.Parse.Text.Content)
	if err != nil {
		return nil, fmt.Errorf("convert to markdown: %w", err)
	}

	// Extract links
	links := make([]string, 0, len(resp.Parse.Links))
	for _, link := range resp.Parse.Links {
		links = append(links, link.Title)
	}

	// Count words
	wordCount := wiki.CountWords(markdown)

	// Build response
	pageFull := &wiki.PageFull{
		Title:     resp.Parse.Title,
		Content:   markdown,
		Links:     links,
		WordCount: wordCount,
	}

	// Add warning for large pages
	if wordCount > 5000 {
		warning := fmt.Sprintf("Large page (%d words). Consider using wiki_page_outline + wiki_page_section for targeted retrieval.", wordCount)
		pageFull.Warning = &warning
	}

	// Cache the result
	client.GetCache().Set(cacheKey, pageFull, client.GetCacheTTL())

	return pageFull, nil
}
