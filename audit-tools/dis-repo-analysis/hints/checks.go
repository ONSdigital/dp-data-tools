package hints

import (
	"path"
	"slices"
	"strings"

	"github.com/google/go-github/v74/github"
)

// IgnoredPath checks if the given path should be ignored based on common directory names and patterns.
func IgnoredPath(name string) bool {
	for part := range strings.SplitSeq(name, "/") {
		switch part {
		case ".git", "vendor", "node_modules", "dist", "build", "target", "coverage", ".venv", "venv", "__pycache__", ".next", "testdata":
			return true
		}
	}
	return false
}

// SupportingTypePath checks if the given path belongs to directories typically used for supporting files, such as examples or tests.
func SupportingTypePath(name string) bool {
	for directory := range strings.SplitSeq(name, "/") {
		if directory == "examples" || directory == "test" || directory == "tests" {
			return true
		}
	}
	return false
}

// RepositoryFiles filters the given GitHub tree entries to return only regular files that are not ignored.
func RepositoryFiles(entries []*github.TreeEntry) []string {
	var files []string
	for _, entry := range entries {
		name := entry.GetPath()
		if entry.GetType() != "blob" || entry.GetMode() == "120000" || IgnoredPath(name) {
			continue
		}
		files = append(files, name)
	}
	slices.Sort(files)
	return files
}

// DocumentationOnly checks if the given list of files contains only documentation and configuration files, optionally allowing certain config files.
func DocumentationOnly(files []string, allowConfig bool) bool {
	hasDocs := false
	for _, name := range files {
		if allowConfig {
			switch strings.ToLower(name) {
			case ".editorconfig", ".markdownlint.json", ".markdownlint.yaml", ".markdownlint.yml",
				"mkdocs.yml", "mkdocs.yaml", "_config.yml":
				continue
			}
		}
		base := strings.ToLower(path.Base(name))
		switch path.Ext(base) {
		case ".md", ".rst", ".adoc":
			hasDocs = true
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".txt":
		case ".yml", ".yaml":
			if !strings.HasPrefix(name, ".github/") {
				return false
			}
		default:
			if base != "license" && base != "codeowners" && base != ".gitignore" && base != ".gitattributes" {
				return false
			}
		}
	}
	return hasDocs
}
