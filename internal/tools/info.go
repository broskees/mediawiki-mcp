package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetWikiInfo retrieves metadata about a wiki
func GetWikiInfo(ctx context.Context, client *wiki.Client, wikiURL string) (*wiki.WikiInfo, error) {
	// Check cache
	cacheKey := wiki.InfoCacheKey(wikiURL)
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.WikiInfo), nil
	}

	// Build API request
	params := url.Values{}
	params.Set("action", "query")
	params.Set("meta", "siteinfo")
	params.Set("siprop", "general|namespaces|statistics")

	// Make request
	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get wiki info: %w", err)
	}

	if resp.Query == nil {
		return nil, fmt.Errorf("empty query response")
	}

	// Build response
	info := &wiki.WikiInfo{
		BaseURL:    wikiURL,
		Namespaces: make(map[string]string),
	}

	if resp.Query.General != nil {
		info.Name = resp.Query.General.Sitename
		info.MainPage = resp.Query.General.MainPage
		info.Language = resp.Query.General.Lang
	}

	if resp.Query.Statistics != nil {
		info.ArticleCount = resp.Query.Statistics.Articles
	}

	// Extract namespaces
	for _, ns := range resp.Query.Namespaces {
		info.Namespaces[strconv.Itoa(ns.ID)] = ns.Name
	}

	// Cache the result
	client.GetCache().Set(cacheKey, info, client.GetCacheTTLInfo())

	return info, nil
}
