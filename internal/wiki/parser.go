package wiki

import (
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

var (
	// Converter is a shared HTML to Markdown converter
	converter *md.Converter
)

func init() {
	// Initialize converter with MediaWiki-friendly options
	converter = md.NewConverter("", true, &md.Options{
		HeadingStyle:     "atx", // Use # style headings
		HorizontalRule:   "---",
		BulletListMarker: "-",
		CodeBlockStyle:   "fenced", // Use ``` for code blocks
		StrongDelimiter:  "**",
		EmDelimiter:      "*",
	})

	// Add custom rules for MediaWiki-specific elements
	converter.AddRules(
		// Remove edit section links
		md.Rule{
			Filter: []string{"span"},
			AdvancedReplacement: func(content string, selec *goquery.Selection, opt *md.Options) (md.AdvancedResult, bool) {
				if selec.HasClass("mw-editsection") {
					return md.AdvancedResult{Markdown: ""}, true
				}
				return md.AdvancedResult{}, false
			},
		},
		// Clean up reference markers
		md.Rule{
			Filter: []string{"sup"},
			AdvancedReplacement: func(content string, selec *goquery.Selection, opt *md.Options) (md.AdvancedResult, bool) {
				if selec.HasClass("reference") {
					// Keep reference numbers in a cleaner format
					text := selec.Text()
					return md.AdvancedResult{Markdown: "[" + text + "]"}, true
				}
				return md.AdvancedResult{}, false
			},
		},
	)
}

// HTMLToMarkdown converts MediaWiki HTML to Markdown
func HTMLToMarkdown(html string) (string, error) {
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Clean up the markdown
	markdown = cleanupMarkdown(markdown)

	return markdown, nil
}

// cleanupMarkdown performs post-conversion cleanup
func cleanupMarkdown(md string) string {
	// Remove excessive newlines (more than 2 consecutive)
	md = regexp.MustCompile(`\n{3,}`).ReplaceAllString(md, "\n\n")

	// Trim whitespace
	md = strings.TrimSpace(md)

	return md
}

// ExtractLinks extracts all links from HTML
func ExtractLinks(html string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	links := make([]string, 0)
	seen := make(map[string]bool)

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Extract page title from href
		title := extractTitleFromHref(href)
		if title == "" {
			return
		}

		// Deduplicate
		if !seen[title] {
			seen[title] = true
			links = append(links, title)
		}
	})

	return links
}

// extractTitleFromHref extracts the page title from a MediaWiki href
func extractTitleFromHref(href string) string {
	// MediaWiki links are typically /wiki/Page_Title or /w/index.php?title=Page_Title
	if strings.HasPrefix(href, "/wiki/") {
		title := strings.TrimPrefix(href, "/wiki/")
		// Remove anchor
		if idx := strings.Index(title, "#"); idx != -1 {
			title = title[:idx]
		}
		return decodeTitle(title)
	}

	// Handle /w/index.php?title=Page_Title format
	if strings.Contains(href, "title=") {
		parts := strings.Split(href, "title=")
		if len(parts) > 1 {
			title := parts[1]
			// Remove other query params
			if idx := strings.Index(title, "&"); idx != -1 {
				title = title[:idx]
			}
			// Remove anchor
			if idx := strings.Index(title, "#"); idx != -1 {
				title = title[:idx]
			}
			return decodeTitle(title)
		}
	}

	return ""
}

// decodeTitle converts URL-encoded titles to readable format
func decodeTitle(title string) string {
	// Replace underscores with spaces
	title = strings.ReplaceAll(title, "_", " ")
	return title
}

// CountWords counts words in text
func CountWords(text string) int {
	// Remove markdown formatting for more accurate count
	text = stripMarkdownFormatting(text)

	// Split on whitespace
	words := strings.Fields(text)
	return len(words)
}

// stripMarkdownFormatting removes markdown syntax for word counting
func stripMarkdownFormatting(text string) string {
	// Remove bold/italic markers
	text = regexp.MustCompile(`\*+`).ReplaceAllString(text, "")

	// Remove links but keep text [text](url) -> text
	text = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`).ReplaceAllString(text, "$1")

	// Remove headers
	text = regexp.MustCompile(`^#+\s+`).ReplaceAllString(text, "")

	// Remove code blocks
	text = regexp.MustCompile("```[^`]*```").ReplaceAllString(text, "")

	// Remove inline code
	text = regexp.MustCompile("`[^`]+`").ReplaceAllString(text, "")

	return text
}

// ExtractPreview extracts the first N words from markdown as a preview
func ExtractPreview(markdown string, maxWords int) string {
	// Strip formatting
	text := stripMarkdownFormatting(markdown)

	// Split into words
	words := strings.Fields(text)

	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}

	preview := strings.Join(words[:maxWords], " ")
	return preview + "..."
}
