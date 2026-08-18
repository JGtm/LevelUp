package replay

// objectifs_phase0_controles_test.go — LES CONTROLES DE L'ITEM 0.1.
//
// Une valeur qui distingue le portage ne prouve rien tant que trois questions n'ont pas de
// reponse chiffree, et ce fichier les porte toutes les trois :
//
//	1. LE BALAYAGE VOIT-IL CE QU'IL DOIT VOIR ? Controle POSITIF : les familles d'arme du
//	   catalogue doivent se retrouver dans l'emprise des memes records. Si elles n'y sont
//	   pas, l'emprise est fausse et tout le reste est du bruit — quel que soit l'ecart mesure.
//	2. COMBIEN DE MOTIFS DISTINCTS, VRAIMENT ? Une fenetre glissante rend le MEME motif a
//	   quatre decalages successifs sous quatre valeurs differentes. Les compter comme quatre
//	   candidates independantes gonflerait le resultat d'un facteur quatre pour rien.
//	3. LE SEUIL DU PLAN EST-IL TENU ? Il est ecrit en FENETRES (>= 90 % de celles qui
//	   contiennent une image-cle), pas en records, et il vient avec son temoin (un slot NON
//	   porteur tire sur les memes instants, <= 5 %).

import (
	"sort"

	"levelup/go-api/internal/analysis/weaponv3"
)

// objControlePositif rend, sur un lot de records, la part de ceux dont l'emprise contient au
// moins une famille d'arme CONNUE, et le nombre moyen de familles connues par record.
//
// C'est le controle qui valide l'EMPRISE : `keyframe_loadout.go` etablit qu'un record de
// bipede porte ses armes (deux emplacements, plus leurs alias). Si ce balayage-ci ne les
// retrouve pas, il ne lit pas les bons bits.
func objControlePositif(recs []objRecord) (avec, total int, moyenne float64) {
	somme := 0
	for _, r := range recs {
		total++
		n := 0
		for _, v := range r.Vals {
			if _, ok := weaponv3.KnownWeaponHigh32[v]; ok {
				n++
			}
		}
		if n > 0 {
			avec++
		}
		somme += n
	}
	if total > 0 {
		moyenne = float64(somme) / float64(total)
	}
	return avec, total, moyenne
}

// objGroupesDecalage regroupe les valeurs qui ne sont que des vues DECALEES d'un meme motif.
//
// Deux fenetres consecutives de 32 bits sont liees par w = v<<1 | b : deux valeurs voisines
// par cette relation designent le meme endroit du flux, lu un bit plus loin. On rend une
// racine par groupe (la plus petite valeur), et la taille du groupe.
func objGroupesDecalage(vals []uint32) map[uint32][]uint32 {
	ens := make(map[uint32]bool, len(vals))
	for _, v := range vals {
		ens[v] = true
	}
	vu := make(map[uint32]bool, len(vals))
	groupes := map[uint32][]uint32{}
	for _, v := range vals {
		if vu[v] {
			continue
		}
		file := []uint32{v}
		vu[v] = true
		var membres []uint32
		for len(file) > 0 {
			c := file[0]
			file = file[1:]
			membres = append(membres, c)
			for _, n := range objVoisinsDecalage(c) {
				if ens[n] && !vu[n] {
					vu[n] = true
					file = append(file, n)
				}
			}
		}
		sort.Slice(membres, func(i, j int) bool { return membres[i] < membres[j] })
		groupes[membres[0]] = membres
	}
	return groupes
}

// objVoisinsDecalage rend les quatre valeurs atteignables par un decalage d'un bit dans un
// sens ou dans l'autre.
func objVoisinsDecalage(v uint32) [4]uint32 {
	return [4]uint32{v << 1, v<<1 | 1, v >> 1, v>>1 | 1<<31}
}

// objFenetreStat porte le compte du plan : fenetres de portage utilisables, fenetres ou le
// motif est present, et les memes grandeurs pour le temoin.
type objFenetreStat struct {
	AvecKF, Touchees         int
	TemoinKF, TemoinTouchees int
}

// objStatFenetres mesure, fenetre de portage par fenetre de portage, la presence du motif
// chez le PORTEUR et chez un slot NON PORTEUR tire sur les memes instants.
//
// LE TEMOIN PASSE PAR LE MEME CODE ET LES MEMES INSTANTS : seul le slot change. C'est la
// regle du depot (patron du lot des armes au sol) — un temoin qui emprunterait un autre
// chemin ne mesurerait pas le meme hasard.
func objStatFenetres(recs []objRecord, b objBridge, wins []objWindow, val uint32) objFenetreStat {
	parInstant := map[uint64][]objRecord{}
	for _, r := range recs {
		if _, ok := b.SlotXUID[r.Slot]; !ok {
			continue
		}
		parInstant[r.TS] = append(parInstant[r.TS], r)
	}
	var st objFenetreStat
	for _, w := range wins {
		porteur, temoin := objRecordsDeFenetre(parInstant, b, w)
		if len(porteur) == 0 {
			continue
		}
		st.AvecKF++
		if objUnDesRecordsPorte(porteur, val) {
			st.Touchees++
		}
		if len(temoin) > 0 {
			st.TemoinKF++
			if objUnDesRecordsPorte(temoin, val) {
				st.TemoinTouchees++
			}
		}
	}
	return st
}

// objRecordsDeFenetre rend les records du PORTEUR dans la fenetre, et un record TEMOIN par
// instant (un autre joueur, choisi de facon deterministe).
func objRecordsDeFenetre(parInstant map[uint64][]objRecord, b objBridge, w objWindow) (porteur, temoin []objRecord) {
	instants := make([]uint64, 0, len(parInstant))
	for ts := range parInstant {
		instants = append(instants, ts)
	}
	sort.Slice(instants, func(i, j int) bool { return instants[i] < instants[j] })
	for _, ts := range instants {
		matchMS := int64(ts/1000) - b.OffsetMS
		if matchMS < w.T0 || matchMS > w.T1 {
			continue
		}
		var autres []objRecord
		for _, r := range parInstant[ts] {
			if b.SlotXUID[r.Slot] == w.XUID {
				porteur = append(porteur, r)
				continue
			}
			autres = append(autres, r)
		}
		if len(autres) > 0 {
			sort.Slice(autres, func(i, j int) bool { return autres[i].Slot < autres[j].Slot })
			temoin = append(temoin, autres[objTirage(ts, w.T0, len(autres))])
		}
	}
	return porteur, temoin
}

// objTirage rend un index deterministe dans [0, n) — reproductible d'une session a l'autre,
// ce qu'un generateur ensemence a l'horloge ne serait pas.
func objTirage(ts uint64, t0 int64, n int) int {
	h := ts/1000 + uint64(t0)*2654435761
	return int(h % uint64(n))
}

// objUnDesRecordsPorte dit si l'un des records contient la valeur.
func objUnDesRecordsPorte(recs []objRecord, val uint32) bool {
	for _, r := range recs {
		if objPorte(r, val) {
			return true
		}
	}
	return false
}

// objPorte dit si l'emprise du record contient la valeur (les valeurs sont triees).
func objPorte(r objRecord, val uint32) bool {
	i := sort.Search(len(r.Vals), func(i int) bool { return r.Vals[i] >= val })
	return i < len(r.Vals) && r.Vals[i] == val
}

// objFamillesConnuesParCamp rend, pour chaque famille d'arme CONNUE, ses comptes en portage
// et hors portage.
//
// POURQUOI CE CROISEMENT EST DECISIF POUR LE NOMMAGE : en CTF, le porteur de drapeau perd
// l'usage de ses armes et ne garde qu'une arme de poing. Si le portage se lisait seulement
// par un CHANGEMENT D'ARMEMENT, ce sont des familles CONNUES qui le trahiraient — et le
// motif inconnu ne serait pas le drapeau mais une consequence. Le croisement separe les
// deux lectures.
func objFamillesConnuesParCamp(t objTable) []objCandidat {
	var out []objCandidat
	for v, c := range t.Par {
		if _, ok := weaponv3.KnownWeaponHigh32[v]; !ok {
			continue
		}
		tp, th := 0.0, 0.0
		if t.Portage > 0 {
			tp = float64(c.Portage) / float64(t.Portage)
		}
		if t.Hors > 0 {
			th = float64(c.Hors) / float64(t.Hors)
		}
		out = append(out, objCandidat{Val: v, Portage: c.Portage, Hors: c.Hors, TauxP: tp, TauxH: th})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TauxP != out[j].TauxP {
			return out[i].TauxP > out[j].TauxP
		}
		return out[i].Val < out[j].Val
	})
	return out
}

// objPositionsDansRecord rend, pour une valeur, les positions de son PREMIER bit relatives au
// debut du record qui la contient — la mediane dit si le motif vit la ou vivent les armes
// (~+1950 bits, cf. `keyframe_loadout.go`) ou ailleurs.
func objPositionsDansRecord(pay []byte, from, to int, val uint32) []int {
	total := len(pay) * 8
	if to > total {
		to = total
	}
	var out []int
	var w uint32
	for b := from; b < total && b < to+31; b++ {
		w = w<<1 | uint32(pay[b>>3]>>(7-uint(b&7))&1)
		if debut := b - 31; debut >= from && debut < to && w == val {
			out = append(out, debut-from)
		}
	}
	return out
}

// objMediane rend la mediane d'une serie d'entiers (0 si vide).
func objMediane(v []int) int {
	if len(v) == 0 {
		return 0
	}
	c := append([]int{}, v...)
	sort.Ints(c)
	return c[len(c)/2]
}
