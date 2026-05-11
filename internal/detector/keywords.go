package detector

import (
	"fmt"
	"strings"
)

// spamKeywords son palabras o frases que no deberían aparecer en una denuncia
// ciudadana legítima ante INDECOPI. Se dividen en tres categorías:
//
//  1. Spam / phishing: lenguaje propio de correo basura o estafas digitales
//  2. Contenido sintético: frases que delatan texto auto-generado por plantillas
//  3. Formato markdown: caracteres de formateo que un ciudadano no escribiría
var spamKeywords = []string{
	// ── Spam / phishing / estafa ──────────────────────────────────────────────
	"urgente", "urgentemente", "viral", "click aquí", "haz click",
	"enlace", "link", "bitcoin", "criptomoneda", "inversión garantizada",
	"transferencia inmediata", "premio", "ganaste", "ganador",
	"oferta limitada", "gratis", "descuento especial", "actúa ya",
	"dinero fácil", "multiplica tu dinero",

	// ── Contenido sintético / auto-generado ───────────────────────────────────
	"sector no especificado", "ese período", "empresa no identificada",
	"la oficina de protección al consumidor",
	"la comisión de conciliación de indecopi",
	"la sala de protección al consumidor",
	"la dirección de defensa ambiental",
	"la dirección de normalización",
	"la dirección de signos distintivos",
	"la comisión de libre competencia",
	"local comercial", "punto de venta", // demasiado genérico para plantilla

	// ── Markdown / formato no humano ──────────────────────────────────────────
	"**", "##", "* ", "- [", "```",
}

// DetectKeywordSpam escanea cada registro en busca de palabras clave sospechosas.
// Es O(n·k) donde k es el número de keywords.
// Retorna un flag KEYWORD_SPAM por cada registro que contenga al menos
// opts.KeywordThreshold coincidencias.
func DetectKeywordSpam(records []DetectorRecord, opts Options) []RedFlag {
	var flags []RedFlag
	threshold := opts.KeywordThreshold
	if threshold <= 0 {
		threshold = 1
	}

	for i, r := range records {
		queja := strings.ToLower(r.Queja)
		var matched []string
		for _, kw := range spamKeywords {
			if strings.Contains(queja, kw) {
				matched = append(matched, kw)
			}
		}
		if len(matched) >= threshold {
			flags = append(flags, RedFlag{
				RecordIndex:  i,
				Expediente:   r.Expediente,
				DatasetID:    r.DatasetID,
				FlagType:     FlagKeywordSpam,
				Score:        float64(len(matched)),
				Details:      fmt.Sprintf("keywords detectadas: %s", strings.Join(matched, ", ")),
				QuejaPreview: quejaPreview(r.Queja),
			})
		}
	}
	return flags
}
