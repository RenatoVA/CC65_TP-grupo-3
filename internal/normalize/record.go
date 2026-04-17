package normalize

import (
	"fmt"
	"strings"
)

type RawRecord struct {
	DatasetID string
	SheetName string
	RowIndex  int
	Values    map[string]string
}

type ConsolidatedRecord struct {
	DatasetID         string
	SourceSheet       string
	SourceRow         int
	Expediente        string
	TipoExpediente    string
	FechaPresentacion string
	Denunciante       string
	Denunciado        string
	RUC               string
	Materia           string
	Resumen           string
	RawColumnCount    int
}

func FromMap(raw RawRecord) ConsolidatedRecord {
	normalized := make(map[string]string, len(raw.Values))
	for k, v := range raw.Values {
		nk := normalizeKey(k)
		if nk == "" {
			continue
		}
		normalized[nk] = strings.TrimSpace(v)
	}

	get := func(keys ...string) string {
		for _, key := range keys {
			if v := normalized[normalizeKey(key)]; v != "" {
				return v
			}
		}
		return ""
	}

	return ConsolidatedRecord{
		DatasetID:         raw.DatasetID,
		SourceSheet:       raw.SheetName,
		SourceRow:         raw.RowIndex,
		Expediente:        get("expediente", "nro expediente", "numero expediente", "nº expediente", "nro de expediente", "nro_expediente", "ingreso en sala"),
		TipoExpediente:    get("tipo expediente", "tipo de expediente", "tipo de exp.", "tipo_expediente", "vc tipo expediente", "procedimiento"),
		FechaPresentacion: get("fecha", "fecha presentacion", "fecha de presentacion", "fecha_presentacion", "f.pres.", "fec pre"),
		Denunciante:       get("denunciante", "denunciantes", "reclamante", "solicitante", "titular"),
		Denunciado:        get("denunciado", "denunciados", "proveedor", "empresa", "administrados"),
		RUC:               get("ruc", "ruc denunciados", "documento denunciados", "documento de denunciados", "ruc_denunciados", "ruc administrados"),
		Materia:           get("materia", "materias", "submateria", "motivo", "vc rubro", "sector", "sub sector", "subsector", "sub_sector", "tipo de obra", "ciiu"),
		Resumen:           buildSummary(raw.Values),
		RawColumnCount:    len(raw.Values),
	}
}

func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ñ", "n",
		"_", " ",
		"-", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(s)), " ")
}

func buildSummary(values map[string]string) string {
	parts := make([]string, 0, len(values))
	for k, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " | ")
}
