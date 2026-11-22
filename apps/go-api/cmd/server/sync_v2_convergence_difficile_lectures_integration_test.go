//go:build integration

// sync_v2_convergence_difficile_lectures_integration_test.go : cycles joués par l'ordonnanceur et lectures de la vérité terrain (bases DuckDB, journal d'écriture).
//
// Voir le commentaire de sync_v2_convergence_difficile_env_integration_test.go pour le banc dans son ensemble.
package main

import (
	"context"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/historystate"
	syncv2 "levelup/go-api/internal/sync/v2"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------------------------
// Cycles et lectures de la vérité terrain.
// ---------------------------------------------------------------------------------------------

// difficileCycle est ce qu'un cycle de l'ordonnanceur a produit.
type difficileCycle struct {
	ran    bool // l'orchestrateur a été appelé
	err    error
	result syncv2.CycleResult
	reads  map[string][]int // xuid -> décalages des pages d'historique lues pendant le cycle
	stats  map[string]int   // match -> demandes de stats pendant le cycle
}

func (c difficileCycle) player(j difficileJoueur) syncv2.PlayerOutcome {
	for _, o := range c.result.PerPlayer {
		if o.XUID == j.xuid {
			return o
		}
	}
	return syncv2.PlayerOutcome{}
}

func (c difficileCycle) statsTotal() int {
	n := 0
	for _, v := range c.stats {
		n += v
	}
	return n
}

// cycle lance un cycle PAR L'ORDONNANCEUR (RunOnceTrigger), compteurs remis à zéro avant.
func (b *difficileBoot) cycle() difficileCycle { return b.cycleCtx(b.ctx) }

func (b *difficileBoot) cycleCtx(ctx context.Context) difficileCycle {
	b.env.t.Helper()
	b.env.monde.resetCounters()
	b.rec.mu.Lock()
	before := len(b.rec.runs)
	b.rec.mu.Unlock()
	if res := b.sched.RunOnceTrigger(ctx, "manual"); res == nil {
		b.env.t.Fatal("RunOnceTrigger a rendu nil")
	}
	out := difficileCycle{reads: b.env.monde.reads(), stats: b.env.monde.statsSnapshot()}
	b.rec.mu.Lock()
	defer b.rec.mu.Unlock()
	if len(b.rec.runs) > before {
		out.ran = true
		out.result, out.err = b.rec.runs[len(b.rec.runs)-1], b.rec.errs[len(b.rec.errs)-1]
	}
	return out
}

func (b *difficileBoot) waitCadence(want time.Duration) {
	b.env.t.Helper()
	(&convergenceEnv{t: b.env.t, sched: b.sched}).waitCadence(want)
}

// difficileCounts : les comptes de la base partagée, pour la preuve d'absence de doublon.
type difficileCounts struct {
	registryRows, registryDistinct int
	partRows, partDistinct         int            // match_participants (match, xuid)
	perXUID                        map[string]int // xuid -> lignes de match_participants
}

func (b *difficileBoot) counts() difficileCounts {
	t := b.env.t
	t.Helper()
	db, release, err := b.provider.Get(b.ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	var c difficileCounts
	scan := func(q string, dst ...any) {
		if err := db.QueryRow(q).Scan(dst...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	scan(`SELECT COUNT(*), COUNT(DISTINCT match_id) FROM match_registry`, &c.registryRows, &c.registryDistinct)
	scan(`SELECT COUNT(*), COUNT(DISTINCT (match_id, xuid)) FROM match_participants`, &c.partRows, &c.partDistinct)
	c.perXUID = map[string]int{}
	rows, err := db.Query(`SELECT xuid, COUNT(*) FROM match_participants GROUP BY xuid`)
	if err != nil {
		t.Fatalf("participants par xuid: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var x string
		var n int
		if err := rows.Scan(&x, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		c.perXUID[x] = n
	}
	return c
}

// registryIDs rend les matchs du registre partagé.
func (b *difficileBoot) registryIDs() map[string]bool {
	t := b.env.t
	t.Helper()
	db, release, err := b.provider.Get(b.ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	rows, err := db.Query(`SELECT match_id FROM match_registry`)
	if err != nil {
		t.Fatalf("lecture match_registry: %v", err)
	}
	defer rows.Close()
	ids := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids[id] = true
	}
	return ids
}

// participantsOf rend, par match du registre, les xuids de match_participants.
func (b *difficileBoot) participantsOf() map[string]map[string]bool {
	t := b.env.t
	t.Helper()
	db, release, err := b.provider.Get(b.ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	rows, err := db.Query(`SELECT match_id, xuid FROM match_participants`)
	if err != nil {
		t.Fatalf("lecture match_participants: %v", err)
	}
	defer rows.Close()
	out := map[string]map[string]bool{}
	for rows.Next() {
		var id, x string
		if err := rows.Scan(&id, &x); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if out[id] == nil {
			out[id] = map[string]bool{}
		}
		out[id][x] = true
	}
	return out
}

// historyState lit la clé de fin (sa valeur brute), et le marque-page dans la base joueur.
func (b *difficileBoot) historyState(j difficileJoueur) (complete bool, cursor *historystate.Cursor) {
	t := b.env.t
	t.Helper()
	db, release, err := duckdbpkg.OpenReadForQuery(b.env.playerPath(j))
	if err != nil {
		t.Fatalf("lecture base joueur %s: %v", j.gamertag, err)
	}
	defer release()
	complete = historystate.Complete(b.ctx, db)
	if cur, ok := historystate.LoadCursor(b.ctx, db); ok {
		cursor = &cur
	}
	return complete, cursor
}

// walPending : fichiers du journal d'écriture de la file encore présents, et existence du dossier des
// journaux corrompus.
func (e *difficileEnv) walState() (pending int, corrupted bool) {
	dir := e.pr.WALDir()
	entries, _ := os.ReadDir(dir)
	for _, en := range entries {
		switch {
		case en.IsDir() && en.Name() == "corrupted":
			corrupted = true
		case !en.IsDir() && strings.HasSuffix(en.Name(), ".json"):
			pending++
		}
	}
	return pending, corrupted
}

// waitWALEmpty attend que le journal d'écriture soit vide (worker de recovery terminé).
func (e *difficileEnv) waitWALEmpty(d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		n, _ := e.walState()
		if n == 0 || time.Now().After(deadline) {
			return n
		}
		time.Sleep(50 * time.Millisecond)
	}
}
