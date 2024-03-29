package main

// This utility is a work in progress ...
// currently it gets all of the commit hash's from a repo.
//
// It will utilise the file ../results/all-ami-ids.txt
//
// It writes out a file in ../tmp/dp-setup.json
// that contains all of the diff info for all relevant commits.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	tmpDir        = "../tmp/"
	resultsDir    = "../results/"
	amiIdFileName = "all-ami-ids.txt"
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

type commitInfo struct {
	commitHash string
	commitDate time.Time
}

var AllImageInfo []AmiNameAndData

func (element *AmiNameAndData) AddItem(occurrence AmiOccurrences) {
	element.Occurrences = append(element.Occurrences, occurrence)
}

func main() {
	// read in the ami id's, creation date and name
	amiDataFile, err := os.Open(resultsDir + amiIdFileName)
	if err != nil {
		fmt.Printf("error opening %s file. Error %v\n", resultsDir+amiIdFileName, err)
		os.Exit(1)
	}

	defer func() {
		cerr := amiDataFile.Close()
		if cerr != nil {
			fmt.Printf("problem closing: %s : %v\n", resultsDir+amiIdFileName, cerr)
		}
	}()

	// first read through file to get amiId, creation date and name
	amiDataScan := bufio.NewScanner(amiDataFile)

	var totalAmis int

	// process each line of ami info
	for amiDataScan.Scan() {
		fields := strings.Fields(amiDataScan.Text())

		if len(fields) != 3 {
			fmt.Printf("error in %s file. expected 'ami ID', 'creation date', 'name', but got %v\n", resultsDir+amiIdFileName, fields)
			os.Exit(1)
		}
		var info AmiNameAndData

		info.ImageId = strings.TrimRight(fields[0], ",")      // remove comma separator
		info.CreationDate = strings.TrimRight(fields[1], ",") // remove comma separator
		info.Name = fields[2]

		f, ferr := time.Parse(time.RFC3339, info.CreationDate) // time format with nanoseconds
		if ferr != nil {
			fmt.Printf("error in info.CreationDate: %v\n", ferr)
			os.Exit(1)
		} else {
			// subtract a day from start time , for later comparisons to function
			f = f.AddDate(0, 0, -1)

			info.ConvertedDate = f
		}

		AllImageInfo = append(AllImageInfo, info)
		totalAmis++
	}
	err = amiDataScan.Err()
	if err != nil {
		fmt.Printf("problem scannnig: %s : %v\n", resultsDir+amiIdFileName, err)
		os.Exit(1)
	}

	start := time.Now()
	// pass into gitLogDiffProcess, the oldest creation date to limit how far back it looks to this date
	gitLogDiffProcess("dp-setup", AllImageInfo[0].CreationDate)
	elapsed := time.Since(start)
	fmt.Printf("gitLogDiffProcess took: %s", elapsed)
}

func gitLogDiffProcess(repoName string, oldestAmiCreationDate string) {
	// construct the path to the repo to be processed
	directory := "../../../../" + repoName
	// Opens an already existing repository.
	r, err := git.PlainOpen(directory)
	CheckIfError(err, fmt.Sprintf("Can't open git directory: %s", directory))

	// Retrieves the branch pointed to by HEAD
	ref, err := r.Head()
	CheckIfError(err, "Problem getting HEAD for commits")

	since := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC) //initialise to 2010

	f, ferr := time.Parse(time.RFC3339, oldestAmiCreationDate) // time format with nanoseconds
	if ferr != nil {
		fmt.Printf("error in oldestAmiCreationDate: %v\n", ferr)
	} else {
		since = f
	}

	// subtract a month from start time to ensure we capture all ami usage (just to be sure)
	since = since.AddDate(0, -1, 0)

	// Retrieve the commit history
	until := time.Now()
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
	CheckIfError(err, "Problem getting iterator for commits")

	var totalCommits int

	var commitList []commitInfo

	// Iterate over the commits, saving commit hash and commit date
	err = cIter.ForEach(func(c *object.Commit) error {
		totalCommits++
		var cInfo commitInfo
		cInfo.commitHash = c.Hash.String()
		cInfo.commitDate = c.Author.When
		commitList = append(commitList, cInfo)
		return nil
	})
	CheckIfError(err, "Problem iterating over commits")

	fmt.Printf("Number of logs found in %s, is: %d\n", repoName, totalCommits)

	// the commitList is not sorted, so ...
	// sort commits by commit date, oldest last ... so that the search order aligns with true github history.
	sort.Slice(commitList, func(i, j int) bool {
		return commitList[i].commitDate.Before(commitList[j].commitDate)
	})

	var gitRepo Repository
	gitRepo.Ctx = context.Background()
	gitRepo.Path = directory

	// Process all of the commits, looking to see where ami.Name is used
	for i := totalCommits - 1; i > 1; i-- {
		diff, err := GetDiff(&gitRepo, &DiffOptions{
			AfterCommitID:     commitList[i].commitHash,
			MaxLineCharacters: 5000,
			MaxLines:          4000,
			MaxFiles:          1000,
		})
		if err != nil {
			os.Exit(1)
			// TODO
			//return nil, err
		}

		processDiff(diff, commitList, i, repoName)
	}

	// save struct as a json file, that will make it easy for next app to read in
	file, _ := json.MarshalIndent(AllImageInfo, "", " ")

	err = os.WriteFile(tmpDir+repoName+".json", file, 0644)
	CheckIfError(err, fmt.Sprint("Can't write to file: %s", tmpDir+repoName+".json"))
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error, reason string) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("reason: %s, error: %s", reason, err))
	os.Exit(1)
}

func processDiff(diff *Diff, commitList []commitInfo, i int, repoName string) {
	fmt.Printf("\ni is: %d    Date: %v  Hash: %v\n", i, commitList[i].commitDate, commitList[i].commitHash)
	for _, diffFile := range diff.Files {
		fmt.Printf("Name: %s\n", diffFile.Name)
		for imageIndex, ami := range AllImageInfo {
			// look for ami id's in line
			if commitList[i].commitDate.After(ami.ConvertedDate) {
				for sectionIndex, section := range diffFile.Sections {
					for lineIndex, line := range section.Lines {
						if line.Content[0] == '+' || line.Content[0] == '-' {
							// check changed lines
							if strings.Contains(line.Content, ami.Name) {
								fmt.Printf("found: %s\n", line.Content)
								// and save info in time based ordering, effectively in the order of the commit date - that is oldest last,
								// so that next app can pick out the first occurences of lines with ami id's in them easily on
								// a file by file basis.
								var occurrence AmiOccurrences
								occurrence.CommitHash = commitList[i].commitHash
								occurrence.CommitDate = commitList[i].commitDate
								occurrence.FilePathAndName = diffFile.Name
								occurrence.Line = line.Content
								occurrence.LineIndex = lineIndex
								occurrence.RepoName = repoName
								occurrence.SectionIndex = sectionIndex
								AllImageInfo[imageIndex].AddItem(occurrence)
							}
						}
					}
				}
			}
		}
	}
}
