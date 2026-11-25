package teams

import (
	"bufio"
	"fmt"
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

func ChooseTeam() string {
	fmt.Println("Choose a code-owner, enter one of your choice, or q to quit")
	for i, team := range teams {
		fmt.Printf(" %d - %s\n", i+1, team)
	}
	reader := bufio.NewReader(os.Stdin)
	chosenTeam := ""
	for chosenTeam == "" {
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
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
