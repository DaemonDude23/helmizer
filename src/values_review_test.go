package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveReleaseValuesFileInfersSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte("replicas: 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := ResolveReleaseValuesFile(HelmfileRelease{
		Name:   "demo",
		Values: []any{"values.yaml"},
	}, tmpDir, "")
	if err != nil {
		t.Fatalf("ResolveReleaseValuesFile() error = %v", err)
	}
	if got != valuesPath {
		t.Fatalf("ResolveReleaseValuesFile() = %q, want %q", got, valuesPath)
	}
}

func TestResolveReleaseValuesFileRequiresOverrideForInlineValues(t *testing.T) {
	_, err := ResolveReleaseValuesFile(HelmfileRelease{
		Name: "demo",
		Values: []any{
			map[string]any{"replicas": 1},
		},
	}, t.TempDir(), "")
	if err == nil || !strings.Contains(err.Error(), "pass --values-file") {
		t.Fatalf("ResolveReleaseValuesFile() error = %v, want --values-file guidance", err)
	}
}

func TestBuildValuesReviewCandidatesCollapsesNestedNewClause(t *testing.T) {
	currentDefaults := []byte(`
image:
  repository: quay.io/example/app
`)
	targetDefaults := []byte(`
image:
  repository: quay.io/example/app
networkPolicy:
  enabled: false
  ingress:
    - ports:
        - port: http
`)
	localValues := []byte(`
image:
  tag: v1.0.0
`)

	got, err := BuildValuesReviewCandidates(currentDefaults, targetDefaults, localValues)
	if err != nil {
		t.Fatalf("BuildValuesReviewCandidates() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("BuildValuesReviewCandidates() len = %d, want 1 (%+v)", len(got), got)
	}
	if got[0].Path != "networkPolicy" {
		t.Fatalf("candidate path = %q, want %q", got[0].Path, "networkPolicy")
	}
	if got[0].Status != "added" {
		t.Fatalf("candidate status = %q, want %q", got[0].Status, "added")
	}
	if got[0].Risk != "high" {
		t.Fatalf("candidate risk = %q, want high", got[0].Risk)
	}
}

func TestBuildValuesReviewCandidatesRanksHighRiskFirst(t *testing.T) {
	currentDefaults := []byte(`
annotations:
  checksum: old
feature:
  enabled: false
replicas: 1
`)
	targetDefaults := []byte(`
annotations:
  checksum: new
feature:
  enabled: true
replicas: 2
`)

	got, err := BuildValuesReviewCandidates(currentDefaults, targetDefaults, nil)
	if err != nil {
		t.Fatalf("BuildValuesReviewCandidates() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("BuildValuesReviewCandidates() len = %d, want 3 (%+v)", len(got), got)
	}
	if got[0].Path != "replicas" || got[0].Risk != "high" {
		t.Fatalf("first candidate = %+v, want high risk replicas", got[0])
	}
	if got[2].Path != "annotations.checksum" || got[2].Risk != "low" {
		t.Fatalf("last candidate = %+v, want low risk annotations.checksum", got[2])
	}
}

func TestReviewValuesFileAcceptsAndDiscardsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	valuesFile := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesFile, []byte("image:\n  tag: v1.0.0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	currentDefaults := []byte(`
image:
  repository: quay.io/example/app
`)
	targetDefaults := []byte(`
feature:
  enabled: true
image:
  name: app
  repository: quay.io/example/app
`)

	var out bytes.Buffer
	err := ReviewValuesFile(ValuesReviewOptions{
		ReleaseName:        "demo",
		ValuesFile:         valuesFile,
		CurrentVersion:     "1.0.0",
		TargetVersion:      "1.1.0",
		CurrentChartValues: currentDefaults,
		TargetChartValues:  targetDefaults,
		In:                 strings.NewReader("a\nd\n"),
		Out:                &out,
	})
	if err != nil {
		t.Fatalf("ReviewValuesFile() error = %v", err)
	}

	updated, err := os.ReadFile(valuesFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(updated), "feature:") {
		t.Fatalf("accepted feature clause was not written:\n%s", updated)
	}
	if strings.Contains(string(updated), "name: app") {
		t.Fatalf("discarded image.name was written:\n%s", updated)
	}
	if !strings.Contains(out.String(), "Inline file diff if accepted") {
		t.Fatalf("review output did not include inline diff:\n%s", out.String())
	}
}
