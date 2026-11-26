# Repository analysis

Reports repositories in ONSdigital whose CODEOWNERS file matches a selected team.
Writes visibility, code owners, languages and a heuristic
repository type to a CSV report, plus a Markdown chart summary. Classification
evidence can optionally be included in the CSV.

## Running

Set `GITHUB_PERSONAL_ACCESS_TOKEN` to a token with access to the repositories, then
run from this directory:

```sh
make repo-analysis
# Or directly:
go run .
```

To include classification evidence:

```sh
make repo-analysis-with-evidence
# Or directly:
go run . -with-evidence
```

Choose a numbered team or enter `@ONSdigital/team-slug`.

Enter `q` at the team prompt to quit without searching repositories or writing reports.

## Languages

GitHub's Languages API reports source-byte counts calculated by Linguist. Only
languages strictly above **10%** of the complete total are shown, with percentages.
Exactly 10% is excluded.

## Repository-type heuristics

GitHub has no authoritative application/library field. The tool uses these initial
signals, reporting the relevant paths as evidence:

| Type           | Supported evidence                                                                                                                                                                                           |
|----------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| application    | In a Go module with `go.mod`, `func main()` in module-root Go files or nested `main.go`; npm `start` script or executable `bin`; Python `__main__.py`                                                        |
| documentation  | Documentation-only files and supporting assets/metadata, without substantive code or project manifests                                                                                                       |
| infrastructure | Helm-prefixed repository names; Helm `Chart.yaml` projects; Terraform `.tf` configuration without unresolved source/project evidence in that project; literal `-infra-` name fallback for remaining unknowns |
| lambda         | A hyphen-delimited `lambda` token in the repository name, overriding other types unless a PoC name token also matches                                                                                        |
| library        | For otherwise-unknown repositories, regular files under root `ci/`, or `.github/workflows/ci.yml` or `.github/workflows/ci.yaml`, provide a library fallback                                                 |
| poc            | A hyphen-delimited `poc` token in the repository name, or supported proof-of-concept self-description                                                                                                        |
| testing        | Explicit test-content/data or fixture self-description, or a `-tests` name suffix, only for otherwise unknown repositories                                                                                   |
| unknown        | Insufficient, unsupported or differing independent project evidence not resolved by hints; otherwise-unclassified empty repositories                                                                         |

### Description and README evidence

When structural analysis is unknown, the repository description and GitHub-selected
primary README can establish application/library intent. Examples include
`A Go library for ...`, `This repository is an application ...`, or
`<repository-name> is a library ...`.

A standalone, case-insensitive `library for` phrase anywhere in the GitHub
description establishes library.

An explicit self-description as a `wrapper for` or `wrapper around` another tool
also establishes library intent, in the GitHub description or supported introductory
README sections. For example, `dis-vulncheck is a wrapper around govulncheck ...`
is library evidence.

Wrapper self-descriptions take precedence over application evidence and ordinary
application claims, retaining earlier evidence and recording the winning rule.

Library-priority rules do not override PoC, Lambda, infrastructure, testing or
documentation, or resolve conflicting testing claims to library.

For repositories still unknown without an explicit type conflict,
the following fallback rules run in order:

1. An explicit test-data/fixture self-description establishes `testing`.
1. With no reported languages, documentation plus recognised supporting
   configuration can establish `documentation`.
1. A standalone `API` or `platform` word in the GitHub description, or an exact
   `Dockerfile.concourse` filename at root or in a meaningful nested directory,
   establishes application.
1. A case-insensitive literal `-library-` within the repository name
   establishes library.
1. Regular files under root `ci/` (such as `ci/audit.yml` or
   `ci/build/task.yml`) or exact `.github/workflows/ci.yml`/`ci.yaml` paths establish
   library.
1. A case-insensitive literal `-infra-` within the repository name
   establishes infrastructure.
1. A case-insensitive `-tests` suffix establishes testing.
