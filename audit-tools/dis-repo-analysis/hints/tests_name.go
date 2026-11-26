package hints

import "strings"

// applyTestsNameHint applies a fallback hint based on the repository name containing "-tests".
func applyTestsNameHint(result Result, name string) Result {
	if result.Type == "unknown" && strings.HasSuffix(strings.ToLower(name), "-tests") {
		result.Type = "testing"
		result.Evidence = append(result.Evidence, "repository name "+name+": -tests suffix fallback")
	}
	return result
}
