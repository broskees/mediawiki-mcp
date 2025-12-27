package wiki

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client handles MediaWiki API requests
type Client struct {
	httpClient   *http.Client
	userAgent    string
	cache        *Cache
	cacheTTL     time.Duration
	cacheTTLInfo time.Duration

	// Rate limiters per wiki domain
	limiters  map[string]*rate.Limiter
	limiterMu sync.RWMutex
	rateLimit rate.Limit

	// API path cache per wiki domain
	apiPaths   map[string]string
	apiPathsMu sync.RWMutex
}

// NewClient creates a new MediaWiki API client
func NewClient(userAgent string, timeout time.Duration, rateLimit float64, cacheTTL, cacheTTLInfo time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent:    userAgent,
		cache:        NewCache(),
		cacheTTL:     cacheTTL,
		cacheTTLInfo: cacheTTLInfo,
		limiters:     make(map[string]*rate.Limiter),
		rateLimit:    rate.Limit(rateLimit),
		apiPaths:     make(map[string]string),
	}
}

// getLimiter returns a rate limiter for a wiki domain
func (c *Client) getLimiter(wikiURL string) *rate.Limiter {
	c.limiterMu.RLock()
	limiter, exists := c.limiters[wikiURL]
	c.limiterMu.RUnlock()

	if exists {
		return limiter
	}

	c.limiterMu.Lock()
	defer c.limiterMu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := c.limiters[wikiURL]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(c.rateLimit, 1)
	c.limiters[wikiURL] = limiter
	return limiter
}

// getAPIPath discovers and caches the API path for a wiki
func (c *Client) getAPIPath(ctx context.Context, wikiURL string) (string, error) {
	// Check cache first
	c.apiPathsMu.RLock()
	if path, exists := c.apiPaths[wikiURL]; exists {
		c.apiPathsMu.RUnlock()
		return path, nil
	}
	c.apiPathsMu.RUnlock()

	// Try common API paths in order of prevalence
	// /api.php is the default MediaWiki path
	paths := []string{"/api.php", "/w/api.php"}

	for _, path := range paths {
		apiURL := wikiURL + path
		testURL := apiURL + "?action=query&meta=siteinfo&format=json"

		req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Cache the working path
			c.apiPathsMu.Lock()
			c.apiPaths[wikiURL] = path
			c.apiPathsMu.Unlock()
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find valid API endpoint for %s (tried %v)", wikiURL, paths)
}

// MakeRequest makes an HTTP GET request to the MediaWiki API
func (c *Client) MakeRequest(ctx context.Context, wikiURL string, params url.Values) (*mwResponse, error) {
	// Apply rate limiting
	limiter := c.getLimiter(wikiURL)
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	// Discover API path
	apiPath, err := c.getAPIPath(ctx, wikiURL)
	if err != nil {
		return nil, err
	}

	// Build API URL
	apiURL := wikiURL + apiPath

	// Add common parameters
	params.Set("format", "json")
	params.Set("formatversion", "2")
	params.Set("utf8", "1")
	params.Set("maxlag", "5")

	fullURL := apiURL + "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept-Encoding", "gzip")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var bodyStr string
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			bodyStr = "(compressed error response)"
		} else {
			body, _ := io.ReadAll(resp.Body)
			bodyStr = string(body)
		}
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, bodyStr)
	}

	// Handle gzip encoding
	reader := resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Parse response
	var mwResp mwResponse
	if err := json.NewDecoder(reader).Decode(&mwResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Check for API errors
	if mwResp.Error != nil {
		return nil, &APIError{
			Code:    mwResp.Error.Code,
			Message: mwResp.Error.Info,
		}
	}

	return &mwResp, nil
}

// APIError represents a MediaWiki API error
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("mediawiki api error: %s: %s", e.Code, e.Message)
}

// GetCache returns the cache instance
func (c *Client) GetCache() *Cache {
	return c.cache
}

// GetCacheTTL returns the default cache TTL
func (c *Client) GetCacheTTL() time.Duration {
	return c.cacheTTL
}

// GetCacheTTLInfo returns the cache TTL for wiki info
func (c *Client) GetCacheTTLInfo() time.Duration {
	return c.cacheTTLInfo
}
