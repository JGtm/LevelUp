// Package duckdb â€” squad_repo.go : accÃ¨s DB pour la page Escouade et SynthÃ¨se.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
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
// TODO follow-up post-sprint B1 : split+merge cross-DB. Q29TopTeammates joint
// player_match_enrichment ⨝ shared.match_participants (x2) ⨝ v_gamertag_lookup.
// Tant que attachShared reste en place dans le pool, la query reste sur
// pdb.ReadDB() (player conn avec ATTACH).
func (r *SquadRepo) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q29TopTeammates, xuid, xuid)
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

	// shared.xuid_aliases : (xuid PK, gamertag, last_seen, source, updated_at)
	// Source canonique pour ce titre — peuplée par le sync engine. La DB globale
	// est obsolète (migration one-shot souvent vide).
	const q = `
SELECT xuid
FROM shared.xuid_aliases
WHERE gamertag ILIKE ?
ORDER BY last_seen DESC NULLS LAST
LIMIT 1`

	rows, err := r.pdb.ReadDB().Query(ctx, q, gamertag)
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
// TODO follow-up post-sprint B1 : split+merge cross-DB (Q30SquadMatches : shared.match_participants
// + v_match_full + medals_earned subquery + LEFT JOIN player_match_enrichment).
// Reste sur pdb.ReadDB() tant que attachShared est en place.
func (r *SquadRepo) LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q30SquadMatches, teammateXUID, playerXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
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
			&row.TeamMMR,
			&row.SessionID,
			&row.SessionLabel,
			&row.PerformanceScore,
			&row.IsWithFriends,
			&row.HeadshotKills,
			&row.PerfectKills,
			&row.EnemyMMR,
			&row.MyTeamScore,
			&row.EnemyTeamScore,
			&row.MapID,
			&row.PlaylistID,
		); err != nil {
			return nil, fmt.Errorf("LoadSquadMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadTeammateMatches charge les stats du coÃ©quipier sur les matchs communs (Q31).
//
// TODO follow-up post-sprint B1 : split+merge cross-DB (Q31TeammateMatches : shared.match_participants
// x2 + v_match_full). Strictement shared mais le filtre p_main JOIN crée une
// dépendance multi-row par match — peut être migré directement vers SharedReader
// au prochain commit. Reste sur pdb.ReadDB() pour stabilité court terme.
func (r *SquadRepo) LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q31TeammateMatches, playerXUID, teammateXUID)
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

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
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
	return result, rows.Err()
}

// LoadMainTeamParticipants charge tous les participants de l'équipe alliée
// du joueur principal pour une liste de matchs (Q32b). Pour chaque match,
// retourne les rows match_participants où team_id = team_id du mainXUID dans
// ce match (le main inclus).
func (r *SquadRepo) LoadMainTeamParticipants(ctx context.Context, mainXUID string, matchIDs []string) ([]domain.AllyParticipant, error) {
	if len(matchIDs) == 0 || mainXUID == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, 1+len(matchIDs))
	args = append(args, mainXUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q32bMainTeamParticipantsTemplate, strings.Join(placeholders, ","))

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: %w", err)
	}
	defer rows.Close()

	var result []domain.AllyParticipant
	for rows.Next() {
		var row domain.AllyParticipant
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Outcome,
		); err != nil {
			return nil, fmt.Errorf("LoadMainTeamParticipants scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisHeatmap charge les donnÃ©es heatmap mapÃ—mode (Q33).
func (r *SquadRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Sprint B1 commit 8k.13 : shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q33SynthesisHeatmap, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: %w", err)
	}
	defer rows.Close()

	var result []domain.SynthesisHeatmapRow
	for rows.Next() {
		var row domain.SynthesisHeatmapRow
		if err := rows.Scan(
			&row.MapName,
			&row.ModeName,
			&row.MatchCount,
			&row.Wins,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisHeatmap scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisMatches charge les matchs du joueur pour le calcul top_weeks (Q33b).
//
// TODO follow-up post-sprint B1 : split+merge cross-DB (Q33bSynthesisMatches :
// shared.match_participants + shared.match_registry + LEFT JOIN
// player_match_enrichment). Reste sur pdb.ReadDB().
func (r *SquadRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q33bSynthesisMatches, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	defer rows.Close()

	var result []legacymatch.SynthesisMatchRow
	for rows.Next() {
		var row legacymatch.SynthesisMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.KDA,
			&row.IsWithFriends,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.PerformanceScore,
			&row.SessionLabel,
			&row.IsRanked,
			&row.IsFirefight,
			&row.PlaylistName,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMapStatsForSquad calcule par carte (map_id) le winrate et la performance
// moyenne du joueur principal sur les matchs où TOUS les xuids du squad sont
// participants. Aucun filtre temporel — c'est l'historique complet "avec cette
// escouade exacte".
//
// TODO follow-up post-sprint B1 : split+merge cross-DB (Q42MapStatsForSquadTemplate :
// shared.match_participants x2 + shared.match_registry + LEFT JOIN
// player_match_enrichment pour perf_avg). Reste sur pdb.ReadDB().
//
// Comportement :
//   - squadXUIDs vide : retourne nil, nil (pas de squad sélectionné).
//   - squadXUIDs ne contenant que mainXUID : tombe sur les stats solo du main
//     (cas dégénéré utile pour le mode solo).
//   - mainXUID inclus dans squadXUIDs : pas de doublonnage côté SQL grâce au
//     COUNT(DISTINCT xuid) dans squad_matches.
//
// Retour : map keyée sur map_id (jamais vide ; clé absente = aucun match avec
// ce squad sur cette carte).
func (r *SquadRepo) LoadMapStatsForSquad(ctx context.Context, mainXUID string, squadXUIDs []string) (map[string]domain.MapSquadStats, error) {
	if mainXUID == "" || len(squadXUIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	placeholders := strings.TrimRight(strings.Repeat("?,", len(squadXUIDs)), ",")
	q := fmt.Sprintf(Q42MapStatsForSquadTemplate, placeholders, len(uniqueXUIDs(squadXUIDs)))
	args := make([]any, 0, len(squadXUIDs)+1)
	for _, x := range squadXUIDs {
		args = append(args, x)
	}
	args = append(args, mainXUID)

	rows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMapStatsForSquad: %w", err)
	}
	defer rows.Close()

	result := make(map[string]domain.MapSquadStats)
	for rows.Next() {
		var (
			mapID       string
			total, wins int
			perf        sql.NullFloat64
		)
		if err := rows.Scan(&mapID, &total, &wins, &perf); err != nil {
			return nil, fmt.Errorf("LoadMapStatsForSquad scan: %w", err)
		}
		if mapID == "" {
			continue
		}
		s := domain.MapSquadStats{Wins: wins, Total: total}
		if perf.Valid {
			v := perf.Float64
			s.PerfAvg = &v
		}
		result[mapID] = s
	}
	return result, rows.Err()
}

// uniqueXUIDs déduplique sans modifier l'ordre — utile uniquement pour le
// HAVING COUNT(DISTINCT) du SQL squad_matches (la cardinalité ne doit pas
// compter les doublons applicatifs).
func uniqueXUIDs(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := xs[:0:0]
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// LoadAssetTranslationsFR retourne les traductions FR depuis metadata.asset_translations.
// Wrapper mince autour du résolveur unifié `MetadataRepo.ResolveAssetNamesBulk`
// (cf. metadata_repo_assets.go). Cohérent avec home_repo.resolveAssetNames.
// assetType : "map" | "playlist" | "game_variant" | "pair".
func (r *SquadRepo) LoadAssetTranslationsFR(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error) {
	if len(assetIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	out, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(
		ctx, assetType, assetIDs, PreferredLangsForLocale("fr"),
	)
	if err != nil && isTableNotFoundErr(err) {
		return nil, nil
	}
	return out, err
}

// LoadModeTranslationsFR retourne les traductions FR depuis metadata.mode_name_tr.
// Calqué sur homeRepo.loadHomeModeNameTranslations — même table, même logique.
func (r *SquadRepo) LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error) {
	if len(modeENs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENs)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(modeENs))
	for i, n := range modeENs {
		args[i] = n
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadModeTranslationsFR: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string, len(modeENs))
	for rows.Next() {
		var en, fr string
		if err := rows.Scan(&en, &fr); err != nil {
			continue
		}
		if strings.TrimSpace(fr) != "" {
			result[en] = fr
		}
	}
	return result, rows.Err()
}

// Ensure SquadRepo implements port.SquadRepository at compile time.
// (Vérification implicite via injection dans le service.)
