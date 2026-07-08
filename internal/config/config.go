// Package config holds runtime configuration shared across the app.
package config

import (
	"fmt"
	"log"
	"os"
)

// Config is passed into telegram.New and internal/schedule.
type Config struct {
	Debug bool

	BotToken           string
	CredentialsPath    string
	FirestoreProjectID string

	// Webhook mode (optional); if Port/Domain are empty, long polling is used.
	Port   string
	Domain string
}

func New(debug bool) *Config {
	cfg := &Config{
		Debug: debug,

		BotToken:           os.Getenv("BOT_TOKEN"),
		CredentialsPath:    getenvDefault("CREDENTIALS_PATH", "credentials.json"),
		FirestoreProjectID: os.Getenv("FIRESTORE_PROJECT_ID"),

		Port:   os.Getenv("PORT"),
		Domain: os.Getenv("DOMAIN"),
	}

	if err := cfg.validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	return cfg
}

func (c *Config) validate() error {
	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}
	if c.CredentialsPath == "" {
		return fmt.Errorf("CREDENTIALS_PATH is required")
	}
	return nil
}

// UseWebhook reports whether the bot should run in webhook mode.
func (c *Config) UseWebhook() bool {
	return c.Port != "" && c.Domain != ""
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
