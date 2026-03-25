// Package config loads all application configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr           = ":8080"
	defaultMaxUploadBytes = 25 * 1024 * 1024 // 25 MB
	defaultGeminiModel    = "gemini-1.5-flash"
	defaultRavenDBURLs    = "http://localhost:8080"
	defaultRavenDBName    = "AudioTranscriptions"
	defaultReadTimeout    = 30 * time.Second
	defaultWriteTimeout   = 120 * time.Second
)

// Config holds all runtime configuration for the application.
type Config struct {
	Server  ServerConfig
	Gemini  GeminiConfig
	RavenDB RavenDBConfig
	Public  PublicConfig
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

// RavenDBConfig holds connection settings for RavenDB.
type RavenDBConfig struct {
	URLs         []string
	DatabaseName string
}

// LoadFromEnv reads all configuration from environment variables.
func LoadFromEnv() (Config, error) {
	geminiKey := os.Getenv("GEMINI_API_KEY")

	ravenURLsRaw := getEnvOrDefault("RAVENDB_URLS", defaultRavenDBURLs)
	ravenURLs := strings.Split(ravenURLsRaw, ",")
	for i := range ravenURLs {
		ravenURLs[i] = strings.TrimSpace(ravenURLs[i])
	}

	maxUploadBytes, err := parseInt64Env("MAX_UPLOAD_BYTES", defaultMaxUploadBytes)
	if err != nil {
		return Config{}, fmt.Errorf("config: invalid MAX_UPLOAD_BYTES: %w", err)
	}

	return Config{
		Server: ServerConfig{
			Addr:           getServerAddr(),
			ReadTimeout:    defaultReadTimeout,
			WriteTimeout:   defaultWriteTimeout,
			MaxUploadBytes: maxUploadBytes,
		},
		Gemini: GeminiConfig{
			APIKey:    geminiKey,
			ModelName: getEnvOrDefault("GEMINI_MODEL", defaultGeminiModel),
		},
		RavenDB: RavenDBConfig{
			URLs:         ravenURLs,
			DatabaseName: getEnvOrDefault("RAVENDB_DATABASE", defaultRavenDBName),
		},
		Public: getPublicConfig(),
	}, nil
}

func getServerAddr() string {
	if addr := os.Getenv("ADDR"); addr != "" {
		return addr
	}
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

func parseInt64Env(key string, defaultVal int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func getPublicConfig() PublicConfig {
	baseURL := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	if baseURL == "" {
		return PublicConfig{}
	}

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
