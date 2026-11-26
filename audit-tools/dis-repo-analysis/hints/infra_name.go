package hints

import "strings"

// applyInfraNameHint applies a fallback hint based on the repository name containing "-infra-".
func applyInfraNameHint(result Result, name string) Result {
	if result.Type == "unknown" && strings.Contains(strings.ToLower(name), "-infra-") {
		result.Type = "infrastructure"
		result.Evidence = append(result.Evidence, "repository name "+name+": -infra- fallback")
	}
	return result
}
