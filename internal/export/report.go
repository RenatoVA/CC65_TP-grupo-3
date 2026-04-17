package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Report struct {
	DatasetsConfigured int      `json:"datasets_configured"`
	FilesDownloaded    int      `json:"files_downloaded"`
	RowsConsolidated   int      `json:"rows_consolidated"`
	DuplicateKeys      int      `json:"duplicate_keys"`
	Warnings           []string `json:"warnings"`
}

func WriteReport(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir report dir: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}
