package tasks

import (
	"os"
	"path/filepath"
	"strings"
)

// Scanner scans markdown files for tasks
type Scanner struct {
	brainPath string
}

// NewScanner creates a new task scanner
func NewScanner(brainPath string) *Scanner {
	return &Scanner{
		brainPath: brainPath,
	}
}

// ScanBrain scans all markdown files in a brain for tasks
func (s *Scanner) ScanBrain() ([]Task, error) {
	var allTasks []Task

	// Supported extensions
	extensions := map[string]bool{
		".md":       true,
		".markdown": true,
	}

	err := filepath.Walk(s.brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip common exclude directories
			name := info.Name()
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a markdown file
		ext := strings.ToLower(filepath.Ext(path))
		if !extensions[ext] {
			return nil
		}

		// Scan file for tasks
		tasks, err := s.ScanFile(path)
		if err != nil {
			// Log error but continue scanning
			return nil
		}

		allTasks = append(allTasks, tasks...)
		return nil
	})

	return allTasks, err
}

// ScanFile scans a single markdown file for tasks
func (s *Scanner) ScanFile(filePath string) ([]Task, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var tasks []Task
	var currentTask *Task
	var metadataLines []string
	var currentSection string

	// Create context
	context := TaskContext{
		BrainPath: s.brainPath,
		FilePath:  filePath,
		FileName:  filepath.Base(filePath),
	}

	for i, line := range lines {
		lineNum := i + 1

		// Track current section (markdown headings)
		if strings.HasPrefix(line, "#") {
			currentSection = strings.TrimSpace(strings.TrimLeft(line, "#"))
		}

		// Check if line is a task
		if IsTaskLine(line) {
			// Save previous task with its metadata
			if currentTask != nil {
				if len(metadataLines) > 0 {
					ParseTaskMetadata(currentTask, metadataLines)
				}
				tasks = append(tasks, *currentTask)
				metadataLines = nil
			}

			// Set context section
			context.Section = currentSection
			context.LineNumber = lineNum

			// Parse new task
			task, err := ParseTask(line, lineNum, context)
			if err != nil {
				continue
			}
			if task == nil {
				continue
			}

			currentTask = task
		} else if currentTask != nil && IsMetadataLine(line) {
			// Collect metadata for current task
			metadataLines = append(metadataLines, line)
		} else {
			// Not a task or metadata - save current task if any
			if currentTask != nil {
				if len(metadataLines) > 0 {
					ParseTaskMetadata(currentTask, metadataLines)
				}
				tasks = append(tasks, *currentTask)
				currentTask = nil
				metadataLines = nil
			}
		}
	}

	// Save last task if any
	if currentTask != nil {
		if len(metadataLines) > 0 {
			ParseTaskMetadata(currentTask, metadataLines)
		}
		tasks = append(tasks, *currentTask)
	}

	return tasks, nil
}

// ScanMultipleBrains scans multiple brains for tasks
func ScanMultipleBrains(brainPaths map[string]string) ([]Task, error) {
	var allTasks []Task

	for brainName, brainPath := range brainPaths {
		scanner := NewScanner(brainPath)
		tasks, err := scanner.ScanBrain()
		if err != nil {
			// Log but continue with other brains
			continue
		}

		// Set brain name in context
		for i := range tasks {
			tasks[i].Context.BrainName = brainName
		}

		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}
