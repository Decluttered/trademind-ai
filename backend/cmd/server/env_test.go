package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testEnvKey = "TRADEMIND_DOTENV_TEST_VALUE"

func TestLoadDotEnvSupportsUTF8BOM(t *testing.T) {
	repoRoot := writeTestRepo(t, append(append([]byte{}, utf8BOM...), []byte(testEnvKey+"=loaded\n")...))
	prepareDotEnvTest(t, repoRoot)

	if err := loadDotEnv(); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv(testEnvKey); got != "loaded" {
		t.Fatalf("%s = %q, want loaded", testEnvKey, got)
	}
}

func TestLoadDotEnvPreservesProcessEnvironment(t *testing.T) {
	repoRoot := writeTestRepo(t, []byte(testEnvKey+"=from-file\n"))
	prepareDotEnvTest(t, repoRoot)
	t.Setenv(testEnvKey, "from-process")

	if err := loadDotEnv(); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv(testEnvKey); got != "from-process" {
		t.Fatalf("%s = %q, want process environment to win", testEnvKey, got)
	}
}

func TestLoadDotEnvReturnsSanitizedParseError(t *testing.T) {
	repoRoot := writeTestRepo(t, []byte("BROKEN ENV LINE\nSECRET_VALUE=must-not-appear\n"))
	prepareDotEnvTest(t, repoRoot)

	err := loadDotEnv()
	if err == nil {
		t.Fatal("loadDotEnv() error = nil, want invalid dotenv syntax")
	}
	if !strings.Contains(err.Error(), "invalid dotenv syntax") {
		t.Fatalf("loadDotEnv() error = %q, want sanitized syntax error", err)
	}
	if strings.Contains(err.Error(), "must-not-appear") {
		t.Fatalf("loadDotEnv() error exposed an environment value: %q", err)
	}
}

func TestLoadDotEnvSkipsPerformanceHarness(t *testing.T) {
	repoRoot := writeTestRepo(t, []byte(testEnvKey+"=from-file\n"))
	prepareDotEnvTest(t, repoRoot)
	t.Setenv("APP_ENV", "performance")
	t.Setenv("PERFORMANCE_TEST_MODE", "true")

	if err := loadDotEnv(); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if _, exists := os.LookupEnv(testEnvKey); exists {
		t.Fatalf("%s was loaded for the isolated performance harness", testEnvKey)
	}
}

func writeTestRepo(t *testing.T, envContents []byte) string {
	t.Helper()
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "pnpm-workspace.yaml"), []byte("packages: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repoRoot, "backend"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "backend", "go.mod"), []byte("module example.invalid/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".env"), envContents, 0o600); err != nil {
		t.Fatal(err)
	}
	return repoRoot
}

func prepareDotEnvTest(t *testing.T, repoRoot string) {
	t.Helper()
	t.Setenv("TRADEMIND_REPO_ROOT", repoRoot)
	t.Setenv("APP_ENV", "development")
	t.Setenv("PERFORMANCE_TEST_MODE", "false")
	unsetEnv(t, testEnvKey)
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
			return
		}
		_ = os.Unsetenv(key)
	})
}
