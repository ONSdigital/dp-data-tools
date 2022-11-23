package main

// This utility reads the given file that contains a list of AWS AMI Image info
// (that is created by bash script) and produces a report on the images grouped by age
// older than two years,
// one to two years old,
// 6 to 12 months old
// 0 to 6 months old

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var stagingFiles = [...]string{"staging-amis.json", "sandbox-amis.json", "prod-amis.json"}

// AmiImages was created using:
// https://mholt.github.io/json-to-go/
//
// whereby I copied in the output of running (against Staging):
// aws ec2 describe-images --owner self --output json | jq . >staging-amis.json
// and then modifying the 'CreationDate' to be a string
//
// and then i used: https://json2struct.mervine.net/
// with about 100,000 lines of the output from:
// aws ec2 describe-images --output json | jq . >st-amis.json
// and then merged in additional struct elements.

type AmiImages struct {
	Images []struct {
		Architecture        string `json:"Architecture"`
		CreationDate        string `json:"CreationDate"`
		ImageID             string `json:"ImageId"`
		ImageLocation       string `json:"ImageLocation"`
		ImageType           string `json:"ImageType"`
		Public              bool   `json:"Public"`
		OwnerID             string `json:"OwnerId"`
		PlatformDetails     string `json:"PlatformDetails"`
		UsageOperation      string `json:"UsageOperation"`
		State               string `json:"State"`
		BlockDeviceMappings []struct {
			DeviceName string `json:"DeviceName"`
			Ebs        struct {
				DeleteOnTermination bool   `json:"DeleteOnTermination"`
				Iops                int64  `json:"Iops"`
				SnapshotID          string `json:"SnapshotId"`
				VolumeSize          int64  `json:"VolumeSize"`
				Throughput          int64  `json:"Throughput"`
				VolumeType          string `json:"VolumeType"`
				Encrypted           bool   `json:"Encrypted"`
			} `json:"Ebs,omitempty"`
			VirtualName string `json:"VirtualName,omitempty"`
		} `json:"BlockDeviceMappings"`
		BootMode        string `json:"BootMode"`
		DeprecationTime string `json:"DeprecationTime"`
		Description     string `json:"Description"`
		EnaSupport      bool   `json:"EnaSupport"`
		Hypervisor      string `json:"Hypervisor"`
		ImageOwnerAlias string `json:"ImageOwnerAlias"`
		KernelID        string `json:"KernelId"`
		Name            string `json:"Name"`
		Platform        string `json:"Platform"`
		ProductCodes    []struct {
			ProductCodeID   string `json:"ProductCodeId"`
			ProductCodeType string `json:"ProductCodeType"`
		} `json:"ProductCodes"`
		RootDeviceName  string `json:"RootDeviceName"`
		RootDeviceType  string `json:"RootDeviceType"`
		SriovNetSupport string `json:"SriovNetSupport"`
		TpmSupport      string `json:"TpmSupport"`
		Tags            []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"Tags"`
		VirtualizationType string `json:"VirtualizationType"`
	} `json:"Images"`
}

type AmiNameAndData struct {
	Name          string
	ImageId       string
	CreationDate  string
	ConvertedDate time.Time
}

var AllImageIds []string

func main() {
	// check each manifest file
	for _, jName := range stagingFiles {
		jFile, err := os.ReadFile(jName)
		if err != nil {
			fmt.Printf("Failed reading %s, with error: %v\n", jName, err)
			os.Exit(100)
		}

		amiInfo := AmiImages{}
		err = json.Unmarshal(jFile, &amiInfo)
		if err != nil {
			fmt.Printf("Failed unmarshaling json file %s, with error: %v\n", jName, err)
			os.Exit(101)
		}

		var imageFiles []AmiNameAndData

		for _, image := range amiInfo.Images {
			var imageFile AmiNameAndData
			imageFile.ImageId = image.ImageID
			imageFile.Name = image.Name
			imageFile.CreationDate = image.CreationDate
			f, ferr := time.Parse(time.RFC3339, imageFile.CreationDate) // time format with nanoseconds
			if ferr != nil {
				fmt.Printf("error in CreationDate: %v\n", ferr)
			} else {
				imageFile.ConvertedDate = f
			}
			AllImageIds = append(AllImageIds, imageFile.ImageId)
			imageFiles = append(imageFiles, imageFile)
		}

		sort.Slice(imageFiles, func(i, j int) bool { return imageFiles[i].ConvertedDate.Before(imageFiles[j].ConvertedDate) })

		var printedSixMonths bool
		sixMonthsAgo := time.Now().AddDate(0, -6, 0)

		var printedTwelveMonths bool
		twelveMonthsAgo := time.Now().AddDate(0, -12, 0)

		var printedTwentyFourMonths bool
		twentyFourMonthsAgo := time.Now().AddDate(0, -24, 0)

		fmt.Printf("Sorted Images: %s\n", jName)
		fmt.Printf("%-50s, %-25s, %s\n", "Name", "ImageId", "CreationDate")
		for _, image := range imageFiles {
			if !printedTwentyFourMonths && (image.ConvertedDate).After(twentyFourMonthsAgo) {
				printedTwentyFourMonths = true
				fmt.Printf("Less than 24 months old:\n")
			}
			if !printedTwelveMonths && (image.ConvertedDate).After(twelveMonthsAgo) {
				printedTwelveMonths = true
				fmt.Printf("Less than 12 months old:\n")
			}
			if !printedSixMonths && (image.ConvertedDate).After(sixMonthsAgo) {
				printedSixMonths = true
				fmt.Printf("Less than 6 months old:\n")
			}
			fmt.Printf("%50s, %25s, %s\n", image.Name, image.ImageId, image.CreationDate)
		}
		fmt.Println()
	}

	AllImageIds = removeDuplicateStr(AllImageIds)

	sort.Strings(AllImageIds)
	fmt.Printf("ALL %d AMI's (with duplicates removed):\n", len(AllImageIds))
	for _, ami := range AllImageIds {
		fmt.Printf("%s\n", ami)
	}

	//gitLog("dp-setup")
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func gitLog(repoName string) {
	// Clones the given repository, creating the remote, the local branches
	// and fetching the objects, everything in memory:
	/*	fullRepoURL := "https://github.com/ONSdigital/" + repoName
		Info("git clone:")
		Info(fullRepoURL)
		r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
			URL: fullRepoURL,
		})
		CheckIfError(err)*/

	directory := "../../../" + repoName
	// Opens an already existing repository.
	r, err := git.PlainOpen(directory)
	CheckIfError(err)

	//	w, err := r.Worktree()
	//	CheckIfError(err)

	// Gets the HEAD history from HEAD, just like this command:
	Info("git log")
	Info(directory)

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)

	// ... retrieves the commit history
	since := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)   //!!! fix to 2010
	until := time.Date(2022, 11, 23, 0, 0, 0, 0, time.UTC) //!!! fix to current data and time
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
	CheckIfError(err)

	var total int
	var hashes []object.Hash
	var hashesString []string
	// ... just iterates over the commits, printing it
	err = cIter.ForEach(func(c *object.Commit) error {
		fmt.Println(c)
		total++
		hashes = append(hashes, object.Hash(c.Hash))
		hashesString = append(hashesString, c.Hash.String())
		return nil
	})
	CheckIfError(err)
	fmt.Printf("Number of logs found in %s, is: %d\n", repoName, total)

	for i, hash := range hashesString {
		if i > 7089 {
			fmt.Printf("%v\n", hash)
		}
	}
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
