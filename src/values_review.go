package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	yaml "gopkg.in/yaml.v3"
)

type ValuesReviewCandidate struct {
	Status     string
	Path       string
	Risk       string
	RiskReason string
	OldDefault *yaml.Node
	NewDefault *yaml.Node
	Local      *yaml.Node
}

func RunChartsReview(args ChartsReviewArgs) int {
	targets, err := ResolveChartTargets(args.ConfigFilePath, args.ConfigGlob, args.HelmfilePath)
	if err != nil {
		log.Error(err)
		return 1
	}
	if len(targets) != 1 {
		log.Errorf("charts review expects exactly one target; got %d", len(targets))
		return 1
	}
	target := targets[0]

	options := chartReviewHelmfileOptions(args)
	state, _, err := LoadHelmfileState(target, options)
	if err != nil {
		log.Error(err)
		return 1
	}

	release, found := findHelmfileRelease(state.Releases, args.Release)
	if !found {
		log.Errorf("release %q was not found in %s", args.Release, target.HelmfilePath)
		return 1
	}

	resolved, reason := ResolveHelmfileReleaseChart(release, state.Repositories, target.BaseDir)
	if reason != "" {
		log.Errorf("release %q cannot be reviewed: %s", release.Name, reason)
		return 1
	}

	valuesFile, err := ResolveReleaseValuesFile(release, target.BaseDir, args.ValuesFile)
	if err != nil {
		log.Error(err)
		return 1
	}

	targetVersion := strings.TrimSpace(args.To)
	if targetVersion == "" || strings.EqualFold(targetVersion, "latest") {
		repoCache := map[string]HelmRepositoryIndex{}
		client := &http.Client{Timeout: 30 * time.Second}
		targetVersion, err = resolveLatestTargetVersion(repoCache, client, resolved, release.Version, args.Policy, args.Constraint, args.IncludePreRelease)
		if err != nil {
			log.Error(err)
			return 1
		}
	}

	currentDefaults, err := runHelmShowValues(target.BaseDir, resolved, release.Version)
	if err != nil {
		log.Error(err)
		return 1
	}
	targetDefaults, err := runHelmShowValues(target.BaseDir, resolved, targetVersion)
	if err != nil {
		log.Error(err)
		return 1
	}

	if err := ReviewValuesFile(ValuesReviewOptions{
		ReleaseName:        release.Name,
		ValuesFile:         valuesFile,
		CurrentVersion:     release.Version,
		TargetVersion:      targetVersion,
		CurrentChartValues: currentDefaults,
		TargetChartValues:  targetDefaults,
		In:                 os.Stdin,
		Out:                os.Stdout,
		DryRun:             args.DryRun,
	}); err != nil {
		log.Error(err)
		return 1
	}
	return 0
}

func ResolveReleaseValuesFile(release HelmfileRelease, baseDir string, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return resolveValuesFilePath(baseDir, override)
	}

	var paths []string
	for _, value := range release.Values {
		path, ok := value.(string)
		if !ok || strings.TrimSpace(path) == "" {
			continue
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("release %q has no values file entry; pass --values-file", release.Name)
	}
	if len(paths) > 1 {
		return "", fmt.Errorf("release %q has multiple values file entries; pass --values-file", release.Name)
	}
	return resolveValuesFilePath(baseDir, paths[0])
}

func resolveValuesFilePath(baseDir string, rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", fmt.Errorf("remote values files are not supported for review: %s", path)
	}
	if strings.Contains(path, "{{") || strings.Contains(path, "}}") {
		return "", fmt.Errorf("templated values file paths are not supported for review: %s", path)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("values file not found: %s", path)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("values path is not a file: %s", path)
	}
	return path, nil
}

type ValuesReviewOptions struct {
	ReleaseName        string
	ValuesFile         string
	CurrentVersion     string
	TargetVersion      string
	CurrentChartValues []byte
	TargetChartValues  []byte
	In                 io.Reader
	Out                io.Writer
	DryRun             bool
}

func ReviewValuesFile(options ValuesReviewOptions) error {
	in := options.In
	if in == nil {
		in = os.Stdin
	}
	out := options.Out
	if out == nil {
		out = os.Stdout
	}

	localValues, err := os.ReadFile(options.ValuesFile)
	if err != nil {
		return err
	}
	fileMode := os.FileMode(0o644)
	if info, err := os.Stat(options.ValuesFile); err == nil {
		fileMode = info.Mode().Perm()
	}
	doc, err := parseEditableYAMLDocument(localValues)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", options.ValuesFile, err)
	}
	candidates, err := BuildValuesReviewCandidates(options.CurrentChartValues, options.TargetChartValues, localValues)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Fprintf(out, "No reviewable values changes for %s from %s to %s\n", options.ReleaseName, options.CurrentVersion, options.TargetVersion)
		return nil
	}

	session := &valuesReviewSession{
		options:     options,
		doc:         doc,
		candidates:  candidates,
		states:      make([]string, len(candidates)),
		currentData: localValues,
	}

	if inFile, inOK := in.(*os.File); inOK {
		if outFile, outOK := out.(*os.File); outOK && isTerminalFile(inFile) && isTerminalFile(outFile) {
			return reviewValuesFileTUI(session, inFile, outFile, fileMode)
		}
	}

	return reviewValuesFilePrompt(session, in, out, fileMode)
}

type valuesReviewSession struct {
	options     ValuesReviewOptions
	doc         *yaml.Node
	candidates  []ValuesReviewCandidate
	states      []string
	currentData []byte
	accepted    int
	index       int
	scroll      int
}

func reviewValuesFilePrompt(session *valuesReviewSession, in io.Reader, out io.Writer, fileMode os.FileMode) error {
	reader := bufio.NewReader(in)
reviewLoop:
	for session.index < len(session.candidates) {
		candidate := session.candidates[session.index]
		renderValuesReviewCandidate(out, session.options, candidate, session.currentData, session.index+1, len(session.candidates))
		decision, err := readValuesReviewDecision(reader, out)
		if err != nil {
			return err
		}
		switch decision {
		case "quit":
			break reviewLoop
		case "accept":
			if err := session.acceptCurrent(); err != nil {
				return err
			}
		default:
			session.discardCurrent()
		}
	}
	return session.finish(out, fileMode)
}

func (session *valuesReviewSession) acceptCurrent() error {
	if session.index >= len(session.candidates) {
		return nil
	}
	candidate := session.candidates[session.index]
	if err := setYAMLNodeAtPath(documentRoot(session.doc), candidate.Path, candidate.NewDefault); err != nil {
		return err
	}
	currentData, err := marshalYAMLDocument(session.doc)
	if err != nil {
		return err
	}
	session.currentData = currentData
	session.states[session.index] = "accepted"
	session.accepted++
	session.next()
	return nil
}

func (session *valuesReviewSession) discardCurrent() {
	if session.index >= len(session.candidates) {
		return
	}
	session.states[session.index] = "skipped"
	session.next()
}

func (session *valuesReviewSession) next() {
	if session.index < len(session.candidates) {
		session.index++
	}
	session.scroll = 0
}

func (session *valuesReviewSession) previous() {
	if session.index > 0 {
		session.index--
	}
	session.scroll = 0
}

func (session *valuesReviewSession) finish(out io.Writer, fileMode os.FileMode) error {
	if session.accepted == 0 {
		fmt.Fprintln(out, "No changes accepted.")
		return nil
	}
	if session.options.DryRun {
		fmt.Fprintf(out, "Dry run: %d change(s) accepted, %s not written.\n", session.accepted, session.options.ValuesFile)
		return nil
	}
	if err := os.WriteFile(session.options.ValuesFile, session.currentData, fileMode); err != nil {
		return err
	}
	fmt.Fprintf(out, "Wrote %d accepted change(s) to %s\n", session.accepted, session.options.ValuesFile)
	return nil
}

func BuildValuesReviewCandidates(currentDefaults []byte, targetDefaults []byte, localValues []byte) ([]ValuesReviewCandidate, error) {
	currentDoc, err := parseEditableYAMLDocument(currentDefaults)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current chart values: %w", err)
	}
	targetDoc, err := parseEditableYAMLDocument(targetDefaults)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target chart values: %w", err)
	}
	localDoc, err := parseEditableYAMLDocument(localValues)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local values: %w", err)
	}

	currentFlat, err := flattenYAMLValues(currentDefaults)
	if err != nil {
		return nil, err
	}
	targetFlat, err := flattenYAMLValues(targetDefaults)
	if err != nil {
		return nil, err
	}
	diffs := compareYAMLValuePaths(currentFlat, targetFlat)

	currentRoot := documentRoot(currentDoc)
	targetRoot := documentRoot(targetDoc)
	localRoot := documentRoot(localDoc)
	seen := map[string]bool{}
	var candidates []ValuesReviewCandidate

	for _, diff := range diffs {
		if diff.status == "removed" {
			continue
		}
		path := reviewCandidatePath(diff.path, currentRoot)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true

		newDefault, ok := getYAMLNodeAtPath(targetRoot, path)
		if !ok {
			continue
		}
		oldDefault, oldFound := getYAMLNodeAtPath(currentRoot, path)
		local, localFound := getYAMLNodeAtPath(localRoot, path)
		if localFound && yamlNodesEqual(local, newDefault) {
			continue
		}

		status := "changed"
		if !oldFound {
			status = "added"
		}
		candidates = append(candidates, ValuesReviewCandidate{
			Status:     status,
			Path:       path,
			Risk:       diff.risk,
			RiskReason: diff.riskReason,
			OldDefault: oldDefault,
			NewDefault: newDefault,
			Local:      local,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if valueRiskRank(candidates[i].Risk) != valueRiskRank(candidates[j].Risk) {
			return valueRiskRank(candidates[i].Risk) < valueRiskRank(candidates[j].Risk)
		}
		return candidates[i].Path < candidates[j].Path
	})
	return candidates, nil
}

func reviewCandidatePath(path string, currentRoot *yaml.Node) string {
	mappingPath := mappingOnlyPath(path)
	if mappingPath == "" {
		return ""
	}
	parts := strings.Split(mappingPath, ".")
	for i := range parts {
		candidate := strings.Join(parts[:i+1], ".")
		if _, ok := getYAMLNodeAtPath(currentRoot, candidate); !ok {
			return candidate
		}
	}
	return mappingPath
}

func mappingOnlyPath(path string) string {
	parts := strings.Split(path, ".")
	mapped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return ""
		}
		if index := strings.Index(part, "["); index >= 0 {
			if index > 0 {
				mapped = append(mapped, part[:index])
			}
			break
		}
		mapped = append(mapped, part)
	}
	return strings.Join(mapped, ".")
}

func renderValuesReviewCandidate(out io.Writer, options ValuesReviewOptions, candidate ValuesReviewCandidate, currentData []byte, current int, total int) {
	left := "<missing>"
	if local := currentLocalNode(candidate, currentData); local != nil {
		left = renderYAMLNode(local)
	}
	right := renderYAMLNode(candidate.NewDefault)

	proposedDoc, err := parseEditableYAMLDocument(currentData)
	diff := ""
	if err == nil {
		_ = setYAMLNodeAtPath(documentRoot(proposedDoc), candidate.Path, candidate.NewDefault)
		proposedData, marshalErr := marshalYAMLDocument(proposedDoc)
		if marshalErr == nil {
			diff = UnifiedDiff(options.ValuesFile+" current", string(currentData), options.ValuesFile+" accepted "+candidate.Path, string(proposedData))
		}
	}

	fmt.Fprintf(out, "\n[%d/%d] %s %s %s\n", current, total, strings.ToUpper(candidate.Risk), strings.ToUpper(candidate.Status), candidate.Path)
	if candidate.RiskReason != "" {
		fmt.Fprintf(out, "Risk note: %s\n", candidate.RiskReason)
	}
	fmt.Fprintln(out, "Current file value                                      New chart value")
	fmt.Fprintln(out, strings.Repeat("-", 100))
	fmt.Fprint(out, formatSideBySide(left, right, 52))
	if diff != "" {
		fmt.Fprintln(out, "\nInline file diff if accepted:")
		fmt.Fprint(out, diff)
	}
}

func currentLocalNode(candidate ValuesReviewCandidate, currentData []byte) *yaml.Node {
	doc, err := parseEditableYAMLDocument(currentData)
	if err != nil {
		return candidate.Local
	}
	if local, ok := getYAMLNodeAtPath(documentRoot(doc), candidate.Path); ok {
		return local
	}
	return nil
}

func formatSideBySide(left string, right string, width int) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(right, "\n"), "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	var out strings.Builder
	for i := 0; i < maxLines; i++ {
		leftLine := ""
		if i < len(leftLines) {
			leftLine = truncateLine(leftLines[i], width)
		}
		rightLine := ""
		if i < len(rightLines) {
			rightLine = truncateLine(rightLines[i], width)
		}
		out.WriteString(fmt.Sprintf("%-*s | %s\n", width, leftLine, rightLine))
	}
	return out.String()
}

func truncateLine(line string, width int) string {
	if len(line) <= width {
		return line
	}
	if width <= 1 {
		return line[:width]
	}
	if width <= 3 {
		return line[:width]
	}
	return line[:width-3] + "..."
}

func readValuesReviewDecision(reader *bufio.Reader, out io.Writer) (string, error) {
	for {
		fmt.Fprint(out, "\n[a]ccept, [d]iscard, [q]uit: ")
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		choice := strings.ToLower(strings.TrimSpace(line))
		switch choice {
		case "a", "accept", "y", "yes":
			return "accept", nil
		case "", "d", "discard", "s", "skip", "n", "no":
			return "discard", nil
		case "q", "quit":
			return "quit", nil
		default:
			fmt.Fprintln(out, "Enter a, d, or q.")
		}
		if err == io.EOF {
			return "discard", nil
		}
	}
}

func reviewValuesFileTUI(session *valuesReviewSession, in *os.File, out *os.File, fileMode os.FileMode) error {
	state, err := makeTerminalRaw(in)
	if err != nil {
		return err
	}
	defer restoreTerminal(in, state)

	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b[2J")
	restoredScreen := false
	defer func() {
		if !restoredScreen {
			fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
		}
	}()

	for {
		rows, cols := terminalSize(out)
		fmt.Fprint(out, renderValuesReviewScreen(session, rows, cols))
		key, err := readTUIKey(in)
		if err != nil {
			return err
		}
		switch key {
		case "accept":
			if err := session.acceptCurrent(); err != nil {
				return err
			}
		case "discard":
			session.discardCurrent()
		case "next":
			session.next()
		case "previous":
			session.previous()
		case "scroll-down":
			session.scroll++
		case "scroll-up":
			if session.scroll > 0 {
				session.scroll--
			}
		case "quit":
			fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
			restoredScreen = true
			restoreTerminal(in, state)
			return session.finish(out, fileMode)
		}
		if session.index >= len(session.candidates) {
			fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
			restoredScreen = true
			restoreTerminal(in, state)
			return session.finish(out, fileMode)
		}
	}
}

func renderValuesReviewScreen(session *valuesReviewSession, rows int, cols int) string {
	if rows < 10 {
		rows = 10
	}
	if cols < 80 {
		cols = 80
	}
	listWidth := cols / 3
	if listWidth < 26 {
		listWidth = 26
	}
	if listWidth > 46 {
		listWidth = 46
	}
	bodyRows := rows - 2
	rightWidth := cols - listWidth - 3
	if rightWidth < 30 {
		rightWidth = 30
	}

	header := fmt.Sprintf(" Helmizer Values Review  %s  %s -> %s  accepted:%d/%d ",
		session.options.ReleaseName,
		session.options.CurrentVersion,
		session.options.TargetVersion,
		session.accepted,
		len(session.candidates),
	)
	footer := " a accept  d discard  j/down next  k/up previous  l/right scroll down  h/left scroll up  q quit "

	rightLines := session.renderRightPane(rightWidth)
	if session.scroll > len(rightLines)-bodyRows {
		session.scroll = maxInt(0, len(rightLines)-bodyRows)
	}
	if session.scroll < 0 {
		session.scroll = 0
	}

	var out strings.Builder
	out.WriteString("\x1b[H")
	out.WriteString(inverseLine(header, cols))
	out.WriteString("\n")
	listStart := reviewListStart(session.index, len(session.candidates), bodyRows)
	for row := 0; row < bodyRows; row++ {
		left := session.renderListLine(listStart+row, listWidth)
		right := ""
		rightIndex := session.scroll + row
		if rightIndex >= 0 && rightIndex < len(rightLines) {
			right = truncateLine(rightLines[rightIndex], rightWidth)
		}
		out.WriteString(padRight(left, listWidth))
		out.WriteString(" | ")
		out.WriteString(padRight(right, rightWidth))
		out.WriteString("\x1b[K\n")
	}
	out.WriteString(inverseLine(footer, cols))
	return out.String()
}

func (session *valuesReviewSession) renderRightPane(width int) []string {
	if len(session.candidates) == 0 || session.index >= len(session.candidates) {
		return []string{"No reviewable changes."}
	}
	candidate := session.candidates[session.index]
	left := "<missing>"
	if local := currentLocalNode(candidate, session.currentData); local != nil {
		left = renderYAMLNode(local)
	}
	right := renderYAMLNode(candidate.NewDefault)

	var proposedDiff string
	proposedDoc, err := parseEditableYAMLDocument(session.currentData)
	if err == nil {
		_ = setYAMLNodeAtPath(documentRoot(proposedDoc), candidate.Path, candidate.NewDefault)
		proposedData, marshalErr := marshalYAMLDocument(proposedDoc)
		if marshalErr == nil {
			proposedDiff = UnifiedDiff(session.options.ValuesFile+" current", string(session.currentData), session.options.ValuesFile+" accepted "+candidate.Path, string(proposedData))
		}
	}

	sideWidth := maxInt(16, (width-3)/2)
	lines := []string{
		fmt.Sprintf("[%d/%d] %s %s %s", session.index+1, len(session.candidates), strings.ToUpper(candidate.Risk), strings.ToUpper(candidate.Status), candidate.Path),
		fmt.Sprintf("Risk note: %s", candidate.RiskReason),
		"",
		fmt.Sprintf("%-*s | %s", sideWidth, "Current values file", "New chart default"),
		strings.Repeat("-", width),
	}
	lines = append(lines, strings.Split(strings.TrimRight(formatSideBySide(left, right, sideWidth), "\n"), "\n")...)
	lines = append(lines, "", "Inline file diff if accepted:", "")
	if proposedDiff != "" {
		lines = append(lines, strings.Split(strings.TrimRight(proposedDiff, "\n"), "\n")...)
	}
	return lines
}

func (session *valuesReviewSession) renderListLine(index int, width int) string {
	if index < 0 || index >= len(session.candidates) {
		return ""
	}
	candidate := session.candidates[index]
	pointer := " "
	if index == session.index {
		pointer = ">"
	}
	state := " "
	switch session.states[index] {
	case "accepted":
		state = "A"
	case "skipped":
		state = "D"
	}
	status := "C"
	if candidate.Status == "added" {
		status = "+"
	}
	line := fmt.Sprintf("%s %s %s %s %s", pointer, state, riskInitial(candidate.Risk), status, candidate.Path)
	return truncateLine(line, width)
}

func riskInitial(risk string) string {
	switch risk {
	case "high":
		return "H"
	case "medium":
		return "M"
	case "low":
		return "L"
	default:
		return "?"
	}
}

func reviewListStart(index int, total int, height int) int {
	if total <= height {
		return 0
	}
	start := index - height/2
	if start < 0 {
		return 0
	}
	if start+height > total {
		return total - height
	}
	return start
}

func inverseLine(line string, width int) string {
	return "\x1b[7m" + padRight(truncateLine(line, width), width) + "\x1b[0m"
}

func padRight(line string, width int) string {
	if len(line) >= width {
		return truncateLine(line, width)
	}
	return line + strings.Repeat(" ", width-len(line))
}

func readTUIKey(in *os.File) (string, error) {
	for {
		var b [1]byte
		n, err := in.Read(b[:])
		if err == io.EOF {
			return "quit", nil
		}
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		switch b[0] {
		case 'a', 'A', 'y', 'Y':
			return "accept", nil
		case 'd', 'D', 's', 'S', 'n', 'N', '\r', '\n':
			return "discard", nil
		case 'q', 'Q':
			return "quit", nil
		case 'j', 'J':
			return "next", nil
		case 'k', 'K':
			return "previous", nil
		case 'l', 'L', ' ':
			return "scroll-down", nil
		case 'h', 'H':
			return "scroll-up", nil
		case 0x1b:
			var seq [2]byte
			n, _ := in.Read(seq[:])
			if n == 2 && seq[0] == '[' {
				switch seq[1] {
				case 'A':
					return "previous", nil
				case 'B':
					return "next", nil
				case 'C':
					return "scroll-down", nil
				case 'D':
					return "scroll-up", nil
				}
			}
		}
	}
}

func isTerminalFile(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	return err == nil
}

func makeTerminalRaw(file *os.File) (*unix.Termios, error) {
	fd := int(file.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	raw := *state
	raw.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Cflag |= unix.CS8
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return nil, err
	}
	return state, nil
}

func restoreTerminal(file *os.File, state *unix.Termios) {
	if state == nil {
		return
	}
	_ = unix.IoctlSetTermios(int(file.Fd()), unix.TCSETS, state)
}

func terminalSize(file *os.File) (int, int) {
	size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	if err != nil || size.Row == 0 || size.Col == 0 {
		return 30, 120
	}
	return int(size.Row), int(size.Col)
}

func parseEditableYAMLDocument(data []byte) (*yaml.Node, error) {
	if strings.TrimSpace(string(data)) == "" {
		return &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
			}},
		}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a YAML mapping at document root")
	}
	return &doc, nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func getYAMLNodeAtPath(root *yaml.Node, path string) (*yaml.Node, bool) {
	node := root
	for _, part := range strings.Split(path, ".") {
		if node == nil || node.Kind != yaml.MappingNode {
			return nil, false
		}
		node = mappingValue(*node, part)
		if node == nil {
			return nil, false
		}
	}
	return node, true
}

func setYAMLNodeAtPath(root *yaml.Node, path string, value *yaml.Node) error {
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping root")
	}
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("empty values path")
	}

	node := root
	for _, part := range parts[:len(parts)-1] {
		next := mappingValue(*node, part)
		if next == nil {
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part},
				next,
			)
		}
		if next.Kind != yaml.MappingNode {
			return fmt.Errorf("cannot set %s below non-map value at %s", path, part)
		}
		node = next
	}

	key := parts[len(parts)-1]
	cloned := cloneYAMLNode(value)
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = cloned
			return nil
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		cloned,
	)
	return nil
}

func cloneYAMLNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		clone.Content[i] = cloneYAMLNode(child)
	}
	return &clone
}

func yamlNodesEqual(a *yaml.Node, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	aRendered := renderYAMLNode(a)
	bRendered := renderYAMLNode(b)
	return aRendered == bRendered
}

func renderYAMLNode(node *yaml.Node) string {
	if node == nil {
		return "<missing>"
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	_ = encoder.Encode(node)
	_ = encoder.Close()
	return strings.TrimSpace(out.String())
}

func marshalYAMLDocument(doc *yaml.Node) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
