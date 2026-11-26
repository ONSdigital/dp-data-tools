package hints

import "regexp"

var libraryDescription = regexp.MustCompile(`(?i)\blibrary\s+for\b`)

// applyLibraryIntent applies a hint based on the repository description indicating a library intent.
func applyLibraryIntent(result Result, description string, wrapperIntent bool) (Result, bool) {
	if match := libraryDescription.FindString(description); match != "" {
		result.Type = "library"
		result.Evidence = append(result.Evidence, "repository description: library intent takes precedence ("+match+")")
		return result, true
	}
	return applyWrapperIntent(result, wrapperIntent)
}
