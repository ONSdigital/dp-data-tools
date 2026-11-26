package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/analysis"
)

type Record struct {
	Repository string
	Visibility string
	CodeOwners []string
	Languages  []analysis.Language
	Analysis   analysis.Analysis
}

// WriteCSV replaces the destination only after the complete report is written and closed.
func WriteCSV(destination string, records []Record, withEvidence bool) error {
	return writeReport(destination, "CSV", func(w io.Writer) error { return writeCSV(w, records, withEvidence) })
}

// writeCSV generates a CSV report for the given records and writes it to the provided writer.
func writeCSV(w io.Writer, records []Record, withEvidence bool) error {
	header := []string{"Repository", "Visibility", "Code owners", "Languages", "Type"}
	if withEvidence {
		header = append(header, "Evidence")
	}

	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		return err
	}

	ordered := slices.Clone(records)
	slices.SortFunc(ordered, func(a, b Record) int { return strings.Compare(a.Repository, b.Repository) })
	for _, record := range ordered {
		row := []string{
			stripOrganisationPrefix(record.Repository), record.Visibility, formatList(record.CodeOwners),
			formatLanguages(record.Languages),
			record.Analysis.Type,
		}
		if withEvidence {
			row = append(row, formatList(record.Analysis.Evidence))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// stripOrganisationPrefix removes the organisation prefix from the repository name if it matches known prefixes.
func stripOrganisationPrefix(value string) string {
	for _, prefix := range []string{"ONSdigital/"} {
		if len(value) >= len(prefix) && strings.EqualFold(value[:len(prefix)], prefix) {
			return value[len(prefix):]
		}
	}
	return value
}

// formatList formats a list of strings for CSV output, returning "none detected" if the list is empty.
func formatList(values []string) string {
	if len(values) == 0 {
		return "none detected"
	}
	return strings.Join(values, "; ")
}

// formatLanguages formats a list of programming languages with their percentages for CSV output.
func formatLanguages(languages []analysis.Language) string {
	formatted := make([]string, 0, len(languages))
	for _, language := range languages {
		formatted = append(formatted, fmt.Sprintf("%s (%.2f%%)", language.Name, language.Percentage))
	}
	return formatList(formatted)
}
