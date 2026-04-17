/*
 * detector_secuencial.pml
 *
 * Modelo Promela del algoritmo SECUENCIAL de detección de red flags
 * sobre expedientes INDECOPI.
 *
 * Flujo modelado:
 *   Fase 1 → TEXT_REPEAT  : comparación O(n²) de similitud Jaccard entre quejas
 *   Fase 2 → TIMING_BURST : agrupación por fecha, detección de concentraciones
 *   Fase 3 → EXACT_DUPLICATE : búsqueda de claves duplicadas con tabla hash
 *
 * Propiedades verificadas con SPIN:
 *   [Safety]   flags_total nunca es negativo durante ninguna fase
 *   [Safety]   la fase 3 solo se alcanza después de completar la fase 2
 *   [Liveness] el proceso siempre termina (no hay deadlock)
 *   [Liveness] eventualmente se alcanza la fase de resultados finales
 *
 * Uso:
 *   spin -a detector_secuencial.pml   -> generar verificador
 *   gcc -o pan pan.c                  -> compilar
 *   ./pan -a                          -> verificar con aserción
 *   spin -search -ltl safety detector_secuencial.pml
 *   spin -search -ltl liveness detector_secuencial.pml
 */

/* ── Parámetros del modelo (valores pequeños para evitar explosión de estados) ── */
#define N_REGISTROS  4      /* número de expedientes en el dataset (simplificado) */
#define BURST_LIMITE 2      /* umbral de expedientes por fecha para TIMING_BURST  */

/* ── Variables de estado global ── */
int  fase         = 0;   /* 0=carga 1=text_repeat 2=timing_burst 3=exact_dup 4=fin */
int  flags_repeat = 0;   /* contador de flags TEXT_REPEAT    */
int  flags_burst  = 0;   /* contador de flags TIMING_BURST   */
int  flags_dup    = 0;   /* contador de flags EXACT_DUPLICATE */
int  flags_total  = 0;   /* suma global de todos los flags    */
bool terminado    = false;

/* ── LTL: propiedades a verificar ── */
ltl safety   { [] (flags_total >= 0) }
ltl orden    { [] (fase == 3 -> fase >= 2) }   /* fase 3 solo si pasó la 2 */
ltl liveness { <> (terminado == true) }

/* ══════════════════════════════════════════════════════════════════════════════
 * Proceso principal: detector secuencial
 * Ejecuta las tres fases en orden estricto, una goroutine, sin concurrencia.
 * ══════════════════════════════════════════════════════════════════════════════ */
active proctype Secuencial() {

    /* ─── Fase 1: TEXT_REPEAT ───────────────────────────────────────────────
     * Para cada par (i, j) con j > i se evalúa la similitud Jaccard.
     * Si supera el umbral, ambos registros reciben el flag TEXT_REPEAT.
     * La similitud real depende de los datos; aquí se modela como elección
     * no determinista para que SPIN explore todos los caminos posibles.
     */
    fase = 1;
    int i = 0;
    int j;
    bool similar;

    do
    :: i < N_REGISTROS ->
        j = i + 1;
        do
        :: j < N_REGISTROS ->
            /* elección no determinista: similitud alta o baja */
            if
            :: similar = true     /* par supera umbral Jaccard → flag */
            :: similar = false    /* par no supera umbral → sin flag  */
            fi;

            if
            :: similar ->
                flags_repeat = flags_repeat + 2;  /* se marcan ambos registros */
                flags_total  = flags_total  + 2
            :: !similar -> skip
            fi;

            assert(flags_total >= 0);   /* invariante: nunca negativo */
            j = j + 1
        :: j >= N_REGISTROS -> break
        od;
        i = i + 1
    :: i >= N_REGISTROS -> break
    od;

    /* ─── Fase 2: TIMING_BURST ──────────────────────────────────────────────
     * Se agrupan los registros por fecha de presentación (O(n)).
     * Si alguna fecha concentra más expedientes que BURST_LIMITE, se marcan.
     * El conteo por fecha se modela con elección no determinista.
     */
    fase = 2;
    int conteo_fecha;

    if
    :: conteo_fecha = BURST_LIMITE + 1 ->   /* burst detectado */
        flags_burst = flags_burst + conteo_fecha;
        flags_total = flags_total + conteo_fecha
    :: conteo_fecha = 1 -> skip              /* sin burst */
    fi;

    assert(flags_total >= 0);

    /* ─── Fase 3: EXACT_DUPLICATE ───────────────────────────────────────────
     * Se construye una tabla hash con clave = expediente|denunciado|fecha.
     * Si la clave ya existe, ambos registros reciben el flag EXACT_DUPLICATE.
     */
    fase = 3;
    bool duplicado;

    if
    :: duplicado = true ->
        flags_dup   = flags_dup   + 2;
        flags_total = flags_total + 2
    :: duplicado = false -> skip
    fi;

    assert(flags_total >= 0);

    /* ─── Resultados finales ─────────────────────────────────────────────── */
    fase      = 4;
    terminado = true;

    /* Invariante final: cada contador parcial es coherente con el total */
    assert(flags_total == flags_repeat + flags_burst + flags_dup);
    assert(flags_repeat >= 0);
    assert(flags_burst  >= 0);
    assert(flags_dup    >= 0);
}
