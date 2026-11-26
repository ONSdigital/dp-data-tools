package analysis

// isProjectManifest checks if the given file name corresponds to a recognised project manifest.
func isProjectManifest(name string) bool {
	switch name {
	case "go.mod", "pom.xml", "package.json", "pyproject.toml", "setup.py", "setup.cfg", "requirements.txt":
		return true
	}
	return false
}

// manifestType determines the type of project based on the manifest file name and its contents.
func manifestType(name string, data []byte) (string, string, error) {
	switch name {
	case "package.json":
		return npmType(data)
	}
	return "", "", nil
}
