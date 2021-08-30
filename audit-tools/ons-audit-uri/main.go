package main

// NOTE: Any changes to the ONS website might stop this App working and it will need adjusting ..
//       That is, the struct(s) may need additional fields.

// NOTE: to grab all output info from running this use:
//       go run main.go >t.txt

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents config for ons-scrape
type Config struct {
	FullDepth              bool `envconfig:"FULL_DEPTH"`               // search deeper through the whole site from content down
	OnlyFirstFullDepth     bool `envconfig:"ONLY_FIRST_FULL_DEPTH"`    // enable this to minimise the amount of full depth for code development / testing
	SkipVersions           bool `envconfig:"SKIP_VERSIONS"`            // when doing FULL_DEPTH, this skips processing of version files (to save time when developing this code)
	PlayNice               bool `envconfig:"PLAY_NICE"`                // add a little delay before reading each page
	UseThreads             bool `envconfig:"USE_THREADS"`              // set true to use more than 1 thread to read pages (to run faster)
	SaveSite               bool `envconfig:"SAVE_SITE"`                // set true to create mongo init scripts for all page types
	LimitReads             bool `envconfig:"LIMIT_READS"`              // set true to limit data read per second
	IncludeReleaseCalendar bool `envconfig:"INCLUDE_RELEASE_CALENDAR"` // set to true to include processing the release calendar
}

var cfg *Config

// Sensible combinations of flags are:
//
//    This also generates:
//    broken_links.txt : a report on links in ONS site that are not working
//           (in directory: observations)
//    If a page has problems being decoded into a go struct two files may get written
//    into directory 'temp' depending on the nature of the problem.
//    These two .json files can be formatted in vscode, saved and then visually compared
//    in an application like 'meld' to determine how to adjust go struct to decode page /data.
//    There are a number of other files generated for diagnostics and future development,
//    see the main() for functions called to see what these are (they are in a state of flux
//    and may change).
//
// 2. FullDepth: true
//    This will scrape through the whole ONS site and generate more mongo init scripts named by
//    the 'type' of the page found.
//    This can take 30 minutes, to 5 hours to run, depending on which part of ONS site is scanned.
//    Also, the broken_links.txt file may be a lot bigger.
//
// 3. Other flags: OnlyFirstFullDepth, SkipVersions
//    When 'FullDepth' is true, setting these flags 'true' will reduce the amount of the ONS site
//    that is scanned. This is useful when developing the code and the definitions of the struct's.
//

// InitConfig returns the default config with any modifications through environment
// variables
func InitConfig() error {
	cfg = &Config{
		FullDepth:              true,
		OnlyFirstFullDepth:     false,
		SkipVersions:           false,
		PlayNice:               false,
		UseThreads:             true,
		SaveSite:               false,
		LimitReads:             true,
		IncludeReleaseCalendar: true,
	}

	return envconfig.Process("", cfg)
}

/*
	a full depth search on 24th Jan 2021 got the number of URI's to search per depth:

	Number to check at depth: 1   is: 4
	Number to check at depth: 2   is: 32
	Number to check at depth: 3   is: 375
	Number to check at depth: 4   is: 4723
	Number to check at depth: 5   is: 12556
	Number to check at depth: 6   is: 12585
	Number to check at depth: 7   is: 6948
	Number to check at depth: 8   is: 700
	Number to check at depth: 9   is: 217
	Number to check at depth: 10  is: 81
	Number to check at depth: 11  is: 49
	Number to check at depth: 12  is: 10

*/

// NOTE: the order that fields appear within structs is critical for this App to work !

// DataResponse is the whole data page that gets filled from Unmarshal'ing pages like:
// https://www.ons.gov.uk/businessindustryandtrade/data
// and this is a top level 'node' like page
type DataResponse struct {
	Sections                  *[]SubLink                        `bson:"sections,omitempty"                   json:"sections,omitempty"`
	Items                     *[]ItemLinks                      `bson:"items,omitempty"                      json:"items,omitempty"`
	Datasets                  *[]DatasetLinks                   `bson:"datasets,omitempty"                   json:"datasets,omitempty"`
	HighlightedLinks          *[]HighlightLinks                 `bson:"highlightedLinks,omitempty"           json:"highlightedLinks,omitempty"`
	StatsBulletins            *[]StatsBulletLinks               `bson:"statsBulletins,omitempty"             json:"statsBulletins,omitempty"`
	RelatedArticles           *[]RelatedArticleLinks            `bson:"relatedArticles,omitempty"            json:"relatedArticles,omitempty"`
	RelatedMethodology        *[]RelatedMethodLinks             `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]RelatedMethodologyArticleLinks `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	HighlightedContent        *[]HighlightedContentLinks        `bson:"highlightedContent,omitempty"         json:"highlightedContent,omitempty"`
	Index                     *int                              `bson:"index,omitempty"                      json:"index,omitempty"`
	Type                      *string                           `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                           `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *Descript                         `bson:"description,omitempty"                json:"description,omitempty"`
}

// SubLink is the sub page links
type SubLink struct {
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
	Index *int    `bson:"index,omitempty"  json:"index,omitempty"`
}

// HighlightLinks are highlights
type HighlightLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// Descript is the page description
type Descript struct {
	Title           *string   `bson:"title,omitempty"            json:"title,omitempty"`
	Summary         *string   `bson:"summary,omitempty"          json:"summary,omitempty"`
	Keywords        *[]string `bson:"keywords,omitempty"         json:"keywords,omitempty"`
	MetaDescription *string   `bson:"metaDescription,omitempty"  json:"metaDescription,omitempty"`
	Unit            *string   `bson:"unit,omitempty"             json:"unit,omitempty"`
	PreUnit         *string   `bson:"preUnit,omitempty"          json:"preUnit,omitempty"`
	Source          *string   `bson:"source,omitempty"           json:"source,omitempty"`
}

// ItemLinks are highlights
type ItemLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// DatasetLinks are highlights
type DatasetLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// StatsBulletLinks are stats bulletins
type StatsBulletLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// RelatedArticleLinks are related articles
type RelatedArticleLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// RelatedMethodLinks are related methodologies
type RelatedMethodLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// RelatedMethodologyArticleLinks are related methodology articles
type RelatedMethodologyArticleLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// HighlightedContentLinks are highlighted content
type HighlightedContentLinks struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// ContactDetails represents an object containing information of the contact
type ContactDetails struct {
	Email     *string `bson:"email,omitempty"      json:"email,omitempty"`
	Name      *string `bson:"name,omitempty"       json:"name,omitempty"`
	Telephone *string `bson:"telephone,omitempty"  json:"telephone,omitempty"`
}

// HomePageResponse represents the home page
type HomePageResponse struct {
	FeaturedContent *[]FeaturedContentLinks `bson:"featuredContent,omitempty"  json:"featuredContent,omitempty"`
	AroundONS       *[]string               `bson:"aroundONS,omitempty"        json:"aroundONS,omitempty"`
	ServiceMessage  *string                 `bson:"serviceMessage,omitempty"   json:"serviceMessage,omitempty"`
	Type            *string                 `bson:"type,omitempty"             json:"type,omitempty"`
	URI             *string                 `bson:"uri,omitempty"              json:"uri,omitempty"`
	Description     *Descript               `bson:"description,omitempty"      json:"description,omitempty"`
}

// FeaturedContentLinks - sub section links on home page
type FeaturedContentLinks struct {
	Title       *string `bson:"title,omitempty"        json:"title,omitempty"`
	URI         *string `bson:"uri,omitempty"          json:"uri,omitempty"`
	Description *string `bson:"description,omitempty"  json:"description,omitempty"`
	Image       *string `bson:"image,omitempty"        json:"image,omitempty"`
}

// ===
// pageShape used to determine how to decode a page
type pageShape struct {
	Type *string `bson:"type,omitempty"  json:"type,omitempty"`
}

// ===
type articleResponse struct {
	RelatedArticles           *[]relatedArticles           `bson:"relatedArticles,omitempty"            json:"relatedArticles,omitempty"`
	PdfTable                  *[]pdfTable                  `bson:"pdfTable,omitempty"                   json:"pdfTable,omitempty"`
	IsPrototypeArticle        *bool                        `bson:"isPrototypeArticle,omitempty"         json:"isPrototypeArticle,omitempty"`
	IsReleaseDateEnabled      *bool                        `bson:"isReleaseDateEnabled,omitempty"       json:"isReleaseDateEnabled,omitempty"`
	ImageURI                  *string                      `bson:"imageUri,omitempty"                   json:"imageUri,omitempty"`
	Sections                  *[]sections                  `bson:"sections,omitempty"                   json:"sections,omitempty"`
	Accordion                 *[]accordion                 `bson:"accordion,omitempty"                  json:"accordion,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Charts                    *[]charts                    `bson:"charts,omitempty"                     json:"charts,omitempty"`
	Tables                    *[]tables                    `bson:"tables,omitempty"                     json:"tables,omitempty"`
	Images                    *[]images                    `bson:"images,omitempty"                     json:"images,omitempty"`
	Equations                 *[]equations                 `bson:"equations,omitempty"                  json:"equations,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

// ===
type articleDownloadResponse struct {
	Downloads                 *[]downloads                 `bson:"downloads,omitempty"                  json:"downloads,omitempty"`
	Markdown                  *[]string                    `bson:"markdown,omitempty"                   json:"markdown,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Charts                    *[]charts                    `bson:"charts,omitempty"                     json:"charts,omitempty"`
	Tables                    *[]tables                    `bson:"tables,omitempty"                     json:"tables,omitempty"`
	Images                    *[]images                    `bson:"images,omitempty"                     json:"images,omitempty"`
	Equations                 *[]equations                 `bson:"equations,omitempty"                  json:"equations,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

// ===
type bulletinResponse struct {
	RelatedBulletins          *[]relatedBulletins          `bson:"relatedBulletins,omitempty"           json:"relatedBulletins,omitempty"`
	PdfTable                  *[]pdfTable                  `bson:"pdfTable,omitempty"                   json:"pdfTable,omitempty"`
	Sections                  *[]sections                  `bson:"sections,omitempty"                   json:"sections,omitempty"`
	Accordion                 *[]accordion                 `bson:"accordion,omitempty"                  json:"accordion,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Charts                    *[]charts                    `bson:"charts,omitempty"                     json:"charts,omitempty"`
	Tables                    *[]tables                    `bson:"tables,omitempty"                     json:"tables,omitempty"`
	Images                    *[]images                    `bson:"images,omitempty"                     json:"images,omitempty"`
	Equations                 *[]equations                 `bson:"equations,omitempty"                  json:"equations,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

// ===
type compendiumDataResponse struct {
	Downloads                 *[]downloads                 `bson:"downloads,omitempty"                  json:"downloads,omitempty"`
	RelatedDatasets           *[]relatedDatasets           `bson:"relatedDatasets,omitempty"            json:"relatedDatasets,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

type compendiumLandingPageResponse struct {
	Datasets                  *[]datasets                  `bson:"datasets,omitempty"                   json:"datasets,omitempty"`
	Chapters                  *[]chapters                  `bson:"chapters,omitempty"                   json:"chapters,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

type datasetLandingPageResponse struct {
	Section                   *section                     `bson:"section,omitempty"                    json:"section,omitempty"`
	RelatedFilterableDatasets *[]relatedFilterableDatasets `bson:"relatedFilterableDatasets,omitempty"  json:"relatedFilterableDatasets,omitempty"`
	RelatedDatasets           *[]relatedDatasets           `bson:"relatedDatasets,omitempty"            json:"relatedDatasets,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Datasets                  *[]datasets                  `bson:"datasets,omitempty"                   json:"datasets,omitempty"`
	Timeseries                *bool                        `bson:"timeseries,omitempty"                 json:"timeseries,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

type staticMethodologyResponse struct {
	PdfTable                  *[]pdfTable                  `bson:"pdfTable,omitempty"                   json:"pdfTable,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Downloads                 *[]downloads                 `bson:"downloads,omitempty"                  json:"downloads,omitempty"`
	Sections                  *[]sections                  `bson:"sections,omitempty"                   json:"sections,omitempty"`
	Accordion                 *[]accordion                 `bson:"accordion,omitempty"                  json:"accordion,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	Charts                    *[]charts                    `bson:"charts,omitempty"                     json:"charts,omitempty"`
	Tables                    *[]tables                    `bson:"tables,omitempty"                     json:"tables,omitempty"`
	Images                    *[]images                    `bson:"images,omitempty"                     json:"images,omitempty"`
	Equations                 *[]equations                 `bson:"equations,omitempty"                  json:"equations,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
	Topics                    *[]ctopics                   `bson:"topics,omitempty"                     json:"topics,omitempty"`
}

type staticMethodologyDownloadResponse struct {
	RelatedDocuments *[]relatedDocuments `bson:"relatedDocuments,omitempty"  json:"relatedDocuments,omitempty"`
	RelatedDatasets  *[]relatedDatasets  `bson:"relatedDatasets,omitempty"   json:"relatedDatasets,omitempty"`
	PdfDownloads     *[]pdfDownloads     `bson:"pdfDownloads,omitempty"      json:"pdfDownloads,omitempty"`
	Alerts           *[]alerts           `bson:"alerts,omitempty"            json:"alerts,omitempty"`
	Downloads        *[]downloads        `bson:"downloads,omitempty"         json:"downloads,omitempty"`
	Markdown         *[]string           `bson:"markdown,omitempty"          json:"markdown,omitempty"`
	Links            *[]links            `bson:"links,omitempty"             json:"links,omitempty"`
	Type             *string             `bson:"type,omitempty"              json:"type,omitempty"`
	URI              *string             `bson:"uri,omitempty"               json:"uri,omitempty"`
	Description      *cDescription       `bson:"description,omitempty"       json:"description,omitempty"`
	Topics           *[]ctopics          `bson:"topics,omitempty"            json:"topics,omitempty"`
}

type staticQmiResponse struct {
	RelatedDocuments *[]relatedDocuments `bson:"relatedDocuments,omitempty"  json:"relatedDocuments,omitempty"`
	RelatedDatasets  *[]relatedDatasets  `bson:"relatedDatasets,omitempty"   json:"relatedDatasets,omitempty"`
	Downloads        *[]downloads        `bson:"downloads,omitempty"         json:"downloads,omitempty"`
	Markdown         *[]string           `bson:"markdown,omitempty"          json:"markdown,omitempty"`
	Links            *[]links            `bson:"links,omitempty"             json:"links,omitempty"`
	Type             *string             `bson:"type,omitempty"              json:"type,omitempty"`
	URI              *string             `bson:"uri,omitempty"               json:"uri,omitempty"`
	Description      *cDescription       `bson:"description,omitempty"       json:"description,omitempty"`
	Topics           *[]ctopics          `bson:"topics,omitempty"            json:"topics,omitempty"`
}

type timeseriesResponse struct {
	Years            *[]years            `bson:"years,omitempty"             json:"years,omitempty"`
	Quarters         *[]quarters         `bson:"quarters,omitempty"          json:"quarters,omitempty"`
	Months           *[]months           `bson:"months,omitempty"            json:"months,omitempty"`
	SourceDatasets   *[]sourceDatasets   `bson:"sourceDatasets,omitempty"    json:"sourceDatasets,omitempty"`
	RelatedDatasets  *[]relatedDatasets  `bson:"relatedDatasets,omitempty"   json:"relatedDatasets,omitempty"`
	Section          *section            `bson:"section,omitempty"           json:"section,omitempty"`
	Notes            *[]string           `bson:"notes,omitempty"             json:"notes,omitempty"`
	RelatedDocuments *[]relatedDocuments `bson:"relatedDocuments,omitempty"  json:"relatedDocuments,omitempty"`
	RelatedData      *[]relatedData      `bson:"relatedData,omitempty"       json:"relatedData,omitempty"`
	Alerts           *[]alerts           `bson:"alerts,omitempty"            json:"alerts,omitempty"`
	Versions         *[]versions         `bson:"versions,omitempty"          json:"versions,omitempty"`
	Type             *string             `bson:"type,omitempty"              json:"type,omitempty"`
	URI              *string             `bson:"uri,omitempty"               json:"uri,omitempty"`
	Description      *cDescription       `bson:"description,omitempty"       json:"description,omitempty"`
	Topics           *[]ctopics          `bson:"topics,omitempty"            json:"topics,omitempty"`
}

type chartResponse struct {
	Subtitle           *string `bson:"subtitle,omitempty"            json:"subtitle,omitempty"`
	Filename           *string `bson:"filename,omitempty"            json:"filename,omitempty"`
	Source             *string `bson:"source,omitempty"              json:"source,omitempty"`
	Notes              *string `bson:"notes,omitempty"               json:"notes,omitempty"`
	AltText            *string `bson:"altText,omitempty"             json:"altText,omitempty"`
	LabelInterval      *string `bson:"labelInterval,omitempty"       json:"labelInterval,omitempty"`
	DecimalPlaces      *string `bson:"decimalPlaces,omitempty"       json:"decimalPlaces,omitempty"`
	DecimalPlacesYaxis *string `bson:"decimalPlacesYaxis,omitempty"  json:"decimalPlacesYaxis,omitempty"`
	Palette            *string `bson:"palette,omitempty"             json:"palette,omitempty"`
	XAxisPos           *string `bson:"xAxisPos,omitempty"            json:"xAxisPos,omitempty"`
	YAxisPos           *string `bson:"yAxisPos,omitempty"            json:"yAxisPos,omitempty"`
	YAxisMax           *string `bson:"yAxisMax,omitempty"            json:"yAxisMax,omitempty"`
	YMin               *string `bson:"yMin,omitempty"                json:"yMin,omitempty"`
	YMax               *string `bson:"yMax,omitempty"                json:"yMax,omitempty"`
	YxisInterval       *string `bson:"yAxisInterval,omitempty"       json:"yAxisInterval,omitempty"`
	Highlight          *string `bson:"highlight,omitempty"           json:"highlight,omitempty"`
	Alpha              *string `bson:"alpha,omitempty"               json:"alpha,omitempty"`
	Unit               *string `bson:"unit,omitempty"                json:"unit,omitempty"`
	XAxisLabel         *string `bson:"xAxisLabel,omitempty"          json:"xAxisLabel,omitempty"`
	AspectRatio        *string `bson:"aspectRatio,omitempty"         json:"aspectRatio,omitempty"`
	ChartType          *string `bson:"chartType,omitempty"           json:"chartType,omitempty"`
	// NOTE: 'go' does not maintain the order of the items in the 'map',
	//       though the order of the array is maintained in 'Data' below
	Data            *[]map[string]string `bson:"data,omitempty"             json:"data,omitempty"`
	Headers         *[]string            `bson:"headers,omitempty"          json:"headers,omitempty"`
	Series          *[]string            `bson:"series,omitempty"           json:"series,omitempty"`
	Categories      *[]string            `bson:"categories,omitempty"       json:"categories,omitempty"`
	ChartTypes      *map[string]string   `bson:"chartTypes,omitempty"       json:"chartTypes,omitempty"`
	LineTypes       *map[string]string   `bson:"lineTypes,omitempty"        json:"lineTypes,omitempty"`
	Groups          *[][]string          `bson:"groups,omitempty"           json:"groups,omitempty"`
	StartFromZero   *bool                `bson:"startFromZero,omitempty"    json:"startFromZero,omitempty"`
	FinishAtHundred *bool                `bson:"finishAtHundred,omitempty"  json:"finishAtHundred,omitempty"`
	IsStacked       *bool                `bson:"isStacked,omitempty"        json:"isStacked,omitempty"`
	IsReversed      *bool                `bson:"isReversed,omitempty"       json:"isReversed,omitempty"`
	ShowTooltip     *bool                `bson:"showTooltip,omitempty"      json:"showTooltip,omitempty"`
	ShowMarker      *bool                `bson:"showMarker,omitempty"       json:"showMarker,omitempty"`
	HasLineBreak    *bool                `bson:"hasLineBreak,omitempty"     json:"hasLineBreak,omitempty"`
	HasConnectNull  *bool                `bson:"hasConnectNull,omitempty"   json:"hasConnectNull,omitempty"`
	IsEditor        *bool                `bson:"isEditor,omitempty"         json:"isEditor,omitempty"`
	Annotations     *[]annotations       `bson:"annotations,omitempty"      json:"annotations,omitempty"`
	Devices         *devices             `bson:"devices,omitempty"          json:"devices,omitempty"`
	Files           *[]files             `bson:"files,omitempty"            json:"files,omitempty"`
	Title           *string              `bson:"title,omitempty"            json:"title,omitempty"`
	Type            *string              `bson:"type,omitempty"             json:"type,omitempty"`
	URI             *string              `bson:"uri,omitempty"              json:"uri,omitempty"`
}

type tableResponse struct {
	Filename       *string        `bson:"filename,omitempty"        json:"filename,omitempty"`
	FirstLineTitle *bool          `bson:"firstLineTitle,omitempty"  json:"firstLineTitle,omitempty"`
	HeaderRows     *string        `bson:"headerRows,omitempty"      json:"headerRows,omitempty"`
	Modifications  *modifications `bson:"modifications,omitempty"   json:"modifications,omitempty"`
	Files          *[]files       `bson:"files,omitempty"           json:"files,omitempty"`
	Title          *string        `bson:"title,omitempty"           json:"title,omitempty"`
	Type           *string        `bson:"type,omitempty"            json:"type,omitempty"`
	URI            *string        `bson:"uri,omitempty"             json:"uri,omitempty"`
}

type equationResponse struct {
	Filename *string  `bson:"filename,omitempty"  json:"filename,omitempty"`
	Content  *string  `bson:"content,omitempty"   json:"content,omitempty"`
	Files    *[]files `bson:"files,omitempty"     json:"files,omitempty"`
	Title    *string  `bson:"title,omitempty"     json:"title,omitempty"`
	Type     *string  `bson:"type,omitempty"      json:"type,omitempty"`
	URI      *string  `bson:"uri,omitempty"       json:"uri,omitempty"`
}

type imageResponse struct {
	Subtitle *string  `bson:"subtitle,omitempty"  json:"subtitle,omitempty"`
	Filename *string  `bson:"filename,omitempty"  json:"filename,omitempty"`
	Source   *string  `bson:"source,omitempty"    json:"source,omitempty"`
	Notes    *string  `bson:"notes,omitempty"     json:"notes,omitempty"`
	AltText  *string  `bson:"altText,omitempty"   json:"altText,omitempty"`
	Files    *[]files `bson:"files,omitempty"     json:"files,omitempty"`
	Title    *string  `bson:"title,omitempty"     json:"title,omitempty"`
	Type     *string  `bson:"type,omitempty"      json:"type,omitempty"`
	URI      *string  `bson:"uri,omitempty"       json:"uri,omitempty"`
}

type releaseResponse struct {
	Markdown                  *[]string                    `bson:"markdown,omitempty"                   json:"markdown,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	RelatedDatasets           *[]relatedDatasets           `bson:"relatedDatasets,omitempty"            json:"relatedDatasets,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	DateChanges               *[]dateChanges               `bson:"dateChanges,omitempty"                json:"dateChanges,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
}

type dateChanges struct {
	PreviousDate *string `bson:"previousDate,omitempty"  json:"previousDate,omitempty"`
	ChangeNotice *string `bson:"changeNotice,omitempty"  json:"changeNotice,omitempty"`
}

type listResponse struct {
	Type     *string `bson:"type,omitempty"      json:"type,omitempty"`
	ListType *string `bson:"listType,omitempty"  json:"listType,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
	Result   *result `bson:"result,omitempty"    json:"result,omitempty"`
	Counts   *counts `bson:"counts,omitempty"    json:"counts,omitempty"`
}

type counts struct {
	NumberOfResults *int       `bson:"numberOfResults,omitempty"  json:"numberOfResults,omitempty"`
	Took            *int       `bson:"took,omitempty"             json:"took,omitempty"`
	Results         *[]int     `bson:"results,omitempty"          json:"results,omitempty"`
	Suggestions     *[]int     `bson:"suggestions,omitempty"      json:"suggestions,omitempty"`
	DocCounts       *docCounts `bson:"docCounts,omitempty"        json:"docCounts,omitempty"`
}

type docCounts struct {
	DatasetLandingPage *int `bson:"dataset_landing_page,omitempty"  json:"dataset_landing_page,omitempty"`
	StaticAdhoc        *int `bson:"static_adhoc,omitempty"          json:"static_adhoc,omitempty"`
}

type result struct {
	NumberOfResults *int              `bson:"numberOfResults,omitempty"  json:"numberOfResults,omitempty"`
	Took            *int              `bson:"took,omitempty"             json:"took,omitempty"`
	Results         *[]results        `bson:"results,omitempty"          json:"results,omitempty"`
	Suggestions     *[]string         `bson:"suggestions,omitempty"      json:"suggestions,omitempty"`
	DocCounts       *map[string]int64 `bson:"docCounts,omitempty"        json:"docCounts,omitempty"`
	Paginator       *paginator        `bson:"paginator,omitempty"        json:"paginator,omitempty"`
	// NOTE: DocCounts has not been seen in any data read and its type has been derived from a chat with Jon who
	//       provided the following link to its java code from where the type now used has been derived:
	// https://github.com/ONSdigital/babbage/blob/fdfaa41528649c3dfb1165634330bfcb2c535fed/src/main/java/com/github/onsdigital/babbage/search/model/SearchResult.java#L43
	SortBy *string `bson:"sortBy,omitempty"           json:"sortBy,omitempty"`
}

type results struct {
	ResultType  *string            `bson:"_type,omitempty"        json:"_type,omitempty"`
	Description *resultDescription `bson:"description,omitempty"  json:"description,omitempty"`
	SearchBoost *[]string          `bson:"searchBoost,omitempty"  json:"searchBoost,omitempty"`
	Type        *string            `bson:"type,omitempty"         json:"type,omitempty"`
	URI         *string            `bson:"uri,omitempty"          json:"uri,omitempty"`
	Topics      *[]string          `bson:"topics,omitempty"       json:"topics,omitempty"`
	//	Topics      *[]resultsTopics   `bson:"topics,omitempty"       json:"topics,omitempty"`
	// for above, see link:
	// https://github.com/ONSdigital/zebedee/blob/957fb22f141546ae056f76bf1dbc23f4df8407de/zebedee-reader/src/main/java/com/github/onsdigital/zebedee/search/model/SearchDocument.java#L16
	// and then:
	// https://docs.oracle.com/javase/7/docs/api/java/net/URI.html
	// Jon suggests java struct: (but the above works ..)
	/*
		public URI(String scheme,
		   String userInfo,
		   String host,
		   int port,
		   String path,
		   String query,
		   String fragment)
	*/
}

type paginator struct {
	NumberOfPages *int   `bson:"numberOfPages,omitempty"  json:"numberOfPages,omitempty"`
	CurrentPage   *int   `bson:"currentPage,omitempty"    json:"currentPage,omitempty"`
	Start         *int   `bson:"start,omitempty"          json:"start,omitempty"`
	End           *int   `bson:"end,omitempty"            json:"end,omitempty"`
	Pages         *[]int `bson:"pages,omitempty"          json:"pages,omitempty"`
}

/*type resultsTopics struct {
	UserInfo *string `bson:"userInfo,omitempty"  json:"userInfo,omitempty"`
	Host     *string `bson:"host,omitempty"      json:"host,omitempty"`
	Port     *int    `bson:"port,omitempty"      json:"port,omitempty"`
	Path     *string `bson:"path,omitempty"      json:"path,omitempty"`
	Query    *string `bson:"query,omitempty"     json:"query,omitempty"`
	Fragment *string `bson:"fragment,omitempty"  json:"fragment,omitempty"`
}*/

type resultDescription struct {
	Summary            *string         `bson:"summary,omitempty"            json:"summary,omitempty"`
	NextRelease        *string         `bson:"nextRelease,omitempty"        json:"nextRelease,omitempty"`
	Keywords           *[]string       `bson:"keywords,omitempty"           json:"keywords,omitempty"`
	ReleaseDate        *string         `bson:"releaseDate,omitempty"        json:"releaseDate,omitempty"`
	Edition            *string         `bson:"edition,omitempty"            json:"edition,omitempty"`
	Language           *string         `bson:"language,omitempty"           json:"language,omitempty"`
	DatasetID          *string         `bson:"datasetId,omitempty"          json:"datasetId,omitempty"`
	MetaCmd            *string         `bson:"metaCmd,omitempty"            json:"metaCmd,omitempty"`
	Finalised          *bool           `bson:"finalised,omitempty"          json:"finalised,omitempty"`
	Source             *string         `bson:"source,omitempty"             json:"source,omitempty"`
	Published          *bool           `bson:"published,omitempty"          json:"published,omitempty"`
	Title              *string         `bson:"title,omitempty"              json:"title,omitempty"`
	MetaDescription    *string         `bson:"metaDescription,omitempty"    json:"metaDescription,omitempty"`
	NationalStatistic  *bool           `bson:"nationalStatistic,omitempty"  json:"nationalStatistic,omitempty"`
	Abstract           *string         `bson:"_abstract,omitempty"          json:"_abstract,omitempty"`
	LatestRelease      *bool           `bson:"latestRelease,omitempty"      json:"latestRelease,omitempty"`
	Reference          *string         `bson:"reference,omitempty"          json:"reference,omitempty"`
	Unit               *string         `bson:"unit,omitempty"               json:"unit,omitempty"`
	Headline1          *string         `bson:"headline1,omitempty"          json:"headline1,omitempty"`
	Headline2          *string         `bson:"headline2,omitempty"          json:"headline2,omitempty"`
	Contacts           *ContactDetails `bson:"contact,omitempty"            json:"contact,omitempty"`
	Headline3          *string         `bson:"headline3,omitempty"          json:"headline3,omitempty"`
	ProvisionalDate    *string         `bson:"provisionalDate,omitempty"    json:"provisionalDate,omitempty"`
	Cancelled          *bool           `bson:"cancelled,omitempty"          json:"cancelled,omitempty"`
	PreUnit            *string         `bson:"preUnit,omitempty"            json:"preUnit,omitempty"`
	CancellationNotice *[]string       `bson:"cancellationNotice,omitempty" json:"cancellationNotice,omitempty"`
	Authors            *[]string       `bson:"authors,omitempty"            json:"authors,omitempty"`
}

type staticPageResponse struct {
	Charts      *[]charts     `bson:"charts,omitempty"       json:"charts,omitempty"`
	Tables      *[]tables     `bson:"tables,omitempty"       json:"tables,omitempty"`
	Images      *[]images     `bson:"images,omitempty"       json:"images,omitempty"`
	Equations   *[]equations  `bson:"equations,omitempty"    json:"equations,omitempty"`
	Downloads   *[]downloads  `bson:"downloads,omitempty"    json:"downloads,omitempty"`
	Markdown    *[]string     `bson:"markdown,omitempty"     json:"markdown,omitempty"`
	Links       *[]links      `bson:"links,omitempty"        json:"links,omitempty"`
	Type        *string       `bson:"type,omitempty"         json:"type,omitempty"`
	URI         *string       `bson:"uri,omitempty"          json:"uri,omitempty"`
	Description *cDescription `bson:"description,omitempty"  json:"description,omitempty"`
}

type staticAdhocResponse struct {
	Downloads   *[]downloads  `bson:"downloads,omitempty"    json:"downloads,omitempty"`
	Markdown    *[]string     `bson:"markdown,omitempty"     json:"markdown,omitempty"`
	Links       *[]links      `bson:"links,omitempty"        json:"links,omitempty"`
	Type        *string       `bson:"type,omitempty"         json:"type,omitempty"`
	URI         *string       `bson:"uri,omitempty"          json:"uri,omitempty"`
	Description *cDescription `bson:"description,omitempty"  json:"description,omitempty"`
}

type referenceTablesResponse struct {
	Migrated           *bool                 `bson:"migrated,omitempty"            json:"migrated,omitempty"`
	Downloads          *[]downloads          `bson:"downloads,omitempty"           json:"downloads,omitempty"`
	RelatedDocuments   *[]relatedDocuments   `bson:"relatedDocuments,omitempty"    json:"relatedDocuments,omitempty"`
	RelatedMethodology *[]relatedMethodology `bson:"relatedMethodology,omitempty"  json:"relatedMethodology,omitempty"`
	Type               *string               `bson:"type,omitempty"                json:"type,omitempty"`
	URI                *string               `bson:"uri,omitempty"                 json:"uri,omitempty"`
	Description        *cDescription         `bson:"description,omitempty"         json:"description,omitempty"`
}

type compendiumChapterResponse struct {
	PdfTable                  *[]pdfTable                  `bson:"pdfTable,omitempty"                   json:"pdfTable,omitempty"`
	Sections                  *[]sections                  `bson:"sections,omitempty"                   json:"sections,omitempty"`
	Accordion                 *[]accordion                 `bson:"accordion,omitempty"                  json:"accordion,omitempty"`
	RelatedData               *[]relatedData               `bson:"relatedData,omitempty"                json:"relatedData,omitempty"`
	RelatedDocuments          *[]relatedDocuments          `bson:"relatedDocuments,omitempty"           json:"relatedDocuments,omitempty"`
	Charts                    *[]charts                    `bson:"charts,omitempty"                     json:"charts,omitempty"`
	Tables                    *[]tables                    `bson:"tables,omitempty"                     json:"tables,omitempty"`
	Images                    *[]images                    `bson:"images,omitempty"                     json:"images,omitempty"`
	Equations                 *[]equations                 `bson:"equations,omitempty"                  json:"equations,omitempty"`
	Links                     *[]links                     `bson:"links,omitempty"                      json:"links,omitempty"`
	Alerts                    *[]alerts                    `bson:"alerts,omitempty"                     json:"alerts,omitempty"`
	RelatedMethodology        *[]relatedMethodology        `bson:"relatedMethodology,omitempty"         json:"relatedMethodology,omitempty"`
	RelatedMethodologyArticle *[]relatedMethodologyArticle `bson:"relatedMethodologyArticle,omitempty"  json:"relatedMethodologyArticle,omitempty"`
	Versions                  *[]versions                  `bson:"versions,omitempty"                   json:"versions,omitempty"`
	Type                      *string                      `bson:"type,omitempty"                       json:"type,omitempty"`
	URI                       *string                      `bson:"uri,omitempty"                        json:"uri,omitempty"`
	Description               *cDescription                `bson:"description,omitempty"                json:"description,omitempty"`
}

type staticLandingPageResponse struct {
	Sections    *[]sections   `bson:"sections,omitempty"     json:"sections,omitempty"`
	Markdown    *[]string     `bson:"markdown,omitempty"     json:"markdown,omitempty"`
	Links       *[]links      `bson:"links,omitempty"        json:"links,omitempty"`
	Type        *string       `bson:"type,omitempty"         json:"type,omitempty"`
	URI         *string       `bson:"uri,omitempty"          json:"uri,omitempty"`
	Description *cDescription `bson:"description,omitempty"  json:"description,omitempty"`
}

type staticArticleResponse struct {
	Links       *[]links      `bson:"links,omitempty"        json:"links,omitempty"`
	Downloads   *[]downloads  `bson:"downloads,omitempty"    json:"downloads,omitempty"`
	Sections    *[]sections   `bson:"sections,omitempty"     json:"sections,omitempty"`
	Accordion   *[]accordion  `bson:"accordion,omitempty"    json:"accordion,omitempty"`
	Charts      *[]charts     `bson:"charts,omitempty"       json:"charts,omitempty"`
	Tables      *[]tables     `bson:"tables,omitempty"       json:"tables,omitempty"`
	Images      *[]images     `bson:"images,omitempty"       json:"images,omitempty"`
	Equations   *[]equations  `bson:"equations,omitempty"    json:"equations,omitempty"`
	Alerts      *[]alerts     `bson:"alerts,omitempty"       json:"alerts,omitempty"`
	Type        *string       `bson:"type,omitempty"         json:"type,omitempty"`
	URI         *string       `bson:"uri,omitempty"          json:"uri,omitempty"`
	Description *cDescription `bson:"description,omitempty"  json:"description,omitempty"`
}

type datasetResponse struct {
	Downloads          *[]downloads          `bson:"downloads,omitempty"           json:"downloads,omitempty"`
	SupplementaryFiles *[]supplementaryFiles `bson:"supplementaryFiles,omitempty"  json:"supplementaryFiles,omitempty"`
	Versions           *[]versions           `bson:"versions,omitempty"            json:"versions,omitempty"`
	Type               *string               `bson:"type,omitempty"                json:"type,omitempty"`
	URI                *string               `bson:"uri,omitempty"                 json:"uri,omitempty"`
	Description        *cDescription         `bson:"description,omitempty"         json:"description,omitempty"`
}

type timeseriesDatasetResponse struct {
	Downloads          *[]downloads          `bson:"downloads,omitempty"           json:"downloads,omitempty"`
	SupplementaryFiles *[]supplementaryFiles `bson:"supplementaryFiles,omitempty"  json:"supplementaryFiles,omitempty"`
	Versions           *[]versions           `bson:"versions,omitempty"            json:"versions,omitempty"`
	Type               *string               `bson:"type,omitempty"                json:"type,omitempty"`
	URI                *string               `bson:"uri,omitempty"                 json:"uri,omitempty"`
	Description        *cDescription         `bson:"description,omitempty"         json:"description,omitempty"`
}

type supplementaryFiles struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	File  *string `bson:"file,omitempty"   json:"file,omitempty"`
}

type modifications struct {
	RowsExcluded  *[]int `bson:"rowsExcluded,omitempty"   json:"rowsExcluded,omitempty"`
	HeaderRows    *[]int `bson:"headerRows,omitempty"     json:"headerRows,omitempty"`
	HeaderColumns *[]int `bson:"headerColumns,omitempty"  json:"headerColumns,omitempty"`
}

type files struct {
	Type     *string `bson:"type,omitempty"      json:"type,omitempty"`
	Filename *string `bson:"filename,omitempty"  json:"filename,omitempty"`
	FileType *string `bson:"fileType,omitempty"  json:"fileType,omitempty"`
}

type annotations struct {
	ID          *string  `bson:"id,omitempty"           json:"id,omitempty"`
	X           *string  `bson:"x,omitempty"            json:"x,omitempty"`
	Y           *string  `bson:"y,omitempty"            json:"y,omitempty"`
	Title       *string  `bson:"title,omitempty"        json:"title,omitempty"`
	Orientation *string  `bson:"orientation,omitempty"  json:"orientation,omitempty"`
	IsHidden    *bool    `bson:"isHidden,omitempty"     json:"isHidden,omitempty"`
	IsPlotline  *bool    `bson:"isPlotline,omitempty"   json:"isPlotline,omitempty"`
	IsPlotband  *bool    `bson:"isPlotband,omitempty"   json:"isPlotband,omitempty"`
	BandWidth   *string  `bson:"bandWidth,omitempty"    json:"bandWidth,omitempty"`
	Width       *string  `bson:"width,omitempty"        json:"width,omitempty"`
	Height      *string  `bson:"height,omitempty"       json:"height,omitempty"`
	Devices     *devices `bson:"devices,omitempty"      json:"devices,omitempty"`
}

type devices struct {
	Sm *deviceType `bson:"sm,omitempty"  json:"sm,omitempty"`
	Md *deviceType `bson:"md,omitempty"  json:"md,omitempty"`
	Lg *deviceType `bson:"lg,omitempty"  json:"lg,omitempty"`
}

type deviceType struct {
	AspectRatio   *string `bson:"aspectRatio,omitempty"    json:"aspectRatio,omitempty"`
	LabelInterval *string `bson:"labelInterval,omitempty"  json:"labelInterval,omitempty"`
	IsHidden      *bool   `bson:"isHidden,omitempty"       json:"isHidden,omitempty"`
	XAxisLabel    *string `bson:"x,omitempty"              json:"x,omitempty"`
	Y             *string `bson:"y,omitempty"              json:"y,omitempty"`
}

type years struct {
	Date          *string `bson:"date,omitempty"           json:"date,omitempty"`
	Value         *string `bson:"value,omitempty"          json:"value,omitempty"`
	Label         *string `bson:"label,omitempty"          json:"label,omitempty"`
	Year          *string `bson:"year,omitempty"           json:"year,omitempty"`
	Month         *string `bson:"month,omitempty"          json:"month,omitempty"`
	Quarter       *string `bson:"quarter,omitempty"        json:"quarter,omitempty"`
	SourceDataset *string `bson:"sourceDataset,omitempty"  json:"sourceDataset,omitempty"`
	UpdateDate    *string `bson:"updateDate,omitempty"     json:"updateDate,omitempty"`
}

type quarters struct {
	Date          *string `bson:"date,omitempty"           json:"date,omitempty"`
	Value         *string `bson:"value,omitempty"          json:"value,omitempty"`
	Label         *string `bson:"label,omitempty"          json:"label,omitempty"`
	Year          *string `bson:"year,omitempty"           json:"year,omitempty"`
	Month         *string `bson:"month,omitempty"          json:"month,omitempty"`
	Quarter       *string `bson:"quarter,omitempty"        json:"quarter,omitempty"`
	SourceDataset *string `bson:"sourceDataset,omitempty"  json:"sourceDataset,omitempty"`
	UpdateDate    *string `bson:"updateDate,omitempty"     json:"updateDate,omitempty"`
}

type months struct {
	Date          *string `bson:"date,omitempty"           json:"date,omitempty"`
	Value         *string `bson:"value,omitempty"          json:"value,omitempty"`
	Label         *string `bson:"label,omitempty"          json:"label,omitempty"`
	Year          *string `bson:"year,omitempty"           json:"year,omitempty"`
	Month         *string `bson:"month,omitempty"          json:"month,omitempty"`
	Quarter       *string `bson:"quarter,omitempty"        json:"quarter,omitempty"`
	SourceDataset *string `bson:"sourceDataset,omitempty"  json:"sourceDataset,omitempty"`
	UpdateDate    *string `bson:"updateDate,omitempty"     json:"updateDate,omitempty"`
}
type downloads struct {
	Title           *string `bson:"title,omitempty"            json:"title,omitempty"`
	File            *string `bson:"file,omitempty"             json:"file,omitempty"`
	FileDescription *string `bson:"fileDescription,omitempty"  json:"fileDescription,omitempty"`
}
type relatedArticles struct {
	Article *string `bson:"article,omitempty"  json:"article,omitempty"`
}

type relatedBulletins struct {
	Bulletin *string `bson:"bulletin,omitempty"  json:"bulletin,omitempty"`
}

type pdfTable struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	File  *string `bson:"file,omitempty"   json:"file,omitempty"`
}

type pdfDownloads struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	File  *string `bson:"file,omitempty"   json:"file,omitempty"`
}

type sections struct {
	Summary  *string `bson:"summary,omitempty"   json:"summary,omitempty"`
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
	Markdown *string `bson:"markdown,omitempty"  json:"markdown,omitempty"`
}

type accordion struct {
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	Markdown *string `bson:"markdown,omitempty"  json:"markdown,omitempty"`
}

type chapters struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

type relatedData struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

type section struct {
	Markdown *string `bson:"markdown,omitempty"  json:"markdown,omitempty"`
}

// relatedFilterableDatasets is the sub page links
type relatedFilterableDatasets struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// relatedDatasets is the sub page links
type relatedDatasets struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
}

type sourceDatasets struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// relatedDocuments is the sub page links
type relatedDocuments struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
}

// datasets is the sub page links
type datasets struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

type charts struct {
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	Filename *string `bson:"filename,omitempty"  json:"filename,omitempty"`
	Version  *string `bson:"version,omitempty"   json:"version,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
}

type tables struct {
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	Filename *string `bson:"filename,omitempty"  json:"filename,omitempty"`
	Version  *string `bson:"version,omitempty"   json:"version,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
}

type images struct {
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	Filename *string `bson:"filename,omitempty"  json:"filename,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
}

type equations struct {
	Title    *string `bson:"title,omitempty"     json:"title,omitempty"`
	Filename *string `bson:"filename,omitempty"  json:"filename,omitempty"`
	URI      *string `bson:"uri,omitempty"       json:"uri,omitempty"`
}

// links is the sub page links
type links struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
}

// alerts is the sub page links
type alerts struct {
	Date     *string `bson:"date,omitempty"      json:"date,omitempty"`
	Markdown *string `bson:"markdown,omitempty"  json:"markdown,omitempty"`
	Type     *string `bson:"type,omitempty"      json:"type,omitempty"`
}

// relatedMethodology is the sub page links
type relatedMethodology struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
}

// relatedMethodologyArticle is the sub page links
type relatedMethodologyArticle struct {
	Title *string `bson:"title,omitempty"  json:"title,omitempty"`
	URI   *string `bson:"uri,omitempty"    json:"uri,omitempty"`
}

type versions struct {
	URI              *string `bson:"uri,omitempty"               json:"uri,omitempty"`
	UpdateDate       *string `bson:"updateDate,omitempty"        json:"updateDate,omitempty"`
	CorrectionNotice *string `bson:"correctionNotice,omitempty"  json:"correctionNotice,omitempty"`
	Label            *string `bson:"label,omitempty"             json:"label,omitempty"`
}

// ctopics is the sub page links
type ctopics struct {
	URI *string `bson:"uri,omitempty"  json:"uri,omitempty"`
}

// cDescription is the page description
type cDescription struct {
	Finalised          *bool           `bson:"finalised,omitempty"           json:"finalised,omitempty"`
	Title              *string         `bson:"title,omitempty"               json:"title,omitempty"`
	Summary            *string         `bson:"summary,omitempty"             json:"summary,omitempty"`
	Keywords           *[]string       `bson:"keywords,omitempty"            json:"keywords,omitempty"`
	MetaDescription    *string         `bson:"metaDescription,omitempty"     json:"metaDescription,omitempty"`
	NationalStatistic  *bool           `bson:"nationalStatistic,omitempty"   json:"nationalStatistic,omitempty"`
	LatestRelease      *bool           `bson:"latestRelease,omitempty"       json:"latestRelease,omitempty"`
	Contacts           *ContactDetails `bson:"contact,omitempty"             json:"contact,omitempty"`
	ReleaseDate        *string         `bson:"releaseDate,omitempty"         json:"releaseDate,omitempty"`
	NextRelease        *string         `bson:"nextRelease,omitempty"         json:"nextRelease,omitempty"`
	Language           *string         `bson:"language,omitempty"            json:"language,omitempty"`
	Edition            *string         `bson:"edition,omitempty"             json:"edition,omitempty"`
	DatasetID          *string         `bson:"datasetId,omitempty"           json:"datasetId,omitempty"`
	DatasetURI         *string         `bson:"datasetUri,omitempty"          json:"datasetUri,omitempty"`
	Cdid               *string         `bson:"cdid,omitempty"                json:"cdid,omitempty"`
	Abstract           *string         `bson:"_abstract,omitempty"           json:"_abstract,omitempty"`
	Authors            *[]string       `bson:"authors,omitempty"             json:"authors,omitempty"`
	Headline           *string         `bson:"headline,omitempty"            json:"headline,omitempty"`
	Headline1          *string         `bson:"headline1,omitempty"           json:"headline1,omitempty"`
	Headline2          *string         `bson:"headline2,omitempty"           json:"headline2,omitempty"`
	Headline3          *string         `bson:"headline3,omitempty"           json:"headline3,omitempty"`
	Unit               *string         `bson:"unit,omitempty"                json:"unit,omitempty"`
	PreUnit            *string         `bson:"preUnit,omitempty"             json:"preUnit,omitempty"`
	Source             *string         `bson:"source,omitempty"              json:"source,omitempty"`
	Reference          *string         `bson:"reference,omitempty"           json:"reference,omitempty"`
	Cancelled          *bool           `bson:"cancelled,omitempty"           json:"cancelled,omitempty"`
	CancellationNotice *[]string       `bson:"cancellationNotice,omitempty"  json:"cancellationNotice,omitempty"`
	Published          *bool           `bson:"published,omitempty"           json:"published,omitempty"`
	ProvisionalDate    *string         `bson:"provisionalDate,omitempty"     json:"provisionalDate,omitempty"`
	MonthLabelStyle    *string         `bson:"monthLabelStyle,omitempty"     json:"monthLabelStyle,omitempty"`
	Date               *string         `bson:"date,omitempty"                json:"date,omitempty"`
	Number             *string         `bson:"number,omitempty"              json:"number,omitempty"`
	SurveyName         *string         `bson:"surveyName,omitempty"          json:"surveyName,omitempty"`
	Frequency          *string         `bson:"frequency,omitempty"           json:"frequency,omitempty"`
	Compilation        *string         `bson:"compilation,omitempty"         json:"compilation,omitempty"`
	GeographicCoverage *string         `bson:"geographicCoverage,omitempty"  json:"geographicCoverage,omitempty"`
	MetaCmd            *string         `bson:"metaCmd,omitempty"             json:"metaCmd,omitempty"`
	KeyNote            *string         `bson:"keyNote,omitempty"             json:"keyNote,omitempty"`
	SampleSize         *string         `bson:"sampleSize,omitempty"          json:"sampleSize,omitempty"`
	LastRevised        *string         `bson:"lastRevised,omitempty"         json:"lastRevised,omitempty"`
	VersionLabel       *string         `bson:"versionLabel,omitempty"        json:"versionLabel,omitempty"`
}

// ===

var attemptedGetCount int32

// store info about each page to do broken page report
type pageData struct {
	subSectionIndex int
	pageBroken      bool
	depth           int
	shortURI        string
	parentURI       string
	fieldName       string
}

var listOfPageData []pageData
var listMu sync.Mutex

var releaseRelatedDatasets []string
var releaseRelatedDatasetsMu sync.Mutex

var releaseRelatedDocuments []string
var releaseRelatedDocumentsMu sync.Mutex

func replaceUnicodeWithASCII(b []byte) []byte {
	l := len(b)
	var dst int

	for src := 0; src < l; src++ {
		if b[src] == '\\' {
			if src < l-5 {
				if b[src+1] == 'u' {
					// We have a unicode sequence, and we assume it is 6 characters long
					// get 4 hex characters
					hexstring := string([]byte{b[src+2], b[src+3], b[src+4], b[src+5]})
					num, err := hex.DecodeString(hexstring)
					if err != nil {
						// we can get an error whilst trying to decode an 'equation' type of page that has
						// special encodings to describe the equation drawn on the web page, a page like:
						// https://www.ons.gov.uk/employmentandlabourmarket/peopleinwork/earningsandworkinghours/articles/understandingthegenderpaygapintheuk/2018-01-17/3b240f04/data
						/*fmt.Printf("Error in DecodeString: %v\n", hexstring)
						fmt.Printf("byte 0: %v\n", b[src+2])
						fmt.Printf("byte 1: %v\n", b[src+3])
						fmt.Printf("byte 2: %v\n", b[src+4])
						fmt.Printf("byte 3: %v\n", b[src+5])
						panic(err)*/
						// So, we have to assume that this is an equation and just copy the character ..
						b[dst] = b[src]
					} else {
						if num[1] < 32 && num[1] != 13 && num[1] != 10 {
							fmt.Printf("Control character: %d, pos %d, dst %d\n", num[1], src, dst)
							fmt.Printf("\n")
							fmt.Printf("No error in DecodeString: %v\n", hexstring)
							fmt.Printf("byte 0: %v\n", b[src+2])
							fmt.Printf("byte 1: %v\n", b[src+3])
							fmt.Printf("byte 2: %v\n", b[src+4])
							fmt.Printf("byte 3: %v\n", b[src+5])
							// drop control characters, that should not be in a json string
							dst--
						} else {
							b[dst] = num[1] // get ASCII character
						}
						src += 5 // skip past unicode sequence - the for loop increment makes this an increase of 6
					}
				} else {
					if b[src] < 32 && b[src] != 13 && b[src] != 10 {
						fmt.Printf("Control character: %d\n", b[src])
						fmt.Printf("\n")
						// NOTE: this code is here, for debug as it might be that this code needs
						// further work to drop any not wanted control characters if they cause grief
					}
					b[dst] = b[src]
				}
				dst++
			}
		} else {
			b[dst] = b[src]
			dst++
		}
	}
	return b[0:dst]
}

func removeNotAllowedEscapeCodes(b []byte) []byte {
	l := len(b)
	var dst int

	for src := 0; src < l; src++ {
		if b[src] == '\\' {
			if b[src] == 92 && b[src+1] == 102 {
				fmt.Printf("b[src]: %d\nb[src+1]: %d\n", b[src], b[src+1])
			}
			if src < l-2 {
				if b[src+1] == 'f' {
					fmt.Printf("b[src+1]: %d\n", b[src+1])
					// We have a escape sequence that should not be in the json as it breaks unmarshaling
					src += 1 // skip past escape sequence - the for loop increment makes this an increase of 2
				} else {
					b[dst] = b[src]
					dst++
				}
			}
		} else {
			b[dst] = b[src]
			dst++
		}
	}
	return b[0:dst]
}

// the following is examples of the \u0027  and its change into a single quote:
/*
 {"items":[{"uri":"/economy/economicoutputandproductivity/output/timeseries/k27q"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k222"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k27y"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k22a"}],"datasets":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/topsiproductionandservicesturnover"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/turnoverandordersintheproductionandservicesindustriesdataset"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/topsimanufacturingexportturnover"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/ukmanufacturerssalesbyproductprodcom"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/ukmanufacturerssalesbyproductprodcomintermediateresults2013andfinalresults2012referencetables"}],"statsBulletins":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/bulletins/ukmanufacturerssalesbyproductprodcom/latest"},{"uri":"/businessindustryandtrade/business/businessservices/bulletins/uknonfinancialbusinesseconomy/latest"}],"relatedArticles":[],"relatedMethodology":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/qmis/ukmanufacturerssalesbyproductqmi"},{"uri":"/businessindustryandtrade/business/businessservices/qmis/monthlybusinesssurveyqmi"}],"relatedMethodologyArticle":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/methodologies/ukmanufacturerssalesbyproductprodcom"}],"index":5,"type":"product_page","uri":"/businessindustryandtrade/manufacturingandproductionindustry","description":{"title":"Manufacturing and production industry","summary":"UK manufacturing and other production industries (such as mining and quarrying, energy supply, water supply and waste management), including total UK production output, and
UK manufactures\u0027
 sales by product and industrial division, with EU comparisons.","metaDescription":"UK manufacturing and other production industries (such as mining and quarrying, energy supply, water supply and waste management), including total UK production output, and UK manufactures\u0027 sales by product and industrial division, with EU comparisons.","unit":"","preUnit":"","source":""}}

{"items":[{"uri":"/economy/economicoutputandproductivity/output/timeseries/k27q"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k222"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k27y"},{"uri":"/economy/economicoutputandproductivity/output/timeseries/k22a"}],"datasets":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/topsiproductionandservicesturnover"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/turnoverandordersintheproductionandservicesindustriesdataset"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/topsimanufacturingexportturnover"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/ukmanufacturerssalesbyproductprodcom"},{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/datasets/ukmanufacturerssalesbyproductprodcomintermediateresults2013andfinalresults2012referencetables"}],"statsBulletins":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/bulletins/ukmanufacturerssalesbyproductprodcom/latest"},{"uri":"/businessindustryandtrade/business/businessservices/bulletins/uknonfinancialbusinesseconomy/latest"}],"relatedArticles":[],"relatedMethodology":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/qmis/ukmanufacturerssalesbyproductqmi"},{"uri":"/businessindustryandtrade/business/businessservices/qmis/monthlybusinesssurveyqmi"}],"relatedMethodologyArticle":[{"uri":"/businessindustryandtrade/manufacturingandproductionindustry/methodologies/ukmanufacturerssalesbyproductprodcom"}],"index":5,"type":"product_page","uri":"/businessindustryandtrade/manufacturingandproductionindustry","description":{"title":"Manufacturing and production industry","summary":"UK manufacturing and other production industries (such as mining and quarrying, energy supply, water supply and waste management), including total UK production output, and
UK manufactures'
 sales by product and industrial division, with EU comparisons.","metaDescription":"UK manufacturing and other production industries (such as mining and quarrying, energy supply, water supply and waste management), including total UK production output, and UK manufactures' sales by product and industrial division, with EU comparisons.","unit":"","preUnit":"","source":""}}
*/

func doAndShowDelay71() {
	fmt.Printf("Got a 429, backing off for 71 seconds ..\n")
	for delay := 0; delay < 71; delay++ {
		time.Sleep(1 * time.Second)
		if delay%3 == 0 {
			fmt.Printf("Seconds remaining: %d\n", 71-delay)
		}
	}
}

var homePageCount int32
var taxonomyLandingPageCount int32
var productPageCount int32
var articleCount int32
var articleDownloadCount int32
var bulletinCount int32
var compendiumDataCount int32
var compendiumLandingPageCount int32
var datasetLandingPageCount int32
var staticMethodologyCount int32
var staticMethodologyDownloadCount int32
var staticQmiCount int32
var timeseriesCount int32
var chartCount int32
var tableCount int32
var equationCount int32
var imageCount int32
var releaseCount int32
var listCount int32
var staticPageCount int32
var staticAdhocCount int32
var referenceTablesCount int32
var compendiumChapterCount int32
var staticLandingPageCount int32
var staticArticleCount int32
var datasetCount int32
var timeseriesDatasetCount int32

// store the shortUri and count to prevent processing a page more than once
type safeContentDuplicateCheck struct {
	mu sync.Mutex
	v  map[string]int
}

// Inc increments the counter for the given key.
func (c *safeContentDuplicateCheck) Inc(key string) {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	c.v[key]++
	c.mu.Unlock()
}

// Value returns the current value of the counter for the given key.
func (c *safeContentDuplicateCheck) Value(key string) int {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	return c.v[key]
}

// Length returns the number of unique counter.
func (c *safeContentDuplicateCheck) Length() int {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	return len(c.v)
}

var contentDuplicateCheck = safeContentDuplicateCheck{v: make(map[string]int)}

type safeURICollectionName struct {
	mu sync.Mutex
	v  map[string]string
}

// Store returns the number of unique counter.
// key: shortURI, value: the name of the collection that the URI is stored in
func (c *safeURICollectionName) Store(key, value string) {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	c.v[key] = value
}

// Length returns the number of values storred.
func (c *safeURICollectionName) Length() int {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	return len(c.v)
}

var uriCollectionName = safeURICollectionName{v: make(map[string]string)}

var saveMutex sync.Mutex

func saveContentPageToCollection(collectionJsFile *os.File, id *int32, collectionName string, bodyTextCopy []byte, shortURI string) {
	// The original /data endpoint information read has passed tests so now we
	// write out the original json code together with an extra 'id'

	// NOTE: splitting the content info out to separate collections as per the content 'Type' may not
	//       be whats needed, but its a good start for demonstrating that the content has been
	//       extracted without error.

	if contentDuplicateCheck.Value(shortURI) > 0 {
		// keep incrementing the duplication count (in case its of use at some point)
		contentDuplicateCheck.Inc(shortURI)
		return
	}
	contentDuplicateCheck.Inc(shortURI)
	uriCollectionName.Store(shortURI, collectionName)

	saveMutex.Lock()
	defer saveMutex.Unlock()

	// Increment this collectionName's counter whilst being protected by above mutex
	*id++

	if !cfg.SaveSite {
		return
	}

	_, err := fmt.Fprint(collectionJsFile, "db."+collectionName+".insertOne({")
	check(err)

	// write out an 'id' for this data file
	_, err = fmt.Fprint(collectionJsFile, "\n    \"id\": \""+strconv.Itoa(int(*id))+"\",\n")
	check(err)

	// write out what should be a unique key that can be indexed on ..
	_, err = fmt.Fprint(collectionJsFile, "    \"id_uri\": \""+shortURI+"\",\n")
	check(err)

	// Strip out the first character which is an opening curly brace so that we get a correctly formed
	// java script line
	_, err = collectionJsFile.Write(bodyTextCopy[1:])
	check(err)

	_, err = fmt.Fprint(collectionJsFile, ")\n")
	check(err)
}

func addSections(uriList *[]urilist, field *[]sections, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "sections", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedData(uriList *[]urilist, field *[]relatedData, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedData", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedDocuments(uriList *[]urilist, field *[]relatedDocuments, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedDocuments", parentURI, depth, index})
				}
			}
		}
	}
}

func addCharts(uriList *[]urilist, field *[]charts, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "charts", parentURI, depth, index})
				}
			}
		}
	}
}

func addTables(uriList *[]urilist, field *[]tables, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "tables", parentURI, depth, index})
				}
			}
		}
	}
}

func addImages(uriList *[]urilist, field *[]images, parentURI string, depth int) {
	// We can't read an image, so we don't check that the link is OK
	// If a way could be found to check that a link to a .png or .jpg is OK
	// then some code could be added to do another audit check.
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					if strings.HasSuffix(*info.URI, ".png") || strings.HasSuffix(*info.URI, ".jpg") {
						// This is a picture which we can't check
						fmt.Printf("Picture Parent URI: %s\n", parentURI)
						fmt.Printf("Picture URI: %s\n", *info.URI)
					} else {
						*uriList = append(*uriList, urilist{*info.URI, "images", parentURI, depth, index})
					}
				}
			}
		}
	}
}

func addEquations(uriList *[]urilist, field *[]equations, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "equations", parentURI, depth, index})
				}
			}
		}
	}
}

func addLinks(uriList *[]urilist, field *[]links, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "links", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedMethodology(uriList *[]urilist, field *[]relatedMethodology, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedMethodology", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedMethodologyArticle(uriList *[]urilist, field *[]relatedMethodologyArticle, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedMethodologyArticle", parentURI, depth, index})
				}
			}
		}
	}
}

func addVersions(uriList *[]urilist, field *[]versions, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "versions", parentURI, depth, index})
				}
			}
		}
	}
}

func addTopics(uriList *[]urilist, field *[]ctopics, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "topics", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedDatasets(uriList *[]urilist, field *[]relatedDatasets, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedDatasets", parentURI, depth, index})
				}
			}
		}
	}
}

func addDatasets(uriList *[]urilist, field *[]datasets, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "datasets", parentURI, depth, index})
				}
			}
		}
	}
}

func addChapters(uriList *[]urilist, field *[]chapters, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "chapters", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedFilterableDatasets(uriList *[]urilist, field *[]relatedFilterableDatasets, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					if !strings.HasPrefix(*info.URI, "/datasets/") {
						// This is NOT a 'Choose My Data' page, so add it
						// (we skip CMD pages because they do not have a '/data' suffix indicating
						//  this is a page that can not be processed)
						*uriList = append(*uriList, urilist{*info.URI, "relatedFilterableDatasets", parentURI, depth, index})
					}
				}
			}
		}
	}
}

func addSourceDatasets(uriList *[]urilist, field *[]sourceDatasets, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "sourceDatasets", parentURI, depth, index})
				}
			}
		}
	}
}

func addFeaturedContent(uriList *[]urilist, field *[]FeaturedContentLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "sections", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseSections(uriList *[]urilist, field *[]SubLink, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "sections", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseItems(uriList *[]urilist, field *[]ItemLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "items", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseDatasets(uriList *[]urilist, field *[]DatasetLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "datasets", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseHighlightedLinks(uriList *[]urilist, field *[]HighlightLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "highlightedLinks", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseStatsBulletins(uriList *[]urilist, field *[]StatsBulletLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "statsBulletins", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseRelatedArticles(uriList *[]urilist, field *[]RelatedArticleLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedArticles", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseRelatedMethodology(uriList *[]urilist, field *[]RelatedMethodLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedMethodology", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseRelatedMethodologyArticle(uriList *[]urilist, field *[]RelatedMethodologyArticleLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedMethodologyArticle", parentURI, depth, index})
				}
			}
		}
	}
}

func addDataResponseHighlightedContent(uriList *[]urilist, field *[]HighlightedContentLinks, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "highlightedContent", parentURI, depth, index})
				}
			}
		}
	}
}

func addRelatedResult(uriList *[]urilist, field *[]results, parentURI string, depth int) {
	if field != nil {
		if len(*field) > 0 {
			for index, info := range *field {
				if info.URI != nil {
					*uriList = append(*uriList, urilist{*info.URI, "relatedResult", parentURI, depth, index})
				}
			}
		}
	}
}

func getURIListFromProductPage(containintURI string, data *DataResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	addDataResponseItems(&uriList, data.Items, parentURI, depth)
	addDataResponseDatasets(&uriList, data.Datasets, parentURI, depth)
	addDataResponseHighlightedLinks(&uriList, data.HighlightedLinks, parentURI, depth)
	addDataResponseStatsBulletins(&uriList, data.StatsBulletins, parentURI, depth)
	addDataResponseRelatedArticles(&uriList, data.RelatedArticles, parentURI, depth)
	addDataResponseRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addDataResponseRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)

	return uriList
}

func getURIListFromTaxonomyLandingPage(containintURI string, data *DataResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	addDataResponseSections(&uriList, data.Sections, parentURI, depth)
	addDataResponseHighlightedContent(&uriList, data.HighlightedContent, parentURI, depth)

	return uriList
}

func getURIListFromHomePage(containintURI string, data *HomePageResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	addFeaturedContent(&uriList, data.FeaturedContent, parentURI, depth)

	return uriList
}

func getURIListFromArticle(containintURI string, data *articleResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	addSections(&uriList, data.Sections, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromArticleDownload(containintURI string, data *articleDownloadResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromBulletin(containintURI string, data *bulletinResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addSections(&uriList, data.Sections, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromCompendiumData(containintURI string, data *compendiumDataResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromCompendiumLandingPage(containintURI string, data *compendiumLandingPageResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addDatasets(&uriList, data.Datasets, parentURI, depth)
	addChapters(&uriList, data.Chapters, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromDatasetLandingPage(containintURI string, data *datasetLandingPageResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedFilterableDatasets(&uriList, data.RelatedFilterableDatasets, parentURI, depth)
	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addDatasets(&uriList, data.Datasets, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromStaticMethodology(containintURI string, data *staticMethodologyResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addSections(&uriList, data.Sections, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromStaticMethodologyDownload(containintURI string, data *staticMethodologyDownloadResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromStaticQmi(containintURI string, data *staticQmiResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromTimeseries(containintURI string, data *timeseriesResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addSourceDatasets(&uriList, data.SourceDatasets, parentURI, depth)
	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)
	addTopics(&uriList, data.Topics, parentURI, depth)

	return uriList
}

func getURIListFromRelease(containintURI string, data *releaseResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedDatasets(&uriList, data.RelatedDatasets, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)

	return uriList
}

func getURIListFromList(containintURI string, data *listResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedResult(&uriList, data.Result.Results, parentURI, depth)

	return uriList
}

func getURIListFromStaticPage(containintURI string, data *staticPageResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addLinks(&uriList, data.Links, parentURI, depth)

	return uriList
}

func getURIListFromStaticAdhoc(containintURI string, data *staticAdhocResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addLinks(&uriList, data.Links, parentURI, depth)

	return uriList
}

func getURIListFromReferenceTables(containintURI string, data *referenceTablesResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)

	return uriList
}

func getURIListFromCompendiumChapter(containintURI string, data *compendiumChapterResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addSections(&uriList, data.Sections, parentURI, depth)
	addRelatedData(&uriList, data.RelatedData, parentURI, depth)
	addRelatedDocuments(&uriList, data.RelatedDocuments, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)
	addRelatedMethodology(&uriList, data.RelatedMethodology, parentURI, depth)
	addRelatedMethodologyArticle(&uriList, data.RelatedMethodologyArticle, parentURI, depth)
	addVersions(&uriList, data.Versions, parentURI, depth)

	return uriList
}

func getURIListFromStaticLandingPage(containintURI string, data *staticLandingPageResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addSections(&uriList, data.Sections, parentURI, depth)
	addLinks(&uriList, data.Links, parentURI, depth)

	return uriList
}

func getURIListFromStaticArticle(containintURI string, data *staticArticleResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addLinks(&uriList, data.Links, parentURI, depth)
	addSections(&uriList, data.Sections, parentURI, depth)
	addCharts(&uriList, data.Charts, parentURI, depth)
	addTables(&uriList, data.Tables, parentURI, depth)
	addImages(&uriList, data.Images, parentURI, depth)
	addEquations(&uriList, data.Equations, parentURI, depth)

	return uriList
}

func getURIListFromDataset(containintURI string, data *datasetResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addVersions(&uriList, data.Versions, parentURI, depth)

	return uriList
}

func getURIListFromTimeseriesDataset(containintURI string, data *timeseriesDatasetResponse, parentURI string, depth int) []urilist {
	var uriList []urilist

	if cfg.OnlyFirstFullDepth {
		return uriList
	}

	addVersions(&uriList, data.Versions, parentURI, depth)

	return uriList
}

var depth int = 1

type urilist struct {
	uri       string
	field     string
	parentURI string
	depth     int
	index     int
}

var dataRead int64

func unmarshalFail(uri string, err error, location int) {
	fmt.Printf("fullURI: %s\n", uri)
	fmt.Println(err)
	fmt.Printf("getPageData: json.Unmarshal failed %v\n", location)
	os.Exit(100)
}

func marshalFail(uri string, err error, location int) {
	fmt.Printf("fullURI: %s\n", uri)
	fmt.Println(err)
	fmt.Printf("getPageData: json.Marshal failed %v\n", location)
	os.Exit(101)
}

func checkMarshaling(fullURI string, err error, location int, payload *[]byte, fixedJSON *[]byte, structName string) {
	if err != nil {
		marshalFail(fullURI, err, location)
	}
	fixedPayloadJSON := replaceUnicodeWithASCII(*payload)

	// This effectively checks that the struct 'structName' has all the fields needed ..
	// the 'payLoad' should equal the 'fixedJSON' .. if not structName needs adjusting
	if !bytes.Equal(fixedPayloadJSON, *fixedJSON) {
		fmt.Printf("Processing page: %s\n", fullURI)
		fmt.Printf("Unmarshal / Marshal mismatch - %v.\nInspect the saved .json files and fix struct %s\n", location, structName)
		_, err = fmt.Fprintf(bodyTextFile, "%s\n", fixedJSON)
		check(err)
		_, err = fmt.Fprintf(checkFile, "%s\n", fixedPayloadJSON)
		check(err)
		os.Exit(102)
	}
}

func checkMarshalingDeepEqual(fullURI string, err error, location int, payload *[]byte, fixedJSON *[]byte, structName string) {
	if err != nil {
		marshalFail(fullURI, err, location)
	}
	fixedPayloadJSON := replaceUnicodeWithASCII(*payload)

	// This effectively checks that the struct 'structName' has all the fields needed ..
	// the 'payLoad' should equal the 'fixedJSON' .. if not structName needs adjusting
	if !bytes.Equal(fixedPayloadJSON, *fixedJSON) {
		// The binary comparison will typically fail for struct 'chartResponse'
		// because it contains map[string]string which after unmarshaling and marshaling ..
		// items in the maps may not in the same order.

		// So, we do a an unraveling of the binary JSON to lines of text, sort and then compare ..

		var prettyJSON1 bytes.Buffer
		err = json.Indent(&prettyJSON1, fixedPayloadJSON, "", "    ")
		check(err) // should nt get an error, but just in case

		var prettyJSON2 bytes.Buffer
		err = json.Indent(&prettyJSON2, *fixedJSON, "", "    ")
		check(err) // should not get an error, but just in case

		line1 := strings.Split(prettyJSON1.String(), "\n")
		line2 := strings.Split(prettyJSON2.String(), "\n")

		sort.Strings(line1)
		sort.Strings(line2)

		// maps don't have their fields sorted which results in otherwise equal lines having and not having commas
		// on the end of them, so to allow the DeepEqual below to work, the commas on the ends of the lines need removing.
		for i := 0; i < len(line1); i++ {
			line1[i] = strings.TrimSuffix(line1[i], ",")
		}
		for i := 0; i < len(line2); i++ {
			line2[i] = strings.TrimSuffix(line2[i], ",")
		}

		if !reflect.DeepEqual(line1, line2) {
			fmt.Printf("DeepEqual comparison failed\n")
			fmt.Printf("Processing page: %s\n", fullURI)
			fmt.Printf("Unmarshal / Marshal mismatch(2) - %v.\nInspect the saved .json files and fix struct %s\n", location, structName)
			fmt.Printf("It helps to open these files in vscode and right click in file, select format Docuemnt\n")
			fmt.Printf(" and then save each document and then do a file comparison in an App like meld.")
			// NOTE: In the files, ignore the '&' character at the begining of one of the lines as this is just
			//       a result of one of the unmarshaled JSON lines being a pointer.
			_, err = fmt.Fprintf(checkFile, "%s\n", fixedPayloadJSON)
			check(err)
			_, err = fmt.Fprintf(checkFile, "%s\n", line1)
			check(err)
			_, err = fmt.Fprintf(bodyTextFile, "%s\n", fixedJSON)
			check(err)
			_, err = fmt.Fprintf(bodyTextFile, "%s\n", line2)
			check(err)
			os.Exit(103)
		}
	}
}

func saveReleaseInfoToCSV(shortURI string, data *releaseResponse) {
	fieldDatasets := data.RelatedDatasets

	if fieldDatasets != nil {
		releaseAndDate := shortURI + ","
		if data.Description.ReleaseDate != nil {
			releaseAndDate += *data.Description.ReleaseDate
		} else {
			releaseAndDate += "NO Date"
		}
		releaseAndDate += ","

		if len(*fieldDatasets) > 0 {
			for _, info := range *fieldDatasets {
				if info.URI != nil {
					infoToSave := releaseAndDate + *info.URI

					releaseRelatedDatasetsMu.Lock()
					releaseRelatedDatasets = append(releaseRelatedDatasets, infoToSave)
					releaseRelatedDatasetsMu.Unlock()
				}
			}
		}
	}

	fieldDocuments := data.RelatedDocuments

	if fieldDocuments != nil {
		releaseAndDate := shortURI + ","
		if data.Description.ReleaseDate != nil {
			releaseAndDate += *data.Description.ReleaseDate
		} else {
			releaseAndDate += "NO Date"
		}
		releaseAndDate += ","

		if len(*fieldDocuments) > 0 {
			for _, info := range *fieldDocuments {
				if info.URI != nil {
					infoToSave := releaseAndDate + *info.URI

					releaseRelatedDocumentsMu.Lock()
					releaseRelatedDocuments = append(releaseRelatedDocuments, infoToSave)
					releaseRelatedDocumentsMu.Unlock()
				}
			}
		}
	}
}

func getPageData(shortURI string, fieldName string, parentURI string, index int, depth int) (int, []urilist) {
	// Create a list of URIs
	var URIList []urilist

	if cfg.FullDepth {
		if contentDuplicateCheck.Value(shortURI) > 0 {
			// strange we've seen this link before and filtering elsewhere did not catch it.
			return 503, URIList
		}
	}

	// Add prefix and '/data' to shortURI name
	var fullURI = "https://www.production.onsdigital.co.uk" + shortURI + "/data"

	atomic.AddInt32(&attemptedGetCount, 1)
	if cfg.PlayNice {
		// a little delay to play nice with ONS site and 'hopefully' not have cloudflare 'reset' the connection
		time.Sleep(50 * time.Millisecond)
	}
	response, err := http.Get(fullURI)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("getPageData: http.Get(fullURI) failed\n")
		fmt.Printf("We now fabricate the response code to a 429 to instigate a retry after a delay 2\n")
		return 429, URIList
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		if response.StatusCode != 404 {
			if response.StatusCode != 429 {
				// a 503 is being seen at this point .. (it could be some other error, but whatever it is we do error action)
				fmt.Printf("\nERROR on ONS website /data field: %v\n\n", response.StatusCode)
				fmt.Printf("URI does not exist:  %v\n", fullURI)

				listMu.Lock()
				listOfPageData = append(listOfPageData, pageData{
					subSectionIndex: index,
					pageBroken:      true,
					depth:           depth,
					shortURI:        shortURI,
					parentURI:       parentURI,
					fieldName:       fieldName,
				})
				listMu.Unlock()
			} else {
				fmt.Printf("\nToo many requests\n")
				// caller will call this function again for a 429
			}
		}
		return response.StatusCode, URIList
	}
	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("getPageData: RealAll failed\n")
		os.Exit(7)
	}

	atomic.AddInt64(&dataRead, int64(len(bodyText)))

	// Take a copy into another block of memory before the call to replaceUnicodeWithASCII()
	// strips out the unicode characters.
	// .. thus retaining any unicode to write back out after checks made.
	var bodyTextCopy []byte = make([]byte, len(bodyText))
	copy(bodyTextCopy, bodyText)

	fixJSON := replaceUnicodeWithASCII(bodyText)
	fixedJSON := removeNotAllowedEscapeCodes(fixJSON)

	var shape pageShape
	// Unmarshal body bytes to model
	if err := json.Unmarshal(fixedJSON, &shape); err != nil {
		fmt.Println(err)
		fmt.Printf("getPageData: json.Unmarshal failed 1\n")
		// we can get here from:
		// /employmentandlabourmarket/peopleinwork/workplacepensions#publications
		// where the '#publications' jumps one some way into a page that has already been processed
		// (though #'s are now caught elsewhere)
		//
		// OR
		//    from:
		//    https://www.ons.gov.uk/search?q=interactivetool
		//    which is a valid page, but has no data structure to extract info from
		if shortURI == "/search?q=interactivetool" {
			// NOTE: the number of exception pages may need to grow
			// say the page is unavailable
			return 503, URIList
		}
		fmt.Printf("Unknown problem on page: %s\n", fullURI)
		fmt.Printf("shortURI: %s\n", shortURI)
		os.Exit(8)
	}

	// Decode each page into a specific structure according to the 'Type' of the page ..
	// NOTE: This is done to ensure that the structure definitions are fully defined to read ALL
	//       the info in the /data endpoint.
	switch *shape.Type {
	case homePageCollectionName:
		var data HomePageResponse

		// sanity check the page has some fields for later use

		// Unmarshal body bytes to model
		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 1)
		}

		// Marshal provided model
		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 1, &payload, &fixedJSON, "HomePageResponse")

		saveContentPageToCollection(homePageJsFile, &homePageCount, homePageCollectionName, bodyTextCopy, shortURI)
		URIList = getURIListFromHomePage(fullURI, &data, shortURI, depth)

		// The 'home_page' has no links to the rest of the site .. so we seed the crawl ..
		URIList = append(URIList, urilist{"/businessindustryandtrade", "home_page", "/", depth, 0})
		URIList = append(URIList, urilist{"/economy", "home_page", "/", depth, 1})
		URIList = append(URIList, urilist{"/employmentandlabourmarket", "home_page", "/", depth, 2})
		URIList = append(URIList, urilist{"/peoplepopulationandcommunity", "home_page", "/", depth, 3})

		if cfg.IncludeReleaseCalendar {
			URIList = append(URIList, urilist{"/releasecalendar", "home_page", "/", depth, 4})
		}

		// NOTE: comment out the above URIList append's and use the following line to quickly get
		// to a page that was causing code to fial as the following URI leads to a page that has an
		// escape sequence of '\f' that breaks golang's unmarshaling - that is worked around with
		// the function: removeNotAllowedEscapeCodes()
		// URIList = append(URIList, urilist{"/economy/inflationandpriceindices/articles/consumerpricesindexincludingowneroccupiershousingcostshistoricalseries/1988to2004", "home_page", "/", depth, 4})

	case taxonomyLandingPageCollectionName:
		var data DataResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 2)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 2, &payload, &fixedJSON, "DataResponse")

		saveContentPageToCollection(taxonomyLandingPageJsFile, &taxonomyLandingPageCount, taxonomyLandingPageCollectionName, bodyTextCopy, shortURI)
		URIList = getURIListFromTaxonomyLandingPage(fullURI, &data, shortURI, depth)

	case productPageCollectionName:
		var data DataResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 3)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 3, &payload, &fixedJSON, "DataResponse")

		saveContentPageToCollection(productPageJsFile, &productPageCount, productPageCollectionName, bodyTextCopy, shortURI)
		URIList = getURIListFromProductPage(fullURI, &data, shortURI, depth)

	case articleCollectionName:
		var data articleResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 4)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 4, &payload, &fixedJSON, "articleResponse")

		saveContentPageToCollection(articleJsFile, &articleCount, articleCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromArticle(fullURI, &data, shortURI, depth)
		}

	case articleDownloadCollectionName:
		var data articleDownloadResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 5)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 5, &payload, &fixedJSON, "articleDownloadResponse")

		saveContentPageToCollection(articleDownloadJsFile, &articleDownloadCount, articleDownloadCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromArticleDownload(fullURI, &data, shortURI, depth)
		}

	case bulletinnCollectionName:
		var data bulletinResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 6)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 6, &payload, &fixedJSON, "bulletinResponse")

		saveContentPageToCollection(bulletinJsFile, &bulletinCount, bulletinnCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromBulletin(fullURI, &data, shortURI, depth)
		}

	case compendiumDataCollectionName:
		var data compendiumDataResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 7)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 7, &payload, &fixedJSON, "compendiumDataResponse")

		saveContentPageToCollection(compendiumDataJsFile, &compendiumDataCount, compendiumDataCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromCompendiumData(fullURI, &data, shortURI, depth)
		}

	case compendiumLandingPageCollectionName:
		var data compendiumLandingPageResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 8)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 8, &payload, &fixedJSON, "compendiumLandingPageResponse")

		saveContentPageToCollection(compendiumLandingPageJsFile, &compendiumLandingPageCount, compendiumLandingPageCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromCompendiumLandingPage(fullURI, &data, shortURI, depth)
		}

	case datasetLandingPageCollectionName:
		var data datasetLandingPageResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 9)
		}

		payload, err := json.Marshal(data)
		checkMarshalingDeepEqual(fullURI, err, 9, &payload, &fixedJSON, "datasetLandingPageResponse")

		saveContentPageToCollection(datasetLandingPageJsFile, &datasetLandingPageCount, datasetLandingPageCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromDatasetLandingPage(fullURI, &data, shortURI, depth)
		}

	case staticMethodologyCollectionName:
		var data staticMethodologyResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 10)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 10, &payload, &fixedJSON, "staticMethodologyResponse")

		saveContentPageToCollection(staticMethodologyJsFile, &staticMethodologyCount, staticMethodologyCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticMethodology(fullURI, &data, shortURI, depth)
		}

	case staticMethodologyDownloadCollectionName:
		var data staticMethodologyDownloadResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 11)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 11, &payload, &fixedJSON, "staticMethodologyDownloadResponse")

		saveContentPageToCollection(staticMethodologyDownloadJsFile, &staticMethodologyDownloadCount, staticMethodologyDownloadCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticMethodologyDownload(fullURI, &data, shortURI, depth)
		}

	case staticQmiCollectionName:
		var data staticQmiResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 12)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 12, &payload, &fixedJSON, "staticQmiResponse")

		saveContentPageToCollection(staticQmiJsFile, &staticQmiCount, staticQmiCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticQmi(fullURI, &data, shortURI, depth)
		}

	case timeseriesCollectionName:
		var data timeseriesResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 13)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 13, &payload, &fixedJSON, "timeseriesResponse")

		saveContentPageToCollection(timeseriesJsFile, &timeseriesCount, timeseriesCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromTimeseries(fullURI, &data, shortURI, depth)
		}

	case chartCollectionName:
		var data chartResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 14)
		}

		payload, err := json.Marshal(data)
		checkMarshalingDeepEqual(fullURI, err, 14, &payload, &fixedJSON, "chartResponse")

		saveContentPageToCollection(chartJsFile, &chartCount, chartCollectionName, bodyTextCopy, shortURI)
		// NOTE: this page has no URI links to add to list

	case tableCollectionName:
		var data tableResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 15)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 15, &payload, &fixedJSON, "tableResponse")

		saveContentPageToCollection(tableJsFile, &tableCount, tableCollectionName, bodyTextCopy, shortURI)
		// NOTE: this page has no URI links to add to list

	case equationCollectionName:
		var data equationResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 16)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 16, &payload, &fixedJSON, "equationResponse")

		saveContentPageToCollection(equationJsFile, &equationCount, equationCollectionName, bodyTextCopy, shortURI)
		// NOTE: this page has no URI links to add to list

	case imageCollectionName:
		var data imageResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 17)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 17, &payload, &fixedJSON, "imageResponse")

		saveContentPageToCollection(imageJsFile, &imageCount, imageCollectionName, bodyTextCopy, shortURI)
		// NOTE: this page has no URI links to add to list

	case releaseCollectionName:
		var data releaseResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 18)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 18, &payload, &fixedJSON, "releaseResponse")

		if cfg.IncludeReleaseCalendar {
			saveReleaseInfoToCSV(shortURI, &data)
		}
		saveContentPageToCollection(releaseJsFile, &releaseCount, releaseCollectionName, bodyTextCopy, shortURI)

		if cfg.FullDepth {
			URIList = getURIListFromRelease(fullURI, &data, shortURI, depth)
		}

	case listCollectionName:
		var data listResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 19)
		}

		payload, err := json.Marshal(data)
		checkMarshalingDeepEqual(fullURI, err, 19, &payload, &fixedJSON, "listResponse")

		saveContentPageToCollection(listJsFile, &listCount, listCollectionName, bodyTextCopy, shortURI)
		// NOTE: this page has no 'DIRECT' URI links to add to list
		if cfg.FullDepth {
			// But, it may have links in any results array
			URIList = getURIListFromList(fullURI, &data, shortURI, depth)
		}
		if data.Result.Results != nil {
			var field *[]results = data.Result.Results

			fmt.Printf("field %v, nof results %v\n", len(*field), *data.Result.NumberOfResults)
			if len(*field) < *data.Result.NumberOfResults {
				fmt.Printf("\nnumberOfPages %d\n", *data.Result.NumberOfResults)

				redirectedURI := response.Request.URL.Path
				if strings.HasPrefix(redirectedURI, "//") {
					fmt.Printf("Double slash    : %s\n", redirectedURI)
					// Remove the first slash as it messes up the usage of the link
					// (the double slash signifies the link uses the same http or https protocol as whatever was being used)
					redirectedURI = redirectedURI[1:]
				}

				var fullURI = "https://www.production.onsdigital.co.uk" + shortURI + "/data"

				fmt.Printf("shortURI      URI: %v\n", shortURI)
				fmt.Printf("fullURI       URI: %v\n", fullURI)
				fmt.Printf("redirected    URI: %v\n", redirectedURI)

				// loop through getting other URI list pages, etc (and doing all necessary checks)
				for pageNumber := 2; pageNumber <= *data.Result.Paginator.NumberOfPages; pageNumber++ {

					// build up the required paginated 'list' page URI
					specificPage := fullURI + "?:uri="
					specificPage += shortURI[1:] // skip the leading 'forward slash'
					specificPage += "&page="
					specificPage += fmt.Sprintf("%d", pageNumber)
					fmt.Printf("specificPage  URI: %v\n", specificPage)
					good, pageURIList := getIndexedListPage(specificPage, shortURI, fieldName, parentURI, index, depth)
					if good == 200 {
						for _, eachURI := range pageURIList {
							URIList = append(URIList, eachURI)
						}
					}
				}
				/*
					Examples of what the URI's look like for list pages that have extra paginated pages
					the first page:

					https://www.ons.gov.uk/releasecalendar/data

					https://www.ons.gov.uk/economy/grossdomesticproductgdp/bulletins/gdpfirstquarterlyestimateuk/previousReleases/data

					where

					paginator
						numberOfPages 	412
						currentPage 	1

					the second page:

					https://www.ons.gov.uk/releasecalendar/data?:uri=releasecalendar&page=2

					https://www.ons.gov.uk/economy/grossdomesticproductgdp/bulletins/gdpfirstquarterlyestimateuk/previousReleases/data?:uri=economy/grossdomesticproductgdp/bulletins/gdpfirstquarterlyestimateuk/previousReleases&page=2

					where

					paginator
						numberOfPages 	412
						currentPage 	2

					last page:

					https://www.ons.gov.uk/releasecalendar/data?:uri=releasecalendar&page=412

					paginator
						numberOfPages 	412
						currentPage 	412

				*/
			}
		}

	case staticPageCollectionName:
		var data staticPageResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 20)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 20, &payload, &fixedJSON, "staticPageResponse")

		saveContentPageToCollection(staticPageJsFile, &staticPageCount, staticPageCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticPage(fullURI, &data, shortURI, depth)
		}

	case staticAdhocCollectionName:
		var data staticAdhocResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 21)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 21, &payload, &fixedJSON, "staticAdhocResponse")

		saveContentPageToCollection(staticAdhocJsFile, &staticAdhocCount, staticAdhocCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticAdhoc(fullURI, &data, shortURI, depth)
		}

	case referenceTablesCollectionName:
		var data referenceTablesResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 22)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 22, &payload, &fixedJSON, "referenceTablesResponse")

		saveContentPageToCollection(referenceTablesJsFile, &referenceTablesCount, referenceTablesCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromReferenceTables(fullURI, &data, shortURI, depth)
		}

	case compendiumChapterCollectionName:
		var data compendiumChapterResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 23)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 23, &payload, &fixedJSON, "compendiumChapterResponse")

		saveContentPageToCollection(compendiumChapterJsFile, &compendiumChapterCount, compendiumChapterCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromCompendiumChapter(fullURI, &data, shortURI, depth)
		}

	case staticLandingPageCollectionName:
		var data staticLandingPageResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 24)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 24, &payload, &fixedJSON, "staticLandingPageResponse")

		saveContentPageToCollection(staticLandingPageJsFile, &staticLandingPageCount, staticLandingPageCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticLandingPage(fullURI, &data, shortURI, depth)
		}

	case staticArticleCollectionName:
		var data staticArticleResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 25)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 25, &payload, &fixedJSON, "staticArticleResponse")

		saveContentPageToCollection(staticArticleJsFile, &staticArticleCount, staticArticleCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromStaticArticle(fullURI, &data, shortURI, depth)
		}

	case datasetCollectionName:
		var data datasetResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 26)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 26, &payload, &fixedJSON, "datasetResponse")

		saveContentPageToCollection(datasetJsFile, &datasetCount, datasetCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromDataset(fullURI, &data, shortURI, depth)
		}

	case timeseriesDatasetCollectionName:
		var data timeseriesDatasetResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 27)
		}

		payload, err := json.Marshal(data)
		checkMarshaling(fullURI, err, 27, &payload, &fixedJSON, "timeseriesDatasetResponse")

		saveContentPageToCollection(timeseriesDatasetJsFile, &timeseriesDatasetCount, timeseriesDatasetCollectionName, bodyTextCopy, shortURI)
		if cfg.FullDepth {
			URIList = getURIListFromTimeseriesDataset(fullURI, &data, shortURI, depth)
		}

	default:
		fmt.Printf("Unknown page Type ..\n")
		fmt.Printf("shape: %s\n", *shape.Type)
		fmt.Printf("URI: %s\n", fullURI)

		_, err = fmt.Fprint(bodyTextFile, "Unknown JSON body:\n")
		check(err)
		_, err = bodyTextFile.Write(bodyTextCopy)
		check(err)
		_, err = fmt.Fprint(bodyTextFile, "\n")
		check(err)

		os.Exit(82)

		// NOTE:
		//
		// home_page : whose uri is "/" .. this would need custom processing to explicitly add sub uri's
		//
		// taxonomy_landing_page : is the the first level down from 'home_page'
		//
		// product_page : is the level down from 'taxonomy_landing_page'
		//
	}

	// good 200 response, save page for later
	listMu.Lock()
	listOfPageData = append(listOfPageData, pageData{
		subSectionIndex: index,
		pageBroken:      false,
		shortURI:        shortURI,
		parentURI:       parentURI,
	})
	listMu.Unlock()

	currentTime := time.Now()
	fmt.Print(currentTime.Format("2006.01.02 15:04:05  "))

	fmt.Printf("Depth: %d : %v\n", depth, shortURI)

	if len(URIList) > 0 {
		var validURI []urilist

		for _, subURI := range URIList {
			// check thru sub URI's, if not already seen (or some other exclusion applies)

			if strings.Contains(subURI.uri, "http://www.ons.gov.uk") {
				fmt.Printf("WARNING: bad link to site using only HTTP and NOT HTTPS: %s\n", subURI.uri)
			}

			// some of the URI links have the 'ons' site in them which we don't want, so remove if present:
			subURI.uri = strings.ReplaceAll(subURI.uri, "https://www.ons.gov.uk", "")
			subURI.uri = strings.ReplaceAll(subURI.uri, "http://www.ons.gov.uk", "")

			if strings.Contains(subURI.uri, "https://") || strings.Contains(subURI.uri, "http://") {
				fmt.Printf("External site: %s\n", subURI.uri)
				continue
			}
			if subURI.uri[0] != '/' {
				fmt.Printf("Adding missing forward slash to: %s\n", subURI.uri)
				// In at least one place on ONS site a URI was missing a forward slash as the first character
				// and that breaks the attempt to open the URI in the code, so we add the missing '/'
				subURI.uri = "/" + subURI.uri
			}
			if contentDuplicateCheck.Value(subURI.uri) > 0 {
				contentDuplicateCheck.Inc(subURI.uri)
				//					fmt.Printf("Already processed: %s\n", subURI)
				continue
			}
			if strings.HasPrefix(subURI.uri, "/ons/external-links/") {
				fmt.Printf("A URI to external site: %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			if strings.HasPrefix(subURI.uri, "/ons/rel/") {
				fmt.Printf("A URI to external site: %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			if strings.HasSuffix(subURI.uri, ".doc") {
				fmt.Printf("A URI to .doc file: %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			if strings.HasSuffix(subURI.uri, "/index.html") {
				fmt.Printf("A URI to /index.html: %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			if strings.Contains(subURI.uri, "#") {
				hashURIMutex.Lock()
				hashURI[subURI.uri]++
				hashURIMutex.Unlock()
				fmt.Printf("A URI with a '#': %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			if strings.Contains(subURI.uri, "?") {
				questionURIMutex.Lock()
				questionURI[subURI.uri]++
				questionURIMutex.Unlock()
				fmt.Printf("A URI with a '?': %s\n%s\n", subURI.uri, fullURI)
				continue
			}
			parts := strings.Split(subURI.uri, "/")
			last := parts[len(parts)-1]
			var versionFound bool
			if cfg.SkipVersions {
				if len(last) > 1 {
					if last[0] == 'v' && (last[1] >= '0' && last[1] <= '9') {
						// we found what looks like a version number on the end of the URI path
						versionFound = true
					}
				}
			}
			if versionFound {
				skippedVersionURIMutex.Lock()
				skippedVersionURI[subURI.uri]++
				skippedVersionURIMutex.Unlock()
				fmt.Printf("Skipping URI with version on end: %s\n", subURI.uri)
				continue
			}

			// build up list to return ..
			validURI = append(validURI, subURI)
		}
		return 200, validURI
	}

	return response.StatusCode, URIList
}

func getIndexedListPage(specificPage string, shortURI string, fieldName string, parentURI string, index int, depth int) (int, []urilist) {
	// Create a list of URIs
	var URIList []urilist

	var fullURI = specificPage

	atomic.AddInt32(&attemptedGetCount, 1)
	if cfg.PlayNice {
		// a little delay to play nice with ONS site and 'hopefully' not have cloudflare 'reset' the connection
		time.Sleep(50 * time.Millisecond)
	}
	response, err := http.Get(fullURI)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("getIndexedListPage: http.Get(fullURI) failed\n")
		fmt.Printf("We now fabricate the response code to a 429 to instigate a retry after a delay 2\n")
		return 429, URIList
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		if response.StatusCode != 404 {
			if response.StatusCode != 429 {
				// a 503 is being seen at this point .. (it could be some other error, but whatever it is we do error action)
				fmt.Printf("\nERROR on ONS website /data field: %v\n\n", response.StatusCode)
				fmt.Printf("URI does not exist:  %v\n", fullURI)

				listMu.Lock()
				listOfPageData = append(listOfPageData, pageData{
					subSectionIndex: index,
					pageBroken:      true,
					depth:           depth,
					shortURI:        shortURI,
					parentURI:       parentURI,
					fieldName:       fieldName,
				})
				listMu.Unlock()
			} else {
				fmt.Printf("\nToo many requests\n")
				// caller will call this function again for a 429
			}
		}
		return response.StatusCode, URIList
	}
	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("getIndexedListPage: RealAll failed\n")
		os.Exit(10)
	}

	atomic.AddInt64(&dataRead, int64(len(bodyText)))

	// Take a copy into another block of memory before the call to replaceUnicodeWithASCII()
	// strips out the unicode characters.
	// .. thus retaining any unicode to write back out after checks made.
	var bodyTextCopy []byte = make([]byte, len(bodyText))
	copy(bodyTextCopy, bodyText)

	fixedJSON := replaceUnicodeWithASCII(bodyText)

	var shape pageShape
	// Unmarshal body bytes to model
	if err := json.Unmarshal(fixedJSON, &shape); err != nil {
		fmt.Println(err)
		fmt.Printf("getIndexedListPage: json.Unmarshal failed 1\n")
		// we can get here from:
		// /employmentandlabourmarket/peopleinwork/workplacepensions#publications
		// where the '#publications' jumps one some way into a page that has already been processed
		// (though #'s are now caught elsewhere)
		//
		// OR
		//    from:
		//    https://www.ons.gov.uk/search?q=interactivetool
		//    which is a valid page, but has no data structure to extract info from
		if shortURI == "/search?q=interactivetool" {
			// NOTE: the number of exception pages may need to grow
			// say the page is unavailable
			return 503, URIList
		}
		fmt.Printf("Unknown problem on page: %s\n", fullURI)
		fmt.Printf("shortURI: %s\n", shortURI)
		os.Exit(11)
	}

	// Decode each page into a specific structure according to the 'Type' of the page ..
	// NOTE: This is done to ensure that the structure definitions are fully defined to read ALL
	//       the info in the /data endpoint.
	switch *shape.Type {
	case listCollectionName:
		var data listResponse

		if err := json.Unmarshal(fixedJSON, &data); err != nil {
			unmarshalFail(fullURI, err, 19)
		}

		payload, err := json.Marshal(data)
		checkMarshalingDeepEqual(fullURI, err, 19, &payload, &fixedJSON, "listResponse")

		saveContentPageToCollection(listJsFile, &listCount, listCollectionName, bodyTextCopy, shortURI)
		URIList = getURIListFromList(fullURI, &data, shortURI, depth)

	default:
		fmt.Printf("Unknown page Type .. should only be seeing %s\n", listCollectionName)
		fmt.Printf("shape: %s\n", *shape.Type)
		fmt.Printf("URI: %s\n", fullURI)

		_, err = fmt.Fprint(bodyTextFile, "Unknown JSON body:\n")
		check(err)
		_, err = bodyTextFile.Write(bodyTextCopy)
		check(err)
		_, err = fmt.Fprint(bodyTextFile, "\n")
		check(err)

		os.Exit(83)
	}

	return 200, URIList
}

var skippedVersionURI = make(map[string]int) // key: shortURI, value: count unique URI's with version number that has been skipped
var skippedVersionURIMutex sync.Mutex

var hashURI = make(map[string]int) // key: shortURI, value: count unique URI's with HASH that has been skipped
var hashURIMutex sync.Mutex

var questionURI = make(map[string]int) // key: shortURI, value: count unique URI's with question mark that has been skipped
var questionURIMutex sync.Mutex

func getPageDataRetry(index int, shortURI string, fieldName string, parentFullURI string, depth int) (bool, []urilist) {
	var backOff int = 71
	var status int

	var validURI []urilist

	for {
		status, validURI = getPageData(shortURI, fieldName, parentFullURI, index, depth)
		if status == 200 {
			return true, validURI
		}
		if status == 404 || status == 503 {
			break
		}
		// got error 429 due to making too many requests in a short period of time
		fmt.Printf("backing Off for: %v\n", backOff)
		for delay := 0; delay < backOff; delay++ {
			time.Sleep(1 * time.Second)
			if delay%3 == 0 {
				fmt.Printf("Seconds remaining: %d\n", backOff-delay)
			}
		}
		backOff += 60
		if backOff > 200 {
			// probably a broken URIm but go try without /data on the end ..
			status = 404
			break
		}
	}
	if status == 404 {
		// try reading page without data on the end ..
		noDataURI := "https://www.ons.gov.uk" + shortURI
		// noDataURI := "https://www.production.onsdigital.co.uk" + shortURI

		var response *http.Response
		var err error
		var attempts int

		fmt.Printf("\nGetting /data failed, trying without /data to look for 'redirect'\n")
		for {
			atomic.AddInt32(&attemptedGetCount, 1)
			if cfg.PlayNice {
				// a little delay to play nice with ONS site and 'hopefully' not have cloudflare 'reset' the connection
				time.Sleep(50 * time.Millisecond)
			}
			response, err = http.Get(noDataURI)
			if err != nil {
				fmt.Println(err)
				fmt.Printf("getPageDataRetry: http.Get(noDataURI) failed\n")
				fmt.Printf("We now fabricate the response code to a 429 to instigate a retry after a delay 3\n")
				doAndShowDelay71()
			} else {
				if response.StatusCode == 429 {
					response.Body.Close()
					doAndShowDelay71()
				} else {
					// we got some response we can work with
					break
				}
			}
			attempts++
			if attempts >= 3 {
				// Possible problems are:
				// 1. URI on ONS is broke
				// 2. ONS site is down
				// 3. Network connection to ONS is down
				// SO, give up on this URI ..
				fmt.Printf("URI does not exist:  %v\n", shortURI)
				listMu.Lock()
				listOfPageData = append(listOfPageData, pageData{
					subSectionIndex: index,
					pageBroken:      true,
					depth:           depth,
					shortURI:        shortURI,
					parentURI:       parentFullURI,
					fieldName:       fieldName,
				})
				listMu.Unlock()

				return false, validURI
			}
		}

		defer response.Body.Close()

		// we read the Body as there is no there way to discover the length of the Body
		bodyText, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
			fmt.Printf("getPageData: RealAll failed\n")
			os.Exit(20)
		}

		atomic.AddInt64(&dataRead, int64(len(bodyText)))

		redirectedURI := response.Request.URL.Path
		if strings.HasPrefix(redirectedURI, "//") {
			fmt.Printf("Double slash    : %s\n", redirectedURI)
			// Remove the first slash as it messes up the usage of the link
			// (the double slash signifies the link uses the same http or https protocol as whatever was being used)
			redirectedURI = redirectedURI[1:]
		}
		fmt.Printf("failed to get URI: %v\n", shortURI)

		if shortURI != redirectedURI {
			fmt.Printf("redirected    URI: %v\n", redirectedURI)
			// we have a re-direction, so lets try that with /data
			backOff = 71
			for {
				status, validURI = getPageData(redirectedURI, fieldName, parentFullURI, index, depth)
				if status == 200 {
					// redirect worked, and page data was saved in the call to getPageData()
					fmt.Printf("redirect worked OK\n")
					return true, validURI
				}
				if status == 404 || status == 503 {
					break
				}
				// got error 429 due to making too many requests in a short period of time
				fmt.Printf("backing Off for: %v\n", backOff)
				for delay := 0; delay < backOff; delay++ {
					time.Sleep(1 * time.Second)
					if delay%3 == 0 {
						fmt.Printf("Seconds remaining: %d\n", backOff-delay)
					}
				}
				backOff += 60
				if backOff > 200 {
					// probably a broken URI,so give up
					status = 404
					break
				}
			}
		}
		// NOTE: a true 503 error will already have been recorded in getPageData()
		if status == 404 {
			// if we got a 404 then shortURI or redirectedURI is definitely broke

			fmt.Printf("\nERROR on ONS website /data field: %v\n\n", status)
			if shortURI != redirectedURI {
				fmt.Printf("redirected URI does not exist:  %v\n", redirectedURI)
			}
			fmt.Printf("URI does not exist:  %v\n", shortURI)
			listMu.Lock()
			listOfPageData = append(listOfPageData, pageData{
				subSectionIndex: index,
				pageBroken:      true,
				depth:           depth,
				shortURI:        shortURI,
				parentURI:       parentFullURI,
				fieldName:       fieldName,
			})
			listMu.Unlock()
		}
	}

	return false, validURI
}

func createBrokenLinkFile() {
	// no locks on listOfPageData needed in this function

	if listOfPageData == nil {
		return
	}
	fmt.Printf("\nNof listOfPageData: %v\n", len(listOfPageData))
	if len(listOfPageData) > 0 {
		brokenLinkTextFile, err := os.Create(observationsDir + "/broken_links.txt")
		check(err)
		defer brokenLinkTextFile.Close()

		brokenLinkWithoutVersionsTextFile, err := os.Create(observationsDir + "/broken_links_without_versions.txt")
		check(err)
		defer brokenLinkWithoutVersionsTextFile.Close()

		var errorCount int
		fmt.Printf("Showing: listOfPageData\n")
		for _, pagesData := range listOfPageData {
			if pagesData.pageBroken {
				errorCount++
				parentFullURI := pagesData.parentURI
				if parentFullURI[0] == '/' {
					parentFullURI = "https://www.ons.gov.uk" + parentFullURI
				}
				fmt.Printf("Error on page: %v\n    Broken link: ", parentFullURI)

				// save to file
				_, err = fmt.Fprintf(brokenLinkTextFile, "=================== Depth: %d\n", pagesData.depth)
				check(err)
				_, err = fmt.Fprintf(brokenLinkTextFile, "%v - Error on page: %v\n\n", errorCount, parentFullURI)
				check(err)
				_, err = fmt.Fprintf(brokenLinkTextFile, "%s:\n", pagesData.fieldName)
				check(err)
				_, err = fmt.Fprintf(brokenLinkTextFile, "  %v:\n", pagesData.subSectionIndex)
				check(err)
				_, err = fmt.Fprintf(brokenLinkTextFile, "    Broken link: uri: %v\n\n", pagesData.shortURI)
				check(err)
				_, err = fmt.Fprintf(brokenLinkTextFile, "    Broken link: %v\n\n", "https://www.ons.gov.uk"+pagesData.shortURI)
				check(err)

				// see if broken link is from a 'version' page
				parts := strings.Split(parentFullURI, "/")
				last := parts[len(parts)-1]
				var versionFound bool
				if len(last) > 1 {
					if last[0] == 'v' && (last[1] >= '0' && last[1] <= '9') {
						// we found what looks like a version number on the end of the URI path
						versionFound = true
					}
				}
				if !versionFound {
					// No version on end of path, save to 'without version' file
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "=================== Depth: %d\n", pagesData.depth)
					check(err)
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "%v - Error on page: %v\n\n", errorCount, parentFullURI)
					check(err)
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "%s:\n", pagesData.fieldName)
					check(err)
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "  %v:\n", pagesData.subSectionIndex)
					check(err)
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "    Broken link: uri: %v\n\n", pagesData.shortURI)
					check(err)
					_, err = fmt.Fprintf(brokenLinkWithoutVersionsTextFile, "    Broken link: %v\n\n", "https://www.ons.gov.uk"+pagesData.shortURI)
					check(err)
				}
			}
		}
	}
}

func uniqueString(stringSlice []string) []string {
	keys := make(map[string]bool)
	listString := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			listString = append(listString, entry)
		}
	}
	return listString
}

func createReleaseRelatedInfoFiles() {
	if releaseRelatedDatasets != nil {
		checkFile, err := os.Create("mongo-init-scripts/release-related-datasets.csv")
		check(err)
		defer checkFile.Close()

		sort.Strings(releaseRelatedDatasets)

		uniqueStrings := uniqueString(releaseRelatedDatasets)

		for _, releaseLine := range uniqueStrings {
			_, err = fmt.Fprintf(checkFile, "%s\n", releaseLine)
			check(err)
		}
	}

	if releaseRelatedDocuments != nil {
		checkFile, err := os.Create("mongo-init-scripts/release-related-documents.csv")
		check(err)
		defer checkFile.Close()

		sort.Strings(releaseRelatedDocuments)

		uniqueStrings := uniqueString(releaseRelatedDocuments)

		for _, releaseLine := range uniqueStrings {
			_, err = fmt.Fprintf(checkFile, "%s\n", releaseLine)
			check(err)
		}
	}
}

// create file that contains list of URI's saved when doing deeper scan together with
// the name of the collection that the URI info is stored in - that is the 'type'
// of the page and thus one knows the struct to use to read the URI
func createURICollectionNamesFile() {
	var nofURIs int

	defer func() {
		fmt.Printf("\nNof URI's / keys stored: %v\n", nofURIs)
	}()

	//	if uriCollectionName == nil {
	//		return
	//	}

	nofURIs = uriCollectionName.Length()

	if nofURIs == 0 {
		return
	}

	namesTextFile, err := os.Create("mongo-init-scripts/uri_collection_names.txt")
	check(err)
	defer namesTextFile.Close()

	// We are running single thread, so no need to protect access to uriCollectionsName with Mutex's

	// Create a list of sorted URIs
	URIs := make([]string, 0, nofURIs)
	for k := range uriCollectionName.v {
		URIs = append(URIs, k)
	}
	sort.Strings(URIs)

	// Use sorted list of URIs to iterate through 'uriCollectionName' in order to
	// save the URI's and their collection name in order of URI's
	for _, shortURI := range URIs {
		_, err = fmt.Fprintf(namesTextFile, "%s,%s\n", shortURI, uriCollectionName.v[shortURI])
		check(err)
	}
}

func createContentCountsFile() {
	countsTextFile, err := os.Create("mongo-init-scripts/collection_lengths.txt")
	check(err)
	defer countsTextFile.Close()

	_, err = fmt.Fprintf(countsTextFile, "article_download collection quantity: %d\n", articleDownloadCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "article collection quantity: %d\n", articleCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "bulletin collection quantity: %d\n", bulletinCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "chart collection quantity: %d\n", chartCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "compendium_chapter collection quantity: %d\n", compendiumChapterCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "compendium_data collection quantity: %d\n", compendiumDataCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "compendium_landing_page collection quantity: %d\n", compendiumLandingPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "dataset_landing_page collection quantity: %d\n", datasetLandingPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "dataset collection quantity: %d\n", datasetCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "equation collection quantity: %d\n", equationCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "image collection quantity: %d\n", imageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "list collection quantity: %d\n", listCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "product_page collection quantity: %d\n", productPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "reference_tables collection quantity: %d\n", referenceTablesCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "release collection quantity: %d\n", releaseCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_adhoc collection quantity: %d\n", staticAdhocCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_article collection quantity: %d\n", staticArticleCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_landing_page collection quantity: %d\n", staticLandingPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_methodology_download collection quantity: %d\n", staticMethodologyDownloadCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_methodology collection quantity: %d\n", staticMethodologyCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_page collection quantity: %d\n", staticPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "static_qmi collection quantity: %d\n", staticQmiCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "table collection quantity: %d\n", tableCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "taxonomy_landing_page collection quantity: %d\n", taxonomyLandingPageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "home_page collection quantity: %d\n", homePageCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "timeseries_dataset collection quantity: %d\n", timeseriesDatasetCount)
	check(err)
	_, err = fmt.Fprintf(countsTextFile, "timeseries collection quantity: %d\n", timeseriesCount)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var articleJsFile *os.File

const articleCollectionName = "article"

var articleDownloadJsFile *os.File

const articleDownloadCollectionName = "article_download"

var bulletinJsFile *os.File

const bulletinnCollectionName = "bulletin"

var compendiumDataJsFile *os.File

const compendiumDataCollectionName = "compendium_data"

var compendiumLandingPageJsFile *os.File

const compendiumLandingPageCollectionName = "compendium_landing_page"

var datasetLandingPageJsFile *os.File

const datasetLandingPageCollectionName = "dataset_landing_page"

var staticMethodologyJsFile *os.File

const staticMethodologyCollectionName = "static_methodology"

var staticMethodologyDownloadJsFile *os.File

const staticMethodologyDownloadCollectionName = "static_methodology_download"

var staticQmiJsFile *os.File

const staticQmiCollectionName = "static_qmi"

var timeseriesJsFile *os.File

const timeseriesCollectionName = "timeseries"

var chartJsFile *os.File

const chartCollectionName = "chart"

var productPageJsFile *os.File

const productPageCollectionName = "product_page"

var tableJsFile *os.File

const tableCollectionName = "table"

var equationJsFile *os.File

const equationCollectionName = "equation"

var imageJsFile *os.File

const imageCollectionName = "image"

var releaseJsFile *os.File

const releaseCollectionName = "release"

var listJsFile *os.File

const listCollectionName = "list"

var staticPageJsFile *os.File

const staticPageCollectionName = "static_page"

var staticAdhocJsFile *os.File

const staticAdhocCollectionName = "static_adhoc"

var referenceTablesJsFile *os.File

const referenceTablesCollectionName = "reference_tables"

var compendiumChapterJsFile *os.File

const compendiumChapterCollectionName = "compendium_chapter"

var staticLandingPageJsFile *os.File

const staticLandingPageCollectionName = "static_landing_page"

var staticArticleJsFile *os.File

const staticArticleCollectionName = "static_article"

var datasetJsFile *os.File

const datasetCollectionName = "dataset"

var timeseriesDatasetJsFile *os.File

const timeseriesDatasetCollectionName = "timeseries_dataset"

var taxonomyLandingPageJsFile *os.File

const taxonomyLandingPageCollectionName = "taxonomy_landing_page"

var homePageJsFile *os.File

const homePageCollectionName = "home_page"

var bodyTextFile *os.File
var checkFile *os.File

func initialiseCollectionDatabase(collectionName string, collectionFile *os.File) {
	line1 := "db = db.getSiblingDB('" + topicsDbName + "')\n"
	line2 := "db." + collectionName + ".remove({})\n"

	_, err := fmt.Fprint(collectionFile, line1)
	check(err)
	_, err = fmt.Fprint(collectionFile, line2)
	check(err)
}

func finaliseCollectionDatabase(collectionName string, collectionFile *os.File) {
	// Add code to read back each document written (for visual inspection)
	// NOTE: these lines in script are commented out to speed the process up for long scripts
	//       they are placed in init script should they need to be uncomented ..
	_, err := fmt.Fprint(collectionFile, "//db."+collectionName+".find().forEach(function(doc) {\n")
	check(err)
	_, err = fmt.Fprint(collectionFile, "//    printjson(doc);\n")
	check(err)
	_, err = fmt.Fprint(collectionFile, "//})\n")
	check(err)
}

const (
	topicsDbName    = "topics"
	initDir         = "mongo-init-scripts"
	tempDir         = "temp"
	observationsDir = "observations"
)

func ensureDirectoryExists(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		check(os.Mkdir(dirName, 0700))
	}
}

func main() {
	// Read config
	err := InitConfig()
	if err != nil {
		fmt.Printf("error initialising config\n")
		return
	}

	ensureDirectoryExists(initDir)

	if cfg.SaveSite {
		// Create files 'article' content creation file
		articleJsFile, err = os.Create(initDir + "/" + articleCollectionName + "-init.js")
		check(err)
		defer articleJsFile.Close()
		initialiseCollectionDatabase(articleCollectionName, articleJsFile)

		// Create files 'article_download' content creation file
		articleDownloadJsFile, err = os.Create(initDir + "/" + articleDownloadCollectionName + "-init.js")
		check(err)
		defer articleDownloadJsFile.Close()
		initialiseCollectionDatabase(articleDownloadCollectionName, articleDownloadJsFile)

		// Create files 'bulletin' content creation file
		bulletinJsFile, err = os.Create(initDir + "/" + bulletinnCollectionName + "-init.js")
		check(err)
		defer bulletinJsFile.Close()
		initialiseCollectionDatabase(bulletinnCollectionName, bulletinJsFile)

		// Create files 'compendium_data' content creation file
		compendiumDataJsFile, err = os.Create(initDir + "/" + compendiumDataCollectionName + "-init.js")
		check(err)
		defer compendiumDataJsFile.Close()
		initialiseCollectionDatabase(compendiumDataCollectionName, compendiumDataJsFile)

		// Create files 'compendium_landing_page' content creation file
		compendiumLandingPageJsFile, err = os.Create(initDir + "/" + compendiumLandingPageCollectionName + "-init.js")
		check(err)
		defer compendiumLandingPageJsFile.Close()
		initialiseCollectionDatabase(compendiumLandingPageCollectionName, compendiumLandingPageJsFile)

		// Create files 'dataset_landing_page' content creation file
		datasetLandingPageJsFile, err = os.Create(initDir + "/" + datasetLandingPageCollectionName + "-init.js")
		check(err)
		defer datasetLandingPageJsFile.Close()
		initialiseCollectionDatabase(datasetLandingPageCollectionName, datasetLandingPageJsFile)

		// Create files 'static_methodology' content creation file
		staticMethodologyJsFile, err = os.Create(initDir + "/" + staticMethodologyCollectionName + "-init.js")
		check(err)
		defer staticMethodologyJsFile.Close()
		initialiseCollectionDatabase(staticMethodologyCollectionName, staticMethodologyJsFile)

		// Create files 'static_methodology_download' content creation file
		staticMethodologyDownloadJsFile, err = os.Create(initDir + "/" + staticMethodologyDownloadCollectionName + "-init.js")
		check(err)
		defer staticMethodologyDownloadJsFile.Close()
		initialiseCollectionDatabase(staticMethodologyDownloadCollectionName, staticMethodologyDownloadJsFile)

		// Create files 'static_qmi' content creation file
		staticQmiJsFile, err = os.Create(initDir + "/" + staticQmiCollectionName + "-init.js")
		check(err)
		defer staticQmiJsFile.Close()
		initialiseCollectionDatabase(staticQmiCollectionName, staticQmiJsFile)

		// Create files 'timeseries' content creation file
		timeseriesJsFile, err = os.Create(initDir + "/" + timeseriesCollectionName + "-init.js")
		check(err)
		defer timeseriesJsFile.Close()
		initialiseCollectionDatabase(timeseriesCollectionName, timeseriesJsFile)

		// Create files 'chart' content creation file
		chartJsFile, err = os.Create(initDir + "/" + chartCollectionName + "-init.js")
		check(err)
		defer chartJsFile.Close()
		initialiseCollectionDatabase(chartCollectionName, chartJsFile)

		// Create files 'product_page' content creation file
		productPageJsFile, err = os.Create(initDir + "/" + productPageCollectionName + "-init.js")
		check(err)
		defer productPageJsFile.Close()
		initialiseCollectionDatabase(productPageCollectionName, productPageJsFile)

		// Create files 'table' content creation file
		tableJsFile, err = os.Create(initDir + "/" + tableCollectionName + "-init.js")
		check(err)
		defer tableJsFile.Close()
		initialiseCollectionDatabase(tableCollectionName, tableJsFile)

		// Create files 'equation' content creation file
		equationJsFile, err = os.Create(initDir + "/" + equationCollectionName + "-init.js")
		check(err)
		defer equationJsFile.Close()
		initialiseCollectionDatabase(equationCollectionName, equationJsFile)

		// Create files 'image' content creation file
		imageJsFile, err = os.Create(initDir + "/" + imageCollectionName + "-init.js")
		check(err)
		defer imageJsFile.Close()
		initialiseCollectionDatabase(imageCollectionName, imageJsFile)

		// Create files 'release' content creation file
		releaseJsFile, err = os.Create(initDir + "/" + releaseCollectionName + "-init.js")
		check(err)
		defer releaseJsFile.Close()
		initialiseCollectionDatabase(releaseCollectionName, releaseJsFile)

		// Create files 'list' content creation file
		listJsFile, err = os.Create(initDir + "/" + listCollectionName + "-init.js")
		check(err)
		defer listJsFile.Close()
		initialiseCollectionDatabase(listCollectionName, listJsFile)

		// Create files 'static_page' content creation file
		staticPageJsFile, err = os.Create(initDir + "/" + staticPageCollectionName + "-init.js")
		check(err)
		defer staticPageJsFile.Close()
		initialiseCollectionDatabase(staticPageCollectionName, staticPageJsFile)

		// Create files 'static_adhoc' content creation file
		staticAdhocJsFile, err = os.Create(initDir + "/" + staticAdhocCollectionName + "-init.js")
		check(err)
		defer staticAdhocJsFile.Close()
		initialiseCollectionDatabase(staticAdhocCollectionName, staticAdhocJsFile)

		// Create files 'reference_tables' content creation file
		referenceTablesJsFile, err = os.Create(initDir + "/" + referenceTablesCollectionName + "-init.js")
		check(err)
		defer referenceTablesJsFile.Close()
		initialiseCollectionDatabase(referenceTablesCollectionName, referenceTablesJsFile)

		// Create files 'compendium_chapter' content creation file
		compendiumChapterJsFile, err = os.Create(initDir + "/" + compendiumChapterCollectionName + "-init.js")
		check(err)
		defer compendiumChapterJsFile.Close()
		initialiseCollectionDatabase(compendiumChapterCollectionName, compendiumChapterJsFile)

		// Create files 'static_landing_page' content creation file
		staticLandingPageJsFile, err = os.Create(initDir + "/" + staticLandingPageCollectionName + "-init.js")
		check(err)
		defer staticLandingPageJsFile.Close()
		initialiseCollectionDatabase(staticLandingPageCollectionName, staticLandingPageJsFile)

		// Create files 'static_article' content creation file
		staticArticleJsFile, err = os.Create(initDir + "/" + staticArticleCollectionName + "-init.js")
		check(err)
		defer staticArticleJsFile.Close()
		initialiseCollectionDatabase(staticArticleCollectionName, staticArticleJsFile)

		// Create files 'dataset' content creation file
		datasetJsFile, err = os.Create(initDir + "/" + datasetCollectionName + "-init.js")
		check(err)
		defer datasetJsFile.Close()
		initialiseCollectionDatabase(datasetCollectionName, datasetJsFile)

		// Create files 'timeseries_dataset' content creation file
		timeseriesDatasetJsFile, err = os.Create(initDir + "/" + timeseriesDatasetCollectionName + "-init.js")
		check(err)
		defer timeseriesDatasetJsFile.Close()
		initialiseCollectionDatabase(timeseriesDatasetCollectionName, timeseriesDatasetJsFile)

		// Create files 'taxonomy_landing_page' content creation file
		taxonomyLandingPageJsFile, err = os.Create(initDir + "/" + taxonomyLandingPageCollectionName + "-init.js")
		check(err)
		defer taxonomyLandingPageJsFile.Close()
		initialiseCollectionDatabase(taxonomyLandingPageCollectionName, taxonomyLandingPageJsFile)

		// Create files 'home_page' content creation file
		homePageJsFile, err = os.Create(initDir + "/" + homePageCollectionName + "-init.js")
		check(err)
		defer homePageJsFile.Close()
		initialiseCollectionDatabase(homePageCollectionName, homePageJsFile)
	}

	ensureDirectoryExists(tempDir)

	// These files are saved for visual comparison when a structure decode and encode differ.
	// Open both files in vscode, right click in them and select 'Format Document' to expand the json,
	// save each expanded .json file and then do a visual diff between them with meld.
	// It is recommended that you use meld because some timeseries can be over 12,000 lines long.
	bodyTextFile, err = os.Create(tempDir + "/bodyText_all.json")
	check(err)
	defer bodyTextFile.Close()

	checkFile, err = os.Create(tempDir + "/bodyText_all_processed.json")
	check(err)
	defer checkFile.Close()

	ensureDirectoryExists(observationsDir)

	var validURI []urilist
	var toProcessURI []urilist
	var success bool

	// Get initial URI's to search
	success, validURI = getPageDataRetry(0, "", "root", "", depth)
	if !success {
		fmt.Printf("No pages found - Game Over\n")
		return
	}

	var concurrent int = 1
	if cfg.UseThreads {
		concurrent = 10 // Don't make this bigger that TEN, as we want to PLAY NICE !! (and not overload the site with requests)
	}

	var semaphoreChan = make(chan struct{}, concurrent)

	oneSecTick := time.NewTicker(time.Millisecond * 250)

	go func() {
		for {
			select {
			case <-oneSecTick.C:
				dRead := atomic.LoadInt64(&dataRead)
				if dRead > 1*512*1024 {
					// allow more reads every 250ms
					atomic.StoreInt64(&dataRead, dRead-1*512*1024)
				} else {
					// we have not read the limit for this time slot, so zero the count
					atomic.StoreInt64(&dataRead, 0)
				}
			}
		}
	}()

	// process complete list of URI's one depth at a time
	for len(validURI) > 0 {

		fmt.Printf("Number to check at depth: %d  is: %d\n", depth, len(validURI))

		depth++

		toProcessURI = toProcessURI[:0]
		// copy the 'next' list for processing
		toProcessURI = append(toProcessURI, validURI...)

		validURI = validURI[:0]

		var validMutex sync.Mutex

		var wg sync.WaitGroup // number of working goroutines

		numberToProcess := len(toProcessURI)
		for index, item := range toProcessURI {
			fmt.Printf("%d of %d : ", index, numberToProcess)

			semaphoreChan <- struct{}{} // block while full

			wg.Add(1)
			// Worker
			go func(item urilist) {
				defer func() {
					<-semaphoreChan // read to release a slot
				}()

				defer wg.Done()

				if cfg.LimitReads {
					for {
						dRead := atomic.LoadInt64(&dataRead)
						if dRead > 1*512*1024 {
							// we've read enough data this time slot, so pause before doing any more
							// (this seems to do a reasonable job of limiting spikes and limiting the amount of data read per second)
							time.Sleep(1 * time.Millisecond)
						} else {
							break
						}
					}
				}

				// Get a list of URI's from page
				_, nextURIList := getPageDataRetry(item.index, item.uri, item.field, item.parentURI, depth)

				if len(nextURIList) > 0 {
					// accumulate the next list of URI's to search
					validMutex.Lock()
					validURI = append(validURI, nextURIList...)
					validMutex.Unlock()
				}
			}(item)
		}

		wg.Wait()
	}

	if cfg.SaveSite {
		// close the content database creation files ..

		finaliseCollectionDatabase(articleCollectionName, articleJsFile)
		finaliseCollectionDatabase(articleDownloadCollectionName, articleDownloadJsFile)
		finaliseCollectionDatabase(bulletinnCollectionName, bulletinJsFile)
		finaliseCollectionDatabase(compendiumDataCollectionName, compendiumDataJsFile)
		finaliseCollectionDatabase(compendiumLandingPageCollectionName, compendiumLandingPageJsFile)
		finaliseCollectionDatabase(datasetLandingPageCollectionName, datasetLandingPageJsFile)
		finaliseCollectionDatabase(staticMethodologyCollectionName, staticMethodologyJsFile)
		finaliseCollectionDatabase(staticMethodologyDownloadCollectionName, staticMethodologyDownloadJsFile)
		finaliseCollectionDatabase(staticQmiCollectionName, staticQmiJsFile)
		finaliseCollectionDatabase(timeseriesCollectionName, timeseriesJsFile)
		finaliseCollectionDatabase(chartCollectionName, chartJsFile)
		finaliseCollectionDatabase(productPageCollectionName, productPageJsFile)
		finaliseCollectionDatabase(tableCollectionName, tableJsFile)
		finaliseCollectionDatabase(equationCollectionName, equationJsFile)
		finaliseCollectionDatabase(imageCollectionName, imageJsFile)
		finaliseCollectionDatabase(releaseCollectionName, releaseJsFile)
		finaliseCollectionDatabase(listCollectionName, listJsFile)
		finaliseCollectionDatabase(staticPageCollectionName, staticPageJsFile)
		finaliseCollectionDatabase(staticAdhocCollectionName, staticAdhocJsFile)
		finaliseCollectionDatabase(referenceTablesCollectionName, referenceTablesJsFile)
		finaliseCollectionDatabase(compendiumChapterCollectionName, compendiumChapterJsFile)
		finaliseCollectionDatabase(staticLandingPageCollectionName, staticLandingPageJsFile)
		finaliseCollectionDatabase(staticArticleCollectionName, staticArticleJsFile)
		finaliseCollectionDatabase(datasetCollectionName, datasetJsFile)
		finaliseCollectionDatabase(timeseriesDatasetCollectionName, timeseriesDatasetJsFile)
		finaliseCollectionDatabase(taxonomyLandingPageCollectionName, taxonomyLandingPageJsFile)
		finaliseCollectionDatabase(homePageCollectionName, homePageJsFile)

		createReleaseRelatedInfoFiles()
	}
	createContentCountsFile()

	// ===

	createBrokenLinkFile()

	createURICollectionNamesFile()

	fmt.Printf("\nmaxDepth: %d\n", depth)

	fmt.Printf("\nattemptedGetCount is: %v\n", attemptedGetCount)

	fmt.Printf("\nLength of contentDuplicateCheck (URI's saved) is: %d\n", contentDuplicateCheck.Length())

	fmt.Printf("\nNumber of URI's not saved with Version Number on end: %d\n", len(skippedVersionURI))

	fmt.Printf("\nNumber of URI's not saved with # (hash) in them: %d\n", len(hashURI))

	fmt.Printf("\nNumber of URI's not saved with ? (question mark) in them: %d\n", len(questionURI))

	fmt.Printf("\nAll Done.\n")
}
