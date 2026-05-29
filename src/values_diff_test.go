package main

import (
	"strings"
	"testing"
)

func TestDiffYAMLValuePaths(t *testing.T) {
	oldData := []byte(`
image:
  repository: quay.io/example/app
  tag: v1.0.0
resources: {}
tolerations: []
`)
	newData := []byte(`
image:
  repository: quay.io/example/app
  tag: v1.1.0
resources:
  requests:
    cpu: 10m
replicas: 2
`)

	got, err := DiffYAMLValuePaths("old values", oldData, "new values", newData)
	if err != nil {
		t.Fatalf("DiffYAMLValuePaths() error = %v", err)
	}

	for _, want := range []string{
		"--- old values\n",
		"+++ new values\n",
		"Summary: 3 high, 2 medium, 0 low\n",
		"medium  changed  image.tag",
		"medium  removed  resources",
		"high    added    resources.requests.cpu",
		"high    added    replicas",
		"high    removed  tolerations",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("DiffYAMLValuePaths() missing %q in:\n%s", want, got)
		}
	}
}

func TestDiffYAMLValuePathsRanksRiskBeforePath(t *testing.T) {
	oldData := []byte(`
annotations:
  checksum: old
feature:
  enabled: false
replicas: 1
`)
	newData := []byte(`
annotations:
  checksum: new
feature:
  enabled: true
replicas: 2
`)

	got, err := DiffYAMLValuePaths("old values", oldData, "new values", newData)
	if err != nil {
		t.Fatalf("DiffYAMLValuePaths() error = %v", err)
	}

	highIndex := strings.Index(got, "high    changed  replicas")
	mediumIndex := strings.Index(got, "medium  changed  feature.enabled")
	lowIndex := strings.Index(got, "low     changed  annotations.checksum")
	if highIndex < 0 || mediumIndex < 0 || lowIndex < 0 {
		t.Fatalf("DiffYAMLValuePaths() missing ranked rows:\n%s", got)
	}
	if !(highIndex < mediumIndex && mediumIndex < lowIndex) {
		t.Fatalf("DiffYAMLValuePaths() did not rank rows by risk:\n%s", got)
	}
}

func TestDiffYAMLValuePathsReturnsEmptyForEqualValuesWithDifferentComments(t *testing.T) {
	oldData := []byte("# old comment\nreplicas: 1\n")
	newData := []byte("# new comment\nreplicas: 1\n")

	got, err := DiffYAMLValuePaths("old values", oldData, "new values", newData)
	if err != nil {
		t.Fatalf("DiffYAMLValuePaths() error = %v", err)
	}
	if got != "" {
		t.Fatalf("DiffYAMLValuePaths() = %q, want empty", got)
	}
}
