package migration

import (
	"fmt"
	"strings"
	"time"
)

// ProgressCallback is called during migration to report progress
type ProgressCallback func(current, total int, item string)

// ProgressBar displays a simple terminal progress bar
type ProgressBar struct {
	Total     int
	Current   int
	Width     int
	StartTime time.Time
	LastPrint time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int) *ProgressBar {
	return &ProgressBar{
		Total:     total,
		Current:   0,
		Width:     50,
		StartTime: time.Now(),
		LastPrint: time.Now(),
	}
}

// Update increments the progress and displays the bar
func (pb *ProgressBar) Update(itemName string) {
	pb.Current++
	
	// Only update display every 100ms to avoid flicker
	if time.Since(pb.LastPrint) < 100*time.Millisecond && pb.Current < pb.Total {
		return
	}
	pb.LastPrint = time.Now()
	
	pb.Display(itemName)
}

// Display renders the progress bar
func (pb *ProgressBar) Display(itemName string) {
	percent := float64(pb.Current) / float64(pb.Total) * 100
	filled := int(float64(pb.Width) * float64(pb.Current) / float64(pb.Total))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.Width-filled)
	
	// Calculate ETA
	elapsed := time.Since(pb.StartTime)
	var eta string
	if pb.Current > 0 {
		avgTime := elapsed / time.Duration(pb.Current)
		remaining := time.Duration(pb.Total-pb.Current) * avgTime
		eta = formatDuration(remaining)
	} else {
		eta = "calculating..."
	}
	
	// Truncate item name if too long
	maxItemLen := 40
	if len(itemName) > maxItemLen {
		itemName = itemName[:maxItemLen-3] + "..."
	}
	
	// Print progress bar (overwrite previous line)
	fmt.Printf("\r[%s] %3.0f%% (%d/%d) ETA: %s | %s", 
		bar, percent, pb.Current, pb.Total, eta, itemName)
	
	// Add newline on completion
	if pb.Current >= pb.Total {
		fmt.Println()
	}
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish() {
	pb.Current = pb.Total
	pb.Display("Complete")
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
