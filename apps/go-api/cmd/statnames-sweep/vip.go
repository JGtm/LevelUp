package main

// vip.go — CONFRONTATION VIP : le statborg replique-t-il `TimesSelectedAsVip` par joueur ?
//
// AUCUN film n'est decode ici (comme confront.go) : entree = TSV du balayage + oracle JSON
// fige. La difference avec la confrontation Oddball (moities disjointes recherche/verif) est
// le CORPUS MINCE : 3 films VIP, aucun split franc possible. Le protocole
// (`PROTOCOLE_RESOLUTION_VIP_ASSAUT.md` R2, recopie de `PROTOCOLE_REMESURE_ODDBALL_VIP.md`
// §3.5/§3.6) remplace le split par TROIS garde-fous : accord >= 90 % sur >= 2/3 films,
// STABILITE (meme comp meilleur sur 3/3), et TEMOIN permute <= 20 % par film ; plus un test
// SOMME-FILM immune au pont. Les loaders (loadSweep, loadOracle), l'encodage (encode) et
// l'inventaire des emplacements (allKeys) sont ceux de confront.go — aucune seconde copie.

import (
	"fmt"
	"sort"
)

const (
	// vipAccordMin / vipNonNullesMin : seuils du protocole §3.5, recopies sans modification.
	vipAccordMin     = 0.90
	vipNonNullesMin  = 3
	vipTemoinMax     = 0.20
	vipFilmsPourGate = 2 // >= 2 des 3 films
)

// vipCols : la cible principale et les cibles secondaires (entiers additifs, encodage [n]).
// Les durees (TimeAsVip, LongestTimeAsVip) et le MAX (MaxKillingSpreeAsVip) sont HORS gate.
var vipPrimary = "TimesSelectedAsVip"
var vipSecondary = []string{"KillsAsVip", "VipKills"}

// runConfrontVIP charge les entrees et rend le verdict VIP (primaire + secondaires + somme-film).
func runConfrontVIP(a runArgs) error {
	if a.sweepPath == "" || a.oraclePath == "" || len(a.films) == 0 {
		return fmt.Errorf("-confront -vip exige -sweep, -oracle et -films (les 3 films VIP)")
	}
	sw, err := loadSweep(a.sweepPath)
	if err != nil {
		return err
	}
	oracle, err := loadOracle(a.oraclePath)
	if err != nil {
		return err
	}
	fmt.Printf("CONFRONTATION VIP : films %v · seuils accord >= %.0f %% sur >= %d/3, stabilite 3/3, "+
		"temoin permute <= %.0f %%, >= %d paires non nulles/film\n",
		a.films, 100*vipAccordMin, vipFilmsPourGate, 100*vipTemoinMax, vipNonNullesMin)
	vipConfrontColonne(sw, oracle, vipPrimary, a.films, true)
	for _, col := range vipSecondary {
		vipConfrontColonne(sw, oracle, col, a.films, false)
	}
	vipSommeFilm(sw, oracle, vipPrimary, a.films)
	return nil
}

// vipEmplacement porte l'accord par film d'un emplacement pour une colonne.
type vipEmplacement struct {
	k       slotKey
	parFilm map[string]accord // film -> accord signal
	temoin  map[string]accord // film -> accord temoin permute
}

// nFilmsAuSeuil compte les films ou le signal atteint le seuil (accord + paires non nulles).
func (e vipEmplacement) nFilmsAuSeuil() int {
	n := 0
	for _, ac := range e.parFilm {
		if ac.taux() >= vipAccordMin && ac.nonNulles >= vipNonNullesMin {
			n++
		}
	}
	return n
}

// moyenne rend l'accord moyen sur les films confrontes (departage des candidats).
func (e vipEmplacement) moyenne() float64 {
	if len(e.parFilm) == 0 {
		return 0
	}
	s := 0.0
	for _, ac := range e.parFilm {
		s += ac.taux()
	}
	return s / float64(len(e.parFilm))
}

// vipConfrontColonne mesure tous les emplacements pour une colonne, elit le candidat et
// rend le verdict avec ses garde-fous. `primaire` declenche le gate complet ; les secondaires
// ajoutent la note anti-aliasing.
func vipConfrontColonne(sw sweepData, oracle oracleData, col string, films []string, primaire bool) {
	keys := allKeys(sw)
	emps := make([]vipEmplacement, 0, len(keys))
	for _, k := range keys {
		e := vipEmplacement{k: k, parFilm: map[string]accord{}, temoin: map[string]accord{}}
		for _, f := range films {
			e.parFilm[f] = vipAccordFilm(sw, oracle, k, col, f, 0)
			e.temoin[f] = vipAccordFilm(sw, oracle, k, col, f, 1)
		}
		emps = append(emps, e)
	}
	sort.SliceStable(emps, func(i, j int) bool {
		if emps[i].nFilmsAuSeuil() != emps[j].nFilmsAuSeuil() {
			return emps[i].nFilmsAuSeuil() > emps[j].nFilmsAuSeuil()
		}
		if emps[i].moyenne() != emps[j].moyenne() {
			return emps[i].moyenne() > emps[j].moyenne()
		}
		if emps[i].k.comp != emps[j].k.comp {
			return emps[i].k.comp < emps[j].k.comp
		}
		return emps[i].k.side < emps[j].k.side
	})
	if len(emps) == 0 {
		fmt.Printf("COLONNE %s : aucun emplacement balaye\n", col)
		return
	}
	best := emps[0]
	fmt.Printf("COLONNE %s : candidat %s — %d/3 film(s) au seuil, accord moyen %.1f %%\n",
		col, best.k, best.nFilmsAuSeuil(), 100*best.moyenne())
	for _, f := range films {
		fmt.Printf("   %s : signal %s ; temoin permute %s\n", f, best.parFilm[f], best.temoin[f])
	}
	stable := vipStabilite(emps, best.k, films)
	temoinOK := true
	for _, f := range films {
		if best.temoin[f].taux() > vipTemoinMax {
			temoinOK = false
		}
	}
	vipVerdict(col, best, stable, temoinOK, films, primaire)
}

// vipVerdict ecrit le verdict nomme d'une colonne.
func vipVerdict(col string, best vipEmplacement, stable, temoinOK bool, films []string, primaire bool) {
	gate := best.nFilmsAuSeuil() >= vipFilmsPourGate && stable && temoinOK
	label := "NE REPLIQUE PAS"
	if gate {
		label = "REPLIQUE"
	}
	fmt.Printf("VERDICT %s : le statborg %s %s — %s ; %d/3 au seuil, stabilite 3/3 %v, temoin OK %v\n",
		col, label, col, best.k, best.nFilmsAuSeuil(), stable, temoinOK)
	if !primaire {
		fmt.Printf("   (%s : cible SECONDAIRE — anti-aliasing a verifier a la main : comp distinct de "+
			"2 A/3 A et valeur par slot <= comp generique)\n", col)
	}
	_ = films
}

// vipStabilite dit si l'emplacement k est le MEILLEUR (a egalite pres) sur CHACUN des films —
// le garde-fou du corpus mince (§3.5). Un comp meilleur sur 2 films mais supplante sur le 3e
// est rejete.
func vipStabilite(emps []vipEmplacement, k slotKey, films []string) bool {
	for _, f := range films {
		var meilleur float64
		for _, e := range emps {
			if e.parFilm[f].taux() > meilleur {
				meilleur = e.parFilm[f].taux()
			}
		}
		var kt float64
		for _, e := range emps {
			if e.k == k {
				kt = e.parFilm[f].taux()
			}
		}
		if kt < meilleur {
			return false
		}
	}
	return true
}

// vipAccordFilm compte les paires (joueur) d'UN film pour un emplacement et une colonne.
// `shift` > 0 applique la permutation cyclique du TEMOIN (chaque xuid recoit l'oracle du
// suivant, ordre trie), l'equivalent statborg de l'attribution aleatoire des comps.
func vipAccordFilm(sw sweepData, oracle oracleData, k slotKey, col, film string, shift int) accord {
	var out accord
	xuids := vipConfrontables(sw, oracle, film)
	n := len(xuids)
	for i, xuid := range xuids {
		src := xuids[(i+shift)%n]
		want, ok := encode(oracle[film][src][col], "n")
		if !ok {
			continue
		}
		out.paires++
		if want != 0 {
			out.nonNulles++
		}
		if sw.finals[film][k][xuid] == want {
			out.matches++
		}
	}
	return out
}

// vipConfrontables rend, tries, les xuids d'un film presents A LA FOIS dans l'oracle et dans
// le pont du balayage — l'ensemble sur lequel la permutation cyclique est fermee.
func vipConfrontables(sw sweepData, oracle oracleData, film string) []string {
	var out []string
	for xuid := range oracle[film] {
		if sw.joueurs[film][xuid] {
			out = append(out, xuid)
		}
	}
	sort.Strings(out)
	return out
}

// vipSommeFilm confronte la SOMME sur les slots joueurs de la valeur finale d'un comp au
// total-film de la colonne (immune au pont). Candidat si S == O sur >= 2/3 (O >= 1) ; temoin
// decale = re-appariement cyclique film -> total, 0 faux candidat exige (§3.6).
func vipSommeFilm(sw sweepData, oracle oracleData, col string, films []string) {
	totaux := map[string]int64{}
	for _, f := range films {
		var o int64
		for _, cols := range oracle[f] {
			v, ok := encode(cols[col], "n")
			if ok {
				o += v
			}
		}
		totaux[f] = o
	}
	keys := allKeys(sw)
	type res struct {
		k        slotKey
		signal   int
		temoin   int
		disponib int
	}
	var best res
	found := false
	for _, k := range keys {
		var r res
		r.k = k
		for i, f := range films {
			s := vipSommeSlots(sw, f, k)
			if totaux[f] >= 1 {
				r.disponib++
				if s == totaux[f] {
					r.signal++
				}
			}
			// temoin : total de l'AUTRE film (permutation cyclique du couple film->total).
			decale := totaux[films[(i+1)%len(films)]]
			if decale >= 1 && s == decale {
				r.temoin++
			}
		}
		if r.signal >= vipFilmsPourGate && (!found || r.signal > best.signal) {
			best, found = r, true
		}
	}
	fmt.Printf("SOMME-FILM %s : totaux %v\n", col, totaux)
	if !found {
		fmt.Printf("SOMME-FILM %s : aucun comp ne somme au total-film sur >= %d/3 films (NEGATIF)\n",
			col, vipFilmsPourGate)
		return
	}
	fmt.Printf("SOMME-FILM %s : candidat %s — signal %d/%d film(s), temoin decale %d (exige 0)\n",
		col, best.k, best.signal, best.disponib, best.temoin)
}

// vipSommeSlots somme la valeur finale d'un comp sur TOUS les slots joueurs pontes d'un film.
func vipSommeSlots(sw sweepData, film string, k slotKey) int64 {
	var s int64
	for _, v := range sw.finals[film][k] {
		s += v
	}
	return s
}
