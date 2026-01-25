package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Parse CLI arguments
	var args CLIArgs
	arg.MustParse(&args)
	SetupLogging(args)

	configPaths, err := ResolveConfigPaths(args)
	if err != nil {
		log.Fatal(err)
	}

	for _, configPath := range configPaths {
		runArgs := args
		runArgs.ConfigFilePath = configPath
		if len(configPaths) > 1 {
			log.Infof("Processing config: %s", configPath)
		}
		RunHelmizer(runArgs)
	}
	os.Exit(0)
}

func RunHelmizer(args CLIArgs) {
	// Read config file and reconcile it with arguments
	// Arguments override the config file
	reconciledHelmizerConfig := ReconcileHelmizerConfig(args, ReadYamlFile(args))

	// Run commands before constructing the kustomization file
	if !reconciledHelmizerConfig.Helmizer.SkipAllCommands && !reconciledHelmizerConfig.Helmizer.SkipPreCommands {
		ExecuteCommands("pre", args, reconciledHelmizerConfig.Helmizer)
	} else {
		log.Info("Skipping pre-command sequence")
	}

	// Construct the kustomization.yaml file object
	k := Kustomization{}
	k.ApiVersion = reconciledHelmizerConfig.Helmizer.ApiVersion
	k.Kind = "Kustomization"
	k.Namespace = reconciledHelmizerConfig.Kustomize.Namespace

	k.BuildMetadata = reconciledHelmizerConfig.Kustomize.BuildMetadata
	k.CommonAnnotations = reconciledHelmizerConfig.Kustomize.CommonAnnotations
	k.CommonLabels = reconciledHelmizerConfig.Kustomize.CommonLabels
	k.ConfigMapGenerator = reconciledHelmizerConfig.Kustomize.ConfigMapGenerator
	k.Crds = GetFilesOrURLs("Crds", args.ConfigFilePath, reconciledHelmizerConfig)
	k.GeneratorOptions = reconciledHelmizerConfig.Kustomize.GeneratorOptions
	k.HelmCharts = reconciledHelmizerConfig.Kustomize.HelmCharts
	k.Images = reconciledHelmizerConfig.Kustomize.Images
	k.Labels = reconciledHelmizerConfig.Kustomize.Labels
	k.NamePrefix = reconciledHelmizerConfig.Kustomize.NamePrefix
	k.NameSuffix = reconciledHelmizerConfig.Kustomize.NameSuffix
	k.OpenAPI = reconciledHelmizerConfig.Kustomize.OpenAPI
	k.Patches = reconciledHelmizerConfig.Kustomize.Patches
	k.PatchesJson6902 = reconciledHelmizerConfig.Kustomize.PatchesJson6902
	k.PatchesStrategicMerge = GetFilesOrURLs("PatchesStrategicMerge", args.ConfigFilePath, reconciledHelmizerConfig)
	k.Replacements = reconciledHelmizerConfig.Kustomize.Replacements
	k.Replicas = reconciledHelmizerConfig.Kustomize.Replicas
	k.SecretGenerator = reconciledHelmizerConfig.Kustomize.SecretGenerator
	k.SortOptions = reconciledHelmizerConfig.Kustomize.SortOptions
	k.Vars = reconciledHelmizerConfig.Kustomize.Vars
	k.Resources = GetFilesOrURLs("Resources", args.ConfigFilePath, reconciledHelmizerConfig)

	// Construct file paths relative to the helmizer config
	kFilePath := fmt.Sprintf("%s/%s", reconciledHelmizerConfig.Helmizer.KustomizationPath, "kustomization.yaml")
	kFileAbsolute, _ := ConstructPaths(filepath.Dir(args.ConfigFilePath), kFilePath)

	// Write kustomization file
	if !reconciledHelmizerConfig.Helmizer.DryRun {
		WriteKustomizationFile(args, reconciledHelmizerConfig, k, kFileAbsolute)
	}

	// Run commands after constructing the kustomization file
	if !reconciledHelmizerConfig.Helmizer.SkipAllCommands && !reconciledHelmizerConfig.Helmizer.SkipPostCommands {
		ExecuteCommands("post", args, reconciledHelmizerConfig.Helmizer)
	} else {
		log.Info("Skipping post-command sequence")
	}
}
