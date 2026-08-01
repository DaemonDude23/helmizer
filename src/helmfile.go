package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type ChartTarget struct {
	ConfigPath   string `json:"config" yaml:"config"`
	BaseDir      string `json:"-" yaml:"-"`
	HelmfilePath string `json:"helmfile" yaml:"helmfile"`
}

type HelmfileCommandOptions struct {
	HelmfileEnvironment    string
	HelmfileSelectors      []string
	StateValuesFiles       []string
	StateValuesSet         []string
	StateValuesSetString   []string
	AdditionalHelmfileArgs []string
}

type HelmfileState struct {
	Filepath     string               `yaml:"filepath,omitempty" json:"filepath,omitempty"`
	Repositories []HelmfileRepository `yaml:"repositories,omitempty" json:"repositories,omitempty"`
	Releases     []HelmfileRelease    `yaml:"releases,omitempty" json:"releases,omitempty"`
}

type HelmfileRepository struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
}

type HelmfileRelease struct {
	Chart     string            `yaml:"chart" json:"chart"`
	Version   string            `yaml:"version" json:"version"`
	Name      string            `yaml:"name" json:"name"`
	Namespace string            `yaml:"namespace" json:"namespace"`
	Installed *bool             `yaml:"installed,omitempty" json:"installed,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Values    []any             `yaml:"values,omitempty" json:"values,omitempty"`
	Raw       map[string]any    `yaml:",inline" json:"-"`
}

type ResolvedChart struct {
	ChartName     string
	RepositoryURL string
}

func ResolveChartTargets(configFilePath string, configGlob string, helmfilePath string) ([]ChartTarget, error) {
	if strings.TrimSpace(configFilePath) == "" && strings.TrimSpace(configGlob) == "" {
		if strings.TrimSpace(helmfilePath) != "" {
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			resolved, err := resolveHelmfilePath(wd, helmfilePath)
			if err != nil {
				return nil, err
			}
			return []ChartTarget{{BaseDir: wd, HelmfilePath: resolved}}, nil
		}
		if _, err := os.Stat("helmizer.yaml"); err == nil {
			configFilePath = "helmizer.yaml"
		}
	}

	configPaths, err := ResolveConfigPaths(CLIArgs{
		ConfigFilePath: configFilePath,
		ConfigGlob:     configGlob,
	})
	if err != nil {
		return nil, err
	}

	var targets []ChartTarget
	for _, configPath := range configPaths {
		baseDir := filepath.Dir(configPath)
		resolved, err := findHelmfileForConfig(baseDir, helmfilePath)
		if err != nil {
			return nil, err
		}
		targets = append(targets, ChartTarget{
			ConfigPath:   configPath,
			BaseDir:      baseDir,
			HelmfilePath: resolved,
		})
	}
	return targets, nil
}

func findHelmfileForConfig(baseDir string, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return resolveHelmfilePath(baseDir, override)
	}

	for _, candidate := range []string{"helmfile.yaml", "helmfile.yaml.gotmpl", "helmfile.d"} {
		fullPath := filepath.Join(baseDir, candidate)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if candidate == "helmfile.d" && !info.IsDir() {
			continue
		}
		if candidate != "helmfile.d" && !info.Mode().IsRegular() {
			continue
		}
		return fullPath, nil
	}

	return "", fmt.Errorf("no Helmfile state found next to %s", filepath.Join(baseDir, "helmizer.yaml"))
}

func resolveHelmfilePath(baseDir string, rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", fmt.Errorf("empty Helmfile path")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("Helmfile path not found: %s", path)
	}
	return path, nil
}

func LoadHelmfileState(target ChartTarget, options HelmfileCommandOptions) (HelmfileState, []byte, error) {
	args := helmfileGlobalArgs(target.HelmfilePath, options, nil)
	args = append(args, "build")

	stdout, stderr, err := runCommand(target.BaseDir, "helmfile", args...)
	if err != nil {
		return HelmfileState{}, nil, fmt.Errorf("helmfile build failed for %s: %w\n%s", target.HelmfilePath, err, strings.TrimSpace(stderr))
	}

	state, err := ParseHelmfileState(stdout)
	if err != nil {
		return HelmfileState{}, stdout, fmt.Errorf("failed to parse helmfile build output for %s: %w", target.HelmfilePath, err)
	}
	return state, stdout, nil
}

func ParseHelmfileState(data []byte) (HelmfileState, error) {
	var state HelmfileState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return HelmfileState{}, err
	}
	return state, nil
}

func ResolveHelmfileReleaseChart(release HelmfileRelease, repositories []HelmfileRepository, baseDir string) (ResolvedChart, string) {
	chartRef := strings.TrimSpace(release.Chart)
	if chartRef == "" {
		return ResolvedChart{}, "missing chart"
	}
	if strings.HasPrefix(chartRef, "oci://") {
		return ResolvedChart{}, "OCI charts are not supported in v1"
	}
	if strings.HasPrefix(chartRef, "http://") || strings.HasPrefix(chartRef, "https://") {
		return ResolvedChart{}, "direct chart URLs are not supported in v1"
	}
	if isLocalChartRef(chartRef, baseDir) {
		return ResolvedChart{}, "local chart paths are not supported in v1"
	}

	parts := strings.SplitN(chartRef, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ResolvedChart{}, "chart repository alias is missing"
	}

	repoURL := ""
	for _, repo := range repositories {
		if repo.Name == parts[0] {
			repoURL = strings.TrimSpace(repo.URL)
			break
		}
	}
	if repoURL == "" {
		return ResolvedChart{}, fmt.Sprintf("repository alias %q was not found", parts[0])
	}
	if strings.HasPrefix(repoURL, "oci://") {
		return ResolvedChart{}, "OCI chart repositories are not supported in v1"
	}
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		return ResolvedChart{}, "only HTTP(S) chart repositories are supported in v1"
	}

	return ResolvedChart{ChartName: parts[1], RepositoryURL: repoURL}, ""
}

func isLocalChartRef(chartRef string, baseDir string) bool {
	if strings.HasPrefix(chartRef, ".") || strings.HasPrefix(chartRef, "/") || strings.HasPrefix(chartRef, "~") {
		return true
	}
	if strings.Contains(chartRef, "/") {
		return false
	}
	if _, err := os.Stat(filepath.Join(baseDir, chartRef)); err == nil {
		return true
	}
	return false
}

func helmfileGlobalArgs(helmfilePath string, options HelmfileCommandOptions, extraSelectors []string) []string {
	args := []string{"-f", helmfilePath}
	if options.HelmfileEnvironment != "" {
		args = append(args, "-e", options.HelmfileEnvironment)
	}
	for _, selector := range options.HelmfileSelectors {
		if strings.TrimSpace(selector) != "" {
			args = append(args, "--selector", selector)
		}
	}
	for _, selector := range extraSelectors {
		if strings.TrimSpace(selector) != "" {
			args = append(args, "--selector", selector)
		}
	}
	for _, path := range options.StateValuesFiles {
		if strings.TrimSpace(path) != "" {
			args = append(args, "--state-values-file", path)
		}
	}
	for _, value := range options.StateValuesSet {
		if strings.TrimSpace(value) != "" {
			args = append(args, "--state-values-set", value)
		}
	}
	for _, value := range options.StateValuesSetString {
		if strings.TrimSpace(value) != "" {
			args = append(args, "--state-values-set-string", value)
		}
	}
	args = append(args, options.AdditionalHelmfileArgs...)
	return args
}

func runCommand(workDir string, name string, args ...string) ([]byte, string, error) {
	cmd := exec.Command(name, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.String(), err
}
