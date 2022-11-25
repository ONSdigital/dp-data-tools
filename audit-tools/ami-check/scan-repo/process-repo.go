package main

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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-billy.v4/util"
)

const (
	tmpDir        = "../tmp"
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

type AmiOccurences struct {
	Filename   string `json:"Filename"`
	Line       string `json:"Line"`
	CommitHash string `json:"CommitHash"`
	RepoName   string `json:"RepoName"`
}

type AmiNameAndData struct {
	Name          string          `json:"Name"`
	ImageId       string          `json:"ImageId"`
	CreationDate  string          `json:"CreationDate"`
	ConvertedDate time.Time       `json:"ConvertedDate"`
	Status        amiStatus       `json:"Status"`
	Occurences    []AmiOccurences `json:"Occurences"`
}

var AllImageInfo []AmiNameAndData

func (element *AmiNameAndData) AddItem(occurence AmiOccurences) {
	element.Occurences = append(element.Occurences, occurence)
}

func main() {
	// read in the ami id's, creation date and name
	amiDataFile, err := os.Open(resultsDir + amiIdFileName)
	check(err)
	defer func() {
		cerr := amiDataFile.Close()
		if cerr != nil {
			fmt.Printf("problem closing: %s : %v\n", resultsDir+amiIdFileName, cerr)
		}
	}()

	// first read through file to get amiId, creation date and name
	amiDataScan := bufio.NewScanner(amiDataFile)

	var totalAmis int

	// for each event line extract container name and buld up a map of individual container names
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
			//!!! may need to stop here as we will need a good creation date for later processing.
		} else {
			info.ConvertedDate = f
		}

		AllImageInfo = append(AllImageInfo, info)
		totalAmis++
	}
	err = amiDataScan.Err()
	check(err)

	//!!! fix logging of errors.

	// pass into gitlog, the oldest creation date to limit how far back it looks to this date minus 1 month (just to be sure)
	gitLog("dp-setup", AllImageInfo[totalAmis-1].CreationDate)
	// !!! then do processing of all commits for returned commit log for the list of ami's
	// NOPE, save to results dir by repo-name and wether we are on main or awsb ...

	//!!! check what other(s) repo's to process	gitLog("dp-ci")
	// !!! then do processing of all commits for returned commit log for the list of ami's
	// NOPE, save to results dir by repo-name and wether we are on main or awsb ...

	// !!! a 3rd app will either merge the results and process alll into some sort of final result
	// ... or process each of the results at a time and create some final result ...
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func displayAndSave(resultsFile *os.File, line string) {
	fmt.Printf("%s", line)
	_, err := fmt.Fprint(resultsFile, line)
	check(err)
}

func TemporalDir() (path string, clean func()) {
	fs := osfs.New(os.TempDir())
	path, err := util.TempDir(fs, "", "")
	if err != nil {
		panic(err)
	}

	return fs.Join(fs.Root(), path), func() {
		util.RemoveAll(fs, path)
	}
}

// !!! clean all of this function up ...
func gitLog(repoName string, oldestAmiCreationDate string) {
	// Clones the given repository, creating the remote, the local branches
	// and fetching the objects, everything in memory:
	/* fullRepoURL := "https://github.com/ONSdigital/" + repoName
	Info("git clone:")
	Info(fullRepoURL)
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: fullRepoURL,
	})
	CheckIfError(err)*/

	//dir, clean := TemporalDir()
	//defer clean()

	// read 'repoName' into in memory structure for SUPER FAST processing later on
	// (we will need to access and manipulate the in memory structure a lot, so speed will be very important)
	//	r, err := git.PlainClone(dir, false, &git.CloneOptions{
	//		URL:        "../../../../" + repoName,
	//		RemoteName: "test",
	//	})

	// Get the working directory for the repository
	//	w, err := r.Worktree()
	//	CheckIfError(err)

	directory := "../../../../" + repoName
	// Opens an already existing repository.
	r, err := git.PlainOpen(directory)
	CheckIfError(err)

	//	w, err := r.Worktree()
	//	CheckIfError(err)

	// Gets the HEAD history from HEAD, just like this command:
	Info("git log")
	//	Info(directory)

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)

	since := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC) //initialise to 2010

	f, ferr := time.Parse(time.RFC3339, oldestAmiCreationDate) // time format with nanoseconds
	if ferr != nil {
		fmt.Printf("error in oldestAmiCreationDate: %v\n", ferr)
	} else {
		since = f
	}

	// subtract a month from start time (just to be sure)
	since = since.AddDate(0, -1, 0)

	// ... retrieves the commit history
	until := time.Now()
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
	CheckIfError(err)

	// !!! have a struct for the commit to store the below +, expanding these: Author Signature
	// to contain the time ...
	/*
			   commit 19b2a81203a35e717b316fd9c05edef0d2f73e24
		   Author: red <red54321@outlook.com>
		   Date:   Thu Nov 24 13:39:20 2022 +0000

		       Merge branch 'release/1.145.0'
	*/

	var total int
	type commitInfo struct {
		commitHash string
		commitDate time.Time
	}
	var commitList []commitInfo

	// ... just iterates over the commits, printing it
	err = cIter.ForEach(func(c *object.Commit) error {
		// fmt.Println(c)
		total++
		var cInfo commitInfo
		cInfo.commitHash = c.Hash.String()
		cInfo.commitDate = c.Author.When
		commitList = append(commitList, cInfo)
		return nil
	})
	CheckIfError(err)

	// the above does not include HEAD, so manually add:
	total++
	var cInfo commitInfo
	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)
	cInfo.commitHash = commit.Hash.String()
	cInfo.commitDate = commit.Author.When
	commitList = append(commitList, cInfo)

	fmt.Printf("Number of logs found in %s, is: %d\n", repoName, total)

	/*	fmt.Printf("\nShowing last 10 hashes:\n")
		for i, comm := range commitList {
			if i >= total-10 {
				fmt.Printf("%v, %v\n", comm.commitHash, comm.commitDate)
			}
		}*/

	/*
	   !!! try to get some sort of diff between commits working, such that th ediff can be searched for ami id's

	   	... may need to use worktree or tree's somehow ...

	   var fromCommit git.Commit

	   fromCommit.Hash = &hashes[total-2]

	   var toCommit git.Commit

	   toCommit.Hash = &hashes[total-1]

	   patch, err := &fromCommit.Patch(&toCommit)
	   fmt.Printf("%v\n", *patch)
	*/

	// ... retrieving the branch being pointed by HEAD
	//ref, err := r.Head()
	//CheckIfError(err)
	// ... retrieving the commit object
	//	commit, err := r.CommitObject(ref.Hash())
	//	CheckIfError(err)

	fmt.Println(commit)
	spew.Dump(commit)

	var gitRepo Repository
	gitRepo.Ctx = context.Background()
	gitRepo.Path = directory

	/*	var repo repo_Repository
		repo.OwnerName = "ONSdigital"
		repo.Name = "dp-setup"
		//!!! init its path ?
	*/

	/* the following is a known commit for checking purposes:

	   commit 19b2a81203a35e717b316fd9c05edef0d2f73e24
	   Author: red <red54321@outlook.com>
	   Date:   Thu Nov 24 13:39:20 2022 +0000

	       Merge branch 'release/1.145.0'

	*/

	/*	commitIdString := "19b2a81203a35e717b316fd9c05edef0d2f73e24"*/

	/*	// Retrieve files affected by the commit
		fileStatus, err := GetCommitFileStatus(gitRepo.Ctx, repo.RepoPath(), commitIdString)
		if err != nil {
			os.Exit(2)
			// TODO
			//		return nil, err
		}
		affectedFileList := make([]*CommitAffectedFiles, 0, len(fileStatus.Added)+len(fileStatus.Removed)+len(fileStatus.Modified))
		for _, files := range [][]string{fileStatus.Added, fileStatus.Removed, fileStatus.Modified} {
			for _, filename := range files {
				affectedFileList = append(affectedFileList, &CommitAffectedFiles{
					Filename: filename,
				})
			}
		}
		spew.Dump(affectedFileList)*/

	/*	diff, err := GetDiff(&gitRepo, &DiffOptions{
			AfterCommitID:     commitIdString, //hashesString[total-1],
			MaxLineCharacters: 5000,
			MaxLines:          1000,
			MaxFiles:          1000,
		})
		if err != nil {
			os.Exit(1)
			// TODO
			//return nil, err
		}
		//	fmt.Printf("diff is: %v\n", diff)
		//	spew.Dump(diff)

		for _, diffFile := range diff.Files {
			fmt.Printf("\nName: %s\n\n", diffFile.Name)
			for _, section := range diffFile.Sections {
				for _, line := range section.Lines {
					fmt.Println(line.Content)
				}
			}
		}*/

	count := 20 //!!! temp, trash
	// lets go crazy on all commits
	for i := total - 1; i > 1; i-- {
		diff, err := GetDiff(&gitRepo, &DiffOptions{
			AfterCommitID:     commitList[i].commitHash,
			MaxLineCharacters: 5000,
			MaxLines:          1000,
			MaxFiles:          1000,
		})
		if err != nil {
			os.Exit(1)
			// TODO
			//return nil, err
		}

		// !!! an optimisation:
		// !!! loop for all amiId's
		//!!! amiId creation date minus 1 month is greater than the commit date,
		// then remove it from the list of amiId's to search for in each line

		fmt.Printf("\ni is: %d    Date: %v\n", i, commitList[i].commitDate)
		for _, diffFile := range diff.Files {
			fmt.Printf("Name: %s\n", diffFile.Name)
			for _, section := range diffFile.Sections {
				for _, line := range section.Lines {
					if line.Content[0] == '+' || line.Content[0] == '-' {
						// check changed lines
						for imageIndex, ami := range AllImageInfo {
							// look for ami id's in line
							if strings.Contains(line.Content, ami.Name) {
								fmt.Printf("found: %s\n", line.Content)
								// and save info
								var occurence AmiOccurences
								occurence.CommitHash = commitList[i].commitHash
								occurence.Filename = diffFile.Name
								occurence.Line = line.Content
								occurence.RepoName = repoName
								AllImageInfo[imageIndex].AddItem(occurence)
								count++ //!!! temp, trash
							}
						}
					}
				}
			}
		}
		// if count > 10 { //!!! temp, trash
		// 	break
		// }
	}

	// !!! for debug, display all info found ... eventually save this to a file with some ssort of structure to it ?
	// ... and then a 3rd app can analyse, as this app takes a long time to run and figuring out analysis may
	// take many iterations
	for _, ami := range AllImageInfo {
		fmt.Printf("ami name: %s\n", ami.Name)
		for _, occ := range ami.Occurences {
			fmt.Printf("%v\n", occ)
		}
	}
	//!!! save struct as a json file, that will make it easy for next app to read in !!!

	file, _ := json.MarshalIndent(AllImageInfo, "", " ")

	err = ioutil.WriteFile("test.json", file, 0644)
	CheckIfError(err)
	// res.Files = affectedFileList
	// res.Stats = &api.CommitStats{
	// 	Total:     diff.TotalAddition + diff.TotalDeletion,
	// 	Additions: diff.TotalAddition,
	// 	Deletions: diff.TotalDeletion,
	// }
}

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	fmt.Printf("\x1b[34;1m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// !!! look thru:
// ~/go/pkg/mod/github.com/go-git

/*
in there see:
worktree_commit_test.go
	function: TestCommitTreeSort
		... what can be written into: TemporalFilesystem()

also, see:
worktree_test.go
	function: TestPullFastForward
		... or do i use TemporalDir()

	function: TestPullAdd()
		where: &CloneOptions{
				 URL: filepath.Join(path, ".git"),
				}


GetDiff( in gitea's services/gitdiff/gitdiff.go   ... copy and adapt this function ?
	... and then how this is called from routers/web/repo/commit.go

	the simplest call to it is (which gets diffs after commit.ID ), from function ToCommit(),
	in modules/convert/git_commit.go:
		diff, err := gitdiff.GetDiff(gitRepo, &gitdiff.DiffOptions{
			AfterCommitID: commit.ID.String(),
		})
		if err != nil {
			return nil, err
		}
		res.Files = affectedFileList
		res.Stats = &api.CommitStats{
			Total:     diff.TotalAddition + diff.TotalDeletion,
			Additions: diff.TotalAddition,
			Deletions: diff.TotalDeletion,
		}

	... also see pull.go use of GetDiff()
*/
//fixtures.
//local

// for code to try and list files and their paths in a commit
// possibly code that uses: NewTreeWalker() ... look at test code.

// !!! go git stuff to read over:

/*

https://ish-ar.io/tutorial-go-git/

https://www.youtube.com/watch?v=tg2yN6ax-xs

https://medium.com/@clm160/tag-example-with-go-git-library-4377a84bbf17

https://pkg.go.dev/github.com/go-git/go-git/v5

https://github.com/go-git/go-git

https://chromium.googlesource.com/external/github.com/src-d/go-git/+/8b0c2116cea2bbcc8d0075e762b887200a1898e1/example_test.go

Also pull the code for 'gitea' and see how that uses go-git lib


also pulumi:

how does this use go-git:

https://github.com/pulumi/pulumi/tree/master/pkg

and look at these links:

https://github.com/search?q=org%3Apulumi+go-git&type=Code


this code looks useful:

https://github.com/pulumi/pulumi/blob/4478bc0f695b17ec68e8d8e92a3202a038999741/sdk/go/auto/git_test.go



*/
