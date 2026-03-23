package app

import (
	"fmt"
	"os"
)

// Config holds runtime configuration for the plugin.
type Config struct {
	Token     string // HA_TOKEN — if non-empty, validated against hello auth field
	Port      string // HA_PORT — WebSocket server port; "0" for random
	SystemUUID string // SYSTEM_UUID — stable identity for discovery dedup
	SystemMAC  string // SYSTEM_MAC — stable fake MAC for mDNS / network identity
}

func loadConfig() (Config, error) {
	uuid := os.Getenv("SYSTEM_UUID")
	if uuid == "" {
		return Config{}, fmt.Errorf("SYSTEM_UUID environment variable is required")
	}
	mac := os.Getenv("SYSTEM_MAC")
	if mac == "" {
		return Config{}, fmt.Errorf("SYSTEM_MAC environment variable is required")
	}
	return Config{
		Token:      getEnv("HA_TOKEN", ""),
		Port:       getEnv("HA_PORT", "0"),
		SystemUUID: uuid,
		SystemMAC:  mac,
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
