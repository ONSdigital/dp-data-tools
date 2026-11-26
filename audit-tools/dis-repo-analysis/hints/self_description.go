package hints

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	inlineCode       = regexp.MustCompile("`+[^`]*`+")
	markdownLink     = regexp.MustCompile(`!?\[[^\]]*\]\([^)]*\)`)
	htmlComment      = regexp.MustCompile(`(?s)<!--.*?-->`)
	markdownEmphasis = regexp.MustCompile(`\*{1,2}([^*]+)\*{1,2}|_{1,2}([^_]+)_{1,2}`)
)

const (
	pocNoun         = `(?:proof[ -]of[ -]concept|poc)(?:\s+(?:application|project|repository|implementation|experiment))?`
	testingNoun     = `test[ -](?:data|content)|fixtures?`
	applicationNoun = `application`
	libraryNoun     = `library`
	typeNoun        = `(?:` + pocNoun + `|` + testingNoun + `|` + applicationNoun + `|` + libraryNoun + `)`
	typeQualifiers  = `(?:(?:a|an|the|go|golang|java|javascript|typescript|python|node\.?js|reusable|shared|client|simple|small|experimental|prototype|web|backend|frontend|microservice|utility)\s+){0,6}`
)

// selfDescriptionHints extracts type hints from the repository's self-description text.
func selfDescriptionHints(name, source, text string, readme bool) []typeHint {
	// Match self-description grammar, not arbitrary mentions of dependencies or examples.
	subject := `(?:(?:this(?:\s+(?:repository|repo|project|application|library))?|` + regexp.QuoteMeta(name) + `)\s+(?:is|provides|contains)\s+)`
	pattern := regexp.MustCompile(`(?i)^(?:` + subject + `)?` + typeQualifiers + `(` + typeNoun + `)\b(?:\s*$|\s*[.!:,()]|\s+(?:for|to|that|which|with|providing|implementing|used|written|built|in|of|by)\b)`)
	wrapper := wrapperPattern(subject)
	var hints []typeHint
	fence := ""
	introduction := true
	for _, line := range strings.Split(htmlComment.ReplaceAllString(text, ""), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			switch fence {
			case "":
				fence = marker
			case marker:
				fence = ""
			}
			continue
		}
		if fence != "" || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") ||
			strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			continue
		}
		if readme && strings.HasPrefix(trimmed, "##") {
			heading := strings.ToLower(strings.TrimSpace(strings.Trim(trimmed, "#")))
			introduction = heading == "overview" || heading == "about" || heading == "about us" || heading == "introduction"
		}
		if readme && !introduction {
			continue
		}

		trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		trimmed = inlineCode.ReplaceAllString(trimmed, "[code]")
		trimmed = markdownLink.ReplaceAllString(trimmed, "[link]")
		trimmed = markdownEmphasis.ReplaceAllString(trimmed, "$1$2")
		if match := wrapper.FindString(strings.TrimSpace(trimmed)); match != "" {
			hints = append(hints, wrapperHint(source, match))
			continue
		}

		match := pattern.FindStringSubmatch(strings.TrimSpace(trimmed))
		if match == nil {
			continue
		}

		kind := strings.ToLower(match[1])
		switch {
		case strings.HasPrefix(kind, "test") || strings.HasPrefix(kind, "fixture"):
			kind = "testing"
		case kind != "application" && kind != "library":
			kind = "poc"
		}
		hints = append(hints, typeHint{kind: kind, evidence: fmt.Sprintf("%s: explicit %s self-description (%s)", source, kind, strings.TrimSpace(match[0]))})
	}
	return hints
}

// applyTypeHints applies the extracted type hints to determine the repository type.
func applyTypeHints(result Result, hints []typeHint) Result {
	if result.Type != "unknown" && result.Type != "application" && result.Type != "library" {
		return result
	}

	kinds := make(map[string]bool)
	for _, hint := range hints {
		kinds[hint.kind] = true
	}

	switch {
	case kinds["poc"]:
		result.Type = "poc"
	case result.Type != "unknown":
		return result
	case len(kinds) == 1:
		for kind := range kinds {
			result.Type = kind
		}
	case len(kinds) > 1:
		if kinds["testing"] {
			result.Evidence = append(result.Evidence, "conflicting repository-type self-descriptions; type remains unknown")
		} else {
			result.Evidence = append(result.Evidence, "conflicting application/library self-descriptions; type remains unknown")
		}
	}
	for _, hint := range hints {
		result.Evidence = append(result.Evidence, hint.evidence)
	}
	return normaliseEvidence(result)
}
