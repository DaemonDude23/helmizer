package main

import (
	"strings"
	"testing"
)

func TestUnifiedDiffReturnsEmptyForEqualText(t *testing.T) {
	if got := UnifiedDiff("old", "same\n", "new", "same\n"); got != "" {
		t.Fatalf("UnifiedDiff() = %q, want empty", got)
	}
}

func TestUnifiedDiffUsesContextHunks(t *testing.T) {
	oldText := strings.Join([]string{
		"line01",
		"line02",
		"line03",
		"line04",
		"line05",
		"line06",
		"line07",
		"line08",
		"line09",
		"line10",
		"line11",
		"line12",
	}, "\n")
	newText := strings.Replace(oldText, "line07", "changed07", 1)

	got := UnifiedDiff("old", oldText, "new", newText)

	for _, want := range []string{
		"--- old\n",
		"+++ new\n",
		"@@ -4,7 +4,7 @@\n",
		" line04\n",
		"-line07\n",
		"+changed07\n",
		" line10\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("UnifiedDiff() missing %q in:\n%s", want, got)
		}
	}

	for _, unwanted := range []string{" line01\n", " line02\n", " line12\n"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("UnifiedDiff() included unwanted context %q in:\n%s", unwanted, got)
		}
	}
}
