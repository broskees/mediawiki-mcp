# MediaWiki MCP Server

A lightweight HTTP MCP (Model Context Protocol) server that provides agent-friendly access to any MediaWiki-powered wiki (Wikipedia, Fandom, internal wikis, etc.).

## Features

- **8 agent-optimized tools** for exploring wikis
- **Smart hierarchical page access** - Outline → Section → Full content
- **Automatic HTML→Markdown conversion** for clean, token-efficient responses
- **Infobox extraction** - Structured data from wiki templates
- **Built-in rate limiting and caching** to be a good citizen
- **Works with any MediaWiki site** - Pass the wiki URL per request
- **Lightweight Go implementation** - Single binary, minimal dependencies

## Tools

| Tool | Purpose |
|------|---------|
| `wiki_info` | Get wiki metadata (name, language, article count, namespaces) |
| `wiki_search` | Search for pages by keyword |
| `wiki_page_outline` | Get page structure with sections, summary, infobox, links |
| `wiki_page_section` | Retrieve full content of a specific section |
| `wiki_page_full` | Get entire page content (with size warning) |
| `wiki_category` | Browse pages in a category |
| `wiki_backlinks` | Find pages linking to a given page |
| `wiki_compare` | Compare two revisions to see changes |

## Quick Start

### Build

```bash
go build -o mediawiki-mcp
```

### Run

```bash
./mediawiki-mcp
```

The server will start on port 8080 by default with the following endpoints:

- `http://localhost:8080/mcp` - MCP endpoint (Streamable HTTP transport)
- `http://localhost:8080/health` - Health check

### Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_PORT` | `8080` | HTTP server port |
| `MCP_RATE_LIMIT` | `10` | Requests per second per wiki |
| `MCP_CACHE_TTL` | `300` | Default cache TTL in seconds |
| `MCP_CACHE_TTL_INFO` | `3600` | Cache TTL for wiki_info |
| `MCP_USER_AGENT` | `MediaWikiMCP/1.0` | User-Agent for API requests |
| `MCP_REQUEST_TIMEOUT` | `30` | HTTP request timeout in seconds |

Example:

```bash
export MCP_PORT=9000
export MCP_RATE_LIMIT=5
./mediawiki-mcp
```

## Usage Examples

### Search Wikipedia

```json
{
  "tool": "wiki_search",
  "arguments": {
    "wiki_url": "https://en.wikipedia.org",
    "query": "quantum mechanics",
    "limit": 5
  }
}
```

### Get Page Outline

```json
{
  "tool": "wiki_page_outline",
  "arguments": {
    "wiki_url": "https://en.wikipedia.org",
    "title": "Albert Einstein"
  }
}
```

Response includes:
- Summary (first paragraph)
- Structured section tree with previews
- Infobox data (birth date, field, etc.)
- Categories and "See also" links
- Word count per section

### Get Specific Section

```json
{
  "tool": "wiki_page_section",
  "arguments": {
    "wiki_url": "https://en.wikipedia.org",
    "title": "Albert Einstein",
    "section_index": 3
  }
}
```

### Browse a Category

```json
{
  "tool": "wiki_category",
  "arguments": {
    "wiki_url": "https://en.wikipedia.org",
    "category": "Nobel laureates in Physics",
    "limit": 20
  }
}
```

## Workflow for Agents

The recommended workflow for efficient page exploration:

1. **Search** - `wiki_search` to find pages
2. **Outline** - `wiki_page_outline` to understand structure (~500 tokens)
3. **Sections** - `wiki_page_section` for specific content (targeted)
4. **Navigate** - Use links/categories to explore related pages

This approach minimizes context usage while giving agents full visibility into page structure.

## Design Philosophy

### Agent-Friendly

- **Progressive disclosure** - Outline → Section → Full
- **Structured output** - Markdown + metadata, not raw HTML
- **Clear error messages** - Hints for recovery (e.g., "call wiki_page_outline to refresh indices")
- **Self-describing** - Tools have detailed descriptions and schemas

### Respectful to Wikis

- **Rate limiting** per wiki domain (default 10 req/s)
- **Caching** to reduce duplicate requests
- **Proper User-Agent** header
- **maxlag parameter** for non-interactive tasks
- **Serialized requests** per domain

### Simple & Maintainable

- **Minimal dependencies** - Go stdlib + MCP SDK + HTML parser
- **Single binary** - Easy deployment
- **Clear separation** - wiki client, tools, MCP server
- **No premature abstraction** - Straightforward code

## Architecture

```
mediawiki-mcp/
├── main.go                   # HTTP server + MCP handler
├── config/                   # Environment configuration
├── internal/
│   ├── wiki/                # MediaWiki API client
│   │   ├── client.go        # HTTP, rate limiting, caching
│   │   ├── parser.go        # HTML→Markdown conversion
│   │   ├── infobox.go       # Template extraction
│   │   └── types.go         # Data structures
│   ├── tools/               # Tool implementations
│   │   ├── info.go
│   │   ├── search.go
│   │   ├── outline.go
│   │   ├── section.go
│   │   ├── full.go
│   │   ├── category.go
│   │   ├── backlinks.go
│   │   └── compare.go
│   └── mcp/                 # MCP server
│       ├── server.go        # Tool registration + handlers
│       └── errors.go        # Structured error responses
```

## Dependencies

- [go-sdk](https://github.com/modelcontextprotocol/go-sdk) - Official MCP Go SDK
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) - HTML parsing
- [rate](https://golang.org/x/time/rate) - Rate limiting

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o mediawiki-mcp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/mediawiki-mcp /usr/local/bin/
EXPOSE 8080
CMD ["mediawiki-mcp"]
```

Build and run:

```bash
docker build -t mediawiki-mcp .
docker run -p 8080:8080 mediawiki-mcp
```

### Systemd Service

```ini
[Unit]
Description=MediaWiki MCP Server
After=network.target

[Service]
Type=simple
User=mcp
ExecStart=/usr/local/bin/mediawiki-mcp
Restart=on-failure
Environment="MCP_PORT=8080"
Environment="MCP_RATE_LIMIT=10"

[Install]
WantedBy=multi-user.target
```

## Error Handling

The server returns structured errors with helpful hints:

```json
{
  "error": "section_not_found",
  "message": "Section index 5 does not exist. The page may have been edited.",
  "hint": "Call wiki_page_outline to get fresh section indices.",
  "details": {
    "section_index": 5,
    "available_sections": 12
  }
}
```

Common error codes:
- `missingtitle` - Page doesn't exist (hint: use wiki_search)
- `nosuchsection` - Section index invalid (hint: refresh outline)
- `maxlag` - Wiki server busy (hint: retry after delay)
- `section_not_found` - Section not found (hint: call outline)

## Testing

Test against Wikipedia:

```bash
# Search
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"wiki_search","arguments":{"wiki_url":"https://en.wikipedia.org","query":"test"}}}'
```

## Contributing

This is a reference implementation. Feel free to:

- Add more tools (edit pages, upload files, etc.)
- Improve wikitext parsing
- Add unit/integration tests
- Optimize caching strategies
- Support other MediaWiki extensions

## License

MIT

## Author

Built following principles of simplicity, clarity, and respect for both agents and wiki servers.
