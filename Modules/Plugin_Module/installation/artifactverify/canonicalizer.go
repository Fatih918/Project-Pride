package artifactverify

import "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"

// JCSCanonicalizer implements the RFC 8785 JSON Canonicalization Scheme.
type JCSCanonicalizer struct{}

func (JCSCanonicalizer) Canonicalize(value []byte) ([]byte, error) {
	return jsoncanonicalizer.Transform(value)
}
