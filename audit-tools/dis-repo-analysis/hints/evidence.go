package hints

import "slices"

// normaliseEvidence sorts and removes duplicate entries from the evidence list in the result.
func normaliseEvidence(result Result) Result {
	slices.Sort(result.Evidence)
	result.Evidence = slices.Compact(result.Evidence)
	return result
}
