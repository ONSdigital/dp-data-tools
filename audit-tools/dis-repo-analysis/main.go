package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/auth"
	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/teams"
)

func main() {
	withEvidence := flag.Bool("with-evidence", false, "include classification evidence in the CSV report")
	flag.Parse()
	ctx := context.Background()

	client, err := auth.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	codeOwner := teams.ChooseTeam()
	if strings.EqualFold(strings.TrimSpace(codeOwner), "q") {
		return
	}
	if codeOwner == "" {
		log.Fatal("no code owner supplied")
	}

	if err := runAnalysis(ctx, client, codeOwner, "repo-analysis.csv", *withEvidence, log.Default()); err != nil {
		log.Fatal(err)
	}
}
