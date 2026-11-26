package output

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// writeReport writes the report to a temporary file and then moves it to the destination.
func writeReport(destination, format string, write func(io.Writer) error) (err error) {
	file, err := os.CreateTemp(filepath.Dir(destination), ".repo-analysis-*."+strings.ToLower(format))
	if err != nil {
		return fmt.Errorf("create temporary %s for %s: %w", format, destination, err)
	}

	closed, renamed := false, false
	defer func() {
		if !closed {
			if closeErr := file.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close temporary %s: %w", format, closeErr))
			}
		}
		if !renamed {
			if removeErr := os.Remove(file.Name()); removeErr != nil {
				err = errors.Join(err, fmt.Errorf("remove temporary %s: %w", format, removeErr))
			}
		}
	}()

	if err := write(file); err != nil {
		return fmt.Errorf("write %s for %s: %w", format, destination, err)
	}

	closeErr := file.Close()
	closed = true
	if closeErr != nil {
		return fmt.Errorf("close %s for %s: %w", format, destination, closeErr)
	}

	if err := os.Rename(file.Name(), destination); err != nil {
		return fmt.Errorf("replace %s %s: %w", format, destination, err)
	}
	renamed = true
	return nil
}
