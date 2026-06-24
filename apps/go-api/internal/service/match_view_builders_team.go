// Package service — builders pour l'onglet Team de la Match View.
//
// Extrait de match_view_service.go (audit #1 god files). Couvre :
//   - buildTeamTabFull : scoreboard rows + nemesis + encounters.
//   - convertEncounters + buildEncounterBadgesFrom* + encounterWinrate +
//     convertNarrativeBadges : conversion encounters raw -> domain.
//   - buildMediaTab : onglet médias (proche par taille, garder ensemble).
//   - buildNemesisMap : agrégation kvPairs -> nemesis par adversaire.
package service

import (
	"fmt"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// Team Tab
// ---------------------------------------------------------------------------

// buildTeamTabFull assemble scoreboard + kvPairs + encounters + medals + weapons
// en sortie unique pour l'onglet Team. titleSlug maintenu pour future dispatch
// multi-titres (canonical adapter resolution).
//
//nolint:funlen,unparam // Orchestrateur cohésif ; splitter perdrait la vue d'ensemble.
func buildTeamTabFull(
	scoreboard []domain.ScoreboardRaw,
	kvPairs []domain.KVPairRaw,
	encounters []domain.EncounterRaw,
	encounterStats []domain.EncounterStatsRaw,
	bulkMedals []domain.BulkMedalRaw,
	bulkWeapons []domain.BulkWeaponKillRaw,
	myXUID string,
	titleSlug string,
	myEnrich *domain.MatchEnrichmentRaw,
	mySkillRank *domain.SkillRankRaw,
	friendsExtras map[string]port.FriendMatchExtras,
	sharedCSRs map[string]*domain.SkillRankRaw,
	assetURL games.TitleAssetURLAdapter, //nolint:PLR0913 — coordinator function
) domain.MatchTeamTab {
	// Index bulk medals et weapons par XUID pour O(1) lookup (extract helpers).
	medalsByXUID := indexBulkMedalsByXUID(bulkMedals, assetURL, len(scoreboard))
	weaponsByXUID := indexBulkWeaponsByXUID(bulkWeapons, assetURL, len(scoreboard))

	extremes := analysis.ComputeMVPLVP(scoreboard)

	rows := make([]domain.MatchScoreboardRow, 0, len(scoreboard))
	for _, s := range scoreboard {
		oc, dr, dpk, dpd := computeScoreboardRowCombatYield(s, games.EffectiveHpToKill(titleSlug))

		row := domain.MatchScoreboardRow{
			XUID:                s.XUID,
			Gamertag:            s.Gamertag,
			IsBot:               s.IsBot,
			IsMe:                s.XUID == myXUID,
			IsMVP:               extremes.MVPXUID != "" && s.XUID == extremes.MVPXUID,
			IsLVP:               extremes.LVPXUID != "" && s.XUID == extremes.LVPXUID,
			Rank:                s.RankInTeam,
			Kills:               &s.Kills,
			Deaths:              &s.Deaths,
			Assists:             &s.Assists,
			KDA:                 s.KDA,
			Accuracy:            s.Accuracy,
			DamageDealt:         s.DamageDealt,
			DamageTaken:         s.DamageTaken,
			ShotsFired:          s.ShotsFired,
			ShotsHit:            s.ShotsHit,
			AvgLifeSeconds:      s.AvgLifeSeconds,
			HeadshotKills:       s.HeadshotKills,
			MaxKillingSpree:     s.MaxKillingSpree,
			GrenadeKills:        s.GrenadeKills,
			MeleeKills:          s.MeleeKills,
			PowerWeaponKills:    s.PowerWeaponKills,
			OutcomeLabel:        outcomeLabel(s.OutcomeCode),
			Score:               toIntPtr(s.PersonalScore),
			PerfectKills:        &s.PerfectKills,
			TopWeaponID:         s.TopWeaponID,
			TopWeaponLabel:      s.TopWeaponLabel,
			OffensiveConversion: oc,
			DefensiveResistance: dr,
			DamagePerKill:       dpk,
			DamagePerDeath:      dpd,
			ExpectedKills:       s.KillsExpected,
			ExpectedDeaths:      s.DeathsExpected,
			ExpectedAssists:     s.AssistsExpected,
			LocallyEstimated:    s.LocallyEstimated,
			KillsStdDev:         s.KillsStdDev,
			DeathsStdDev:        s.DeathsStdDev,
			Medals:              medalsByXUID[s.XUID],
			WeaponKills:         weaponsByXUID[s.XUID],
		}
		if s.TeamID != nil {
			team := fmt.Sprintf("t%d", *s.TeamID)
			row.TeamSide = &team
		}
		// Section "Local" du panneau d'expander : populée différemment selon
		// que le joueur est `me` (depuis myEnrich + skillRank du contexte) ou
		// un ami (depuis friendsExtras chargés via loader per-DB).
		if row.IsMe {
			if myEnrich != nil {
				if myEnrich.PerformanceScore != nil {
					v := *myEnrich.PerformanceScore
					row.PerformanceScore = &v
				}
				// HadBotTeammate du main player : domain.MatchEnrichmentRaw
				// ne l'expose pas directement (cf. Q18 actuel). Le front lit
				// header.had_bot_teammate qui est rempli ailleurs (page card).
			}
			if mySkillRank != nil {
				iconURL := resolveSkillIconURL(mySkillRank.Tier, mySkillRank.SubTier, mySkillRank.TierLabel, assetURL)
				var iconURLPtr *string
				if iconURL != "" {
					iconURLPtr = &iconURL
				}
				row.SkillRank = &domain.MatchScoreboardSkillRank{
					RatingType:  mySkillRank.RatingType,
					TierLabel:   mySkillRank.TierLabel,
					RatingValue: mySkillRank.RatingValue,
					RatingDelta: mySkillRank.RatingDelta,
					IconURL:     iconURLPtr,
				}
			}
		} else if extras, ok := friendsExtras[s.XUID]; ok {
			row.PerformanceScore = extras.PerformanceScore
			row.HadBotTeammate = extras.HadBotTeammate
			// Priorité : player DB de l'ami (via FriendsExtras) > shared.match_csrs.
			// Si l'ami n'a pas de player DB mais est dans shared.match_csrs, le bloc
			// else ci-dessous prendra le relais.
			row.SkillRank = extras.SkillRank
			if row.SkillRank == nil {
				row.SkillRank = sharedCSRToScoreboardRank(sharedCSRs[s.XUID], assetURL)
			}
			if extras.AssistsModel != nil {
				dd := 0.0
				if s.DamageDealt != nil {
					dd = *s.DamageDealt
				}
				dt := 0.0
				if s.DamageTaken != nil {
					dt = *s.DamageTaken
				}
				mmrDelta := 0.0
				if s.TeamMMR != nil && s.EnemyMMR != nil {
					mmrDelta = *s.TeamMMR - *s.EnemyMMR
				}
				m := extras.AssistsModel
				raw := m.Intercept +
					m.CoefKills*float64(s.Kills) +
					m.CoefDeaths*float64(s.Deaths) +
					m.CoefDamageDealt*dd +
					m.CoefDamageTaken*dt +
					m.CoefMMRDelta*mmrDelta
				v := math.Round(raw*100) / 100
				row.ExpectedAssists = &v
			}
		} else {
			// Joueur non-tracké : fallback sur shared.match_csrs_latest.
			row.SkillRank = sharedCSRToScoreboardRank(sharedCSRs[s.XUID], assetURL)
		}
		rows = append(rows, row)
	}

	// Nemesis depuis KV pairs
	nemesisByXUID := buildNemesisMap(kvPairs, myXUID, scoreboard)
	nemesisList := make([]domain.MatchNemesisRow, 0, len(nemesisByXUID))
	for xuid, n := range nemesisByXUID {
		nemesisList = append(nemesisList, domain.MatchNemesisRow{
			XUID:     xuid,
			Gamertag: n.Gamertag,
			KilledMe: n.KilledMe,
			IKilled:  n.IKilled,
		})
	}
	sortNemesisByKilledMe(nemesisList)

	return domain.MatchTeamTab{
		Roster:     []domain.MatchRosterRow{},
		Scoreboard: rows,
		Nemesis:    nemesisList,
		Encounters: convertEncounters(encounters, encounterStats),
	}
}

// sharedCSRToScoreboardRank convertit un SkillRankRaw depuis shared.match_csrs
// en MatchScoreboardSkillRank avec l'icon_url résolu. Retourne nil si raw est nil.
func sharedCSRToScoreboardRank(raw *domain.SkillRankRaw, assetURL games.TitleAssetURLAdapter) *domain.MatchScoreboardSkillRank {
	if raw == nil {
		return nil
	}
	iconURL := resolveSkillIconURL(raw.Tier, raw.SubTier, raw.TierLabel, assetURL)
	var iconURLPtr *string
	if iconURL != "" {
		iconURLPtr = &iconURL
	}
	return &domain.MatchScoreboardSkillRank{
		RatingType:  raw.RatingType,
		TierLabel:   raw.TierLabel,
		RatingValue: raw.RatingValue,
		RatingDelta: raw.RatingDelta,
		IconURL:     iconURLPtr,
	}
}

// ---------------------------------------------------------------------------
// Encounters
// ---------------------------------------------------------------------------

func convertEncounters(
	raw []domain.EncounterRaw,
	stats []domain.EncounterStatsRaw,
) []domain.MatchEncounterRow {
	if len(raw) == 0 {
		return []domain.MatchEncounterRow{}
	}
	// Index stats par xuid pour O(1) lookup. Optionnel : si stats nil ou
	// vide, on retombe sur le badge ordinal seul (chunk MV4.C legacy).
	statsByXUID := make(map[string]domain.EncounterStatsRaw, len(stats))
	for _, s := range stats {
		statsByXUID[s.XUID] = s
	}

	result := make([]domain.MatchEncounterRow, 0, len(raw))
	for _, e := range raw {
		s, hasStats := statsByXUID[e.XUID]
		var badges []domain.MatchEncounterBadge
		if hasStats {
			badges = buildEncounterBadgesFromStats(e, s)
		} else {
			// MV4.C fallback : seul le badge ordinal attribuable.
			badges = buildEncounterBadgesFromRaw(e)
		}
		row := domain.MatchEncounterRow{
			XUID:          e.XUID,
			Gamertag:      e.Gamertag,
			IsBot:         e.IsBot,
			CountTogether: e.CountTogether,
			IsAlly:        e.IsAlly,
			Badges:        badges,
		}
		if hasStats {
			ally, enemy := s.AllyCount, s.EnemyCount
			kills, deaths := s.KillsDealt, s.DeathsSuffered
			row.AllyCount = &ally
			row.EnemyCount = &enemy
			row.KillsDealt = &kills
			row.DeathsSuffered = &deaths
			row.WinrateAsAlly = encounterWinrate(s.WinsAsAlly, s.LossesAsAlly)
			row.WinrateVsEnemy = encounterWinrate(s.WinsVsEnemy, s.LossesVsEnemy)
			if !s.LastSeenAt.IsZero() {
				ts := s.LastSeenAt
				row.LastSeenAt = &ts
			}
		}
		result = append(result, row)
	}
	return result
}

// buildEncounterBadgesFromRaw : fallback MV4.C — sans stats riches, seul le
// badge ordinal est attribuable.
func buildEncounterBadgesFromRaw(e domain.EncounterRaw) []domain.MatchEncounterBadge {
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	return convertNarrativeBadges(narrative.ComputeEncounterBadges(stats, ordinal))
}

// buildEncounterBadgesFromStats : MV4.C' — utilise les stats riches Q23b
// pour permettre à narrative.ComputeEncounterBadges d'attribuer ally_plus
// et tough_enemy en plus d'ordinal.
func buildEncounterBadgesFromStats(
	e domain.EncounterRaw,
	s domain.EncounterStatsRaw,
) []domain.MatchEncounterBadge {
	winrateAsAlly := encounterWinrate(s.WinsAsAlly, s.LossesAsAlly)
	winrateVsEnemy := encounterWinrate(s.WinsVsEnemy, s.LossesVsEnemy)
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
		AllyCount:       s.AllyCount,
		EnemyCount:      s.EnemyCount,
		WinrateAsAlly:   winrateAsAlly,
		WinrateVsEnemy:  winrateVsEnemy,
		KillsDealt:      s.KillsDealt,
		DeathsSuffered:  s.DeathsSuffered,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	return convertNarrativeBadges(narrative.ComputeEncounterBadges(stats, ordinal))
}

// encounterWinrate : nil si W+L == 0 (pas assez de matchs pour calculer),
// sinon ratio. narrative.ComputeEncounterBadges traite nil comme "pas
// d'attribution ally_plus".
func encounterWinrate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// convertNarrativeBadges : narrative.EncounterBadge -> domain.MatchEncounterBadge.
func convertNarrativeBadges(raw []narrative.EncounterBadge) []domain.MatchEncounterBadge {
	if len(raw) == 0 {
		return nil
	}
	out := make([]domain.MatchEncounterBadge, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Media + Nemesis
// ---------------------------------------------------------------------------

// buildMediaTab construit l'onglet médias.
func buildMediaTab(media []domain.MediaAssocRaw) domain.MatchMediaTab {
	if len(media) == 0 {
		return domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{}}
	}
	items := make([]domain.MatchAssociatedMedia, 0, len(media))
	for _, m := range media {
		items = append(items, domain.MatchAssociatedMedia{
			FileID:       m.FileID,
			FileName:     m.FileName,
			FilePath:     m.FilePath,
			Kind:         m.Kind,
			ThumbnailURL: m.ThumbnailPath,
			CaptureTime:  m.CaptureTime,
			Liked:        m.Liked,
		})
	}
	return domain.MatchMediaTab{MediaItems: items}
}

type nemesisEntry struct {
	Gamertag string
	KilledMe int
	IKilled  int
}

func buildNemesisMap(
	kvPairs []domain.KVPairRaw,
	myXUID string,
	scoreboard []domain.ScoreboardRaw,
) map[string]*nemesisEntry {
	gtMap := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		gtMap[s.XUID] = s.Gamertag
	}

	result := make(map[string]*nemesisEntry)
	for _, kv := range kvPairs {
		if kv.VictimXUID == myXUID {
			if _, ok := result[kv.KillerXUID]; !ok {
				gt := gtMap[kv.KillerXUID]
				if gt == "" {
					gt = kv.KillerGT
				}
				result[kv.KillerXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.KillerXUID].KilledMe += kv.KillCount
		}
		if kv.KillerXUID == myXUID {
			if _, ok := result[kv.VictimXUID]; !ok {
				gt := gtMap[kv.VictimXUID]
				if gt == "" {
					gt = kv.VictimGT
				}
				result[kv.VictimXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.VictimXUID].IKilled += kv.KillCount
		}
	}
	return result
}
