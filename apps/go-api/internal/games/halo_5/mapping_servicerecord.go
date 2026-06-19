// Package halo_5 — mapping_servicerecord.go : projection du service record Halo 5
// (agregat arena) vers PlayerStats + CareerSnapshot (CSR natif).
package halo_5

import (
	"strconv"

	"levelup/go-api/internal/games/canonical"
)

// h5Designations mappe DesignationId (palier CSR majeur) -> labels EN/FR.
// Ordre officiel Halo 5 : Bronze < Silver < Gold < Platinum < Diamond < Onyx.
var h5Designations = []struct{ en, fr string }{
	{"bronze", "Bronze"},
	{"silver", "Argent"},
	{"gold", "Or"},
	{"platinum", "Platine"},
	{"diamond", "Diamant"},
	{"onyx", "Onyx"},
}

// designationLabels retourne (labelEN, labelFR) pour un DesignationId, ("", "")
// si hors borne.
func designationLabels(designationID int) (string, string) {
	if designationID < 0 || designationID >= len(h5Designations) {
		return "", ""
	}
	return h5Designations[designationID].en, h5Designations[designationID].fr
}

// firstArenaResult retourne le corps arena du joueur requete (ResultCode 0), ou nil.
func firstArenaResult(resp *H5ServiceRecordResponse) *H5ArenaStats {
	if resp == nil {
		return nil
	}
	for i := range resp.Results {
		if resp.Results[i].ResultCode == 0 && resp.Results[i].Result.ArenaStats != nil {
			return resp.Results[i].Result.ArenaStats
		}
	}
	return nil
}

// aggregatePlayerStats agrege les ArenaPlaylistStats du service record en
// PlayerStats canonique. nil si pas de stats arena. Les ratios sont des pointeurs
// (distinction "indisponible" vs "0").
func aggregatePlayerStats(resp *H5ServiceRecordResponse, gamertag string) *canonical.PlayerStats {
	arena := firstArenaResult(resp)
	if arena == nil {
		return nil
	}
	var kills, deaths, assists, wins, losses, ties, games, shotsFired, shotsLanded int
	for i := range arena.ArenaPlaylistStats {
		p := &arena.ArenaPlaylistStats[i]
		kills += p.TotalKills
		deaths += p.TotalDeaths
		assists += p.TotalAssists
		wins += p.TotalGamesWon
		losses += p.TotalGamesLost
		ties += p.TotalGamesTied
		games += p.TotalGamesCompleted
		shotsFired += p.TotalShotsFired
		shotsLanded += p.TotalShotsLanded
	}

	stats := &canonical.PlayerStats{
		Identity:      h5Identity(gamertag),
		MatchesPlayed: games,
		Wins:          wins,
		Losses:        losses,
		Ties:          ties,
		Kills:         kills,
		Deaths:        deaths,
		Assists:       assists,
	}
	if games > 0 {
		wr := float64(wins) / float64(games)
		stats.WinRate = &wr
	}
	if deaths > 0 {
		kdr := float64(kills) / float64(deaths)
		stats.KDR = &kdr
		kda := (float64(kills) + float64(assists)/3.0) / float64(deaths)
		stats.KDA = &kda
	}
	if shotsFired > 0 {
		acc := float64(shotsLanded) / float64(shotsFired)
		stats.Accuracy = &acc
	}
	return stats
}

// mapCareerSnapshot projette le pic CSR natif (HighestCsrAttained) vers
// CareerSnapshot. Halo 5 n'a PAS de progression XP facon rang carriere HINF :
// seuls le palier CSR (RankTier/RankName) et la valeur CSR (Onyx) sont alimentes.
// nil si pas de CSR atteint.
func mapCareerSnapshot(resp *H5ServiceRecordResponse, gamertag string) *canonical.CareerSnapshot {
	arena := firstArenaResult(resp)
	if arena == nil || arena.HighestCsrAttained == nil {
		return nil
	}
	csr := arena.HighestCsrAttained
	en, fr := designationLabels(csr.DesignationId)

	snap := &canonical.CareerSnapshot{
		Player: h5Identity(gamertag),
	}
	if fr != "" {
		tier := fr
		snap.RankTier = &tier
		// Nom de rang lisible : "Diamant 5" (palier + sous-palier) ; Onyx sans
		// sous-palier ("Onyx").
		name := fr
		if en != "onyx" && csr.Tier > 0 {
			name = fr + " " + strconv.Itoa(csr.Tier)
		}
		snap.RankName = &name
	}
	// La valeur CSR brute n'est significative que pour Onyx (sinon le palier est
	// porte par Tier+pourcentage). On expose HighestCSR seulement si > 0.
	if csr.Csr > 0 {
		v := csr.Csr
		snap.HighestCSR = &v
	}
	return snap
}
