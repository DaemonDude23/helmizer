package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestSelectChartVersionHonorsPolicyAndSkipsPrereleases(t *testing.T) {
	versions := []HelmChartVersion{
		{Version: "2.0.0", AppVersion: "2.0.0"},
		{Version: "1.3.0", AppVersion: "1.3.0"},
		{Version: "1.2.4-alpha.1", AppVersion: "1.2.4-alpha.1"},
		{Version: "1.2.4", AppVersion: "1.2.4"},
		{Version: "1.2.3", AppVersion: "1.2.3"},
	}

	got, err := SelectChartVersion(versions, "1.2.3", "same-minor", "", false)
	if err != nil {
		t.Fatalf("SelectChartVersion() error = %v", err)
	}

	if got.Latest != "2.0.0" {
		t.Fatalf("Latest = %q, want %q", got.Latest, "2.0.0")
	}
	if got.LatestAllowed != "1.2.4" {
		t.Fatalf("LatestAllowed = %q, want %q", got.LatestAllowed, "1.2.4")
	}
	if got.LatestAllowedAppVersion != "1.2.4" {
		t.Fatalf("LatestAllowedAppVersion = %q, want %q", got.LatestAllowedAppVersion, "1.2.4")
	}
	if got.LatestSkipped != "2.0.0" {
		t.Fatalf("LatestSkipped = %q, want %q", got.LatestSkipped, "2.0.0")
	}
	if got.CurrentAppVersion != "1.2.3" {
		t.Fatalf("CurrentAppVersion = %q, want %q", got.CurrentAppVersion, "1.2.3")
	}
}

func TestPrintChartCheckResultsMarkdown(t *testing.T) {
	results := []ChartCheckResult{{
		Release:                 "cert-manager",
		Chart:                   "jetstack/cert-manager",
		CurrentVersion:          "1.19.2",
		CurrentAppVersion:       "v1.19.2",
		LatestVersion:           "2.0.0",
		LatestAppVersion:        "v2.0.0",
		LatestAllowedVersion:    "1.20.0",
		LatestAllowedAppVersion: "v1.20.0",
		Risk:                    "medium",
		Status:                  "update-available",
	}}

	stdout := os.Stdout
	read, write, _ := os.Pipe()
	os.Stdout = write
	err := PrintChartCheckResults(results, "markdown")
	_ = write.Close()
	os.Stdout = stdout
	if err != nil {
		t.Fatalf("PrintChartCheckResults() error = %v", err)
	}
	buf, _ := io.ReadAll(read)
	output := string(buf)

	for _, want := range []string{
		"| Release |",
		"| cert-manager |",
		"1.19.2 (v1.19.2)",
		"1.20.0 (v1.20.0)",
		"2.0.0 (v2.0.0)",
		"update-available",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, output)
		}
	}
}

func TestVersionAllowedByPolicyConstraint(t *testing.T) {
	current, err := ParseSemVer("1.2.3")
	if err != nil {
		t.Fatalf("ParseSemVer(current) error = %v", err)
	}
	candidate, err := ParseSemVer("1.3.5")
	if err != nil {
		t.Fatalf("ParseSemVer(candidate) error = %v", err)
	}

	if !VersionAllowedByPolicy(current, candidate, "constraint", ">=1.3.0 <2.0.0") {
		t.Fatal("VersionAllowedByPolicy() = false, want true")
	}
	if VersionAllowedByPolicy(current, candidate, "constraint", "1.2.x") {
		t.Fatal("VersionAllowedByPolicy() = true, want false")
	}
}

func TestResolveHelmfileReleaseChart(t *testing.T) {
	got, reason := ResolveHelmfileReleaseChart(
		HelmfileRelease{Chart: "jetstack/cert-manager"},
		[]HelmfileRepository{{Name: "jetstack", URL: "https://charts.jetstack.io"}},
		t.TempDir(),
	)
	if reason != "" {
		t.Fatalf("ResolveHelmfileReleaseChart() reason = %q, want empty", reason)
	}
	if got.ChartName != "cert-manager" {
		t.Fatalf("ChartName = %q, want %q", got.ChartName, "cert-manager")
	}
	if got.RepositoryURL != "https://charts.jetstack.io" {
		t.Fatalf("RepositoryURL = %q, want %q", got.RepositoryURL, "https://charts.jetstack.io")
	}
}

func TestReplaceReleaseVersionInHelmfileState(t *testing.T) {
	input := []byte(`
repositories:
  - name: jetstack
    url: https://charts.jetstack.io
releases:
  - chart: jetstack/cert-manager
    name: cert-manager
    version: 1.19.2
  - chart: jetstack/trust-manager
    name: trust-manager
    version: 0.19.0
`)

	output, err := ReplaceReleaseVersionInHelmfileState(input, "cert-manager", "1.20.0")
	if err != nil {
		t.Fatalf("ReplaceReleaseVersionInHelmfileState() error = %v", err)
	}
	state, err := ParseHelmfileState(output)
	if err != nil {
		t.Fatalf("ParseHelmfileState() error = %v", err)
	}

	release, found := findHelmfileRelease(state.Releases, "cert-manager")
	if !found {
		t.Fatal("cert-manager release was not found")
	}
	if release.Version != "1.20.0" {
		t.Fatalf("cert-manager version = %q, want %q", release.Version, "1.20.0")
	}

	otherRelease, found := findHelmfileRelease(state.Releases, "trust-manager")
	if !found {
		t.Fatal("trust-manager release was not found")
	}
	if otherRelease.Version != "0.19.0" {
		t.Fatalf("trust-manager version = %q, want %q", otherRelease.Version, "0.19.0")
	}
}

func TestChartVersionUpdateRisk(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		risk    string
	}{
		{name: "major", current: "1.2.3", target: "2.0.0", risk: "high"},
		{name: "minor", current: "1.2.3", target: "1.3.0", risk: "medium"},
		{name: "patch", current: "1.2.3", target: "1.2.4", risk: "low"},
		{name: "unparseable", current: "2024.04", target: "2024.05", risk: "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := chartVersionUpdateRisk(tt.current, tt.target)
			if got != tt.risk {
				t.Fatalf("chartVersionUpdateRisk() risk = %q, want %q (reason %q)", got, tt.risk, reason)
			}
			if reason == "" {
				t.Fatal("chartVersionUpdateRisk() reason was empty")
			}
		})
	}
}

func TestIsChartsCommand(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{name: "charts subcommand", argv: []string{"charts", "check"}, want: true},
		{name: "global flag before charts", argv: []string{"--log-level", "DEBUG", "charts", "check"}, want: true},
		{name: "ordinary config path", argv: []string{"helmizer.yaml"}, want: false},
		{name: "charts as config glob value", argv: []string{"--config-glob", "charts"}, want: false},
		{name: "after separator", argv: []string{"--", "charts"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isChartsCommand(tt.argv); got != tt.want {
				t.Fatalf("isChartsCommand(%v) = %t, want %t", tt.argv, got, tt.want)
			}
		})
	}
}
