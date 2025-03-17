# Publishing timings tool

## Overview

This tool traverses the publish-log directory in the zebedee content workspace and extracts timings to provide an
illustration of the length of time scheduled collections and the releases they are part of take to pre-publish and
publish. It does this by identifying the relevant collections, getting the earliest and latest timestamps in the file
related to [pre-]publishing activities. It then augments this with file counts and

## How to use

To run locally you can simply run the main file using `go run main.go` although you would be advised to at least supply
a corrected path to your own zebedee content via the `-dir /path/to/dir`.

To build the binary and deploy it to an environment run `make deploy` for sandbox or `ENV=staging make deploy` etc. for
other environments. You can then ssh onto the `publishing_mount` box and run the `./publish-timings` tool from the home
directory. The tool will print a lot of logging output (more available with `-debug) and then output two csv files
containing the results.

The following command line arguments are available to modify the default settings.

| Flag     | Description                                          | Default                             | 
|----------|------------------------------------------------------|-------------------------------------|
| `-dir`   | path to publish-log directory                        | `/var/florence/zebedee/publish-log` |
| `-from`  | date from (eg. `2024-03-17`)                         | 365 days ago                        |
| `-to`    | date to   (eg. `2025-03-17`)                         | today                               |
| `-times` | publishing times (comma seperated eg, `07:00,09:30`) | `07:00`                             |
| `-extra` | extra minutes to add when matching timestamps        | `2`                                 |
| `-cols`  | filename of cols csv to output                       | `collections.csv`                   |
| `-rels`  | filename of releases csv to output                   | `releases.csv`                      |
| `-debug` | debug mode                                           |                                     |

Note: If a collection takes more than a minute to publish, its timestamp will be later than the publish time. Therefore
we also search for collections with a timestamp some extra minutes after the desired timestamp. I.e. `-extra 2` means
that as well as `07-00`, files with time stamps of `07-01`/`07-02` will also be checked to see if they are part of the
desired release. Also, the times themselves are always local time so `07:00` as an input will yield results for `07:00`
during GMT but `06:00` during BST as expected. Times in the output will always be given in UTC.

## Assumptions and limitations.

- Error handling is simplistic. The script will panic on most errors so don't be surprised by that if something goes
  wrong. You may have to fix issues as they arise.
- For publish timings, each collection logs the start and end times in the json explicitly (in the `publishStartDate`
  and `publishEndDate` fields). However, this is not done for pre-publish timings. We assume the start and end times by
  taking the earliest and latest train transaction activity from the `publishResults` object in the collection json
  which will give a pretty accurate estimate albeit not an absolutely accurate figure.
- The counting of files and sizes does not take into account the `timeseries-to-publish.zip` files. This content is
  therefore double counted in both compressed and uncompressed states. It could be possible to exclude the
  non-compressed timeseries content, however this would massively reduce the file-count which would be misleading as it
  wouldn't show the number of files copied over on the web/train side at publishing after it has been decompressed. 
