//go:build research

package grammar

// r_loc_vuea_research_test.go — compagnon de `r_loc_ls_research_test.go` (chantier r_loc,
// 2026-10-02) : R-L1 (c), la vue A des paquets a evenements.
//
// CE QUE L ECRIVAIN DIT (Ghidra, lecture seule, extraits dans
// `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_loc_ghidra/`) : `FUN_142f2c050` recopie, sans
// prefixe de longueur, des messages DEJA serialises (`FUN_1406d60f4(w, _, 0x1451f9920 + off,
// nbits)`), chacun ecrit par `FUN_140bbd474` = `1` + `R(7)` genre + trois references gardees +
// la charge du genre (`FUN_1424d80bc` -> `vtable[0x60]` du genre) ; l enregistreur pose ensuite
// le terminateur `0` (`FUN_1406d49c4(w, 0)`) et la vue B commence au bit suivant. La fin de la vue
// A EST donc le debut de la vue B — mais elle ne se lit qu en lisant la charge de CHAQUE message.
//
// L ORACLE DE FIN DE VUE A (une MESURE, pas un localisateur) : sur un paquet que le localisateur ne
// trouve pas, la premiere position `e` precedee d un bit nul, qui porte un en-tete de record
// lisible (DELTA qui se decode sur un slot lie, generation du profil ; ou NEW de la bande
// annoncee), et d ou la marche complete FERME la vue C au bit pres. Si la lecture de la vue A
// etait portee, elle rendrait une position de cette famille ; l oracle en est la borne, et son
// CONTROLE (paquets localises par la signature du slot 123) dit combien de fois le premier depart
// qui ferme est coherent avec la signature (la chaine des records qui en part tombe sur elle).

import (
	"fmt"
)

// rlocPlafondEssais borne les marches d essai d un paquet : garde de boucle hors ligne, pas une
// largeur de grammaire.
const rlocPlafondEssais = 256

// rlocDebutMinVueA : le plus petit debut de vue B apres une vue A NON vide (configuration,
// continuation, genre R(7), trois gardes de reference nulles, terminateur).
const rlocDebutMinVueA = 2 + LargeurGenreVueA + 3 + 1

// rlocEssaiVueA : le debut que l oracle a trouve (-1 : aucun) et le nombre de marches essayees.
type rlocEssaiVueA struct{ debut, essais int }

// rlocCandidatsVueA rend, tries, les departs `e < fin` precedes d un bit nul qui portent un
// en-tete de record lisible.
func rlocCandidatsVueA(pay []byte, w *World, cfg FrameConfig, fin int) []int {
	defer cfg.Obs.neutraliserLesCrochetsDeCanal()()
	neufs := map[int]bool{}
	if motFacultatifDEnTete(cfg) == 0 {
		for _, p := range candidatsDeTete(pay, fin, w) {
			neufs[p] = true
		}
	}
	var out []int
	for e := rlocDebutMinVueA; e < fin && e < len(pay)*8; e++ {
		if bitAvant(pay, e) != 0 {
			continue
		}
		if neufs[e] {
			out = append(out, e)
			continue
		}
		snap := w.Snapshot()
		rec, _, ok := TryDeltaAt(pay, e, w, cfg)
		ok = ok && w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte)
		w.Restore(snap)
		if ok {
			out = append(out, e)
		}
	}
	return out
}

// rlocFerme dit si la marche complete partie de `e` ferme la vue C au bit pres (monde restaure).
func rlocFerme(pay []byte, w *World, cfg FrameConfig, e int) bool {
	essai := cfg
	obs := NouvelleObservation()
	ferme := false
	obs.VueControleHook = func(l LectureVueC) { ferme = l.Fermee }
	essai.Obs = obs
	snap := w.Snapshot()
	DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, e)
	w.Restore(snap)
	return ferme
}

// rlocPremierQuiFerme : le premier candidat `e < fin` d ou la marche ferme.
func rlocPremierQuiFerme(pay []byte, w *World, cfg FrameConfig, fin int) rlocEssaiVueA {
	r := rlocEssaiVueA{debut: -1}
	for _, e := range rlocCandidatsVueA(pay, w, cfg, fin) {
		if r.essais >= rlocPlafondEssais {
			break
		}
		r.essais++
		if rlocFerme(pay, w, cfg, e) {
			r.debut = e
			return r
		}
	}
	return r
}

// rlocOracleVueA : l oracle de fin de vue A sur tout le payload.
func rlocOracleVueA(pay []byte, w *World, cfg FrameConfig) rlocEssaiVueA {
	return rlocPremierQuiFerme(pay, w, cfg, len(pay)*8)
}

// rlocChaine marche les records de `pos` a `debut` par les pas de la chaine de tete
// ([pasDEssai]) ; vrai si elle tombe EXACTEMENT sur `debut`. Rend aussi le nombre de pas.
func rlocChaine(pay []byte, pos, debut int, w *World, cfg FrameConfig) (bool, int) {
	essai := cfg
	essai.Obs = nil
	extra := motFacultatifDEnTete(cfg)
	snap := w.Snapshot()
	defer w.Restore(snap)
	n := 0
	for ; n < plafondChaineDeTete && pos < debut; n++ {
		fin, ok := pasDEssai(pay, pos, extra, w, essai)
		if !ok || fin <= pos {
			return false, n
		}
		pos = fin
	}
	return pos == debut, n
}

// rlocControle : le controle de l oracle sur les paquets que la signature du slot 123 localise.
type rlocControle struct {
	max, faits int
	t          cmTables
}

// essayer rend la classe du controle d un paquet localise en `s123` ("" au-dela du plafond).
func (c *rlocControle) essayer(pay []byte, w *World, cfg FrameConfig, s123 int) string {
	if c.faits >= c.max {
		return ""
	}
	c.faits++
	r := rlocPremierQuiFerme(pay, w, cfg, s123+1)
	switch {
	case r.debut < 0 && r.essais >= rlocPlafondEssais:
		return "aucun depart ne ferme (plafond d essais atteint)"
	case r.debut < 0:
		return "aucun depart ne ferme jusqu a la signature"
	case r.debut == s123:
		return "le premier depart qui ferme EST la signature"
	}
	ok, n := rlocChaine(pay, r.debut, s123, w, cfg)
	if ok {
		return fmt.Sprintf("depart anterieur, chaine coherente jusqu a la signature (%s records)", rlocClasseN(n))
	}
	return "depart anterieur, chaine NON coherente avec la signature"
}

// rlocClasseN : classe de taille d une chaine.
func rlocClasseN(n int) string {
	switch {
	case n <= 3:
		return "1-3"
	case n <= 10:
		return "4-10"
	}
	return "> 10"
}

// rlocClasseEssais : classe du nombre de marches d essai de l oracle.
func rlocClasseEssais(n int) string {
	switch {
	case n == 0:
		return "0 essai"
	case n <= 3:
		return "1-3 essais"
	case n <= 16:
		return "4-16 essais"
	case n < rlocPlafondEssais:
		return "17-255 essais"
	}
	return "plafond de 256 essais"
}
