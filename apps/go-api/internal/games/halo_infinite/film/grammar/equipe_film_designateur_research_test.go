package grammar

// equipe_film_designateur_research_test.go — PHASE 3, LA MESURE ET SES CONTROLES.
//
// LE CANDIDAT, ET D'OU IL VIENT (chaine 1, l'executable, aucune mesure sur les films).
// Le descripteur de composant `managed-player-team-designator-component` est a
// `0x143d08ad0` : le thunk de nom est en `+0x08` (`0x141177eb0` ->
// `"managed-player-team-designator-component"` @`0x143c953c0`), l'ECRIVAIN en `+0x18`
// (`0x142edbd3c`) et le LECTEUR en `+0x30` (`0x140f581e8`). Le lecteur ne fait qu'un appel :
// `FUN_1407ef804`, dont le desassemblage donne la largeur SANS ambiguite
// (`ADD dword ptr [RCX+0x2c],0x4` = 4 bits) et la convention de valeur
// (`SHR R9,0x3c ; DEC R9B` = la valeur STOCKEE vaut le brut MOINS 1, donc brut 0 = -1 =
// aucune equipe). La table de noms est l'enumeration de script `mp_team_designator`
// (descripteur `0x1445c0c00`, 9 entrees a `0x144723da0`) : `First, Second, Third, Fourth,
// Fifth, Sixth, Seventh, Eighth, Neutral`. PREDICTION testable : `brut = team_id + 1`.
//
// CE QUI RESTE A MESURER, ET C'EST TOUT L'OBJET DE CE FICHIER : la POSITION du champ dans le
// record d'image-cle. L'en-tete par entite est une VARIABLE du dossier (64 / 108 / 47 selon la
// source) et le corps du record commence par un bloc d'etat par defaut de largeur non tranchee.
// Le decalage est donc CHERCHE par balayage, et departage par des controles ecrits AVANT la
// mesure.
//
// LES CONTROLES, ECRITS AVANT DE MESURER :
//
//	C1 STABILITE — l'equipe d'un joueur ne change pas pendant un match : le vecteur des
//	   valeurs doit etre IDENTIQUE sur TOUS les paquets de type 2 retenus du film.
//	C2 EQUILIBRE — oracle INTERNE : un champ d'equipe partage le roster d'une partie a deux
//	   equipes en deux moities de taille egale (4-4 en arene, 12-12 en Grande bataille).
//	C3 DOMAINE — toutes les valeurs dans 1..9 (designateur 0..8 plus un).
//	C4 PLANCHER DE BRUIT — le NOMBRE de decalages qui passent C1+C2+C3 par hasard, MESURE sur
//	   le flux reel (methode, regle 4), pas calcule. Publie par film.
//	C5 INTER-FILMS — le decalage retenu doit etre LE MEME sur tous les films. Un decalage qui
//	   change de film en film est un artefact du balayage.
//	C6 TEMOIN NEGATIF NATUREL — un film de match SANS equipe (FFA) ne doit PAS produire de
//	   partage equilibre a deux valeurs.
//
// REGLE DE SELECTION DES PAQUETS, elle aussi ecrite d'avance : `equipePaquetsModaux` ne garde
// que les paquets dont le nombre d'entites ti=9 vaut le mode du film. Un joueur qui arrive ou
// part change ce nombre, et comparer un vecteur de 8 a un vecteur de 7 n'a pas de sens. Les
// paquets ecartes sont comptes et publies.
//
// Garde CHUNK00_FILMS. Oracle externe : voir `equipe_film_oracle_research_test.go`.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// equipeDecalageMesure est le decalage, en bits depuis le debut du record d'image-cle, ou le
// champ de 4 bits a ete MESURE (cf. TestEquipeFilmDecalage et la note du 2026-09-12). Il n'est
// pas porte depuis l'executable : la largeur de l'en-tete par entite et celle du bloc d'etat
// par defaut de ti=9 ne sont pas tranchees par le dossier.
const equipeDecalageMesure = 186

// equipeVecteur est la suite des valeurs BRUTES de 4 bits lues a un decalage donne, dans
// l'ordre des slots des records ti=9.
type equipeVecteur []int

// equipePaire est un couple (decalage, nombre de films) pour le releve de convergence C5.
type equipePaire struct{ d, n int }

// equipeFormeEquilibree dit si le vecteur prend EXACTEMENT deux valeurs distinctes, en parts
// egales (controle C2). Un vecteur de taille impaire ne peut pas le satisfaire.
func equipeFormeEquilibree(v equipeVecteur) bool {
	if len(v) < 2 || len(v)%2 != 0 {
		return false
	}
	comptes := map[int]int{}
	for _, x := range v {
		comptes[x]++
	}
	if len(comptes) != 2 {
		return false
	}
	for _, c := range comptes {
		if c != len(v)/2 {
			return false
		}
	}
	return true
}

// equipeDomaineValide dit si toutes les valeurs tiennent dans 1..9 (controle C3).
func equipeDomaineValide(v equipeVecteur) bool {
	for _, x := range v {
		if x < 1 || x > 9 {
			return false
		}
	}
	return true
}

// equipeEgal compare deux vecteurs (controle C1).
func equipeEgal(a, b equipeVecteur) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equipeVecteurA lit le vecteur des designateurs bruts au decalage d, pour un paquet.
func equipeVecteurA(pq equipePaquet, d int) equipeVecteur {
	v := make(equipeVecteur, 0, len(pq.Recs))
	for _, r := range pq.Recs {
		v = append(v, int(kfReadBits(pq.Pay, r.Bit+d, equipeDesignatorBits)))
	}
	return v
}

// equipeLongueurCommune rend la longueur de record si TOUS les records ti=9 ont la meme, et 0
// sinon. Sans longueur commune, un balayage par decalage relatif n'a pas de sens.
func equipeLongueurCommune(pqs []equipePaquet) int {
	l := -1
	for _, pq := range pqs {
		for _, r := range pq.Recs {
			if l < 0 {
				l = r.LenBits
			} else if r.LenBits != l {
				return 0
			}
		}
	}
	if l < 0 {
		return 0
	}
	return l
}

// equipeVue rassemble ce qu'un film offre au balayage, apres la regle des paquets modaux.
type equipeVue struct {
	Paquets  []equipePaquet
	Long     int // longueur commune d'un record ti=9, en bits
	Card     int // nombre d'entites ti=9 (le mode)
	Ecartes  int // paquets ecartes par la regle modale
	Bruts    int // paquets portant ti=9 avant la regle modale
	Longueur bool
}

// equipePrepare applique la regle des paquets modaux et mesure longueur et cardinal.
func equipePrepare(dir string) (equipeVue, error) {
	pqs, err := equipePaquetsArch(dir, equipeTI)
	if err != nil {
		return equipeVue{}, err
	}
	if len(pqs) == 0 {
		return equipeVue{}, fmt.Errorf("aucun paquet portant ti=%d", equipeTI)
	}
	bruts := len(pqs)
	gardes, ecartes := equipePaquetsModaux(pqs)
	v := equipeVue{Paquets: gardes, Ecartes: ecartes, Bruts: bruts, Card: len(gardes[0].Recs)}
	v.Long = equipeLongueurCommune(gardes)
	v.Longueur = v.Long > 0
	return v, nil
}

// equipeDecalagesRetenus balaie les decalages 0..L-4 et rend ceux qui passent C1 (stabilite
// sur tous les paquets retenus), C2 (partage equilibre a deux valeurs) et C3 (domaine 1..9),
// avec le vecteur observe, plus le compte de decalages STABLES (base du plancher C4).
func equipeDecalagesRetenus(v equipeVue) (map[int]equipeVecteur, int) {
	retenus := map[int]equipeVecteur{}
	stables := 0
	for d := 0; d+equipeDesignatorBits <= v.Long; d++ {
		ref := equipeVecteurA(v.Paquets[0], d)
		stable := true
		for _, pq := range v.Paquets[1:] {
			if !equipeEgal(ref, equipeVecteurA(pq, d)) {
				stable = false
				break
			}
		}
		if !stable {
			continue
		}
		stables++
		if equipeDomaineValide(ref) && equipeFormeEquilibree(ref) {
			retenus[d] = ref
		}
	}
	return retenus, stables
}

// TestEquipeFilmDecalage est la mesure centrale : par film, les decalages qui passent
// C1+C2+C3, avec le plancher de bruit (C4) et la convergence inter-films (C5).
func TestEquipeFilmDecalage(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	commun := map[int]int{}
	films := 0
	for _, dir := range dirs {
		v, err := equipePrepare(dir)
		if err != nil {
			t.Logf("%s : %v", filepath.Base(dir), err)
			continue
		}
		if !v.Longueur {
			t.Logf("%s : longueur de record NON commune — ECARTE", filepath.Base(dir))
			continue
		}
		films++
		retenus, stables := equipeDecalagesRetenus(v)
		ds := make([]int, 0, len(retenus))
		for d := range retenus {
			ds = append(ds, d)
			commun[d]++
		}
		sort.Ints(ds)
		t.Logf("%s : %d/%d paquets retenus (%d ecartes), %d records de %d bits ; "+
			"C1 %d decalages stables sur %d ; C2+C3 : %d retenus %v",
			filepath.Base(dir), len(v.Paquets), v.Bruts, v.Ecartes, v.Card, v.Long,
			stables, v.Long-equipeDesignatorBits+1, len(retenus), ds)
		for _, d := range ds {
			t.Logf("    decalage %4d : bruts %v", d, retenus[d])
		}
	}
	if films == 0 {
		t.Skip("aucun film exploitable")
	}
	var tous []equipePaire
	for d, n := range commun {
		tous = append(tous, equipePaire{d, n})
	}
	sort.Slice(tous, func(i, j int) bool {
		if tous[i].n != tous[j].n {
			return tous[i].n > tous[j].n
		}
		return tous[i].d < tous[j].d
	})
	t.Logf("C5 CONVERGENCE sur %d films : %s", films, equipePairesTexte(tous, 12))
}

// equipePairesTexte rend les n premieres paires (decalage, films) en une ligne.
func equipePairesTexte(ps []equipePaire, n int) string {
	var b strings.Builder
	for i, p := range ps {
		if i >= n {
			b.WriteString(" ...")
			break
		}
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "d=%d:%dfilms", p.d, p.n)
	}
	return b.String()
}

// TestEquipeFilmBrutAuDecalage lit le vecteur au decalage MESURE, sans exiger aucune forme.
// C'est le releve qui sert de temoin negatif naturel (C6) : un film FFA doit y montrer autre
// chose qu'un partage equilibre a deux valeurs, et c'est mesurable sans oracle externe.
func TestEquipeFilmBrutAuDecalage(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			t.Logf("%s : ECARTE (%v)", filepath.Base(dir), err)
			continue
		}
		ref := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		// Une entite dont le designateur CHANGE d'une image-cle a l'autre est un changement
		// d'equipe reel (equilibrage, retour de joueur). On les compte au lieu de rendre un
		// simple booleen : le compte est la mesure, le booleen n'est qu'un seuil.
		bouge := map[int]bool{}
		for _, pq := range v.Paquets[1:] {
			cur := equipeVecteurA(pq, equipeDecalageMesure)
			for i := range ref {
				if i < len(cur) && cur[i] != ref[i] {
					bouge[i] = true
				}
			}
		}
		comptes := map[int]int{}
		for _, x := range ref {
			comptes[x]++
		}
		t.Logf("%s : %d entites ti=%d ; d=%d bruts %v ; %d valeurs distinctes %s ; "+
			"%d entites changent sur %d images-cles (slots identiques partout : %v) ; "+
			"equilibre=%v domaine=%v",
			filepath.Base(dir), v.Card, equipeTI, equipeDecalageMesure, ref,
			len(comptes), equipeHistoTexte(comptes), len(bouge), len(v.Paquets),
			equipeSlotsConstants(v), equipeFormeEquilibree(ref), equipeDomaineValide(ref))
	}
}

// equipeSlotsConstants dit si la SUITE DES SLOTS des entites ti=9 est la meme sur tous les
// paquets retenus.
//
// C'EST LA QUESTION QUI DEPARTAGE DEUX LECTURES d'un designateur qui bouge d'une image-cle a
// l'autre : soit le joueur a change d'equipe, soit la i-eme entite ti=9 n'est plus le meme
// joueur (slot reattribue, arrivee, depart). Sans cette mesure, les deux sont indistinguables
// et l'une des deux serait choisie sans preuve (methode, erreur E).
func equipeSlotsConstants(v equipeVue) bool {
	if len(v.Paquets) == 0 {
		return false
	}
	ref := make([]int, 0, len(v.Paquets[0].Recs))
	for _, r := range v.Paquets[0].Recs {
		ref = append(ref, r.Slot)
	}
	for _, pq := range v.Paquets[1:] {
		if len(pq.Recs) != len(ref) {
			return false
		}
		for i, r := range pq.Recs {
			if r.Slot != ref[i] {
				return false
			}
		}
	}
	return true
}
