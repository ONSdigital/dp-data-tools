package hints

import (
	"path"
	"slices"
	"strings"

	"github.com/google/go-github/v74/github"
)

// applyCIHint checks for CI configuration files in the repository and updates the result.
func applyCIHint(result Result, entries []*github.TreeEntry) Result {
	if result.Type != "unknown" {
		return result
	}

	var ciPaths []string
	for _, entry := range entries {
		name := entry.GetPath()
		ciPath := name
		if after, ok := strings.CutPrefix(name, "ci/"); ok {
			ciPath = strings.TrimPrefix(after, "build/")
		}
		if entry.GetType() != "blob" || entry.GetMode() == "120000" ||
			IgnoredPath(ciPath) || SupportingTypePath(path.Dir(name)) {
			continue
		}
		if strings.HasPrefix(name, "ci/") || name == ".github/workflows/ci.yml" || name == ".github/workflows/ci.yaml" {
			ciPaths = append(ciPaths, name)
		}
	}
	if len(ciPaths) > 0 {
		slices.Sort(ciPaths)
		result.Type = "library"
		result.Evidence = append(result.Evidence, ciPaths[0]+": CI file library fallback")
	}
	return result
}
