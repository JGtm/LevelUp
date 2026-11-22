//go:build integration

// sync_v2_convergence_difficile_porte_integration_test.go : porte du worker de persistance (retient un batch, le laisse passer, ou coupe).
//
// Voir le commentaire de sync_v2_convergence_difficile_env_integration_test.go pour le banc dans son ensemble.
package main

import (
	"context"
	"errors"
	"levelup/go-api/internal/persist"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------------------------
// La porte du worker de persistance.
// ---------------------------------------------------------------------------------------------

var errDifficileArretBrutal = errors.New("arrêt brutal simulé")

// difficileGate enveloppe le persister du worker : il laisse passer n batches, puis retient le suivant
// jusqu'à release ou kill. kill(true) laisse le batch retenu s'écrire en base SANS que le worker
// l'acquitte (coupure entre l'écriture et l'ACK du journal) ; tous les batches suivants échouent sans
// rien écrire (le processus n'existe plus).
type difficileGate struct {
	inner persist.BatchPersister

	mu              sync.Mutex
	limit           int // batches laissés passer avant de retenir ; < 0 : tous passent
	passed          int
	killed          bool
	blocked         chan struct{} // fermé quand un batch est retenu
	release         chan struct{}
	kill_           chan struct{}
	inFlightWritten bool
}

func newDifficileGate(inner persist.BatchPersister) *difficileGate {
	return &difficileGate{inner: inner, limit: -1, blocked: make(chan struct{}), release: make(chan struct{}), kill_: make(chan struct{})}
}

// arm : les n prochains batches passent, le suivant est retenu.
func (g *difficileGate) arm(n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.limit, g.passed = n, 0
}

func (g *difficileGate) waitBlocked(t *testing.T, d time.Duration) {
	t.Helper()
	select {
	case <-g.blocked:
	case <-time.After(d):
		t.Fatalf("aucun batch retenu par la porte après %v", d)
	}
}

func (g *difficileGate) letGo() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.limit = -1
	select {
	case <-g.release:
	default:
		close(g.release)
	}
}

func (g *difficileGate) kill(writeInFlight bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.killed = true
	g.inFlightWritten = writeInFlight
	select {
	case <-g.kill_:
	default:
		close(g.kill_)
	}
}

func (g *difficileGate) Persist(ctx context.Context, b *persist.MatchBatch) error {
	g.mu.Lock()
	if g.killed {
		g.mu.Unlock()
		return errDifficileArretBrutal
	}
	hold := g.limit >= 0 && g.passed >= g.limit
	g.passed++
	if hold {
		select {
		case <-g.blocked:
		default:
			close(g.blocked)
		}
	}
	g.mu.Unlock()
	if hold {
		select {
		case <-g.release:
		case <-g.kill_:
			g.mu.Lock()
			write := g.inFlightWritten
			g.mu.Unlock()
			if write {
				_ = g.inner.Persist(ctx, b)
			}
			return errDifficileArretBrutal
		}
	}
	return g.inner.Persist(ctx, b)
}
