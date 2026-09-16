package main

// pool_engine_test.go — la CLI mono-joueur n'exige plus le token du joueur visé.
//
// POURQUOI (2026-09-16). `sync-delta/sync-full --gamertag X` passait par
// `haloTokensForPlayer(X)` et refusait de démarrer si X n'avait pas SON refresh token ;
// `--all` sautait le même joueur avec `SKIP reason=not_in_pool`. Un profil suivi sans token
// propre (Nuzzles) n'était donc jamais synchronisé, alors que l'historique, les stats, les
// films et les CSR sont des endpoints PUBLICS que n'importe quel token du parc sert.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
	go_sync "levelup/go-api/internal/sync"
)

// poolUnSlot : pool de fixture à UN slot (JGtm). Round-robin sert ce token à tout le monde ;
// l'acquisition épinglée n'aboutit que pour JGtm — exactement le parc du 2026-09-16.
type poolUnSlot struct {
	proprietaire string
	acquis       []string // politiques observées, pour l'assertion
}

func (p *poolUnSlot) Acquire(_ context.Context, politique pool.AcquirePolicy, epingle string) (*pool.Lease, error) {
	tokens := &domain.HaloTokens{SpartanToken: "spartan-test", ClearanceToken: "clearance-test"}
	switch politique {
	case pool.PolicyAnyPublic:
		p.acquis = append(p.acquis, "public")
		return &pool.Lease{Tokens: tokens, Gamertag: p.proprietaire, Release: func() {}}, nil
	default:
		p.acquis = append(p.acquis, "pinned:"+epingle)
		if epingle != p.proprietaire {
			return nil, errNonEpingle
		}
		return &pool.Lease{Tokens: tokens, Gamertag: p.proprietaire, Release: func() {}}, nil
	}
}

func (p *poolUnSlot) Size() int                           { return 1 }
func (p *poolUnSlot) HasPlayer(gamertag string) bool      { return gamertag == p.proprietaire }
func (p *poolUnSlot) MarkUnhealthy(string, error)         {}
func (p *poolUnSlot) OnHTTPError(int, time.Duration)      {}
func (p *poolUnSlot) On429ForToken(string, time.Duration) {}
func (p *poolUnSlot) Close()                              {}
func (p *poolUnSlot) AddOrUpdateSource(context.Context, pool.CredentialSource) error {
	return nil
}

var errNonEpingle = errString("aucun token pour ce gamertag")

type errString string

func (e errString) Error() string { return string(e) }

// TestRangDeCarriereSeDegradeSeul — le SEUL endpoint privacy-gated rend ErrNoPinnedToken pour
// un joueur hors pool : dégradation par endpoint, pas échec du sync.
func TestRangDeCarriereSeDegradeSeul(t *testing.T) {
	p := &poolUnSlot{proprietaire: "JGtm"}
	client := go_sync.NewPooledHaloClient(p, "Nuzzles", "2533274800000000", 0)

	rang, err := client.GetCareerRank(context.Background(), "2533274800000000")
	if rang != nil {
		t.Errorf("rang de carrière non nil pour un joueur sans token propre : %v", rang)
	}
	if err == nil || err.Error() != go_sync.ErrNoPinnedToken.Error() {
		t.Errorf("GetCareerRank = %v, attendu ErrNoPinnedToken", err)
	}
}
