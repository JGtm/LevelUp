// Package analysis — world_stats.go : extraction + agrégation PURE des stats
// joueur du classement mondial (Phase C, PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Zéro accès API/DB : opère sur le JSON brut d'un match (GetMatchStats, map) et
// produit des compteurs agrégés par (saison CSR, playlist). L'orchestration
// (fetch via le pool de tokens + persistance) vit dans internal/service.
//
// Chemins JSON confirmés en Phase A (probe) :
//   - Players[] ciblé par PlayerId == "xuid(N)"
//   - Outcome numérique (2=W, 3=L, 1=T, 4=DNF)
//   - CoreStats dans PlayerTeamStats[0].Stats.CoreStats (Kills/Deaths/Assists)
//   - ParticipationInfo.TimePlayed = durée ISO-8601 (PT10M39.203S)
//   - MatchInfo.SeasonId = "Csr/Seasons/CsrSeason13-2.json" (à normaliser)
//   - MatchInfo.Playlist.AssetId = id playlist (= season_id/playlist_id snapshot)
package analysis

import (
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// PlayerMatchStat : stats d'UN match pour le joueur cible (extraction pure).
type PlayerMatchStat struct {
	SeasonID    string // saison CSR normalisée (ex: "csrseason13-2"), "" si hors-CSR
	PlaylistID  string // MatchInfo.Playlist.AssetId
	Outcome     int    // 2=W, 3=L, 1=T, 4=DNF, 0=inconnu
	Kills       int64
	Deaths      int64
	Assists     int64
	PlaytimeSec float64
}

// NormalizeSeasonID convertit MatchInfo.SeasonId ("Csr/Seasons/CsrSeason13-2.json")
// au format des snapshots du classement ("csrseason13-2"). Les matchs hors-CSR
// (ancien format "Seasons/Season6.json") donnent "seasonN" → ne matcheront aucune
// saison CSR scrapée (ignorés à l'agrégation, attendu).
func NormalizeSeasonID(raw string) string {
	s := raw
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimSuffix(s, ".json")
	return strings.ToLower(strings.TrimSpace(s))
}

// ExtractPlayerMatchStat extrait du JSON d'un match (GetMatchStats) les stats du
// joueur `xuid`. ok=false si le joueur n'est pas dans Players[] (match d'un autre).
func ExtractPlayerMatchStat(matchJSON map[string]any, xuid string) (PlayerMatchStat, bool) {
	mi, _ := matchJSON["MatchInfo"].(map[string]any)
	if mi == nil {
		return PlayerMatchStat{}, false
	}
	season := NormalizeSeasonID(strOf(mi["SeasonId"]))
	playlistID := ""
	if pl, ok := mi["Playlist"].(map[string]any); ok {
		playlistID, _ = pl["AssetId"].(string)
	}

	players, _ := matchJSON["Players"].([]any)
	want := "xuid(" + xuid + ")"
	for _, p := range players {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := pm["PlayerId"].(string); id != want {
			continue
		}
		core := coreStatsOf(pm)
		return PlayerMatchStat{
			SeasonID:    season,
			PlaylistID:  playlistID,
			Outcome:     intOf(pm["Outcome"]),
			Kills:       i64Of(core, "Kills"),
			Deaths:      i64Of(core, "Deaths"),
			Assists:     i64Of(core, "Assists"),
			PlaytimeSec: iso8601Seconds(timePlayedOf(pm)),
		}, true
	}
	return PlayerMatchStat{}, false
}

// AccumulateWorldStats bucket les stats par (saison, playlist) et somme les
// compteurs bruts. Les stats sans saison/playlist (hors-CSR, ou playlist absente)
// sont ignorées. L'ordre de sortie suit l'ordre de première apparition.
func AccumulateWorldStats(gamertag string, stats []PlayerMatchStat) []domain.WorldPlayerSeasonStats {
	type key struct{ season, playlist string }
	buckets := map[key]*domain.WorldPlayerSeasonStats{}
	var order []key
	for _, s := range stats {
		if s.SeasonID == "" || s.PlaylistID == "" {
			continue
		}
		k := key{s.SeasonID, s.PlaylistID}
		b, ok := buckets[k]
		if !ok {
			b = &domain.WorldPlayerSeasonStats{
				TitleSlug: "halo_infinite", Gamertag: gamertag,
				SeasonID: s.SeasonID, PlaylistID: s.PlaylistID,
			}
			buckets[k] = b
			order = append(order, k)
		}
		b.MatchCount++
		switch s.Outcome {
		case 2:
			b.WinCount++
		case 3:
			b.LossCount++
		case 1:
			b.TieCount++
		case 4:
			b.DnfCount++
		}
		b.Kills += s.Kills
		b.Deaths += s.Deaths
		b.Assists += s.Assists
		b.PlaytimeSec += int64(s.PlaytimeSec)
	}
	out := make([]domain.WorldPlayerSeasonStats, 0, len(order))
	for _, k := range order {
		out = append(out, *buckets[k])
	}
	return out
}

// ─── Helpers d'extraction (chemins Phase A) ───

// coreStatsOf retourne PlayerTeamStats[0].Stats.CoreStats (fallback [0].CoreStats).
func coreStatsOf(pl map[string]any) map[string]any {
	pts, _ := pl["PlayerTeamStats"].([]any)
	if len(pts) == 0 {
		return nil
	}
	first, _ := pts[0].(map[string]any)
	if first == nil {
		return nil
	}
	if st, ok := first["Stats"].(map[string]any); ok {
		if core, ok := st["CoreStats"].(map[string]any); ok {
			return core
		}
	}
	core, _ := first["CoreStats"].(map[string]any)
	return core
}

func timePlayedOf(pl map[string]any) string {
	pi, _ := pl["ParticipationInfo"].(map[string]any)
	if pi == nil {
		return ""
	}
	tp, _ := pi["TimePlayed"].(string)
	return tp
}

func strOf(v any) string {
	s, _ := v.(string)
	return s
}

func intOf(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func i64Of(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	f, _ := m[key].(float64)
	return int64(f)
}

// iso8601Seconds parse une durée ISO-8601 ("PT10M39.203S") en secondes.
func iso8601Seconds(s string) float64 {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "PT")
	var total float64
	num := ""
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			num += string(r)
			continue
		}
		v, _ := strconv.ParseFloat(num, 64)
		num = ""
		switch r {
		case 'H':
			total += v * 3600
		case 'M':
			total += v * 60
		case 'S':
			total += v
		}
	}
	return total
}
