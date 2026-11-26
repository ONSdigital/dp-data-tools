package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/repos"
	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/analysis"
	"github.com/ONSdigital/dp-data-tools/audit-tools/dis-repo-analysis/output"
	"github.com/google/go-github/v74/github"
)

// runAnalysis performs a comprehensive analysis of repositories owned by the specified code owner, generating CSV and Markdown reports.
func runAnalysis(ctx context.Context, client *github.Client, codeOwner, destination string, withEvidence bool, logger *log.Logger) error {
	if withEvidence {
		logger.Println("Starting repository analysis with evidence")
	} else {
		logger.Println("Starting repository analysis")
	}

	logger.Println("Searching repositories")
	repositories := repos.SearchRepos(ctx, client, codeOwner)

	logger.Println("Completed searching repositories")
	repoNames := make([]string, 0, len(repositories))
	for name := range repositories {
		repoNames = append(repoNames, name)
	}

	slices.Sort(repoNames)
	records := make([]output.Record, len(repoNames))
	if err := runRepositoryStage(logger, "repository visibility", repoNames, func(i int, repoName string) error {
		repo := repositories[repoName]
		details, _, err := client.Repositories.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName())
		if err != nil {
			return fmt.Errorf("get repository details for %s: %w", repoName, err)
		}
		repositories[repoName] = details
		visibility, err := repos.RepoVisibility(details)
		if err != nil {
			return err
		}
		records[i] = output.Record{Repository: repoName, Visibility: visibility}
		return nil
	}); err != nil {
		return err
	}

	if err := runRepositoryStage(logger, "repository code owners", repoNames, func(i int, repoName string) error {
		repo := repositories[repoName]
		codeOwners, err := repos.GetCodeOwners(ctx, client, repo)
		if err != nil {
			return err
		}
		records[i].CodeOwners = codeOwners
		return nil
	}); err != nil {
		return err
	}

	if err := runRepositoryStage(logger, "repository languages", repoNames, func(i int, repoName string) error {
		repo := repositories[repoName]
		languages, err := analysis.RepoLanguages(ctx, client, repo)
		if err != nil {
			return err
		}
		records[i].Languages = languages
		return nil
	}); err != nil {
		return err
	}

	if err := runRepositoryStage(logger, "repository types", repoNames, func(i int, repoName string) error {
		repo := repositories[repoName]
		analysisResult, err := analysis.AnalyseRepo(ctx, client, repo, records[i].Languages)
		if err != nil {
			return err
		}
		records[i].Analysis = analysisResult
		return nil
	}); err != nil {
		return err
	}

	logger.Println("Writing CSV report")
	if err := output.WriteCSV(destination, records, withEvidence); err != nil {
		return err
	}
	logger.Printf("Completed writing CSV report: %s", destination)

	markdownDestination := strings.TrimSuffix(destination, filepath.Ext(destination)) + ".md"
	logger.Println("Writing Markdown report")
	if err := output.WriteMarkdown(markdownDestination, codeOwner, records); err != nil {
		return err
	}

	logger.Printf("Completed writing Markdown report: %s", markdownDestination)
	logger.Println("Completed repository analysis")
	return nil
}
