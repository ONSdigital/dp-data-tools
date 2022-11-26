package main

//!!! fix these comments to say what inputs used, what is done, and what is saved

// This utility is a work in progress ...
// currently it gets all of the commit hash's from a repo.
//
// It will utilise the file ../results/all-ami-ids.txt
//
// Ultimately it will try to determine how ONS creaded AMI's are used, by determining:
// 1. what ami id's are in use
// 2. what ami id's were used (and if they are the parent or grandparent to one that is in use
//	 - this bit may be VERY difficult to determine)
// 3. what ami id's have never been used
//
// together with the age of the ami id's grouped by age
// older than two years,
// one to two years old,
// 6 to 12 months old
// 0 to 6 months old

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const (
	tmpDir        = "../tmp"
	resultsDir    = "../results/"
	amiIdFileName = "all-ami-ids.txt"
	repoName      = "dp-setup"
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
	Occurrences   []AmiOccurrences `json:"Occurrences"`
}

var AllImageInfo []AmiNameAndData

func (element *AmiNameAndData) AddItem(occurrence AmiOccurrences) {
	element.Occurrences = append(element.Occurrences, occurrence)
}

func main() {
	// read in json struct of scanned data ...
	file, err := ioutil.ReadFile(tmpDir + repoName + ".json")
	//!!! improve the following
	check(err)

	err = json.Unmarshal([]byte(file), &AllImageInfo)
	//!!! improve the following
	check(err)

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

// !!! clean all of this function up ...
func processScan() {
	fmt.Printf("Number of amis to process is: %d\n", len(AllImageInfo))

	// the commitList is not sorted, so ...
	// sort commits by commit date, oldest last ... so that the search order aligns with true github history.
	/*	sort.Slice(commitList, func(i, j int) bool {
		return commitList[i].commitDate.Before(commitList[j].commitDate)
	})*/

	neverUsedCount := 0
	for _, ami := range AllImageInfo {
		var amiUsed bool
		if ami.Occurrences == nil {
			fmt.Printf("Not used: %s\n", ami.Name)
			neverUsedCount++
		} else {
			fmt.Printf("\nHistory of: %s\n", ami.Name)
			var occurrences []AmiOccurrences

			//!!! have a var for the lastUsedDate  that is initialised to 2010
			// build list of all occurrences and list of all unique filenames
			var fileNames []string
			for _, occurrence := range ami.Occurrences {
				occurrences = append(occurrences, occurrence)
				if contains(fileNames, occurrence.FilePathAndName) == false {
					fileNames = append(fileNames, occurrence.FilePathAndName)
				}
			}
			for _, fileName := range fileNames {
				fmt.Printf("  File: %s\n", fileName)
				var inUse bool
				var firstName bool
				for _, occurrence := range ami.Occurrences {
					if fileName == occurrence.FilePathAndName {
						if firstName == false && occurrence.Line[0] == '-' {
							//!!! save fileName and date in lastUsedDate if the date of this line is newer than what is in lastUsedDate
							// maybe save the whole occurrens struct ?
						}
						if firstName == false && occurrence.Line[0] == '+' {
							inUse = true
						}
						firstName = true
						fmt.Printf("    %s ||| %3d ||| %v\n", occurrence.Line, occurrence.LineIndex, occurrence.CommitDate)
					}
				}
				if inUse {
					fmt.Printf(">>>>>>>> In Use <<<<<<<<\n")
					amiUsed = true
				} else {
					//!!! now we need to go check if the ami Name no longer exists in the 'reponame'+occurrence.FilePathAndName
					//occurrence.RepoName = repoName
					// because the last change may have been to delete it in one place in a file BUT it still exists in
					// one or more places in the file.
					// !!! if the ami name no longer exists in te file, declare its lastUsedDate, etc
					fmt.Printf("-------- Not in Use ----\n")
				}
			}
		}
		if amiUsed {
			fmt.Printf("\n******** AMI used: %s ********\n\n", ami.Name)
		}
	}
	fmt.Printf("Out of: %d ami's, Not used is: %d\n", len(AllImageInfo), neverUsedCount)
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
