// Package sync — transforms.go : transformation JSON API Halo → structs DB.
//
// Portage de transformers/_match.py + transformers/_medals.py (Python).
// Toutes les fonctions sont pures (stateless, sans accès DB).
//
// Structures produites :
//
//	MatchRegistryRow  → match_registry (shared)
//	ParticipantRow    → match_participants (shared)
//	MedalRow          → medals_earned (shared)
//
// Helpers privés (parseurs, regex, accésseurs primitifs) → transforms_helpers.go
package sync

import (
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
)

// ──────────────────────────────────────────────────────────────────────────────
// Structs de sortie
// ──────────────────────────────────────────────────────────────────────────────

// MatchRegistryRow — alias vers domain.MatchRegistryRow.
// La définition canonique vit dans internal/domain/match_rows.go (déplacé
// 2026-05-23 pour casser le cycle d'import sync ⇄ persist).
type MatchRegistryRow = domain.MatchRegistryRow

// ParticipantRow — alias vers domain.MatchParticipantRow (la row COMPLÈTE).
// À ne pas confondre avec domain.ParticipantRow (minimal, 5 champs, pour analysis).
type ParticipantRow = domain.MatchParticipantRow

// MedalRow — alias vers domain.MedalRow.
type MedalRow = domain.MedalRow

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
		MatchID:      matchID,
		StartTime:    startTime,
		ModeCategory: modeCategoryOther,
		FirstSyncBy:  syncBy,
	}

	// Assets
	row.PlaylistID = strPtrNonEmpty(extractAssetID(matchInfo, "Playlist"))
	row.PlaylistName = strPtrNonEmpty(extractPublicName(matchInfo, "Playlist"))
	row.PlaylistVersionID = strPtrNonEmpty(extractVersionID(matchInfo, "Playlist"))
	row.MapID = strPtrNonEmpty(extractAssetID(matchInfo, "MapVariant"))
	row.MapName = strPtrNonEmpty(extractPublicName(matchInfo, "MapVariant"))
	row.MapVersionID = strPtrNonEmpty(extractVersionID(matchInfo, "MapVariant"))
	row.PairID = strPtrNonEmpty(extractAssetID(matchInfo, "PlaylistMapModePair"))
	row.PairName = strPtrNonEmpty(extractPublicName(matchInfo, "PlaylistMapModePair"))
	row.PairVersionID = strPtrNonEmpty(extractVersionID(matchInfo, "PlaylistMapModePair"))
	row.GameVariantID = strPtrNonEmpty(extractAssetID(matchInfo, "UgcGameVariant"))
	row.GameVariantName = strPtrNonEmpty(extractPublicName(matchInfo, "UgcGameVariant"))
	row.GameVariantVersionID = strPtrNonEmpty(extractVersionID(matchInfo, "UgcGameVariant"))

	// Fallback nom → ID
	row.PlaylistName = coalesceStrPtr(row.PlaylistName, row.PlaylistID)
	row.MapName = coalesceStrPtr(row.MapName, row.MapID)
	row.PairName = coalesceStrPtr(row.PairName, row.PairID)
	row.GameVariantName = coalesceStrPtr(row.GameVariantName, row.GameVariantID)

	// Flags
	row.IsRanked = isRankedPlaylist(matchInfo)
	row.IsFirefight = isFirefightMatch(matchInfo)

	// SeasonID lu depuis le payload Halo (matchInfo["SeasonId"]). Pivote vers le
	// catalogue local (csr_placement_thresholds) pour calculer le bon "(X/N)" en
	// display selon la saison du match. Si absent du payload (anciennes saisons
	// ou drift API), reste nil — la migration backfill le populera via dérivation
	// depuis start_time.
	if sid, _ := matchInfo["SeasonId"].(string); sid != "" {
		row.SeasonID = strPtrNonEmpty(sid)
	}
	if row.PairName != nil {
		row.ModeCategory = determineModeCategory(*row.PairName)
	}

	// Durées
	row.DurationSeconds = parsePTDuration(asString(matchInfo["Duration"]))
	row.PlayableDurationSeconds = parsePTDuration(asString(matchInfo["PlayableDuration"]))

	// end_time
	if et, err2 := parseISO(asString(matchInfo["EndTime"])); err2 == nil {
		row.EndTime = &et
	} else if row.DurationSeconds != nil {
		t := startTime.Add(time.Duration(*row.DurationSeconds) * time.Second)
		row.EndTime = &t
	}

	// real_start_time = début réel du gameplay = start_time + T0 (countdown
	// pré-match), calculé depuis les first_joined_time des joueurs présents au
	// début (analysis/timeline.ComputeT0). Remplace l'ancien calcul via
	// playable_duration (non fiable Ranked Arena où playable==duration → T0=0
	// à tort). T0 non exploitable → RealStartTime nil (fallback runtime T0=0).
	// ART-safe : valeur injectée dans le row AVANT persistence, pas d'UPDATE.
	if t0ms, q := computeMatchT0(matchJSON, startTime); q.Computed() {
		rst := startTime.Add(time.Duration(t0ms) * time.Millisecond)
		row.RealStartTime = &rst
	}

	// Team scores (depuis Teams[].Stats.CoreStats.Score)
	t0, t1 := ExtractTeamScoresByID(matchJSON)
	row.Team0Score = t0
	row.Team1Score = t1

	// Team PersonalScore aggregates (somme par équipe sur Players[].PersonalScore).
	// L'API ne fournit pas d'agrégat — on le calcule depuis les participants.
	ps0, ps1 := extractTeamPSScores(matchJSON)
	row.Team0PSScore = ps0
	row.Team1PSScore = ps1

	return row, nil
}

// computeMatchT0 calcule le T0 (countdown pré-match, ms) d'un match depuis les
// first_joined_time des joueurs présents au début, via timeline.ComputeT0.
// Réutilise l'extraction xuid/participation du même JSON que ExtractParticipants.
func computeMatchT0(matchJSON map[string]any, startUTC time.Time) (int64, timeline.T0Quality) {
	players, _ := matchJSON["Players"].([]any)
	inputs := make([]timeline.ParticipationT0Input, 0, len(players))
	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		pinfo, ok := player["ParticipationInfo"].(map[string]any)
		if !ok {
			continue
		}
		fjt, err := parseISO(asString(pinfo["FirstJoinedTime"]))
		if err != nil {
			continue
		}
		present := false
		if b := jsonBoolPtr(pinfo, "PresentAtBeginning"); b != nil {
			present = *b
		}
		xuid := extractXUID(asString(player[jsonKeyPlayerID]))
		inputs = append(inputs, timeline.ParticipationT0Input{
			FirstJoinedTime:    fjt,
			PresentAtBeginning: present,
			IsBot:              !isNumericXUID(xuid),
		})
	}
	return timeline.ComputeT0(inputs, startUTC)
}

// computeMatchWindow résout l'horloge de gameplay d'un match depuis son JSON
// sous forme de domain.MatchTimeline (source unique du vrai début/fin/durée :
// GameplayStartUTC / GameplayEndUTC / GameplayDurationSeconds).
//
// Sert au recompute time_played_seconds par joueur (cf. ComputeTimePlayedFor).
// ok=false si StartTime/Duration manquants → l'appelant conserve la valeur API.
func computeMatchWindow(matchJSON map[string]any) (domain.MatchTimeline, bool) {
	matchInfo, isMap := matchJSON["MatchInfo"].(map[string]any)
	if !isMap {
		return domain.MatchTimeline{}, false
	}
	startTime, err := parseISO(asString(matchInfo["StartTime"]))
	if err != nil {
		return domain.MatchTimeline{}, false
	}
	dur := parsePTDuration(asString(matchInfo["Duration"]))
	if dur == nil || *dur <= 0 {
		return domain.MatchTimeline{}, false
	}
	startUTC := startTime.UTC()
	var t0ms int64
	if v, q := computeMatchT0(matchJSON, startUTC); q.Computed() {
		t0ms = v
	}
	return domain.NewMatchTimelineAt(startUTC, int64(*dur)*1000, t0ms), true
}

// isNumericXUID retourne true si l'xuid est un identifiant joueur numérique
// (bots Halo : xuid non-numérique ou vide).
func isNumericXUID(xuid string) bool {
	if xuid == "" {
		return false
	}
	for _, c := range xuid {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// extractTeamPSScores somme PersonalScore (CoreStats) par team_id sur tous
// les Players[]. Retourne (team0_total, team1_total) ou (nil, nil) si Players
// est vide ou si aucun PersonalScore n'a été trouvé.
func extractTeamPSScores(matchJSON map[string]any) (*int, *int) {
	players, _ := matchJSON["Players"].([]any)
	if len(players) == 0 {
		return nil, nil
	}
	var t0, t1 int
	var has0, has1 bool
	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		teamID, _ := player["LastTeamId"].(float64)
		core := findCoreStats(player)
		if core == nil {
			continue
		}
		ps := intFrom(core, "PersonalScore")
		switch int(teamID) {
		case 0:
			t0 += ps
			has0 = true
		case 1:
			t1 += ps
			has1 = true
		}
	}
	var p0, p1 *int
	if has0 {
		v := t0
		p0 = &v
	}
	if has1 {
		v := t1
		p1 = &v
	}
	return p0, p1
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
	var rows []ParticipantRow //nolint:prealloc
	seen := map[string]bool{}

	// Horloge de gameplay (countdown retranché) pour recomputer time_played par
	// joueur. winOK=false (StartTime/Duration manquants) → on garde la valeur API.
	matchTL, winOK := computeMatchWindow(matchJSON)

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player[jsonKeyPlayerID]))
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
			row.Score = intPtrFrom(core, "PersonalScore")
			row.Kills = intPtrFrom(core, "Kills")
			row.Deaths = intPtrFrom(core, "Deaths")
			row.Assists = intPtrFrom(core, "Assists")
			row.ShotsFired = intPtrFrom(core, "ShotsFired")
			row.ShotsHit = intPtrFrom(core, "ShotsHit")
			row.DamageDealt = floatPtrFrom(core, "DamageDealt")
			row.DamageTaken = floatPtrFrom(core, "DamageTaken")
			row.PersonalScore = intPtrFrom(core, "PersonalScore")
			// AverageLifeDuration est une string ISO-8601 "PT30S", pas un float.
			// Bug pré-existant : floatPtrFrom retournait toujours nil.
			if dur := parsePTDuration(asString(core["AverageLifeDuration"])); dur != nil {
				v := float64(*dur)
				row.AvgLifeSeconds = &v
			}
			row.HeadshotKills = intPtrFrom(core, "HeadshotKills")

			// MaxKillingSpree : pic de spree, présent dans CoreStats.
			row.MaxKillingSpree = intPtrFrom(core, "MaxKillingSpree")

			// Kills par type d'arme : grenade/melee/power_weapon (présents dans CoreStats).
			row.GrenadeKills = intPtrFrom(core, "GrenadeKills")
			row.MeleeKills = intPtrFrom(core, "MeleeKills")
			row.PowerWeaponKills = intPtrFrom(core, "PowerWeaponKills")

			// KDA + Accuracy : valeurs natives du jeu, lues telles quelles depuis
			// l'API (CoreStats). Halo expose le KDA officiel (Kills + Assists/3 −
			// Deaths, peut être négatif) et l'Accuracy (pourcentage 0..100). On ne
			// recalcule PAS : soit la donnée vient de l'API, soit elle est absente
			// (nil → "-" côté UI). Le ratio (kills+assists)/deaths reste une
			// métrique INTERNE du performance_score (analysis.KDA), distincte du
			// FDA affiché — cf. performance.go.
			row.KDA = floatPtrFrom(core, "KDA")
			row.Accuracy = floatPtrFrom(core, "Accuracy")
		}

		// Gamertag
		gt := asString(player["Gamertag"])
		if gt == "" {
			gt = asString(player["PlayerName"])
		}
		if gt != "" {
			row.Gamertag = &gt
		}

		// ParticipationInfo : TimePlayed + 4 booleans + 2 timestamps (LUSR v2).
		// L'objet est imbriqué dans le JSON (pas une clé plate).
		if pinfo, ok := player["ParticipationInfo"].(map[string]any); ok {
			if dur := parsePTDuration(asString(pinfo["TimePlayed"])); dur != nil {
				row.TimePlayedSeconds = dur
			}
			row.PresentAtBeginning = jsonBoolPtr(pinfo, "PresentAtBeginning")
			row.PresentAtCompletion = jsonBoolPtr(pinfo, "PresentAtCompletion")
			row.JoinedInProgress = jsonBoolPtr(pinfo, "JoinedInProgress")
			row.LeftInProgress = jsonBoolPtr(pinfo, "LeftInProgress")
			if ts, err := parseISO(asString(pinfo["FirstJoinedTime"])); err == nil {
				row.FirstJoinedTime = &ts
			}
			if ts, err := parseISO(asString(pinfo["LastLeaveTime"])); err == nil {
				row.LastLeaveTime = &ts
			}
		}

		// Recompute time_played_seconds = temps de jeu réel (countdown retranché,
		// quitters/latecomers gérés). L'API TimePlayed inclut le countdown et
		// surévalue les latecomers (cf. Phase 4). Override seulement si la fenêtre
		// est connue et le calcul exploitable (qualité ok) ; sinon valeur API.
		if winOK && row.FirstJoinedTime != nil {
			if secs, q := timeline.ComputeTimePlayedFor(
				timeline.TimePlayedInput{FirstJoinedTime: row.FirstJoinedTime.UTC(), LastLeaveTime: row.LastLeaveTime},
				matchTL,
			); q == timeline.TimePlayedOK {
				v := int(secs)
				row.TimePlayedSeconds = &v
			}
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
	var rows []MedalRow //nolint:prealloc

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player[jsonKeyPlayerID]))
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
			count := intFrom(medal, "Count")
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
