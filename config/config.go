// Package config loads all application configuration from environment variables.
package config

import (
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultAddr           = ":8080"
	defaultMaxUploadBytes = 25 * 1024 * 1024 // 25 MB
	defaultGeminiModel    = "gemini-2.5-flash"
	defaultReadTimeout    = 30 * time.Second
	defaultWriteTimeout   = 120 * time.Second
)

// Config holds all runtime configuration for the application.
type Config struct {
	Server ServerConfig
	Gemini GeminiConfig
	Public PublicConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxUploadBytes int64
}

// PublicConfig holds externally visible app URLs.
type PublicConfig struct {
	BaseURL string
	Host    string
	Scheme  string
}

// GeminiConfig holds Google Gemini API credentials.
type GeminiConfig struct {
	APIKey    string
	ModelName string
}

// LoadFromEnv reads all configuration from environment variables.
func LoadFromEnv() (Config, error) {
	geminiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))

	return Config{
		Server: ServerConfig{
			Addr:           getServerAddr(),
			ReadTimeout:    defaultReadTimeout,
			WriteTimeout:   defaultWriteTimeout,
			MaxUploadBytes: defaultMaxUploadBytes,
		},
		Gemini: GeminiConfig{
			APIKey:    geminiKey,
			ModelName: getEnvOrDefault("GEMINI_MODEL", defaultGeminiModel),
		},
		Public: getPublicConfig(),
	}, nil
}

func getServerAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return defaultAddr
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getPublicConfig() PublicConfig {
	publicDomain := strings.TrimSpace(os.Getenv("RAILWAY_PUBLIC_DOMAIN"))
	if publicDomain == "" {
		return PublicConfig{}
	}

	baseURL := "https://" + publicDomain

	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return PublicConfig{BaseURL: baseURL}
	}

	return PublicConfig{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Host:    u.Host,
		Scheme:  u.Scheme,
	}
}
