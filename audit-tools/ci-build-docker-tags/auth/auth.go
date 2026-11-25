package auth

import (
	"fmt"
	"os"

	"github.com/google/go-github/v74/github"
)

func NewClient() (*github.Client, error) {
	token := os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("no auth token supplied")
	}

	return github.NewClient(nil).WithAuthToken(token), nil
}
