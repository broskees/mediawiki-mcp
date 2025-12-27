package wiki

import (
	"encoding/json"
	"fmt"
	"time"
)

// WikiInfo contains metadata about a wiki
type WikiInfo struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	MainPage     string            `json:"main_page"`
	Language     string            `json:"language"`
	ArticleCount int               `json:"article_count"`
	Namespaces   map[string]string `json:"namespaces"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title        string   `json:"title"`
	Snippet      string   `json:"snippet"`
	SnippetLinks []string `json:"snippet_links"`
	WordCount    int      `json:"word_count"`
}

// SearchResponse contains search results
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	TotalHits  int            `json:"total_hits"`
	Suggestion *string        `json:"suggestion,omitempty"`
}

// Section represents a page section
type Section struct {
	Index       int        `json:"index"`
	Title       string     `json:"title"`
	Level       int        `json:"level"`
	Preview     string     `json:"preview,omitempty"`
	Content     string     `json:"content,omitempty"`
	Links       []string   `json:"links,omitempty"`
	WordCount   int        `json:"word_count"`
	Subsections []*Section `json:"subsections,omitempty"`
}

// PageOutline contains page structure without full content
type PageOutline struct {
	Title          string                 `json:"title"`
	Exists         bool                   `json:"exists"`
	Redirect       *string                `json:"redirect,omitempty"`
	Summary        string                 `json:"summary"`
	SummaryLinks   []string               `json:"summary_links"`
	Infobox        map[string]interface{} `json:"infobox,omitempty"`
	Sections       []*Section             `json:"sections"`
	Categories     []string               `json:"categories"`
	SeeAlso        []string               `json:"see_also"`
	TotalWordCount int                    `json:"total_word_count"`
}

// PageSection contains full content of a specific section
type PageSection struct {
	Title         string   `json:"title"`
	Section       *Section `json:"section"`
	ParentSection *struct {
		Index int    `json:"index"`
		Title string `json:"title"`
	} `json:"parent_section,omitempty"`
	Adjacent *struct {
		Previous *struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		} `json:"previous,omitempty"`
		Next *struct {
			Index int    `json:"index"`
			Title string `json:"title"`
		} `json:"next,omitempty"`
	} `json:"adjacent,omitempty"`
}

// PageFull contains entire page content
type PageFull struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Links     []string `json:"links"`
	WordCount int      `json:"word_count"`
	Warning   *string  `json:"warning,omitempty"`
}

// CategoryMember represents a member of a category
type CategoryMember struct {
	Title string `json:"title"`
	Type  string `json:"type"` // "page" or "subcat"
}

// CategoryResponse contains category information
type CategoryResponse struct {
	Category         string           `json:"category"`
	Members          []CategoryMember `json:"members"`
	ParentCategories []string         `json:"parent_categories,omitempty"`
	TotalMembers     int              `json:"total_members"`
	ContinueToken    *string          `json:"continue_token,omitempty"`
}

// Backlink represents a page that links to another
type Backlink struct {
	Title string `json:"title"`
}

// BacklinksResponse contains backlinks information
type BacklinksResponse struct {
	Title         string     `json:"title"`
	Backlinks     []Backlink `json:"backlinks"`
	TotalCount    int        `json:"total_count"`
	ContinueToken *string    `json:"continue_token,omitempty"`
}

// RevisionInfo contains information about a revision
type RevisionInfo struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user"`
}

// CompareResponse contains revision comparison
type CompareResponse struct {
	Title        string       `json:"title"`
	From         RevisionInfo `json:"from"`
	To           RevisionInfo `json:"to"`
	DiffSummary  string       `json:"diff_summary"`
	DiffMarkdown string       `json:"diff_markdown"`
}

// MediaWiki API response structures (internal use)

type mwResponse struct {
	Query   *mwQuery   `json:"query"`
	Parse   *mwParse   `json:"parse"`
	Compare *mwCompare `json:"compare"`
	Error   *mwError   `json:"error"`
}

type mwQuery struct {
	General         *mwGeneral             `json:"general"`
	Namespaces      map[string]mwNamespace `json:"namespaces"`
	Statistics      *mwStatistics          `json:"statistics"`
	Search          []mwSearchResult       `json:"search"`
	SearchInfo      *mwSearchInfo          `json:"searchinfo"`
	Pages           map[string]mwPage      `json:"pages"`
	Backlinks       []mwBacklink           `json:"backlinks"`
	Categorymembers []mwCategoryMember     `json:"categorymembers"`
}

type mwGeneral struct {
	Sitename string `json:"sitename"`
	Base     string `json:"base"`
	MainPage string `json:"mainpage"`
	Lang     string `json:"lang"`
}

type mwNamespace struct {
	ID   int    `json:"id"`
	Name string `json:"*"`
}

type mwStatistics struct {
	Articles int `json:"articles"`
}

type mwSearchResult struct {
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	WordCount int    `json:"wordcount"`
}

type mwSearchInfo struct {
	Suggestion string `json:"suggestion"`
}

type mwPage struct {
	PageID     int          `json:"pageid"`
	Title      string       `json:"title"`
	Missing    bool         `json:"missing"`
	Redirect   bool         `json:"redirect"`
	Revisions  []mwRevision `json:"revisions"`
	Categories []mwCategory `json:"categories"`
	Links      []MWLink     `json:"links"`
}

type mwRevision struct {
	Content string `json:"*"`
}

type mwCategory struct {
	Title string `json:"title"`
}

// MWLink represents a MediaWiki link (exported for use in tools)
type MWLink struct {
	Title string `json:"title"`
}

type mwParse struct {
	Title      string       `json:"title"`
	PageID     int          `json:"pageid"`
	Text       mwText       `json:"text"`
	Sections   []MWSection  `json:"sections"`
	Categories []mwCategory `json:"categories"`
	Links      []MWLink     `json:"links"`
	Properties mwProperties `json:"properties,omitempty"`
}

type mwText struct {
	Content string
}

// UnmarshalJSON handles both string and object formats for text
func (t *mwText) UnmarshalJSON(data []byte) error {
	// Try as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Content = s
		return nil
	}

	// Try as object with * field
	var obj struct {
		Content string `json:"*"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		t.Content = obj.Content
		return nil
	}

	return fmt.Errorf("text must be string or object with * field")
}

// MWSection represents a MediaWiki section (exported for use in tools)
type MWSection struct {
	TocLevel int    `json:"toclevel"`
	Level    string `json:"level"`
	Line     string `json:"line"`
	Number   string `json:"number"`
	Index    string `json:"index"`
}

type mwProperties struct {
	WikibaseItem string `json:"wikibase_item"`
}

type mwBacklink struct {
	PageID int    `json:"pageid"`
	Title  string `json:"title"`
}

type mwCategoryMember struct {
	PageID int    `json:"pageid"`
	Title  string `json:"title"`
	Type   string `json:"type"`
}

type mwCompare struct {
	FromID    int    `json:"fromid"`
	FromRevID int    `json:"fromrevid"`
	ToID      int    `json:"toid"`
	ToRevID   int    `json:"torevid"`
	Body      string `json:"*"`
}

type mwError struct {
	Code string `json:"code"`
	Info string `json:"info"`
}
