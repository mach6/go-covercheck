package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func handleHistoryOperations(cmd *cobra.Command, results compute.Results, cfg *config.Config) error {
	historyLimit, _ := cmd.Flags().GetInt(HistoryLimitFlag)

	// compare results against history, when requested
	compareRef, _ := cmd.Flags().GetString(CompareHistoryFlag)
	if compareRef != "" {
		if err := compareHistory(cmd, compareRef, results); err != nil {
			return err
		}
	}

	// save results to history, when requested.
	bSaveHistory, _ := cmd.Flags().GetBool(SaveHistoryFlag)
	if bSaveHistory {
		if err := saveHistory(cmd, results, historyLimit, cfg); err != nil {
			return err
		}
	}

	return nil
}

func getHistory(cmd *cobra.Command) (*history.History, error) {
	historyFile, err := getHistoryPath(cmd)
	if err != nil {
		return nil, err
	}
	// loads previous history if it exists
	return history.Load(historyFile)
}

func getHistoryPath(cmd *cobra.Command) (string, error) {
	historyFile, _ := cmd.Flags().GetString(HistoryFileFlag)
	if historyFile == "" {
		return "", errors.New("no history file specified")
	}
	return historyFile, nil
}

func saveHistory(cmd *cobra.Command, results compute.Results, historyLimit int, cfg *config.Config) error {
	h, err := getHistory(cmd)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			path, _ := cmd.Flags().GetString(HistoryFileFlag)
			h = history.New(path)
		} else {
			return fmt.Errorf("failed to load history: %w", err)
		}
	}

	label, _ := cmd.Flags().GetString(HistoryLabelFlag)
	h.AddResults(results, label)

	if err := h.Save(historyLimit); err != nil {
		return err
	}

	// Only show success messages for non-JSON/YAML formats
	if cfg.Format != config.FormatJSON && cfg.Format != config.FormatYAML {
		if label != "" {
			fmt.Printf("≡ Saved history entry with label: %s\n", label)
		} else {
			fmt.Printf("≡ Saved history entry\n")
		}
	}
	return nil
}

func compareHistory(cmd *cobra.Command, compareRef string, results compute.Results) error {
	h, err := getHistory(cmd)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	refEntry := h.FindByRef(compareRef)
	if refEntry == nil {
		return fmt.Errorf("no history entry found for ref: %s", compareRef)
	}
	output.CompareHistory(compareRef, refEntry, results)
	return nil
}

func showHistory(cmd *cobra.Command, historyLimit int, cfg *config.Config) error {
	h, err := getHistory(cmd)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	output.ShowHistory(h, historyLimit, cfg)
	return nil
}

func deleteHistory(cmd *cobra.Command, deleteRef string, historyLimit int) error {
	h, err := getHistory(cmd)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	deleted := h.DeleteByRef(deleteRef)
	if !deleted {
		return fmt.Errorf("no history entry found for ref: %s", deleteRef)
	}

	if err := h.Save(historyLimit); err != nil {
		return fmt.Errorf("failed to save history after deletion: %w", err)
	}

	fmt.Printf("≡ Deleted history entry for ref: %s\n", deleteRef)

	// Check if history is now empty and prompt for file removal in interactive mode
	if len(h.Entries) == 0 {
		return promptForHistoryFileRemoval(cmd)
	}

	return nil
}

func promptForHistoryFileRemoval(cmd *cobra.Command) error {
	// Only prompt if stdin is a terminal (interactive mode)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	fmt.Print("History file is now empty. Remove the history file? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return nil
	}

	historyPath, err := getHistoryPath(cmd)
	if err != nil {
		return fmt.Errorf("failed to get history path: %w", err)
	}

	if err := os.Remove(historyPath); err != nil {
		return fmt.Errorf("failed to remove history file: %w", err)
	}

	fmt.Println("≡ History file removed")
	return nil
}
