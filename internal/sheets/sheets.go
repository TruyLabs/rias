package sheets

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client wraps the Google Sheets API.
type Client struct {
	svc *sheets.Service
}

// NewClient creates a Sheets client using a service account JSON key file.
func NewClient(ctx context.Context, serviceAccountPath string) (*Client, error) {
	svc, err := sheets.NewService(ctx, option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// ReadResult holds the output of a sheet read.
type ReadResult struct {
	SpreadsheetID string     `json:"spreadsheet_id"`
	Title         string     `json:"title"`
	Range         string     `json:"range"`
	Headers       []string   `json:"headers"`
	Rows          [][]string `json:"rows"`
	Markdown      string     `json:"markdown"`
}

// Read fetches data from a spreadsheet. If readRange is empty, it reads the first sheet.
func (c *Client) Read(ctx context.Context, spreadsheetID, readRange string) (*ReadResult, error) {
	// Get spreadsheet metadata for title and default sheet name.
	meta, err := c.svc.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get spreadsheet: %w", err)
	}

	title := meta.Properties.Title

	if readRange == "" && len(meta.Sheets) > 0 {
		readRange = meta.Sheets[0].Properties.Title
	}

	resp, err := c.svc.Spreadsheets.Values.Get(spreadsheetID, readRange).
		ValueRenderOption("FORMATTED_VALUE").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read range %q: %w", readRange, err)
	}

	if len(resp.Values) == 0 {
		return &ReadResult{
			SpreadsheetID: spreadsheetID,
			Title:         title,
			Range:         readRange,
			Markdown:      "(empty sheet)",
		}, nil
	}

	// First row as headers.
	headers := toStrings(resp.Values[0])

	var rows [][]string
	for _, row := range resp.Values[1:] {
		rows = append(rows, toStrings(row))
	}

	md := toMarkdownTable(headers, rows)

	return &ReadResult{
		SpreadsheetID: spreadsheetID,
		Title:         title,
		Range:         readRange,
		Headers:       headers,
		Rows:          rows,
		Markdown:      md,
	}, nil
}

func toStrings(row []interface{}) []string {
	out := make([]string, len(row))
	for i, v := range row {
		out[i] = fmt.Sprintf("%v", v)
	}
	return out
}

func toMarkdownTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return "(no data)"
	}

	var sb strings.Builder

	// Header row.
	sb.WriteString("| ")
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString(" |\n")

	// Separator.
	sb.WriteString("|")
	for range headers {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Data rows.
	colCount := len(headers)
	for _, row := range rows {
		sb.WriteString("| ")
		padded := make([]string, colCount)
		for i := range padded {
			if i < len(row) {
				padded[i] = row[i]
			}
		}
		// Include extra columns beyond headers.
		if len(row) > colCount {
			padded = append(padded, row[colCount:]...)
		}
		sb.WriteString(strings.Join(padded, " | "))
		sb.WriteString(" |\n")
	}

	return sb.String()
}
