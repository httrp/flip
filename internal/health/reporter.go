package health

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Reporter formats and displays health check results
type Reporter struct {
	result *CheckResult
}

// NewReporter creates a new reporter
func NewReporter(result *CheckResult) *Reporter {
	return &Reporter{result: result}
}

// Print displays the health check results in a formatted way
func (r *Reporter) Print() {
	r.printHeader()
	r.printStats()
	r.printIssues()
	r.printSummary()
}

// printHeader displays brain information
func (r *Reporter) printHeader() {
	fmt.Println()
	color.New(color.Bold, color.FgCyan).Printf("🧠 Brain Health Check\n")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Brain:  %s\n", r.result.BrainInfo.Name)
	fmt.Printf("  Type:   %s\n", r.result.BrainInfo.Type.String())
	fmt.Printf("  Path:   %s\n", r.result.BrainInfo.Path)
	fmt.Printf("  Notes:  %d markdown files\n", r.result.BrainInfo.NoteCount)
	fmt.Printf("  Assets: %d files\n", r.result.BrainInfo.AssetCount)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
}

// printStats displays scan statistics
func (r *Reporter) printStats() {
	color.New(color.Bold).Println("📊 Scan Statistics")
	fmt.Printf("  Files scanned:   %d\n", r.result.Stats.FilesScanned)
	fmt.Printf("  Links checked:   %d\n", r.result.Stats.LinksChecked)
	fmt.Printf("  Assets checked:  %d\n", r.result.Stats.AssetsChecked)
	fmt.Println()
}

// printIssues displays all issues grouped by severity
func (r *Reporter) printIssues() {
	if len(r.result.Issues) == 0 {
		color.Green("✅ No issues found! Your brain is healthy.\n")
		fmt.Println()
		return
	}

	// Group issues by severity
	errors := make([]Issue, 0)
	warnings := make([]Issue, 0)
	infos := make([]Issue, 0)

	for _, issue := range r.result.Issues {
		switch issue.Severity {
		case SeverityError:
			errors = append(errors, issue)
		case SeverityWarning:
			warnings = append(warnings, issue)
		case SeverityInfo:
			infos = append(infos, issue)
		}
	}

	// Print errors
	if len(errors) > 0 {
		color.New(color.Bold, color.FgRed).Printf("❌ Errors (%d)\n", len(errors))
		for i, issue := range errors {
			if i >= 10 {
				fmt.Printf("  ... and %d more errors\n", len(errors)-10)
				break
			}
			r.printIssue(issue)
		}
		fmt.Println()
	}

	// Print warnings
	if len(warnings) > 0 {
		color.New(color.Bold, color.FgYellow).Printf("⚠️  Warnings (%d)\n", len(warnings))
		for i, issue := range warnings {
			if i >= 10 {
				fmt.Printf("  ... and %d more warnings\n", len(warnings)-10)
				break
			}
			r.printIssue(issue)
		}
		fmt.Println()
	}

	// Print info items
	if len(infos) > 0 {
		color.New(color.Bold, color.FgCyan).Printf("ℹ️  Info (%d)\n", len(infos))
		for i, issue := range infos {
			if i >= 5 {
				fmt.Printf("  ... and %d more info items\n", len(infos)-5)
				break
			}
			r.printIssue(issue)
		}
		fmt.Println()
	}
}

// printIssue displays a single issue
func (r *Reporter) printIssue(issue Issue) {
	// Color based on severity
	var severityColor *color.Color
	switch issue.Severity {
	case SeverityError:
		severityColor = color.New(color.FgRed)
	case SeverityWarning:
		severityColor = color.New(color.FgYellow)
	case SeverityInfo:
		severityColor = color.New(color.FgCyan)
	}

	// Print file and line
	fmt.Printf("  ")
	color.New(color.FgWhite, color.Bold).Printf("%s", issue.File)
	if issue.Line > 0 {
		fmt.Printf(":%d", issue.Line)
	}
	fmt.Println()

	// Print message
	fmt.Printf("    ")
	severityColor.Printf("→ %s\n", issue.Message)

	// Print details if available
	if issue.Details != "" {
		fmt.Printf("      %s\n", color.New(color.Faint).Sprint(issue.Details))
	}
}

// printSummary displays the final summary
func (r *Reporter) printSummary() {
	fmt.Println(strings.Repeat("─", 60))

	if r.result.Stats.ErrorCount == 0 && r.result.Stats.WarningCount == 0 {
		color.New(color.Bold, color.FgGreen).Println("✨ All checks passed!")
	} else {
		fmt.Printf("  Summary: ")
		if r.result.Stats.ErrorCount > 0 {
			color.Red("%d errors ", r.result.Stats.ErrorCount)
		}
		if r.result.Stats.WarningCount > 0 {
			color.Yellow("%d warnings ", r.result.Stats.WarningCount)
		}
		if r.result.Stats.InfoCount > 0 {
			color.Cyan("%d info", r.result.Stats.InfoCount)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
}

// PrintJSON outputs the result as JSON (for programmatic use)
func (r *Reporter) PrintJSON() error {
	data, err := json.MarshalIndent(r.result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal health report: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
