package duckdb

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

func projectPlayerMatchRow(s playerMatchScanResult) canonical.PlayerMatchRow {
	outcome := projectOutcome(s)
	teams := projectTeamScores(s)
	skillSnap := projectSkillSnapshot(s)
	dmgDealt, dmgTaken := projectDamageStats(s)

	return canonical.PlayerMatchRow{
		Summary:    projectMatchSummary(s, outcome, teams),
		Self:       projectSelfParticipant(s, outcome, dmgDealt, dmgTaken),
		Enrichment: projectEnrichment(s, skillSnap),
	}
}

// projectOutcome retourne l'Outcome canonique. Outcome vide si NULL/0 en DB.
func projectOutcome(s playerMatchScanResult) canonical.Outcome {
	if s.outcomeCode.Valid && s.outcomeCode.Int64 != 0 {
		return OutcomeFromInt(int(s.outcomeCode.Int64))
	}
	return ""
}

// projectTeamScores assemble les TeamSnapshot depuis team_0_score / team_1_score.
// Une valeur -1 (COALESCE) signifie absent et est exclue.
func projectTeamScores(s playerMatchScanResult) []canonical.TeamSnapshot {
	var teams []canonical.TeamSnapshot
	if s.team0Score >= 0 || s.team0RoundsWon >= 0 {
		teams = append(teams, canonical.TeamSnapshot{
			TeamID: 0, Score: nonNegPtr(s.team0Score), RoundsWon: nonNegPtr(s.team0RoundsWon),
		})
	}
	if s.team1Score >= 0 || s.team1RoundsWon >= 0 {
		teams = append(teams, canonical.TeamSnapshot{
			TeamID: 1, Score: nonNegPtr(s.team1Score), RoundsWon: nonNegPtr(s.team1RoundsWon),
		})
	}
	return teams
}

// nonNegPtr rend un pointeur sur v, ou nil si v porte la sentinelle -1 du COALESCE.
// Un ZÉRO est une mesure (« zéro manche gagnée ») et passe donc bien en pointeur.
func nonNegPtr(v int) *int {
	if v < 0 {
		return nil
	}
	out := v
	return &out
}

// projectSkillSnapshot extrait le SkillSnapshot depuis match_skill_rank.
// Retourne nil si LEFT JOIN absent.
func projectSkillSnapshot(s playerMatchScanResult) *canonical.SkillSnapshot {
	if !s.skillRatingType.Valid || s.skillRatingType.String == "" {
		return nil
	}
	snap := canonical.SkillSnapshot{
		RatingType:      canonical.RatingType(strings.ToLower(s.skillRatingType.String)),
		RatingValue:     nullFloatPtr(s.skillRatingValue),
		Delta:           nullFloatPtr(s.skillDelta),
		PlaylistGroup:   nullStringPtr(s.skillPlaylistGroup),
		SeasonID:        nullStringPtr(s.skillSeasonID),
		ExpectedWinProb: nullFloatPtr(s.skillExpectedWinProb),
	}
	if s.skillMeasurementRemaining.Valid {
		mr := int(s.skillMeasurementRemaining.Int64)
		snap.MeasurementRemaining = &mr
	}
	if s.skillTier.Valid && s.skillTier.String != "" {
		tier := strings.ToLower(s.skillTier.String)
		snap.TierCode = &tier
	}
	if s.skillTierFR.Valid && s.skillTierFR.String != "" {
		tierFR := s.skillTierFR.String
		snap.TierCodeFR = &tierFR
	}
	if s.skillSubTier.Valid {
		st := int(s.skillSubTier.Int64)
		snap.SubTier = &st
	}
	return &snap
}

// projectDamageStats convertit damage_dealt / damage_taken (DOUBLE en DB) en *int.
// Arrondi (math.Round) plutôt que troncature pour ne pas biaiser systématiquement
// les dégâts vers le bas (impacte le combat yield dérivé).
func projectDamageStats(s playerMatchScanResult) (*int, *int) {
	var dmgDealt, dmgTaken *int
	if s.damageDealt.Valid {
		v := int(math.Round(s.damageDealt.Float64))
		dmgDealt = &v
	}
	if s.damageTaken.Valid {
		v := int(math.Round(s.damageTaken.Float64))
		dmgTaken = &v
	}
	return dmgDealt, dmgTaken
}

// projectMatchSummary projette la section Summary depuis les champs scannés.
func projectMatchSummary(s playerMatchScanResult, outcome canonical.Outcome, teams []canonical.TeamSnapshot) canonical.MatchSummary {
	durationPtr := s.durationSeconds
	return canonical.MatchSummary{
		MatchID:         s.matchID,
		StartedAtUTC:    s.startTime,
		DurationSeconds: &durationPtr,
		MatchType:       MatchTypeFromFlags(s.isRanked, s.isFirefight),
		Playlist:        AssetReference("playlist", s.playlistID, s.playlistName, s.playlistNameFR),
		Map:             AssetReference("map", s.mapID, s.mapName, s.mapNameFR),
		GameVariant:     AssetReference("game_variant", s.variantID, s.variantName, ""),
		RoundsTotal:     nonNegPtr(s.roundsTotal),
		PairMode:        AssetReference("pair_mode", s.pairID, s.pairName, s.pairNameFR),
		IsRanked:        &s.isRanked,
		IsPvE:           &s.isFirefight,
		Outcome:         outcome,
		Teams:           teams,
		T0Ms:            nullInt64ToInt64Ptr(s.t0Ms),
	}
}

// projectSelfParticipant projette la section Self depuis les champs scannés.
func projectSelfParticipant(s playerMatchScanResult, outcome canonical.Outcome, dmgDealt, dmgTaken *int) canonical.MatchParticipant {
	teamIDPtr := s.teamID
	killsPtr, deathsPtr, assistsPtr := s.kills, s.deaths, s.assists
	headshotPtr := s.headshotKills
	timePlayedPtr := s.timePlayedSeconds
	return canonical.MatchParticipant{
		Identity:           canonical.PlayerIdentity{XUID: s.xuid, Gamertag: s.gamertag},
		TeamID:             &teamIDPtr,
		Outcome:            outcome,
		Kills:              &killsPtr,
		Deaths:             &deathsPtr,
		Assists:            &assistsPtr,
		HeadshotKills:      &headshotPtr,
		KDA:                nullFloatPtr(s.kda),
		Accuracy:           effectiveAccuracyPct(s),
		AvgLifeSeconds:     nullFloatPtr(s.avgLifeSeconds),
		TimePlayed:         &timePlayedPtr,
		DamageDealt:        dmgDealt,
		DamageTaken:        dmgTaken,
		MaxKillingSpree:    nullInt64ToIntPtr(s.maxKillingSpree),
		PersonalScore:      nullInt64ToIntPtr(s.personalScore),
		RankInMatch:        nullInt64ToIntPtr(s.rankInMatch),
		GrenadeKills:       nullInt64ToIntPtr(s.grenadeKills),
		MeleeKills:         nullInt64ToIntPtr(s.meleeKills),
		PowerWeaponKills:   nullInt64ToIntPtr(s.powerWeaponKills),
		AssassinationKills: nullInt64ToIntPtr(s.assassinationKills),
		GroundPoundKills:   nullInt64ToIntPtr(s.groundPoundKills),
		ShoulderBashKills:  nullInt64ToIntPtr(s.shoulderBashKills),
		ShotsFired:         nullInt64ToIntPtr(s.shotsFired),
		ShotsHit:           nullInt64ToIntPtr(s.shotsHit),
		PerfectKills:       nullInt64ToIntPtr(s.perfectKills),
		KillsExpected:      nullFloatPtr(s.killsExpected),
		DeathsExpected:     nullFloatPtr(s.deathsExpected),
	}
}

// projectEnrichment projette la section Enrichment (PME + skill).
func projectEnrichment(s playerMatchScanResult, skillSnap *canonical.SkillSnapshot) canonical.PlayerMatchEnrichment {
	return canonical.PlayerMatchEnrichment{
		SessionID:           nullStringPtr(s.sessionID),
		SessionLabel:        nullStringPtr(s.sessionLabel),
		PairName:            strValueToPtr(s.pairName),
		PerformanceScore:    nullFloatPtr(s.performanceScore),
		DominanceFlag:       canonical.DominanceFlag(s.dominanceFlag),
		HadBotTeammate:      s.hadBotTeammate,
		IsWithFriends:       s.isWithFriends,
		TeamMMR:             nullFloatPtr(s.teamMMR),
		EnemyMMR:            nullFloatPtr(s.enemyMMR),
		SkillSnapshot:       skillSnap,
		EngagementScoreBrut: nullFloatPtr(s.engagementScoreBrut),
		EngagementPaceRatio: nullFloatPtr(s.engagementPaceRatio),
	}
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

// OutcomeFromInt convertit le code int DB vers un canonical.Outcome.
func OutcomeFromInt(i int) canonical.Outcome {
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

// MatchTypeFromFlags choisit un MatchType canonique a partir de is_ranked /
// is_firefight (selection prioritaire : firefight > ranked > social).
func MatchTypeFromFlags(isRanked, isFirefight bool) canonical.MatchType {
	if isFirefight {
		return canonical.MatchTypeFirefight
	}
	if isRanked {
		return canonical.MatchTypeRanked
	}
	return canonical.MatchTypeSocial
}

// AssetReference compose un canonical.AssetReference depuis les colonnes DB.
// Retourne nil si aucun ID ni label.
func AssetReference(kind, id, name, nameFR string) *canonical.AssetReference {
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

// ErrUnknownOrderBy est retournee si OrderBy n'est pas dans la whitelist.
// Utilisée par classifyOrderBy (cf. partie split shared/post-merge).
var ErrUnknownOrderBy = errors.New("PlayerMatchesRepo: unknown OrderBy")

// effectiveAccuracyPct retourne la précision par-match en échelle 0..100 (même
// convention que la colonne `accuracy` stockée pour Infinite, cf. consumers
// match_view_radar.go / session_compare_participation_helpers.go qui font /100).
//
// Title-agnostique : si la colonne `accuracy` est peuplée et > 0 (Infinite), on
// la renvoie telle quelle (no-op). Sinon (Halo 5 : `accuracy` NULL mais la
// carnage fournit shots_fired/shots_hit), on la calcule depuis les tirs. nil si
// aucune des deux sources n'est exploitable.
func effectiveAccuracyPct(s playerMatchScanResult) *float64 {
	if s.accuracy.Valid && s.accuracy.Float64 > 0 {
		v := s.accuracy.Float64
		return &v
	}
	if s.shotsFired.Valid && s.shotsFired.Int64 > 0 && s.shotsHit.Valid {
		v := float64(s.shotsHit.Int64) * 100.0 / float64(s.shotsFired.Int64)
		return &v
	}
	return nil
}

// nullFloatPtr convertit sql.NullFloat64 en *float64.
func nullFloatPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// strValueToPtr renvoie nil pour la chaîne vide, sinon un pointeur vers la valeur.
// Utilisé pour surfacer pair_name (colonne shared non-nullable, "" = absent).
func strValueToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullInt64ToIntPtr convertit sql.NullInt64 en *int.
func nullInt64ToIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// nullInt64ToInt64Ptr convertit sql.NullInt64 en *int64 (sans troncature).
// Utilisé pour T0Ms (offset countdown en ms).
func nullInt64ToInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
