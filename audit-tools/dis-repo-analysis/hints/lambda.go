package hints

import "regexp"

var lambdaName = regexp.MustCompile(`(?i)(?:^|-)lambda(?:-|$)`)

// applyLambdaHint applies a fallback hint based on the repository name containing "lambda".
func applyLambdaHint(result Result, name string) Result {
	if lambdaName.MatchString(name) {
		result.Type = "lambda"
		result.Evidence = append(result.Evidence, "repository name "+name+": Lambda token")
	}
	return result
}
