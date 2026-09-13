//go:build integration

// Package sync — engine_lusr_title_ctx_integration_test.go : garde C.1 du lot
// finitions LUSR (2026-09-13).
//
// Contrat prouvé ici : SyncEngine.RecomputeLUSRCanonical écrit la chaîne LUSR du
// titre du MOTEUR (e.titleSlug, celui dont il ouvre les bases), et JAMAIS celle
// du titre porté par le ctx entrant. C'est le correctif de la classe de défaut qui
// a produit la corruption h5_arena du 2026-06-26 : handles DB d'un titre, chaîne
// LUSR d'un autre (.ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §3, §5.3 G4).
//
// Le test passe par le seam title-aware réel (SetLUSRChainClassifierForTitle /
// GetLUSRChainForTitle) avec deux slugs SYNTHÉTIQUES : aucun autre test du binaire
// ne les consomme, et le Cleanup les ramène au comportement « non enregistré »
// (délégation au classifier par défaut) — le seam n'expose pas de désinscription.
package sync

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const (
	lusrCtxTestEngineSlug = "synthetic_lusr_engine_title"
	lusrCtxTestCtxSlug    = "synthetic_lusr_ctx_title"
	lusrCtxTestEngineChan = "engine_chain"
	lusrCtxTestForeignCha = "foreign_ctx_chain"
)

func TestRecomputeLUSRCanonical_UsesEngineTitleNotCtxTitle(t *testing.T) {
	t.Setenv("LEVELUP_LUSR_V2_ENABLED", "1")
	t.Setenv("LEVELUP_LUSR_CANONICAL", "LUSR_V2")

	const gamertag = "LUSRTitleCtxPlayer"
	const xuid = "owner"

	// Classifiers title-aware : un par slug synthétique, sorties distinctes → la
	// chaîne écrite NOMME le titre dont le classifier a été consulté.
	SetLUSRChainClassifierForTitle(lusrCtxTestEngineSlug, func(string) string { return lusrCtxTestEngineChan })
	SetLUSRChainClassifierForTitle(lusrCtxTestCtxSlug, func(string) string { return lusrCtxTestForeignCha })
	t.Cleanup(func() {
		// Restauration : les deux slugs redeviennent équivalents à un slug non
		// enregistré (GetLUSRChain = classifier par défaut, câblé par TestMain).
		SetLUSRChainClassifierForTitle(lusrCtxTestEngineSlug, GetLUSRChain)
		SetLUSRChainClassifierForTitle(lusrCtxTestCtxSlug, GetLUSRChain)
	})

	// Registre runtime : les deux titres synthétiques déclarent CapLUSR (sinon le
	// gate capability du shadow v2 no-op silencieusement).
	oldReg := titlePkg.DefaultRegistry()
	reg := titlePkg.NewRegistry()
	for _, slug := range []string{lusrCtxTestEngineSlug, lusrCtxTestCtxSlug} {
		reg.Register(&titlePkg.TitleDescriptor{
			Slug: slug, Name: slug, Status: titlePkg.StatusActive,
			Capabilities: []titlePkg.Capability{titlePkg.CapLUSR},
		})
	}
	titlePkg.SetDefaultRegistry(reg)
	t.Cleanup(func() { titlePkg.SetDefaultRegistry(oldReg) })

	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(lusrCtxTestEngineSlug)
	playerPath := pr.PlayerDBPath(lusrCtxTestEngineSlug, gamertag)

	seedLUSRCtxSharedDB(t, sharedPath, xuid)
	seedLUSRCtxPlayerDB(t, playerPath)

	// Moteur du titre synthétique « engine » ; ctx porteur du titre « ctx ».
	engine := NewSyncEngineForTitle(repoRoot, lusrCtxTestEngineSlug, gamertag, xuid, nil, nil)
	ctx := ctxkeys.WithTitleSlug(context.Background(), lusrCtxTestCtxSlug)

	n, err := engine.RecomputeLUSRCanonical(ctx)
	if err != nil {
		t.Fatalf("RecomputeLUSRCanonical: %v", err)
	}
	if n != 1 {
		t.Fatalf("matchs traités = %d, want 1 (le match seedé doit passer les filtres)", n)
	}

	// Chaîne persistée côté shared (état TrueSkill) ET côté player (ligne canonique
	// match_skill_rank, la table corrompue en juin) : les deux doivent nommer le
	// titre du MOTEUR.
	if got := readSingleString(t, sharedPath,
		`SELECT playlist_group FROM player_skill_state_v2_latest WHERE xuid = ?`, xuid); got != lusrCtxTestEngineChan {
		t.Errorf("player_skill_state_v2_latest.playlist_group = %q, want %q (le ctx portait %q : fuite de titre)",
			got, lusrCtxTestEngineChan, lusrCtxTestCtxSlug)
	}
	if got := readSingleString(t, playerPath,
		`SELECT playlist_group FROM match_skill_rank_latest WHERE rating_type = 'LUSR'`); got != lusrCtxTestEngineChan {
		t.Errorf("match_skill_rank_latest.playlist_group = %q, want %q (chaîne étrangère écrite = corruption 2026-06-26)",
			got, lusrCtxTestEngineChan)
	}
}

// seedLUSRCtxSharedDB crée la shared DB du titre synthétique (schéma complet via
// les migrations réelles) et y insère un match social 2v2 éligible au LUSR v2.
func seedLUSRCtxSharedDB(t *testing.T, sharedPath, ownerXUID string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir shared dir: %v", err)
	}
	db, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatalf("open shared %s: %v", sharedPath, err)
	}
	defer func() { _ = db.Close() }()
	// Schéma par les MIGRATIONS RÉELLES (title-owned, câblées par TestMain) — pas
	// une DDL recopiée : une copie dérive sans que rien ne le signale.
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(shared): %v", err)
	}
	start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES ('lusr_ctx_m1', ?, ?, 'pair:Slayer', FALSE, FALSE, 600)`, start, start); err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, p := range []struct {
		xuid                string
		team, outcome, k, d int
	}{
		{ownerXUID, 0, 2, 18, 6}, {"teammate", 0, 2, 12, 9},
		{"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14},
	} {
		if _, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES ('lusr_ctx_m1', ?, ?, ?, ?, ?)`,
			p.xuid, p.team, p.outcome, p.k, p.d); err != nil {
			t.Fatalf("insert match_participants %s: %v", p.xuid, err)
		}
	}
}

// seedLUSRCtxPlayerDB crée la player DB du titre synthétique avec le schéma réel
// (EnsurePlayerSchema + migrations title-owned), jamais une DDL recopiée.
func seedLUSRCtxPlayerDB(t *testing.T, playerPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	db, err := sql.Open("duckdb", playerPath)
	if err != nil {
		t.Fatalf("open player %s: %v", playerPath, err)
	}
	defer func() { _ = db.Close() }()
	if err := EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
}

// readSingleString rouvre une DB fichier et lit une colonne texte unique.
func readSingleString(t *testing.T, dbPath, query string, args ...any) string {
	t.Helper()
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", dbPath, err)
	}
	defer func() { _ = db.Close() }()
	var out string
	if err := db.QueryRow(query, args...).Scan(&out); err != nil {
		t.Fatalf("query %q sur %s: %v", query, dbPath, err)
	}
	return out
}
