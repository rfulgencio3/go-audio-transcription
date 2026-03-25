// Package config loads all application configuration from environment variables.
// No configuration is hardcoded. Use a .env file (via godotenv) in development.
package config

import (
	"fmt"
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
	OpenAI  OpenAIConfig
	Gemini  GeminiConfig
	RavenDB RavenDBConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Addr           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxUploadBytes int64
}

// OpenAIConfig holds OpenAI API credentials.
type OpenAIConfig struct {
	APIKey string
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
// Returns a descriptive error listing any missing required variables.
func LoadFromEnv() (Config, error) {
	var missing []string

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		missing = append(missing, "OPENAI_API_KEY")
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		missing = append(missing, "GEMINI_API_KEY")
	}

	ravenURLsRaw := getEnvOrDefault("RAVENDB_URLS", defaultRavenDBURLs)
	ravenURLs := strings.Split(ravenURLsRaw, ",")
	for i := range ravenURLs {
		ravenURLs[i] = strings.TrimSpace(ravenURLs[i])
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required environment variables: %s", strings.Join(missing, ", "))
	}

	maxUploadBytes, err := parseInt64Env("MAX_UPLOAD_BYTES", defaultMaxUploadBytes)
	if err != nil {
		return Config{}, fmt.Errorf("config: invalid MAX_UPLOAD_BYTES: %w", err)
	}

	return Config{
		Server: ServerConfig{
			Addr:           getEnvOrDefault("ADDR", defaultAddr),
			ReadTimeout:    defaultReadTimeout,
			WriteTimeout:   defaultWriteTimeout,
			MaxUploadBytes: maxUploadBytes,
		},
		OpenAI: OpenAIConfig{
			APIKey: openaiKey,
		},
		Gemini: GeminiConfig{
			APIKey:    geminiKey,
			ModelName: getEnvOrDefault("GEMINI_MODEL", defaultGeminiModel),
		},
		RavenDB: RavenDBConfig{
			URLs:         ravenURLs,
			DatabaseName: getEnvOrDefault("RAVENDB_DATABASE", defaultRavenDBName),
		},
	}, nil
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
