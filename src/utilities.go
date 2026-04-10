package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

func (CLIArgs) Version() string {
	return "helmizer 0.19.1"
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
		log.Fatalf("error: %v", marshallErr)
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

func ResolveConfigPaths(args CLIArgs) ([]string, error) {
	var patterns []string

	hasGlob := strings.TrimSpace(args.ConfigGlob) != ""
	configPath := strings.TrimSpace(args.ConfigFilePath)

	if configPath != "" {
		if hasGlob {
			// When --config-glob is set, the positional arg is optional;
			// only include it if the file actually exists
			if _, err := os.Stat(configPath); err == nil {
				patterns = append(patterns, configPath)
			}
		} else {
			patterns = append(patterns, configPath)
		}
	}
	patterns = append(patterns, splitConfigPatterns(args.ConfigGlob)...)

	if len(patterns) == 0 {
		return nil, fmt.Errorf("no Helmizer config file specified; provide a path or --config-glob")
	}

	var matches []string
	for _, pattern := range patterns {
		expanded, err := expandConfigPattern(pattern)
		if err != nil {
			return nil, err
		}
		if len(expanded) == 0 {
			log.Warnf("No Helmizer config files matched: %s", pattern)
			continue
		}
		matches = append(matches, expanded...)
	}

	unique := make(map[string]struct{})
	var results []string
	for _, match := range matches {
		if strings.TrimSpace(match) == "" {
			continue
		}
		abs, err := filepath.Abs(match)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config path %q: %w", match, err)
		}
		if _, exists := unique[abs]; exists {
			continue
		}
		unique[abs] = struct{}{}
		results = append(results, abs)
	}

	sort.Strings(results)
	if len(results) == 0 {
		return nil, fmt.Errorf("no Helmizer config files matched")
	}
	return results, nil
}

func splitConfigPatterns(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	var patterns []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			patterns = append(patterns, trimmed)
		}
	}
	return patterns
}

func expandConfigPattern(pattern string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	if !hasGlobMeta(pattern) {
		return []string{pattern}, nil
	}
	if strings.Contains(pattern, "**") {
		return matchWithDoublestar(pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return filterFileMatches(matches), nil
}

func hasGlobMeta(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func filterFileMatches(matches []string) []string {
	var filtered []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func matchWithDoublestar(pattern string) ([]string, error) {
	base := globBase(pattern)
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var matches []string
	err := filepath.WalkDir(base, func(pathToFile string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if matchGlob(pattern, pathToFile) {
			matches = append(matches, pathToFile)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filterFileMatches(matches), nil
}

func globBase(pattern string) string {
	index := strings.IndexAny(pattern, "*?[")
	if index == -1 {
		return filepath.Dir(pattern)
	}
	prefix := pattern[:index]
	if prefix == "" {
		return "."
	}
	base := filepath.Dir(prefix)
	if base == "." && filepath.IsAbs(pattern) {
		return string(os.PathSeparator)
	}
	return base
}

func matchGlob(pattern string, target string) bool {
	pattern = filepath.ToSlash(filepath.Clean(pattern))
	target = filepath.ToSlash(filepath.Clean(target))

	if filepath.IsAbs(pattern) {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return false
		}
		target = filepath.ToSlash(absTarget)
	}

	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	}

	patternParts := strings.Split(pattern, "/")
	targetParts := strings.Split(target, "/")
	return matchSegments(patternParts, targetParts)
}

func matchSegments(patternParts []string, targetParts []string) bool {
	if len(patternParts) == 0 {
		return len(targetParts) == 0
	}

	if patternParts[0] == "**" {
		if matchSegments(patternParts[1:], targetParts) {
			return true
		}
		for i := 0; i < len(targetParts); i++ {
			if matchSegments(patternParts[1:], targetParts[i+1:]) {
				return true
			}
		}
		return false
	}

	if len(targetParts) == 0 {
		return false
	}

	matched, err := path.Match(patternParts[0], targetParts[0])
	if err != nil || !matched {
		return false
	}
	return matchSegments(patternParts[1:], targetParts[1:])
}
