package analysis

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/hints"
	"github.com/google/go-github/v74/github"
)

type Analysis struct {
	Type     string
	Evidence []string
}

// validateRepository ensures the GitHub client, repository owner, and repository name are provided.
func validateRepository(client *github.Client, repo *github.Repository) error {
	if client == nil || repo == nil || repo.GetOwner().GetLogin() == "" || repo.GetName() == "" {
		return fmt.Errorf("GitHub client, repository owner and name are required")
	}
	return nil
}

// AnalyseRepo requires full repository details, including the default branch.
func AnalyseRepo(ctx context.Context, client *github.Client, repo *github.Repository, languages []Language) (Analysis, error) {
	if err := validateRepository(client, repo); err != nil {
		return Analysis{}, err
	}

	branch := repo.GetDefaultBranch()
	if branch == "" {
		return Analysis{}, fmt.Errorf("no default branch for %s", repo.GetFullName())
	}

	input := hints.Input{Name: repo.GetName(), Description: repo.GetDescription()}
	for _, language := range languages {
		input.Languages = append(input.Languages, language.Name)
	}

	tree, response, err := client.Git.GetTree(ctx, repo.GetOwner().GetLogin(), repo.GetName(), branch, true)
	if err != nil {
		var apiError *github.ErrorResponse
		if response != nil && response.StatusCode == http.StatusConflict &&
			errors.As(err, &apiError) && apiError.Message == "Git Repository is empty." {
			result := hints.ApplyRepository(hints.Result{Type: "unknown", Evidence: []string{"empty repository"}}, input)
			result = hints.ApplyUnknownNameFallbacks(result, input.Name)
			return Analysis{Type: result.Type, Evidence: result.Evidence}, nil
		}
		return Analysis{}, fmt.Errorf("get file tree for %s: %w", repo.GetFullName(), err)
	}
	if tree == nil || tree.GetTruncated() {
		return Analysis{}, fmt.Errorf("incomplete file tree for %s", repo.GetFullName())
	}

	fetchFileContent := func(name string) ([]byte, error) {
		file, _, _, err := client.Repositories.GetContents(ctx, repo.GetOwner().GetLogin(), repo.GetName(), name,
			&github.RepositoryContentGetOptions{Ref: branch})
		if err != nil {
			return nil, fmt.Errorf("get %s for %s: %w", name, repo.GetFullName(), err)
		}
		if file == nil {
			return nil, fmt.Errorf("%s in %s is not a file", name, repo.GetFullName())
		}
		content, err := file.GetContent()
		if err != nil {
			return nil, fmt.Errorf("read %s in %s: %w", name, repo.GetFullName(), err)
		}
		return []byte(content), nil
	}

	analysis, err := analyseTree(tree.Entries, fetchFileContent)
	if err != nil {
		return Analysis{}, fmt.Errorf("analyse %s: %w", repo.GetFullName(), err)
	}

	input.Entries = tree.Entries
	result := hints.ApplyRepository(hints.Result{Type: analysis.Type, Evidence: analysis.Evidence}, input)
	if result.Type != "unknown" && result.Type != "application" && result.Type != "library" {
		return Analysis{Type: result.Type, Evidence: result.Evidence}, nil
	}

	readme, response, err := client.Repositories.GetReadme(ctx, repo.GetOwner().GetLogin(), repo.GetName(),
		&github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			result = hints.Finalise(result, input, "", "")
			return Analysis{Type: result.Type, Evidence: result.Evidence}, nil
		}
		return Analysis{}, fmt.Errorf("get README for %s: %w", repo.GetFullName(), err)
	}
	if readme == nil {
		return Analysis{}, fmt.Errorf("no README content returned for %s", repo.GetFullName())
	}

	content, err := readme.GetContent()
	if err != nil {
		return Analysis{}, fmt.Errorf("read README for %s: %w", repo.GetFullName(), err)
	}

	source := readme.GetPath()
	if source == "" {
		source = "primary README"
	}

	result = hints.Finalise(result, input, source, content)
	return Analysis{Type: result.Type, Evidence: result.Evidence}, nil
}
