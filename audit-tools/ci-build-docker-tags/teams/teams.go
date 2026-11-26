package teams

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var teams []string = []string{
	"@ONSdigital/dissemination-open-sauce",
	"@ONSdigital/dissemination-data-dissemination",
	"@ONSdigital/dissemination-data-pipelines",
	"@ONSdigital/dissemination-platform",
}

func ChooseTeam() (string, error) {
	fmt.Println("Choose an ONSdigital team, enter @ONSdigital/team-slug, or q to quit")
	for i, team := range teams {
		fmt.Printf(" %d - %s\n", i+1, team)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("-> ")
	text, err := reader.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || len(text) == 0) {
		return "", fmt.Errorf("read team selection: %w", err)
	}
	text = strings.TrimSpace(text)
	if i, err := strconv.Atoi(text); err == nil {
		if i < 1 || i > len(teams) {
			return "", fmt.Errorf("choose a team number between 1 and %d or enter @ONSdigital/team-slug", len(teams))
		}
		return teams[i-1], nil
	}
	return text, nil
}
