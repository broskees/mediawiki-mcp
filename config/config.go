package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all server configuration
type Config struct {
	Port           string
	RateLimit      float64 // requests per second per wiki
	CacheTTL       time.Duration
	CacheTTLInfo   time.Duration
	UserAgent      string
	RequestTimeout time.Duration
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port:           getEnv("MCP_PORT", "8080"),
		RateLimit:      getEnvFloat("MCP_RATE_LIMIT", 10.0),
		CacheTTL:       getEnvDuration("MCP_CACHE_TTL", 300),
		CacheTTLInfo:   getEnvDuration("MCP_CACHE_TTL_INFO", 3600),
		UserAgent:      getEnv("MCP_USER_AGENT", "MediaWikiMCP/1.0 (https://github.com/yourusername/mediawiki-mcp)"),
		RequestTimeout: getEnvDuration("MCP_REQUEST_TIMEOUT", 30),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultSeconds int) time.Duration {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return time.Duration(i) * time.Second
		}
	}
	return time.Duration(defaultSeconds) * time.Second
}
