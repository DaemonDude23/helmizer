package main

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Contains CLI arguments
type CLIArgs struct {
	// CLI args only
	LogFormat string `arg:"--log-format" default:"plain" help:"Set log format: plain or JSON"`
	LogLevel  string `arg:"-l, --log-level" default:"INFO" help:"Set log level: INFO, DEBUG, ERROR, WARNING"`
	LogColors bool   `arg:"--log-colors" default:"true" help:"Enables color in the log output"`

	// CLI args override config equivalents
	ApiVersion        string `arg:"--api-version" default:"kustomize.config.k8s.io/v1beta1" help:"Set the API version for the kustomization.yaml file"`
	DryRun            bool   `arg:"--dry-run" default:"false" help:"Don't write the kustomization.yaml file. This does not affect pre/post commands"`
	KustomizationPath string `arg:"--kustomization-path" default:"." help:"Set the path to write kustomization.yaml file"`
	QuietCommands     bool   `arg:"-q, --quiet-commands" default:"false" help:"Don't output stdout/stderr for pre and post command sequences"`
	QuietHelmizer     bool   `arg:"--quiet-helmizer" default:"false" help:"Don't output logs or the kustomization"`
	SkipAllCommands   bool   `arg:"--skip-commands" default:"false" help:"Skip executing pre and post command sequences"`
	SkipPostCommands  bool   `arg:"--skip-postcommands" default:"false" help:"Skip executing the post-command sequence"`
	SkipPreCommands   bool   `arg:"--skip-precommands" default:"false" help:"Skip executing the pre-command sequence"`
	StopOnError       bool   `arg:"--stop-on-error" default:"true" help:"Stop execution on first error"`

	ConfigFilePath string `arg:"positional, required" help:"Path to Helmizer config file"`
}

// Checks the CLI arguments and sets up logging
func SetupLogging(args CLIArgs) bool {
	// Setup logging defaults
	log.SetLevel(log.InfoLevel)

	// Quiet mode
	if !args.QuietHelmizer {
		log.SetOutput(os.Stdout)
	}

	// Set Log level from args
	if strings.EqualFold(args.LogLevel, "TRACE") {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.Debugf("Log level set to TRACE")
	} else if strings.EqualFold(args.LogLevel, "DEBUG") {
		log.SetLevel(log.DebugLevel)
		log.Debugf("Log level set to DEBUG")
	} else if strings.EqualFold(args.LogLevel, "ERROR") {
		log.SetLevel(log.ErrorLevel)
		log.Debugf("Log level set to ERROR")
	} else if strings.EqualFold(args.LogLevel, "WARNING") {
		log.SetLevel(log.WarnLevel)
		log.Debugf("Log level set to WARNING")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Log format
	if strings.EqualFold(args.LogFormat, "JSON") {
		log.SetFormatter(&log.JSONFormatter{})
		log.Debugf("Log format set to JSON")
	} else if !strings.EqualFold(args.LogLevel, "TRACE") && !strings.EqualFold(args.LogLevel, "DEBUG") {
		log.SetFormatter(&log.TextFormatter{ForceColors: args.LogColors, FullTimestamp: false})
		log.Debugf("Log format set to stylized TextFormatter (default)")
	}

	return true
}
