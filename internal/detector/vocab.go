package detector

import "sync"

// Vocab es un vocabulario thread-safe que asigna un ID numérico único a cada token.
// Equivale al mapa interno de tokens de un LLM: "perro" → 1042, "gato" → 891.
//
// Sincronización: RWMutex con double-check locking.
//   - La mayoría de llamadas son lecturas (token ya existe) → RLock sin contención.
//   - Solo cuando aparece un token nuevo se adquiere el Lock de escritura.
//   - Se verifica de nuevo dentro del Lock por si otro goroutine lo insertó mientras
//     esperábamos (evita IDs duplicados sin serializar todas las lecturas).
type Vocab struct {
	mu     sync.RWMutex
	tokens map[string]uint32
	nextID uint32
	idToToken []string
}

func NewVocab() *Vocab {
	return &Vocab{tokens: make(map[string]uint32)}
}

// GetOrAdd devuelve el ID del token si ya existe, o asigna uno nuevo.
func (v *Vocab) GetOrAdd(token string) uint32 {
	// Intento rápido con solo lectura (sin bloquear a otros goroutines)
	v.mu.RLock()
	if id, ok := v.tokens[token]; ok {
		v.mu.RUnlock()
		return id
	}
	v.mu.RUnlock()

	// Token nuevo: adquirir escritura con double-check
	v.mu.Lock()
	defer v.mu.Unlock()
	if id, ok := v.tokens[token]; ok {
		return id // otro goroutine lo insertó mientras esperábamos
	}
	id := v.nextID
	v.tokens[token] = id
	// Mantener reverse index por id. Como id es secuencial, basta append.
	v.idToToken = append(v.idToToken, token)
	v.nextID++
	return id
}

func (v *Vocab) Size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.tokens)
}

// Token devuelve el string asociado a un id (si existe).
func (v *Vocab) Token(id uint32) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if int(id) < 0 || int(id) >= len(v.idToToken) {
		return "", false
	}
	return v.idToToken[id], true
}
