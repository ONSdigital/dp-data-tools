package teams

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v74/github"
)

var teamIdentifier = regexp.MustCompile(`(?i)^@ONSdigital/([a-z0-9-]+)$`)

// ValidateTeam confirms an ONSdigital team is accessible and returns its canonical identifier.
func ValidateTeam(ctx context.Context, client *github.Client, codeOwner string) (string, error) {
	match := teamIdentifier.FindStringSubmatch(strings.TrimSpace(codeOwner))
	if match == nil {
		return "", fmt.Errorf("enter an ONSdigital team in the form @ONSdigital/team-slug")
	}
	if client == nil {
		return "", fmt.Errorf("GitHub client is required to validate a team")
	}

	team, response, err := client.Teams.GetTeamBySlug(ctx, "ONSdigital", match[1])
	if err != nil {
		if response != nil {
			switch response.StatusCode {
			case http.StatusNotFound:
				return "", fmt.Errorf("team %s not found or not accessible: %w", codeOwner, err)
			case http.StatusUnauthorized:
				return "", fmt.Errorf("authentication failed while validating team %s; check the token: %w", codeOwner, err)
			case http.StatusForbidden:
				return "", fmt.Errorf("cannot access team %s; check token permissions: %w", codeOwner, err)
			}
		}
		return "", fmt.Errorf("validate team %s: %w", codeOwner, err)
	}
	if team == nil || !teamIdentifier.MatchString("@ONSdigital/"+team.GetSlug()) {
		return "", fmt.Errorf("GitHub returned no valid team slug for %s", codeOwner)
	}
	return "@ONSdigital/" + team.GetSlug(), nil
}
