// Package sync — transforms.go : transformation JSON API Halo → structs DB.
//
// Portage de transformers/_match.py + transformers/_medals.py (Python).
// Toutes les fonctions sont pures (stateless, sans accès DB).
//
// Structures produites :
//   MatchRegistryRow  → match_registry (shared)
//   ParticipantRow    → match_participants (shared)
//   MedalRow          → medals_earned (shared)
package sync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// Structs de sortie
// ──────────────────────────────────────────────────────────────────────────────

// MatchRegistryRow représente une ligne dans match_registry (shared).
type MatchRegistryRow struct {
	MatchID                string
	StartTime              time.Time
	EndTime                *time.Time
	PlaylistID             *string
	PlaylistName           *string
	MapID                  *string
	MapName                *string
	PairID                 *string
	PairName               *string
	GameVariantID          *string
	GameVariantName        *string
	ModeCategory           string
	IsRanked               bool
	IsFirefight            bool
	DurationSeconds        *int
	PlayableDurationSeconds *int
	RealStartTime          *time.Time
	Team0Score             *int
	Team1Score             *int
	FirstSyncBy            string
}

// ParticipantRow représente une ligne dans match_participants (shared).
type ParticipantRow struct {
	MatchID     string
	XUID        string
	Gamertag    *string
	TeamID      *int
	Outcome     *int
	Rank        *int
	Score       *int
	Kills       *int
	Deaths      *int
	Assists     *int
	ShotsFired  *int
	ShotsHit    *int
	DamageDealt *float64
	DamageTaken *float64
}

// MedalRow représente une ligne dans medals_earned (shared).
type MedalRow struct {
	MatchID     string
	XUID        string
	MedalNameID int64
	Count       int
}

// ──────────────────────────────────────────────────────────────────────────────
// Expressions régulières
// ──────────────────────────────────────────────────────────────────────────────

var (
	xuidRE  = regexp.MustCompile(`xuid\((\d+)\)`)
	ptRE    = regexp.MustCompile(`(?i)PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?`)
)

// ──────────────────────────────────────────────────────────────────────────────
// ExtractRegistry
// ──────────────────────────────────────────────────────────────────────────────

// ExtractRegistry extrait les données de match_registry depuis le JSON brut.
// Portage de extract_match_registry_data() (Python transformers/_match.py).
func ExtractRegistry(matchJSON map[string]any, syncBy string) (*MatchRegistryRow, error) {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil, ErrMissingField("MatchId")
	}

	matchInfo, ok := matchJSON["MatchInfo"].(map[string]any)
	if !ok {
		return nil, ErrMissingField("MatchInfo")
	}

	startTime, err := parseISO(asString(matchInfo["StartTime"]))
	if err != nil {
		return nil, fmt.Errorf("StartTime: %w", err)
	}

	row := &MatchRegistryRow{
		MatchID:     matchID,
		StartTime:   startTime,
		ModeCategory: "Other",
		FirstSyncBy: syncBy,
	}

	// Assets
	row.PlaylistID   = strPtr(extractAssetID(matchInfo, "Playlist"))
	row.PlaylistName = strPtr(extractPublicName(matchInfo, "Playlist"))
	row.MapID        = strPtr(extractAssetID(matchInfo, "MapVariant"))
	row.MapName      = strPtr(extractPublicName(matchInfo, "MapVariant"))
	row.PairID       = strPtr(extractAssetID(matchInfo, "PlaylistMapModePair"))
	row.PairName     = strPtr(extractPublicName(matchInfo, "PlaylistMapModePair"))
	row.GameVariantID   = strPtr(extractAssetID(matchInfo, "UgcGameVariant"))
	row.GameVariantName = strPtr(extractPublicName(matchInfo, "UgcGameVariant"))

	// Fallback nom → ID
	row.PlaylistName  = coalesceStrPtr(row.PlaylistName, row.PlaylistID)
	row.MapName       = coalesceStrPtr(row.MapName, row.MapID)
	row.PairName      = coalesceStrPtr(row.PairName, row.PairID)
	row.GameVariantName = coalesceStrPtr(row.GameVariantName, row.GameVariantID)

	// Flags
	row.IsRanked    = isRankedPlaylist(matchInfo)
	row.IsFirefight = isFirefightMatch(matchInfo)
	if row.PairName != nil {
		row.ModeCategory = determineModeCategory(*row.PairName)
	}

	// Durées
	row.DurationSeconds        = parsePTDuration(asString(matchInfo["Duration"]))
	row.PlayableDurationSeconds = parsePTDuration(asString(matchInfo["PlayableDuration"]))

	// end_time
	if et, err2 := parseISO(asString(matchInfo["EndTime"])); err2 == nil {
		row.EndTime = &et
	} else if row.DurationSeconds != nil {
		t := startTime.Add(time.Duration(*row.DurationSeconds) * time.Second)
		row.EndTime = &t
	}

	// real_start_time
	if row.DurationSeconds != nil && row.PlayableDurationSeconds != nil {
		countdown := *row.DurationSeconds - *row.PlayableDurationSeconds
		if countdown >= 0 {
			rst := startTime.Add(time.Duration(countdown) * time.Second)
			row.RealStartTime = &rst
		}
	}

	// Team scores
	t0, t1 := extractTeamScoresByID(matchJSON)
	row.Team0Score = t0
	row.Team1Score = t1

	return row, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractParticipants
// ──────────────────────────────────────────────────────────────────────────────

// ExtractParticipants extrait tous les participants d'un match.
// Portage de extract_participants() (Python transformers/_match.py).
func ExtractParticipants(matchJSON map[string]any) []ParticipantRow {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil
	}

	players, _ := matchJSON["Players"].([]any)
	var rows []ParticipantRow
	seen := map[string]bool{}

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player["PlayerId"]))
		if xuid == "" || seen[xuid] {
			continue
		}
		seen[xuid] = true

		row := ParticipantRow{
			MatchID: matchID,
			XUID:    xuid,
		}

		// team_id, outcome, rank
		if v, ok := player["LastTeamId"].(float64); ok {
			n := int(v)
			row.TeamID = &n
		}
		if v, ok := player["Outcome"].(float64); ok {
			n := int(v)
			row.Outcome = &n
		}
		if v, ok := player["Rank"].(float64); ok {
			n := int(v)
			row.Rank = &n
		}

		// CoreStats depuis PlayerTeamStats[0]
		core := findCoreStats(player)
		if core != nil {
			row.Score       = intPtrFrom(core, "PersonalScore")
			row.Kills       = intPtrFrom(core, "Kills")
			row.Deaths      = intPtrFrom(core, "Deaths")
			row.Assists     = intPtrFrom(core, "Assists")
			row.ShotsFired  = intPtrFrom(core, "ShotsFired")
			row.ShotsHit    = intPtrFrom(core, "ShotsHit")
			row.DamageDealt = floatPtrFrom(core, "DamageDealt")
			row.DamageTaken = floatPtrFrom(core, "DamageTaken")
		}

		rows = append(rows, row)
	}
	return rows
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractMedals
// ──────────────────────────────────────────────────────────────────────────────

// ExtractMedals extrait les médailles de TOUS les joueurs d'un match.
// Portage de extract_all_medals() (Python transformers/_medals.py).
func ExtractMedals(matchJSON map[string]any) []MedalRow {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil
	}

	players, _ := matchJSON["Players"].([]any)
	var rows []MedalRow

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player["PlayerId"]))
		if xuid == "" {
			continue
		}
		core := findCoreStats(player)
		if core == nil {
			continue
		}
		medals, _ := core["Medals"].([]any)
		for _, m := range medals {
			medal, ok := m.(map[string]any)
			if !ok {
				continue
			}
			nameID := int64From(medal, "NameId")
			count  := intFrom(medal, "Count")
			if nameID == 0 || count == 0 {
				continue
			}
			rows = append(rows, MedalRow{
				MatchID:     matchID,
				XUID:        xuid,
				MedalNameID: nameID,
				Count:       count,
			})
		}
	}
	return rows
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers privés
// ──────────────────────────────────────────────────────────────────────────────

// extractXUID extrait le XUID numérique depuis "xuid(1234567890)".
func extractXUID(playerID string) string {
	if m := xuidRE.FindStringSubmatch(playerID); len(m) == 2 {
		return m[1]
	}
	return ""
}

// extractAssetID extrait AssetId depuis un sous-objet JSON (ex: "Playlist").
func extractAssetID(matchInfo map[string]any, key string) string {
	obj, _ := matchInfo[key].(map[string]any)
	if obj == nil {
		return ""
	}
	id, _ := obj["AssetId"].(string)
	return id
}

// extractPublicName extrait PublicName depuis un sous-objet JSON.
func extractPublicName(matchInfo map[string]any, key string) string {
	obj, _ := matchInfo[key].(map[string]any)
	if obj == nil {
		return ""
	}
	name, _ := obj["PublicName"].(string)
	return name
}

// findCoreStats retourne le dict CoreStats du premier PlayerTeamStats du joueur.
func findCoreStats(player map[string]any) map[string]any {
	pts, _ := player["PlayerTeamStats"].([]any)
	for _, ts := range pts {
		teamStats, ok := ts.(map[string]any)
		if !ok {
			continue
		}
		stats, _ := teamStats["Stats"].(map[string]any)
		if stats == nil {
			continue
		}
		core, _ := stats["CoreStats"].(map[string]any)
		if core != nil {
			return core
		}
	}
	return nil
}

// isRankedPlaylist détermine si le match est un match classé.
// Portage de _is_ranked_playlist() Python.
func isRankedPlaylist(matchInfo map[string]any) bool {
	playlist, _ := matchInfo["Playlist"].(map[string]any)
	if playlist == nil {
		return false
	}
	name, _ := playlist["PublicName"].(string)
	if strings.Contains(strings.ToLower(name), "ranked") {
		return true
	}
	if tags, ok := playlist["Tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok && strings.ToLower(s) == "ranked" {
				return true
			}
		}
	}
	return false
}

// isFirefightMatch détermine si le match est un mode Firefight/PvE.
// Portage de _is_firefight_match() Python.
func isFirefightMatch(matchInfo map[string]any) bool {
	// GameVariantCategory (22 = Firefight Arcade, 32 = Firefight Heroic)
	if cat, ok := matchInfo["GameVariantCategory"].(float64); ok {
		firefightCats := map[int]bool{22: true, 32: true, 40: true, 41: true, 42: true}
		if firefightCats[int(cat)] {
			return true
		}
	}
	if gv, ok := matchInfo["UgcGameVariant"].(map[string]any); ok {
		name, _ := gv["PublicName"].(string)
		if strings.Contains(strings.ToLower(name), "firefight") {
			return true
		}
	}
	return false
}

// determineModeCategory déduit la catégorie custom depuis pair_name.
// Portage simplifié de infer_custom_category_from_pair_name() Python.
func determineModeCategory(pairName string) string {
	lower := strings.ToLower(pairName)
	switch {
	case strings.Contains(lower, "ranked"):
		return "Ranked"
	case strings.Contains(lower, "firefight"):
		return "Firefight"
	case strings.Contains(lower, "btb") || strings.Contains(lower, "big team"):
		return "BTB"
	case strings.Contains(lower, "fiesta"):
		return "Fiesta"
	case strings.Contains(lower, "assassin"):
		return "Assassin"
	default:
		return "Other"
	}
}

// extractTeamScoresByID extrait les scores de team_0 et team_1.
// Portage de _extract_team_scores_by_id() Python.
func extractTeamScoresByID(matchJSON map[string]any) (*int, *int) {
	teams, _ := matchJSON["Teams"].([]any)
	scores := map[int]int{}
	for _, t := range teams {
		team, ok := t.(map[string]any)
		if !ok {
			continue
		}
		id := intFrom(team, "TeamId")
		stats, _ := team["Stats"].(map[string]any)
		if stats == nil {
			continue
		}
		core, _ := stats["CoreStats"].(map[string]any)
		if core == nil {
			continue
		}
		score := intFrom(core, "Score")
		scores[id] = score
	}
	if len(scores) == 0 {
		return nil, nil
	}
	t0, ok0 := scores[0]
	t1, ok1 := scores[1]
	var p0, p1 *int
	if ok0 {
		p0 = &t0
	}
	if ok1 {
		p1 = &t1
	}
	return p0, p1
}

// parsePTDuration convertit une durée ISO 8601 "PT1H2M3.456S" en secondes.
// Portage de _parse_duration_to_seconds() Python.
func parsePTDuration(s string) *int {
	if s == "" {
		return nil
	}
	m := ptRE.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	total := 0
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		total += h * 3600
	}
	if m[2] != "" {
		min, _ := strconv.Atoi(m[2])
		total += min * 60
	}
	if m[3] != "" {
		f, _ := strconv.ParseFloat(m[3], 64)
		total += int(f)
	}
	return &total
}

// parseISO parse une date ISO 8601 UTC.
func parseISO(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parseISO: impossible de parser %q", s)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers primitifs
// ──────────────────────────────────────────────────────────────────────────────

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func coalesceStrPtr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}

func intPtrFrom(m map[string]any, key string) *int {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	n := int(v)
	return &n
}

func floatPtrFrom(m map[string]any, key string) *float64 {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	return &v
}

func intFrom(m map[string]any, key string) int {
	v, _ := m[key].(float64)
	return int(v)
}

func int64From(m map[string]any, key string) int64 {
	v, _ := m[key].(float64)
	return int64(v)
}

// ErrMissingField crée une erreur de champ manquant.
func ErrMissingField(field string) error {
	return fmt.Errorf("champ manquant: %s", field)
}
