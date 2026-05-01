// Package duckdb — player_matches_repo.go : implementation DuckDB du loader
// unifie des matchs joueur (port.PlayerMatchesRepository).
//
// Per-player : un PlayerMatchesRepo est lie a un PlayerDB precis. La resolution
// (slug, gamertag) -> PlayerDB se fait via pool.GetOrOpen au niveau de
// l'adapter qui consomme le repo (chunk ulterieur).
//
// Capability gating : laisse au service appelant pour cette implementation.
// Le repo execute la requete telle quelle ; si le titre n'a pas la capability
// "match.history", c'est au service de retourner games.ErrCapabilityNotSupported
// avant d'appeler le repo.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// PlayerMatchesRepo charge les matchs d'un joueur depuis sa DB stats avec
// shared attache. Implemente une variante per-player de
// port.PlayerMatchesRepository (le slug et le gamertag sont fixes par le
// PlayerDB injecte au constructeur).
type PlayerMatchesRepo struct {
	pdb *PlayerDB
}

// NewPlayerMatchesRepo cree un PlayerMatchesRepo lie a un PlayerDB.
func NewPlayerMatchesRepo(pdb *PlayerDB) *PlayerMatchesRepo {
	return &PlayerMatchesRepo{pdb: pdb}
}

// Load charge les matchs du joueur en suivant les filtres fournis. Retourne
// les rows projetees en canonical.PlayerMatchRow. Trie par r.start_time DESC
// par defaut, override possible via filters.OrderBy (whitelist).
//
// L'appelant doit avoir valide les filtres via filters.Validate() en amont.
// Le repo re-applique aussi sa propre validation defensive (input untrusted).
func (r *PlayerMatchesRepo) Load(
	ctx context.Context,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: %w", err)
	}

	q, args, err := r.buildQuery(filters)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: build query: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: query: %w", err)
	}
	defer rows.Close()

	var out []canonical.PlayerMatchRow
	for rows.Next() {
		row, err := scanPlayerMatchRow(rows, r.pdb.XUID, r.pdb.Gamertag)
		if err != nil {
			return nil, fmt.Errorf("PlayerMatchesRepo.Load: scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PlayerMatchesRepo.Load: rows: %w", err)
	}
	return out, nil
}

// buildQuery compose le SELECT et les WHERE dynamiques selon les filtres.
// Les valeurs scalaires sont passees via placeholders ?, jamais interpolees.
// Seuls les fragments structurels (ORDER BY whitelist, IN (..., ?)) sont
// composes en string.
func (r *PlayerMatchesRepo) buildQuery(f port.PlayerMatchFilters) (string, []any, error) {
	var sb strings.Builder
	sb.WriteString(playerMatchesBaseSelect)

	args := []any{r.pdb.XUID}

	if since := periodSince(f.Period); since != nil {
		sb.WriteString(" AND r.start_time >= ?")
		args = append(args, *since)
	}
	if len(f.OutcomeIn) > 0 {
		placeholders := make([]string, 0, len(f.OutcomeIn))
		for _, o := range f.OutcomeIn {
			placeholders = append(placeholders, "?")
			args = append(args, outcomeToInt(o))
		}
		sb.WriteString(fmt.Sprintf(" AND COALESCE(p.outcome, 0) IN (%s)",
			strings.Join(placeholders, ",")))
	}
	if f.HadBotTeammate != nil {
		sb.WriteString(" AND COALESCE(pme.had_bot_teammate, FALSE) = ?")
		args = append(args, *f.HadBotTeammate)
	}
	if f.IsFirefight != nil {
		sb.WriteString(" AND COALESCE(r.is_firefight, FALSE) = ?")
		args = append(args, *f.IsFirefight)
	}
	if f.IsRanked != nil {
		sb.WriteString(" AND COALESCE(r.is_ranked, FALSE) = ?")
		args = append(args, *f.IsRanked)
	}
	if f.MinTimePlayedSeconds != nil {
		sb.WriteString(" AND COALESCE(p.time_played_seconds, 0) >= ?")
		args = append(args, *f.MinTimePlayedSeconds)
	}
	if len(f.ExcludeFriendsXUIDs) > 0 {
		placeholders := make([]string, 0, len(f.ExcludeFriendsXUIDs))
		for _, x := range f.ExcludeFriendsXUIDs {
			placeholders = append(placeholders, "?")
			args = append(args, x)
		}
		sb.WriteString(fmt.Sprintf(
			" AND p.match_id NOT IN (SELECT match_id FROM shared.match_participants WHERE xuid IN (%s))",
			strings.Join(placeholders, ",")))
	}
	if f.BTBExcluded {
		sb.WriteString(" AND (r.pair_name IS NULL OR LOWER(r.pair_name) NOT LIKE '%btb%')")
	}
	if f.PlaylistKind != nil {
		clause, err := playlistKindClause(*f.PlaylistKind)
		if err != nil {
			return "", nil, err
		}
		if clause != "" {
			sb.WriteString(" AND ")
			sb.WriteString(clause)
		}
	}
	if len(f.MapIDs) > 0 {
		placeholders := make([]string, 0, len(f.MapIDs))
		for _, id := range f.MapIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		sb.WriteString(fmt.Sprintf(" AND COALESCE(r.map_id, '') IN (%s)",
			strings.Join(placeholders, ",")))
	}

	orderBy, err := orderByClause(f.OrderBy)
	if err != nil {
		return "", nil, err
	}
	sb.WriteString(" ORDER BY ")
	sb.WriteString(orderBy)

	if f.Limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, f.Limit)
	}

	return sb.String(), args, nil
}

// playerMatchesBaseSelect est la partie fixe du SELECT (colonnes + JOINs +
// WHERE p.xuid = ?). Les filtres additionnels sont concatenes en AND.
// Bug #2/#7 : on ne fallback PAS sur l'EN dans la projection FR. Si NULL en
// DB, on renvoie chaîne vide ; HomeRepo.EnrichCanonicalAssetTranslations
// remplit ensuite Labels["fr"] depuis metadata.asset_translations.
//
// Bug #3 : projeter damage_dealt / damage_taken pour ComputeCombatYield.
const playerMatchesBaseSelect = `
SELECT
    p.match_id,
    r.start_time,
    COALESCE(r.duration_seconds, 0)                   AS duration_seconds,
    COALESCE(r.map_id, '')                            AS map_id,
    COALESCE(r.map_name, '')                          AS map_name,
    COALESCE(r.map_name_fr, '')                       AS map_name_fr,
    COALESCE(r.playlist_id, '')                       AS playlist_id,
    COALESCE(r.playlist_name, '')                     AS playlist_name,
    COALESCE(r.playlist_name_fr, '')                  AS playlist_name_fr,
    COALESCE(r.game_variant_id, '')                   AS variant_id,
    COALESCE(r.game_variant_name, '')                 AS variant_name,
    COALESCE(r.pair_id, '')                           AS pair_id,
    COALESCE(r.pair_name, '')                         AS pair_name,
    COALESCE(r.pair_name_fr, '')                      AS pair_name_fr,
    COALESCE(r.is_ranked, FALSE)                      AS is_ranked,
    COALESCE(r.is_firefight, FALSE)                   AS is_firefight,
    COALESCE(p.team_id, 0)                            AS team_id,
    p.outcome                                         AS outcome_code,
    COALESCE(p.kills, 0)                              AS kills,
    COALESCE(p.deaths, 0)                             AS deaths,
    COALESCE(p.assists, 0)                            AS assists,
    p.kda,
    COALESCE(p.headshot_kills, 0)                     AS headshot_kills,
    p.accuracy,
    COALESCE(p.time_played_seconds, 0)                AS time_played_seconds,
    p.damage_dealt,
    p.damage_taken,
    p.team_mmr,
    p.enemy_mmr,
    pme.session_id,
    pme.session_label,
    pme.performance_score,
    COALESCE(pme.dominance_flag, 0)                   AS dominance_flag,
    COALESCE(pme.had_bot_teammate, FALSE)             AS had_bot_teammate,
    COALESCE(pme.is_with_friends, FALSE)              AS is_with_friends
FROM shared.match_participants p
JOIN shared.v_match_full r ON r.match_id = p.match_id
LEFT JOIN player_match_enrichment pme ON pme.match_id = p.match_id
WHERE p.xuid = ?`

// scanPlayerMatchRow scanne une row SQL en canonical.PlayerMatchRow. Les
// colonnes nullable utilisent sql.Null* puis sont converties en *T.
func scanPlayerMatchRow(rows *sql.Rows, xuid, gamertag string) (canonical.PlayerMatchRow, error) {
	var (
		matchID, mapID, mapName, mapNameFR                 string
		playlistID, playlistName, playlistNameFR           string
		variantID, variantName                             string
		pairID, pairName, pairNameFR                       string
		startTime                                          time.Time
		durationSeconds, teamID                            int
		outcomeCode                                        sql.NullInt64
		kills, deaths, assists, headshotKills              int
		timePlayedSeconds                                  int
		dominanceFlag                                      int
		isRanked, isFirefight                              bool
		hadBotTeammate, isWithFriends                      bool
		kda, accuracy, teamMMR, enemyMMR, performanceScore sql.NullFloat64
		damageDealt, damageTaken                           sql.NullFloat64
		sessionID                                          sql.NullInt64
		sessionLabel                                       sql.NullString
	)
	if err := rows.Scan(
		&matchID, &startTime, &durationSeconds,
		&mapID, &mapName, &mapNameFR,
		&playlistID, &playlistName, &playlistNameFR,
		&variantID, &variantName,
		&pairID, &pairName, &pairNameFR,
		&isRanked, &isFirefight,
		&teamID, &outcomeCode,
		&kills, &deaths, &assists,
		&kda, &headshotKills, &accuracy,
		&timePlayedSeconds, &damageDealt, &damageTaken,
		&teamMMR, &enemyMMR,
		&sessionID, &sessionLabel, &performanceScore,
		&dominanceFlag, &hadBotTeammate, &isWithFriends,
	); err != nil {
		return canonical.PlayerMatchRow{}, err
	}
	return projectPlayerMatchRow(playerMatchScanResult{
		matchID:           matchID,
		startTime:         startTime,
		durationSeconds:   durationSeconds,
		mapID:             mapID,
		mapName:           mapName,
		mapNameFR:         mapNameFR,
		playlistID:        playlistID,
		playlistName:      playlistName,
		playlistNameFR:    playlistNameFR,
		variantID:         variantID,
		variantName:       variantName,
		pairID:            pairID,
		pairName:          pairName,
		pairNameFR:        pairNameFR,
		isRanked:          isRanked,
		isFirefight:       isFirefight,
		teamID:            teamID,
		outcomeCode:       outcomeCode,
		kills:             kills,
		deaths:            deaths,
		assists:           assists,
		headshotKills:     headshotKills,
		timePlayedSeconds: timePlayedSeconds,
		dominanceFlag:     dominanceFlag,
		hadBotTeammate:    hadBotTeammate,
		isWithFriends:     isWithFriends,
		kda:               kda,
		accuracy:          accuracy,
		damageDealt:       damageDealt,
		damageTaken:       damageTaken,
		teamMMR:           teamMMR,
		enemyMMR:          enemyMMR,
		performanceScore:  performanceScore,
		sessionID:         sessionID,
		sessionLabel:      sessionLabel,
		xuid:              xuid,
		gamertag:          gamertag,
	}), nil
}

// playerMatchScanResult agrege les valeurs scannees pour faciliter la
// projection en canonical.PlayerMatchRow.
type playerMatchScanResult struct {
	matchID, mapID, mapName, mapNameFR       string
	playlistID, playlistName, playlistNameFR string
	variantID, variantName, xuid, gamertag   string
	pairID, pairName, pairNameFR             string
	startTime                                time.Time
	durationSeconds, teamID                  int
	outcomeCode                              sql.NullInt64
	kills, deaths, assists, headshotKills    int
	timePlayedSeconds, dominanceFlag         int
	isRanked, isFirefight, hadBotTeammate    bool
	isWithFriends                            bool
	kda, accuracy, teamMMR, enemyMMR         sql.NullFloat64
	damageDealt, damageTaken                 sql.NullFloat64
	performanceScore                         sql.NullFloat64
	sessionID                                sql.NullInt64
	sessionLabel                             sql.NullString
}

// projectPlayerMatchRow construit la row canonique depuis les valeurs scannees.
func projectPlayerMatchRow(s playerMatchScanResult) canonical.PlayerMatchRow {
	teamIDPtr := s.teamID
	durationPtr := s.durationSeconds
	killsPtr, deathsPtr, assistsPtr := s.kills, s.deaths, s.assists
	headshotPtr := s.headshotKills
	timePlayedPtr := s.timePlayedSeconds

	// Bug #5 : si outcome NULL/0 en DB, l'Outcome canonical reste vide.
	var outcome canonical.Outcome
	if s.outcomeCode.Valid && s.outcomeCode.Int64 != 0 {
		outcome = outcomeFromInt(int(s.outcomeCode.Int64))
	}

	// Bug #3 : damage_dealt/damage_taken sont DOUBLE en DB.
	var dmgDealt, dmgTaken *int
	if s.damageDealt.Valid {
		v := int(s.damageDealt.Float64)
		dmgDealt = &v
	}
	if s.damageTaken.Valid {
		v := int(s.damageTaken.Float64)
		dmgTaken = &v
	}

	row := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:         s.matchID,
			StartedAtUTC:    s.startTime,
			DurationSeconds: &durationPtr,
			MatchType:       matchTypeFromFlags(s.isRanked, s.isFirefight),
			Playlist:        assetReference("playlist", s.playlistID, s.playlistName, s.playlistNameFR),
			Map:             assetReference("map", s.mapID, s.mapName, s.mapNameFR),
			GameVariant:     assetReference("game_variant", s.variantID, s.variantName, ""),
			PairMode:        assetReference("pair_mode", s.pairID, s.pairName, s.pairNameFR),
			IsRanked:        &s.isRanked,
			IsPvE:           &s.isFirefight,
			Outcome:         outcome,
		},
		Self: canonical.MatchParticipant{
			Identity:      canonical.PlayerIdentity{XUID: s.xuid, Gamertag: s.gamertag},
			TeamID:        &teamIDPtr,
			Outcome:       outcome,
			Kills:         &killsPtr,
			Deaths:        &deathsPtr,
			Assists:       &assistsPtr,
			HeadshotKills: &headshotPtr,
			KDA:           nullFloatPtr(s.kda),
			Accuracy:      nullFloatPtr(s.accuracy),
			TimePlayed:    &timePlayedPtr,
			DamageDealt:   dmgDealt,
			DamageTaken:   dmgTaken,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			SessionID:        nullInt64ToStringPtr(s.sessionID),
			SessionLabel:     nullStringPtr(s.sessionLabel),
			PerformanceScore: nullFloatPtr(s.performanceScore),
			DominanceFlag:    canonical.DominanceFlag(s.dominanceFlag),
			HadBotTeammate:   s.hadBotTeammate,
			IsWithFriends:    s.isWithFriends,
			TeamMMR:          nullFloatPtr(s.teamMMR),
			EnemyMMR:         nullFloatPtr(s.enemyMMR),
		},
	}
	return row
}

// outcomeToInt convertit un canonical.Outcome (string) vers le code int stocke
// en DB (1=tie, 2=win, 3=loss, 4=dnf).
func outcomeToInt(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeTie:
		return 1
	case canonical.OutcomeWin:
		return 2
	case canonical.OutcomeLoss:
		return 3
	case canonical.OutcomeDNF:
		return 4
	}
	return 0
}

// outcomeFromInt convertit le code int DB vers un canonical.Outcome.
func outcomeFromInt(i int) canonical.Outcome {
	switch i {
	case 1:
		return canonical.OutcomeTie
	case 2:
		return canonical.OutcomeWin
	case 3:
		return canonical.OutcomeLoss
	case 4:
		return canonical.OutcomeDNF
	}
	return canonical.Outcome("")
}

// matchTypeFromFlags choisit un MatchType canonique a partir de is_ranked /
// is_firefight (selection prioritaire : firefight > ranked > social).
func matchTypeFromFlags(isRanked, isFirefight bool) canonical.MatchType {
	if isFirefight {
		return canonical.MatchTypeFirefight
	}
	if isRanked {
		return canonical.MatchTypeRanked
	}
	return canonical.MatchTypeSocial
}

// assetReference compose un canonical.AssetReference depuis les colonnes DB.
// Retourne nil si aucun ID ni label.
func assetReference(kind, id, name, nameFR string) *canonical.AssetReference {
	if id == "" && name == "" && nameFR == "" {
		return nil
	}
	ref := &canonical.AssetReference{
		Kind:         kind,
		ID:           id,
		DefaultLabel: name,
	}
	if nameFR != "" || name != "" {
		ref.Labels = map[string]string{}
		if name != "" {
			ref.Labels["en"] = name
		}
		if nameFR != "" {
			ref.Labels["fr"] = nameFR
		}
	}
	return ref
}

// periodSince extrait le timestamp depuis temporal.Period, ou nil si absente.
func periodSince(p *temporal.Period) *time.Time {
	if p == nil {
		return nil
	}
	return p.Since(time.Now())
}

// playlistKindClause traduit l'alias court en clause SQL safe (pas de regex
// libre interpolee). Whitelist fermee, conforme au design § 5.3.5 du meta-plan.
//
// Erreurs : retourne ErrUnknownPlaylistKind si l'alias n'est pas dans la
// whitelist (input untrusted).
func playlistKindClause(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "":
		return "", nil
	case "ranked":
		return "COALESCE(r.is_ranked, FALSE) = TRUE", nil
	case "firefight":
		return "COALESCE(r.is_firefight, FALSE) = TRUE", nil
	case "social":
		return "COALESCE(r.is_ranked, FALSE) = FALSE AND COALESCE(r.is_firefight, FALSE) = FALSE", nil
	case "btb":
		return "LOWER(COALESCE(r.pair_name, '')) LIKE '%btb%'", nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownPlaylistKind, kind)
}

// ErrUnknownPlaylistKind est retournee par buildQuery si PlaylistKind n'est pas
// dans la whitelist des alias supportes.
var ErrUnknownPlaylistKind = errors.New("PlayerMatchesRepo: unknown PlaylistKind")

// orderByClause traduit le filtre OrderBy en expression SQL safe (whitelist).
// Vide -> ordre par defaut (start_time DESC).
func orderByClause(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case "", "start_time DESC":
		return "r.start_time DESC", nil
	case "start_time ASC":
		return "r.start_time ASC", nil
	case "performance_score DESC":
		return "pme.performance_score DESC NULLS LAST", nil
	case "performance_score ASC":
		return "pme.performance_score ASC NULLS LAST", nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownOrderBy, s)
}

// ErrUnknownOrderBy est retournee si OrderBy n'est pas dans la whitelist.
var ErrUnknownOrderBy = errors.New("PlayerMatchesRepo: unknown OrderBy")

// nullFloatPtr convertit sql.NullFloat64 en *float64.
func nullFloatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// nullInt64ToStringPtr convertit sql.NullInt64 en *string (les session_id sont
// representes comme string dans le canonical pour l'instant).
func nullInt64ToStringPtr(n sql.NullInt64) *string {
	if !n.Valid {
		return nil
	}
	s := fmt.Sprintf("%d", n.Int64)
	return &s
}
