package main

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all server settings, loaded from the environment.
type Config struct {
	Port          string
	AuthEnabled   bool
	KeyPrefix     string
	ValidKeys     map[string]bool
	MeowMin       int
	MeowMax       int
	StreamDelayMS int
}

func loadConfig() Config {
	c := Config{
		Port:          envOr("PORT", "8080"),
		AuthEnabled:   envBoolOr("AUTH_ENABLED", true),
		KeyPrefix:     envOr("KEY_PREFIX", "sk-"),
		MeowMin:       envIntOr("MEOW_MIN", 1),
		MeowMax:       envIntOr("MEOW_MAX", 20),
		StreamDelayMS: envIntOr("STREAM_DELAY_MS", 0),
	}
	c.ValidKeys = map[string]bool{}
	for _, k := range strings.Split(os.Getenv("VALID_KEYS"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			c.ValidKeys[k] = true
		}
	}
	if c.MeowMin < 1 {
		c.MeowMin = 1
	}
	if c.MeowMax < c.MeowMin {
		c.MeowMax = c.MeowMin
	}
	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBoolOr(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
