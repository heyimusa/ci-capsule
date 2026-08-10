package capsule

import "testing"

func TestSanitizeRedactsCredentialShapesAndPreservesEvidence(t *testing.T) {
	raw := "https://alice:correct-horse-battery-staple@example.test/path\n" +
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.signature01234567890123456789\n" +
		"Authorization: Basic YWxpY2U6c2VjcmV0\n" +
		"glpat-0123456789abcdefghij\nAKIAIOSFODNN7EXAMPLE\n" +
		"-----BEGIN PRIVATE KEY-----\npretend-private-material\n-----END PRIVATE KEY-----\n" +
		"commit=ae9eed66a000e0d5ce54ed8ca6a08c373c0aa270 sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"

	got := SanitizeText(raw, nil)
	for _, secret := range []string{"correct-horse-battery-staple", "eyJhbGciOiJIUzI1NiJ9", "YWxpY2U6c2VjcmV0", "glpat-0123456789abcdefghij", "AKIAIOSFODNN7EXAMPLE", "pretend-private-material"} {
		if contains(got, secret) {
			t.Fatalf("secret leaked: %q", secret)
		}
	}
	for _, evidence := range []string{"ae9eed66a000e0d5ce54ed8ca6a08c373c0aa270", "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"} {
		if !contains(got, evidence) {
			t.Fatalf("evidence unexpectedly removed: %q", evidence)
		}
	}
}

func TestSanitizeAppliesExplicitMask(t *testing.T) {
	if got := SanitizeText("masked=internal-ci-secret", []string{"internal-ci-secret"}); got != "masked=[REDACTED]" {
		t.Fatalf("SanitizeText() = %q", got)
	}
}

func contains(s, part string) bool {
	return len(part) == 0 || (len(s) >= len(part) && index(s, part) >= 0)
}
func index(s, part string) int {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
