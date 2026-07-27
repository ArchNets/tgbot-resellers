package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_YAML(t *testing.T) {
	// Clear env vars
	os.Unsetenv("BOT_TOKEN")
	os.Unsetenv("BACKEND_URL")
	os.Unsetenv("RESELLER_API_KEY")
	os.Unsetenv("BOT_ID")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := []byte(`
bot_token: "test-token-yaml"
backend_url: "https://yaml.example.com"
reseller_api_key: "rn_yaml_key"
bot_id: 100
`)
	if err := os.WriteFile(configPath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BotToken != "test-token-yaml" {
		t.Errorf("expected BotToken test-token-yaml, got %s", cfg.BotToken)
	}
	if cfg.BackendURL != "https://yaml.example.com" {
		t.Errorf("expected BackendURL https://yaml.example.com, got %s", cfg.BackendURL)
	}
	if cfg.ResellerAPIKey != "rn_yaml_key" {
		t.Errorf("expected ResellerAPIKey rn_yaml_key, got %s", cfg.ResellerAPIKey)
	}
	if cfg.BotID != 100 {
		t.Errorf("expected BotID 100, got %d", cfg.BotID)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := []byte(`
bot_token: "test-token-yaml"
backend_url: "https://yaml.example.com"
reseller_api_key: "rn_yaml_key"
bot_id: 100
`)
	if err := os.WriteFile(configPath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	t.Setenv("BOT_TOKEN", "env-token-override")
	t.Setenv("BOT_ID", "999")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BotToken != "env-token-override" {
		t.Errorf("expected BotToken env-token-override, got %s", cfg.BotToken)
	}
	if cfg.BackendURL != "https://yaml.example.com" {
		t.Errorf("expected BackendURL https://yaml.example.com, got %s", cfg.BackendURL)
	}
	if cfg.BotID != 999 {
		t.Errorf("expected BotID 999, got %d", cfg.BotID)
	}
}

func TestLoad_EnvOnly(t *testing.T) {
	t.Setenv("BOT_TOKEN", "env-token")
	t.Setenv("BACKEND_URL", "https://env.example.com")
	t.Setenv("RESELLER_API_KEY", "rn_env_key")
	t.Setenv("BOT_ID", "555")

	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent.yaml")

	cfg, err := Load(nonExistentPath)
	if err != nil {
		t.Fatalf("expected pure env config to succeed without config.yaml, got err: %v", err)
	}

	if cfg.BotToken != "env-token" || cfg.BackendURL != "https://env.example.com" || cfg.ResellerAPIKey != "rn_env_key" || cfg.BotID != 555 {
		t.Errorf("loaded config does not match env vars: %+v", cfg)
	}
}

func TestLoad_InvalidBotID(t *testing.T) {
	t.Setenv("BOT_TOKEN", "env-token")
	t.Setenv("BACKEND_URL", "https://env.example.com")
	t.Setenv("RESELLER_API_KEY", "rn_env_key")
	t.Setenv("BOT_ID", "not-a-number")

	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent.yaml")

	_, err := Load(nonExistentPath)
	if err == nil {
		t.Fatal("expected error for invalid BOT_ID, got nil")
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	os.Unsetenv("BOT_TOKEN")
	os.Unsetenv("BACKEND_URL")
	os.Unsetenv("RESELLER_API_KEY")
	os.Unsetenv("BOT_ID")

	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent.yaml")

	_, err := Load(nonExistentPath)
	if err == nil {
		t.Fatal("expected error when config file and env vars are missing, got nil")
	}
}
