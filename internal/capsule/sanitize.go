package capsule

import "regexp"

var (
	githubGitlabAWS = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,}|glpat-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16})\b`)
	jwtLike         = regexp.MustCompile(`\b[A-Za-z0-9_-]{24,}\.[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{20,}\b`)
	bearerHeader    = regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s]+`)
	basicHeader     = regexp.MustCompile(`(?i)(authorization\s*:\s*basic\s+)[^\s]+`)
	urlPassword     = regexp.MustCompile(`([a-z][a-z0-9+.-]*://[^\s/:@]+:)[^\s@/]+(@)`)
	pemBlock        = regexp.MustCompile(`(?s)-----BEGIN [^-]+-----.*?-----END [^-]+-----`)
)

// SanitizeText applies explicit masks and bounded credential-pattern redaction.
// It cannot prove arbitrary unknown secret formats are absent.
func SanitizeText(text string, explicitMasks []string) string {
	for _, value := range explicitMasks {
		if value != "" {
			text = regexp.MustCompile(regexp.QuoteMeta(value)).ReplaceAllString(text, "[REDACTED]")
		}
	}
	text = pemBlock.ReplaceAllString(text, "[REDACTED PRIVATE KEY]")
	text = githubGitlabAWS.ReplaceAllString(text, "[REDACTED]")
	text = jwtLike.ReplaceAllString(text, "[REDACTED]")
	text = bearerHeader.ReplaceAllString(text, "${1}[REDACTED]")
	text = basicHeader.ReplaceAllString(text, "${1}[REDACTED]")
	return urlPassword.ReplaceAllString(text, "${1}[REDACTED]${2}")
}
