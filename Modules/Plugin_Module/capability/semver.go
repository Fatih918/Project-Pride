package capability

import "strings"

// Version is a parsed Semantic Version 2.0.0 value.
type Version struct {
	Major      string
	Minor      string
	Patch      string
	Prerelease []string
	Build      []string
}

// ParseVersion parses a strict SemVer 2.0.0 value. Numeric core and
// prerelease identifiers are retained as strings to avoid integer overflow.
func ParseVersion(raw string) (Version, error) {
	if len(raw) == 0 || len(raw) > 64 || !isASCII(raw) {
		return Version{}, ErrInvalidVersion
	}

	mainAndBuild := strings.Split(raw, "+")
	if len(mainAndBuild) > 2 || mainAndBuild[0] == "" {
		return Version{}, ErrInvalidVersion
	}
	main := mainAndBuild[0]
	var build []string
	if len(mainAndBuild) == 2 {
		build = strings.Split(mainAndBuild[1], ".")
		if !validIdentifiers(build, false) {
			return Version{}, ErrInvalidVersion
		}
	}

	mainAndPre := strings.Split(main, "-")
	if len(mainAndPre) > 2 {
		// Hyphens are valid inside prerelease identifiers. Split only once.
		first := strings.IndexByte(main, '-')
		mainAndPre = []string{main[:first], main[first+1:]}
	}
	core := strings.Split(mainAndPre[0], ".")
	if len(core) != 3 || !validNumericCore(core) {
		return Version{}, ErrInvalidVersion
	}

	var prerelease []string
	if len(mainAndPre) == 2 {
		prerelease = strings.Split(mainAndPre[1], ".")
		if !validIdentifiers(prerelease, true) {
			return Version{}, ErrInvalidVersion
		}
	}

	return Version{
		Major: core[0], Minor: core[1], Patch: core[2],
		Prerelease: prerelease, Build: build,
	}, nil
}

func validNumericCore(parts []string) bool {
	for _, part := range parts {
		if part == "" || !digits(part) || len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func validIdentifiers(parts []string, rejectNumericLeadingZero bool) bool {
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		numeric := true
		for i := 0; i < len(part); i++ {
			c := part[i]
			if !isSemVerIdentifierChar(c) && c != '-' {
				return false
			}
			if c < '0' || c > '9' {
				numeric = false
			}
		}
		if rejectNumericLeadingZero && numeric && len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func digits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return len(value) > 0
}

// Compare returns -1, 0, or 1 according to SemVer precedence. Build metadata
// is intentionally ignored.
func (v Version) Compare(other Version) int {
	if result := compareNumeric(v.Major, other.Major); result != 0 {
		return result
	}
	if result := compareNumeric(v.Minor, other.Minor); result != 0 {
		return result
	}
	if result := compareNumeric(v.Patch, other.Patch); result != 0 {
		return result
	}
	if len(v.Prerelease) == 0 && len(other.Prerelease) == 0 {
		return 0
	}
	if len(v.Prerelease) == 0 {
		return 1
	}
	if len(other.Prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(v.Prerelease) && i < len(other.Prerelease); i++ {
		left, right := v.Prerelease[i], other.Prerelease[i]
		leftNumeric, rightNumeric := digits(left), digits(right)
		switch {
		case leftNumeric && rightNumeric:
			if result := compareNumeric(left, right); result != 0 {
				return result
			}
		case leftNumeric:
			return -1
		case rightNumeric:
			return 1
		default:
			if left < right {
				return -1
			}
			if left > right {
				return 1
			}
		}
	}
	switch {
	case len(v.Prerelease) < len(other.Prerelease):
		return -1
	case len(v.Prerelease) > len(other.Prerelease):
		return 1
	default:
		return 0
	}
}

func compareNumeric(left, right string) int {
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
