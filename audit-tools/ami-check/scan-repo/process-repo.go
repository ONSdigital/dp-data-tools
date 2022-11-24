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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-billy.v4/util"
)

var AllImageIds []string

const (
	tmpDir     = "../tmp"
	resultsDir = "../results"
)

func main() {
	// !!! read in the ami id's

	gitLog("dp-setup")
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

func gitLog(repoName string) {
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

	// ... retrieves the commit history
	since := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)   //!!! fix to 2010
	until := time.Date(2022, 11, 23, 0, 0, 0, 0, time.UTC) //!!! fix to current data and time
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
	var hashes []object.Hash
	var hashesString []string
	// ... just iterates over the commits, printing it
	err = cIter.ForEach(func(c *object.Commit) error {
		// fmt.Println(c)
		total++
		hashes = append(hashes, object.Hash(c.Hash))
		hashesString = append(hashesString, c.Hash.String())
		return nil
	})
	CheckIfError(err)
	fmt.Printf("Number of logs found in %s, is: %d\n", repoName, total)

	fmt.Printf("\nShowing last 10 hashes:\n")
	for i, hash := range hashesString {
		if i >= total-10 {
			fmt.Printf("%v\n", hash)
		}
	}

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
	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)

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

	commitIdString := "19b2a81203a35e717b316fd9c05edef0d2f73e24"

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

	diff, err := GetDiff(&gitRepo, &DiffOptions{
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
	}

	// lets go crazy on all commits
	for i := total - 1; i > 1; i++ {
		diff, err := GetDiff(&gitRepo, &DiffOptions{
			AfterCommitID:     hashesString[i],
			MaxLineCharacters: 5000,
			MaxLines:          1000,
			MaxFiles:          1000,
		})
		if err != nil {
			os.Exit(1)
			// TODO
			//return nil, err
		}

		fmt.Printf("i is: %d\n", i)
		for _, diffFile := range diff.Files {
			fmt.Printf("\nName: %s\n\n", diffFile.Name)
			for _, section := range diffFile.Sections {
				for _, line := range section.Lines {
					fmt.Println(line.Content)
					break // !!! remove
				}
			}
		}
	}

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
