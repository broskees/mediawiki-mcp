package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetPageOutline retrieves page structure without full content
func GetPageOutline(ctx context.Context, client *wiki.Client, wikiURL, title string) (*wiki.PageOutline, error) {
	// Check cache
	cacheKey := wiki.PageCacheKey(wikiURL, title+":outline")
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.PageOutline), nil
	}

	// First, get the page structure (sections, categories, links) - NO section parameter
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", title)
	params.Set("prop", "sections|categories|links")
	params.Set("disableeditsection", "1")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get page outline: %w", err)
	}

	if resp.Parse == nil {
		return nil, fmt.Errorf("empty parse response")
	}

	// Now get the lead section content
	leadParams := url.Values{}
	leadParams.Set("action", "parse")
	leadParams.Set("page", title)
	leadParams.Set("prop", "text")
	leadParams.Set("section", "0")
	leadParams.Set("disableeditsection", "1")

	leadResp, err := client.MakeRequest(ctx, wikiURL, leadParams)
	if err != nil {
		return nil, fmt.Errorf("get lead section: %w", err)
	}

	// Convert lead section HTML to Markdown
	leadMarkdown, err := wiki.HTMLToMarkdown(leadResp.Parse.Text.Content)
	if err != nil {
		return nil, fmt.Errorf("convert lead to markdown: %w", err)
	}

	// Extract links from lead
	summaryLinks := wiki.ExtractLinks(leadResp.Parse.Text.Content)

	// Create summary (first paragraph)
	summary := wiki.ExtractPreview(leadMarkdown, 100)

	// Build sections tree
	sections := buildSectionsTree(resp.Parse.Sections, wikiURL, title, leadMarkdown)

	// Extract categories
	categories := make([]string, 0, len(resp.Parse.Categories))
	for _, cat := range resp.Parse.Categories {
		// Remove "Category:" prefix
		catName := strings.TrimPrefix(cat.Title, "Category:")
		categories = append(categories, catName)
	}

	// Extract "See also" links (these are typically at the end)
	seeAlso := extractSeeAlsoLinks(resp.Parse.Links)

	// Calculate total word count
	totalWords := wiki.CountWords(leadMarkdown)
	for _, section := range sections {
		totalWords += section.WordCount
		totalWords += countSubsectionWords(section)
	}

	// Get infobox from wikitext
	var infobox map[string]any
	if wikitext, err := getPageWikitext(ctx, client, wikiURL, title); err == nil {
		infobox = wiki.ExtractInfobox(wikitext)
	}

	// Build response
	outline := &wiki.PageOutline{
		Title:          resp.Parse.Title,
		Exists:         true,
		Summary:        summary,
		SummaryLinks:   summaryLinks,
		Infobox:        infobox,
		Sections:       sections,
		Categories:     categories,
		SeeAlso:        seeAlso,
		TotalWordCount: totalWords,
	}

	// Cache the result
	client.GetCache().Set(cacheKey, outline, client.GetCacheTTL())

	return outline, nil
}

// buildSectionsTree builds a hierarchical section structure
func buildSectionsTree(mwSections []wiki.MWSection, wikiURL, title, leadContent string) []*wiki.Section {
	if len(mwSections) == 0 {
		return []*wiki.Section{}
	}

	sections := make([]*wiki.Section, 0)

	// Add lead section
	leadSection := &wiki.Section{
		Index:     0,
		Title:     "Lead",
		Level:     1,
		Preview:   wiki.ExtractPreview(leadContent, 50),
		WordCount: wiki.CountWords(leadContent),
	}
	sections = append(sections, leadSection)

	// Build section hierarchy
	stack := []*wiki.Section{}

	for _, mwSec := range mwSections {
		index, _ := strconv.Atoi(mwSec.Index)
		level := mwSec.TocLevel

		section := &wiki.Section{
			Index:       index,
			Title:       mwSec.Line,
			Level:       level + 1, // Adjust level (+1 because lead is 1)
			Preview:     "",        // Will be filled if we fetch content
			WordCount:   0,         // Estimated or fetch later
			Subsections: []*wiki.Section{},
		}

		// Pop stack until we find the parent level
		for len(stack) > 0 && stack[len(stack)-1].Level >= section.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			// Top-level section
			sections = append(sections, section)
		} else {
			// Add as subsection to parent
			parent := stack[len(stack)-1]
			parent.Subsections = append(parent.Subsections, section)
		}

		stack = append(stack, section)
	}

	return sections
}

// countSubsectionWords recursively counts words in subsections
func countSubsectionWords(section *wiki.Section) int {
	count := 0
	for _, sub := range section.Subsections {
		count += sub.WordCount
		count += countSubsectionWords(sub)
	}
	return count
}

// extractSeeAlsoLinks extracts common "See also" links
func extractSeeAlsoLinks(links []wiki.MWLink) []string {
	// This is a simple heuristic - look for common related pages
	// In a real implementation, we'd parse the "See also" section
	seeAlso := make([]string, 0)
	seen := make(map[string]bool)

	for _, link := range links {
		title := link.Title

		// Skip common Wikipedia meta pages
		if strings.HasPrefix(title, "Category:") ||
			strings.HasPrefix(title, "File:") ||
			strings.HasPrefix(title, "Wikipedia:") ||
			strings.HasPrefix(title, "Template:") ||
			strings.HasPrefix(title, "Help:") {
			continue
		}

		// Avoid duplicates
		if seen[title] {
			continue
		}
		seen[title] = true

		seeAlso = append(seeAlso, title)

		// Limit to reasonable number
		if len(seeAlso) >= 10 {
			break
		}
	}

	return seeAlso
}

// getPageWikitext fetches raw wikitext for a page (for infobox extraction)
func getPageWikitext(ctx context.Context, client *wiki.Client, wikiURL, title string) (string, error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("titles", title)
	params.Set("prop", "revisions")
	params.Set("rvprop", "content")
	params.Set("rvslots", "main")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return "", err
	}

	if resp.Query == nil || len(resp.Query.Pages) == 0 {
		return "", fmt.Errorf("no pages found")
	}

	// Get the first (and only) page
	for _, page := range resp.Query.Pages {
		if len(page.Revisions) > 0 {
			return page.Revisions[0].Content, nil
		}
	}

	return "", fmt.Errorf("no revisions found")
}
