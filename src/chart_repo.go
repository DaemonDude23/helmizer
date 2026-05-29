package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type HelmRepositoryIndex struct {
	Entries map[string][]HelmChartVersion `yaml:"entries"`
}

type HelmChartVersion struct {
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

func FetchHelmRepositoryIndex(repoURL string) (HelmRepositoryIndex, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return FetchHelmRepositoryIndexWithClient(client, repoURL)
}

func FetchHelmRepositoryIndexWithClient(client *http.Client, repoURL string) (HelmRepositoryIndex, error) {
	indexURL := helmRepositoryIndexURL(repoURL)
	resp, err := client.Get(indexURL)
	if err != nil {
		return HelmRepositoryIndex{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return HelmRepositoryIndex{}, fmt.Errorf("GET %s returned %s", indexURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HelmRepositoryIndex{}, err
	}

	var index HelmRepositoryIndex
	if err := yaml.Unmarshal(body, &index); err != nil {
		return HelmRepositoryIndex{}, err
	}
	return index, nil
}

func helmRepositoryIndexURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	if strings.HasSuffix(repoURL, "/index.yaml") {
		return repoURL
	}
	return strings.TrimRight(repoURL, "/") + "/index.yaml"
}

type VersionSelection struct {
	Current                 SemVer
	Latest                  string
	LatestAppVersion        string
	LatestAllowed           string
	LatestAllowedAppVersion string
	LatestSkipped           string
	LatestSkippedAppVersion string
	CurrentAppVersion       string
}

func SelectChartVersion(versions []HelmChartVersion, currentRaw string, policy string, constraint string, includePreRelease bool) (VersionSelection, error) {
	current, err := ParseSemVer(currentRaw)
	if err != nil {
		return VersionSelection{}, err
	}

	selection := VersionSelection{Current: current}
	var latestSemVer *SemVer
	var latestAllowedSemVer *SemVer
	var latestSkippedSemVer *SemVer

	for _, chartVersion := range versions {
		candidate, err := ParseSemVer(chartVersion.Version)
		if err != nil {
			continue
		}
		if !AllowPreRelease(current, candidate, includePreRelease) {
			continue
		}

		if chartVersion.Version == currentRaw {
			selection.CurrentAppVersion = chartVersion.AppVersion
		}

		if latestSemVer == nil || SemVerGreater(candidate, *latestSemVer) {
			candidateCopy := candidate
			latestSemVer = &candidateCopy
			selection.Latest = chartVersion.Version
			selection.LatestAppVersion = chartVersion.AppVersion
		}

		if !SemVerGreater(candidate, current) {
			continue
		}
		if VersionAllowedByPolicy(current, candidate, policy, constraint) {
			if latestAllowedSemVer == nil || SemVerGreater(candidate, *latestAllowedSemVer) {
				candidateCopy := candidate
				latestAllowedSemVer = &candidateCopy
				selection.LatestAllowed = chartVersion.Version
				selection.LatestAllowedAppVersion = chartVersion.AppVersion
			}
			continue
		}
		if latestSkippedSemVer == nil || SemVerGreater(candidate, *latestSkippedSemVer) {
			candidateCopy := candidate
			latestSkippedSemVer = &candidateCopy
			selection.LatestSkipped = chartVersion.Version
			selection.LatestSkippedAppVersion = chartVersion.AppVersion
		}
	}

	return selection, nil
}
