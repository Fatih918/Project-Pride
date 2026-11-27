package capability

import "testing"

func TestParseVersion(t *testing.T) {
	valid := []string{
		"0.0.0", "1.2.3", "1.2.3-alpha", "1.2.3-alpha.1",
		"1.2.3-0A.1", "1.2.3+build.7", "1.2.3-alpha+build.7",
		"999999999999999999999999.0.0",
	}
	for _, raw := range valid {
		if _, err := ParseVersion(raw); err != nil {
			t.Errorf("ParseVersion(%q): %v", raw, err)
		}
	}

	invalid := []string{
		"", "v1.2.3", "1.2", "1.2.3.4", "01.2.3", "1.02.3",
		"1.2.03", "1.2.3-", "1.2.3+", "1.2.3-alpha..1",
		"1.2.3-01", "1.2.3+build..7", " 1.2.3", "1.2.3 ",
		"1.2.3+meta+again", "1.2.3-α",
	}
	for _, raw := range invalid {
		if _, err := ParseVersion(raw); err == nil {
			t.Errorf("ParseVersion(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestVersionCompareSemVerPrecedence(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta",
		"1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11",
		"1.0.0-rc.1", "1.0.0",
	}
	for i := 0; i < len(ordered)-1; i++ {
		left, err := ParseVersion(ordered[i])
		if err != nil {
			t.Fatal(err)
		}
		right, err := ParseVersion(ordered[i+1])
		if err != nil {
			t.Fatal(err)
		}
		if left.Compare(right) >= 0 {
			t.Errorf("expected %s < %s", ordered[i], ordered[i+1])
		}
	}
}

func TestVersionCompareIgnoresBuildMetadata(t *testing.T) {
	left, _ := ParseVersion("1.2.3+one")
	right, _ := ParseVersion("1.2.3+two")
	if left.Compare(right) != 0 {
		t.Fatal("build metadata must not affect precedence")
	}
}
