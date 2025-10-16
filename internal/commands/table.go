package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// renderTable prints a simple ASCII table with headers and rows using tabwriter.
// headers must match the number of columns in each row.
func renderTable(out io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	// Separator (approximation with dashes matching header lengths)
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("-", max(3, len(h)))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	_ = tw.Flush()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// convenience to write to stdout
func renderTableStdout(headers []string, rows [][]string) { renderTable(os.Stdout, headers, rows) }
