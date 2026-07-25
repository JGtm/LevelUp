// Package objective — objective_stats.go : extraction des stats objectifs par joueur
// (modes CTF / Zones (Strongholds+KOTH) / Oddball / Stockpile / Extraction / VIP) depuis
// le payload GetMatchStats Halo Infinite vers shared.match_objective_stats.
//
// Sous-package cohésif extrait de la racine internal/sync/ (ratchet god-package K3c) :
// le seul point d'entrée est ExtractObjectiveStats, consommé par engine_fetch.go et
// engine_v2bridge.go via objective.ExtractObjectiveStats.
//
// Les données sont DÉJÀ dans le payload participants (Players[].PlayerTeamStats[0].
// Stats.<BlocMode>), même parent que CoreStats — aucun appel réseau supplémentaire.
// Pur (aucune I/O). Calqué sur ExtractPveStats (pve.go). Colonnes verrouillées sur
// payload réel (P0, PLAN_V72_OBJECTIVE_STATS.md § « P0 — Mapping figé »).
package objective

import (
	"regexp"
	"strconv"

	"levelup/go-api/internal/persist"
)

// objectiveDurationRE matche une durée ISO-8601 PT[nH][nM][nS]. Sibling de ptRE
// (transforms_helpers.go) MAIS préserve les fractions de seconde (parsePTDuration
// tronque en int) — les colonnes de durée objectif sont DOUBLE _seconds.
var objectiveDurationRE = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?$`)

// ExtractObjectiveStats extrait les stats objectifs par joueur depuis un payload
// GetMatchStats. Produit une row par (match_id, xuid) UNIQUEMENT pour les joueurs dont
// le Stats bundle contient l'un des 6 blocs objectif (CaptureTheFlagStats / ZonesStats /
// OddballStats / StockpileStats / ExtractionStats / VipStats — mutuellement exclusifs par
// mode). Les modes sans objectif (Slayer, etc.) et les payloads sans bloc ne produisent
// aucune row (table volontairement creuse). Seul le niveau JOUEUR est extrait : les mêmes
// blocs existent sous Teams[].Stats (agrégat équipe) et sont ignorés — les totaux équipe
// sont recalculés à la lecture.
//
// EliminationStats et InfectionStats restent NON IMPLÉMENTÉS : aucun payload existant ne
// les porte (V721-02 § 02.1 — les 2 matchs Attrition en base n'ont que CoreStats, aucun
// mode Infection en base). Rien n'est déduit par analogie ; leur ajout attend un match réel.
func ExtractObjectiveStats(matchID string, matchJSON map[string]any) []persist.ObjectiveStatsInsert {
	players, _ := matchJSON["Players"].([]any)
	if len(players) == 0 {
		return nil
	}
	var rows []persist.ObjectiveStatsInsert //nolint:prealloc
	for _, p := range players {
		playerMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractPlayerXUID(playerMap)
		if xuid == "" {
			continue
		}
		stats := findStatsBundle(playerMap)
		if stats == nil {
			continue
		}
		if row, ok := buildObjectiveRow(matchID, xuid, stats); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

// findStatsBundle retourne le sous-dict PlayerTeamStats[0].Stats (parent de CoreStats
// et des blocs par mode), ou nil. Parité de navigation avec findCoreStats.
func findStatsBundle(player map[string]any) map[string]any {
	pts, _ := player["PlayerTeamStats"].([]any)
	for _, ts := range pts {
		teamStats, ok := ts.(map[string]any)
		if !ok {
			continue
		}
		if stats, ok := teamStats["Stats"].(map[string]any); ok && stats != nil {
			return stats
		}
	}
	return nil
}

// buildObjectiveRow construit une row objectif depuis le Stats bundle. Retourne
// (row, false) si aucun bloc objectif n'est présent (mode sans objectif → skip).
func buildObjectiveRow(matchID, xuid string, stats map[string]any) (persist.ObjectiveStatsInsert, bool) {
	row := persist.ObjectiveStatsInsert{MatchID: matchID, XUID: xuid}
	found := false

	if ctf, ok := stats["CaptureTheFlagStats"].(map[string]any); ok {
		found = true
		row.FlagCaptures = intPtrFrom(ctf, "FlagCaptures")
		row.FlagCaptureAssists = intPtrFrom(ctf, "FlagCaptureAssists")
		row.FlagGrabs = intPtrFrom(ctf, "FlagGrabs")
		row.FlagSecures = intPtrFrom(ctf, "FlagSecures")
		row.FlagSteals = intPtrFrom(ctf, "FlagSteals")
		row.FlagReturns = intPtrFrom(ctf, "FlagReturns")
		row.FlagCarriersKilled = intPtrFrom(ctf, "FlagCarriersKilled")
		row.FlagReturnersKilled = intPtrFrom(ctf, "FlagReturnersKilled")
		row.KillsAsFlagCarrier = intPtrFrom(ctf, "KillsAsFlagCarrier")
		row.KillsAsFlagReturner = intPtrFrom(ctf, "KillsAsFlagReturner")
		row.TimeAsFlagCarrierSeconds = objectiveDurationSeconds(asString(ctf["TimeAsFlagCarrier"]))
	}

	if zones, ok := stats["ZonesStats"].(map[string]any); ok {
		found = true
		row.ZoneCaptures = intPtrFrom(zones, "StrongholdCaptures")
		row.ZoneSecures = intPtrFrom(zones, "StrongholdSecures")
		row.ZoneOffensiveKills = intPtrFrom(zones, "StrongholdOffensiveKills")
		row.ZoneDefensiveKills = intPtrFrom(zones, "StrongholdDefensiveKills")
		row.ZoneScoringTicks = intPtrFrom(zones, "StrongholdScoringTicks")
		row.TimeInZonesSeconds = objectiveDurationSeconds(asString(zones["StrongholdOccupationTime"]))
	}

	if odd, ok := stats["OddballStats"].(map[string]any); ok {
		found = true
		row.KillsAsSkullCarrier = intPtrFrom(odd, "KillsAsSkullCarrier")
		row.SkullCarriersKilled = intPtrFrom(odd, "SkullCarriersKilled")
		row.SkullGrabs = intPtrFrom(odd, "SkullGrabs")
		row.SkullScoringTicks = intPtrFrom(odd, "SkullScoringTicks")
		row.TimeAsSkullCarrierSeconds = objectiveDurationSeconds(asString(odd["TimeAsSkullCarrier"]))
		row.LongestTimeAsSkullCarrierSeconds = objectiveDurationSeconds(asString(odd["LongestTimeAsSkullCarrier"]))
	}

	if stock, ok := stats["StockpileStats"].(map[string]any); ok {
		found = true
		row.KillsAsPowerSeedCarrier = intPtrFrom(stock, "KillsAsPowerSeedCarrier")
		row.PowerSeedCarriersKilled = intPtrFrom(stock, "PowerSeedCarriersKilled")
		row.PowerSeedsDeposited = intPtrFrom(stock, "PowerSeedsDeposited")
		row.PowerSeedsStolen = intPtrFrom(stock, "PowerSeedsStolen")
		row.TimeAsPowerSeedCarrierSeconds = objectiveDurationSeconds(asString(stock["TimeAsPowerSeedCarrier"]))
		row.TimeAsPowerSeedDriverSeconds = objectiveDurationSeconds(asString(stock["TimeAsPowerSeedDriver"]))
	}

	if extr, ok := stats["ExtractionStats"].(map[string]any); ok {
		found = true
		row.ExtractionConversionsCompleted = intPtrFrom(extr, "ExtractionConversionsCompleted")
		row.ExtractionConversionsDenied = intPtrFrom(extr, "ExtractionConversionsDenied")
		row.ExtractionInitiationsCompleted = intPtrFrom(extr, "ExtractionInitiationsCompleted")
		row.ExtractionInitiationsDenied = intPtrFrom(extr, "ExtractionInitiationsDenied")
		row.SuccessfulExtractions = intPtrFrom(extr, "SuccessfulExtractions")
	}

	if vip, ok := stats["VipStats"].(map[string]any); ok {
		found = true
		row.KillsAsVip = intPtrFrom(vip, "KillsAsVip")
		row.VipKills = intPtrFrom(vip, "VipKills")
		row.VipAssists = intPtrFrom(vip, "VipAssists")
		row.TimesSelectedAsVip = intPtrFrom(vip, "TimesSelectedAsVip")
		row.MaxKillingSpreeAsVip = intPtrFrom(vip, "MaxKillingSpreeAsVip")
		row.TimeAsVipSeconds = objectiveDurationSeconds(asString(vip["TimeAsVip"]))
		row.LongestTimeAsVipSeconds = objectiveDurationSeconds(asString(vip["LongestTimeAsVip"]))
	}

	return row, found
}

// objectiveDurationSeconds convertit une durée ISO-8601 "PT1M52.7S" en secondes DOUBLE
// (fractions préservées). Retourne nil si vide ou non parseable — la colonne reste NULL.
func objectiveDurationSeconds(s string) *float64 {
	if s == "" {
		return nil
	}
	m := objectiveDurationRE.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	total := 0.0
	if m[1] != "" {
		h, _ := strconv.ParseFloat(m[1], 64)
		total += h * 3600
	}
	if m[2] != "" {
		min, _ := strconv.ParseFloat(m[2], 64)
		total += min * 60
	}
	if m[3] != "" {
		sec, _ := strconv.ParseFloat(m[3], 64)
		total += sec
	}
	return &total
}
