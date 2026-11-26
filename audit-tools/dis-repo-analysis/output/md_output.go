package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// WriteMarkdown writes Mermaid summaries without making additional GitHub requests.
func WriteMarkdown(destination, team string, records []Record) error {
	return writeReport(destination, "md", func(w io.Writer) error { return writeMarkdown(w, team, records) })
}

// writeMarkdown generates a Markdown report with Mermaid pie charts for repository analysis.
func writeMarkdown(w io.Writer, team string, records []Record) error {
	visibility, types, languages := make(map[string]int), make(map[string]int), make(map[string]int)
	for _, record := range records {
		visibility[summaryLabel(record.Visibility)]++
		types[summaryLabel(record.Analysis.Type)]++
		seen := make(map[string]bool)
		for _, language := range record.Languages {
			if !seen[language.Name] {
				languages[language.Name]++
				seen[language.Name] = true
			}
		}
	}

	var report strings.Builder
	fmt.Fprintf(&report, "# Repository analysis\n\nAnalysis for %s repositories\n\nTotal repositories: %d\n\n",
		team, len(records))
	report.WriteString("Visibility and type charts count each repository once.\n\n")
	writePie(&report, "Repository visibility", visibility)
	writePie(&report, "Repository types", types)
	report.WriteString("## Languages\n\nLanguages count repositories where the language exceeds 10% of source bytes. " +
		"A repository can contribute to multiple slices, so these are language occurrences, not unique repositories or byte shares. " +
		"Repositories with no qualifying languages do not contribute a slice.\n\n")
	writePie(&report, "Language occurrences", languages)
	_, err := io.WriteString(w, strings.TrimRight(report.String(), "\n")+"\n")
	return err
}

// summaryLabel returns a human-readable label for summary charts, defaulting to "unknown" for empty values.
func summaryLabel(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

// writePie generates a Mermaid pie chart for the given counts and appends it to the report.
func writePie(report *strings.Builder, title string, counts map[string]int) {
	fmt.Fprintf(report, "## %s\n\n", title)
	if len(counts) == 0 {
		report.WriteString("No data available.\n\n")
		return
	}
	fmt.Fprintf(report, "```mermaid\npie showData\n    title %s\n", title)
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	for _, label := range labels {
		fmt.Fprintf(report, "    \"%s\" : %d\n", escapeMermaidLabel(label), counts[label])
	}
	report.WriteString("```\n\n")
}

// escapeMermaidLabel escapes special characters in Mermaid labels to prevent rendering issues.
func escapeMermaidLabel(value string) string {
	return strings.NewReplacer(
		"#", "#35;", "&", "#38;", "\"", "#quot;", "`", "#96;",
		"<", "#60;", ">", "#62;", "\\", "#92;", "\n", " ", "\r", " ",
	).Replace(value)
}
