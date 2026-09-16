package replay

// roster_hors_ligne_test.go — LE ROSTER D'UNE CUISSON HORS LIGNE (lot 1.6.3).
//
// # CE QUE « HORS LIGNE » VEUT DIRE, ET POURQUOI C'EST LE CAS QUI COMPTE
//
// Une cuisson HORS LIGNE est une cuisson SANS FAITS : ni feuille de match, ni tableau de
// participants, ni bots declares. C'est la propriete que tout ce pipeline cherche a preserver —
// un film doit pouvoir se rejouer sans base — et c'est aussi ce que le corpus gate et le harnais
// d'equivalence font a chaque passe.
//
// # LE TROU QUE LA TABLE DU FILM COMBLE
//
// Sans feuille, le roster d'entree du balayage d'index est celui du FIL DES MORTS
// (`rosterFromDeaths`) : un joueur qui ne meurt JAMAIS n'est dans aucun enregistrement du fil,
// donc dans aucun roster, donc dans aucune table d'index, donc dans aucun roster publie. Le trou
// est structurel, il etait connu (`3372e7eb` : 6 joueurs publies pour 8) et la seule reponse
// etait « fournir la feuille ». La table du film le comble SANS base : elle assoit tout le monde.
//
// # COMMENT LA CUISSON HORS LIGNE SE REJOUE ICI, EXACTEMENT
//
// Les fixtures d'entrees figes sont decodes AVEC la feuille (lot 1.0, constat R1-1) : leur table
// d'index porte donc aussi les joueurs a zero mort. On la RESTREINT aux xuids du fil des morts,
// et c'est un MODELE EXACT de la lecture hors ligne, pas une approximation :
// `ScanPlayerIndices(film, rosterFromDeaths(deaths))` ne cherche QUE ces xuids-la, et la mesure du
// corpus donne 0 desaccord d'index sur les huit builds — les index rendus sont donc les memes.

import (
	"sort"
	"testing"
)

// TestRosterHorsLigneEstCompletParLaTableDuFilm compare, build par build, le roster d'une cuisson
// hors ligne AVEC et SANS la table du film.
func TestRosterHorsLigneEstCompletParLaTableDuFilm(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			opt := g.options()
			opt.Labels = goldenCatalog(t)
			opt.MapQuant = &entry
			// HORS LIGNE : plus rien de la base. La table d'index est restreinte a ce que le fil
			// des morts permet de chercher.
			opt.RosterXUIDs, opt.Participants, opt.Bots, opt.Successions = nil, nil, nil, nil
			opt.PlayerIndices = indexDuFilDesMorts(g.PlayerIndices, g.Deaths)

			sansTable := opt
			sansTable.FilmTable = FilmPlayerTable{Refusal: FilmTableNoSection}
			avant := BuildFromPositions(b.Short8, "halo_infinite", g.Positions, g.Fire, sansTable)
			apres := BuildFromPositions(b.Short8, "halo_infinite", g.Positions, g.Fire, opt)

			manquants := siegesAbsentsDuRoster(g.FilmTable, apres.Roster)
			t.Logf("%s | %s | hors ligne : %d joueur(s) sans la table, %d avec (%d siege(s) au "+
				"film) | sieges absents du roster : %d",
				b.Short8, b.Build, len(avant.Roster), len(apres.Roster), g.FilmTable.Occupied,
				len(manquants))
			if len(manquants) > 0 {
				t.Errorf("%d siege(s) de la table du film absent(s) du roster hors ligne : %v — "+
					"la table assoit tout le monde, le roster doit les porter", len(manquants),
					manquants)
			}
			if len(apres.Roster) < len(avant.Roster) {
				t.Errorf("le roster hors ligne a MAIGRI : %d -> %d", len(avant.Roster),
					len(apres.Roster))
			}
		})
	}
}

// indexDuFilDesMorts restreint une table d'index aux xuids que le fil des morts porte — le
// MODELE EXACT de ce que la lecture rend sans feuille de match (cf. l'en-tete).
func indexDuFilDesMorts(t PlayerIndexTable, deaths []Death) PlayerIndexTable {
	vus := map[uint64]bool{}
	for _, d := range deaths {
		if d.XUID != 0 {
			vus[d.XUID] = true
		}
	}
	out := PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: t.Readings,
		Disagreements: t.Disagreements}
	for x, pi := range t.ByXUID {
		if vus[x] {
			out.ByXUID[x] = pi
		}
	}
	return out
}

// siegesAbsentsDuRoster rend les rangs des sieges de la table du film qu'un roster ne porte pas.
func siegesAbsentsDuRoster(film FilmPlayerTable, roster []RosterEntry) []int {
	publies := map[int]bool{}
	for _, r := range roster {
		if r.XUID != "" {
			publies[r.FilmIndex] = true
		}
	}
	var manquants []int
	for _, s := range film.Seats {
		if !publies[s.FilmIndex] {
			manquants = append(manquants, s.FilmIndex)
		}
	}
	sort.Ints(manquants)
	return manquants
}
