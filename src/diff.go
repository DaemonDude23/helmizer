package main

import (
	"fmt"
	"strings"
)

const defaultDiffContext = 3

type diffEdit struct {
	kind string
	line string
}

type diffRecord struct {
	diffEdit
	oldLine int
	newLine int
}

type diffHunk struct {
	start int
	end   int
}

func UnifiedDiff(oldName string, oldText string, newName string, newText string) string {
	if oldText == newText {
		return ""
	}

	oldLines := splitDiffLines(oldText)
	newLines := splitDiffLines(newText)
	edits := diffLines(oldLines, newLines)
	records := diffRecords(edits)
	hunks := diffHunks(records, defaultDiffContext)

	var out strings.Builder
	out.WriteString(fmt.Sprintf("--- %s\n", oldName))
	out.WriteString(fmt.Sprintf("+++ %s\n", newName))
	for _, hunk := range hunks {
		oldStart, oldCount, newStart, newCount := diffHunkRange(records[hunk.start:hunk.end])
		out.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))
		for _, record := range records[hunk.start:hunk.end] {
			out.WriteString(record.kind)
			out.WriteString(record.line)
			out.WriteString("\n")
		}
	}
	return out.String()
}

func diffRecords(edits []diffEdit) []diffRecord {
	oldLine := 1
	newLine := 1
	records := make([]diffRecord, 0, len(edits))
	for _, edit := range edits {
		record := diffRecord{
			diffEdit: edit,
			oldLine:  oldLine,
			newLine:  newLine,
		}
		records = append(records, record)
		switch edit.kind {
		case " ":
			oldLine++
			newLine++
		case "-":
			oldLine++
		case "+":
			newLine++
		}
	}
	return records
}

func diffHunks(records []diffRecord, context int) []diffHunk {
	var hunks []diffHunk
	for i, record := range records {
		if record.kind == " " {
			continue
		}

		start := maxInt(0, i-context)
		end := minInt(len(records), i+context+1)
		if len(hunks) > 0 && start <= hunks[len(hunks)-1].end {
			if end > hunks[len(hunks)-1].end {
				hunks[len(hunks)-1].end = end
			}
			continue
		}
		hunks = append(hunks, diffHunk{start: start, end: end})
	}
	return hunks
}

func diffHunkRange(records []diffRecord) (int, int, int, int) {
	oldStart := 0
	newStart := 0
	oldCount := 0
	newCount := 0

	for _, record := range records {
		if record.kind != "+" {
			oldCount++
			if oldStart == 0 {
				oldStart = record.oldLine
			}
		}
		if record.kind != "-" {
			newCount++
			if newStart == 0 {
				newStart = record.newLine
			}
		}
	}

	if oldStart == 0 && len(records) > 0 {
		oldStart = maxInt(0, records[0].oldLine-1)
	}
	if newStart == 0 && len(records) > 0 {
		newStart = maxInt(0, records[0].newLine-1)
	}

	return oldStart, oldCount, newStart, newCount
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func splitDiffLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func diffLines(oldLines, newLines []string) []diffEdit {
	if len(oldLines)*len(newLines) > 2_000_000 {
		return positionalDiff(oldLines, newLines)
	}
	return lcsDiff(oldLines, newLines)
}

func positionalDiff(oldLines, newLines []string) []diffEdit {
	var edits []diffEdit
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	for i := 0; i < maxLen; i++ {
		switch {
		case i >= len(oldLines):
			edits = append(edits, diffEdit{kind: "+", line: newLines[i]})
		case i >= len(newLines):
			edits = append(edits, diffEdit{kind: "-", line: oldLines[i]})
		case oldLines[i] == newLines[i]:
			edits = append(edits, diffEdit{kind: " ", line: oldLines[i]})
		default:
			edits = append(edits, diffEdit{kind: "-", line: oldLines[i]})
			edits = append(edits, diffEdit{kind: "+", line: newLines[i]})
		}
	}
	return edits
}

func lcsDiff(oldLines, newLines []string) []diffEdit {
	dp := make([][]int, len(oldLines)+1)
	for i := range dp {
		dp[i] = make([]int, len(newLines)+1)
	}

	for i := len(oldLines) - 1; i >= 0; i-- {
		for j := len(newLines) - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var edits []diffEdit
	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			edits = append(edits, diffEdit{kind: " ", line: oldLines[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			edits = append(edits, diffEdit{kind: "-", line: oldLines[i]})
			i++
		} else {
			edits = append(edits, diffEdit{kind: "+", line: newLines[j]})
			j++
		}
	}
	for i < len(oldLines) {
		edits = append(edits, diffEdit{kind: "-", line: oldLines[i]})
		i++
	}
	for j < len(newLines) {
		edits = append(edits, diffEdit{kind: "+", line: newLines[j]})
		j++
	}
	return edits
}
