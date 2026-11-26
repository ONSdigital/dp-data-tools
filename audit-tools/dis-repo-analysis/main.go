package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/auth"
	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/teams"
	"github.com/google/go-github/v74/github"
)

func main() {
	withEvidence := flag.Bool("with-evidence", false, "include classification evidence in the CSV report")
	flag.Parse()
	ctx := context.Background()

	client, err := auth.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx, client, *withEvidence); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, client *github.Client, withEvidence bool) error {
	codeOwner, err := teams.ChooseTeam()
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(codeOwner), "q") {
		return nil
	}
	codeOwner, err = teams.ValidateTeam(ctx, client, codeOwner)
	if err != nil {
		return err
	}
	return runAnalysis(ctx, client, codeOwner, "repo-analysis.csv", withEvidence, log.Default())
}
