package analysis

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/hints"
)

// classifyProject classifies a project based on its files and manifest contents.
func classifyProject(p *project, read func(string) ([]byte, error)) (string, []string, error) {
	if hints.SupportingTypePath(p.root) {
		return "", nil, nil
	}

	signals := make(map[string][]string)
	add := func(kind, evidence string) { signals[kind] = append(signals[kind], evidence) }
	hasGoModule := slices.Contains(p.files, path.Join(p.root, "go.mod"))
	hasManifest := false
	hasSource := false
	hasChart := false
	for _, name := range p.files {
		if hints.SupportingTypePath(path.Dir(name)) {
			continue
		}
		base := path.Base(name)
		hasManifest = hasManifest || isProjectManifest(base)
		switch path.Ext(name) {
		case ".go", ".java", ".kt", ".scala", ".js", ".jsx", ".ts", ".tsx", ".py", ".rs", ".rb", ".c", ".cpp", ".cs":
			hasSource = true
		}
		if base == "__main__.py" {
			add("application", name+": Python module entry point")
		}
		if path.Ext(name) == ".tf" {
			add("infrastructure", name+": Terraform configuration")
		}
		if base == "Chart.yaml" {
			hasChart = true
			add("infrastructure", name+": Helm chart")
		}
		if hasGoModule && path.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go") &&
			(base == "main.go" || path.Dir(name) == p.root) {
			data, err := read(name)
			if err != nil {
				return "", nil, err
			}
			if err := addGoFileSignals(name, p.root, data, signals); err != nil {
				return "", nil, err
			}
		}
		switch base {
		case "package.json":
			data, err := read(name)
			if err != nil {
				return "", nil, err
			}
			kind, reason, err := manifestType(base, data)
			if err != nil {
				return "", nil, fmt.Errorf("parse %s: %w", name, err)
			}
			if kind != "" {
				add(kind, name+": "+reason)
			}
		}
	}

	if len(signals["application"]) > 0 {
		return "application", signals["application"], nil
	}
	if len(signals["library"]) > 0 {
		return "library", signals["library"], nil
	}
	if hasChart {
		return "infrastructure", signals["infrastructure"], nil
	}
	if hasManifest || hasSource {
		return "unknown", []string{p.root + ": insufficient executable or library evidence in source/manifests"}, nil
	}
	if len(signals["infrastructure"]) > 0 {
		return "infrastructure", signals["infrastructure"], nil
	}
	return "", nil, nil
}
