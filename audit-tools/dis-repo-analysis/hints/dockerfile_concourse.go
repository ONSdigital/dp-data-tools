package hints

import (
	"path"

	"github.com/google/go-github/v74/github"
)

// dockerfileConcourseEvidence returns evidence for Dockerfile.concourse files in the repository.
func dockerfileConcourseEvidence(entries []*github.TreeEntry) []string {
	var evidence []string
	for _, entry := range entries {
		name := entry.GetPath()
		// Git represents symlinks as blobs; exclude them as classification evidence.
		if entry.GetType() == "blob" && entry.GetMode() != "120000" &&
			path.Base(name) == "Dockerfile.concourse" && !IgnoredPath(name) && !SupportingTypePath(path.Dir(name)) {
			evidence = append(evidence, name+": application build configuration")
		}
	}
	return evidence
}
