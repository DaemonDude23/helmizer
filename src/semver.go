package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Original   string
}

func ParseSemVer(raw string) (SemVer, error) {
	original := strings.TrimSpace(raw)
	version := strings.TrimPrefix(original, "v")
	if version == "" {
		return SemVer{}, fmt.Errorf("empty semantic version")
	}

	if plus := strings.Index(version, "+"); plus >= 0 {
		version = version[:plus]
	}

	preRelease := ""
	if dash := strings.Index(version, "-"); dash >= 0 {
		preRelease = version[dash+1:]
		version = version[:dash]
	}

	parts := strings.Split(version, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("invalid semantic version %q", raw)
	}

	nums := []int{0, 0, 0}
	for i, part := range parts {
		if part == "" {
			return SemVer{}, fmt.Errorf("invalid semantic version %q", raw)
		}
		for _, r := range part {
			if !unicode.IsDigit(r) {
				return SemVer{}, fmt.Errorf("invalid semantic version %q", raw)
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
		}
		nums[i] = n
	}

	return SemVer{
		Major:      nums[0],
		Minor:      nums[1],
		Patch:      nums[2],
		PreRelease: preRelease,
		Original:   original,
	}, nil
}

func (v SemVer) Compare(other SemVer) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}
	return comparePreRelease(v.PreRelease, other.PreRelease)
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func comparePreRelease(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] == bParts[i] {
			continue
		}
		aNum, aErr := strconv.Atoi(aParts[i])
		bNum, bErr := strconv.Atoi(bParts[i])
		if aErr == nil && bErr == nil {
			return compareInt(aNum, bNum)
		}
		if aErr == nil {
			return -1
		}
		if bErr == nil {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
		return 1
	}
	return compareInt(len(aParts), len(bParts))
}

func SemVerGreater(a, b SemVer) bool {
	return a.Compare(b) > 0
}

func AllowPreRelease(current, candidate SemVer, includePreRelease bool) bool {
	return includePreRelease || current.PreRelease != "" || candidate.PreRelease == ""
}

func VersionAllowedByPolicy(current, candidate SemVer, policy string, constraint string) bool {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", "same-major":
		return candidate.Major == current.Major
	case "same-minor":
		return candidate.Major == current.Major && candidate.Minor == current.Minor
	case "all":
		return true
	case "constraint":
		return SatisfiesConstraint(candidate, constraint)
	default:
		return false
	}
}

func SatisfiesConstraint(candidate SemVer, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}

	if strings.HasSuffix(raw, ".x") || strings.HasSuffix(raw, ".*") {
		prefix := raw[:len(raw)-2]
		parts := strings.Split(prefix, ".")
		if len(parts) == 1 {
			major, err := strconv.Atoi(strings.TrimPrefix(parts[0], "v"))
			return err == nil && candidate.Major == major
		}
		if len(parts) == 2 {
			major, errMajor := strconv.Atoi(strings.TrimPrefix(parts[0], "v"))
			minor, errMinor := strconv.Atoi(parts[1])
			return errMajor == nil && errMinor == nil && candidate.Major == major && candidate.Minor == minor
		}
		return false
	}

	for _, part := range splitConstraintParts(raw) {
		if part == "" {
			continue
		}
		if !satisfiesConstraintPart(candidate, part) {
			return false
		}
	}
	return true
}

func splitConstraintParts(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	var parts []string
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func satisfiesConstraintPart(candidate SemVer, part string) bool {
	operator := "="
	version := part
	for _, prefix := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(part, prefix) {
			operator = prefix
			version = strings.TrimSpace(strings.TrimPrefix(part, prefix))
			break
		}
	}

	target, err := ParseSemVer(version)
	if err != nil {
		return false
	}
	comparison := candidate.Compare(target)
	switch operator {
	case ">":
		return comparison > 0
	case ">=":
		return comparison >= 0
	case "<":
		return comparison < 0
	case "<=":
		return comparison <= 0
	default:
		return comparison == 0
	}
}
