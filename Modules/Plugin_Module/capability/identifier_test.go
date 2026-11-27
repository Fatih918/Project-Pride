package capability

import "testing"

func TestValidateID(t *testing.T) {
	valid := []string{
		"com.example/document.summarize",
		"org.acme/workflow.plan",
		"dev-team.tools/file-index",
		"a1.b2/c3.d4",
	}
	for _, value := range valid {
		if err := ValidateID(value); err != nil {
			t.Errorf("ValidateID(%q): %v", value, err)
		}
	}

	invalid := []string{
		"", "example/summarize", "com..example/summarize",
		"com.example/-summarize", "com.example/summarize-",
		"com.example/", "com.example/summarize name",
		"Com.example/summarize", "com.example/summarize/",
		"com.example//summarize", "com.example/summarize\n",
		"com.example/ümlaut", "io.pride/text.generate",
	}
	for _, value := range invalid {
		if err := ValidateID(value); err == nil {
			t.Errorf("ValidateID(%q) unexpectedly succeeded", value)
		}
	}
}

func TestValidateIDLength(t *testing.T) {
	value := "ab.cd/" + string(make([]byte, 0))
	_ = value
	tooLong := "ab.cd/" + repeat("a", 250)
	if err := ValidateID(tooLong); err == nil {
		t.Fatal("expected identifier over 255 bytes to be rejected")
	}
}

func TestValidateRuntimeProfileID(t *testing.T) {
	for _, value := range []string{"io.pride.runtime/linux-x64", "io.pride.runtime/container.rootless"} {
		if err := ValidateRuntimeProfileID(value); err != nil {
			t.Errorf("ValidateRuntimeProfileID(%q): %v", value, err)
		}
	}
	for _, value := range []string{"io.pride/runtime", "io.pride.runtime/", "io.pride.runtime/-linux", "com.example/runtime"} {
		if err := ValidateRuntimeProfileID(value); err == nil {
			t.Errorf("ValidateRuntimeProfileID(%q) unexpectedly succeeded", value)
		}
	}
}

func repeat(value string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}
