package mcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yourusername/mediawiki-mcp/config"
	"github.com/yourusername/mediawiki-mcp/internal/tools"
	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// Server wraps the MCP server with our wiki client
type Server struct {
	mcp    *mcp.Server
	client *wiki.Client
	config *config.Config
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		client: wiki.NewClient(
			cfg.UserAgent,
			cfg.RequestTimeout,
			cfg.RateLimit,
			cfg.CacheTTL,
			cfg.CacheTTLInfo,
		),
	}

	// Create MCP server
	impl := &mcp.Implementation{
		Name:    "mediawiki-mcp",
		Version: "1.0.0",
	}

	s.mcp = mcp.NewServer(impl, nil)

	// Register tools
	s.registerTools()

	return s
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcp
}

// registerTools registers all tools with the MCP server
func (s *Server) registerTools() {
	// wiki_info
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_info",
		Description: "Get metadata about a MediaWiki site including name, language, article count, and namespaces",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki (e.g. 'https://en.wikipedia.org')"
				}
			},
			"required": ["wiki_url"]
		}`),
	}, s.handleWikiInfo)

	// wiki_search
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_search",
		Description: "Search a MediaWiki site for pages matching a query. Returns titles, snippets, and page metadata",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"query": {
					"type": "string",
					"description": "Search terms"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of results (default: 10)",
					"default": 10
				}
			},
			"required": ["wiki_url", "query"]
		}`),
	}, s.handleWikiSearch)

	// wiki_page_outline
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_page_outline",
		Description: "Get page structure with section tree, summary, infobox, and metadata. Use this before fetching full content to understand page organization",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"title": {
					"type": "string",
					"description": "Page title"
				}
			},
			"required": ["wiki_url", "title"]
		}`),
	}, s.handlePageOutline)

	// wiki_page_section
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_page_section",
		Description: "Get full content of a specific page section by index. If section index is invalid, an error will suggest calling wiki_page_outline to get fresh indices",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"title": {
					"type": "string",
					"description": "Page title"
				},
				"section_index": {
					"type": "integer",
					"description": "Section index from wiki_page_outline"
				}
			},
			"required": ["wiki_url", "title", "section_index"]
		}`),
	}, s.handlePageSection)

	// wiki_page_full
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_page_full",
		Description: "Get entire page content. Warning: may be large. Consider using wiki_page_outline + wiki_page_section for targeted retrieval",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"title": {
					"type": "string",
					"description": "Page title"
				}
			},
			"required": ["wiki_url", "title"]
		}`),
	}, s.handlePageFull)

	// wiki_category
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_category",
		Description: "Get pages and subcategories within a category",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"category": {
					"type": "string",
					"description": "Category name (with or without 'Category:' prefix)"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of results (default: 20)",
					"default": 20
				}
			},
			"required": ["wiki_url", "category"]
		}`),
	}, s.handleCategory)

	// wiki_backlinks
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_backlinks",
		Description: "Find pages that link to a given page",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"title": {
					"type": "string",
					"description": "Page title to find backlinks for"
				},
				"limit": {
					"type": "integer",
					"description": "Maximum number of results (default: 20)",
					"default": 20
				}
			},
			"required": ["wiki_url", "title"]
		}`),
	}, s.handleBacklinks)

	// wiki_compare
	s.mcp.AddTool(&mcp.Tool{
		Name:        "wiki_compare",
		Description: "Compare two revisions of a page to see what changed",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"wiki_url": {
					"type": "string",
					"description": "Base URL of the wiki"
				},
				"title": {
					"type": "string",
					"description": "Page title"
				},
				"from_revision": {
					"type": "string",
					"description": "Starting revision ('prev' or revision ID)",
					"default": "prev"
				},
				"to_revision": {
					"type": "string",
					"description": "Ending revision ('current', 'next', or revision ID)",
					"default": "current"
				}
			},
			"required": ["wiki_url", "title"]
		}`),
	}, s.handleCompare)
}

// Tool handlers

func (s *Server) handleWikiInfo(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL string `json:"wiki_url"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := tools.GetWikiInfo(ctx, s.client, args.WikiURL)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handleWikiSearch(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL string `json:"wiki_url"`
		Query   string `json:"query"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Limit == 0 {
		args.Limit = 10
	}

	result, err := tools.SearchWiki(ctx, s.client, args.WikiURL, args.Query, args.Limit)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handlePageOutline(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL string `json:"wiki_url"`
		Title   string `json:"title"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := tools.GetPageOutline(ctx, s.client, args.WikiURL, args.Title)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handlePageSection(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL      string `json:"wiki_url"`
		Title        string `json:"title"`
		SectionIndex int    `json:"section_index"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := tools.GetPageSection(ctx, s.client, args.WikiURL, args.Title, args.SectionIndex)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handlePageFull(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL string `json:"wiki_url"`
		Title   string `json:"title"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}

	result, err := tools.GetPageFull(ctx, s.client, args.WikiURL, args.Title)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handleCategory(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL  string `json:"wiki_url"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Limit == 0 {
		args.Limit = 20
	}

	result, err := tools.GetCategory(ctx, s.client, args.WikiURL, args.Category, args.Limit)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handleBacklinks(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL string `json:"wiki_url"`
		Title   string `json:"title"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if args.Limit == 0 {
		args.Limit = 20
	}

	result, err := tools.GetBacklinks(ctx, s.client, args.WikiURL, args.Title, args.Limit)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

func (s *Server) handleCompare(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		WikiURL      string `json:"wiki_url"`
		Title        string `json:"title"`
		FromRevision string `json:"from_revision"`
		ToRevision   string `json:"to_revision"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if args.FromRevision == "" {
		args.FromRevision = "prev"
	}
	if args.ToRevision == "" {
		args.ToRevision = "current"
	}

	result, err := tools.CompareRevisions(ctx, s.client, args.WikiURL, args.Title, args.FromRevision, args.ToRevision)
	if err != nil {
		return s.errorResult(err), nil
	}

	return s.successResult(result)
}

// Helper methods

func (s *Server) successResult(data interface{}) (*mcp.CallToolResult, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil
}

func (s *Server) errorResult(err error) *mcp.CallToolResult {
	errResp := FormatError(err)
	errJSON, _ := json.Marshal(errResp)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(errJSON)},
		},
		IsError: true,
	}
}
