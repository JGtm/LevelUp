// Package duckdb â€” squad_repo.go : accÃ¨s DB pour la page Escouade et SynthÃ¨se.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// SquadRepo implÃ©mente port.SquadRepository.
type SquadRepo struct {
	pdb *PlayerDB
}

// NewSquadRepo crÃ©e un SquadRepo pour un joueur.
func NewSquadRepo(pdb *PlayerDB) *SquadRepo {
	return &SquadRepo{pdb: pdb}
}

// LoadTopTeammates charge les meilleurs coÃ©quipiers du joueur (Q29, top 50).
//
// split cross-DB en 2 étapes.
//
//	Étape 1 (pdb.Player) : match_ids du joueur avec is_with_friends = TRUE.
//	Étape 2 (SharedReader) : aggregation sur match_participants restreinte
//	  aux match_ids retournés en étape 1, groupée par teammate xuid.
func (r *SquadRepo) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	matchIDs, err := r.loadWithFriendsMatchIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: %w", err)
	}
	if len(matchIDs) == 0 {
		return nil, nil
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: shared reader: %w", err)
	}
	defer release()

	// PMT-5 : win/win_rate title-aware (fallback "p1.outcome = 2" byte-identique Halo).
	// Ordre des %s du template : win (wins_together), win (win_rate), puis IN-list.
	winExpr := outcomeSQLEq(ctx, "p1.outcome", canonical.OutcomeWin, "p1.outcome = 2")
	// Masquage Campagne (Halo 5) : le set match_id « avec amis » peut contenir des
	// matchs Campagne coop → by-match-id (p1.match_id), résolu AVANT Sprintf. No-op
	// Infinite. Item backlog H1.
	tpl := resolveCampaignExclusionByMatchID(Q29TopTeammatesSharedTpl, pdbTitleSlug(r.pdb), "p1.match_id")
	query := fmt.Sprintf(tpl, winExpr, winExpr, Placeholders(len(matchIDs)))
	args := make([]any, 0, 2+len(matchIDs))
	args = append(args, xuid)
	args = append(args, ToAnySlice(matchIDs)...)
	args = append(args, xuid)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: %w", err)
	}
	defer rows.Close()

	var result []domain.TopTeammateRow
	for rows.Next() {
		var row domain.TopTeammateRow
		if err := rows.Scan(
			&row.XUID,
			&row.Gamertag,
			&row.GamesTogether,
			&row.WinsTogether,
			&row.WinRate,
			&row.AvgKills,
			&row.AvgDeaths,
			&row.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("LoadTopTeammates scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// loadWithFriendsMatchIDs retourne les match_ids où player_match_enrichment.is_with_friends
// est TRUE. Helper pour le split LoadTopTeammates (commit 9c.1).
func (r *SquadRepo) loadWithFriendsMatchIDs(ctx context.Context) ([]string, error) {
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.QueryRecovered(ctx2,
		`SELECT match_id FROM player_match_enrichment_latest WHERE is_with_friends = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("loadWithFriendsMatchIDs: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("loadWithFriendsMatchIDs scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LookupXUIDByGamertag rÃ©sout un gamertag (ILIKE, case-insensitive) vers son
// XUID via shared.xuid_aliases. Sert de fallback pour les coÃ©quipiers sÃ©lectionnÃ©s
// qui sortent du top 50 LoadTopTeammates (saisie libre dans la combobox).
//
// Si plusieurs aliases correspondent au mÃªme gamertag (changement de pseudo
// historique), on retourne le plus rÃ©cent. Si aucun alias, retourne ("", false, nil).
func (r *SquadRepo) LookupXUIDByGamertag(ctx context.Context, gamertag string) (string, bool, error) {
	gamertag = strings.TrimSpace(gamertag)
	if gamertag == "" {
		return "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// xuid_aliases : (xuid PK, gamertag, last_seen, source, updated_at)
	// Source canonique pour ce titre — peuplée par le sync engine. La DB globale
	// est obsolète (migration one-shot souvent vide).
	//
	// shared-only via SharedReader, naming root-level.
	const q = `
SELECT xuid
FROM xuid_aliases
WHERE gamertag ILIKE ?
ORDER BY last_seen DESC NULLS LAST
LIMIT 1`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag(%q): shared reader: %w", gamertag, err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, gamertag)
	if err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return "", false, rows.Err()
	}
	var xuid string
	if err := rows.Scan(&xuid); err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag scan: %w", err)
	}
	return xuid, xuid != "", nil
}

// LoadSquadMatches charge les matchs communs joueur+coÃ©quipier (Q30).
//
// split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q30SquadMatchesSharedQuery — match_participants
//	  ⨝ v_match_full ⨝ match_participants (teammate filter) + subquery
//	  medals_earned. 25 cols shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : merge LEFT JOIN — hydrate session_id, session_label,
//	  performance_score, is_with_friends.
func (r *SquadRepo) LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Étape 1 : shared rows.
	results, err := r.loadSquadMatchesShared(ctx, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	// Étape 2 + 3 : merge enrichments.
	matchIDs := make([]string, 0, len(results))
	for _, m := range results {
		matchIDs = append(matchIDs, m.MatchID)
	}
	enrichments, err := LoadPlayerMatchEnrichments(ctx, r.pdb.Player, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
	}
	// Proba de victoire pré-match (player.match_skill_rank) — best-effort : si la
	// lecture échoue, on dégrade en nil (colonne « Prob. vic. » vide).
	winProbs := r.loadExpectedWinProbs(ctx, matchIDs)
	for i := range results {
		if e, ok := enrichments[results[i].MatchID]; ok {
			if e.SessionID.Valid {
				// session_id est VARCHAR en prod ; SquadMatchRow.SessionID est *int
				// (legacy domain), on parse en best-effort.
				if v, perr := strconv.Atoi(e.SessionID.String); perr == nil {
					results[i].SessionID = &v
				}
			}
			if e.SessionLabel.Valid {
				v := e.SessionLabel.String
				results[i].SessionLabel = &v
			}
			if e.PerformanceScore.Valid {
				v := e.PerformanceScore.Float64
				results[i].PerformanceScore = &v
			}
			results[i].IsWithFriends = e.IsWithFriends
			// Badge narratif du match (0 = aucun) : alimente le marqueur de
			// dominance de la bande de résultats Escouade. Aucune requête
			// supplémentaire — la colonne est déjà chargée par LoadPlayerMatchEnrichments.
			results[i].DominanceFlag = e.DominanceFlag
		}
		if wp, ok := winProbs[results[i].MatchID]; ok {
			v := wp
			results[i].ExpectedWinProb = &v
		}
	}
	return results, nil
}

// loadExpectedWinProbs charge la proba de victoire pré-match par match_id depuis
// player.match_skill_rank (LUSR v2). MAX(expected_win_prob) car la valeur n'est
// posée que sur les rows LUSR (append-only N versions/match). Best-effort :
// retourne une map vide en cas d'erreur (dégradation gracieuse).
func (r *SquadRepo) loadExpectedWinProbs(ctx context.Context, matchIDs []string) map[string]float64 {
	out := make(map[string]float64, len(matchIDs))
	if len(matchIDs) == 0 {
		return out
	}
	query := fmt.Sprintf(QSquadExpectedWinProbTpl, Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "LoadSquadMatches: expected_win_prob indisponible (best-effort)", "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		var wp sql.NullFloat64
		if err := rows.Scan(&mid, &wp); err != nil {
			slog.WarnContext(ctx, "LoadSquadMatches: scan expected_win_prob", "err", err)
			return out
		}
		if wp.Valid {
			out[mid] = wp.Float64
		}
	}
	return out
}

// loadSquadMatchesShared exécute l'étape 1 du split LoadSquadMatches.
// Retourne les SquadMatchRow sans les cols PME (zero values).
func (r *SquadRepo) loadSquadMatchesShared(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	// Set perfect-kill résolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q30SquadMatchesSharedQuery, "me.medal_name_id", pdbTitleSlug(r.pdb))
	// Masquage Campagne (Halo 5) : liste des matchs communs escouade, registre
	// aliasé "r" (v_match_full). No-op Infinite. Item backlog H1.
	q = resolveCampaignExclusion(q, pdbTitleSlug(r.pdb), "r")
	rows, err := db.QueryContext(ctx, q, teammateXUID, playerXUID)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var result []domain.SquadMatchRow
	for rows.Next() {
		var row domain.SquadMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapName,
			&row.MapUI,
			&row.PairName,
			&row.PlaylistName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.DurationSeconds,
			&row.GameplayDurationSeconds,
			&row.T0Ms,
			&row.TeamMMR,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.EnemyMMR,
			&row.MyTeamScore,
			&row.EnemyTeamScore,
			&row.MapID,
			&row.PlaylistID,
			&row.PairNameFR,
			&row.PairID,
			&row.GameVariantID,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadTeammateMatches charge les stats du coÃ©quipier sur les matchs communs (Q31).
//
// query shared-only (match_participants x2 + v_match_full)
// migrée vers SharedReader.Get.
func (r *SquadRepo) LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTeammateMatches: shared reader: %w", err)
	}
	defer release()

	// Masquage Campagne (Halo 5) : liste des matchs d'un coéquipier, registre
	// aliasé "r" (v_match_full). No-op Infinite. Item backlog H1.
	q := resolveCampaignExclusion(Q31TeammateMatches, pdbTitleSlug(r.pdb), "r")
	rows, err := db.QueryContext(ctx, q, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadTeammateMatches: %w", err)
	}
	defer rows.Close()

	var result []domain.TeammateMatchRow
	for rows.Next() {
		var row domain.TeammateMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapUI,
			&row.PairName,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Ratio,
			&row.TimePlayedSecs,
			&row.TeamMMR,
			&row.Accuracy,
			&row.MyTeamScore,
			&row.EnemyTeamScore,
		); err != nil {
			return nil, fmt.Errorf("LoadTeammateMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadImpactEvents charge les Ã©vÃ©nements highlight pour une liste de match_ids (Q32 dynamique).
// matchIDs est la liste des identifiants â€” si vide, retourne nil directement.
//
// Title-agnostic (centralisé au niveau lecture) : highlight_events ne porte pas
// forcément les kills selon le titre. Infinite y stocke kill/death/medal ; Halo 5
// n'y stocke QUE des médailles, les kills horodatés vivant dans killer_victim_pairs.
// Si le lot chargé ne contient AUCUN kill/death (HasCanonicalKillOrDeath==false),
// on synthétise les kill/death depuis killer_victim_pairs (LoadKVPairs) via le
// helper partagé analysis.SynthesizeKillEventsFromKVPairs (source unique de la
// règle) et on les fusionne, triés par TimeMS. NO-OP sur Infinite (kills déjà
// présents → fallback jamais pris). Évite que les 4 builders Escouade (first
// events, intensité, matrice d'impact, squad V1) restent vides en H5.
func (r *SquadRepo) LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Construire la clause IN dynamiquement.
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q32SquadImpactEventsTemplate, strings.Join(placeholders, ","))

	// shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadImpactEvents: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadImpactEvents: %w", err)
	}
	defer rows.Close()

	var result []domain.ImpactEventRow
	for rows.Next() {
		var row domain.ImpactEventRow
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.EventType,
			&row.TimeMS,
		); err != nil {
			return nil, fmt.Errorf("LoadImpactEvents scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fallback title-agnostic : kills/deaths absents → synthèse depuis kvPairs.
	if !impactRowsHaveKillOrDeath(result) {
		kvPairs, kvErr := r.loadKVPairsOn(ctx, db, matchIDs)
		if kvErr != nil {
			slog.WarnContext(ctx, "LoadImpactEvents: kv pairs fallback indisponible (best-effort)",
				"err", kvErr, "n_matches", len(matchIDs))
			return result, nil
		}
		if synth := synthesizeImpactRowsFromKVPairs(kvPairs); len(synth) > 0 {
			result = mergeImpactRowsByTime(result, synth)
		}
	}
	return result, nil
}

// LoadMainTeamParticipants charge tous les participants de l'équipe alliée
// du joueur principal pour une liste de matchs (Q32b). Pour chaque match,
// retourne les rows match_participants où team_id = team_id du mainXUID dans
// ce match (le main inclus).
