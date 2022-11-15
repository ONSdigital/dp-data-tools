package main

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
// whereby I copied in the manifest file for dp-cantabular-xlsx-csv-exporter
// and then also copied in another that identified 'SecretsStyle',
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
					Count     int `yaml:"count"`
					Resources struct {
						CPU    int `yaml:"cpu"`
						Memory int `yaml:"memory"`
					} `yaml:"resources"`
				} `yaml:"sandbox"`
				Staging struct {
					Count     int `yaml:"count"`
					Resources struct {
						CPU    int `yaml:"cpu"`
						Memory int `yaml:"memory"`
					} `yaml:"resources"`
				} `yaml:"staging"`
				Production struct {
					Count     int `yaml:"count"`
					Resources struct {
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
	file, err := os.Open(manifestPath)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %s", err)
		}
	}()

	list, err := file.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}

	var manifestFiles []string

	for _, f := range list {
		fName := f.Name()
		if strings.Contains(fName, ".yml") {
			manifestFiles = append(manifestFiles, manifestPath+"/"+fName)
		}
	}

	found := false

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

		case "nomad-job", "nomad-pipeline":
			// check these in same way
			for _, group := range manifest.Nomad.Groups {
				// group has Class name of web, publishing or management
				if group.Profiles.Staging.Count != group.Profiles.Production.Count ||
					group.Profiles.Staging.Resources.CPU != group.Profiles.Production.Resources.CPU ||
					group.Profiles.Staging.Resources.Memory != group.Profiles.Production.Resources.Memory {
					found = true
					fmt.Printf("File: %s\n", ymlName)
					fmt.Println("DIFF: ", group.Class)
					if group.Profiles.Staging.Count != group.Profiles.Production.Count {
						fmt.Printf("  Staging Count = %d, Production Count = %d\n", group.Profiles.Staging.Count, group.Profiles.Production.Count)
					}

					if group.Profiles.Staging.Resources.CPU != group.Profiles.Production.Resources.CPU {
						fmt.Printf("  Staging CPU = %d, Production CPU = %d\n", group.Profiles.Staging.Resources.CPU, group.Profiles.Production.Resources.CPU)
					}

					if group.Profiles.Staging.Resources.Memory != group.Profiles.Production.Resources.Memory {
						fmt.Printf("  Staging Memory = %d, Production Memory = %d\n", group.Profiles.Staging.Resources.Memory, group.Profiles.Production.Resources.Memory)
					}
				}
			}

		case "npm", "npm-pkg":
			// do nothing for these app types

		case "static-site":
			// do nothing for this manifest type

		default:
			fmt.Printf("Unknown manifest.Type: %s\n", manifest.Type)
			os.Exit(101)
		}
	}

	if !found {
		fmt.Printf("No differences found between Staging and Production Manifest allocations\n")
	}
}
