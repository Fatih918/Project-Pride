package capability

import "strings"

// ValidateID validates the canonical POCS-0001 capability identifier grammar.
// It performs no normalization: callers must compare the original identifier.
func ValidateID(id string) error {
	if err := validateCanonicalID(id); err != nil {
		return err
	}
	namespace := strings.SplitN(id, "/", 2)[0]
	if namespace == "io.pride" || strings.HasPrefix(namespace, "io.pride.") {
		return ErrReservedNamespace
	}
	return nil
}

// validateCanonicalID checks identifier syntax without deciding namespace
// ownership. It is also used when validating references to PRIDE-owned IDs.
func validateCanonicalID(id string) error {
	if len(id) == 0 || len(id) > 255 || !isASCII(id) {
		return ErrInvalidIdentifier
	}
	parts := strings.Split(id, "/")
	if len(parts) != 2 || !validNamespace(parts[0]) || !validSegments(parts[1]) {
		return ErrInvalidIdentifier
	}
	return nil
}

// ValidateRuntimeProfileID validates a PRIDE-owned runtime profile identifier.
// Runtime profile identifiers are the only third-party-visible use of the
// reserved io.pride namespace defined by POCS-0001.
func ValidateRuntimeProfileID(id string) error {
	if len(id) == 0 || len(id) > 128 || !isASCII(id) {
		return ErrInvalidIdentifier
	}
	const prefix = "io.pride.runtime/"
	if !strings.HasPrefix(id, prefix) {
		return ErrInvalidRuntimeProfileID
	}
	if !validSegments(strings.TrimPrefix(id, prefix)) {
		return ErrInvalidIdentifier
	}
	return nil
}

func validNamespace(namespace string) bool {
	segments := strings.Split(namespace, ".")
	return len(segments) >= 2 && validSegmentList(segments)
}

func validSegments(value string) bool {
	return validSegmentList(strings.Split(value, "."))
}

func validSegmentList(segments []string) bool {
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if !validSegment(segment) {
			return false
		}
	}
	return true
}

func validSegment(segment string) bool {
	if len(segment) == 0 || !isAlphaNumeric(segment[0]) || !isAlphaNumeric(segment[len(segment)-1]) {
		return false
	}
	for i := 0; i < len(segment); i++ {
		c := segment[i]
		if !isAlphaNumeric(c) && c != '-' {
			return false
		}
	}
	return true
}

func isAlphaNumeric(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func isASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 0x7f {
			return false
		}
	}
	return true
}
