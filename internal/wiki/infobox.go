package wiki

import (
	"regexp"
	"strings"
)

// ExtractInfobox extracts infobox data from wikitext
func ExtractInfobox(wikitext string) map[string]any {
	// Find the first infobox template
	infoboxRegex := regexp.MustCompile(`(?s)\{\{Infobox[^\}]*?\n(.*?)\n\}\}`)
	matches := infoboxRegex.FindStringSubmatch(wikitext)

	if len(matches) < 2 {
		return nil
	}

	infoboxContent := matches[1]

	// Parse key-value pairs
	result := make(map[string]any)

	// Split by lines starting with |
	lines := strings.Split(infoboxContent, "\n")

	var currentKey string
	var currentValue strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if this is a new key-value pair
		if strings.HasPrefix(line, "|") {
			// Save previous key-value if exists
			if currentKey != "" {
				result[currentKey] = cleanInfoboxValue(currentValue.String())
			}

			// Parse new key-value
			line = strings.TrimPrefix(line, "|")
			parts := strings.SplitN(line, "=", 2)

			if len(parts) == 2 {
				currentKey = strings.TrimSpace(parts[0])
				currentValue.Reset()
				currentValue.WriteString(strings.TrimSpace(parts[1]))
			} else {
				// Line without = is a continuation
				if currentKey != "" {
					currentValue.WriteString(" ")
					currentValue.WriteString(line)
				}
			}
		} else {
			// Continuation of previous value
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(line)
			}
		}
	}

	// Save last key-value
	if currentKey != "" {
		result[currentKey] = cleanInfoboxValue(currentValue.String())
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// cleanInfoboxValue cleans up wikitext in infobox values
func cleanInfoboxValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove wiki links but keep the display text
	// [[Link]] -> Link
	// [[Link|Display]] -> Display
	linkRegex := regexp.MustCompile(`\[\[([^\|\]]+)(?:\|([^\]]+))?\]\]`)
	value = linkRegex.ReplaceAllStringFunc(value, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		if len(parts) > 2 && parts[2] != "" {
			return parts[2]
		}
		return parts[1]
	})

	// Handle common templates
	value = cleanCommonTemplates(value)

	// Remove remaining template syntax (simple approach)
	templateRegex := regexp.MustCompile(`\{\{[^\}]+\}\}`)
	value = templateRegex.ReplaceAllString(value, "")

	// Clean up HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]+>`)
	value = htmlRegex.ReplaceAllString(value, "")

	// Remove formatting markup
	value = strings.ReplaceAll(value, "'''", "")
	value = strings.ReplaceAll(value, "''", "")

	// Clean up whitespace
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")

	return value
}

// cleanCommonTemplates handles common MediaWiki templates
func cleanCommonTemplates(value string) string {
	// {{birth date|1879|3|14}} -> 1879-03-14
	birthDateRegex := regexp.MustCompile(`\{\{birth date\|(\d+)\|(\d+)\|(\d+)[^\}]*\}\}`)
	value = birthDateRegex.ReplaceAllString(value, "$1-$2-$3")

	// {{death date|1955|4|18}} -> 1955-04-18
	deathDateRegex := regexp.MustCompile(`\{\{death date\|(\d+)\|(\d+)\|(\d+)[^\}]*\}\}`)
	value = deathDateRegex.ReplaceAllString(value, "$1-$2-$3")

	// {{age|1879|3|14}} -> remove (age calculation not useful)
	ageRegex := regexp.MustCompile(`\{\{age\|[^\}]+\}\}`)
	value = ageRegex.ReplaceAllString(value, "")

	// {{circa|1900}} -> circa 1900
	circaRegex := regexp.MustCompile(`\{\{circa\|([^\}]+)\}\}`)
	value = circaRegex.ReplaceAllString(value, "circa $1")

	// {{flag|USA}} -> USA
	flagRegex := regexp.MustCompile(`\{\{flag\|([^\}]+)\}\}`)
	value = flagRegex.ReplaceAllString(value, "$1")

	// {{coord|...}} -> remove (coordinates not useful in text)
	coordRegex := regexp.MustCompile(`\{\{coord\|[^\}]+\}\}`)
	value = coordRegex.ReplaceAllString(value, "")

	return value
}

// ExtractInfoboxFromHTML extracts infobox from parsed HTML
func ExtractInfoboxFromHTML(html string) map[string]any {
	// MediaWiki renders infoboxes as tables with class "infobox"
	// This is more reliable than parsing wikitext
	// We'll use goquery for this

	// For now, return nil - we'll implement HTML parsing if needed
	// The wikitext approach above should work for most cases
	return nil
}
