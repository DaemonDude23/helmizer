package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/alexflint/go-arg"
)

type launchFile struct {
	Configurations []launchConfiguration `json:"configurations"`
}

type launchConfiguration struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Request string            `json:"request"`
	Program string            `json:"program"`
	Cwd     string            `json:"cwd"`
	Console string            `json:"console"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

func TestVSCodeChartsValuesDiffLaunchConfig(t *testing.T) {
	config := readLaunchFile(t)
	launch := findLaunchConfig(t, config, "charts: values diff")

	if launch.Type != "go" {
		t.Fatalf("launch type = %q, want %q", launch.Type, "go")
	}
	if launch.Request != "launch" {
		t.Fatalf("launch request = %q, want %q", launch.Request, "launch")
	}
	if launch.Program != "${workspaceFolder}/src/" {
		t.Fatalf("launch program = %q, want %q", launch.Program, "${workspaceFolder}/src/")
	}
	if launch.Cwd != "${workspaceFolder}/src/" {
		t.Fatalf("launch cwd = %q, want %q", launch.Cwd, "${workspaceFolder}/src/")
	}
	if got := launch.Env["CGO_ENABLED"]; got != "0" {
		t.Fatalf("CGO_ENABLED = %q, want %q", got, "0")
	}
	if !isChartsCommand(launch.Args) {
		t.Fatalf("isChartsCommand(%v) = false, want true", launch.Args)
	}
}

func TestVSCodeChartsValuesDiffArgsParse(t *testing.T) {
	config := readLaunchFile(t)
	launch := findLaunchConfig(t, config, "charts: values diff")

	args := parseChartsLaunchArgs(t, launch)
	if args.Charts == nil || args.Charts.Diff == nil {
		t.Fatalf("charts diff args were not parsed: %+v", args)
	}
	diffArgs := args.Charts.Diff
	if diffArgs.Kind != "values" {
		t.Fatalf("Kind = %q, want %q", diffArgs.Kind, "values")
	}
	if diffArgs.Release != "cert-manager" {
		t.Fatalf("Release = %q, want %q", diffArgs.Release, "cert-manager")
	}
	if diffArgs.ConfigFilePath != "../examples/commonLabels/helmizer.yaml" {
		t.Fatalf("ConfigFilePath = %q, want example config", diffArgs.ConfigFilePath)
	}
	if diffArgs.ValuesMode != "paths" {
		t.Fatalf("ValuesMode = %q, want default %q", diffArgs.ValuesMode, "paths")
	}
}

func TestVSCodeChartsReviewLaunchConfig(t *testing.T) {
	config := readLaunchFile(t)
	launch := findLaunchConfig(t, config, "charts: review values dry-run")

	if launch.Type != "go" {
		t.Fatalf("launch type = %q, want %q", launch.Type, "go")
	}
	if launch.Request != "launch" {
		t.Fatalf("launch request = %q, want %q", launch.Request, "launch")
	}
	if launch.Program != "${workspaceFolder}/src/" {
		t.Fatalf("launch program = %q, want %q", launch.Program, "${workspaceFolder}/src/")
	}
	if launch.Cwd != "${workspaceFolder}/src/" {
		t.Fatalf("launch cwd = %q, want %q", launch.Cwd, "${workspaceFolder}/src/")
	}
	if launch.Console != "integratedTerminal" {
		t.Fatalf("launch console = %q, want %q", launch.Console, "integratedTerminal")
	}
	if got := launch.Env["CGO_ENABLED"]; got != "0" {
		t.Fatalf("CGO_ENABLED = %q, want %q", got, "0")
	}
	if !isChartsCommand(launch.Args) {
		t.Fatalf("isChartsCommand(%v) = false, want true", launch.Args)
	}
}

func TestVSCodeChartsReviewArgsParse(t *testing.T) {
	config := readLaunchFile(t)
	launch := findLaunchConfig(t, config, "charts: review values dry-run")

	args := parseChartsLaunchArgs(t, launch)
	if args.Charts == nil || args.Charts.Review == nil {
		t.Fatalf("charts review args were not parsed: %+v", args)
	}
	reviewArgs := args.Charts.Review
	if !reviewArgs.DryRun {
		t.Fatal("DryRun = false, want true for launch safety")
	}
	if reviewArgs.ValuesFile != "helmizer.yaml" {
		t.Fatalf("ValuesFile = %q, want %q", reviewArgs.ValuesFile, "helmizer.yaml")
	}
	if reviewArgs.Release != "cert-manager" {
		t.Fatalf("Release = %q, want %q", reviewArgs.Release, "cert-manager")
	}
	if reviewArgs.ConfigFilePath != "../examples/commonLabels/helmizer.yaml" {
		t.Fatalf("ConfigFilePath = %q, want example config", reviewArgs.ConfigFilePath)
	}
}

func parseChartsLaunchArgs(t *testing.T, launch launchConfiguration) ChartsCLIArgs {
	t.Helper()
	var args ChartsCLIArgs
	parser, err := arg.NewParser(arg.Config{
		Program: "helmizer",
		Out:     io.Discard,
		Exit:    func(int) {},
	}, &args)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	if err := parser.Parse(launch.Args); err != nil {
		t.Fatalf("Parse(%v) error = %v", launch.Args, err)
	}
	return args
}

func readLaunchFile(t *testing.T) launchFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".vscode", "launch.json"))
	if err != nil {
		t.Fatalf("ReadFile(launch.json) error = %v", err)
	}

	var config launchFile
	if err := json.Unmarshal(stripJSONCTrailingCommas(data), &config); err != nil {
		t.Fatalf("launch.json is not valid JSONC: %v", err)
	}
	return config
}

func stripJSONCTrailingCommas(data []byte) []byte {
	re := regexp.MustCompile(`,\s*([}\]])`)
	return re.ReplaceAll(data, []byte("$1"))
}

func findLaunchConfig(t *testing.T, config launchFile, name string) launchConfiguration {
	t.Helper()
	for _, launch := range config.Configurations {
		if launch.Name == name {
			return launch
		}
	}
	t.Fatalf("launch config %q was not found", name)
	return launchConfiguration{}
}
