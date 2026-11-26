package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/auth"
	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/repos"
	"github.com/ONSdigital/dp-data-tools/audit-tools/ci-build-docker-tags/teams"
	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v74/github"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	ctx := context.Background()

	client, err := auth.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx, client); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, client *github.Client) error {
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

	repositories := repos.SearchRepos(ctx, client, codeOwner)

	fmt.Print("Getting repo image details")
	results := make([]result, 0, len(repositories))
	for repoName, repo := range repositories {
		buildImage, buildTag := getBuildImageTag(ctx, client, repo)
		runImage, runTag := getRunImageTag(ctx, client, repo)
		results = append(results, result{
			name:       repoName,
			buildImage: image{image: buildImage, tag: buildTag},
			runImage:   image{image: runImage, tag: runTag},
		})
		fmt.Print(".")
	}
	fmt.Println()
	outputResults(results)
	return nil
}

type result struct {
	name       string
	buildImage image
	runImage   image
}

type image struct {
	image string
	tag   string
}

func outputResults(results []result) {

	nameLabel := "Repository"
	buildImageLabel := "Build image name"
	buildTagLabel := "Build image tag"
	runImageLabel := "Running image name"
	runTagLabel := "Running image tag"

	maxNameLen := len(nameLabel)
	maxBuildImageLen := len(buildImageLabel)
	maxBuildTagLen := len(buildTagLabel)
	maxRunImageLen := len(runImageLabel)
	maxRunTagLen := len(runTagLabel)

	for _, result := range results {
		if len(result.name) > maxNameLen {
			maxNameLen = len(result.name)
		}
		if len(result.buildImage.image) > maxBuildImageLen {
			maxBuildImageLen = len(result.buildImage.image)
		}
		if len(result.buildImage.tag) > maxBuildTagLen {
			maxBuildTagLen = len(result.buildImage.tag)
		}
		if len(result.runImage.image) > maxRunImageLen {
			maxRunImageLen = len(result.runImage.image)
		}
		if len(result.runImage.tag) > maxRunTagLen {
			maxRunTagLen = len(result.runImage.tag)
		}
	}
	cmpFunc := func(a, b result) int {
		return cmp.Or(
			cmp.Compare(a.buildImage.image, b.buildImage.image),
			-cmp.Compare(a.buildImage.tag, b.buildImage.tag),
			cmp.Compare(a.name, b.name),
		)
	}
	slices.SortFunc(results, cmpFunc)

	fmtString := fmt.Sprintf("%%%ds %%%ds %%%ds %%%ds %%%ds\n", maxNameLen, maxBuildImageLen, maxBuildTagLen, maxRunImageLen, maxRunTagLen)

	if len(results) > 0 {
		fmt.Printf(fmtString, nameLabel, buildImageLabel, buildTagLabel, runImageLabel, runTagLabel)
	}

	for _, result := range results {
		fmt.Printf(fmtString, result.name, result.buildImage.image, result.buildImage.tag, result.runImage.image, result.runImage.tag)
	}
}

func getBuildImageTag(ctx context.Context, client *github.Client, repo *github.Repository) (image, tag string) {
	contents, _, r, err := client.Repositories.GetContents(ctx, repo.Owner.GetLogin(), repo.GetName(), "ci/build.yml", nil)
	if err != nil {
		return
	}
	if r.Response.StatusCode == http.StatusNotFound {
		return
	}
	data, err := contents.GetContent()
	if err != nil {
		log.Fatal(err)
	}

	var ciBuild struct {
		Platform string `yaml:"platform"`
		ImRes    struct {
			Source struct {
				Repository string `yaml:"repository"`
				Tag        string `yaml:"tag"`
			} `yaml:"source"`
		} `yaml:"image_resource"`
	}
	err = yaml.Unmarshal([]byte(data), &ciBuild)
	if err != nil {
		log.Fatal(err)
	}
	image = ciBuild.ImRes.Source.Repository
	tag = ciBuild.ImRes.Source.Tag

	return image, tag
}

func getRunImageTag(ctx context.Context, client *github.Client, repo *github.Repository) (image, tag string) {
	contents, _, r, err := client.Repositories.GetContents(ctx, repo.Owner.GetLogin(), repo.GetName(), "Dockerfile.concourse", nil)
	if err != nil {
		return
	}
	if r.Response.StatusCode == http.StatusNotFound {
		return
	}
	data, err := contents.GetContent()
	if err != nil {
		log.Fatal(err)
	}

	reader := strings.NewReader(data)

	node, err := parser.Parse(reader)
	if err != nil {
		log.Fatal(err)
	}

	for _, child := range node.AST.Children {
		if child.Value == "FROM" {
			baseImage := strings.Split(child.Next.Value, ":")
			if len(baseImage) == 2 {
				return baseImage[0], baseImage[1]
			}
		}
	}

	return "", ""
}
