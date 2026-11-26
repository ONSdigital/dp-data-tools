package hints

import "regexp"

var applicationDescription = regexp.MustCompile(`(?i)\b(?:api|platform)\b`)

// applicationDescriptionEvidence checks the repository description for application-related keywords and returns evidence if found.
func applicationDescriptionEvidence(description string) []string {
	if match := applicationDescription.FindString(description); match != "" {
		return []string{"repository description: application keyword (" + match + ")"}
	}
	return nil
}
