// Package sync — assists_model.go : régression OLS multi-variée pour expected_assists.
//
// Calcule, pour chaque mode de jeu du joueur, un modèle linéaire :
//
//	assists ≈ β0 + β1·kills + β2·deaths + β3·damage_dealt + β4·damage_taken + β5·mmr_delta
//
// La régression est résolue par élimination gaussienne avec pivot partiel sur les
// équations normales 6×6. Aucune dépendance gonum.
//
// Seuil minimum : 15 matchs par mode pour un modèle fiable à 6 paramètres.
// Le fallback sur assists_model_coefs (modèle populationnel) reste dans match_view_service.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

const minAssistsSamples = 15

// assistsSample contient les features d'un match pour la régression.
type assistsSample struct {
	mode        string
	assists     float64
	kills       float64
	deaths      float64
	damageDealt float64
	damageTaken float64
	mmrDelta    float64
}

// assistsCoefs contient les coefficients OLS et les métadonnées d'un modèle.
type assistsCoefs struct {
	gameVariantName string
	intercept       float64
	coefKills       float64
	coefDeaths      float64
	coefDamageDealt float64
	coefDamageTaken float64
	coefMMRDelta    float64
	r2              float64
	n               int
}

// loadAssistsSamples charge les données d'entraînement depuis shared_matches_v2.
func loadAssistsSamples(ctx context.Context, sharedDB *sql.DB, xuid string) ([]assistsSample, error) {
	const q = `
		SELECT
			COALESCE(r.game_variant_name, '__unknown__'),
			CAST(p.assists       AS DOUBLE),
			CAST(p.kills         AS DOUBLE),
			CAST(p.deaths        AS DOUBLE),
			COALESCE(CAST(p.damage_dealt AS DOUBLE), 0.0),
			COALESCE(CAST(p.damage_taken AS DOUBLE), 0.0),
			COALESCE(
				CAST(p.team_mmr  AS DOUBLE) - CAST(p.enemy_mmr AS DOUBLE),
				0.0
			)
		FROM match_participants p
		JOIN match_registry r ON r.match_id = p.match_id
		WHERE p.xuid = ?
		  AND p.kills >= 0
		  AND p.assists IS NOT NULL
		  AND p.outcome != ?
		  AND NOT isnan(COALESCE(CAST(p.damage_dealt AS DOUBLE), 0.0))
		  AND NOT isnan(COALESCE(CAST(p.damage_taken AS DOUBLE), 0.0))
		ORDER BY r.game_variant_name
	`
	// PMT-5 : le code DNF (exclu de la régression) est lié comme PARAMÈTRE depuis la
	// constante canonique — plus de littéral brut « 4 » dans la chaîne SQL. La colonne
	// match_participants.outcome porte les codes bruts du titre (Halo, écrits par CE
	// moteur de sync) ; domain.OutcomeDNF == le raw_code DNF du manifeste Halo.
	rows, err := sharedDB.QueryContext(ctx, q, xuid, domain.OutcomeDNF)
	if err != nil {
		return nil, fmt.Errorf("loadAssistsSamples: %w", err)
	}
	defer rows.Close()

	var samples []assistsSample
	for rows.Next() {
		var s assistsSample
		if err := rows.Scan(&s.mode, &s.assists, &s.kills, &s.deaths,
			&s.damageDealt, &s.damageTaken, &s.mmrDelta); err != nil {
			return nil, fmt.Errorf("loadAssistsSamples scan: %w", err)
		}
		samples = append(samples, s)
	}
	return samples, rows.Err()
}

// groupByMode regroupe les échantillons par mode de jeu.
func groupByMode(samples []assistsSample) map[string][]assistsSample {
	m := make(map[string][]assistsSample)
	for _, s := range samples {
		m[s.mode] = append(m[s.mode], s)
	}
	return m
}

// fitOLS résout β = (X'X)⁻¹X'y par élimination gaussienne avec pivot partiel.
// X est n×6 : [1, kills, deaths, damage_dealt, damage_taken, mmr_delta].
// Retourne nil si le système est singulier ou dégénéré.
func fitOLS(rows []assistsSample) *assistsCoefs {
	n := len(rows)
	if n < minAssistsSamples {
		return nil
	}

	const p = 6 // nombre de paramètres

	// Accumuler X'X (p×p) et X'y (p×1) en une seule passe.
	var xtx [p][p]float64
	var xty [p]float64

	for _, r := range rows {
		x := [p]float64{1, r.kills, r.deaths, r.damageDealt, r.damageTaken, r.mmrDelta}
		y := r.assists
		for i := 0; i < p; i++ {
			xty[i] += x[i] * y
			for j := 0; j < p; j++ {
				xtx[i][j] += x[i] * x[j]
			}
		}
	}

	// Construire la matrice augmentée [X'X | X'y] pour l'élimination.
	var aug [p][p + 1]float64
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			aug[i][j] = xtx[i][j]
		}
		aug[i][p] = xty[i]
	}

	// Élimination gaussienne avec pivot partiel.
	for col := 0; col < p; col++ {
		// Trouver le pivot maximal.
		maxRow := col
		maxVal := math.Abs(aug[col][col])
		for row := col + 1; row < p; row++ {
			if v := math.Abs(aug[row][col]); v > maxVal {
				maxVal = v
				maxRow = row
			}
		}
		if maxVal < 1e-12 {
			return nil // système singulier
		}
		aug[col], aug[maxRow] = aug[maxRow], aug[col]

		pivot := aug[col][col]
		for j := col; j <= p; j++ {
			aug[col][j] /= pivot
		}
		for row := 0; row < p; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for j := col; j <= p; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	beta := [p]float64{}
	for i := 0; i < p; i++ {
		beta[i] = aug[i][p]
	}

	// Calcul R² = 1 - SS_res / SS_tot.
	yMean := 0.0
	for _, r := range rows {
		yMean += r.assists
	}
	yMean /= float64(n)

	ssTot, ssRes := 0.0, 0.0
	for _, r := range rows {
		x := [p]float64{1, r.kills, r.deaths, r.damageDealt, r.damageTaken, r.mmrDelta}
		pred := 0.0
		for i := 0; i < p; i++ {
			pred += beta[i] * x[i]
		}
		ssTot += (r.assists - yMean) * (r.assists - yMean)
		ssRes += (r.assists - pred) * (r.assists - pred)
	}

	r2 := 0.0
	if ssTot > 1e-12 {
		r2 = 1.0 - ssRes/ssTot
		if r2 < 0 {
			r2 = 0
		}
	}

	return &assistsCoefs{
		intercept:       beta[0],
		coefKills:       beta[1],
		coefDeaths:      beta[2],
		coefDamageDealt: beta[3],
		coefDamageTaken: beta[4],
		coefMMRDelta:    beta[5],
		r2:              r2,
		n:               n,
	}
}

// upsertAssistsModels persiste les modèles dans player_assists_model.
func upsertAssistsModels(ctx context.Context, playerDB *sql.DB, models []assistsCoefs) error {
	if len(models) == 0 {
		return nil
	}
	tx, err := playerDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsertAssistsModels begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// ART-safe : SELECT-then-UPDATE-or-INSERT par modèle (pas d'ON CONFLICT, qui
	// réécrit via l'index ART de la PK). player_assists_model : PK game_variant_name,
	// pas d'index secondaire muté. Sérialisé (tx mono-writer), basse fréquence.
	selStmt, err := tx.PrepareContext(ctx, `SELECT 1 FROM player_assists_model WHERE game_variant_name = ?`)
	if err != nil {
		return fmt.Errorf("upsertAssistsModels prepare select: %w", err)
	}
	defer selStmt.Close()
	updStmt, err := tx.PrepareContext(ctx, `
		UPDATE player_assists_model SET coef_intercept = ?, coef_kills = ?, coef_deaths = ?,
			coef_damage_dealt = ?, coef_damage_taken = ?, coef_mmr_delta = ?, r2 = ?, n_samples = ?, computed_at = ?
		WHERE game_variant_name = ?`)
	if err != nil {
		return fmt.Errorf("upsertAssistsModels prepare update: %w", err)
	}
	defer updStmt.Close()
	insStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO player_assists_model (
			game_variant_name, coef_intercept, coef_kills, coef_deaths,
			coef_damage_dealt, coef_damage_taken, coef_mmr_delta, r2, n_samples, computed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("upsertAssistsModels prepare insert: %w", err)
	}
	defer insStmt.Close()

	now := time.Now().UTC()
	for _, m := range models {
		var dummy int
		serr := selStmt.QueryRowContext(ctx, m.gameVariantName).Scan(&dummy)
		switch {
		case serr == nil:
			_, serr = updStmt.ExecContext(ctx, m.intercept, m.coefKills, m.coefDeaths,
				m.coefDamageDealt, m.coefDamageTaken, m.coefMMRDelta, m.r2, m.n, now, m.gameVariantName)
		case errors.Is(serr, sql.ErrNoRows):
			_, serr = insStmt.ExecContext(ctx, m.gameVariantName, m.intercept, m.coefKills, m.coefDeaths,
				m.coefDamageDealt, m.coefDamageTaken, m.coefMMRDelta, m.r2, m.n, now)
		}
		if serr != nil {
			return fmt.Errorf("upsertAssistsModels exec (%s): %w", m.gameVariantName, serr)
		}
	}
	return tx.Commit()
}

// batchComputePlayerAssistsModel calcule et persiste les modèles OLS per-mode.
// Si force=false, ne recalcule que si la table est vide.
// Retourne le nombre de modes mis à jour.
func batchComputePlayerAssistsModel(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, force bool) (int, error) {
	// Vérifier si la table existe (migration appliquée).
	var tableExistsVal string
	err := playerDB.QueryRowContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema='main' AND table_name='player_assists_model'",
	).Scan(&tableExistsVal)
	if err != nil {
		return 0, nil // table absente → skip silencieux
	}

	if !force {
		var existing int
		_ = playerDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM player_assists_model").Scan(&existing)
		if existing > 0 {
			return 0, nil
		}
	}

	samples, err := loadAssistsSamples(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("batchComputePlayerAssistsModel: %w", err)
	}
	if len(samples) == 0 {
		return 0, nil
	}

	byMode := groupByMode(samples)
	var models []assistsCoefs
	for mode, modeSamples := range byMode {
		coefs := fitOLS(modeSamples)
		if coefs == nil {
			continue
		}
		coefs.gameVariantName = mode
		models = append(models, *coefs)
		slog.DebugContext(ctx, "assists_model: mode fitted",
			"mode", mode, "n", coefs.n, "r2", fmt.Sprintf("%.3f", coefs.r2))
	}

	if len(models) == 0 {
		return 0, nil
	}

	if err := upsertAssistsModels(ctx, playerDB, models); err != nil {
		return 0, fmt.Errorf("batchComputePlayerAssistsModel upsert: %w", err)
	}
	return len(models), nil
}

// RunBackfillAssistsModel recalcule les modèles OLS expected_assists pour le joueur.
func (e *SyncEngine) RunBackfillAssistsModel(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillAssistsModel lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillAssistsModel OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 13a : acquireSharedWriter centralise lease + open
	// (Provider en B-swap, dblease + OpenSharedDB en legacy).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_assists_model"))
	if err != nil {
		return 0, fmt.Errorf("RunBackfillAssistsModel: %w", err)
	}
	defer releaseShared()

	return batchComputePlayerAssistsModel(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, force)
}
