package notifications

import (
	"sync"
	"time"
)

// IDGenerator produit des IDs monotones snowflake-like : (unix_ms << seqBits) | seq.
// L'ID tient dans int64 sans dépasser 2^53-1 (JS-safe) jusqu'en ~2106.
// Permet la pagination cursor (before_id) sur l'index created_at DESC.
type IDGenerator struct {
	mu     sync.Mutex
	lastMs int64
	seq    uint16
	clock  func() int64 // injecté pour tests, défaut = time.Now().UnixMilli()
}

const (
	seqBits = 12
	seqMask = (1 << seqBits) - 1 // 4095 IDs/ms max
)

// NewIDGenerator crée un générateur avec horloge réelle.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{clock: nowMilli}
}

func nowMilli() int64 { return time.Now().UnixMilli() }

// Next retourne le prochain ID. Thread-safe.
// Si plus de 4096 appels surviennent dans la même milliseconde, attend
// la suivante (busy-wait court — improbable en pratique).
func (g *IDGenerator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	nowMs := g.clock()
	if nowMs == g.lastMs {
		if g.seq >= seqMask {
			for nowMs == g.lastMs {
				time.Sleep(50 * time.Microsecond)
				nowMs = g.clock()
			}
			g.lastMs = nowMs
			g.seq = 0
		} else {
			g.seq++
		}
	} else {
		g.lastMs = nowMs
		g.seq = 0
	}
	return (nowMs << seqBits) | int64(g.seq)
}
