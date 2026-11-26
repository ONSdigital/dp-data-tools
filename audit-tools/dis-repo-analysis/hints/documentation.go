package hints

import (
	"github.com/google/go-github/v74/github"
)

// applyDocumentationHint checks if the repository contains only documentation and supporting configuration files and updates the result.
func applyDocumentationHint(result Result, entries []*github.TreeEntry, languages []string) (Result, bool) {
	if len(languages) != 0 || !DocumentationOnly(RepositoryFiles(entries), true) {
		return result, false
	}

	result.Type = "documentation"
	result.Evidence = append(result.Evidence, "documentation and supporting configuration without code or project manifests (no reported languages)")
	return result, true
}
