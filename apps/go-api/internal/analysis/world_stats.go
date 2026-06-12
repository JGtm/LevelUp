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
	// Valeurs natives du jeu (CoreStats), lues telles quelles — sommées à l'accumulation.
	KDA         float64 // CoreStats.KDA natif (K + A/3 − D)
	Accuracy    float64 // CoreStats.Accuracy native (%)
	DamageDealt int64   // CoreStats.DamageDealt
	DamageTaken int64   // CoreStats.DamageTaken
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

// matchHeader extrait la saison normalisée, la playlist et la liste des joueurs.
func matchHeader(matchJSON map[string]any) (season, playlistID string, players []any, ok bool) {
	mi, _ := matchJSON["MatchInfo"].(map[string]any)
	if mi == nil {
		return "", "", nil, false
	}
	season = NormalizeSeasonID(strOf(mi["SeasonId"]))
	if pl, okp := mi["Playlist"].(map[string]any); okp {
		playlistID, _ = pl["AssetId"].(string)
	}
	players, _ = matchJSON["Players"].([]any)
	return season, playlistID, players, true
}

// playerStatFrom construit une PlayerMatchStat depuis le bloc joueur (valeurs natives brutes).
func playerStatFrom(pm map[string]any, season, playlistID string) PlayerMatchStat {
	core := coreStatsOf(pm)
	return PlayerMatchStat{
		SeasonID:    season,
		PlaylistID:  playlistID,
		Outcome:     intOf(pm["Outcome"]),
		Kills:       i64Of(core, "Kills"),
		Deaths:      i64Of(core, "Deaths"),
		Assists:     i64Of(core, "Assists"),
		PlaytimeSec: iso8601Seconds(timePlayedOf(pm)),
		KDA:         floatOf(core, "KDA"),
		Accuracy:    floatOf(core, "Accuracy"),
		DamageDealt: i64Of(core, "DamageDealt"),
		DamageTaken: i64Of(core, "DamageTaken"),
	}
}

// xuidFromPlayerID extrait "N" de "xuid(N)" (sinon la chaîne telle quelle).
func xuidFromPlayerID(playerID string) string {
	s := strings.TrimPrefix(playerID, "xuid(")
	return strings.TrimSuffix(s, ")")
}

// ExtractPlayerMatchStat extrait du JSON d'un match (GetMatchStats) les stats du
// joueur `xuid`. ok=false si le joueur n'est pas dans Players[] (match d'un autre).
func ExtractPlayerMatchStat(matchJSON map[string]any, xuid string) (PlayerMatchStat, bool) {
	season, playlistID, players, ok := matchHeader(matchJSON)
	if !ok {
		return PlayerMatchStat{}, false
	}
	want := "xuid(" + xuid + ")"
	for _, p := range players {
		pm, okp := p.(map[string]any)
		if !okp {
			continue
		}
		if id, _ := pm["PlayerId"].(string); id != want {
			continue
		}
		return playerStatFrom(pm, season, playlistID), true
	}
	return PlayerMatchStat{}, false
}

// ExtractWorldPlayersFromMatch extrait, en UNE passe, la stat de CHAQUE joueur
// mondial (xuid ∈ worldXuids) présent dans le match. Clé du dédup : un seul
// GetMatchStats traite jusqu'à 8 joueurs cibles (ils s'affrontent en permanence).
// Retourne la saison normalisée du match + map xuid→stat (joueurs mondiaux présents).
func ExtractWorldPlayersFromMatch(matchJSON map[string]any, worldXuids map[string]bool) (season string, out map[string]PlayerMatchStat) {
	season, playlistID, players, ok := matchHeader(matchJSON)
	if !ok {
		return "", nil
	}
	out = make(map[string]PlayerMatchStat)
	for _, p := range players {
		pm, okp := p.(map[string]any)
		if !okp {
			continue
		}
		id, _ := pm["PlayerId"].(string)
		xuid := xuidFromPlayerID(id)
		if xuid == "" || !worldXuids[xuid] {
			continue
		}
		out[xuid] = playerStatFrom(pm, season, playlistID)
	}
	return season, out
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
		// Sommes des valeurs natives du jeu (brut, aucune dérivation).
		b.KDA += s.KDA
		b.Accuracy += s.Accuracy
		b.DamageDealt += s.DamageDealt
		b.DamageTaken += s.DamageTaken
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

func floatOf(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	f, _ := m[key].(float64)
	return f
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
