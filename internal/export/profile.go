package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type DatasetProfile struct {
	DatasetID   string   `json:"dataset_id"`
	SourceFile  string   `json:"source_file"`
	SheetName   string   `json:"sheet_name"`
	RowCount    int      `json:"row_count"`
	ColumnCount int      `json:"column_count"`
	ColumnNames []string `json:"column_names"`
}

func WriteProfiles(path string, profiles []DatasetProfile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir profiles dir: %w", err)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].DatasetID == profiles[j].DatasetID {
			if profiles[i].SourceFile == profiles[j].SourceFile {
				return profiles[i].SheetName < profiles[j].SheetName
			}
			return profiles[i].SourceFile < profiles[j].SourceFile
		}
		return profiles[i].DatasetID < profiles[j].DatasetID
	})

	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write profiles: %w", err)
	}

	return nil
}
