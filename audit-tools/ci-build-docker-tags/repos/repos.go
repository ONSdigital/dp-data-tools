package repos

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
)

const perPage = 100

// SearchRepos searches for repositories in the ONSdigital organisation that have a CODEOWNERS file containing the specified code owner. It returns a map of repository full names to GitHub repository objects.
func SearchRepos(ctx context.Context, client *github.Client, codeOwner string) map[string]*github.Repository {
	repos := make(map[string]*github.Repository, 0)

	fmt.Printf("Searching repos with owner %s", codeOwner)

	for page := 1; page > 0; {
		code, r, err := client.Search.Code(
			ctx,
			codeOwner+" org:onsdigital filename:CODEOWNERS",
			&github.SearchOptions{ListOptions: github.ListOptions{Page: page, PerPage: perPage}},
		)
		if err != nil {
			log.Fatal("search failed", err)
		}
		for _, result := range code.CodeResults {
			if repo := result.Repository; repo != nil {
				repos[*(repo.FullName)] = repo
			}
		}
		page = r.NextPage
		fmt.Print(".")
	}
	fmt.Println()
	return repos
}

// RepoVisibility reads visibility from full repository details, not search results.
func RepoVisibility(repo *github.Repository) (string, error) {
	if repo == nil {
		return "", fmt.Errorf("repository is required")
	}
	if visibility := repo.GetVisibility(); visibility != "" {
		return visibility, nil
	}
	return "", fmt.Errorf("GitHub returned no visibility for %s", repo.GetFullName())
}

// GetCodeOwners retrieves the list of code owners for the specified repository from its CODEOWNERS file. It returns a slice of code owner strings.
func GetCodeOwners(ctx context.Context, client *github.Client, repo *github.Repository) ([]string, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if client == nil || repo.GetOwner().GetLogin() == "" || repo.GetName() == "" {
		return nil, fmt.Errorf("GitHub client, repository owner and name are required to fetch code owners")
	}

	var fileContent *github.RepositoryContent
	for _, path := range []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"} {
		content, _, response, err := client.Repositories.GetContents(ctx, repo.GetOwner().GetLogin(), repo.GetName(), path, nil)
		if err != nil {
			if response != nil && response.StatusCode == http.StatusNotFound {
				continue
			}
			return nil, fmt.Errorf("get %s for %s: %w", path, repo.GetFullName(), err)
		}
		if content == nil {
			return nil, fmt.Errorf("%s for %s is not a file", path, repo.GetFullName())
		}
		fileContent = content
		break
	}
	if fileContent == nil {
		return nil, fmt.Errorf("no CODEOWNERS file found for %s in .github/, root or docs/", repo.GetFullName())
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("read CODEOWNERS content for %s: %w", repo.GetFullName(), err)
	}

	var codeOwners []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 1 {
			codeOwners = append(codeOwners, parts[1:]...)
		}
	}
	return codeOwners, nil
}
