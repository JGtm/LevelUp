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
	}
	// KDA carrière h5 = FDA NET ((k + a/3) − d) / games. EXCEPTION h5 documentée :
	// l'API ne fournit pas de KDA, on le calcule (forme native h5, peut être négatif),
	// jamais le quotient Infinite. Cf. mapping_carnage.go (même formule, par match).
	if games > 0 {
		kda := (float64(kills) + float64(assists)/3.0 - float64(deaths)) / float64(games)
		stats.KDA = &kda
	}
	if shotsFired > 0 {
		acc := float64(shotsLanded) / float64(shotsFired)
		stats.Accuracy = &acc
	}
	return stats
}

// h5DefaultPlacementMatches : nombre de matchs de placement Halo 5 par defaut (valeur
// historique). Override par TitleDescriptor.PlacementMatches via WithPlacementTotal.
const h5DefaultPlacementMatches = 10

// mapCareerSnapshot projette le service record arena vers CareerSnapshot. Halo 5
// n'a PAS de progression XP facon rang carriere HINF : seuls le palier CSR
// (RankTier/RankName, valeur Onyx) et l'etat PLACEMENT (matchs restants / total
// du titre) sont alimentes. nil si pas de stats arena. placementTotal = nb de
// matchs de placement du titre (TitleDescriptor.PlacementMatches).
func mapCareerSnapshot(resp *H5ServiceRecordResponse, gamertag string, placementTotal int) *canonical.CareerSnapshot {
	arena := firstArenaResult(resp)
	if arena == nil {
		return nil
	}
	snap := &canonical.CareerSnapshot{Player: h5Identity(gamertag)}

	// Total de placement du titre : metadonnee toujours exposee si on a des stats arena.
	pt := placementTotal
	if pt <= 0 {
		pt = h5DefaultPlacementMatches
	}
	snap.PlacementTotal = &pt

	// Pic CSR atteint (si le joueur a place quelque part).
	if csr := arena.HighestCsrAttained; csr != nil {
		en, fr := designationLabels(csr.DesignationId)
		if fr != "" {
			tier := fr
			snap.RankTier = &tier
			name := fr // "Diamant 5" (palier + sous-palier) ; Onyx sans sous-palier.
			if en != "onyx" && csr.Tier > 0 {
				name = fr + " " + strconv.Itoa(csr.Tier)
			}
			snap.RankName = &name
		}
		// Valeur CSR brute significative QU'A Onyx (cf. invariant #8 review).
		if en == "onyx" && csr.Csr > 0 {
			v := csr.Csr
			snap.HighestCSR = &v
		}
	}

	// Etat PLACEMENT : aucun palier resolu (pas encore classe) -> exposer les
	// matchs de placement restants (max sur les playlists). Si un palier est
	// resolu, le joueur est classe -> pas de "restants".
	if snap.RankTier == nil {
		if rem := maxMeasurementMatchesLeft(arena); rem > 0 {
			snap.MeasurementMatchesRemaining = &rem
		}
	}
	return snap
}

// maxMeasurementMatchesLeft retourne le maximum de MeasurementMatchesLeft sur les
// playlists arena (0 si aucune en placement).
func maxMeasurementMatchesLeft(arena *H5ArenaStats) int {
	best := 0
	for i := range arena.ArenaPlaylistStats {
		if n := arena.ArenaPlaylistStats[i].MeasurementMatchesLeft; n > best {
			best = n
		}
	}
	return best
}
