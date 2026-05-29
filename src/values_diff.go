package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	yaml "gopkg.in/yaml.v3"
)

type yamlValuePathDiff struct {
	status     string
	path       string
	old        string
	new        string
	risk       string
	riskReason string
}

func DiffYAMLValuePaths(oldName string, oldData []byte, newName string, newData []byte) (string, error) {
	oldValues, err := flattenYAMLValues(oldData)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", oldName, err)
	}
	newValues, err := flattenYAMLValues(newData)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", newName, err)
	}

	diffs := compareYAMLValuePaths(oldValues, newValues)
	if len(diffs) == 0 {
		return "", nil
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "--- %s\n", oldName)
	fmt.Fprintf(&out, "+++ %s\n", newName)
	fmt.Fprintf(&out, "Summary: %s\n", formatYAMLValueRiskSummary(diffs))
	writer := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "RISK\tSTATUS\tPATH\tOLD\tNEW\tNOTE")
	for _, diff := range diffs {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", diff.risk, diff.status, diff.path, diff.old, diff.new, diff.riskReason)
	}
	_ = writer.Flush()
	return out.String(), nil
}

func flattenYAMLValues(data []byte) (map[string]string, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	values := map[string]string{}
	if len(node.Content) == 0 {
		return values, nil
	}
	flattenYAMLNode(values, "", node.Content[0])
	return values, nil
}

func flattenYAMLNode(values map[string]string, path string, node *yaml.Node) {
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content) == 0 {
			values[path] = "{}"
			return
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			flattenYAMLNode(values, nextPath, node.Content[i+1])
		}
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			values[path] = "[]"
			return
		}
		for i, item := range node.Content {
			flattenYAMLNode(values, fmt.Sprintf("%s[%d]", path, i), item)
		}
	default:
		values[path] = yamlValueString(node)
	}
}

func yamlValueString(node *yaml.Node) string {
	switch node.Tag {
	case "!!null":
		return "null"
	case "!!str":
		if node.Value == "" {
			return `""`
		}
	}
	return strings.ReplaceAll(node.Value, "\n", `\n`)
}

func compareYAMLValuePaths(oldValues map[string]string, newValues map[string]string) []yamlValuePathDiff {
	paths := map[string]bool{}
	for path := range oldValues {
		paths[path] = true
	}
	for path := range newValues {
		paths[path] = true
	}

	sortedPaths := make([]string, 0, len(paths))
	for path := range paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	var diffs []yamlValuePathDiff
	for _, path := range sortedPaths {
		oldValue, oldFound := oldValues[path]
		newValue, newFound := newValues[path]
		switch {
		case !oldFound:
			diffs = append(diffs, newYAMLValuePathDiff("added", path, "<missing>", newValue))
		case !newFound:
			diffs = append(diffs, newYAMLValuePathDiff("removed", path, oldValue, "<missing>"))
		case oldValue != newValue:
			diffs = append(diffs, newYAMLValuePathDiff("changed", path, oldValue, newValue))
		}
	}
	sort.SliceStable(diffs, func(i, j int) bool {
		if valueRiskRank(diffs[i].risk) != valueRiskRank(diffs[j].risk) {
			return valueRiskRank(diffs[i].risk) < valueRiskRank(diffs[j].risk)
		}
		return diffs[i].path < diffs[j].path
	})
	return diffs
}

func newYAMLValuePathDiff(status string, path string, oldValue string, newValue string) yamlValuePathDiff {
	risk, reason := classifyYAMLValuePathRisk(path, status)
	return yamlValuePathDiff{
		status:     status,
		path:       path,
		old:        oldValue,
		new:        newValue,
		risk:       risk,
		riskReason: reason,
	}
}

func formatYAMLValueRiskSummary(diffs []yamlValuePathDiff) string {
	counts := map[string]int{}
	for _, diff := range diffs {
		counts[diff.risk]++
	}
	parts := []string{
		fmt.Sprintf("%d high", counts["high"]),
		fmt.Sprintf("%d medium", counts["medium"]),
		fmt.Sprintf("%d low", counts["low"]),
	}
	return strings.Join(parts, ", ")
}

func classifyYAMLValuePathRisk(path string, status string) (string, string) {
	normalized := normalizeValueRiskPath(path)

	highRules := []struct {
		match  string
		reason string
	}{
		{"securitycontext", "security posture changed"},
		{"podsecuritycontext", "security posture changed"},
		{"containersecuritycontext", "security posture changed"},
		{"allowprivilegeescalation", "privilege setting changed"},
		{"privileged", "privilege setting changed"},
		{"capabilities", "Linux capabilities changed"},
		{"readonlyrootfilesystem", "filesystem security changed"},
		{"runas", "runtime identity changed"},
		{"seccomp", "seccomp setting changed"},
		{"sysctls", "kernel tuning changed"},
		{"rbac", "RBAC behavior changed"},
		{"serviceaccount", "service account behavior changed"},
		{"networkpolicy", "network policy changed"},
		{"ingress", "external access changed"},
		{"route", "external access changed"},
		{"gateway", "external access changed"},
		{"servicetype", "service exposure changed"},
		{"loadbalancer", "service exposure changed"},
		{"persistence", "persistent storage changed"},
		{"persistentvolume", "persistent storage changed"},
		{"storageclass", "persistent storage changed"},
		{"volumes", "volume mounts changed"},
		{"volumemounts", "volume mounts changed"},
		{"resourceslimits", "resource limits changed"},
		{"resourcesrequests", "resource requests changed"},
		{"replicas", "replica count changed"},
		{"replicacount", "replica count changed"},
		{"autoscaling", "autoscaling behavior changed"},
		{"hpa", "autoscaling behavior changed"},
		{"probe", "health check behavior changed"},
		{"affinity", "scheduling behavior changed"},
		{"nodeselector", "scheduling behavior changed"},
		{"tolerations", "scheduling behavior changed"},
		{"topologyspreadconstraints", "scheduling behavior changed"},
		{"poddisruptionbudget", "availability policy changed"},
		{"priorityclassname", "scheduling priority changed"},
		{"hostnetwork", "host networking changed"},
		{"hostpid", "host namespace sharing changed"},
		{"hostipc", "host namespace sharing changed"},
		{"dnspolicy", "DNS behavior changed"},
		{"env", "runtime environment changed"},
		{"extraenv", "runtime environment changed"},
		{"command", "container command changed"},
		{"args", "container arguments changed"},
	}
	for _, rule := range highRules {
		if strings.Contains(normalized, rule.match) {
			return "high", rule.reason
		}
	}

	lowRules := []struct {
		match  string
		reason string
	}{
		{"labels", "metadata-only change"},
		{"annotations", "metadata-only change"},
		{"fullnameoverride", "naming change"},
		{"nameoverride", "naming change"},
		{"dashboard", "dashboard/UI metadata changed"},
		{"notes", "chart notes changed"},
		{"tests", "chart test settings changed"},
	}
	for _, rule := range lowRules {
		if strings.Contains(normalized, rule.match) {
			return "low", rule.reason
		}
	}

	if status == "removed" {
		return "medium", "chart default was removed"
	}
	return "medium", "review chart default change"
}

func normalizeValueRiskPath(path string) string {
	replacer := strings.NewReplacer(".", "", "-", "", "_", "", "[", "", "]", "")
	return strings.ToLower(replacer.Replace(path))
}

func valueRiskRank(risk string) int {
	switch risk {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}
