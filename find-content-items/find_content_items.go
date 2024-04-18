package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Data struct {
	DataType      string `json:"type"`
	LatestRelease bool   `json:"latestRelease"`
}

func main() {
	var (
		filterType        string
		dir               string
		latestReleaseOnly bool
	)
	flag.StringVar(&filterType, "filter", "", "type to filter counts")
	flag.StringVar(&dir, "directory", "", "directory to operate in")
	flag.BoolVar(&latestReleaseOnly, "latestrelease", false, "filter by latest release only")
	flag.Parse()

	if dir == "" {
		fmt.Println("no directory specified")
		os.Exit(1)
	}

	totalFiles := 0
	totalTime := time.Duration(0)

	counts := make(map[string]int)

	count, elapsed, err := findFiles(dir, counts, filterType, latestReleaseOnly)
	if err != nil {
		fmt.Printf("error while searching in %s: %v\n", dir, err)
	}
	totalFiles += count
	totalTime += elapsed
	fmt.Printf("Found %d files in %s\n", count, dir)

	fmt.Printf("Total time taken: %v\n", totalTime)
	fmt.Printf("Total files found: %d\n", totalFiles)

	displayCounts(counts)

	if filterType != "" {
		displayFilteredCounts(counts, filterType)
	}
}

func findFiles(directory string, counts map[string]int, filterType string, latestReleaseOnly bool) (int, time.Duration, error) {
	var count int
	start := time.Now()

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "data.json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var jsonData Data
			if err := json.Unmarshal(data, &jsonData); err != nil {
				return err
			}

			if latestReleaseOnly && !jsonData.LatestRelease {
				return nil
			}

			count++

			if filterType == "" || jsonData.DataType == filterType {
				counts[jsonData.DataType]++
			}

			fmt.Printf("Specific field in %s: %s\n", path, jsonData.DataType)
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	elapsed := time.Since(start)
	return count, elapsed, nil
}

func displayCounts(counts map[string]int) {
	fmt.Println("Counts by specific type:")
	for dataType, count := range counts {
		fmt.Printf("%s: %d\n", dataType, count)
	}
}

func displayFilteredCounts(counts map[string]int, filteredType string) {
	fmt.Printf("Counts of filtered type '%s': %d\n", filteredType, counts[filteredType])
}
