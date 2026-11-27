package capability

import (
	"strings"
)

// ComparatorOperator is one of the comparison operators allowed by POCS-0001.
type ComparatorOperator uint8

const (
	OperatorEqual ComparatorOperator = iota + 1
	OperatorGreater
	OperatorGreaterOrEqual
	OperatorLess
	OperatorLessOrEqual
)

// VersionComparator is one clause in a version requirement.
type VersionComparator struct {
	Operator ComparatorOperator
	Version  Version
}

// VersionRequirement is either one exact version or a conjunction of
// one to four comparators.
type VersionRequirement struct {
	Comparators []VersionComparator
}

// ParseVersionRequirement parses the deliberately narrow POCS-0001 grammar.
// It does not normalize whitespace, operators, or version strings.
func ParseVersionRequirement(raw string) (VersionRequirement, error) {
	if len(raw) == 0 || len(raw) > 128 || !isASCII(raw) {
		return VersionRequirement{}, ErrInvalidVersionRequirement
	}

	// A bare SemVer or explicit equality is an exact requirement.
	if !strings.Contains(raw, " ") {
		if version, err := ParseVersion(raw); err == nil {
			return VersionRequirement{Comparators: []VersionComparator{{Operator: OperatorEqual, Version: version}}}, nil
		}
		operator, versionText, ok := parseComparatorToken(raw)
		if !ok {
			return VersionRequirement{}, ErrInvalidVersionRequirement
		}
		version, err := ParseVersion(versionText)
		if err != nil {
			return VersionRequirement{}, ErrInvalidVersionRequirement
		}
		return VersionRequirement{Comparators: []VersionComparator{{Operator: operator, Version: version}}}, nil
	}

	if raw[0] == ' ' || raw[len(raw)-1] == ' ' || strings.ContainsAny(raw, "\t\r\n") {
		return VersionRequirement{}, ErrInvalidVersionRequirement
	}
	tokens := strings.Fields(raw)
	if len(tokens) == 0 || len(tokens) > 4 {
		return VersionRequirement{}, ErrInvalidVersionRequirement
	}

	comparators := make([]VersionComparator, 0, len(tokens))
	for _, token := range tokens {
		operator, versionText, ok := parseComparatorToken(token)
		if !ok {
			return VersionRequirement{}, ErrInvalidVersionRequirement
		}
		version, err := ParseVersion(versionText)
		if err != nil {
			return VersionRequirement{}, ErrInvalidVersionRequirement
		}
		comparators = append(comparators, VersionComparator{Operator: operator, Version: version})
	}
	if contradictoryComparators(comparators) {
		return VersionRequirement{}, ErrInvalidVersionRequirement
	}
	return VersionRequirement{Comparators: comparators}, nil
}

func parseComparatorToken(token string) (ComparatorOperator, string, bool) {
	switch {
	case strings.HasPrefix(token, ">="):
		return OperatorGreaterOrEqual, token[2:], len(token) > 2
	case strings.HasPrefix(token, "<="):
		return OperatorLessOrEqual, token[2:], len(token) > 2
	case strings.HasPrefix(token, ">"):
		return OperatorGreater, token[1:], len(token) > 1
	case strings.HasPrefix(token, "<"):
		return OperatorLess, token[1:], len(token) > 1
	case strings.HasPrefix(token, "="):
		return OperatorEqual, token[1:], len(token) > 1
	default:
		return OperatorEqual, "", false
	}
}

// contradictoryComparators rejects duplicate clauses and conjunctions whose
// exact/lower/upper bounds cannot possibly be satisfied.
func contradictoryComparators(clauses []VersionComparator) bool {
	var exact *Version
	var lower *Version
	var lowerInclusive bool
	var upper *Version
	var upperInclusive bool

	seen := make(map[string]struct{}, len(clauses))
	for i := range clauses {
		clause := clauses[i]
		key := comparatorKey(clause)
		if _, exists := seen[key]; exists {
			return true
		}
		seen[key] = struct{}{}

		switch clause.Operator {
		case OperatorEqual:
			if exact != nil {
				return true
			}
			v := clause.Version
			exact = &v
		case OperatorGreater, OperatorGreaterOrEqual:
			inclusive := clause.Operator == OperatorGreaterOrEqual
			if lower == nil || clause.Version.Compare(*lower) > 0 ||
				(clause.Version.Compare(*lower) == 0 && !inclusive && lowerInclusive) {
				v := clause.Version
				lower = &v
				lowerInclusive = inclusive
			}
		case OperatorLess, OperatorLessOrEqual:
			inclusive := clause.Operator == OperatorLessOrEqual
			if upper == nil || clause.Version.Compare(*upper) < 0 ||
				(clause.Version.Compare(*upper) == 0 && !inclusive && upperInclusive) {
				v := clause.Version
				upper = &v
				upperInclusive = inclusive
			}
		}
	}

	if exact != nil {
		for _, clause := range clauses {
			if !compareClause(*exact, clause) {
				return true
			}
		}
	}
	if lower != nil && upper != nil {
		comparison := lower.Compare(*upper)
		if comparison > 0 || comparison == 0 && (!lowerInclusive || !upperInclusive) {
			return true
		}
	}
	return false
}

func comparatorKey(clause VersionComparator) string {
	return string(rune(clause.Operator)) + versionPrecedenceKey(clause.Version)
}

func versionPrecedenceKey(v Version) string {
	key := v.Major + "." + v.Minor + "." + v.Patch
	if len(v.Prerelease) > 0 {
		key += "-" + strings.Join(v.Prerelease, ".")
	}
	return key
}

// Matches reports whether candidate satisfies every comparator and SemVer's
// prerelease exclusion rule. Build metadata is ignored for precedence.
func (r VersionRequirement) Matches(candidate Version) bool {
	if len(r.Comparators) == 0 || len(r.Comparators) > 4 {
		return false
	}
	for _, clause := range r.Comparators {
		if !compareClause(candidate, clause) {
			return false
		}
	}
	if len(candidate.Prerelease) == 0 {
		return true
	}

	// A prerelease candidate is eligible only when at least one comparator
	// explicitly mentions a prerelease with the same major/minor/patch tuple.
	for _, clause := range r.Comparators {
		version := clause.Version
		if len(version.Prerelease) > 0 &&
			candidate.Major == version.Major &&
			candidate.Minor == version.Minor &&
			candidate.Patch == version.Patch {
			return true
		}
	}
	return false
}

func compareClause(candidate Version, clause VersionComparator) bool {
	comparison := candidate.Compare(clause.Version)
	switch clause.Operator {
	case OperatorEqual:
		return comparison == 0
	case OperatorGreater:
		return comparison > 0
	case OperatorGreaterOrEqual:
		return comparison >= 0
	case OperatorLess:
		return comparison < 0
	case OperatorLessOrEqual:
		return comparison <= 0
	default:
		return false
	}
}
