package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/pkg/reporoot"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func loadDotEnv() error {
	if config.NormalizeEnv(os.Getenv("APP_ENV")) == config.EnvPerformance && strings.EqualFold(strings.TrimSpace(os.Getenv("PERFORMANCE_TEST_MODE")), "true") {
		return nil
	}

	for _, envPath := range dotEnvCandidates() {
		err := loadDotEnvFile(envPath)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func dotEnvCandidates() []string {
	candidates := make([]string, 0, 4)
	if root, ok := reporoot.Find(); ok {
		candidates = append(candidates, filepath.Join(root, ".env"))
	}
	candidates = append(candidates, ".env", filepath.Join("..", ".env"), filepath.Join("..", "..", ".env"))

	unique := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := filepath.Clean(candidate)
		if absolute, err := filepath.Abs(key); err == nil {
			key = absolute
		}
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func loadDotEnvFile(envPath string) error {
	contents, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}

	values, err := godotenv.UnmarshalBytes(bytes.TrimPrefix(contents, utf8BOM))
	if err != nil {
		// godotenv parse errors can contain the remaining file contents, including secrets.
		return fmt.Errorf("parse env file %q: invalid dotenv syntax", envPath)
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set environment variable %q from %q: %w", key, envPath, err)
		}
	}
	return nil
}
