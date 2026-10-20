//go:build integration

// temps_canonique_e3_test.go — étape E3 du plan .ai/PLAN_BACKLOG_2026-10-15.md : temps
// canonique (règle n°8) du filtre et du tri de CampaignSampleProvider et de
// ListRecentPvPMatchIDs, et erreurs de lecture visibles dans EngagementScoreRepo.
//
// Fixture « import OpenSpartan » : start_time NULL mais start_time_utc posé (ligne mA),
// et lignes dont le start_time naïf diverge de start_time_utc (mB, mD).

package duckdb

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
)

const e3XUID = "2533274800000009"

// newE3PlayerDB : matchs de e3XUID avec temps divergents.
//
//	mA : start_time NULL,         utc 09-01                     kills 1
//	mB : start_time 09-10 (naïf), utc 09-02                     kills 2
//	mC : start_time 09-05,        utc 09-05                     kills 3
//	mD : start_time 09-03 (dans la fenêtre), utc 12-01 (hors)   kills 9
//
// Ordre canonique dans la fenêtre 08-01 / 10-01 : mA, mB, mC (mD exclu).
func newE3PlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	shared := openMemDB(t)
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, game_variant_id VARCHAR, is_firefight BOOLEAN,
			playlist_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER,
			outcome INTEGER, kills INTEGER, deaths INTEGER)`,
		`CREATE TABLE highlight_events (match_id VARCHAR, event_type VARCHAR, time_ms BIGINT, xuid VARCHAR)`,
		`CREATE TABLE ` + KillEventsCanonicalTable + ` (match_id VARCHAR, feed_killer_xuid VARCHAR,
			victim_xuid VARCHAR, time_ms BIGINT)`,
		`INSERT INTO match_registry VALUES
			('mA', NULL,                  '2026-09-01 10:00:00+00', 'v', FALSE, 'pl'),
			('mB', '2026-09-10 10:00:00', '2026-09-02 10:00:00+00', 'v', FALSE, 'pl'),
			('mC', '2026-09-05 10:00:00', '2026-09-05 10:00:00+00', 'v', FALSE, 'pl'),
			('mD', '2026-09-03 10:00:00', '2026-12-01 10:00:00+00', 'v', FALSE, 'pl')`,
		`INSERT INTO match_participants VALUES
			('mA', '` + e3XUID + `', 0, 2, 1, 0), ('mB', '` + e3XUID + `', 0, 2, 2, 0),
			('mC', '` + e3XUID + `', 0, 2, 3, 0), ('mD', '` + e3XUID + `', 0, 2, 9, 0)`,
	}
	for _, s := range stmts {
		if _, err := shared.Exec(ctx, s); err != nil {
			t.Fatalf("fixture E3 : %v\n%s", err, s)
		}
	}
	return &PlayerDB{Player: openMemDB(t), Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: e3XUID, Gamertag: "E3", TitleSlug: "halo_infinite"}
}

var (
	e3Since = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	e3Until = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
)

// E3.1 — LoadAxisSamples (radar) : filtre de fenêtre et tri sur le temps canonique.
func TestE3_LoadAxisSamples_TempsCanonique(t *testing.T) {
	got, err := NewCampaignSampleProvider(newE3PlayerDB(t)).LoadAxisSamples(context.Background(),
		e3XUID, "halo_infinite", "combat", campaign.AxisKindRadar, "all", e3Since, e3Until)
	if err != nil {
		t.Fatalf("LoadAxisSamples : %v", err)
	}
	if want := []float64{1, 2, 3}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("échantillons %v, attendu %v (ordre et fenêtre sur start_time_utc)", got, want)
	}
}

// E3.1 — loadLUSRMatchIDsInWindow (composante LUSR) : même exigence.
func TestE3_LoadAxisSamples_LUSRTempsCanonique(t *testing.T) {
	pdb := newE3PlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `CREATE TABLE lusr_component_history_latest
		(match_id VARCHAR, component_name VARCHAR, value DOUBLE)`); err != nil {
		t.Fatal(err)
	}
	for id, v := range map[string]float64{"mA": 0.1, "mB": 0.2, "mC": 0.3, "mD": 0.9} {
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO lusr_component_history_latest VALUES (?, 'k', ?)`, id, v); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewCampaignSampleProvider(pdb).LoadAxisSamples(ctx, e3XUID, "halo_infinite",
		"k", campaign.AxisKindLUSRComponent, "all", e3Since, e3Until)
	if err != nil {
		t.Fatalf("LoadAxisSamples LUSR : %v", err)
	}
	if want := []float64{0.1, 0.2, 0.3}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("échantillons %v, attendu %v", got, want)
	}
}

// E3.2 — ListRecentPvPMatchIDs : les N plus récents au temps canonique, rendus du plus
// ancien au plus récent ; un import à start_time NULL n'est pas écarté.
func TestE3_ListRecentPvPMatchIDs_TempsCanonique(t *testing.T) {
	repo := NewEngagementScoreRepo(newE3PlayerDB(t))
	// Au temps canonique : mA 09-01, mB 09-02, mC 09-05, mD 12-01.
	for limit, want := range map[int]string{
		4: "[mA mB mC mD]",
		2: "[mC mD]",
	} {
		got, err := repo.ListRecentPvPMatchIDs(context.Background(), e3XUID, limit)
		if err != nil {
			t.Fatalf("limit %d : %v", limit, err)
		}
		if fmt.Sprint(got) != want {
			t.Errorf("limit %d : %v, attendu %s", limit, got, want)
		}
	}
}

// E3.2 — une ligne illisible dans highlight_events fait ÉCHOUER la lecture (l'appelant
// sait traiter l'erreur) au lieu de produire une courbe tronquée en silence.
func TestE3_LoadEventsForMatch_LigneIllisibleRemonte(t *testing.T) {
	pdb := newE3PlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, `INSERT INTO highlight_events VALUES
		('mA', 'kill', 1000, 'x'), ('mA', NULL, 2000, 'x')`); err != nil {
		t.Fatal(err)
	}
	evs, err := NewEngagementScoreRepo(pdb).LoadEventsForMatch(ctx, "mA")
	if err == nil {
		t.Fatalf("erreur attendue pour la ligne à event_type NULL, obtenu %d événements", len(evs))
	}
}

// E3.2 — coefficients : ligne illisible = erreur, pas une liste tronquée.
func TestE3_LoadAllCoefficients_LigneIllisibleRemonte(t *testing.T) {
	db := &PlayerDB{Player: openMemDB(t), XUID: "x"}
	ctx := context.Background()
	if _, err := db.Player.Exec(ctx, `CREATE TABLE engagement_coefficients (xuid VARCHAR, mode_category VARCHAR,
		coef_team_share DOUBLE, coef_lobby_share DOUBLE, n_matches INTEGER, last_updated TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Player.Exec(ctx, `INSERT INTO engagement_coefficients
		(xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated)
		VALUES ('x', 'arena', 1, 1, 5, NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngagementScoreRepo(db).LoadAllCoefficients(ctx, "x"); err == nil {
		t.Fatal("erreur attendue pour la ligne à last_updated NULL")
	}
}

// E3.2 — repli best-effort sur les kills synthétiques : la dégradation (médailles seules)
// est journalisée AVANT d'avoir lieu, avec l'erreur.
func TestE3_KillsSynthetiques_ErreurJournalisee(t *testing.T) {
	pdb := newE3PlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, `INSERT INTO highlight_events VALUES ('mA', 'medal', 500, 'x')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pdb.Shared.Exec(ctx, `INSERT INTO `+KillEventsCanonicalTable+`
		VALUES ('mA', 'k', 'v', NULL)`); err != nil { // time_ms NULL : illisible
		t.Fatal(err)
	}
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(prev)

	evs, err := NewEngagementScoreRepo(pdb).LoadEventsForMatch(ctx, "mA")
	if err != nil {
		t.Fatalf("la dégradation best-effort ne doit pas échouer : %v", err)
	}
	if len(evs) != 1 {
		t.Errorf("médailles seules attendues, obtenu %d événements", len(evs))
	}
	if out := buf.String(); !strings.Contains(out, "level=WARN") || !strings.Contains(out, "err=") {
		t.Errorf("avertissement journalisé avec l'erreur attendu, journal : %q", out)
	}
}
