package config

import "testing"

func TestLoadFromEnv_AllowsMissingGeminiKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("PORT", "")
	t.Setenv("MONGODB_URI", "mongodb://example")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Gemini.APIKey != "" {
		t.Fatalf("Gemini API key = %q, want empty", cfg.Gemini.APIKey)
	}
	if cfg.Server.Addr != defaultAddr {
		t.Fatalf("server addr = %q, want %q", cfg.Server.Addr, defaultAddr)
	}
	if cfg.Gemini.ModelName != defaultGeminiModel {
		t.Fatalf("gemini model = %q, want %q", cfg.Gemini.ModelName, defaultGeminiModel)
	}
	if cfg.Mongo.DatabaseName != defaultMongoDatabase {
		t.Fatalf("mongo database = %q, want %q", cfg.Mongo.DatabaseName, defaultMongoDatabase)
	}
	if cfg.Mongo.CollectionName != defaultMongoCollection {
		t.Fatalf("mongo collection = %q, want %q", cfg.Mongo.CollectionName, defaultMongoCollection)
	}
	if cfg.Server.MaxUploadBytes != defaultMaxUploadBytes {
		t.Fatalf("max upload bytes = %d, want %d", cfg.Server.MaxUploadBytes, defaultMaxUploadBytes)
	}
}

func TestLoadFromEnv_UsesGeminiModelOverride(t *testing.T) {
	t.Setenv("GEMINI_MODEL", "gemini-2.5-flash-lite")
	t.Setenv("MONGODB_URI", "mongodb://example")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Gemini.ModelName != "gemini-2.5-flash-lite" {
		t.Fatalf("gemini model = %q", cfg.Gemini.ModelName)
	}
}

func TestLoadFromEnv_UsesPlatformPort(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("MONGODB_URI", "mongodb://example")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Server.Addr != ":9000" {
		t.Fatalf("server addr = %q, want %q", cfg.Server.Addr, ":9000")
	}
}

func TestLoadFromEnv_UsesRailwayPublicDomain(t *testing.T) {
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "go-audio-transcription.up.railway.app")
	t.Setenv("MONGODB_URI", "mongodb://example")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Public.BaseURL != "https://go-audio-transcription.up.railway.app" {
		t.Fatalf("public base URL = %q", cfg.Public.BaseURL)
	}
	if cfg.Public.Host != "go-audio-transcription.up.railway.app" {
		t.Fatalf("public host = %q", cfg.Public.Host)
	}
	if cfg.Public.Scheme != "https" {
		t.Fatalf("public scheme = %q", cfg.Public.Scheme)
	}
}

func TestLoadFromEnv_WithoutRailwayDomainLeavesPublicConfigEmpty(t *testing.T) {
	t.Setenv("RAILWAY_PUBLIC_DOMAIN", "")
	t.Setenv("MONGODB_URI", "mongodb://example")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Public.BaseURL != "" {
		t.Fatalf("public base URL = %q", cfg.Public.BaseURL)
	}
	if cfg.Public.Host != "" {
		t.Fatalf("public host = %q", cfg.Public.Host)
	}
}

func TestLoadFromEnv_UsesMongoURLFallback(t *testing.T) {
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGO_URL", "mongodb://mongo-service:27017")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Mongo.URI != "mongodb://mongo-service:27017" {
		t.Fatalf("mongo uri = %q", cfg.Mongo.URI)
	}
}

func TestLoadFromEnv_RequiresMongoURI(t *testing.T) {
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGO_URL", "")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
}
