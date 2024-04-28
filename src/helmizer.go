package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

// Meta struct for the Helmizer config file
type Config struct {
	Helmizer  Helmizer  `yaml:"helmizer"`
	Kustomize Kustomize `yaml:"kustomize"`
}

// Contains the subset of the Helmizer config file relating to be kustomized
type Kustomize struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Namespace  string `yaml:"namespace"`

	BuildMetadata         []string               `yaml:"buildMetadata"`
	CommonAnnotations     map[string]string      `yaml:"commonAnnotations"`
	CommonLabels          map[string]string      `yaml:"commonLabels"`
	ConfigMapGenerator    []interface{}          `yaml:"configMapGenerator"`
	Crds                  []string               `yaml:"crds"`
	GeneratorOptions      map[string]interface{} `yaml:"generatorOptions"`
	Images                []interface{}          `yaml:"images"`
	NamePrefix            string                 `yaml:"namePrefix"`
	NameSuffix            string                 `yaml:"nameSuffix"`
	OpenAPI               map[string]interface{} `yaml:"openapi"`
	Patches               []interface{}          `yaml:"patches"`
	PatchesJson6902       []interface{}          `yaml:"patchesJson6902"`
	PatchesStrategicMerge []string               `yaml:"patchesStrategicMerge"`
	Replacements          []interface{}          `yaml:"replacements"`
	Replicas              []interface{}          `yaml:"replicas"`
	Resources             []string               `yaml:"resources"`
	SecretGenerator       []interface{}          `yaml:"secretGenerator"`
	SortOptions           map[string]interface{} `yaml:"sortOptions"`
	Vars                  []interface{}          `yaml:"vars"`
}

// Contains the final data structure that will be marshalled to YAML for the kustomization.yaml file
type Kustomization struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Namespace  string `yaml:"namespace,omitempty"`

	BuildMetadata         []string               `yaml:"buildMetadata,omitempty"`
	CommonAnnotations     map[string]string      `yaml:"commonAnnotations,omitempty"`
	CommonLabels          map[string]string      `yaml:"commonLabels,omitempty"`
	ConfigMapGenerator    []interface{}          `yaml:"configMapGenerator,omitempty"`
	Crds                  []string               `yaml:"crds,omitempty"`
	GeneratorOptions      map[string]interface{} `yaml:"generatorOptions,omitempty"`
	Images                []interface{}          `yaml:"images,omitempty"`
	NamePrefix            string                 `yaml:"namePrefix,omitempty"`
	NameSuffix            string                 `yaml:"nameSuffix,omitempty"`
	OpenAPI               map[string]interface{} `yaml:"openapi,omitempty"`
	Patches               []interface{}          `yaml:"patches,omitempty"`
	PatchesJson6902       []interface{}          `yaml:"patchesJson6902,omitempty"`
	PatchesStrategicMerge []string               `yaml:"patchesStrategicMerge,omitempty"`
	Replacements          []interface{}          `yaml:"replacements,omitempty"`
	Replicas              []interface{}          `yaml:"replicas,omitempty"`
	Resources             []string               `yaml:"resources,omitempty"`
	SecretGenerator       []interface{}          `yaml:"secretGenerator,omitempty"`
	SortOptions           map[string]interface{} `yaml:"sortOptions,omitempty"`
	Vars                  []interface{}          `yaml:"vars,omitempty"`
}

// Contains the subset of the Helmizer config file relating to either command sequence
type Commands struct {
	Arguments []string `yaml:"args"`
	Command   string   `yaml:"command"`
}

// Contains the subset of the Helmizer config file
type Helmizer struct {
	// CLI args only
	LogColors bool   `yaml:"logColors"`
	LogFormat string `yaml:"logFormat"`
	LogLevel  string `yaml:"logLevel"`

	// These don't exist with CLI args
	PostCommands []Commands `yaml:"postCommands"`
	PreCommands  []Commands `yaml:"preCommands"`

	// CLI args override config equivalents
	ApiVersion        string   `yaml:"apiVersion"`
	DryRun            bool     `yaml:"dryRun"`
	Ignore            []string `yaml:"ignore"`
	KustomizationPath string   `yaml:"KustomizationPath"`
	QuietCommands     bool     `yaml:"quietCommands"`
	SkipAllCommands   bool     `yaml:"skipAllCommands"`
	SkipPostCommands  bool     `yaml:"skipPostCommands"`
	SkipPreCommands   bool     `yaml:"skipPreCommands"`
	StopOnError       bool     `yaml:"stopOnError"`
}

// ExecuteCommands executes the command sequence specified in the Helmizer configuration.
func ExecuteCommands(stage string, args CLIArgs, helmizer Helmizer) ([]string, error) {
	var outputs []string

	// Determine which command sequence to execute
	commands := []Commands{}
	if stage == "pre" {
		commands = helmizer.PreCommands
	} else if stage == "post" {
		commands = helmizer.PostCommands
	}

	for _, command := range commands {
		cmd := exec.Command(command.Command, command.Arguments...)

		// Combine the command with its arguments
		commandString := strings.Join(cmd.Args, " ")

		// Set the working directory
		var err error
		cmd.Dir, err = filepath.Abs(filepath.Dir(args.ConfigFilePath))
		if err != nil {
			log.Errorf("Failed to set working directory: %s", err)
			return outputs, err
		}

		// Prepare to capture both stdout and stderr
		var stdoutStderr bytes.Buffer

		// Evaluate if command output should be streamed out
		if !helmizer.QuietCommands {
			cmd.Stdout = &stdoutStderr
			cmd.Stderr = &stdoutStderr
		}

		// Run the command
		err = cmd.Run()
		if err != nil {
			log.Errorf("Failed to execute command: %s\n%s", commandString, stdoutStderr.String())
			if helmizer.StopOnError {
				log.Fatal("stopOnError is set to true - exiting")
				os.Exit(1)
			} else {
				log.Debug("Command failed, but continuing on")
				return outputs, err
			}
		} else {
			// Log the output, trimming command
			output := strings.TrimSuffix(stdoutStderr.String(), "\n")
			log.Infof("Successfully executed command: %s\n%s", commandString, output)
			outputs = append(outputs, output)
		}
	}
	return outputs, nil
}

// Retrieves a list of files or URLs based on the provided configuration file path and configuration.
// It returns a slice of strings containing the absolute or relative paths to the files.
func GetFilesOrURLs(Type string, ConfigFilePath string, config Config) []string {
	var outputList []string // Contains list of URLs or files
	// Get absolute path to parent directory containing helmizer.yaml
	HelmizerConfigAbsDir, _ := filepath.Abs(filepath.Dir(ConfigFilePath))

	typeToIterateOver := []string{}

	if Type == "Crds" {
		typeToIterateOver = config.Kustomize.Crds
	} else if Type == "PatchesStrategicMerge" {
		typeToIterateOver = config.Kustomize.PatchesStrategicMerge
	} else if Type == "Resources" {
		typeToIterateOver = config.Kustomize.Resources
	}

	for _, KustomizeFilePath := range typeToIterateOver {
		var absPathToWalk string
		var isAbsolutePath bool

		// Check if the path is a URL
		if strings.HasPrefix(KustomizeFilePath, "http://") || strings.HasPrefix(KustomizeFilePath, "https://") {
			log.Debugf("Skipping URL: %s", KustomizeFilePath)
			outputList = append(outputList, KustomizeFilePath)
			continue // Skip this URL
		}

		// Check if the path is absolute
		if filepath.IsAbs(KustomizeFilePath) {
			absPathToWalk = KustomizeFilePath
			isAbsolutePath = true
		} else {
			// Convert relative path to absolute path based on HelmizerConfigAbsDir
			var err error
			absPathToWalk, err = filepath.Abs(filepath.Join(HelmizerConfigAbsDir, KustomizeFilePath))
			if err != nil {
				log.Errorf("Error converting to absolute path: %s", err)
				continue // Skip this path
			}
			isAbsolutePath = false
		}

		// Check if the directory exists
		if _, err := os.Stat(absPathToWalk); os.IsNotExist(err) {
			log.Debugf("Directory does not exist: %s", absPathToWalk)
			continue // Skip this directory
		}

		// Walk the directory, accumulating a list of files
		err := filepath.Walk(absPathToWalk, func(pathToFile string, info os.FileInfo, err error) error {
			if err != nil {
				log.Errorf("Error accessing path %q: %v", pathToFile, err)
				return err
			}

			if !info.IsDir() {
				log.Debugf("Found file: %s", pathToFile)
				if isAbsolutePath {
					outputList = append(outputList, pathToFile)
				} else {
					// Convert the absolute path to a relative path
					relativePath, err := filepath.Rel(HelmizerConfigAbsDir, pathToFile)
					if err != nil {
						log.Errorf("Error converting to relative path: %s", err)
						return err
					}
					outputList = append(outputList, relativePath)
				}
			}
			return nil
		})

		if err != nil {
			log.Warnf("Error walking the path %q: %v", absPathToWalk, err)
		}
	}
	outputList = CompareAndStrip(config.Helmizer.Ignore, outputList)
	return outputList
}

// RenameHelmizerKeys reads the Helmizer configuration from a YAML file and renames the keys
func RenameHelmizerKeys(filePath string) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Error(err)
		return err
	}

	// Unmarshal the YAML data into a map
	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Error(err)
		return err
	}

	// Get the Helmizer keys
	helmizer, ok := config["helmizer"].(map[string]interface{})
	if !ok {
		log.Debug("No 'helmizer' key found in the YAML file")
	}

	// Loop through the Helmizer keys and print logs based on the key names
	for key, value := range helmizer {
		switch key {
		case "dry-run":
			delete(helmizer, "dry-run")
			helmizer["DryRun"] = value
			log.Warn("Pre-flight: Helmizer config 'dry-run' was replaced with 'dryRun'. Update your configuration. This will be removed in a future release")
		case "commandSequence":
			delete(helmizer, "commandSequence")
			helmizer["preCommands"] = value
			log.Warn("Pre-flight: Helmizer config 'commandSequence' was replaced with 'preCommands'. Update your configuration. This will be removed in a future release")
		case "kustomization-file-extension", "kustomization-file-name", "sort-keys", "version":
			log.Warnf("Pre-flight: Helmizer config '%s' is deprecated and will be removed in a future release Update your configuration.", key)
		}
	}

	return nil
}

// Reconciles the Helmizer configuration with the CLI arguments and returns a joined config
func ReconcileHelmizerConfig(args CLIArgs, config Config) Config {
	_ = RenameHelmizerKeys(args.ConfigFilePath)

	if config.Helmizer.ApiVersion == "" {
		config.Helmizer.ApiVersion = args.ApiVersion
	}

	if args.DryRun {
		config.Helmizer.DryRun = true
	}

	if config.Helmizer.KustomizationPath == "" {
		config.Helmizer.KustomizationPath = args.KustomizationPath
	}

	if args.LogFormat != "" {
		config.Helmizer.LogFormat = args.LogFormat
	}

	if args.LogLevel != "" {
		config.Helmizer.LogLevel = args.LogLevel
	}

	if args.QuietCommands {
		config.Helmizer.QuietCommands = true
	}

	if !args.StopOnError {
		config.Helmizer.StopOnError = false
	}

	if args.SkipAllCommands {
		config.Helmizer.SkipAllCommands = true
	}

	if args.SkipPostCommands {
		config.Helmizer.SkipPostCommands = true
	}

	if args.SkipPostCommands {
		config.Helmizer.SkipPostCommands = true
	}

	if args.SkipPreCommands {
		config.Helmizer.SkipPreCommands = true
	}

	return config
}

func WriteKustomizationFile(args CLIArgs, helmizer Config, kustomization Kustomization, kFilePath string) {
	// Get the absolute path to the kustomization file
	kAbsPath, _ := filepath.Abs(kFilePath)

	// Read the existing file
	oldContent, err := os.ReadFile(kAbsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// If the file doesn't exist, set oldContent to an empty byte slice
			log.Debugf("File %s does not exist, proceeding with empty content", kAbsPath)
			oldContent = []byte{}
		} else {
			// If some other error occurred, log it and exit
			log.Fatal(err)
		}
	}

	// Fix indentation of the old content
	fixedOldContent := FixYAMLIndentation(oldContent)

	// Calculate MD5 hash of the fixed old content
	oldHash := md5.Sum(fixedOldContent)

	// Marshal kustomization to YAML
	kustomizationYAML, err := yaml.Marshal(&kustomization)
	if err != nil {
		log.Fatal(err)
	}

	// Fix indentation of the new content
	fixedNewContent := FixYAMLIndentation(kustomizationYAML)

	// Calculate MD5 hash of the fixed new content
	newHash := md5.Sum(fixedNewContent)

	// Compare old hash with new hash
	if oldHash != newHash {
		// Write new content to file
		err = os.WriteFile(kAbsPath, fixedNewContent, 0644)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Successfully wrote kustomization file: %s", kAbsPath)
		}
	} else {
		log.Infof("kustomization file's keys/values have not changed - not writing file: %s", kAbsPath)
	}
	log.Infof("Helmizer completed successfully")
}
