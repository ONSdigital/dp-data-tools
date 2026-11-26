package hints

import "github.com/google/go-github/v74/github"

type Result struct {
	Type     string
	Evidence []string
}

type Input struct {
	Name        string
	Description string
	Entries     []*github.TreeEntry
	Languages   []string
}

type typeHint struct {
	kind     string
	evidence string
	wrapper  bool
}

// ApplyRepository applies hints that can settle the type before a README lookup.
func ApplyRepository(result Result, input Input) Result {
	return applyNameHints(applyHelmHints(result, input.Name, input.Entries, input.Languages), input.Name)
}

// Finalise applies text hints and fallbacks after the caller has fetched the README.
// An empty readmeSource represents an absent README.
func Finalise(result Result, input Input, readmeSource, readme string) Result {
	hints := selfDescriptionHints(input.Name, "repository description", input.Description, false)
	if readmeSource != "" {
		hints = append(hints, selfDescriptionHints(input.Name, readmeSource, readme, true)...)
	}

	return finaliseTypeHints(result, hints, input.Name, input.Description, input.Entries, input.Languages)
}

// applyNameHints applies hints based on the repository name.
func applyNameHints(result Result, name string) Result {
	result = applyLambdaHint(result, name)
	result = applyPoCNameHint(result, name)
	return normaliseEvidence(result)
}

// ApplyUnknownNameFallbacks also handles repositories confirmed empty by GitHub.
func ApplyUnknownNameFallbacks(result Result, name string) Result {
	result = applyInfraNameHint(result, name)
	result = applyTestsNameHint(result, name)
	return normaliseEvidence(result)
}

// finaliseTypeHints applies all collected type hints and determines the final repository type.
func finaliseTypeHints(result Result, hints []typeHint, name, description string, entries []*github.TreeEntry, languages []string) Result {
	preserveExplicitIntent := false
	wrapperIntent := false
	for _, hint := range hints {
		preserveExplicitIntent = preserveExplicitIntent || hint.kind == "testing" || hint.kind == "poc"
		wrapperIntent = wrapperIntent || hint.wrapper
	}

	if (result.Type == "unknown" || result.Type == "application" || result.Type == "library") && !preserveExplicitIntent {
		result, applied := applyLibraryIntent(result, description, wrapperIntent)
		if applied {
			for _, hint := range hints {
				result.Evidence = append(result.Evidence, hint.evidence)
			}
			return normaliseEvidence(result)
		}
	}

	result = applyTypeHints(result, hints)
	if result.Type != "unknown" {
		return result
	}

	kinds := make(map[string]bool)
	for _, hint := range hints {
		kinds[hint.kind] = true
	}
	if len(kinds) > 1 {
		return result
	}

	if result, applied := applyDocumentationHint(result, entries, languages); applied {
		return normaliseEvidence(result)
	}

	evidence := applicationDescriptionEvidence(description)
	evidence = append(evidence, dockerfileConcourseEvidence(entries)...)
	if len(evidence) > 0 {
		result.Type = "application"
		result.Evidence = append(result.Evidence, evidence...)
	} else {
		result = applyLibraryNameHint(result, name)
		result = applyCIHint(result, entries)
	}

	return ApplyUnknownNameFallbacks(normaliseEvidence(result), name)
}
