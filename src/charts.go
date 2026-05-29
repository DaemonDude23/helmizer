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
	Check  *ChartsCheckArgs  `arg:"subcommand:check" help:"Check Helmfile chart versions"`
	Diff   *ChartsDiffArgs   `arg:"subcommand:diff" help:"Diff Helmfile chart values or rendered manifests"`
	Review *ChartsReviewArgs `arg:"subcommand:review" help:"Interactively review chart value changes into a values file"`
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

type ChartsDiffArgs struct {
	ConfigGlob           string   `arg:"--config-glob" help:"Glob pattern(s) for Helmizer config files; supports ** and comma-separated values"`
	HelmfilePath         string   `arg:"--helmfile-path" help:"Path to helmfile.yaml, helmfile.yaml.gotmpl, or helmfile.d; defaults to a sibling of the Helmizer config"`
	HelmfileEnvironment  string   `arg:"--helmfile-environment" help:"Helmfile environment to evaluate"`
	HelmfileSelector     []string `arg:"--helmfile-selector" help:"Helmfile selector to apply while building state; can be repeated"`
	StateValuesFile      []string `arg:"--state-values-file" help:"Helmfile state values file; can be repeated"`
	StateValuesSet       []string `arg:"--state-values-set" help:"Helmfile state value override; can be repeated"`
	StateValuesSetString []string `arg:"--state-values-set-string" help:"Helmfile state string value override; can be repeated"`
	Policy               string   `arg:"--policy" default:"same-major" help:"Version policy used when --to latest is set: same-major, same-minor, all, or constraint"`
	Constraint           string   `arg:"--constraint" help:"Version constraint used when --policy constraint is set"`
	IncludePreRelease    bool     `arg:"--include-prerelease" help:"Include prerelease chart versions when --to latest is set"`
	Release              string   `arg:"--release,required" help:"Helmfile release name to diff"`
	To                   string   `arg:"--to" default:"latest" help:"Target chart version or latest"`
	Kind                 string   `arg:"--kind" default:"both" help:"Diff kind: values, manifests, or both"`
	ValuesMode           string   `arg:"--values-mode" default:"paths" help:"Values diff mode: paths or text"`
	ConfigFilePath       string   `arg:"positional" help:"Path to Helmizer config file"`
}

type ChartsReviewArgs struct {
	ConfigGlob           string   `arg:"--config-glob" help:"Glob pattern(s) for Helmizer config files; supports ** and comma-separated values"`
	HelmfilePath         string   `arg:"--helmfile-path" help:"Path to helmfile.yaml, helmfile.yaml.gotmpl, or helmfile.d; defaults to a sibling of the Helmizer config"`
	HelmfileEnvironment  string   `arg:"--helmfile-environment" help:"Helmfile environment to evaluate"`
	HelmfileSelector     []string `arg:"--helmfile-selector" help:"Helmfile selector to apply while building state; can be repeated"`
	StateValuesFile      []string `arg:"--state-values-file" help:"Helmfile state values file; can be repeated"`
	StateValuesSet       []string `arg:"--state-values-set" help:"Helmfile state value override; can be repeated"`
	StateValuesSetString []string `arg:"--state-values-set-string" help:"Helmfile state string value override; can be repeated"`
	Policy               string   `arg:"--policy" default:"same-major" help:"Version policy used when --to latest is set: same-major, same-minor, all, or constraint"`
	Constraint           string   `arg:"--constraint" help:"Version constraint used when --policy constraint is set"`
	IncludePreRelease    bool     `arg:"--include-prerelease" help:"Include prerelease chart versions when --to latest is set"`
	Release              string   `arg:"--release,required" help:"Helmfile release name to review"`
	To                   string   `arg:"--to" default:"latest" help:"Target chart version or latest"`
	ValuesFile           string   `arg:"--values-file" help:"Local Helm values file to update; defaults to the release's single values file entry"`
	DryRun               bool     `arg:"--dry-run" help:"Review changes without writing the values file"`
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
	case args.Diff != nil:
		return RunChartsDiff(*args.Diff)
	case args.Review != nil:
		return RunChartsReview(*args.Review)
	default:
		log.Error("missing charts subcommand: use check, diff, or review")
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

func RunChartsDiff(args ChartsDiffArgs) int {
	targets, err := ResolveChartTargets(args.ConfigFilePath, args.ConfigGlob, args.HelmfilePath)
	if err != nil {
		log.Error(err)
		return 1
	}

	kind := strings.ToLower(strings.TrimSpace(args.Kind))
	if kind == "" {
		kind = "both"
	}
	if kind != "both" && kind != "values" && kind != "manifests" {
		log.Errorf("invalid --kind %q; expected values, manifests, or both", args.Kind)
		return 1
	}
	valuesMode := strings.ToLower(strings.TrimSpace(args.ValuesMode))
	if valuesMode == "" {
		valuesMode = "paths"
	}
	if valuesMode != "paths" && valuesMode != "text" {
		log.Errorf("invalid --values-mode %q; expected paths or text", args.ValuesMode)
		return 1
	}

	options := chartDiffHelmfileOptions(args)
	repoCache := map[string]HelmRepositoryIndex{}
	client := &http.Client{Timeout: 30 * time.Second}
	hadDiff := false

	for _, target := range targets {
		state, buildOutput, err := LoadHelmfileState(target, options)
		if err != nil {
			log.Error(err)
			return 1
		}

		release, found := findHelmfileRelease(state.Releases, args.Release)
		if !found {
			log.Errorf("release %q was not found in %s", args.Release, target.HelmfilePath)
			return 1
		}

		resolved, reason := ResolveHelmfileReleaseChart(release, state.Repositories, target.BaseDir)
		if reason != "" {
			log.Errorf("release %q cannot be diffed: %s", release.Name, reason)
			return 1
		}

		targetVersion := strings.TrimSpace(args.To)
		if targetVersion == "" || strings.EqualFold(targetVersion, "latest") {
			targetVersion, err = resolveLatestTargetVersion(repoCache, client, resolved, release.Version, args.Policy, args.Constraint, args.IncludePreRelease)
			if err != nil {
				log.Error(err)
				return 1
			}
		}

		if len(targets) > 1 {
			fmt.Printf("## %s (%s)\n", target.ConfigPath, target.HelmfilePath)
		}

		if kind == "both" || kind == "values" {
			diff, err := DiffChartValues(target, resolved, release.Version, targetVersion, valuesMode)
			if err != nil {
				log.Error(err)
				return 1
			}
			if diff != "" {
				fmt.Print(diff)
				hadDiff = true
			} else {
				fmt.Printf("No values changes for %s from %s to %s\n", release.Name, release.Version, targetVersion)
			}
		}

		if kind == "both" || kind == "manifests" {
			diff, err := DiffHelmfileManifests(target, options, buildOutput, release.Name, release.Version, targetVersion)
			if err != nil {
				log.Error(err)
				return 1
			}
			if diff != "" {
				fmt.Print(diff)
				hadDiff = true
			} else {
				fmt.Printf("No manifest changes for %s from %s to %s\n", release.Name, release.Version, targetVersion)
			}
		}
	}

	if !hadDiff {
		return 0
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

func chartDiffHelmfileOptions(args ChartsDiffArgs) HelmfileCommandOptions {
	return HelmfileCommandOptions{
		HelmfileEnvironment:  args.HelmfileEnvironment,
		HelmfileSelectors:    args.HelmfileSelector,
		StateValuesFiles:     args.StateValuesFile,
		StateValuesSet:       args.StateValuesSet,
		StateValuesSetString: args.StateValuesSetString,
	}
}

func chartReviewHelmfileOptions(args ChartsReviewArgs) HelmfileCommandOptions {
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

func findHelmfileRelease(releases []HelmfileRelease, name string) (HelmfileRelease, bool) {
	for _, release := range releases {
		if release.Name == name {
			return release, true
		}
	}
	return HelmfileRelease{}, false
}

func resolveLatestTargetVersion(cache map[string]HelmRepositoryIndex, client *http.Client, resolved ResolvedChart, currentVersion string, policy string, constraint string, includePreRelease bool) (string, error) {
	index, err := cachedRepositoryIndex(cache, client, resolved.RepositoryURL)
	if err != nil {
		return "", err
	}
	versions := index.Entries[resolved.ChartName]
	if len(versions) == 0 {
		return "", fmt.Errorf("chart %q was not found in repository index", resolved.ChartName)
	}
	selection, err := SelectChartVersion(versions, currentVersion, policy, constraint, includePreRelease)
	if err != nil {
		return "", err
	}
	if selection.LatestAllowed == "" {
		if selection.LatestSkipped != "" {
			return "", fmt.Errorf("%s is already at the latest version allowed by policy %q (newer %s exists outside policy)", currentVersion, policy, selection.LatestSkipped)
		}
		return "", fmt.Errorf("%s is already at the latest version", currentVersion)
	}
	return selection.LatestAllowed, nil
}

func DiffChartValues(target ChartTarget, resolved ResolvedChart, currentVersion string, targetVersion string, valuesMode string) (string, error) {
	currentValues, err := runHelmShowValues(target.BaseDir, resolved, currentVersion)
	if err != nil {
		return "", err
	}
	targetValues, err := runHelmShowValues(target.BaseDir, resolved, targetVersion)
	if err != nil {
		return "", err
	}
	if valuesMode == "paths" {
		return DiffYAMLValuePaths(
			fmt.Sprintf("%s values %s", resolved.ChartName, currentVersion),
			currentValues,
			fmt.Sprintf("%s values %s", resolved.ChartName, targetVersion),
			targetValues,
		)
	}
	return UnifiedDiff(
		fmt.Sprintf("%s values %s", resolved.ChartName, currentVersion),
		string(currentValues),
		fmt.Sprintf("%s values %s", resolved.ChartName, targetVersion),
		string(targetValues),
	), nil
}

func runHelmShowValues(workDir string, resolved ResolvedChart, version string) ([]byte, error) {
	args := []string{"show", "values", resolved.ChartName, "--repo", resolved.RepositoryURL, "--version", version}
	// Run outside the Helmfile directory so generated chart output directories
	// cannot shadow the remote chart name.
	helmWorkDir := os.TempDir()
	if helmWorkDir == "" {
		helmWorkDir = workDir
	}
	stdout, stderr, err := runCommand(helmWorkDir, "helm", args...)
	if err != nil {
		return nil, fmt.Errorf("helm show values failed for %s %s: %w\n%s", resolved.ChartName, version, err, strings.TrimSpace(stderr))
	}
	return stdout, nil
}

func DiffHelmfileManifests(target ChartTarget, options HelmfileCommandOptions, buildOutput []byte, releaseName string, currentVersion string, targetVersion string) (string, error) {
	currentManifest, err := runHelmfileTemplate(target, options, releaseName)
	if err != nil {
		return "", err
	}

	targetState, err := ReplaceReleaseVersionInHelmfileState(buildOutput, releaseName, targetVersion)
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(target.BaseDir, ".helmizer-chart-diff-*.yaml")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmpFile.Write(targetState); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	targetManifest, err := runHelmfileTemplate(ChartTarget{
		BaseDir:      target.BaseDir,
		HelmfilePath: tmpPath,
	}, options, releaseName)
	if err != nil {
		return "", err
	}

	return UnifiedDiff(
		fmt.Sprintf("%s manifests %s", releaseName, currentVersion),
		string(currentManifest),
		fmt.Sprintf("%s manifests %s", releaseName, targetVersion),
		string(targetManifest),
	), nil
}

func runHelmfileTemplate(target ChartTarget, options HelmfileCommandOptions, releaseName string) ([]byte, error) {
	args := helmfileGlobalArgs(target.HelmfilePath, options, []string{"name=" + releaseName})
	args = append(args, "template")
	args = append(args, options.AdditionalTemplateArgs...)
	stdout, stderr, err := runCommand(target.BaseDir, "helmfile", args...)
	if err != nil {
		return nil, fmt.Errorf("helmfile template failed for %s release %s: %w\n%s", target.HelmfilePath, releaseName, err, strings.TrimSpace(stderr))
	}
	return stdout, nil
}
