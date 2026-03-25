package config

import "testing"

func TestLoadFromEnv_AllowsMissingGeminiKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "")

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
}

func TestLoadFromEnv_UsesPlatformPortWhenAddrUnset(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "9000")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Server.Addr != ":9000" {
		t.Fatalf("server addr = %q, want %q", cfg.Server.Addr, ":9000")
	}
}
