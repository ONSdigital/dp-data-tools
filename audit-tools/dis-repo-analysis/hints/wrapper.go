package hints

import (
	"fmt"
	"regexp"
)

// wrapperPattern returns a compiled regular expression to match explicit wrapper self-descriptions.
func wrapperPattern(subject string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(?:` + subject + `)?` + typeQualifiers + `wrapper\s+(?:for|around)\b`)
}

// wrapperHint constructs a type hint for an explicit wrapper self-description.
func wrapperHint(source, match string) typeHint {
	return typeHint{
		kind: "library", wrapper: true,
		evidence: fmt.Sprintf("%s: explicit library wrapper self-description (%s)", source, match),
	}
}

// applyWrapperIntent applies a hint based on the repository description indicating a wrapper intent.
func applyWrapperIntent(result Result, wrapperIntent bool) (Result, bool) {
	if !wrapperIntent {
		return result, false
	}
	result.Type = "library"
	result.Evidence = append(result.Evidence, "explicit wrapper self-description: library intent takes precedence")
	return result, true
}
