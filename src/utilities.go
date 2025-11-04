package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

func (CLIArgs) Version() string {
	return "helmizer 0.17.0"
}

// Compares two lists and removes any elements from list2 that are present in list1.
func CompareAndStrip(ignoreList, finalList []string) []string {
	var result []string

	// Create a map to store the elements of list1
	elements := make(map[string]bool)
	for _, item := range ignoreList {
		elements[item] = true
	}

	// Iterate over the elements of list2 and add them to the result if they are not present in list1
	for _, item := range finalList {
		if !elements[item] {
			result = append(result, item)
		}
	}
	return result
}

// Sets the indentation of a YAML file to 2
func FixYAMLIndentation(printableYaml []byte) []byte {
	var indentation = 2
	var data yaml.Node // Preserve ordering of keys, do not sort
	_ = yaml.Unmarshal(printableYaml, &data)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(indentation)
	_ = encoder.Encode(&data)

	return buf.Bytes()
}

// Reads a YAML file and returns a Config struct
func ReadYamlFile(args CLIArgs) Config {
	// Stat Helmizer config file
	if _, err := os.Stat(args.ConfigFilePath); err != nil {
		log.Fatalf("Helmizer config file not found: %s", args.ConfigFilePath)
		os.Exit(1)
	}

	// Read config file
	data, err := os.ReadFile(args.ConfigFilePath)
	if err != nil {
		log.Fatalf("Error reading YAML file: %s\n", err)
		os.Exit(1)
	}

	// Parse the config file's YAML content into a Config struct
	var config Config
	unmarshalErr := yaml.Unmarshal(data, &config)
	if unmarshalErr != nil {
		log.Fatalf("Error unmarshaling YAML data: %s\n", unmarshalErr)
		os.Exit(1)
	}

	// Marshal the struct into YAML format
	_, marshallErr := yaml.Marshal(&config)
	if marshallErr != nil {
		log.Fatalf("error: %v", err)
		os.Exit(1)
	}

	return config
}

// Print a YAML file with fixed indentation
func PrintYAML(printableYaml []byte) {
	fmt.Print(string(FixYAMLIndentation(printableYaml)))
}

// Calculate  absolute and relative paths based on the provided directory and file name
func ConstructPaths(d string, f string) (string, string) {
	// Construct absolute path
	absolutePath := filepath.Join(d, f)

	// Construct relative path
	relativePath, err := filepath.Rel(d, absolutePath)
	if err != nil {
		log.Errorf("Failed to construct relative path: %s", err)
		return "", ""
	}

	return absolutePath, relativePath
}
