package hints

import "regexp"

var pocName = regexp.MustCompile(`(?i)(?:^|-)poc(?:-|$)`)

// applyPoCNameHint applies a fallback hint based on the repository name containing "poc".
func applyPoCNameHint(result Result, name string) Result {
	if pocName.MatchString(name) {
		result.Type = "poc"
		result.Evidence = append(result.Evidence, "repository name "+name+": PoC token")
	}
	return result
}
