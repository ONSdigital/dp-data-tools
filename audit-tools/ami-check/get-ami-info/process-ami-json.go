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
)

var environmentFiles = [...]string{"../tmp/staging-amis.json", "../tmp/sandbox-amis.json", "../tmp/prod-amis.json"}

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

var AllImageFiles []AmiNameAndData

const (
	tmpDir     = "../tmp"
	resultsDir = "../results"
)

func main() {
	ensureDirectoryExists(resultsDir)
	resultsFile, err := os.Create(resultsDir + "/sorted-amis.txt")
	check(err)
	defer resultsFile.Close()

	// check each environment file
	for _, jName := range environmentFiles {
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

		var environmentImageFiles []AmiNameAndData

		// extract and accumulate list of info for each ami
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
			AllImageFiles = append(AllImageFiles, imageFile)
			environmentImageFiles = append(environmentImageFiles, imageFile)
		}

		// sort by putting oldest first
		sort.Slice(environmentImageFiles, func(i, j int) bool {
			return environmentImageFiles[i].ConvertedDate.Before(environmentImageFiles[j].ConvertedDate)
		})

		var printedSixMonths bool
		sixMonthsAgo := time.Now().AddDate(0, -6, 0)

		var printedTwelveMonths bool
		twelveMonthsAgo := time.Now().AddDate(0, -12, 0)

		var printedTwentyFourMonths bool
		twentyFourMonthsAgo := time.Now().AddDate(0, -24, 0)

		displayAndSave(resultsFile, fmt.Sprintf("Sorted Images, used in: %s\n", jName))
		displayAndSave(resultsFile, fmt.Sprintf("%-50s, %-25s, %s\n", "Name", "ImageId", "CreationDate"))
		for _, image := range environmentImageFiles {
			if !printedTwentyFourMonths && (image.ConvertedDate).After(twentyFourMonthsAgo) {
				printedTwentyFourMonths = true
				displayAndSave(resultsFile, "Created 24 to 12 months ago:\n")
			}
			if !printedTwelveMonths && (image.ConvertedDate).After(twelveMonthsAgo) {
				printedTwelveMonths = true
				displayAndSave(resultsFile, "Created 12 to 6 months ago:\n")
			}
			if !printedSixMonths && (image.ConvertedDate).After(sixMonthsAgo) {
				printedSixMonths = true
				displayAndSave(resultsFile, "Created in last 6 months:\n")
			}
			displayAndSave(resultsFile, fmt.Sprintf("%50s, %25s, %s\n", image.Name, image.ImageId, image.CreationDate))
		}
		displayAndSave(resultsFile, "\n")
	}

	AllImageFiles = removeDuplicateImageId(AllImageFiles)
	// sort by putting oldest first
	sort.Slice(AllImageFiles, func(i, j int) bool {
		return AllImageFiles[i].ConvertedDate.Before(AllImageFiles[j].ConvertedDate)
	})

	idsFile, err := os.Create(resultsDir + "/all-ami-ids.txt")
	check(err)
	defer idsFile.Close()

	// We dont save the following title to file so that the file only contains a list of all ami id's
	fmt.Printf("ALL %d AMI's (with duplicates removed):\n", len(AllImageFiles))
	// Save list with all into, so that scan-repo can utilise creation dates to determine how far back in time to process other repo's.
	for _, imageFile := range AllImageFiles {
		displayAndSave(idsFile, fmt.Sprintf("%s, %s, %s\n", imageFile.ImageId, imageFile.CreationDate, imageFile.Name))
	}
}

func ensureDirectoryExists(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		check(os.Mkdir(dirName, 0700))
	}
}

func removeDuplicateImageId(itemSlice []AmiNameAndData) []AmiNameAndData {
	allKeys := make(map[string]bool)
	list := []AmiNameAndData{}
	for _, item := range itemSlice {
		if _, value := allKeys[item.ImageId]; !value {
			allKeys[item.ImageId] = true
			list = append(list, item)
		}
	}
	return list
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
