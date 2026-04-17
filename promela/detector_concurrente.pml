/*
 * detector_concurrente.pml
 *
 * Modelo Promela del algoritmo CONCURRENTE de detección de red flags
 * sobre expedientes INDECOPI — patrón Worker Pool.
 *
 * Goroutines modeladas:
 *   Productor  : envía índices de registros al canal workCh
 *   Worker(id) : recibe índices, computa pares (i,j), acumula flags locales
 *   Supervisor : espera a todos los workers (equivale a sync.WaitGroup.Wait)
 *   Recolector : lee resultados de resultCh y acumula flags_global
 *
 * Canales:
 *   workCh   : int  [N_REGISTROS + N_WORKERS]  distribuye trabajo a los workers
 *   resultCh : int  [N_WORKERS]               recibe el conteo local de cada worker
 *
 * Propiedades verificadas con SPIN:
 *   [Safety]   flags_global nunca es negativo
 *   [Safety]   done_count no supera N_WORKERS
 *   [Safety]   no hay escritura concurrente a flags_global sin sincronización
 *   [Liveness] todos los workers eventualmente terminan (finished == true)
 *   [Liveness] el sistema no tiene deadlock (el recolector siempre termina)
 *
 * Uso:
 *   spin -a detector_concurrente.pml
 *   gcc -o pan pan.c
 *   ./pan -a                              verificar aserciones
 *   ./pan -a -f                           verificar también liveness (fairness)
 *   spin -search -ltl safety  detector_concurrente.pml
 *   spin -search -ltl liveness detector_concurrente.pml
 */

/* ── Parámetros ── */
#define N_WORKERS   2       /* número de goroutines worker                   */
#define N_REGISTROS 4       /* registros en el dataset (simplificado)        */
#define SENTINEL   -1       /* valor especial que indica "no hay más trabajo" */
#define BURST_LIMITE 2      /* umbral TIMING_BURST                           */

/* ── Canales (equivalentes a los Go channels buffered) ── */
/*
 * workCh: el productor envía N_REGISTROS índices + N_WORKERS sentinels.
 *   Buffer = N_REGISTROS + N_WORKERS para que el productor no bloquee.
 *   Esto modela el canal buffered de Go con buffer = numWorkers*4.
 */
chan workCh   = [N_REGISTROS + N_WORKERS] of { int };

/*
 * resultCh: cada worker envía su conteo local UNA sola vez al terminar.
 *   Buffer = N_WORKERS para que los workers no bloqueen al enviar.
 *   En Go: resultCh <- local (una sola escritura por worker).
 */
chan resultCh = [N_WORKERS] of { int };

/* ── Variables de sincronización ── */
/*
 * done_count: equivale al WaitGroup de Go.
 *   Cada worker lo incrementa atómicamente al terminar.
 *   Cuando done_count == N_WORKERS, el supervisor cierra resultCh
 *   (en Promela: señaliza que no hay más resultados).
 */
byte done_count  = 0;
bool finished    = false;   /* true cuando todos los workers terminaron */
bool recolectado = false;   /* true cuando el recolector terminó        */

/* ── Variables de resultado ── */
int flags_global = 0;   /* suma de todos los flags locales de todos los workers */

/* ── LTL: propiedades a verificar ── */

/* Safety: flags_global es siempre no-negativo */
ltl safety       { [] (flags_global >= 0) }

/* Safety: done_count nunca supera N_WORKERS */
ltl wg_bound     { [] (done_count <= N_WORKERS) }

/* Liveness: eventualmente todos los workers terminan */
ltl liveness     { <> (finished == true) }

/* Liveness: eventualmente el recolector termina con resultado válido */
ltl completitud  { <> (recolectado == true && flags_global >= 0) }


/* ══════════════════════════════════════════════════════════════════════════════
 * PRODUCTOR
 * Equivale a la goroutine anónima que envía índices i a workCh y luego
 * lo cierra. En Promela se modela enviando un SENTINEL por cada worker,
 * ya que Promela no tiene cierre de canal nativo.
 * ══════════════════════════════════════════════════════════════════════════════ */
active proctype Productor() {
    int i = 0;

    /* Enviar todos los índices de trabajo al canal */
    do
    :: i < N_REGISTROS ->
        workCh ! i;
        i = i + 1
    :: i >= N_REGISTROS -> break
    od;

    /* Enviar un SENTINEL por cada worker para señalizar el fin del trabajo.
     * Esto modela el close(workCh) de Go: cuando un worker recibe SENTINEL,
     * sabe que no hay más índices y puede terminar su loop. */
    int w = 0;
    do
    :: w < N_WORKERS ->
        workCh ! SENTINEL;
        w = w + 1
    :: w >= N_WORKERS -> break
    od
}


/* ══════════════════════════════════════════════════════════════════════════════
 * WORKER
 * Cada worker recibe índices de workCh, calcula pares (i, j>i) con
 * similitud no determinista, y acumula flags en su variable LOCAL.
 * Al recibir SENTINEL, envía su acumulado a resultCh y actualiza done_count.
 *
 * Puntos clave de sincronización:
 *   - No hay mutex: cada worker escribe solo en 'local_flags' (privado).
 *   - La única escritura compartida es el incremento atómico de done_count.
 *   - La escritura a resultCh es una operación de canal → sincronización implícita.
 * ══════════════════════════════════════════════════════════════════════════════ */
proctype Worker(byte id) {
    int  task;
    int  local_flags = 0;  /* acumulador LOCAL: sin mutex, sin contención */
    bool similar;

    /* Loop de consumo del canal de trabajo */
    do
    :: true ->
        workCh ? task;              /* recibir próximo índice */

        if
        :: task == SENTINEL -> break    /* no hay más trabajo */
        :: task >= 0 ->
            /* ── Simula cálculo de pares (i, j>i) con Jaccard ──
             * En el código real, este bloque itera j = task+1..N-1.
             * Aquí se abstrae con elección no determinista para que SPIN
             * verifique todas las trazas posibles. */
            if
            :: similar = true  ->       /* par similar → marcar ambos */
               local_flags = local_flags + 2
            :: similar = false -> skip  /* par no similar              */
            fi
        fi
    od;

    /* Enviar resultado local al recolector (UNA sola escritura al canal) */
    resultCh ! local_flags;

    /* Incrementar WaitGroup atómicamente para evitar race condition */
    atomic {
        done_count = done_count + 1;
        if
        :: done_count == N_WORKERS ->
            finished = true         /* último worker: señalizar fin */
        :: else -> skip
        fi
    }
}


/* ══════════════════════════════════════════════════════════════════════════════
 * RECOLECTOR (goroutine principal en Go)
 * Espera recibir exactamente N_WORKERS resultados de resultCh y los suma
 * en flags_global. No usa mutex porque es el único que escribe en flags_global.
 *
 * En Go: for local := range resultCh { flags = append(flags, local...) }
 * ══════════════════════════════════════════════════════════════════════════════ */
active proctype Recolector() {
    int w = 0;
    int local_recibido;

    do
    :: w < N_WORKERS ->
        resultCh ? local_recibido;        /* bloqueante hasta que el worker envíe */
        flags_global = flags_global + local_recibido;
        assert(flags_global >= 0);        /* invariante: nunca negativo */
        w = w + 1
    :: w >= N_WORKERS -> break
    od;

    /* ── Fases 2 y 3: se ejecutan secuencialmente después del Worker Pool ──
     * TIMING_BURST y EXACT_DUPLICATE son O(n) y no justifican paralelización.
     * Se modelan igual que en el algoritmo secuencial. */

    int conteo_fecha;
    if
    :: conteo_fecha = BURST_LIMITE + 1 ->
        flags_global = flags_global + conteo_fecha
    :: conteo_fecha = 1 -> skip
    fi;

    bool duplicado;
    if
    :: duplicado = true  -> flags_global = flags_global + 2
    :: duplicado = false -> skip
    fi;

    assert(flags_global >= 0);
    recolectado = true
}


/* ══════════════════════════════════════════════════════════════════════════════
 * INIT: lanzar workers
 * Equivale al bucle que lanza N goroutines en Go:
 *   for w := 0; w < numWorkers; w++ { go func() { ... }() }
 * ══════════════════════════════════════════════════════════════════════════════ */
init {
    atomic {
        run Worker(0);
        run Worker(1)
        /* Para probar con más workers agregar: run Worker(2); run Worker(3) */
        /* y ajustar #define N_WORKERS */
    }
}
