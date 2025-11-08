package commands

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TaskHistory stores recently used values for task creation
type TaskHistory struct {
	RecentOrganizations []string `yaml:"recent_organizations"`
	RecentProjects      []string `yaml:"recent_projects"`
	RecentContexts      []string `yaml:"recent_contexts"`
	RecentFiles         []string `yaml:"recent_files"`
}

// getTaskHistoryPath returns the path to the task history file
func getTaskHistoryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	flipDir := filepath.Join(configDir, "flip")
	if err := os.MkdirAll(flipDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(flipDir, "task-history.yaml"), nil
}

// loadTaskHistory loads the task history from disk
func loadTaskHistory() (*TaskHistory, error) {
	path, err := getTaskHistoryPath()
	if err != nil {
		return &TaskHistory{}, err
	}

	// If file doesn't exist, return empty history
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &TaskHistory{
			RecentOrganizations: []string{},
			RecentProjects:      []string{},
			RecentContexts:      []string{},
			RecentFiles:         []string{},
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &TaskHistory{}, err
	}

	var history TaskHistory
	if err := yaml.Unmarshal(data, &history); err != nil {
		return &TaskHistory{}, err
	}

	return &history, nil
}

// saveTaskHistory saves the task history to disk
func saveTaskHistory(history *TaskHistory) error {
	path, err := getTaskHistoryPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(history)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// addToHistory adds a value to a history list, maintaining max size and uniqueness
func addToHistory(list []string, value string, maxSize int) []string {
	if value == "" {
		return list
	}

	// Remove value if it already exists (we'll add it to front)
	filtered := []string{}
	for _, item := range list {
		if item != value {
			filtered = append(filtered, item)
		}
	}

	// Add to front
	result := append([]string{value}, filtered...)

	// Trim to max size
	if len(result) > maxSize {
		result = result[:maxSize]
	}

	return result
}

// updateTaskHistory updates the history with new values
func updateTaskHistory(organization, project, context, file string) error {
	history, err := loadTaskHistory()
	if err != nil {
		return err
	}

	history.RecentOrganizations = addToHistory(history.RecentOrganizations, organization, 10)
	history.RecentProjects = addToHistory(history.RecentProjects, project, 10)
	history.RecentContexts = addToHistory(history.RecentContexts, context, 10)
	history.RecentFiles = addToHistory(history.RecentFiles, file, 5)

	return saveTaskHistory(history)
}
