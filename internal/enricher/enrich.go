package enricher

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"tp-programacion-concurente/internal/normalize"
)

// reNoIdentificada elimina variantes de "empresa no identificada" que el LLM a veces escribe.
var reNoIdentificada = regexp.MustCompile(`(?i)\b(empresa|proveedor|entidad)\s+no\s+identificad[ao]\b`)

// ParseResumen convierte el campo resumen ("K=V | K2=V2") en un mapa con claves en uppercase.
func ParseResumen(resumen string) map[string]string {
	meta := make(map[string]string)
	for _, part := range strings.Split(resumen, " | ") {
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(part[:idx]))
		val := strings.TrimSpace(part[idx+1:])
		if key != "" && val != "" {
			meta[key] = val
		}
	}
	return meta
}

// buildContext extrae los campos comunes usados por ambos prompts (sin año).
func buildContext(r normalize.ConsolidatedRecord, meta map[string]string) (empresa, sector, tipo, actividad, anio string) {
	empresa = firstNonEmpty(r.Denunciado, meta["DENUNCIADOS"])
	if empresa == "" {
		empresa = "empresa no identificada"
	}
	sector = firstNonEmpty(meta["SUB_SECTOR"], meta["SECTOR"], r.Materia)
	if sector == "" {
		sector = "sector no especificado"
	}
	tipo = r.TipoExpediente
	if tipo == "" {
		tipo = "EXPEDIENTE"
	}
	actividad = firstNonEmpty(meta["DESCR_CIIU"], meta["CIIU"])
	anio = firstNonEmpty(meta["ANIO"], yearOf(r.FechaPresentacion))
	return
}

// buildPromptGeneralista genera un prompt directo y estructurado (más formal/robótico).
// No incluye instrucción anti-markdown a propósito → el modelo puede cometer esos errores.
// El año se incluye solo con 25% de probabilidad.
func buildPromptGeneralista(r normalize.ConsolidatedRecord, meta map[string]string) string {
	empresa, sector, tipo, actividad, anio := buildContext(r, meta)
	var sb strings.Builder
	sb.WriteString("Genera una queja/reclamo creativo breve (máximo 30 palabras) como si fuera escrita por un ciudadano peruano en un expediente ante INDECOPI basandote en los datos de las empresas que se te va a proporcionar. Responde solo con el texto, sin saludos ni explicaciones.\n\n")
	sb.WriteString(fmt.Sprintf("Empresa/Entidad: %s\n", empresa))
	sb.WriteString(fmt.Sprintf("Tipo de expediente: %s\n", tipo))
	sb.WriteString(fmt.Sprintf("Sector: %s\n", sector))
	if actividad != "" {
		sb.WriteString(fmt.Sprintf("Actividad: %s\n", actividad))
	}
	// 25% de probabilidad de incluir el año
	if anio != "" && rand.Intn(4) == 0 {
		sb.WriteString(fmt.Sprintf("Año: %s\n", anio))
	}
	return sb.String()
}

// buildPromptPersonalista genera un prompt para quejas personales y narrativas.
// Incluye instrucción explícita de no usar markdown. No incluye año.
func buildPromptPersonalista(r normalize.ConsolidatedRecord, meta map[string]string) string {
	empresa, sector, tipo, actividad, anio := buildContext(r, meta)
	var sb strings.Builder
	sb.WriteString("Genera una queja/reclamo breve (máximo 30 palabras) como si fuera escrita por un ciudadano peruano ante INDECOPI. Debe ser personal, narrativa e inventar un caso particular concreto (un hecho específico vivido por el ciudadano). Menciona la empresa y el sector. Responde solo con el texto, sin saludos ni explicaciones. No uses ningún carácter especial de markdown como asteriscos, guiones, almohadillas ni negritas; solo texto plano.\n\n")
	sb.WriteString(fmt.Sprintf("Empresa/Entidad: %s\n", empresa))
	sb.WriteString(fmt.Sprintf("Tipo de expediente: %s\n", tipo))
	sb.WriteString(fmt.Sprintf("Sector: %s\n", sector))
	if actividad != "" {
		sb.WriteString(fmt.Sprintf("Actividad: %s\n", actividad))
	}
	if anio != "" {
		sb.WriteString(fmt.Sprintf("Año: %s\n", anio))
	}
	return sb.String()
}

// BuildPrompt elige aleatoriamente entre el prompt generalista y el personalista.
func BuildPrompt(r normalize.ConsolidatedRecord, meta map[string]string) string {
	if rand.Intn(2) == 0 {
		return buildPromptGeneralista(r, meta)
	}
	return buildPromptPersonalista(r, meta)
}

// Enrich genera el campo queja para un registro. Retorna ("", nil) si no hay datos suficientes.
func Enrich(ctx context.Context, c *Client, r normalize.ConsolidatedRecord) (string, error) {
	meta := ParseResumen(r.Resumen)

	empresa := firstNonEmpty(r.Denunciado, meta["DENUNCIADOS"])
	sector := firstNonEmpty(meta["SUB_SECTOR"], meta["SECTOR"], r.Materia)
	if empresa == "" && sector == "" {
		return FillFromTemplates(r), nil
	}

	prompt := BuildPrompt(r, meta)
	result, err := c.Complete(ctx, prompt)
	if err != nil {
		return "", err
	}
	cleaned := reNoIdentificada.ReplaceAllString(result, "")
	return strings.TrimSpace(cleaned), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func yearOf(fecha string) string {
	if len(fecha) >= 4 {
		return fecha[:4]
	}
	return ""
}
