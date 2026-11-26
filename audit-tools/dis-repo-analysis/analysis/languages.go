package analysis

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-github/v74/github"
)

type Language struct {
	Name       string
	Bytes      int
	Percentage float64
}

// RepoLanguages uses the complete Linguist byte total, not just displayed languages.
func RepoLanguages(ctx context.Context, client *github.Client, repo *github.Repository) ([]Language, error) {
	if err := validateRepository(client, repo); err != nil {
		return nil, err
	}

	counts, _, err := client.Repositories.ListLanguages(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		return nil, fmt.Errorf("get languages for %s: %w", repo.GetFullName(), err)
	}

	var total int64
	for _, count := range counts {
		total += int64(count)
	}

	languages := make([]Language, 0)
	if total == 0 {
		return languages, nil
	}

	for name, count := range counts {
		// Compare before rounding; exactly 10 percent is excluded.
		if int64(count) <= total/10 {
			continue
		}
		languages = append(languages, Language{
			Name: name, Bytes: count, Percentage: 100 * float64(count) / float64(total),
		})
	}

	slices.SortFunc(languages, func(a, b Language) int {
		if a.Bytes > b.Bytes {
			return -1
		}
		if a.Bytes < b.Bytes {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return languages, nil
}
