package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// GetPageSection retrieves a specific section of a page
func GetPageSection(ctx context.Context, client *wiki.Client, wikiURL, title string, sectionIndex int) (*wiki.PageSection, error) {
	// Check cache
	cacheKey := wiki.SectionCacheKey(wikiURL, title, strconv.Itoa(sectionIndex))
	if cached, ok := client.GetCache().Get(cacheKey); ok {
		return cached.(*wiki.PageSection), nil
	}

	// First, get the page structure to validate section and get context
	outline, err := GetPageOutline(ctx, client, wikiURL, title)
	if err != nil {
		return nil, fmt.Errorf("get page outline: %w", err)
	}

	// Find the requested section
	var targetSection *wiki.Section
	var parentSection *wiki.Section
	var prevSection *wiki.Section
	var nextSection *wiki.Section

	// Flatten sections to find the target
	flatSections := flattenSections(outline.Sections)

	for i, sec := range flatSections {
		if sec.Index == sectionIndex {
			targetSection = sec

			// Find parent (section with lower level before this one)
			for j := i - 1; j >= 0; j-- {
				if flatSections[j].Level < sec.Level {
					parentSection = flatSections[j]
					break
				}
			}

			// Previous section (same or higher level)
			if i > 0 {
				prevSection = flatSections[i-1]
			}

			// Next section
			if i < len(flatSections)-1 {
				nextSection = flatSections[i+1]
			}

			break
		}
	}

	if targetSection == nil {
		return nil, &SectionNotFoundError{
			SectionIndex:      sectionIndex,
			AvailableSections: len(flatSections),
		}
	}

	// Fetch the section content
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", title)
	params.Set("section", strconv.Itoa(sectionIndex))
	params.Set("prop", "text|links")
	params.Set("disableeditsection", "1")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("get section: %w", err)
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

	// Build the section with content
	section := &wiki.Section{
		Index:     targetSection.Index,
		Title:     targetSection.Title,
		Level:     targetSection.Level,
		Content:   markdown,
		Links:     links,
		WordCount: wiki.CountWords(markdown),
	}

	// Build response
	pageSection := &wiki.PageSection{
		Title:   title,
		Section: section,
	}

	// Add parent info
	if parentSection != nil {
		pageSection.ParentSection = &struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		}{
			Index: parentSection.Index,
			Title: parentSection.Title,
		}
	}

	// Add adjacent sections
	pageSection.Adjacent = &struct {
		Previous *struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		} `json:"previous,omitempty"`
		Next *struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		} `json:"next,omitempty"`
	}{}

	if prevSection != nil {
		pageSection.Adjacent.Previous = &struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		}{
			Index: prevSection.Index,
			Title: prevSection.Title,
		}
	}

	if nextSection != nil {
		pageSection.Adjacent.Next = &struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		}{
			Index: nextSection.Index,
			Title: nextSection.Title,
		}
	}

	// Cache the result
	client.GetCache().Set(cacheKey, pageSection, client.GetCacheTTL())

	return pageSection, nil
}

// flattenSections converts a tree of sections to a flat list
func flattenSections(sections []*wiki.Section) []*wiki.Section {
	result := make([]*wiki.Section, 0)

	var flatten func([]*wiki.Section)
	flatten = func(secs []*wiki.Section) {
		for _, sec := range secs {
			result = append(result, sec)
			if len(sec.Subsections) > 0 {
				flatten(sec.Subsections)
			}
		}
	}

	flatten(sections)
	return result
}

// SectionNotFoundError represents an error when a section doesn't exist
type SectionNotFoundError struct {
	SectionIndex      int
	AvailableSections int
}

func (e *SectionNotFoundError) Error() string {
	return fmt.Sprintf("section index %d does not exist (page has %d sections)", e.SectionIndex, e.AvailableSections)
}
