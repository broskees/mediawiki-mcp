package mcp

import (
	"github.com/yourusername/mediawiki-mcp/internal/tools"
	"github.com/yourusername/mediawiki-mcp/internal/wiki"
)

// ErrorResponse represents a structured error response for MCP
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Hint    string                 `json:"hint,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// FormatError converts various error types to structured ErrorResponse
func FormatError(err error) *ErrorResponse {
	if err == nil {
		return nil
	}

	// Handle specific error types
	switch e := err.(type) {
	case *wiki.APIError:
		return formatAPIError(e)
	case *tools.SectionNotFoundError:
		return formatSectionNotFoundError(e)
	default:
		return &ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		}
	}
}

func formatAPIError(err *wiki.APIError) *ErrorResponse {
	resp := &ErrorResponse{
		Error:   err.Code,
		Message: err.Message,
	}

	// Add helpful hints based on error code
	switch err.Code {
	case "missingtitle":
		resp.Hint = "The page doesn't exist. Try using wiki_search to find the correct title."
	case "nosuchsection":
		resp.Hint = "The section doesn't exist. Call wiki_page_outline to get fresh section indices."
	case "maxlag":
		resp.Hint = "The wiki server is experiencing high load. Wait a moment and try again."
	}

	return resp
}

func formatSectionNotFoundError(err *tools.SectionNotFoundError) *ErrorResponse {
	return &ErrorResponse{
		Error:   "section_not_found",
		Message: err.Error(),
		Hint:    "Call wiki_page_outline to get fresh section indices.",
		Details: map[string]interface{}{
			"section_index":      err.SectionIndex,
			"available_sections": err.AvailableSections,
		},
	}
}

// FormatErrorString creates an error response from a simple string
func FormatErrorString(code, message string) *ErrorResponse {
	return &ErrorResponse{
		Error:   code,
		Message: message,
	}
}
