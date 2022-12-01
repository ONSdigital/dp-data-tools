package main

// This app uses the file ../tmp/<repoName>.json
//
// It will determines how ONS creaded AMI's are used, by:
// 1. what ami id's are in use
// 2. what ami id's are no longer used and the last date used.
// 3. what ami id's have never been used
//
// together with the age of the ami id's grouped by age
// older than two years,
// one to two years old,
// 6 to 12 months old
// 0 to 6 months old

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const (
	tmpDir     = "../tmp/"
	resultsDir = "../results/"
	resultFile = "ami-used-status.txt"
	repoName   = "dp-setup"
)

// amiStatus the state of an AMI
type amiStatus int

const (
	AmiUnknown      amiStatus = iota // 0
	AmiInUse                         // 1
	AmiNeverUsed                     // 2
	AmiNoLongerUsed                  // 3
)

type AmiOccurrences struct {
	FilePathAndName string    `json:"FilePathAndName"`
	Line            string    `json:"Line"`
	LineIndex       int       `json:"LineIndex"`
	SectionIndex    int       `json:"SectionIndex"`
	CommitHash      string    `json:"CommitHash"`
	CommitDate      time.Time `json:"CommitDate"`
	RepoName        string    `json:"RepoName"`
}

type AmiNameAndData struct {
	Name          string           `json:"Name"`
	ImageId       string           `json:"ImageId"`
	CreationDate  string           `json:"CreationDate"`
	ConvertedDate time.Time        `json:"ConvertedDate"`
	Status        amiStatus        `json:"Status"`
	LastUsedDate  time.Time        `json:"LastUsedDate"`
	LastUsedFile  string           `json:"LastUsedFile"`
	Occurrences   []AmiOccurrences `json:"Occurrences"`
}

// AllImageInfo is read in from a file and it is pre sorted by oldest ami first
var AllImageInfo []AmiNameAndData

func (element *AmiNameAndData) AddItem(occurrence AmiOccurrences) {
	element.Occurrences = append(element.Occurrences, occurrence)
}

func main() {
	// read in json struct of scanned data ...
	// Open our jsonFile
	jsonFile, err := os.Open(tmpDir + repoName + ".json")
	if err != nil {
		log.Fatalf(fmt.Sprintf("Can't open file: %s", tmpDir+repoName+".json"), err)
	}
	defer jsonFile.Close()

	defer func() {
		cerr := jsonFile.Close()
		if cerr != nil {
			fmt.Printf("problem closing: %s : %v\n", tmpDir+repoName+".json", cerr)
		}
	}()

	file, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Can't read all of file: %s", tmpDir+repoName+".json"), err)
	}

	err = json.Unmarshal([]byte(file), &AllImageInfo)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Can't unmarshal file: %s", tmpDir+repoName+".json"), err)
	}

	start := time.Now()

	processScan()

	elapsed := time.Since(start)
	fmt.Printf("gitLogDiffProcess took: %s\n", elapsed)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func processScan() {
	fmt.Printf("Number of amis to process is: %d\n", len(AllImageInfo))

	neverUsedCount := 0
	for a, ami := range AllImageInfo {
		var amiUsed bool
		amiPluses := 0
		amiMinuses := 0
		if ami.Occurrences == nil {
			fmt.Printf("Never used  : %50s   %v\n", ami.Name, ami.CreationDate)
			AllImageInfo[a].Status = AmiNeverUsed
			neverUsedCount++
		} else {
			fmt.Printf("\nHistory of: %50s   %v\n", ami.Name, ami.CreationDate)

			// Initialize to date beyond any possible oldest
			AllImageInfo[a].LastUsedDate = time.Date(2010, 1, 1, 1, 1, 1, 0, time.Local)

			// build list of all unique filenames that this ami is used in
			var fileNames []string
			for _, occurrence := range ami.Occurrences {
				// is occurrence.FilePathAndName in the list?
				if !contains(fileNames, occurrence.FilePathAndName) {
					// no, so add it to list
					fileNames = append(fileNames, occurrence.FilePathAndName)
				}
			}
			for _, fileName := range fileNames {
				fmt.Printf("  File: %s\n", fileName)
				var inUse bool
				var firstName bool
				for _, occurrence := range ami.Occurrences {
					if fileName == occurrence.FilePathAndName {
						if occurrence.Line[0] == '-' {
							amiMinuses++
							if !firstName {
								// save fileName and date in lastUsedDate if the date of this line is newer than what is in lastUsedDate
								if occurrence.CommitDate.After(AllImageInfo[a].LastUsedDate) {
									AllImageInfo[a].LastUsedDate = occurrence.CommitDate
									AllImageInfo[a].LastUsedFile = fileName
								}
							}
						}
						if occurrence.Line[0] == '+' {
							amiPluses++
							if !firstName {
								inUse = true
							}
						}
						firstName = true
						fmt.Printf("    %s ||| %3d ||| %v\n", occurrence.Line, occurrence.LineIndex, occurrence.CommitDate)
					}
				}
				if inUse {
					fmt.Printf(">>>>>>>> In Use <<<<<<<<\n")
					amiUsed = true
				} else {
					fmt.Printf("-------- Not in Use ----\n")
				}
			}

			if amiUsed {
				fmt.Printf("\n******** AMI used: %s ********\n\n", ami.Name)
				AllImageInfo[a].Status = AmiInUse
			} else {
				found := false
				if amiPluses != amiMinuses {
					// Hmm something may be a little adrift, so:
					// Now we need to go check if the ami Name no longer exists in the list of 'fileNames'
					// because the last change may have been to delete it in one place in a file BUT it still exists in
					// one or more other places in one or more files.
					// ... just to be sure !
					directory := "../../../../" + repoName
					for _, fileName := range fileNames {
						pathAndName := directory + "/" + fileName
						fileHandle, err := os.Open(pathAndName)
						if err == nil {
							// The file still exists, so check if ami.Name is in it
							defer fileHandle.Close()

							defer func() {
								cerr := fileHandle.Close()
								if cerr != nil {
									fmt.Printf("problem closing: %s : %v\n", pathAndName, cerr)
								}
							}()

							scanner := bufio.NewScanner(fileHandle)
							// optionally, resize scanner's capacity for lines over 64K, see next example
							for scanner.Scan() {
								if strings.Contains(scanner.Text(), ami.Name) {
									found = true
									break
								}
								fmt.Println(scanner.Text())
							}

							if err := scanner.Err(); err != nil {
								log.Fatal(err)
							}
						}
						if found {
							break
						}
					}
				}
				if found {
					fmt.Printf("\n******** After deeper file search AMI found to be used: %s ********\n\n", ami.Name)
					AllImageInfo[a].Status = AmiInUse
				} else {
					fmt.Printf("\n******** AMI No Longer used: %s ********  Last Used: %v\n\n", ami.Name, AllImageInfo[a].LastUsedDate)
					AllImageInfo[a].Status = AmiNoLongerUsed
				}
			}
			fmt.Printf("amiPluses: %d   amiMinuses: %d\n", amiPluses, amiMinuses)
		}
	}
	fmt.Printf("Out of: %d ami's, Not used is: %d\n", len(AllImageInfo), neverUsedCount)

	// Output a results file similar to what is done in the first script and tag on the end of each
	// line the ami's Status (in english) and if applicable the last used date.
	resultsFile, err := os.Create(resultsDir + resultFile)
	check(err)
	defer resultsFile.Close()

	var printedSixMonths bool
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	var printedTwelveMonths bool
	twelveMonthsAgo := time.Now().AddDate(0, -12, 0)

	var printedTwentyFourMonths bool
	twentyFourMonthsAgo := time.Now().AddDate(0, -24, 0)

	fmt.Printf("\n\n")

	displayAndSave(resultsFile, "AMI used status:\n")
	displayAndSave(resultsFile, fmt.Sprintf("%-50s, %-25s, %-25s,   Status\n", "Name", "ImageId", "CreationDate"))
	for _, ami := range AllImageInfo {
		if !printedTwentyFourMonths && (ami.ConvertedDate).After(twentyFourMonthsAgo) {
			printedTwentyFourMonths = true
			displayAndSave(resultsFile, "Created 24 to 12 months ago:\n")
		}
		if !printedTwelveMonths && (ami.ConvertedDate).After(twelveMonthsAgo) {
			printedTwelveMonths = true
			displayAndSave(resultsFile, "Created 12 to 6 months ago:\n")
		}
		if !printedSixMonths && (ami.ConvertedDate).After(sixMonthsAgo) {
			printedSixMonths = true
			displayAndSave(resultsFile, "Created in last 6 months:\n")
		}
		displayAndSave(resultsFile, fmt.Sprintf("%50s, %25s, %s  -> ", ami.Name, ami.ImageId, ami.CreationDate))
		switch ami.Status {
		case AmiInUse:
			displayAndSave(resultsFile, "In Use\n")
		case AmiNeverUsed:
			displayAndSave(resultsFile, "Never Used\n")
		case AmiNoLongerUsed:
			displayAndSave(resultsFile, fmt.Sprintf("No longer used since: %v  in file:%s\n", ami.LastUsedDate, ami.LastUsedFile))
		default:
		}
	}
	displayAndSave(resultsFile, "\n")
}

func displayAndSave(resultsFile *os.File, line string) {
	fmt.Printf("%s", line)
	_, err := fmt.Fprint(resultsFile, line)
	check(err)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
