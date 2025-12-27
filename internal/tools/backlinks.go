package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetBacklinks retrieves pages that link to a given page
func GetBacklinks(ctx context.Context, client *wiki.Client, wikiURL, title string, limit int) (*wiki.BacklinksResponse, error) {
	// Check cache
	cacheKey := wiki.BacklinksCacheKey(wikiURL, title+":"+strconv.Itoa(limit))
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.BacklinksResponse), nil
	}

	// Build API request
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "backlinks")
	params.Set("bltitle", title)
	params.Set("bllimit", strconv.Itoa(limit))

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get backlinks: %w", err)
	}

	if resp.Query == nil {
		return nil, fmt.Errorf("empty query response")
	}

	// Build backlinks list
	backlinks := make([]wiki.Backlink, 0, len(resp.Query.Backlinks))
	for _, bl := range resp.Query.Backlinks {
		backlinks = append(backlinks, wiki.Backlink{
			Title: bl.Title,
		})
	}

	// Build response
	backlinksResp := &wiki.BacklinksResponse{
		Title:      title,
		Backlinks:  backlinks,
		TotalCount: len(backlinks),
	}

	// Cache the result
	client.GetCache().Set(cacheKey, backlinksResp, client.GetCacheTTL())

	return backlinksResp, nil
}
