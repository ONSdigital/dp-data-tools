package hints

import (
	"path"
	"strings"

	"github.com/google/go-github/v74/github"
)

// applyHelmHints checks for Helm-related evidence in the repository and updates the result.
func applyHelmHints(result Result, name string, entries []*github.TreeEntry, languages []string) Result {
	helmName := strings.HasPrefix(strings.ToLower(name), "helm")
	if helmName {
		result.Type = "infrastructure"
		result.Evidence = append(result.Evidence, "repository name "+name+": Helm prefix")
	}

	helmChart := false
	for _, entry := range entries {
		if entry.GetType() == "blob" && entry.GetMode() != "120000" &&
			path.Base(entry.GetPath()) == "Chart.yaml" && !IgnoredPath(entry.GetPath()) && !SupportingTypePath(path.Dir(entry.GetPath())) {
			helmChart = true
			if helmName {
				result.Evidence = append(result.Evidence, entry.GetPath()+": Helm chart")
			}
		}
	}

	if result.Type == "infrastructure" && (helmName || helmChart) {
		for _, language := range languages {
			if language == "Shell" {
				result.Evidence = append(result.Evidence, "Shell reported by GitHub: supporting Helm infrastructure evidence")
			}
		}
	}
	return normaliseEvidence(result)
}
