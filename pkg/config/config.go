package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken       string  `yaml:"bot_token"`
	BackendURL     string  `yaml:"backend_url"`
	ResellerAPIKey string  `yaml:"reseller_api_key"`
	AdminChatIDs   []int64 `yaml:"admin_chat_ids"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
