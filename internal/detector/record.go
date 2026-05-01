package detector

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// DetectorRecord es la representación plana de una fila del CSV enriquecido.
type DetectorRecord struct {
	Index             int
	DatasetID         string
	Expediente        string
	TipoExpediente    string
	FechaPresentacion string
	Denunciante       string
	Denunciado        string
	RUC               string
	Materia           string
	Queja             string
	Tokens            []string // pre-computado al cargar
}

// LoadCSV carga el CSV enriquecido y pre-tokeniza el campo queja de cada fila.
func LoadCSV(path string) ([]DetectorRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descartar header
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("leer header: %w", err)
	}

	var records []DetectorRecord
	idx := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // fila malformada, saltar
		}

		get := func(i int) string {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}

		sourceRow, _ := strconv.Atoi(get(2))
		_ = sourceRow

		rec := DetectorRecord{
			Index:             idx,
			DatasetID:         get(0),
			Expediente:        get(3),
			TipoExpediente:    get(4),
			FechaPresentacion: get(5),
			Denunciante:       get(6),
			Denunciado:        get(7),
			RUC:               get(8),
			Materia:           get(9),
			Queja:             get(12),
		}
		rec.Tokens = Tokenize(rec.Queja)
		records = append(records, rec)
		idx++
	}
	return records, nil
}
