package main

// This utility looks for and reports specific differences between Staging and Prod manifests.
// The differences it reports are for: CPU, Memory and Count (number of instances).
// The need for this App arose during TISS Load testing when we wanted Staging and Prod to
// have equivalent resources and manually checking over 100 manifest files needed automating.

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	manifestPath = "../../../dp-configs/manifests"
)

// NomadManifest was created using:
// https://zhwt.github.io/yaml-to-go/
//
// whereby I copied in the manifest file for dp-cantabular-csv-exporter
// and then also copied in another that identified 'SecretsStyle', and another for 'StaticBuckets'
// that was then merged in with the below struct.
type NomadManifest struct {
	Name           string `yaml:"name"`
	RepoURI        string `yaml:"repo_uri"`
	Type           string `yaml:"type"`
	NeedsPrivilege bool   `yaml:"needs_privilege"`
	StaticBuckets  struct {
		Sandbox string `yaml:"sandbox"`
		Prod    string `yaml:"prod"`
	} `yaml:"static_buckets"`
	Nomad struct {
		Groups []struct {
			Class    string `yaml:"class"`
			Profiles struct {
				Sandbox struct {
					Count       int `yaml:"count"`
					MaxParallel int `yaml:"max_parallel"`
					Resources   struct {
						CPU    int `yaml:"cpu"`
						Memory int `yaml:"memory"`
					} `yaml:"resources"`
				} `yaml:"sandbox"`
				Staging struct {
					Count       int `yaml:"count"`
					MaxParallel int `yaml:"max_parallel"`
					Resources   struct {
						CPU    int `yaml:"cpu"`
						Memory int `yaml:"memory"`
					} `yaml:"resources"`
				} `yaml:"staging"`
				Production struct {
					Count       int `yaml:"count"`
					MaxParallel int `yaml:"max_parallel"`
					Resources   struct {
						CPU    int `yaml:"cpu"`
						Memory int `yaml:"memory"`
					} `yaml:"resources"`
				} `yaml:"production"`
			} `yaml:"profiles"`
		} `yaml:"groups"`
	} `yaml:"nomad"`
	Kafka struct {
		SecretsStyle string `yaml:"secrets_style"`
		Topics       []struct {
			Name    string   `yaml:"name"`
			Subnets []string `yaml:"subnets"`
			Access  []string `yaml:"access"`
		} `yaml:"topics"`
	} `yaml:"kafka"`
}

func main() {
	// get file handle for the manifests directory
	file, err := os.Open(manifestPath)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %s", err)
		}
	}()

	// read all files in manifests directory
	list, err := file.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}

	// create list of only the '.yml' manifest files
	var manifestFiles []string
	for _, f := range list {
		fName := f.Name()
		if strings.Contains(fName, ".yml") {
			manifestFiles = append(manifestFiles, manifestPath+"/"+fName)
		}
	}

	differencesFound := false

	// check each manifest file
	for _, ymlName := range manifestFiles {
		yFile, err := os.ReadFile(ymlName)
		if err != nil {
			fmt.Printf("Failed reading %s, with error: %v\n", ymlName, err)
			os.Exit(100)
		}

		manifest := NomadManifest{}
		err = yaml.Unmarshal(yFile, &manifest)
		if err != nil {
			fmt.Printf("Failed unmarshaling yaml file %s, with error: %v\n", ymlName, err)
			os.Exit(101)
		}

		switch manifest.Type {
		case "build-from-s3":
			// do nothing for cantabular app

		case "cdn-asset":
			// do nothing for this app type

		case "library", "library-v2":
			// do nothing for libraries

		case "nomad-job", "nomad-pipeline", "docker-deploy":
			// check these in same way
			for _, group := range manifest.Nomad.Groups {
				// group has Class name of web, publishing or management
				if group.Profiles.Staging.Count != group.Profiles.Production.Count ||
					group.Profiles.Staging.Resources.CPU != group.Profiles.Production.Resources.CPU ||
					group.Profiles.Staging.Resources.Memory != group.Profiles.Production.Resources.Memory {

					differencesFound = true
					fmt.Printf("File: %s\n", ymlName)
					fmt.Println("DIFF: ", group.Class)
					if group.Profiles.Staging.Count != group.Profiles.Production.Count {
						fmt.Printf("  Staging Count  = %4d,  Production Count  = %4d\n", group.Profiles.Staging.Count, group.Profiles.Production.Count)
					}

					if group.Profiles.Staging.Resources.CPU != group.Profiles.Production.Resources.CPU {
						fmt.Printf("  Staging CPU    = %4d,  Production CPU    = %4d\n", group.Profiles.Staging.Resources.CPU, group.Profiles.Production.Resources.CPU)
					}

					if group.Profiles.Staging.Resources.Memory != group.Profiles.Production.Resources.Memory {
						fmt.Printf("  Staging Memory = %4d,  Production Memory = %4d\n", group.Profiles.Staging.Resources.Memory, group.Profiles.Production.Resources.Memory)
					}
				}
			}

		case "npm", "npm-pkg":
			// do nothing for these app types

		case "static-site":
			// do nothing for this manifest type

		default:
			fmt.Printf("In file: %s\n", ymlName)
			fmt.Printf("Unknown manifest.Type: %s\n", manifest.Type)
			os.Exit(101)
		}
	}

	if !differencesFound {
		fmt.Printf("No differences found between Staging and Production Manifest allocations\n")
	}
}
