package brain

import (
	"strings"
)

// EscapeMarkdownTableCell escapes special characters in markdown table cells
func EscapeMarkdownTableCell(s string) string {
	// Remove or replace newlines to prevent breaking table structure
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Collapse multiple spaces into one
	s = strings.Join(strings.Fields(s), " ")

	// Escape HTML special characters to prevent markdown interpretation
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	// Escape pipe characters for markdown tables
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// EscapePipeInMarkdown escapes pipe characters in markdown table cells (deprecated, use EscapeMarkdownTableCell)
func EscapePipeInMarkdown(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// BuildMarkdownTable creates a markdown table from rows, escaping pipes and HTML special chars
func BuildMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	var buf strings.Builder
	header := rows[0]

	// Write header
	buf.WriteString("| ")
	for i, h := range header {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(EscapeMarkdownTableCell(strings.TrimSpace(h)))
	}
	buf.WriteString(" |\n")

	// Write separator
	buf.WriteString("|")
	for range header {
		buf.WriteString(" --- |")
	}
	buf.WriteString("\n")

	// Write data rows
	for _, row := range rows[1:] {
		// Pad row to match header length
		for len(row) < len(header) {
			row = append(row, "")
		}
		buf.WriteString("| ")
		for i := 0; i < len(header); i++ {
			if i > 0 {
				buf.WriteString(" | ")
			}
			buf.WriteString(EscapeMarkdownTableCell(strings.TrimSpace(row[i])))
		}
		buf.WriteString(" |\n")
	}

	return buf.String()
}
