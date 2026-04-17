package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionString(t *testing.T) {
	if got, want := (CLIArgs{}).Version(), "helmizer "+version; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
}

func TestResolveConfigPathsAllowsMissingDefaultWhenGlobMatches(t *testing.T) {
	tmpDir := t.TempDir()
	matchedConfig := filepath.Join(tmpDir, "apps", "demo", "helmizer.yaml")
	if err := os.MkdirAll(filepath.Dir(matchedConfig), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(matchedConfig, []byte("helmizer: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	got, err := ResolveConfigPaths(CLIArgs{
		ConfigFilePath: "helmizer.yaml",
		ConfigGlob:     "**/helmizer.yaml",
	})
	if err != nil {
		t.Fatalf("ResolveConfigPaths() error = %v", err)
	}

	want := []string{matchedConfig}
	if len(got) != len(want) {
		t.Fatalf("ResolveConfigPaths() len = %d, want %d (%v)", len(got), len(want), got)
	}
	if got[0] != want[0] {
		t.Fatalf("ResolveConfigPaths()[0] = %q, want %q", got[0], want[0])
	}
}

func TestResolveConfigPathsDeduplicatesAndSorts(t *testing.T) {
	tmpDir := t.TempDir()
	configA := filepath.Join(tmpDir, "a", "helmizer.yaml")
	configB := filepath.Join(tmpDir, "b", "nested", "helmizer.yaml")
	for _, configPath := range []string{configA, configB} {
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(configPath, []byte("helmizer: {}\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	got, err := ResolveConfigPaths(CLIArgs{
		ConfigFilePath: filepath.Join("b", "nested", "helmizer.yaml"),
		ConfigGlob:     "**/helmizer.yaml",
	})
	if err != nil {
		t.Fatalf("ResolveConfigPaths() error = %v", err)
	}

	want := []string{configA, configB}
	if len(got) != len(want) {
		t.Fatalf("ResolveConfigPaths() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ResolveConfigPaths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
