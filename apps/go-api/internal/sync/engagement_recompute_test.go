//go:build integration

// Package sync — engagement_recompute_test.go : tests integration du
// recompute des coefficients d'engagement (Phase recompute coefs).
//
// Strategie : DuckDB :memory: avec schema realiste (player_match_enrichment +
// engagement_coefficients), inserts en lot, batchRecomputeCoefficients,
// assertions sur le contenu de engagement_coefficients.
//
// Couvre :
//   - happy path : >= 10 samples valides → coef persiste, != 1.0
//   - insufficient history : < 10 samples → no save
//   - missing paces columns → skip silencieux
//   - missing coefficients table → skip silencieux
//   - filtre mode_category : 2 modes → 2 coefs persistes independamment
//   - idempotent : 2 runs successifs UPSERT (last_updated change, valeurs identiques)
//   - filtrage outliers : AFK et lobby AFK exclus de la mediane
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

// =============================================================================
// Setup helpers
// =============================================================================

// openPlayerForRecompute ouvre une DuckDB :memory: avec le schema minimal
// requis par batchRecomputeCoefficients : player_match_enrichment (avec
// les colonnes paces) + engagement_coefficients.
func openPlayerForRecompute(t *testing.T, withPaces, withCoefsTable bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	// Player match enrichment — schema courant + nouvelles colonnes paces.
	pmeDDL := `
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			xuid VARCHAR,
			engagement_score DOUBLE,
			engagement_score_brut DOUBLE,
			engagement_score_confidence VARCHAR,
			mode_category VARCHAR,
			updated_at TIMESTAMP
	`
	if withPaces {
		pmeDDL += `,
			engagement_pace_player DOUBLE,
			engagement_pace_team DOUBLE,
			engagement_pace_lobby DOUBLE,
			engagement_player_activity INTEGER`
	}
	pmeDDL += `);`
	if _, err := db.Exec(pmeDDL); err != nil {
		t.Fatalf("CREATE player_match_enrichment: %v", err)
	}
	// Append-only #23046 : convertit player_match_enrichment + crée la vue _latest
	// (loadRatioSamples lit player_match_enrichment_latest) — UNIQUEMENT si withPaces.
	// Les tests withPaces=false veulent justement des colonnes paces ABSENTES pour
	// exercer le skip 'unavailable' (batchRecomputeCoefficients court-circuite sur
	// pacesColumnsExist AVANT de lire la vue ; la conversion ajouterait les paces).
	if withPaces {
		if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
			t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
		}
	}

	if withCoefsTable {
		coefDDL := `
			CREATE TABLE engagement_coefficients (
				xuid             VARCHAR NOT NULL,
				mode_category    VARCHAR NOT NULL,
				coef_team_share  DOUBLE NOT NULL,
				coef_lobby_share DOUBLE NOT NULL,
				n_matches        INTEGER NOT NULL,
				last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (xuid, mode_category)
			);
		`
		if _, err := db.Exec(coefDDL); err != nil {
			t.Fatalf("CREATE engagement_coefficients: %v", err)
		}
		// Bins de reponse (lobby-anchored v2) — accompagne la table coefs.
		binsDDL := `
			CREATE TABLE engagement_response_bins (
				xuid          VARCHAR NOT NULL,
				mode_category VARCHAR NOT NULL,
				intensity_bin VARCHAR NOT NULL,
				lower_bound   DOUBLE NOT NULL,
				upper_bound   DOUBLE NOT NULL,
				coef_lobby    DOUBLE NOT NULL,
				n_matches     INTEGER NOT NULL,
				last_updated  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (xuid, mode_category, intensity_bin)
			);
		`
		if _, err := db.Exec(binsDDL); err != nil {
			t.Fatalf("CREATE engagement_response_bins: %v", err)
		}
	}

	return db
}

// loadBinsCount lit le nombre de bins persistes et leurs coefs par bin.
func loadBinsCount(t *testing.T, db *sql.DB, xuid, mode string) (n int, coefByBin map[string]float64, nByBin map[string]int) {
	t.Helper()
	rows, err := db.Query(`
		SELECT intensity_bin, coef_lobby, n_matches
		FROM engagement_response_bins
		WHERE xuid = ? AND mode_category = ?
		ORDER BY lower_bound
	`, xuid, mode)
	if err != nil {
		t.Fatalf("load bins: %v", err)
	}
	defer rows.Close()
	coefByBin = map[string]float64{}
	nByBin = map[string]int{}
	for rows.Next() {
		var bin string
		var coef float64
		var nm int
		if err := rows.Scan(&bin, &coef, &nm); err != nil {
			t.Fatalf("scan bin: %v", err)
		}
		coefByBin[bin] = coef
		nByBin[bin] = nm
		n++
	}
	return n, coefByBin, nByBin
}

// insertPaceRow insère une row avec paces — helper pour les tests.
func insertPaceRow(t *testing.T, db *sql.DB, matchID, xuid, mode string, paceJoueur, paceTeam, paceLobby float64, activity int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO player_match_enrichment (
			match_id, xuid, mode_category,
			engagement_pace_player, engagement_pace_team, engagement_pace_lobby,
			engagement_player_activity
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, xuid, mode, paceJoueur, paceTeam, paceLobby, activity)
	if err != nil {
		t.Fatalf("insert pace row %s: %v", matchID, err)
	}
}

// insertConstantRatioBatch insère N matchs avec un ratio team/lobby constant.
func insertConstantRatioBatch(t *testing.T, db *sql.DB, xuid, mode string, n int, ratioTeam, ratioLobby float64) {
	t.Helper()
	const baseTeam = 10.0
	const baseLobby = 12.0
	for i := 0; i < n; i++ {
		paceJoueur := baseTeam * ratioTeam
		paceLobby := baseLobby
		if ratioLobby > 0 {
			paceLobby = paceJoueur / ratioLobby
		}
		insertPaceRow(t, db,
			fmt.Sprintf("%s_m%d", mode, i),
			xuid, mode,
			paceJoueur, baseTeam, paceLobby, 30,
		)
	}
}

// loadCoef lit la row coef pour assertion.
func loadCoef(t *testing.T, db *sql.DB, xuid, mode string) (coefTeam, coefLobby float64, nMatches int, found bool) {
	t.Helper()
	row := db.QueryRow(`
		SELECT coef_team_share, coef_lobby_share, n_matches
		FROM engagement_coefficients
		WHERE xuid = ? AND mode_category = ?
	`, xuid, mode)
	err := row.Scan(&coefTeam, &coefLobby, &nMatches)
	if err == sql.ErrNoRows {
		return 0, 0, 0, false
	}
	if err != nil {
		t.Fatalf("load coef: %v", err)
	}
	return coefTeam, coefLobby, nMatches, true
}

// =============================================================================
// Tests
// =============================================================================

func TestBatchRecomputeCoefficients_HappyPath(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 30, 1.25, 1.10)

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 1 {
		t.Errorf("nUpdated want 1 (PvP_ranked only), got %d", n)
	}

	coefTeam, coefLobby, nMatches, found := loadCoef(t, db, "xuid-1", "PvP_ranked")
	if !found {
		t.Fatal("coef row not persisted")
	}
	// coef_team_share est INERTE (D5) : la colonne reste NOT NULL, on y écrit 1.0.
	if coefTeam != 1.0 {
		t.Errorf("coef_team_share doit être inerte (1.0), got %v", coefTeam)
	}
	if math.Abs(coefLobby-1.10) > 1e-6 {
		t.Errorf("coef_lobby_share want ~1.10, got %v", coefLobby)
	}
	if nMatches != 30 {
		t.Errorf("n_matches want 30, got %d", nMatches)
	}
	// Vérifie que le coef lobby n'est pas resté à 1.0 (sinon courbes superposées).
	if coefLobby == 1.0 {
		t.Errorf("BUG racine non corrigé : coef_lobby reste 1.0 après recompute")
	}
}

// TestBatchRecomputeCoefficients_ResponseBinsPersisted : le recompute persiste
// aussi les 3 bins de reponse (terciles d'intensite) avec des coefs decroissants
// pour un joueur qui repond mal aux matchs intenses.
func TestBatchRecomputeCoefficients_ResponseBinsPersisted(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	xuid, mode := "xuid-1", "PvP_ranked"
	// 15 matchs calmes (paceLobby~2, ratio 1.5), 15 standards (5, 1.0),
	// 15 chaotiques (10, 0.5). paceTeam mis egal a paceLobby (>= seuil).
	insertBinBatch := func(prefix string, n int, paceLobby, ratio float64) {
		for i := 0; i < n; i++ {
			insertPaceRow(t, db, fmt.Sprintf("%s_m%d", prefix, i), xuid, mode,
				paceLobby*ratio, paceLobby, paceLobby, 30)
		}
	}
	insertBinBatch("calme", 15, 2.0, 1.5)
	insertBinBatch("standard", 15, 5.0, 1.0)
	insertBinBatch("chaotique", 15, 10.0, 0.5)

	if _, err := batchRecomputeCoefficients(context.Background(), db, xuid); err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}

	n, coefByBin, nByBin := loadBinsCount(t, db, xuid, mode)
	if n != 3 {
		t.Fatalf("want 3 bins persisted, got %d", n)
	}
	if !(coefByBin["calme"] > coefByBin["standard"] && coefByBin["standard"] > coefByBin["chaotique"]) {
		t.Errorf("coefs doivent decroitre avec l'intensite : %v", coefByBin)
	}
	for _, bin := range []string{"calme", "standard", "chaotique"} {
		if nByBin[bin] < 10 {
			t.Errorf("bin %s : n want >= 10, got %d", bin, nByBin[bin])
		}
	}
}

func TestBatchRecomputeCoefficients_InsufficientHistory(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	// 5 samples → strictement < MinMatchesForCoef=10
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 5, 1.25, 1.10)

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 0 {
		t.Errorf("nUpdated want 0 (insufficient), got %d", n)
	}
	if _, _, _, found := loadCoef(t, db, "xuid-1", "PvP_ranked"); found {
		t.Errorf("coef row should NOT be persisted with insufficient history")
	}
}

func TestBatchRecomputeCoefficients_MissingPacesColumns(t *testing.T) {
	// Pas de colonnes paces — simule pre-migration recompute coefs
	db := openPlayerForRecompute(t, false, true)
	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("expect silent skip, got err: %v", err)
	}
	if n != 0 {
		t.Errorf("nUpdated want 0 on missing paces, got %d", n)
	}
}

func TestBatchRecomputeCoefficients_MissingCoefficientsTable(t *testing.T) {
	// Paces présents mais table coefs absente — simule migration partielle
	db := openPlayerForRecompute(t, true, false)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 30, 1.25, 1.10)

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("expect silent skip, got err: %v", err)
	}
	if n != 0 {
		t.Errorf("nUpdated want 0 on missing coefs table, got %d", n)
	}
}

func TestBatchRecomputeCoefficients_TwoModesIndependent(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 20, 1.40, 1.20)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_unranked", 25, 0.85, 0.90)

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 2 {
		t.Errorf("nUpdated want 2 (both modes), got %d", n)
	}

	_, lR, _, ok := loadCoef(t, db, "xuid-1", "PvP_ranked")
	if !ok || math.Abs(lR-1.20) > 1e-6 {
		t.Errorf("PvP_ranked coef_lobby unexpected: %v", lR)
	}
	_, lU, _, ok := loadCoef(t, db, "xuid-1", "PvP_unranked")
	if !ok || math.Abs(lU-0.90) > 1e-6 {
		t.Errorf("PvP_unranked coef_lobby unexpected: %v", lU)
	}
}

func TestBatchRecomputeCoefficients_Idempotent(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 20, 1.30, 1.10)

	// Run 1
	n1, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("run1: %v", err)
	}
	if n1 != 1 {
		t.Errorf("run1 nUpdated want 1, got %d", n1)
	}
	c1, l1, m1, _ := loadCoef(t, db, "xuid-1", "PvP_ranked")

	// Run 2 — même data, doit produire les mêmes valeurs (UPSERT idempotent)
	n2, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("run2: %v", err)
	}
	if n2 != 1 {
		t.Errorf("run2 nUpdated want 1, got %d", n2)
	}
	c2, l2, m2, _ := loadCoef(t, db, "xuid-1", "PvP_ranked")
	if c1 != c2 || l1 != l2 || m1 != m2 {
		t.Errorf("non-idempotent: run1=(%v,%v,%d) run2=(%v,%v,%d)", c1, l1, m1, c2, l2, m2)
	}
}

func TestBatchRecomputeCoefficients_OutlierFiltering(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	// 15 samples valides ratio=1.0
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 15, 1.0, 1.0)
	// 10 samples AFK (activity=2 < seuil 3) avec ratio aberrant 50.0
	for i := 0; i < 10; i++ {
		insertPaceRow(t, db,
			fmt.Sprintf("afk_m%d", i),
			"xuid-1", "PvP_ranked",
			500, 10, 10, 2,
		)
	}
	// 5 samples lobby AFK (PaceTeam < 1.0) — exclus du ratio team
	for i := 0; i < 5; i++ {
		insertPaceRow(t, db,
			fmt.Sprintf("lobbyAfk_m%d", i),
			"xuid-1", "PvP_ranked",
			0.4, 0.5, 0.4, 30,
		)
	}

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 1 {
		t.Errorf("nUpdated want 1, got %d", n)
	}
	_, coefLobby, nMatches, _ := loadCoef(t, db, "xuid-1", "PvP_ranked")
	// Sans filtre, mediane serait tirée par les outliers à 50. Avec filtre,
	// la mediane reste à 1.0 (les 15 samples valides dominent).
	if math.Abs(coefLobby-1.0) > 1e-9 {
		t.Errorf("outlier filter failed: coef_lobby want 1.0, got %v", coefLobby)
	}
	if nMatches != 15 {
		t.Errorf("n_matches want 15 (valid only), got %d", nMatches)
	}
}

func TestBatchRecomputeCoefficients_RespectsLimitMostRecent(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	xuid := "xuid-1"
	mode := "PvP_ranked"

	// Insère 250 matchs : les 50 premiers (anciens via match_id "old_*") avec
	// ratio 0.5, les 200 suivants ("recent_*") avec ratio 1.5. ORDER BY
	// match_id DESC dans LoadRatioSamples → "recent_*" gagnent (préfixe
	// "recent_" > "old_"), donc le LIMIT=200 doit retenir QUE les recents.
	for i := 0; i < 50; i++ {
		paceJoueur := 5.0
		insertPaceRow(t, db, fmt.Sprintf("old_%03d", i), xuid, mode, paceJoueur, 10, 10, 30)
	}
	for i := 0; i < 200; i++ {
		paceJoueur := 15.0
		insertPaceRow(t, db, fmt.Sprintf("recent_%03d", i), xuid, mode, paceJoueur, 10, 10, 30)
	}

	n, err := batchRecomputeCoefficients(context.Background(), db, xuid)
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 1 {
		t.Errorf("nUpdated want 1, got %d", n)
	}
	_, coefLobby, nMatches, _ := loadCoef(t, db, xuid, mode)
	// 200 samples "recent_*" tous à ratio 1.5 → mediane 1.5
	if math.Abs(coefLobby-1.5) > 1e-9 {
		t.Errorf("limit=200 most recent should give 1.5, got %v (drift if limit ignored)", coefLobby)
	}
	if nMatches != 200 {
		t.Errorf("n_matches want 200 (limit), got %d", nMatches)
	}
}

func TestBatchRecomputeCoefficients_ObservabilityCounters(t *testing.T) {
	observability.Reset()
	db := openPlayerForRecompute(t, true, true)
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_ranked", 15, 1.20, 1.10)
	// PvP_unranked : 5 samples insuffisants → skip counter
	insertConstantRatioBatch(t, db, "xuid-1", "PvP_unranked", 5, 1.0, 1.0)

	if _, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1"); err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}

	if got := observability.LoadCounter("engagement_coef_recomputed_total"); got != 1 {
		t.Errorf("engagement_coef_recomputed_total want 1, got %d", got)
	}
	if got := observability.LoadCounter("engagement_coef_skipped_insufficient_history"); got != 1 {
		t.Errorf("engagement_coef_skipped_insufficient_history want 1, got %d", got)
	}
	// Bucket 1.1..1.3 doit avoir +1 (coef_lobby = 1.10)
	if got := observability.LoadCounter("engagement_coef_lobby_bucket_1_1_to_1_3"); got != 1 {
		t.Errorf("bucket 1_1_to_1_3 want 1, got %d", got)
	}
}

func TestBatchRecomputeCoefficients_ObservabilityUnavailable(t *testing.T) {
	observability.Reset()
	// Pas de paces columns → unavailable counter incremented
	db := openPlayerForRecompute(t, false, true)
	if _, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1"); err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if got := observability.LoadCounter("engagement_unavailable_skips_total"); got != 1 {
		t.Errorf("engagement_unavailable_skips_total want 1, got %d", got)
	}
}

func TestCoefBucket(t *testing.T) {
	cases := []struct {
		coef float64
		want string
	}{
		{0.3, "lt_0_5"},
		{0.5, "0_5_to_0_7"},
		{0.7, "0_7_to_0_9"},
		{0.9, "0_9_to_1_1"},
		{1.0, "0_9_to_1_1"},
		{1.1, "1_1_to_1_3"},
		{1.3, "1_3_to_1_5"},
		{1.5, "1_5_to_2_0"},
		{2.0, "gte_2_0"},
		{5.0, "gte_2_0"},
	}
	for _, c := range cases {
		if got := coefBucket(c.coef); got != c.want {
			t.Errorf("coefBucket(%v) want %s, got %s", c.coef, c.want, got)
		}
	}
}

func TestBatchRecomputeCoefficients_PvEFiltered(t *testing.T) {
	db := openPlayerForRecompute(t, true, true)
	// Insère 15 samples PvE (mode_category="PvE_firefight" — non géré par
	// engagementCoefModes). batchRecomputeCoefficients ne doit pas créer de
	// coef pour cette catégorie.
	for i := 0; i < 15; i++ {
		insertPaceRow(t, db,
			fmt.Sprintf("pve_m%d", i),
			"xuid-1", "PvE_firefight",
			15, 10, 10, 30,
		)
	}

	n, err := batchRecomputeCoefficients(context.Background(), db, "xuid-1")
	if err != nil {
		t.Fatalf("batchRecomputeCoefficients: %v", err)
	}
	if n != 0 {
		t.Errorf("nUpdated want 0 (PvE not in engagementCoefModes), got %d", n)
	}
	if _, _, _, found := loadCoef(t, db, "xuid-1", "PvE_firefight"); found {
		t.Errorf("PvE coef should NOT be persisted")
	}
}
