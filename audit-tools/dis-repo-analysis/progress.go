package main

import (
	"errors"
	"fmt"
	"log"
)

// runRepositoryStage executes a given stage of repository processing, providing progress feedback via the logger.
func runRepositoryStage(logger *log.Logger, stage string, names []string, process func(int, string) error) error {
	if _, err := fmt.Fprintf(logger.Writer(), "Getting %s", stage); err != nil {
		return fmt.Errorf("write progress message: %w", err)
	}

	finishLine := func(err error) error {
		if _, writeErr := fmt.Fprintln(logger.Writer()); writeErr != nil {
			return errors.Join(err, fmt.Errorf("write progress newline: %w", writeErr))
		}
		return err
	}
	for i, name := range names {
		if err := process(i, name); err != nil {
			return finishLine(err)
		}
		if _, err := fmt.Fprint(logger.Writer(), "."); err != nil {
			return finishLine(fmt.Errorf("write progress dot: %w", err))
		}
	}
	if err := finishLine(nil); err != nil {
		return err
	}
	logger.Printf("Completed getting %s", stage)
	return nil
}
