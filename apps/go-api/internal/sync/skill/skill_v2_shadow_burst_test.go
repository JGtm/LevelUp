//go:build cgo

package skill

// skill_v2_shadow_burst_test.go — verrouille le fix hotfix/lusr-shadow-ro
// (régression prod 2026-07-03) : le shadow LUSR v2 persiste via des bursts Write
// (RW), jamais sur le handle de LECTURE (RO en mode burst). Deux propriétés :
//   1. Read-only guard : Read sert un attach READ_ONLY (la sélection y passe),
//      Write sert un attach RW (le persist y passe) → le run traite ET persiste,
//      le watermark avance. Sur l'ancien code (persist via le handle unique de
//      lecture) l'INSERT échouait « attached in read-only mode » → processed=0.
//   2. Anti-deadlock : aucun burst Write n'est demandé pendant qu'un Read du même
//      accès est en vol (le garde de SharedAccess.Write transformerait le bug en
//      erreur) — vérifié y compris sur > 1 chunk.

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// seedShadow2v2 insère un match 2v2 LUSR-éligible (owner+teammate1 vs opp1+opp2,
// pair_name "Slayer" → arena_slayer, owner gagnant) dans le handle fourni.
func seedShadow2v2(t *testing.T, db *sql.DB, matchID string, start time.Time) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`, matchID, start, start); err != nil {
		t.Fatalf("seed match_registry(%s): %v", matchID, err)
	}
	for _, q := range []struct {
		x      string
		tm, oc int
		k, d   int
	}{{"owner", 0, 2, 18, 6}, {"teammate1", 0, 2, 12, 9}, {"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14}} {
		if _, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			matchID, q.x, q.tm, q.oc, q.k, q.d); err != nil {
			t.Fatalf("seed participant(%s,%s): %v", matchID, q.x, err)
		}
	}
}

// openShadowTestFileDB crée une DB DuckDB FICHIER seedée avec le schéma shadow,
// et retourne son chemin (slash-normalisé pour ATTACH). La connexion RW de seed
// est FERMÉE avant retour → les attach RO/RW ultérieurs ne se disputent pas le lock.
func openShadowTestFileDB(t *testing.T) string {
	t.Helper()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "shared.duckdb"))
	rw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open file db: %v", err)
	}
	if _, err := rw.Exec(shadowSchemaDDL); err != nil {
		_ = rw.Close()
		t.Fatalf("file DDL: %v", err)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("close seed conn: %v", err)
	}
	return path
}

// roRwSplitAccess : SharedAccessor où Read ouvre un attach READ_ONLY du fichier
// (SELECT OK, INSERT interdit) et Write un attach RW. Reproduit la topologie prod
// B-swap : Read et Write ne sont JAMAIS ouverts simultanément (le shadow release
// le Read avant tout burst), donc pas de conflit de lock fichier.
type roRwSplitAccess struct {
	path       string
	readCalls  int
	writeCalls int
}

func (a *roRwSplitAccess) attach(ctx context.Context, readOnly bool) (*sql.DB, error) {
	c, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}
	mode := ""
	if readOnly {
		mode = " (READ_ONLY)"
	}
	if _, err := c.ExecContext(ctx, "ATTACH '"+a.path+"' AS s"+mode); err != nil {
		_ = c.Close()
		return nil, err
	}
	if _, err := c.ExecContext(ctx, "USE s"); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (a *roRwSplitAccess) Read(ctx context.Context) (*sql.DB, func(), error) {
	a.readCalls++
	c, err := a.attach(ctx, true)
	if err != nil {
		return nil, nil, err
	}
	return c, func() { _ = c.Close() }, nil
}

func (a *roRwSplitAccess) Write(ctx context.Context, _ string) (*sql.DB, func(), error) {
	a.writeCalls++
	c, err := a.attach(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	return c, func() { _ = c.Close() }, nil
}

// TestLUSRV2Shadow_PersistsViaWriteBurst_WhenReadHandleIsReadOnly : la sélection
// passe par un handle read-only, le persist par un burst RW → traite ET persiste,
// le watermark avance. Sur l'ancien code (handle unique de lecture) : ROUGE.
func TestLUSRV2Shadow_PersistsViaWriteBurst_WhenReadHandleIsReadOnly(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "") // shadow-only (v1 canonical par défaut)

	path := openShadowTestFileDB(t)
	// seed via une connexion RW éphémère (refermée avant les attach).
	seed, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open seed: %v", err)
	}
	seedShadow2v2(t, seed, "m1", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	acc := &roRwSplitAccess{path: path}
	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, acc, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1 (persist doit passer par le burst Write RW)", processed)
	}
	if acc.writeCalls < 1 {
		t.Errorf("writeCalls = %d, want >= 1 (le persist doit acquérir un burst Write)", acc.writeCalls)
	}

	// Vérifie l'avance du watermark : l'état owner existe et porte last_match_at.
	verify, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open verify: %v", err)
	}
	defer verify.Close() //nolint:errcheck
	var exp int
	var lastAt sql.NullTime
	err = verify.QueryRow(`SELECT experience, last_match_at FROM player_skill_state_v2_latest
		WHERE xuid = 'owner' AND playlist_group = 'arena_slayer'`).Scan(&exp, &lastAt)
	if err != nil {
		t.Fatalf("state owner introuvable après run (persist non effectué ?): %v", err)
	}
	if exp != 1 {
		t.Errorf("experience = %d, want 1", exp)
	}
	if !lastAt.Valid {
		t.Error("last_match_at NULL — le watermark n'a pas avancé")
	}
}

// orderTrackingAccess : SharedAccessor sur un handle unique in-memory qui ÉCHOUE
// le test si un Write est demandé pendant qu'un Read est en vol (garde
// anti-deadlock). Compte aussi les bursts Write.
type orderTrackingAccess struct {
	db              *sql.DB
	mu              sync.Mutex
	readOutstanding int
	writeCalls      int
	violation       string
}

func (a *orderTrackingAccess) Read(context.Context) (*sql.DB, func(), error) {
	a.mu.Lock()
	a.readOutstanding++
	a.mu.Unlock()
	return a.db, func() {
		a.mu.Lock()
		a.readOutstanding--
		a.mu.Unlock()
	}, nil
}

func (a *orderTrackingAccess) Write(_ context.Context, step string) (*sql.DB, func(), error) {
	a.mu.Lock()
	a.writeCalls++
	if a.readOutstanding > 0 && a.violation == "" {
		a.violation = fmt.Sprintf("Write(%s) demandé avec %d Read en vol", step, a.readOutstanding)
	}
	a.mu.Unlock()
	return a.db, func() {}, nil
}

// TestLUSRV2Shadow_ReleasesReadBeforeWriteBurst_MultiChunk : sur 4 matchs (> 1
// chunk de 3), le shadow traite tout, acquiert plusieurs bursts Write, et ne
// demande JAMAIS un Write pendant qu'un Read est en vol.
func TestLUSRV2Shadow_ReleasesReadBeforeWriteBurst_MultiChunk(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "")

	db := openShadowTestDB(t)
	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		seedShadow2v2(t, db, fmt.Sprintf("mc%d", i), base.Add(time.Duration(i)*time.Hour))
	}

	acc := &orderTrackingAccess{db: db}
	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, acc, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 4 {
		t.Errorf("processed = %d, want 4 (tous les matchs des 2 chunks)", processed)
	}
	if acc.violation != "" {
		t.Errorf("garde anti-deadlock violée : %s", acc.violation)
	}
	if acc.writeCalls < 2 {
		t.Errorf("writeCalls = %d, want >= 2 (4 matchs → chunks de 3 → 2 bursts)", acc.writeCalls)
	}
}
