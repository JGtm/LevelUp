package replay

import (
	"slices"
	"sort"
)

// killpos_posthume.go — LE TUEUR DEJA MORT AU MOMENT DU FRAG (decision utilisateur D-17 du
// 2026-10-17).
//
// # LE FAIT
//
// Un projectile en vol (grenade, roquette, tir lent) tue APRES la mort de son tireur. A l'instant du
// frag, le tueur n'a plus de corps : aucun siege ne le porte, ou le siege qui le nommait n'a plus
// d'echantillon. `positionOf` rend nil, et c'est juste — mais le frag a bien une origine, et c'est
// l'endroit ou le tueur est mort.
//
// # LA REGLE
//
// Le tueur est POSTHUME quand, a l'instant du frag, (1) le placement direct ne le trouve pas,
// (2) le journal porte une mort de ce joueur STRICTEMENT AVANT le frag (une mort a la meme
// milliseconde est un echange ou un suicide : le placement direct en a deja decide), et (3) aucune vie
// NOMMEE a son nom ne s'ouvre entre cette mort et le frag (il n'a pas reapparu). Sa position est
// alors celle de sa mort, lue par la MEME fonction que la position d'une victime (`positionOf` sur
// ses sieges a l'instant de sa mort) ; si sa mort n'est pas localisable, il reste nil.
//
// AUCUNE BORNE DE DUREE : un joueur mort et non reapparu n'a qu'un lieu, celui de sa mort. Le
// repli est nomme (`repli_position_posthume_du_tueur`) et compte au compteur de la passe ; le
// rapport le ventile (`KillPosReport.Posthumous`, `PosthumousUnplaced`).
//
// CE QUE LA REGLE NE VOIT PAS : une reapparition dans une vie que le registre ne NOMME pas. Le
// tueur y serait vivant et pose au lieu de sa mort. Le residu des vies sans nom est le chantier du
// registre (son compte voyage dans `coverage.bridge.unnamedLives`), pas celui de ce fichier.

// mortsDuTueur indexe, par joueur, les instants de mort du journal (horloge du MATCH, ms) et ses
// vies nommees (horloge du FILM, µs) — ce que la regle posthume consulte.
type mortsDuTueur struct {
	morts map[uint64][]int64
	vies  map[uint64][]int64 // debuts des vies nommees, tries
}

// nouvellesMortsDuTueur prepare l'index. Nil quand le journal ne porte aucune mort : la regle ne
// s'applique pas.
func nouvellesMortsDuTueur(journal []MortDuJournal, reg IdentityRegistry) *mortsDuTueur {
	if len(journal) == 0 {
		return nil
	}
	m := &mortsDuTueur{morts: map[uint64][]int64{}, vies: map[uint64][]int64{}}
	for _, d := range journal {
		if d.VictimeXUID != 0 {
			m.morts[d.VictimeXUID] = append(m.morts[d.VictimeXUID], d.TempsMS)
		}
	}
	for _, l := range reg.Vies() {
		if l.xuid != 0 {
			m.vies[l.xuid] = append(m.vies[l.xuid], l.from)
		}
	}
	for _, v := range m.morts {
		slices.Sort(v)
	}
	for _, v := range m.vies {
		slices.Sort(v)
	}
	return m
}

// derniereMort rend l instant (ms de match) de la derniere mort du joueur STRICTEMENT avant `tMS`, si le
// joueur n'a pas reapparu depuis dans une vie nommee.
func (m *mortsDuTueur) derniereMort(xuid uint64, tMS, offsetUS int64) (int64, bool) {
	morts := m.morts[xuid]
	i := sort.Search(len(morts), func(k int) bool { return morts[k] >= tMS })
	if i == 0 {
		return 0, false
	}
	mortMS := morts[i-1]
	mortUS, fragUS := mortMS*1000+offsetUS, tMS*1000+offsetUS
	vies := m.vies[xuid]
	j := sort.Search(len(vies), func(k int) bool { return vies[k] > mortUS })
	if j < len(vies) && vies[j] <= fragUS {
		return 0, false // reapparu avant le frag : il n'est pas mort a cet instant
	}
	return mortMS, true
}

// placerPosthume applique la regle a un tueur que le placement direct n'a pas trouve. Le second
// retour dit que la regle s'applique (le tueur est mort a l'instant), le premier ou il est mort.
func (m *mortsDuTueur) placerPosthume(p *killPlacement, reg IdentityRegistry, sieges []uint32,
	k KillRef, offsetUS int64,
) (*Vec3, bool) {
	if m == nil {
		return nil, false
	}
	mortMS, mort := m.derniereMort(k.KillerXUID, k.TimeMS, offsetUS)
	if !mort {
		return nil, false
	}
	tUS := uint64(mortMS*1000 + offsetUS) //nolint:gosec // horloge du film, bien sous 2^63
	v, _ := positionOf(p.tracks, siegesDe(reg, sieges, k.KillerXUID, tUS), tUS)
	return v, true
}

// placerLeTueurPosthume pose le tueur d'une mort au lieu de sa propre mort quand la regle
// s'applique, et le compte. Le SLOT du cote reste 0 : seul `BuildKillOpenings` le lit, et il ne
// demande jamais la regle.
func placerLeTueurPosthume(out *killPlacement, m *mortsDuTueur, reg IdentityRegistry,
	sieges []uint32, kp *KillPosition, offsetUS int64,
) {
	v, mort := m.placerPosthume(out, reg, sieges, kp.KillRef, offsetUS)
	switch {
	case !mort:
	case v == nil:
		out.report.PosthumousUnplaced++
	default:
		kp.Killer = v
		out.report.Posthumous++
	}
}
