package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken           string            `yaml:"bot_token"`
	BackendURL         string            `yaml:"backend_url"`
	ResellerAPIKey     string            `yaml:"reseller_api_key"`
	BotID              int64             `yaml:"bot_id"`
	AdminChatIDs       []int64           `yaml:"admin_chat_ids"`
	HostMappings       map[string]string `yaml:"host_mappings"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"`
}

func Load(path string) (*Config, error) {
	envBotToken := os.Getenv("BOT_TOKEN")
	envBackendURL := os.Getenv("BACKEND_URL")
	envResellerAPIKey := os.Getenv("RESELLER_API_KEY")
	envBotIDStr := os.Getenv("BOT_ID")

	allEnvPresent := envBotToken != "" && envBackendURL != "" && envResellerAPIKey != "" && envBotIDStr != ""

	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && allEnvPresent {
			// Config file missing, but all required environment variables are set
		} else {
			if errors.Is(err, os.ErrNotExist) {
				var missing []string
				if envBotToken == "" {
					missing = append(missing, "BOT_TOKEN")
				}
				if envBackendURL == "" {
					missing = append(missing, "BACKEND_URL")
				}
				if envResellerAPIKey == "" {
					missing = append(missing, "RESELLER_API_KEY")
				}
				if envBotIDStr == "" {
					missing = append(missing, "BOT_ID")
				}
				return nil, fmt.Errorf("config file %q not found and missing required environment variables: %s", path, strings.Join(missing, ", "))
			}
			return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
		}
	}

	// Environment variable overrides
	if envBotToken != "" {
		cfg.BotToken = envBotToken
	}
	if envBackendURL != "" {
		cfg.BackendURL = envBackendURL
	}
	if envResellerAPIKey != "" {
		cfg.ResellerAPIKey = envResellerAPIKey
	}
	if envBotIDStr != "" {
		botID, err := strconv.ParseInt(envBotIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid BOT_ID environment variable %q: %w", envBotIDStr, err)
		}
		cfg.BotID = botID
	}

	// Validation of required fields
	var missingFields []string
	if cfg.BotToken == "" {
		missingFields = append(missingFields, "bot_token")
	}
	if cfg.BackendURL == "" {
		missingFields = append(missingFields, "backend_url")
	}
	if cfg.ResellerAPIKey == "" {
		missingFields = append(missingFields, "reseller_api_key")
	}

	// If loaded purely via environment variables (no config file), BOT_ID is also required
	if data == nil && cfg.BotID == 0 {
		missingFields = append(missingFields, "bot_id")
	}

	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing required configuration: %s", strings.Join(missingFields, ", "))
	}

	return cfg, nil
}

func (c *Config) LogSummary() {
	botTokenState := "[NOT SET]"
	if c.BotToken != "" {
		botTokenState = "[SET]"
	}

	apiKeyState := "[NOT SET]"
	if c.ResellerAPIKey != "" {
		apiKeyState = "[SET]"
	}

	log.Printf("Configuration loaded: BackendURL=%s, BotID=%d, BotToken=%s, ResellerAPIKey=%s, InsecureSkipVerify=%v",
		c.BackendURL, c.BotID, botTokenState, apiKeyState, c.InsecureSkipVerify)
}
