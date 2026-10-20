package killsource

// index_motif_tueurs.go — LES TUEURS SANS MORT COMPLETENT LE LIEN PAR MOTIF, ILS NE LE FONT JAMAIS
// TOMBER (lot J7.3, constat FK-3, et sa revue adverse).
//
// FK-3 fait chercher au motif les joueurs qui TUENT sans mourir : un remplacant absent de la table
// (ecrite a l ouverture) n etait cherche nulle part. Mais un remplacant qui reprend l indice d un
// PARTANT de la table est lu au MEME indice que lui ; traite comme les autres xuids, il creait une
// collision, et [indexParMotif.refuserSiElleSeContredit] vidait tout l epinglage par motif du film —
// le defaut que le lot 5.2b.1 avait ferme sur `b1ad85eb`.
//
// LA REGLE : ces candidats passent APRES les autres et ne font que completer. Un tueur sans mort
// dont la lecture se contredit (deux indices), ou dont l indice est deja retenu, est ecarte SEUL et
// compte (`tueursEcartes`) ; deux tueurs sans mort au meme indice sont ecartes tous deux. Aucune de
// ces issues n entre dans `desaccords` : la regle « une lecture qui se contredit n est pas une
// lecture » reste celle des xuids que la table ou une mort du feed nomment.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// retenirLesLectures range les lectures unanimes : d abord les xuids nommes par la table ou par une
// mort du kill-feed, avec la regle d avant ; puis les tueurs sans mort, qui ne font que completer.
func (m *indexParMotif) retenirLesLectures(vus map[uint64]map[int]int, xuids []uint64,
	nomDuXUID map[uint64]string, tueurs map[uint64]bool) {
	for _, x := range xuids {
		if tueurs[x] {
			continue
		}
		switch par := vus[x]; {
		case len(par) == 0:
			m.absents++
		case len(par) > 1:
			m.desaccords++
		default:
			for pi := range par {
				m.retenir(pi, nomDuXUID[x])
			}
		}
	}
	m.completerParLesTueurs(vus, xuids, nomDuXUID, tueurs)
}

// completerParLesTueurs : le second etage. `pris` retient les indices poses a CET etage, pour
// ecarter les deux candidats d une collision entre tueurs sans mort.
func (m *indexParMotif) completerParLesTueurs(vus map[uint64]map[int]int, xuids []uint64,
	nomDuXUID map[uint64]string, tueurs map[uint64]bool) {
	pris, bloques := map[int]bool{}, map[int]bool{}
	for _, x := range xuids {
		if !tueurs[x] {
			continue
		}
		par := vus[x]
		switch {
		case len(par) == 0:
			m.absents++
			continue
		case len(par) > 1:
			m.tueursEcartes++
			continue
		}
		for pi := range par {
			nom := nomDuXUID[x]
			if nom == "" || pi < 0 || pi >= 32 {
				continue
			}
			if _, deja := m.nomParIndex[pi]; deja || bloques[pi] {
				m.tueursEcartes++
				if pris[pi] && !bloques[pi] { // l occupant est lui aussi un tueur sans mort : les deux sortent
					delete(m.nomParIndex, pi)
					m.tueursEcartes++
					bloques[pi] = true
				}
				continue
			}
			m.nomParIndex[pi] = nom
			pris[pi] = true
		}
	}
}

// tueursSansMort : les xuids cherches que NI la table NI une mort du kill-feed ne nomment — ceux que
// FK-3 (lot J7.3) ajoute parce qu ils tuent.
func tueursSansMort(slots []types.PlayerSlot, kf *killFeed, nomDuXUID map[uint64]string) map[uint64]bool {
	connus := make(map[uint64]bool, len(slots))
	for _, s := range slots {
		connus[s.XUID] = true
	}
	morts := make(map[string]bool, len(kf.events))
	for _, e := range kf.events {
		if e.victim != "" {
			morts[e.victim] = true
		}
	}
	out := map[uint64]bool{}
	for x, nom := range nomDuXUID {
		if !connus[x] && !morts[nom] {
			out[x] = true
		}
	}
	return out
}
