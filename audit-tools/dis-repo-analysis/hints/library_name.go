package hints

import "strings"

// applyLibraryNameHint applies a fallback hint based on the repository name containing "-library-".
func applyLibraryNameHint(result Result, name string) Result {
	if result.Type == "unknown" && strings.Contains(strings.ToLower(name), "-library-") {
		result.Type = "library"
		result.Evidence = append(result.Evidence, "repository name "+name+": -library- fallback")
	}
	return result
}
