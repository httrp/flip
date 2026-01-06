package tasks

import (
	"sort"
	"strings"
	"time"
)

// TaskFilter defines criteria for filtering tasks
type TaskFilter struct {
	Status       Status
	Priority     Priority
	Tags         []string
	Project      string
	Organization string
	Context      string
	DueBefore    *time.Time
	DueAfter     *time.Time
	Overdue      bool
	DueToday     bool
	DueThisWeek  bool
	FrogOnly     bool   // Only show 🐸 eat-the-frog tasks
	SearchText   string
	SortBy       string // "due", "priority", "created"
}

// TaskIndex provides fast task queries and filtering
type TaskIndex struct {
	tasks      []*Task
	byStatus   map[Status][]*Task
	byPriority map[Priority][]*Task
	byTag      map[string][]*Task
	byProject  map[string][]*Task
	byDueDate  map[string][]*Task
}

// NewTaskIndex creates a new task index
func NewTaskIndex() *TaskIndex {
	return &TaskIndex{
		tasks:      []*Task{},
		byStatus:   make(map[Status][]*Task),
		byPriority: make(map[Priority][]*Task),
		byTag:      make(map[string][]*Task),
		byProject:  make(map[string][]*Task),
		byDueDate:  make(map[string][]*Task),
	}
}

// Build creates indices from a task list
func (idx *TaskIndex) Build(tasks []Task) {
	// Clear existing indices
	idx.tasks = make([]*Task, 0, len(tasks))
	idx.byStatus = make(map[Status][]*Task)
	idx.byPriority = make(map[Priority][]*Task)
	idx.byTag = make(map[string][]*Task)
	idx.byProject = make(map[string][]*Task)
	idx.byDueDate = make(map[string][]*Task)

	// Build indices
	for i := range tasks {
		task := &tasks[i]
		idx.tasks = append(idx.tasks, task)

		// Index by status
		idx.byStatus[task.Status] = append(idx.byStatus[task.Status], task)

		// Index by priority
		if task.Priority != PriorityNone {
			idx.byPriority[task.Priority] = append(idx.byPriority[task.Priority], task)
		}

		// Index by tags
		for _, tag := range task.Tags {
			idx.byTag[tag] = append(idx.byTag[tag], task)
		}

		// Index by project
		if task.Project != "" {
			idx.byProject[task.Project] = append(idx.byProject[task.Project], task)
		}

		// Index by due date
		if task.Due != nil {
			dateKey := task.Due.Format("2006-01-02")
			idx.byDueDate[dateKey] = append(idx.byDueDate[dateKey], task)
		}
	}
}

// Query filters tasks based on criteria
func (idx *TaskIndex) Query(filter TaskFilter) []*Task {
	var results []*Task

	// Start with status filter if specified
	if filter.Status != "" {
		results = append(results, idx.byStatus[filter.Status]...)
	} else {
		// Start with all tasks
		results = append(results, idx.tasks...)
	}

	// Apply additional filters
	results = idx.applyFilters(results, filter)

	// Sort
	idx.sortTasks(results, filter.SortBy)

	return results
}

// applyFilters applies all filter criteria
func (idx *TaskIndex) applyFilters(tasks []*Task, filter TaskFilter) []*Task {
	var filtered []*Task

	for _, task := range tasks {
		// Priority filter
		if filter.Priority != "" && task.Priority != filter.Priority {
			continue
		}

		// Frog filter (eat-the-frog)
		if filter.FrogOnly && !task.Frog {
			continue
		}

		// Tags filter (all tags must match)
		if len(filter.Tags) > 0 {
			if !hasAllTags(task, filter.Tags) {
				continue
			}
		}

		// Project filter
		if filter.Project != "" && task.Project != filter.Project {
			continue
		}

		// Organization filter
		if filter.Organization != "" && task.Organization != filter.Organization {
			continue
		}

		// Context filter
		if filter.Context != "" && task.ContextTag != filter.Context {
			continue
		}

		// Due date filters
		if filter.DueBefore != nil && (task.Due == nil || task.Due.After(*filter.DueBefore)) {
			continue
		}

		if filter.DueAfter != nil && (task.Due == nil || task.Due.Before(*filter.DueAfter)) {
			continue
		}

		// Overdue filter
		if filter.Overdue && !task.IsOverdue() {
			continue
		}

		// Due today filter
		if filter.DueToday && !task.IsDueToday() {
			continue
		}

		// Due this week filter
		if filter.DueThisWeek && !task.IsDueThisWeek() {
			continue
		}

		// Search text filter
		if filter.SearchText != "" {
			searchLower := strings.ToLower(filter.SearchText)
			descLower := strings.ToLower(task.Description)
			if !strings.Contains(descLower, searchLower) {
				continue
			}
		}

		filtered = append(filtered, task)
	}

	return filtered
}

// sortTasks sorts tasks by the specified field
func (idx *TaskIndex) sortTasks(tasks []*Task, sortBy string) {
	switch sortBy {
	case "due":
		sort.Slice(tasks, func(i, j int) bool {
			// Tasks with no due date go last
			if tasks[i].Due == nil {
				return false
			}
			if tasks[j].Due == nil {
				return true
			}
			return tasks[i].Due.Before(*tasks[j].Due)
		})
	case "priority":
		sort.Slice(tasks, func(i, j int) bool {
			return priorityValue(tasks[i].Priority) > priorityValue(tasks[j].Priority)
		})
	case "created":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Created.After(tasks[j].Created)
		})
	default:
		// Default: sort by due date, then priority
		sort.Slice(tasks, func(i, j int) bool {
			// Overdue tasks first
			iOverdue := tasks[i].IsOverdue()
			jOverdue := tasks[j].IsOverdue()
			if iOverdue != jOverdue {
				return iOverdue
			}

			// Then by due date
			if tasks[i].Due != nil && tasks[j].Due != nil {
				if !tasks[i].Due.Equal(*tasks[j].Due) {
					return tasks[i].Due.Before(*tasks[j].Due)
				}
			} else if tasks[i].Due != nil {
				return true // Tasks with due date before those without
			} else if tasks[j].Due != nil {
				return false
			}

			// Then by priority
			if tasks[i].Priority != tasks[j].Priority {
				return priorityValue(tasks[i].Priority) > priorityValue(tasks[j].Priority)
			}

			// Finally by created date
			return tasks[i].Created.After(tasks[j].Created)
		})
	}
}

// hasAllTags checks if a task has all specified tags
func hasAllTags(task *Task, tags []string) bool {
	taskTags := make(map[string]bool)
	for _, tag := range task.Tags {
		taskTags[strings.ToLower(tag)] = true
	}

	for _, requiredTag := range tags {
		if !taskTags[strings.ToLower(requiredTag)] {
			return false
		}
	}

	return true
}

// priorityValue converts priority to numeric value for sorting
func priorityValue(p Priority) int {
	switch p {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

// GetOpenTasks returns all open tasks
func (idx *TaskIndex) GetOpenTasks() []*Task {
	return idx.byStatus[StatusOpen]
}

// GetOverdueTasks returns all overdue tasks
func (idx *TaskIndex) GetOverdueTasks() []*Task {
	var overdue []*Task
	for _, task := range idx.tasks {
		if task.IsOverdue() {
			overdue = append(overdue, task)
		}
	}
	return overdue
}

// GetTasksDueToday returns tasks due today
func (idx *TaskIndex) GetTasksDueToday() []*Task {
	today := time.Now().Format("2006-01-02")
	return idx.byDueDate[today]
}

// GetTasksDueThisWeek returns tasks due in the next 7 days
func (idx *TaskIndex) GetTasksDueThisWeek() []*Task {
	var thisWeek []*Task
	for _, task := range idx.tasks {
		if task.IsDueThisWeek() {
			thisWeek = append(thisWeek, task)
		}
	}
	return thisWeek
}

// GetTasksByProject returns all tasks for a specific project
func (idx *TaskIndex) GetTasksByProject(project string) []*Task {
	return idx.byProject[project]
}

// GetTasksByTag returns all tasks with a specific tag
func (idx *TaskIndex) GetTasksByTag(tag string) []*Task {
	return idx.byTag[tag]
}

// Count returns the total number of tasks
func (idx *TaskIndex) Count() int {
	return len(idx.tasks)
}

// Stats returns statistics about tasks
func (idx *TaskIndex) Stats() TaskStats {
	stats := TaskStats{
		Total: len(idx.tasks),
	}

	for _, task := range idx.tasks {
		switch task.Status {
		case StatusOpen:
			stats.Open++
		case StatusInProgress:
			stats.InProgress++
		case StatusDone:
			stats.Done++
		case StatusDeferred:
			stats.Deferred++
		case StatusCancelled:
			stats.Cancelled++
		}

		if task.IsOverdue() {
			stats.Overdue++
		}
		if task.IsDueToday() {
			stats.DueToday++
		}
		if task.IsDueThisWeek() {
			stats.DueThisWeek++
		}
	}

	return stats
}

// TaskStats contains statistics about tasks
type TaskStats struct {
	Total       int
	Open        int
	InProgress  int
	Done        int
	Deferred    int
	Cancelled   int
	Overdue     int
	DueToday    int
	DueThisWeek int
}
