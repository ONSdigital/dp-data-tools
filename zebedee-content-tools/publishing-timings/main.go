package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	defaultExtraMins      = 2
	defaultDir            = "/var/florence/zebedee/publish-log"
	defaultTime           = "07:00"
	defaultCollectionsCSV = "collections.csv"
	defaultReleasesCSV    = "releases.csv"
)

var timeLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		log.Fatal(err)
	}
	timeLocation = loc
}

func main() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))
	slog.Info("starting…")

	defaultTo := time.Now().Format(time.DateOnly)
	defaultFrom := time.Now().Add(-24 * 365 * time.Hour).Format(time.DateOnly) // One year Ago

	config := runConfig{}
	flag.StringVar(&config.Dir, "dir", defaultDir, "path to publish-log directory")
	flag.StringVar(&config.CollectionsFile, "cols", defaultCollectionsCSV, "filename of cols csv")
	flag.StringVar(&config.ReleasesFile, "rels", defaultReleasesCSV, "filename of releases csv")
	flag.StringVar(&config.DateFrom, "from", defaultFrom, "date from")
	flag.StringVar(&config.DateTo, "to", defaultTo, "date to")
	flag.IntVar(&config.ExtraMins, "extra", defaultExtraMins, "extra minutes to add when matching timestamps")
	timesStr := flag.String("times", defaultTime, "publishing times (comma seperated)")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	if *debug {
		logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(logHandler))
		slog.Debug("debug mode")
	}

	config.Times = strings.Split(*timesStr, ",")
	slog.Debug("parsed config", "config", config)

	err := run(config)
	if err != nil {
		log.Fatalf("run: %v", err)
	}
	slog.Info("done")
}

type runConfig struct {
	Dir             string
	Times           []string
	DateFrom        string
	DateTo          string
	CollectionsFile string
	ReleasesFile    string
	ExtraMins       int
}

func run(config runConfig) error {
	slog.Info("run start", "Dir", config.Dir)

	dateFrom, err := time.Parse(time.DateOnly, config.DateFrom)
	if err != nil {
		return fmt.Errorf("parse dateFrom: %v", err)
	}
	dateTo, err := time.Parse(time.DateOnly, config.DateTo)
	if err != nil {
		return fmt.Errorf("parse dateTo: %v", err)
	}
	dfs := os.DirFS(config.Dir)

	// main workers
	globs := identifyGlobs(dateFrom, dateTo, config.Times, config.ExtraMins)
	collectionsRead := readCollections(dfs, globs)
	collectionsFiltered := filterCollections(collectionsRead, config.Times)
	colStats := calculateCollectionStats(dfs, collectionsFiltered)
	appendedStats := appendDirStats(dfs, colStats)

	cStats, rStats := collectStats(appendedStats)
	slog.Debug("col stats", "colstats", cStats)
	slog.Debug("release stats", "relstats", rStats)

	outputCollections(config.CollectionsFile, cStats)
	outputReleases(config.ReleasesFile, rStats)

	slog.Info("run end")
	return nil
}

// create potential globs of files to process
func identifyGlobs(dateFrom, dateTo time.Time, times []string, extraMins int) chan string {
	slog.Debug("identifying globs", "dateFrom", dateFrom, "dateTo", dateTo, "times", strings.Join(times, ","), "extraMins", extraMins)
	timestamps := potentialTimesStamps(times, extraMins)
	globs := make(chan string)

	go func() {
		defer close(globs)
		for day := dateFrom; !day.After(dateTo); day = day.AddDate(0, 0, 1) {
			for _, timestamp := range timestamps {
				// Timestamps in filename are in london time not UTC
				localtime := time.Date(day.Year(), day.Month(), day.Day(), timestamp.Hour(), timestamp.Minute(), 0, 0, timeLocation)
				glob := fmt.Sprintf("%s*.json", localtime.Format("2006-01-02-15-04"))
				globs <- glob
			}
		}
	}()
	return globs
}

// To cater for late publishes, we use the wanted timestamps plus a minute or two
func potentialTimesStamps(times []string, extraMins int) []time.Time {
	timestamps := make([]time.Time, 0, len(times)*(extraMins+1))
	for _, ts := range times {
		t, err := time.ParseInLocation("15:04", ts, time.UTC) // 15:04 is a format not a value
		if err != nil {
			log.Fatalf("parse time: %v", err)
		}
		for e := 0; e <= extraMins; e++ {
			et := t.Add(time.Duration(e) * time.Minute)
			timestamps = append(timestamps, et)
		}
	}
	return timestamps
}

type Collection struct {
	filename         string
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	PublishDate      time.Time       `json:"publishDate"`
	PublishResults   []PublishResult `json:"publishResults"`
	PublishStartDate time.Time       `json:"publishStartDate"`
	PublishEndDate   time.Time       `json:"publishEndDate"`
}

type PublishResult struct {
	Transaction Transaction `json:"transaction"`
}
type Transaction struct {
	Id        string    `json:"id"`
	StartDate string    `json:"startDate"`
	UriInfos  []UriInfo `json:"uriInfos"`
}

type UriInfo struct {
	Action string `json:"action"`
	End    string `json:"end"`
	URI    string `json:"uri"`
}

// Read the collections from the collection JSON
func readCollections(dfs fs.FS, globs chan string) chan Collection {
	cols := make(chan Collection)
	go func() {
		defer close(cols)
		for glob := range globs {
			slog.Debug("searching for collection files", "glob", glob)
			filenames := findJSONFiles(dfs, glob)
			for _, filename := range filenames {
				slog.Info("reading collection file", "filename", filename)

				file, err := dfs.Open(filename)
				if err != nil {
					log.Fatal("open file", "filename", filename, "err", err)
				}
				bytes, err := io.ReadAll(file)
				if err != nil {
					log.Fatal("read file", "filename", filename, "err", err)
				}
				col := Collection{}
				err = json.Unmarshal(bytes, &col)
				if err != nil {
					log.Fatal("unmarshal collection", "filename", filename, "err", err)
				}
				col.filename = filename
				cols <- col
			}
		}
	}()
	return cols
}

func findJSONFiles(dfs fs.FS, glob string) []string {
	matches, err := fs.Glob(dfs, glob)
	if err != nil {
		log.Fatalf("run: %v", err)
	}
	return matches
}

// Filter the collections to contain only scheduled collections of the correct timestamp
func filterCollections(cols chan Collection, times []string) chan Collection {
	filtered := make(chan Collection)
	go func() {
		defer close(filtered)
		for col := range cols {
			slog.Debug("filtering collection", "col", col.filename)
			if col.Type != "scheduled" {
				slog.Debug("skipping non-scheduled collection", "name", col.Name, "type", col.Type)
				continue
			}
			colLocalTime := col.PublishDate.In(timeLocation).Format("15:04")
			if !slices.Contains(times, colLocalTime) {
				slog.Debug("skipping collection with unmatched publish time", "name", col.Name, "colLocalTime", colLocalTime)
				continue
			}

			filtered <- col
		}
	}()
	return filtered
}

type CollectionStats struct {
	Filename        string
	Name            string
	PublishDate     time.Time
	PrepublishStart *time.Time
	PrepublishEnd   *time.Time
	PublishStart    *time.Time
	PublishEnd      *time.Time
	TotalFiles      int64
	TotalBytes      int64
}

// calculate stats for an individual collection
func calculateCollectionStats(dfs fs.FS, cols chan Collection) chan CollectionStats {
	colstats := make(chan CollectionStats)
	go func() {
		defer close(colstats)
		for col := range cols {
			slog.Debug("calculating collection stats", "col", col.filename)
			colStat := CollectionStats{
				Filename:    col.filename,
				Name:        col.Name,
				PublishDate: col.PublishDate,
			}

			// Take publishing start and end directly from collection JSON
			ps, pe := col.PublishStartDate, col.PublishEndDate
			colStat.PublishStart = &ps
			colStat.PublishEnd = &pe

			prePubUris := loadPrePubUrisFromManifest(dfs, col.filename)

			// Next look at the train transaction timestamps in the collection json (publishResults[])
			for _, publishResult := range col.PublishResults {
				// First get the overall start time of the transaction for pre-publish start time
				trs, err := time.Parse("2006-01-02T15:04:05.999999999+0000", publishResult.Transaction.StartDate)
				if err != nil {
					log.Fatalf("parse time: %v", err)
				}
				if colStat.PrepublishStart == nil || trs.Before(*colStat.PrepublishStart) {
					newStart := trs
					colStat.PrepublishStart = &newStart
				}
				// Set the prepublish end to the latest start time in case there are no uris, to avoid null
				if colStat.PrepublishEnd == nil || trs.After(*colStat.PrepublishEnd) {
					newEnd := trs
					colStat.PrepublishEnd = &newEnd
				}

				// Now go through every uri in the transaction and get the max end time for pre-publishing
				for _, t := range publishResult.Transaction.UriInfos {
					if t.Action == "created" && slices.Contains(prePubUris, t.URI) {
						// Non-standard timestamp format in transaction publishResults :-(
						te, err := time.Parse("2006-01-02T15:04:05.999999999+0000", t.End)
						if err != nil {
							log.Fatalf("parse time: %v", err)
						}
						if colStat.PrepublishEnd == nil || te.After(*colStat.PrepublishEnd) {
							newEnd := te
							colStat.PrepublishEnd = &newEnd
						}
					}
				}
			}
			slog.Debug("collection stats", "colstat", colStat)

			colstats <- colStat
		}
	}()
	return colstats
}

type manifest struct {
	FilesToCopy []fileToCopy `json:"filesToCopy"`
}
type fileToCopy struct {
	Target string `json:"target"`
}

func loadPrePubUrisFromManifest(dfs fs.FS, filename string) []string {
	slog.Debug("loading uris from manifest", "filename", filename)
	dirname := filename[:strings.Index(filename, ".json")]
	sub, err := fs.Sub(dfs, dirname)
	if err != nil {
		log.Fatal("cannot open subdir: %v", err)
	}

	file, err := sub.Open("manifest.json")
	if err != nil {
		log.Fatal("cannot open manifest: %v", err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("read file", "filename", filename, "err", err)
	}
	mf := manifest{}
	err = json.Unmarshal(bytes, &mf)
	if err != nil {
		log.Fatal("unmarshal manifest", "filename", filename, "err", err)
	}

	uris := make([]string, 0, len(mf.FilesToCopy))
	for _, ftc := range mf.FilesToCopy {
		uris = append(uris, ftc.Target)
	}
	return uris
}

func appendDirStats(dfs fs.FS, cols chan CollectionStats) chan CollectionStats {
	appendedCols := make(chan CollectionStats)
	go func() {
		defer close(appendedCols)
		for col := range cols {
			slog.Debug("appending dir stats", "col", col.Filename)
			dirname := col.Filename[:strings.Index(col.Filename, ".json")]
			sub, err := fs.Sub(dfs, dirname)
			if err != nil {
				log.Fatal("cannot open subdir: %v", err)
			}
			var files, size int64
			fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
				if !d.IsDir() {
					files++
					info, err := d.Info()
					if err != nil {
						log.Fatal("cannot get info for file: %v", err)
					}
					size += info.Size()
					//slog.Debug("files size", "name", d.Name(), "size", info.Size())
				}
				return nil
			})
			slog.Debug("files size", "col", col.Filename, "name", files, "size", size)
			col.TotalFiles = files
			col.TotalBytes = size
			appendedCols <- col
		}
	}()
	return appendedCols
}

type ReleaseStats struct {
	PublishDate        time.Time
	ReleaseDescription string
	TotalCollections   int
	PrepublishStart    *time.Time
	PrepublishEnd      *time.Time
	PublishStart       *time.Time
	PublishEnd         *time.Time
	TotalFiles         int64
	TotalBytes         int64
}

func collectStats(stats chan CollectionStats) ([]CollectionStats, map[time.Time]ReleaseStats) {
	collections := make([]CollectionStats, 0)
	releases := make(map[time.Time]ReleaseStats)
	for col := range stats {
		slog.Info("collating stats for collection", "col", col.Name, "publishDate", col.PublishDate)
		slog.Debug("Results", "col", col)
		collections = append(collections, col)
		releases[col.PublishDate] = updateReleaseStats(releases[col.PublishDate], col)
	}
	return collections, releases
}

func updateReleaseStats(rs ReleaseStats, cs CollectionStats) ReleaseStats {
	rs.PublishDate = cs.PublishDate
	rs.TotalCollections++
	if rs.PrepublishStart == nil || cs.PrepublishStart.Before(*rs.PrepublishStart) {
		rs.PrepublishStart = cs.PrepublishStart
	}
	if rs.PrepublishEnd == nil || cs.PrepublishEnd.After(*rs.PrepublishEnd) {
		rs.PrepublishEnd = cs.PrepublishEnd
	}
	if rs.PublishStart == nil || cs.PublishStart.Before(*rs.PublishStart) {
		rs.PublishStart = cs.PublishStart
	}
	if rs.PublishEnd == nil || cs.PublishEnd.After(*rs.PublishEnd) {
		rs.PublishEnd = cs.PublishEnd
	}
	rs.TotalFiles += cs.TotalFiles
	rs.TotalBytes += cs.TotalBytes

	if rs.ReleaseDescription == "" {
		rs.ReleaseDescription = guessReleaseDescription(cs.Filename)
	}
	return rs
}

func guessReleaseDescription(filename string) string {
	if matched, _ := regexp.MatchString("-retailsales", filename); matched {
		return "Retail Sales"
	}
	if matched, _ := regexp.MatchString("-labourmarket", filename); matched {
		return "Labour Market"
	}
	if matched, _ := regexp.MatchString("-gdpmonthly", filename); matched {
		return "GDP"
	}
	if matched, _ := regexp.MatchString("-consumerpriceinflation", filename); matched {
		return "CPI"
	}
	if matched, _ := regexp.MatchString("-gdpquarterly", filename); matched {
		return "Quarterly National Accounts"
	}
	if matched, _ := regexp.MatchString("-publicsector", filename); matched {
		return "Public Sector Finances"
	}
	return ""
}

func outputCollections(filename string, stats []CollectionStats) {
	slog.Info("saving collections", "filename", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("cannot create collections file: %v", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	header := []string{
		"PublishDate",
		"Name",
		"Prepublish Time",
		"Publish Time",
		"Files",
		"Size",
	}
	defer w.Flush()
	w.Write(header)
	for _, col := range stats {
		prePubDuration := col.PrepublishEnd.Sub(*col.PrepublishStart)
		pubDuration := col.PublishEnd.Sub(*col.PublishStart)
		record := []string{
			col.PublishDate.Format(time.DateTime),
			col.Name,
			fmt.Sprintf("%.2f", prePubDuration.Seconds()),
			fmt.Sprintf("%.2f", pubDuration.Seconds()),
			fmt.Sprintf("%d", col.TotalFiles),
			fmt.Sprintf("%d", col.TotalBytes),
		}
		w.Write(record)
	}
	slog.Info("generated collections csv", "filename", filename)
}

func outputReleases(filename string, releases map[time.Time]ReleaseStats) {
	slog.Info("saving releases", "filename", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("cannot create releases file: %v", err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	header := []string{
		"PublishDate",
		"Description",
		"Collections",
		"Prepublish Time",
		"Publish Time",
		"Files",
		"Size",
	}
	defer w.Flush()
	w.Write(header)

	dates := slices.SortedFunc(maps.Keys(releases), func(t time.Time, t2 time.Time) int {
		return t.Compare(t2)
	})
	for _, date := range dates {
		release := releases[date]
		prePubDuration := release.PrepublishEnd.Sub(*release.PrepublishStart)
		pubDuration := release.PublishEnd.Sub(*release.PublishStart)
		record := []string{
			release.PublishDate.Format(time.DateTime),
			release.ReleaseDescription,
			fmt.Sprintf("%d", release.TotalCollections),
			fmt.Sprintf("%.2f", prePubDuration.Seconds()),
			fmt.Sprintf("%.2f", pubDuration.Seconds()),
			fmt.Sprintf("%d", release.TotalFiles),
			fmt.Sprintf("%d", release.TotalBytes),
		}
		w.Write(record)
	}
	slog.Info("generated releases csv", "filename", filename)
}
