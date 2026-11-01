package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env, Addr, BaseURL, DatabaseURL, CookieName, StorageRoot, DefaultOrgSlug string
	SessionTTL                                                               time.Duration
	CookieSecure                                                             bool
	MaxUploadBytes                                                           int64
}

func Load() (Config, error) {
	_ = loadDotEnv(".env")
	ttl, err := time.ParseDuration(env("SESSION_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_TTL: %w", err)
	}
	max, err := strconv.ParseInt(env("MAX_UPLOAD_BYTES", "104857600"), 10, 64)
	if err != nil || max <= 0 {
		return Config{}, fmt.Errorf("MAX_UPLOAD_BYTES must be positive")
	}
	secure, err := strconv.ParseBool(env("SESSION_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_SECURE: %w", err)
	}
	c := Config{Env: env("APP_ENV", "development"), Addr: env("APP_ADDR", ":8080"), BaseURL: env("APP_BASE_URL", "http://localhost:8080"), DatabaseURL: os.Getenv("DATABASE_URL"), CookieName: env("SESSION_COOKIE_NAME", "cloud_drive_session"), StorageRoot: env("STORAGE_ROOT", "./storage"), DefaultOrgSlug: env("DEFAULT_ORGANIZATION_SLUG", "default"), SessionTTL: ttl, CookieSecure: secure, MaxUploadBytes: max}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			k = strings.TrimSpace(k)
			v = strings.Trim(strings.TrimSpace(v), "\"'")
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
	return s.Err()
}
