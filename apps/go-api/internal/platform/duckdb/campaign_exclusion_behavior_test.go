//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/campaign"
)

// TestCampaignExclusion_FiltersCampaignMatch — preuve COMPORTEMENTALE (item H1) :
// sur une DuckDB en mémoire reproduisant match_registry ⨝ match_participants, une
// requête de stats portant le token d'exclusion résolu pour Halo 5 EXCLUT le match
// de mode Campagne (identifié par game_variant_id) tout en gardant le match
// d'arène ; résolue pour Infinite (no-op), elle renvoie les deux (title-aware).
func TestCampaignExclusion_FiltersCampaignMatch(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	// Schéma minimal (colonnes utilisées par la requête témoin).
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE match_registry (match_id VARCHAR, game_variant_id VARCHAR);
		CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	campGUID := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	const arenaGUID = "aaaaaaaa-0000-0000-0000-000000000001"
	const xuid = "2533274800000001"

	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_registry VALUES ('arena1', ?), ('camp1', ?)`, arenaGUID, campGUID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_participants VALUES ('arena1', ?), ('camp1', ?)`, xuid, xuid); err != nil {
		t.Fatalf("seed participants: %v", err)
	}

	// Requête témoin : même squelette JOIN + token que les requêtes de stats
	// per-player réelles (Q22/Q23/Q26/...).
	base := `
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry r ON r.match_id = mp.match_id
		WHERE mp.xuid = ? ` + campaignExclusionToken + `
		ORDER BY mp.match_id`

	// Halo 5 : le match Campagne doit disparaître.
	got := queryMatchIDs(t, db, resolveCampaignExclusion(base, "halo_5", "r"), xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("halo_5 : attendu %v (Campagne exclue), obtenu %v", want, got)
	}

	// Infinite : no-op, les deux matchs remontent (le GUID Campagne n'existe pas
	// pour Infinite → aucune régression sur les autres titres).
	got = queryMatchIDs(t, db, resolveCampaignExclusion(base, "halo_infinite", "r"), xuid)
	if want := []string{"arena1", "camp1"}; !equalStrings(got, want) {
		t.Errorf("halo_infinite : attendu %v (no-op), obtenu %v", want, got)
	}

	// Forme SOUS-REQUÊTE (participants-only, SANS jointure registre — chemin
	// weapon_kills / compare / count / leaderboard).
	sub := `SELECT mp.match_id FROM match_participants mp WHERE mp.xuid = ?` +
		excludeCampaignByMatchID("halo_5", "mp.match_id") + ` ORDER BY mp.match_id`
	got = queryMatchIDs(t, db, sub, xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("sous-requête halo_5 : attendu %v (Campagne exclue), obtenu %v", want, got)
	}

	// Variante title-agnostic (GamertagRepo) : mêmes GUID, no-op si absents.
	subAll := `SELECT mp.match_id FROM match_participants mp WHERE mp.xuid = ?` +
		excludeAllCampaignByMatchID("mp.match_id") + ` ORDER BY mp.match_id`
	got = queryMatchIDs(t, db, subAll, xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("sous-requête title-agnostic : attendu %v, obtenu %v", want, got)
	}

	// Forme TOKEN → SOUS-REQUÊTE (resolveCampaignExclusionByMatchID) : chemin des
	// CTE my_history / lecteurs squad qui portent le token SANS alias registre dans
	// sa portée (Q10/Q23b/Q26/Q28/Q29 relations & squad). Résolu AVANT fmt.Sprintf.
	tokSub := `SELECT mp.match_id FROM match_participants mp WHERE mp.xuid = ?` +
		campaignExclusionToken + ` ORDER BY mp.match_id`
	got = queryMatchIDs(t, db, resolveCampaignExclusionByMatchID(tokSub, "halo_5", "mp.match_id"), xuid)
	if want := []string{"arena1"}; !equalStrings(got, want) {
		t.Errorf("token→sous-requête halo_5 : attendu %v (Campagne exclue), obtenu %v", want, got)
	}
	got = queryMatchIDs(t, db, resolveCampaignExclusionByMatchID(tokSub, "halo_infinite", "mp.match_id"), xuid)
	if want := []string{"arena1", "camp1"}; !equalStrings(got, want) {
		t.Errorf("token→sous-requête halo_infinite : attendu %v (no-op), obtenu %v", want, got)
	}
}

func queryMatchIDs(t *testing.T, db *sql.DB, query, xuid string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), query, xuid)
	if err != nil {
		t.Fatalf("query: %v\n%s", err, query)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ─── Lecteurs corrigés au lot B4 (backlog 2026-09-26, D-5) ───────────────────
//
// Chaque cas sème, pour le même joueur, un match d'arène et un match de Campagne
// (game_variant_id de la source unique), puis lit par le VRAI lecteur. Halo 5 :
// la Campagne disparaît. Halo Infinite (aucune variante masquée) : le résolveur
// est neutre, les deux matchs restent.

const (
	b4XUIDA = "2533274800000001"
	b4XUIDB = "2533274800000002"
)

// b4Titres : titre → la Campagne est-elle masquée ?
var b4Titres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}}

// newB4PlayerDB : shared minimal (colonnes lues par les lecteurs du lot) avec un
// match d'arène (arena1, 01/09) et un match de Campagne PLUS RÉCENT (camp1, 02/09),
// joués par A et B dans la même équipe. A : 5 frags en arène, 50 en Campagne.
func newB4PlayerDB(t *testing.T, titleSlug string) *PlayerDB {
	t.Helper()
	shared := openMemDB(t)
	ctx := context.Background()
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	stmts := []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, game_variant_id VARCHAR, is_firefight BOOLEAN,
			playlist_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER,
			outcome INTEGER, kills INTEGER, deaths INTEGER)`,
		`INSERT INTO match_registry VALUES
			('arena1', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', 'aaaaaaaa-0000-0000-0000-000000000001', FALSE, 'pl'),
			('camp1',  '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', '` + camp + `', FALSE, 'pl')`,
		`INSERT INTO match_participants VALUES
			('arena1', '` + b4XUIDA + `', 0, 2, 5, 1), ('arena1', '` + b4XUIDB + `', 0, 2, 3, 2),
			('camp1',  '` + b4XUIDA + `', 0, 2, 50, 0), ('camp1',  '` + b4XUIDB + `', 0, 2, 40, 0)`,
	}
	for _, s := range stmts {
		if _, err := shared.Exec(ctx, s); err != nil {
			t.Fatalf("fixture B4 : %v\n%s", err, s)
		}
	}
	return &PlayerDB{Player: openMemDB(t), Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: b4XUIDA, Gamertag: "B4", TitleSlug: titleSlug}
}

// b4Attendu rend la valeur attendue selon que la Campagne est masquée.
func b4Attendu[T any](masquee bool, sans, avec T) T {
	if masquee {
		return sans
	}
	return avec
}

// B4.2.1 — CompareRepo.GetEncounterStats : matchs communs A/B.
func TestCampaignExclusion_GetEncounterStats(t *testing.T) {
	for _, tc := range b4Titres {
		enc, err := NewCompareRepo(newB4PlayerDB(t, tc.slug)).GetEncounterStats(context.Background(), b4XUIDA, b4XUIDB)
		if err != nil || enc == nil {
			t.Fatalf("%s : GetEncounterStats = %v, %v", tc.slug, enc, err)
		}
		if want := b4Attendu(tc.masquee, 1, 2); enc.TotalEncounters != want {
			t.Errorf("%s : %d rencontres, attendu %d (Campagne masquée : %v)", tc.slug, enc.TotalEncounters, want, tc.masquee)
		}
	}
}

// B4.2.2 — CampaignSampleProvider.LoadAxisSamples (axe radar « combat » = frags).
func TestCampaignExclusion_LoadAxisSamples(t *testing.T) {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range b4Titres {
		got, err := NewCampaignSampleProvider(newB4PlayerDB(t, tc.slug)).LoadAxisSamples(context.Background(),
			b4XUIDA, tc.slug, "combat", campaign.AxisKindRadar, "all", since, until)
		if err != nil {
			t.Fatalf("%s : LoadAxisSamples : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, []float64{5}, []float64{5, 50}); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s : échantillons %v, attendu %v", tc.slug, got, want)
		}
	}
}

// B4.2.3 — EngagementScoreRepo.ListRecentPvPMatchIDs.
func TestCampaignExclusion_ListRecentPvPMatchIDs(t *testing.T) {
	for _, tc := range b4Titres {
		got, err := NewEngagementScoreRepo(newB4PlayerDB(t, tc.slug)).ListRecentPvPMatchIDs(context.Background(), b4XUIDA, 10)
		if err != nil {
			t.Fatalf("%s : ListRecentPvPMatchIDs : %v", tc.slug, err)
		}
		if want := b4Attendu(tc.masquee, []string{"arena1"}, []string{"arena1", "camp1"}); !equalStrings(got, want) {
			t.Errorf("%s : %v, attendu %v", tc.slug, got, want)
		}
	}
}

// B4.2.4 — CountCrossTitleCooccurrences (q31) : lecture du shared d'un AUTRE titre,
// sans slug : exclusion title-agnostic (excludeAllCampaignByMatchID).
func TestCampaignExclusion_CountCrossTitleCooccurrences(t *testing.T) {
	pdb := newB4PlayerDB(t, "halo_5")
	got, err := CountCrossTitleCooccurrences(context.Background(), pdb.Shared.SQLDb(), b4XUIDA, []string{b4XUIDB}, 1)
	if err != nil {
		t.Fatalf("CountCrossTitleCooccurrences : %v", err)
	}
	if got[b4XUIDB] != 1 {
		t.Errorf("matchs communs avec B = %d, attendu 1 (Campagne masquée) — %v", got[b4XUIDB], got)
	}
}
