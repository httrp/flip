package commands

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/httrp/flip/internal/tasks"
)

type TaskStatusOptions struct {
	ID     string
	File   string
	Line   int
	Status string
}

// NewTaskStatusCommand updates the status of an existing task by ID or file/line.
func NewTaskStatusCommand() *cobra.Command {
	opts := TaskStatusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Update the status of an existing task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ID == "" && opts.File == "" {
				return errors.New("provide either --id or --file with --line to identify the task")
			}

			if opts.File != "" && opts.Line == 0 {
				return errors.New("when using --file you must also provide --line")
			}

			status, err := parseStatus(opts.Status)
			if err != nil {
				return err
			}

			activeWs, err := getActiveWorkspace()
			if err != nil {
				return err
			}

			brainPaths := make(map[string]string)
			for _, b := range activeWs.Brains {
				brainPaths[b.Name] = b.Path
			}

			var candidate *tasks.Task
			if opts.ID != "" {
				candidate, err = tasks.FindTaskByID(brainPaths, opts.ID)
				if err != nil {
					return err
				}
			} else {
				candidate, err = findTaskByFileAndLine(opts.File, opts.Line, brainPaths)
				if err != nil {
					return err
				}
			}

			if candidate == nil {
				return errors.New("task not found")
			}

			if err := tasks.SetTaskStatus(candidate, status); err != nil {
				return fmt.Errorf("set task status: %w", err)
			}

			jsonFlag := cmd.Flag("json")
			if jsonFlag != nil && jsonFlag.Changed {
				res := TaskResult{
					Action:      "status",
					Path:        candidate.Context.FilePath,
					Line:        candidate.Context.LineNumber,
					Description: candidate.Description,
					Status:      string(candidate.Status),
					BrainName:   candidate.Context.BrainName,
					BrainPath:   candidate.Context.BrainPath,
				}
				OutputJSONSuccess("status", res)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✅ Updated task status to %s: %s\n", candidate.Status, candidate.Description)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "ID of the task to update")
	cmd.Flags().StringVar(&opts.File, "file", "", "File containing the task")
	cmd.Flags().IntVar(&opts.Line, "line", 0, "Line number of the task (1-based)")
	cmd.Flags().StringVar(&opts.Status, "status", "", "New status (open, in-progress, done, deferred, cancelled)")
	cmd.MarkFlagRequired("status")
	cmd.Flags().Bool("json", false, "Output result as JSON")

	return cmd
}

func findTaskByFileAndLine(file string, line int, brainPaths map[string]string) (*tasks.Task, error) {
	absFile, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("resolve file path: %w", err)
	}

	matches, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return nil, err
	}

	for i := range matches {
		task := &matches[i]
		if task.Context.FilePath == absFile && task.Context.LineNumber == line {
			return task, nil
		}
	}

	// fallback: if only one task in file, allow using file without exact line match
	var inFile []*tasks.Task
	for i := range matches {
		task := &matches[i]
		if task.Context.FilePath == absFile {
			inFile = append(inFile, task)
		}
	}

	if len(inFile) == 1 {
		return inFile[0], nil
	}

	if len(inFile) > 1 {
		return nil, fmt.Errorf("multiple tasks found in file; provide exact line")
	}

	return nil, fmt.Errorf("no task found in file %s", absFile)
}

func parseStatus(raw string) (tasks.Status, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))

	switch raw {
	case "open":
		return tasks.StatusOpen, nil
	case "in-progress", "in_progress", "inprogress":
		return tasks.StatusInProgress, nil
	case "done", "complete", "completed":
		return tasks.StatusDone, nil
	case "deferred", "waiting":
		return tasks.StatusDeferred, nil
	case "cancelled", "canceled":
		return tasks.StatusCancelled, nil
	default:
		return tasks.Status(""), fmt.Errorf("invalid status: %s", raw)
	}
}
