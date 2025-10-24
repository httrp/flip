package commands

import (
	"testing"

	"github.com/httrp/flip/internal/tasks"
)

func TestNewTaskBrowserModel_InitialFilter(t *testing.T) {
	// Create some test tasks
	testTasks := []*tasks.Task{
		{
			Description: "Open task",
			Status:      tasks.StatusOpen,
		},
		{
			Description: "Done task",
			Status:      tasks.StatusDone,
		},
		{
			Description: "In progress task",
			Status:      tasks.StatusInProgress,
		},
	}

	// Create model
	model := NewTaskBrowser(testTasks)

	// Check initial filters
	if model.scopeFilter != "all" {
		t.Errorf("Expected scopeFilter to be 'all', got '%s'", model.scopeFilter)
	}
	if model.statusFilter != "open" {
		t.Errorf("Expected statusFilter to be 'open', got '%s'", model.statusFilter)
	}

	// Check that filter was applied (should only show open tasks)
	if len(model.filteredTasks) != 1 {
		t.Errorf("Expected 1 filtered task (open), got %d", len(model.filteredTasks))
	}

	// Verify the filtered task is the open one
	if len(model.filteredTasks) > 0 && model.filteredTasks[0].Status != tasks.StatusOpen {
		t.Errorf("Expected filtered task to be open, got status %v", model.filteredTasks[0].Status)
	}
}

func TestNewTaskBrowserModel_AllTasksAvailable(t *testing.T) {
	testTasks := []*tasks.Task{
		{Description: "Task 1", Status: tasks.StatusOpen},
		{Description: "Task 2", Status: tasks.StatusDone},
		{Description: "Task 3", Status: tasks.StatusInProgress},
	}

	model := NewTaskBrowser(testTasks)

	// All tasks should be stored
	if len(model.tasks) != 3 {
		t.Errorf("Expected 3 total tasks, got %d", len(model.tasks))
	}
}
