package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetCategory retrieves pages in a category
func GetCategory(ctx context.Context, client *wiki.Client, wikiURL, category string, limit int) (*wiki.CategoryResponse, error) {
	// Check cache
	cacheKey := wiki.CategoryCacheKey(wikiURL, category+":"+strconv.Itoa(limit))
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.CategoryResponse), nil
	}

	// Ensure category has "Category:" prefix
	if !strings.HasPrefix(category, "Category:") {
		category = "Category:" + category
	}

	// Build API request for category members
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "categorymembers")
	params.Set("cmtitle", category)
	params.Set("cmlimit", strconv.Itoa(limit))
	params.Set("cmprop", "title|type")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}

	if resp.Query == nil {
		return nil, fmt.Errorf("empty query response")
	}

	// Build members list
	members := make([]wiki.CategoryMember, 0, len(resp.Query.Categorymembers))
	for _, member := range resp.Query.Categorymembers {
		memberType := "page"
		if member.Type == "subcat" {
			memberType = "subcat"
		}

		members = append(members, wiki.CategoryMember{
			Title: member.Title,
			Type:  memberType,
		})
	}

	// Get parent categories
	parentCategories, err := getParentCategories(ctx, client, wikiURL, category)
	if err != nil {
		// Non-fatal, continue without parent categories
		parentCategories = []string{}
	}

	// Build response
	categoryResp := &wiki.CategoryResponse{
		Category:         strings.TrimPrefix(category, "Category:"),
		Members:          members,
		ParentCategories: parentCategories,
		TotalMembers:     len(members),
	}

	// Cache the result
	client.GetCache().Set(cacheKey, categoryResp, client.GetCacheTTL())

	return categoryResp, nil
}

// getParentCategories retrieves parent categories for a given category
func getParentCategories(ctx context.Context, client *wiki.Client, wikiURL, category string) ([]string, error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("titles", category)
	params.Set("prop", "categories")
	params.Set("cllimit", "10")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, err
	}

	if resp.Query == nil || len(resp.Query.Pages) == 0 {
		return []string{}, nil
	}

	parents := make([]string, 0)
	for _, page := range resp.Query.Pages {
		for _, cat := range page.Categories {
			catName := strings.TrimPrefix(cat.Title, "Category:")
			parents = append(parents, catName)
		}
	}

	return parents, nil
}
