package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// SearchWiki searches for pages by keyword
func SearchWiki(ctx context.Context, client *wiki.Client, wikiURL, query string, limit int) (*wiki.SearchResponse, error) {
	// Check cache
	cacheKey := wiki.SearchCacheKey(wikiURL, query+":"+strconv.Itoa(limit))
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.SearchResponse), nil
	}

	// Build API request
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", query)
	params.Set("srlimit", strconv.Itoa(limit))
	params.Set("srprop", "snippet|wordcount")

	// Make request
	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("search wiki: %w", err)
	}

	if resp.Query == nil {
		return nil, fmt.Errorf("empty query response")
	}

	// Build response
	searchResp := &wiki.SearchResponse{
		Results:   make([]wiki.SearchResult, 0, len(resp.Query.Search)),
		TotalHits: len(resp.Query.Search),
	}

	for _, result := range resp.Query.Search {
		// Convert HTML snippet to markdown
		markdown, err := wiki.HTMLToMarkdown(result.Snippet)
		if err != nil {
			markdown = result.Snippet // fallback to raw HTML
		}

		// Extract links from snippet
		links := wiki.ExtractLinks(result.Snippet)

		searchResp.Results = append(searchResp.Results, wiki.SearchResult{
			Title:        result.Title,
			Snippet:      markdown,
			SnippetLinks: links,
			WordCount:    result.WordCount,
		})
	}

	// Add suggestion if available
	if resp.Query.SearchInfo != nil && resp.Query.SearchInfo.Suggestion != "" {
		searchResp.Suggestion = &resp.Query.SearchInfo.Suggestion
	}

	// Cache the result (short TTL for search)
	client.GetCache().Set(cacheKey, searchResp, 1*60) // 1 minute

	return searchResp, nil
}
