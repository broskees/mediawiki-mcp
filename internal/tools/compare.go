package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// CompareRevisions compares two revisions of a page
func CompareRevisions(ctx context.Context, client *wiki.Client, wikiURL, title, fromRev, toRev string) (*wiki.CompareResponse, error) {
	// Build API request
	params := url.Values{}
	params.Set("action", "compare")
	params.Set("fromtitle", title)

	// Handle special revision specifiers
	if fromRev == "prev" || fromRev == "current" {
		params.Set("fromrelative", fromRev)
	} else {
		params.Set("fromrev", fromRev)
	}

	if toRev == "prev" || toRev == "current" || toRev == "next" {
		params.Set("torelative", toRev)
	} else {
		params.Set("torev", toRev)
	}

	params.Set("prop", "diff|ids|timestamp|user|comment")

	resp, err := client.MakeRequest(ctx, wikiURL, params)
	if err != nil {
		return nil, fmt.Errorf("compare revisions: %w", err)
	}

	if resp.Compare == nil {
		return nil, fmt.Errorf("empty compare response")
	}

	// Convert HTML diff to markdown (simplified)
	diffMarkdown, err := wiki.HTMLToMarkdown(resp.Compare.Body)
	if err != nil {
		diffMarkdown = resp.Compare.Body // Fallback to raw HTML
	}

	// Build response
	compareResp := &wiki.CompareResponse{
		Title: title,
		From: wiki.RevisionInfo{
			ID: resp.Compare.FromRevID,
			// Note: timestamp and user would need additional API call
		},
		To: wiki.RevisionInfo{
			ID: resp.Compare.ToRevID,
		},
		DiffSummary:  "Changes between revisions",
		DiffMarkdown: diffMarkdown,
	}

	return compareResp, nil
}
