package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

const exitUpdatesFound = 10

type ChartsArgs struct {
	Check *ChartsCheckArgs `arg:"subcommand:check" help:"Check Helmfile chart versions"`
}

type ChartsCheckArgs struct {
	ConfigGlob           string   `arg:"--config-glob" help:"Glob pattern(s) for Helmizer config files; supports ** and comma-separated values"`
	HelmfilePath         string   `arg:"--helmfile-path" help:"Path to helmfile.yaml, helmfile.yaml.gotmpl, or helmfile.d; defaults to a sibling of the Helmizer config"`
	HelmfileEnvironment  string   `arg:"--helmfile-environment" help:"Helmfile environment to evaluate"`
	HelmfileSelector     []string `arg:"--helmfile-selector" help:"Helmfile selector to apply while building state; can be repeated"`
	StateValuesFile      []string `arg:"--state-values-file" help:"Helmfile state values file; can be repeated"`
	StateValuesSet       []string `arg:"--state-values-set" help:"Helmfile state value override; can be repeated"`
	StateValuesSetString []string `arg:"--state-values-set-string" help:"Helmfile state string value override; can be repeated"`
	Policy               string   `arg:"--policy" default:"same-major" help:"Version policy: same-major, same-minor, all, or constraint"`
	Constraint           string   `arg:"--constraint" help:"Version constraint used when --policy constraint is set, for example 1.x, 1.19.x, or <2.0.0"`
	IncludePreRelease    bool     `arg:"--include-prerelease" help:"Include prerelease chart versions"`
	Output               string   `arg:"--output" default:"table" help:"Output format: table, markdown, json, or yaml"`
	FailOnUpdate         bool     `arg:"--fail-on-update" help:"Exit with status 10 when an allowed update is available"`
	ConfigFilePath       string   `arg:"positional" help:"Path to Helmizer config file"`
}

type ChartCheckResult struct {
	Config                  string `json:"config,omitempty" yaml:"config,omitempty"`
	Helmfile                string `json:"helmfile" yaml:"helmfile"`
	Release                 string `json:"release" yaml:"release"`
	Namespace               string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Chart                   string `json:"chart" yaml:"chart"`
	Repository              string `json:"repository,omitempty" yaml:"repository,omitempty"`
	CurrentVersion          string `json:"currentVersion,omitempty" yaml:"currentVersion,omitempty"`
	CurrentAppVersion       string `json:"currentAppVersion,omitempty" yaml:"currentAppVersion,omitempty"`
	LatestVersion           string `json:"latestVersion,omitempty" yaml:"latestVersion,omitempty"`
	LatestAppVersion        string `json:"latestAppVersion,omitempty" yaml:"latestAppVersion,omitempty"`
	LatestAllowedVersion    string `json:"latestAllowedVersion,omitempty" yaml:"latestAllowedVersion,omitempty"`
	LatestAllowedAppVersion string `json:"latestAllowedAppVersion,omitempty" yaml:"latestAllowedAppVersion,omitempty"`
	SkippedVersion          string `json:"skippedVersion,omitempty" yaml:"skippedVersion,omitempty"`
	Risk                    string `json:"risk,omitempty" yaml:"risk,omitempty"`
	RiskReason              string `json:"riskReason,omitempty" yaml:"riskReason,omitempty"`
	Status                  string `json:"status" yaml:"status"`
	Reason                  string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

func RunCharts(args ChartsArgs) int {
	switch {
	case args.Check != nil:
		return RunChartsCheck(*args.Check)
	default:
		log.Error("missing charts subcommand: use check")
		return 1
	}
}

func RunChartsCheck(args ChartsCheckArgs) int {
	targets, err := ResolveChartTargets(args.ConfigFilePath, args.ConfigGlob, args.HelmfilePath)
	if err != nil {
		log.Error(err)
		return 1
	}

	options := chartCheckHelmfileOptions(args)
	results, hadErrors := evaluateChartTargets(targets, options, args.Policy, args.Constraint, args.IncludePreRelease)
	if err := PrintChartCheckResults(results, args.Output); err != nil {
		log.Error(err)
		return 1
	}

	if hadErrors {
		return 1
	}
	if args.FailOnUpdate && chartResultsHaveUpdates(results) {
		return exitUpdatesFound
	}
	return 0
}

func chartCheckHelmfileOptions(args ChartsCheckArgs) HelmfileCommandOptions {
	return HelmfileCommandOptions{
		HelmfileEnvironment:  args.HelmfileEnvironment,
		HelmfileSelectors:    args.HelmfileSelector,
		StateValuesFiles:     args.StateValuesFile,
		StateValuesSet:       args.StateValuesSet,
		StateValuesSetString: args.StateValuesSetString,
	}
}

func evaluateChartTargets(targets []ChartTarget, options HelmfileCommandOptions, policy string, constraint string, includePreRelease bool) ([]ChartCheckResult, bool) {
	repoCache := map[string]HelmRepositoryIndex{}
	client := &http.Client{Timeout: 30 * time.Second}
	var results []ChartCheckResult
	hadErrors := false

	for _, target := range targets {
		state, _, err := LoadHelmfileState(target, options)
		if err != nil {
			hadErrors = true
			results = append(results, ChartCheckResult{
				Config:   target.ConfigPath,
				Helmfile: target.HelmfilePath,
				Status:   "error",
				Reason:   err.Error(),
			})
			continue
		}
		targetResults, targetHadErrors := EvaluateHelmfileChartUpdates(target, state, repoCache, client, policy, constraint, includePreRelease)
		if targetHadErrors {
			hadErrors = true
		}
		results = append(results, targetResults...)
	}
	return results, hadErrors
}

func EvaluateHelmfileChartUpdates(target ChartTarget, state HelmfileState, repoCache map[string]HelmRepositoryIndex, client *http.Client, policy string, constraint string, includePreRelease bool) ([]ChartCheckResult, bool) {
	var results []ChartCheckResult
	hadErrors := false

	for _, release := range state.Releases {
		result := ChartCheckResult{
			Config:         target.ConfigPath,
			Helmfile:       target.HelmfilePath,
			Release:        release.Name,
			Namespace:      release.Namespace,
			Chart:          release.Chart,
			CurrentVersion: release.Version,
		}

		if release.Installed != nil && !*release.Installed {
			result.Status = "skipped"
			result.Reason = "release is marked installed: false"
			results = append(results, result)
			continue
		}
		if strings.TrimSpace(release.Version) == "" {
			result.Status = "skipped"
			result.Reason = "release has no chart version"
			results = append(results, result)
			continue
		}

		resolved, reason := ResolveHelmfileReleaseChart(release, state.Repositories, target.BaseDir)
		if reason != "" {
			result.Status = "skipped"
			result.Reason = reason
			results = append(results, result)
			continue
		}
		result.Repository = resolved.RepositoryURL

		index, err := cachedRepositoryIndex(repoCache, client, resolved.RepositoryURL)
		if err != nil {
			hadErrors = true
			result.Status = "error"
			result.Reason = err.Error()
			results = append(results, result)
			continue
		}

		versions := index.Entries[resolved.ChartName]
		if len(versions) == 0 {
			result.Status = "skipped"
			result.Reason = fmt.Sprintf("chart %q was not found in repository index", resolved.ChartName)
			results = append(results, result)
			continue
		}

		selection, err := SelectChartVersion(versions, release.Version, policy, constraint, includePreRelease)
		if err != nil {
			result.Status = "skipped"
			result.Reason = err.Error()
			results = append(results, result)
			continue
		}

		result.CurrentAppVersion = selection.CurrentAppVersion
		result.LatestVersion = selection.Latest
		result.LatestAppVersion = selection.LatestAppVersion
		result.LatestAllowedVersion = selection.LatestAllowed
		result.LatestAllowedAppVersion = selection.LatestAllowedAppVersion
		result.SkippedVersion = selection.LatestSkipped
		switch {
		case selection.LatestAllowed != "":
			result.Status = "update-available"
			result.Risk, result.RiskReason = chartVersionUpdateRisk(release.Version, selection.LatestAllowed)
		case selection.LatestSkipped != "":
			result.Status = "version-skipped"
			result.Reason = "newer version exists outside the selected policy"
		default:
			result.Status = "current"
		}
		results = append(results, result)
	}

	return results, hadErrors
}

func cachedRepositoryIndex(cache map[string]HelmRepositoryIndex, client *http.Client, repoURL string) (HelmRepositoryIndex, error) {
	if index, ok := cache[repoURL]; ok {
		return index, nil
	}
	index, err := FetchHelmRepositoryIndexWithClient(client, repoURL)
	if err != nil {
		return HelmRepositoryIndex{}, err
	}
	cache[repoURL] = index
	return index, nil
}

func PrintChartCheckResults(results []ChartCheckResult, output string) error {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "table":
		printChartCheckTable(results)
	case "markdown", "md":
		printChartCheckMarkdown(results)
	case "json":
		encoded, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
	case "yaml":
		encoded, err := yaml.Marshal(results)
		if err != nil {
			return err
		}
		fmt.Print(string(encoded))
	default:
		return fmt.Errorf("invalid --output %q; expected table, markdown, json, or yaml", output)
	}
	return nil
}

func printChartCheckTable(results []ChartCheckResult) {
	for _, group := range groupChartResults(results) {
		fmt.Println(formatChartGroupHeader(group))
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "RELEASE\tCHART\tCURRENT\tALLOWED\tLATEST\tRISK\tSTATUS\tNOTES")
		for _, result := range group.results {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				result.Release,
				result.Chart,
				versionPair(result.CurrentVersion, result.CurrentAppVersion),
				versionPair(result.LatestAllowedVersion, result.LatestAllowedAppVersion),
				versionPair(result.LatestVersion, result.LatestAppVersion),
				result.Risk,
				result.Status,
				chartResultNotes(result),
			)
		}
		_ = writer.Flush()
		fmt.Println()
	}
}

func printChartCheckMarkdown(results []ChartCheckResult) {
	for i, group := range groupChartResults(results) {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("### %s\n\n", formatChartGroupHeader(group))
		fmt.Println("| Release | Chart | Current (app) | Allowed (app) | Latest (app) | Risk | Status | Notes |")
		fmt.Println("| --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, result := range group.results {
			fmt.Printf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
				mdEscape(result.Release),
				mdEscape(result.Chart),
				mdEscape(versionPair(result.CurrentVersion, result.CurrentAppVersion)),
				mdEscape(versionPair(result.LatestAllowedVersion, result.LatestAllowedAppVersion)),
				mdEscape(versionPair(result.LatestVersion, result.LatestAppVersion)),
				mdEscape(result.Risk),
				mdEscape(result.Status),
				mdEscape(chartResultNotes(result)),
			)
		}
	}
}

type chartResultGroup struct {
	config   string
	helmfile string
	results  []ChartCheckResult
}

func groupChartResults(results []ChartCheckResult) []chartResultGroup {
	var groups []chartResultGroup
	keyToIndex := map[string]int{}
	for _, result := range results {
		key := result.Config + "\x00" + result.Helmfile
		if idx, ok := keyToIndex[key]; ok {
			groups[idx].results = append(groups[idx].results, result)
			continue
		}
		keyToIndex[key] = len(groups)
		groups = append(groups, chartResultGroup{
			config:   result.Config,
			helmfile: result.Helmfile,
			results:  []ChartCheckResult{result},
		})
	}
	return groups
}

func formatChartGroupHeader(group chartResultGroup) string {
	config := compactPath(group.config)
	helmfile := compactPath(group.helmfile)
	switch {
	case config != "" && helmfile != "" && filepath.Dir(group.config) == filepath.Dir(group.helmfile):
		return fmt.Sprintf("%s (%s)", config, filepath.Base(helmfile))
	case config != "" && helmfile != "":
		return fmt.Sprintf("%s (helmfile: %s)", config, helmfile)
	case config != "":
		return config
	case helmfile != "":
		return helmfile
	default:
		return "(unknown source)"
	}
}

func compactPath(path string) string {
	if path == "" {
		return ""
	}
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, path); err == nil {
			return rel
		}
	}
	return path
}

func chartResultNotes(result ChartCheckResult) string {
	parts := []string{}
	if result.Reason != "" {
		parts = append(parts, result.Reason)
	} else if result.RiskReason != "" {
		parts = append(parts, result.RiskReason)
	}
	if result.SkippedVersion != "" && result.LatestAllowedVersion != result.SkippedVersion {
		parts = append(parts, fmt.Sprintf("skipped %s by policy", result.SkippedVersion))
	}
	return strings.Join(parts, "; ")
}

func versionPair(version string, appVersion string) string {
	if version == "" {
		return ""
	}
	if appVersion == "" || appVersion == version {
		return version
	}
	return fmt.Sprintf("%s (%s)", version, appVersion)
}

func mdEscape(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func chartVersionUpdateRisk(currentVersion string, targetVersion string) (string, string) {
	current, currentErr := ParseSemVer(currentVersion)
	target, targetErr := ParseSemVer(targetVersion)
	if currentErr != nil || targetErr != nil {
		return "medium", "chart version changed"
	}

	switch {
	case target.Major != current.Major:
		return "high", "major chart version change"
	case target.Minor != current.Minor:
		return "medium", "minor chart version change"
	default:
		return "low", "patch chart version change"
	}
}

func chartResultsHaveUpdates(results []ChartCheckResult) bool {
	for _, result := range results {
		if result.Status == "update-available" {
			return true
		}
	}
	return false
}
