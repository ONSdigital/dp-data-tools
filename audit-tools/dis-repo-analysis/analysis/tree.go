package analysis

import (
	"path"
	"slices"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/hints"
	"github.com/google/go-github/v74/github"
)

type project struct {
	root  string
	files []string
}

type projectType struct {
	root     string
	kind     string
	evidence []string
}

// analyseTree analyses the repository tree to classify projects and determine the overall repository type.
func analyseTree(entries []*github.TreeEntry, read func(string) ([]byte, error)) (Analysis, error) {
	result := Analysis{Type: "unknown"}
	files := hints.RepositoryFiles(entries)
	roots := map[string]bool{".": true}
	hasManifest := false
	for _, name := range files {
		if isProjectManifest(path.Base(name)) {
			hasManifest = true
			roots[path.Dir(name)] = true
		}
		if path.Ext(name) == ".tf" || path.Base(name) == "Chart.yaml" {
			roots[path.Dir(name)] = true
		}
	}

	projects := make(map[string]*project)
	for root := range roots {
		projects[root] = &project{root: root}
	}
	for _, file := range files {
		root := path.Dir(file)
		for !roots[root] {
			root = path.Dir(root)
		}
		projects[root].files = append(projects[root].files, file)
	}

	projectRoots := make([]string, 0, len(projects))
	for root := range projects {
		projectRoots = append(projectRoots, root)
	}
	slices.Sort(projectRoots)
	var detected []projectType
	ambiguous := false
	for _, root := range projectRoots {
		kind, evidence, err := classifyProject(projects[root], read)
		if err != nil {
			return Analysis{}, err
		}
		if kind == "unknown" {
			ambiguous = true
			result.Evidence = append(result.Evidence, evidence...)
		} else if kind != "" {
			detected = append(detected, projectType{root: root, kind: kind, evidence: evidence})
		}
	}

	kinds := make(map[string]bool)
	for _, detectedProject := range detected {
		if detectedProject.kind == "infrastructure" {
			supporting := false
			for _, other := range detected {
				if (other.kind == "application" || other.kind == "library") &&
					(other.root == "." || strings.HasPrefix(detectedProject.root, other.root+"/")) {
					supporting = true
				}
			}
			if supporting {
				for _, evidence := range detectedProject.evidence {
					if strings.HasSuffix(evidence, ": Helm chart") {
						result.Evidence = append(result.Evidence, evidence+" (supporting deployment configuration)")
					}
				}
				continue
			}
		}
		kinds[detectedProject.kind] = true
		result.Evidence = append(result.Evidence, detectedProject.evidence...)
	}
	switch {
	case len(kinds) == 1 && !ambiguous:
		for kind := range kinds {
			result.Type = kind
		}
	case len(kinds) == 0 && !hasManifest && hints.DocumentationOnly(files, false):
		result.Type = "documentation"
		result.Evidence = append(result.Evidence, "documentation files without code or project manifests")
	}
	if result.Type == "unknown" && len(result.Evidence) == 0 {
		result.Evidence = append(result.Evidence, "insufficient supported type evidence")
	}
	slices.Sort(result.Evidence)
	return result, nil
}
