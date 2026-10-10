package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

type indexTransport func(*http.Request) (*http.Response, error)

func (f indexTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type repeatReader struct{}

func (repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestRepositoryIndexRejectsOversizedResponse(t *testing.T) {
	client := &http.Client{Transport: indexTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(repeatReader{})}, nil
	})}
	_, err := FetchHelmRepositoryIndexWithClient(client, "https://example.invalid")
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}

func TestGenerationPreservesNamespaceAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "helmizer.yaml")
	if err := os.WriteFile(config, []byte("helmizer: {}\nkustomize:\n  namespace: integration-test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := CLIArgs{ConfigFilePath: config, KustomizationPath: ".", ApiVersion: "kustomize.config.k8s.io/v1beta1", SkipAllCommands: true}
	RunHelmizer(args)
	output := filepath.Join(dir, "kustomization.yaml")
	first, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]interface{}
	if err := yaml.Unmarshal(first, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["namespace"] != "integration-test" || manifest["kind"] != "Kustomization" {
		t.Fatalf("unexpected manifest: %v", manifest)
	}
	RunHelmizer(args)
	second, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated generation changed output")
	}
	args.DryRun = true
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	RunHelmizer(args)
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("dry run created output: %v", err)
	}
}

func TestCommandFailureReturnsErrorAndStopsSequence(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "unexpected")
	_, err := ExecuteCommands("pre", CLIArgs{ConfigFilePath: filepath.Join(dir, "helmizer.yaml")}, Helmizer{PreCommands: []Commands{
		{Command: "sh", Arguments: []string{"-c", "exit 7"}},
		{Command: "touch", Arguments: []string{marker}},
	}})
	if err == nil {
		t.Fatal("expected command failure")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("commands continued after failure")
	}
}
