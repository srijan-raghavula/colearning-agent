package config

import "testing"

func TestLoadAIDefaultsToGroqFromAPIKey(t *testing.T) {
	t.Setenv("API_KEY", "gsk_test")
	t.Setenv("COLEARN_AI_API_KEY", "")
	t.Setenv("COLEARN_AI_PROVIDER", "")
	t.Setenv("COLEARN_AI_BASE_URL", "")
	t.Setenv("COLEARN_AI_MODEL", "")

	got := LoadAI()
	if got.Provider != "groq" {
		t.Fatalf("provider = %q, want groq", got.Provider)
	}
	if got.BaseURL != "https://api.groq.com/openai/v1" {
		t.Fatalf("base URL = %q", got.BaseURL)
	}
	if got.Model != "openai/gpt-oss-120b" {
		t.Fatalf("model = %q", got.Model)
	}
}

func TestLoadAIPrefersCanonicalAPIKey(t *testing.T) {
	t.Setenv("API_KEY", "gsk_canonical")
	t.Setenv("COLEARN_AI_API_KEY", "legacy")
	t.Setenv("COLEARN_AI_PROVIDER", "")
	t.Setenv("COLEARN_AI_BASE_URL", "")
	t.Setenv("COLEARN_AI_MODEL", "")

	if got := LoadAI(); got.APIKey != "gsk_canonical" {
		t.Fatalf("API key = %q, want canonical API_KEY", got.APIKey)
	}
}
