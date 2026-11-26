package analysis

import "encoding/json"

// npmType determines the type of npm project based on its manifest contents.
func npmType(data []byte) (string, string, error) {
	var manifest struct {
		Private bool              `json:"private"`
		Main    string            `json:"main"`
		Module  string            `json:"module"`
		Files   []string          `json:"files"`
		Exports any               `json:"exports"`
		Bin     any               `json:"bin"`
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", "", err
	}

	hasValue := func(value any) bool {
		switch value := value.(type) {
		case string:
			return value != ""
		case map[string]any:
			return len(value) > 0
		case []any:
			return len(value) > 0
		default:
			return false
		}
	}
	if manifest.Scripts["start"] != "" || hasValue(manifest.Bin) {
		return "application", "npm start script or executable bin", nil
	}
	if !manifest.Private && (manifest.Module != "" || hasValue(manifest.Exports) ||
		(manifest.Main != "" && len(manifest.Files) > 0)) {
		return "library", "public package entry point", nil
	}
	return "", "", nil
}
