package main

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v74/github"
)

const perPage = 100

// TODO maybe do a search using the api to compute this list?
var teams []string = []string{
	"@ONSdigital/dissemination-open-sauce",
	"@ONSdigital/dissemination-data-dissemination",
	"@ONSdigital/dissemination-data-pipelines",
	"@ONSdigital/dissemination-platform",
}

func main() {
	ctx := context.Background()

	authToken := os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	if authToken == "" {
		log.Fatal("no auth token supplied")
	}

	codeOwner := chooseTeam()
	if strings.ToLower(codeOwner) == "q" {
		os.Exit(0)
	}

	client := github.NewClient(nil).WithAuthToken(authToken)

	repos := searchRepos(ctx, client, codeOwner)

	fmt.Print("Getting repo build image details")
	results := make([]result, 0, len(repos))
	for repoName, repo := range repos {
		image, tag := getBuildImageTag(ctx, client, repo)
		results = append(results, result{repoName, image, tag})
		fmt.Print(".")
	}
	fmt.Println()
	outputResults(results)
}

func chooseTeam() string {
	fmt.Println("Choose a code-owner or enter one of your choice")
	for i, team := range teams {
		fmt.Printf(" %d - %s\n", i+1, team)
	}
	reader := bufio.NewReader(os.Stdin)
	chosenTeam := ""
	for chosenTeam == "" {
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)
		i, err := strconv.Atoi(text)
		if err == nil {
			if i > 0 && i <= len(teams) {
				chosenTeam = teams[i-1]
			}
		} else {
			chosenTeam = text
		}
	}
	return chosenTeam
}

func searchRepos(ctx context.Context, client *github.Client, codeOwner string) map[string]*github.Repository {
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

type result struct {
	name  string
	image string
	tag   string
}

func outputResults(results []result) {
	maxNameLen := 0
	maxImageLen := 0
	maxTagLen := 0
	for _, result := range results {
		if len(result.name) > maxNameLen {
			maxNameLen = len(result.name)
		}
		if len(result.image) > maxImageLen {
			maxImageLen = len(result.image)
		}
		if len(result.tag) > maxTagLen {
			maxTagLen = len(result.tag)
		}
	}
	cmpFunc := func(a, b result) int {
		return cmp.Or(
			cmp.Compare(a.image, b.image),
			-cmp.Compare(a.tag, b.tag),
			cmp.Compare(a.name, b.name),
		)
	}
	slices.SortFunc(results, cmpFunc)

	fmtString := fmt.Sprintf("%%%ds %%%ds %%%ds\n", maxNameLen, maxImageLen, maxTagLen)
	for _, result := range results {
		fmt.Printf(fmtString, result.name, result.image, result.tag)
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
