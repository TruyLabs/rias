package module

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/norenis/kai/internal/brain"
)

const sheetsAPIBase = "https://sheets.googleapis.com/v4/spreadsheets"

// googleSheetsModule reads rows from a Google Sheet and imports them as brain knowledge.
type googleSheetsModule struct {
	apiKey        string
	spreadsheetID string
	cellRange     string
	category      string
	topic         string
}

// NewGoogleSheetsModule creates a google_sheets module from a raw config map.
//
// Required config keys:
//   - api_key: Google API key (the sheet must be publicly accessible)
//   - spreadsheet_id: the ID from the sheet URL
//
// Optional config keys:
//   - range: cell range to read, e.g. "Sheet1!A:Z" (default: "Sheet1!A:Z")
//   - category: brain category to store into (default: "knowledge")
//   - topic: brain file slug (default: derived from spreadsheet_id)
func NewGoogleSheetsModule(cfg map[string]interface{}) (Module, error) {
	apiKey := cfgString(cfg, "api_key", "")
	if apiKey == "" {
		return nil, fmt.Errorf("google_sheets: 'api_key' is required")
	}
	spreadsheetID := cfgString(cfg, "spreadsheet_id", "")
	if spreadsheetID == "" {
		return nil, fmt.Errorf("google_sheets: 'spreadsheet_id' is required")
	}

	idSlug := spreadsheetID
	if len(idSlug) > 12 {
		idSlug = idSlug[:12]
	}

	return &googleSheetsModule{
		apiKey:        apiKey,
		spreadsheetID: spreadsheetID,
		cellRange:     cfgString(cfg, "range", "Sheet1!A:Z"),
		category:      cfgString(cfg, "category", "knowledge"),
		topic:         cfgString(cfg, "topic", "google-sheet-"+idSlug),
	}, nil
}

func (m *googleSheetsModule) Name() string        { return "google_sheets" }
func (m *googleSheetsModule) Description() string { return "Read a Google Sheet into the brain" }

type sheetsValuesResponse struct {
	Values [][]interface{} `json:"values"`
}

func (m *googleSheetsModule) Fetch(ctx context.Context) ([]brain.Learning, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	endpoint := fmt.Sprintf("%s/%s/values/%s?key=%s",
		sheetsAPIBase,
		url.PathEscape(m.spreadsheetID),
		url.PathEscape(m.cellRange),
		url.QueryEscape(m.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google_sheets: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("google_sheets: Sheets API %d: %s", resp.StatusCode, string(body))
	}

	var result sheetsValuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("google_sheets: decode: %w", err)
	}

	if len(result.Values) == 0 {
		return nil, nil
	}

	return []brain.Learning{{
		Category:   m.category,
		Topic:      m.topic,
		Tags:       []string{"google-sheets", "spreadsheet"},
		Content:    valuesToCompactTable(m.spreadsheetID, m.cellRange, result.Values),
		Confidence: "high",
		Action:     "replace",
		Source:     "module:google_sheets",
	}}, nil
}

// valuesToCompactTable converts sheet rows into a compact pipe-separated table.
// Omits markdown separator rows and decorative headers to minimise token count.
// Empty rows are skipped. Cells are whitespace-trimmed.
func valuesToCompactTable(spreadsheetID, cellRange string, values [][]interface{}) string {
	var sb strings.Builder

	// Compact single-line metadata — no markdown decoration.
	sb.WriteString(fmt.Sprintf("sheet:%s range:%s\n", spreadsheetID, cellRange))

	for _, row := range values {
		// Skip rows that are entirely empty.
		hasValue := false
		for _, cell := range row {
			if strings.TrimSpace(fmt.Sprintf("%v", cell)) != "" {
				hasValue = true
				break
			}
		}
		if !hasValue {
			continue
		}

		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = strings.TrimSpace(fmt.Sprintf("%v", cell))
		}
		sb.WriteString(strings.Join(cells, " | "))
		sb.WriteByte('\n')
	}

	return sb.String()
}
