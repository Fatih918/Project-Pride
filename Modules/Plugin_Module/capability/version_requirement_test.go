package capability

import "testing"

func mustRequirement(t *testing.T, raw string) VersionRequirement {
	t.Helper()
	requirement, err := ParseVersionRequirement(raw)
	if err != nil {
		t.Fatalf("ParseVersionRequirement(%q): %v", raw, err)
	}
	return requirement
}

func mustVersion(t *testing.T, raw string) Version {
	t.Helper()
	version, err := ParseVersion(raw)
	if err != nil {
		t.Fatalf("ParseVersion(%q): %v", raw, err)
	}
	return version
}

func TestParseVersionRequirementValidGrammar(t *testing.T) {
	valid := []string{
		"1.2.3",
		"=1.2.3",
		">1.2.3",
		">=1.2.0 <2.0.0",
		">1.4.0 <=1.8.2",
		">=1.0.0 <2.0.0 <=1.8.0",
		">=1.0.0  <2.0.0",
	}
	for _, raw := range valid {
		if _, err := ParseVersionRequirement(raw); err != nil {
			t.Errorf("expected %q to be valid: %v", raw, err)
		}
	}
}

func TestParseVersionRequirementRejectsUnsupportedGrammar(t *testing.T) {
	invalid := []string{
		"", " ", " 1.2.3", "1.2.3 ", "1.2", "v1.2.3",
		"^1.2.3", "~1.2.3", "*", "1.x", "1.2.x",
		">=1.2.0 || <2.0.0", ">=1.2.0, <2.0.0",
		">=1.2.0\t<2.0.0", ">=1.2.0\n<2.0.0",
		">=", "><1.2.3", "==1.2.3",
		">=1.0.0 >=1.0.0", ">=2.0.0 <1.0.0",
		">1.0.0 <=1.0.0", ">1.0.0 <1.0.0", "1.2.3 <2.0.0",
		">=1.0.0 <2.0.0 <=1.5.0 >1.6.0",
	}
	for _, raw := range invalid {
		if _, err := ParseVersionRequirement(raw); err == nil {
			t.Errorf("expected %q to be rejected", raw)
		}
	}
}

func TestVersionRequirementMatchesComparators(t *testing.T) {
	tests := []struct {
		requirement string
		candidate   string
		want        bool
	}{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"=1.2.3", "1.2.3+build.5", true},
		{">1.2.3", "1.2.4", true},
		{">1.2.3", "1.2.3", false},
		{">=1.2.3", "1.2.3", true},
		{"<2.0.0", "1.99.99", true},
		{"<=2.0.0", "2.0.0", true},
		{">=1.2.0 <2.0.0", "1.8.2", true},
		{">=1.2.0 <2.0.0", "2.0.0", false},
	}
	for _, test := range tests {
		requirement := mustRequirement(t, test.requirement)
		candidate := mustVersion(t, test.candidate)
		if got := requirement.Matches(candidate); got != test.want {
			t.Errorf("%q matches %q = %v, want %v", test.requirement, test.candidate, got, test.want)
		}
	}
}

func TestVersionRequirementPrereleaseExclusion(t *testing.T) {
	tests := []struct {
		requirement string
		candidate   string
		want        bool
	}{
		{">=1.0.0 <2.0.0", "1.5.0-alpha", false},
		{">=1.0.0-alpha <2.0.0", "1.0.0-alpha", true},
		{">=1.0.0-alpha <2.0.0", "1.5.0-beta", false},
		{">=1.0.0-alpha <2.0.0", "1.0.0-beta", true},
		{"=1.0.0-alpha", "1.0.0-alpha", true},
		{"=1.0.0", "1.0.0-alpha", false},
	}
	for _, test := range tests {
		requirement := mustRequirement(t, test.requirement)
		candidate := mustVersion(t, test.candidate)
		if got := requirement.Matches(candidate); got != test.want {
			t.Errorf("%q matches %q = %v, want %v", test.requirement, test.candidate, got, test.want)
		}
	}
}

func TestVersionRequirementRejectsMoreThanFourComparators(t *testing.T) {
	if _, err := ParseVersionRequirement(">=1.0.0 >1.1.0 <2.0.0 <=1.9.0 =1.5.0"); err == nil {
		t.Fatal("expected more than four comparators to be rejected")
	}
}

func TestVersionRequirementBuildMetadataDoesNotAffectMatching(t *testing.T) {
	requirement := mustRequirement(t, "=1.2.3+first")
	candidate := mustVersion(t, "1.2.3+second")
	if !requirement.Matches(candidate) {
		t.Fatal("build metadata must not affect SemVer precedence matching")
	}
}
