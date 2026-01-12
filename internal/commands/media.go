package commands

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/httrp/flip/internal/health"
    "github.com/spf13/cobra"
)

// NewMediaCommand groups media-related utilities
func NewMediaCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "media",
        Short: "Media utilities (normalize assets)",
    }

    cmd.AddCommand(newMediaNormalizeCommand())
    return cmd
}

func newMediaNormalizeCommand() *cobra.Command {
    var jsonOutput bool
    var fix bool
    var dryRun bool

    cmd := &cobra.Command{
        Use:   "normalize [path]",
        Short: "Normalize media filenames and locations",
        Long: "Scan a brain for media issues (generic names, wrong locations) and optionally fix them.\n" +
            "- Flags: --fix to apply repairs, --dry-run with --fix to preview only, --json for structured output.",
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            brainPath := "."
            if len(args) > 0 {
                brainPath = args[0]
            }
            return runMediaNormalize(brainPath, jsonOutput, fix, dryRun)
        },
    }

    cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
    cmd.Flags().BoolVarP(&fix, "fix", "f", false, "Automatically fix media issues where possible")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview repairs without applying them (use with --fix)")

    return cmd
}

// mediaJSONResponse mirrors health JSON structure but filters to media
// We reuse healthJSONRepairPayload from brain_check.go (same package)
type mediaJSONResponse struct {
    Result  *health.CheckResult      `json:"result"`
    Repairs *healthJSONRepairPayload `json:"repairs,omitempty"`
    Notes   map[string]any           `json:"notes,omitempty"`
}

func runMediaNormalize(brainPath string, jsonOutput, fix, dryRun bool) error {
    checker, err := health.NewChecker(brainPath)
    if err != nil {
        return fmt.Errorf("failed to init health checker: %w", err)
    }

    result, err := checker.Check()
    if err != nil {
        return fmt.Errorf("failed to run health check: %w", err)
    }

    mediaIssues := filterMediaIssues(result.Issues)
    recalcMediaStats(result, mediaIssues)

    if jsonOutput {
        resp := mediaJSONResponse{Result: result}
        if fix && len(mediaIssues) > 0 {
            repairs, stats := repairMediaIssues(brainPath, mediaIssues, dryRun)
            resp.Repairs = &healthJSONRepairPayload{Results: repairs, Stats: stats, NotRepairableCount: stats.NotRepairable}
        }
        
        // Wrap in standard FlipResult format for consistency
        output := map[string]interface{}{
            "success": true,
            "command": "media-normalize",
            "data":    resp,
        }
        
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(output)
    }

    // Human-readable output
    fmt.Printf("📸 Media normalize in %s\n", brainPath)
    if len(mediaIssues) == 0 {
        fmt.Println("✓ No media issues found")
    } else {
        fmt.Printf("Found %d media issue(s):\n", len(mediaIssues))
        for _, issue := range mediaIssues {
            fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.File, issue.Message)
            if issue.Details != "" {
                fmt.Printf("    %s\n", issue.Details)
            }
        }
    }

    if fix && len(mediaIssues) > 0 {
        repairs, stats := repairMediaIssues(brainPath, mediaIssues, dryRun)
        fmt.Printf("\n🔧 Repairs: %d succeeded, %d failed, %d skipped, %d not repairable\n", stats.Repaired, stats.Failed, stats.Skipped, stats.NotRepairable)
        if dryRun {
            fmt.Println("(dry-run, no files changed)")
        }
        for _, r := range repairs {
            status := "OK"
            if !r.Success {
                if r.SkipReason != "" {
                    status = "SKIP"
                } else {
                    status = "FAIL"
                }
            }
            fmt.Printf("- [%s] %s\n", status, r.Issue.File)
            if r.Message != "" {
                fmt.Printf("    %s\n", r.Message)
            }
            if r.SkipReason != "" {
                fmt.Printf("    %s\n", r.SkipReason)
            }
        }
    } else if fix {
        fmt.Println("No media issues to fix.")
    }

    return nil
}

func filterMediaIssues(issues []health.Issue) []health.Issue {
    filtered := make([]health.Issue, 0)
    for _, issue := range issues {
        switch issue.Type {
        case health.IssueTypeWrongMediaFilename, health.IssueTypeWrongMediaLocation:
            filtered = append(filtered, issue)
        }
    }
    return filtered
}

func recalcMediaStats(result *health.CheckResult, issues []health.Issue) {
    result.Issues = issues
    result.Stats.ErrorCount = 0
    result.Stats.WarningCount = 0
    result.Stats.InfoCount = 0
    for _, issue := range issues {
        switch issue.Severity {
        case health.SeverityError:
            result.Stats.ErrorCount++
        case health.SeverityWarning:
            result.Stats.WarningCount++
        case health.SeverityInfo:
            result.Stats.InfoCount++
        }
    }
}

func repairMediaIssues(brainPath string, issues []health.Issue, dryRun bool) ([]health.RepairResult, health.RepairStats) {
    repairer, err := health.NewRepairer(brainPath, dryRun)
    if err != nil {
        return []health.RepairResult{{
            Issue:   health.Issue{Message: "failed to init repairer"},
            Success: false,
            Message: err.Error(),
        }}, health.RepairStats{Failed: 1, TotalIssues: 1}
    }

    // Only pass issues that are repairable by media actions
    fixable := make([]health.Issue, 0)
    for _, issue := range issues {
        if repairer.CanRepair(issue.Type) {
            fixable = append(fixable, issue)
        }
    }

    results := repairer.RepairIssues(fixable)
    stats := health.CalculateStats(results)
    return results, stats
}
