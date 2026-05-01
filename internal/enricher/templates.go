package enricher

import (
	"hash/fnv"
	"math/rand"
	"strconv"
	"strings"

	"tp-programacion-concurente/internal/normalize"
)

var templates = []string{
	"Presenté {tipo} ante {area} porque me {verbo} {producto} con {defecto} en {lugar}.",
	"El {producto} que adquirí presentó {defecto}, causándome {consecuencia}.",
	"Acudí a {area} tras sufrir {defecto} con {producto} que no cumplía lo ofrecido.",
	"Interpuse {tipo} en {area} por {defecto} en el servicio recibido en {anio}.",
	"Me {verbo} {producto} en {lugar} y al reclamar ignoraron mi solicitud.",
	"En {anio} sufrí {defecto} por parte de un proveedor; presenté {tipo} ante {area}.",
	"El {producto} adquirido en {lugar} resultó con {defecto}, generándome {consecuencia}.",
	"Solicité solución por {defecto} en {lugar} pero no obtuve respuesta, por eso acudo a {area}.",
	"Me {verbo} {producto} defectuoso en {lugar}; al reclamar me indicaron que no procedía devolución.",
	"Presento {tipo} porque en {anio} recibí {producto} con {defecto} y no se respetó la garantía.",
	"El servicio contratado en {lugar} presentó {defecto}; esto me ocasionó {consecuencia}.",
	"Tras {defecto} en {lugar}, interpuse {tipo} ante {area} exigiendo compensación.",
	"Me {verbo} {producto} con características distintas a las ofrecidas, causando {consecuencia}.",
	"Denuncio {defecto} ocurrido en {anio} ante {area} porque afectó mis derechos como consumidor.",
	"En {lugar} me {verbo} por {producto} que nunca funcionó correctamente desde el primer día.",
	"Presento reclamo ante {area} por {defecto} sufrido al adquirir {producto} en {lugar}.",
	"El proveedor en {lugar} incumplió con {producto}; recurro a {area} en busca de solución.",
	"Solicité {producto} y recibí uno con {defecto}, ocasionándome {consecuencia} en {anio}.",
	"Acudo a {area} porque el {producto} comprado en {lugar} presentó {defecto} desde el inicio.",
	"En {anio} fui víctima de {defecto} al adquirir {producto}; presento {tipo} ante {area}.",
}

var verbos = []string{
	"vendieron", "entregaron", "facturaron", "cobraron", "prometieron",
	"ofrecieron", "suministraron", "despacharon", "proporcionaron", "transfirieron",
	"asignaron", "cargaron", "procesaron", "registraron", "liquidaron",
}

var defectos = []string{
	"fallas técnicas graves", "incumplimiento de garantía", "cobro indebido",
	"información falsa en la publicidad", "demora injustificada en la entrega",
	"negativa a efectuar el reembolso", "producto en mal estado",
	"servicio interrumpido sin previo aviso", "incumplimiento del contrato",
	"trato discriminatorio", "cargo no autorizado en mi cuenta",
	"cambio de condiciones sin notificación", "producto adulterado",
	"doble cobro por el mismo concepto", "falta de respuesta al reclamo previo",
}

var consecuencias = []string{
	"pérdida económica significativa", "perjuicio a mi salud",
	"daño moral y estrés", "imposibilidad de usar el bien adquirido",
	"gastos adicionales no previstos", "pérdida de tiempo laboral",
	"afectación a mi familia", "deterioro en mi calidad de vida",
	"imposibilidad de cumplir mis obligaciones", "quebranto patrimonial",
}

var lugares = []string{
	"tienda", "local comercial", "plataforma web", "punto de venta",
	"sucursal", "establecimiento", "centro comercial", "agencia",
	"oficina", "portal en línea",
}

// productsPorArea devuelve la lista de productos/servicios según el área del dataset.
func productsPorArea(datasetID string) []string {
	id := strings.ToLower(datasetID)
	switch {
	case strings.Contains(id, "dda"):
		return []string{
			"servicio de gestión de residuos", "emisión contaminante",
			"producto con impacto ambiental", "servicio de saneamiento",
			"insumo industrial contaminante", "tratamiento de efluentes",
			"manejo de residuos sólidos", "producto químico peligroso",
			"servicio de fumigación", "disposición de desechos",
		}
	case strings.Contains(id, "din"):
		return []string{
			"producto sin certificación", "bien sin etiquetado correcto",
			"producto importado sin norma técnica", "servicio no certificado",
			"bien con medidas incorrectas", "producto sin registro sanitario",
			"mercadería sin homologación", "artículo sin norma INDECOPI",
			"producto con etiqueta engañosa", "bien fuera de estándar",
		}
	case strings.Contains(id, "dsd"):
		return []string{
			"marca registrada sin autorización", "signo distintivo copiado",
			"nombre comercial usurpado", "logotipo reproducido ilegalmente",
			"denominación de origen falsa", "patente infringida",
			"diseño industrial copiado", "etiqueta con marca ajena",
			"slogan registrado usado sin licencia", "trade dress imitado",
		}
	case strings.Contains(id, "clc"):
		return []string{
			"servicio en posición dominante", "acuerdo anticompetitivo",
			"práctica restrictiva del mercado", "abuso de posición de dominio",
			"fusión sin autorización", "acuerdo de precios entre competidores",
			"bloqueo de acceso al mercado", "discriminación de precios",
			"negativa de venta injustificada", "reparto de mercado ilegal",
		}
	case strings.Contains(id, "spc"):
		return []string{
			"producto de consumo masivo", "servicio financiero",
			"bien inmueble adquirido", "servicio educativo",
			"producto farmacéutico", "servicio de transporte",
			"bien de uso doméstico", "servicio de telecomunicaciones",
			"producto alimenticio", "servicio de salud privado",
		}
	case strings.Contains(id, "cc"):
		return []string{
			"servicio contratado", "producto adquirido",
			"bien de consumo", "servicio de mantenimiento",
			"electrodoméstico", "servicio de reparación",
			"artículo del hogar", "servicio técnico especializado",
			"producto electrónico", "servicio de instalación",
		}
	default: // ops y resto
		return []string{
			"electrodoméstico", "ropa y calzado", "alimento procesado",
			"servicio de internet", "producto de higiene", "servicio bancario",
			"equipo electrónico", "servicio de telefonía", "artículo deportivo",
			"servicio de delivery", "producto cosmético", "servicio educativo",
			"mueble del hogar", "servicio de streaming", "producto farmacéutico",
		}
	}
}

// areaDesc mapea el dataset_id a una descripción legible del área INDECOPI.
func areaDesc(datasetID string) string {
	id := strings.ToLower(datasetID)
	switch {
	case strings.Contains(id, "ops"):
		return "la oficina de protección al consumidor"
	case strings.Contains(id, "cc"):
		return "la comisión de conciliación de INDECOPI"
	case strings.Contains(id, "spc"):
		return "la sala de protección al consumidor"
	case strings.Contains(id, "dda"):
		return "la dirección de defensa ambiental"
	case strings.Contains(id, "din"):
		return "la dirección de normalización"
	case strings.Contains(id, "dsd"):
		return "la dirección de signos distintivos"
	case strings.Contains(id, "clc"):
		return "la comisión de libre competencia"
	default:
		return "INDECOPI"
	}
}

// rowHash genera un seed determinista a partir de los identificadores únicos de la fila.
func rowHash(r normalize.ConsolidatedRecord) int64 {
	h := fnv.New64a()
	h.Write([]byte(r.DatasetID + "|" + r.SourceSheet + "|" + strconv.Itoa(r.SourceRow)))
	return int64(h.Sum64())
}

// FillFromTemplates genera una queja mediante plantillas combinatorias sin usar LLM.
// El resultado es determinista: misma fila → mismo texto siempre.
func FillFromTemplates(r normalize.ConsolidatedRecord) string {
	rng := rand.New(rand.NewSource(rowHash(r)))

	tipo := strings.ToLower(r.TipoExpediente)
	if tipo == "" {
		tipo = "expediente"
	}

	anio := yearOf(r.FechaPresentacion)
	if anio == "" {
		anio = "ese período"
	}

	productos := productsPorArea(r.DatasetID)

	pick := func(list []string) string { return list[rng.Intn(len(list))] }

	replacer := strings.NewReplacer(
		"{area}", areaDesc(r.DatasetID),
		"{tipo}", tipo,
		"{anio}", anio,
		"{verbo}", pick(verbos),
		"{producto}", pick(productos),
		"{defecto}", pick(defectos),
		"{consecuencia}", pick(consecuencias),
		"{lugar}", pick(lugares),
	)

	tmpl := templates[rng.Intn(len(templates))]
	return replacer.Replace(tmpl)
}
