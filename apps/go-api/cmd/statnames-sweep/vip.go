package main

// vip.go — CONFRONTATION VIP : le statborg replique-t-il `TimesSelectedAsVip` par joueur ?
//
// AUCUN film n'est decode ici (comme confront.go) : entree = TSV du balayage + oracle JSON
// fige. La difference avec la confrontation Oddball (moities disjointes recherche/verif) est
// le CORPUS MINCE : 3 films VIP, aucun split franc possible.
//
// # LE TEMOIN CORRIGE (2026-08-27, `VIP_COURONNE_PROTOCOLE.md`)
//
// La premiere version gate-ait sur un TEMOIN PERMUTE (permutation cyclique de l'affectation
// xuid -> oracle) `<= 20 %`. Il etait INAPTE, et c'est mesure : `TimesSelectedAsVip` est un
// compteur a FAIBLE VARIANCE (six joueurs a « 2 » sur huit dans un film), donc l'accord attendu
// sous permutation vaut la self-similarite de l'oracle `sum_v p_v^2` (~34-62 %) et NE PEUT PAS
// descendre sous 20 %, quelle que soit la justesse du comp. Le permute mesurait la
// self-similarite de la donnee, pas la discrimination du comp.
//
// On le remplace par le PLANCHER ANALYTIQUE correct : `plancher(f) = sum_v p_v(f)^2` (le null,
// calcule depuis l'oracle), et on exige `accord_signal - plancher >= 0,30`. Les seuils d'accord
// (`>= 90 %`), la stabilite 3/3 et le temoin decale de la somme-film (immune au pont, deja 0)
// sont INCHANGES. Ce n'est pas un abaissement : c'est un test de discrimination que le permute
// ne pouvait structurellement pas offrir. Le permute reste IMPRIME comme diagnostic, non gating.
//
// Les loaders (loadSweep, loadOracle), l'encodage (encode) et l'inventaire des emplacements
// (allKeys) sont ceux de confront.go — aucune seconde copie.

import (
	"fmt"
	"sort"
)

const (
	// vipAccordMin / vipNonNullesMin : seuils du protocole §3.5, INCHANGES.
	vipAccordMin    = 0.90
	vipNonNullesMin = 3
	// vipMargeMin : marge exigee entre l'accord du signal et le plancher analytique du null
	// (`sum p_v^2`). Seuil GENERIQUE de « domine clairement le null » : au-dessus de la
	// granularite d'echantillonnage sur 8 paires (12,5 pp), fixe par la structure du compteur
	// (comp parfait = 100 %, null = 34-62 %), jamais regle sur le resultat observe.
	vipMargeMin      = 0.30
	vipFilmsPourGate = 2 // >= 2 des 3 films
)

// vipPrimary / vipSecondary : la cible principale et les cibles secondaires (entiers additifs,
// encodage [n]). Les durees et le MAX sont HORS gate.
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
	fmt.Printf("CONFRONTATION VIP (temoin corrige) : films %v · gate accord >= %.0f %% ET "+
		"accord - plancher(sum p_v^2) >= %.0f pp sur >= %d/3, stabilite 3/3, somme-film decale = 0, "+
		">= %d paires non nulles/film\n",
		a.films, 100*vipAccordMin, 100*vipMargeMin, vipFilmsPourGate, vipNonNullesMin)
	// La somme-film (immune au pont) est calculee AVANT les verdicts : le temoin decale = 0 est
	// une clause du gate corrige, et il faut le connaitre pour statuer le primaire.
	decale := map[string]int{}
	for _, col := range append([]string{vipPrimary}, vipSecondary...) {
		decale[col] = vipSommeFilm(sw, oracle, col, a.films)
	}
	vipConfrontColonne(sw, oracle, vipPrimary, a.films, true, decale[vipPrimary] == 0)
	for _, col := range vipSecondary {
		vipConfrontColonne(sw, oracle, col, a.films, false, decale[col] == 0)
	}
	return nil
}

// vipEmplacement porte l'accord par film d'un emplacement pour une colonne.
type vipEmplacement struct {
	k       slotKey
	parFilm map[string]accord // film -> accord signal
	temoin  map[string]accord // film -> accord temoin permute (DIAGNOSTIC, non gating)
}

// nFilmsAuSeuil compte les films ou le signal atteint le seuil d'accord (exactitude seule).
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

// vipPlancherFilm rend le plancher analytique du null (`sum_v p_v^2`) pour une colonne et un
// film : la self-similarite de l'oracle sur les joueurs confrontables. C'est l'accord
// par-joueur attendu de toute affectation decorrelee de la vraie — le null CORRECT d'un
// compteur additif a faible variance (le temoin permute, lui, ne peut structurellement pas
// descendre sous ce plancher, cf. VIP_COURONNE_PROTOCOLE.md §1).
func vipPlancherFilm(sw sweepData, oracle oracleData, col, film string) float64 {
	counts := map[int64]int{}
	n := 0
	for _, x := range vipConfrontables(sw, oracle, film) {
		v, ok := encode(oracle[film][x][col], "n")
		if !ok {
			continue
		}
		counts[v]++
		n++
	}
	if n == 0 {
		return 0
	}
	var s float64
	for _, c := range counts {
		p := float64(c) / float64(n)
		s += p * p
	}
	return s
}

// vipConfrontColonne mesure tous les emplacements pour une colonne, elit le candidat et rend
// le verdict sous le gate CORRIGE (accord + marge sur le plancher + stabilite + decale). Le
// temoin permute est imprime comme DIAGNOSTIC. `decaleOK` = (temoin decale somme-film == 0).
func vipConfrontColonne(sw sweepData, oracle oracleData, col string, films []string, primaire, decaleOK bool) {
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
	floors := map[string]float64{}
	nGate := 0
	for _, f := range films {
		floors[f] = vipPlancherFilm(sw, oracle, col, f)
		ac := best.parFilm[f]
		if ac.taux() >= vipAccordMin && ac.nonNulles >= vipNonNullesMin &&
			ac.taux()-floors[f] >= vipMargeMin {
			nGate++
		}
	}
	fmt.Printf("COLONNE %s : candidat %s — %d/3 film(s) au gate corrige, accord moyen %.1f %%\n",
		col, best.k, nGate, 100*best.moyenne())
	for _, f := range films {
		ac := best.parFilm[f]
		fmt.Printf("   %s : signal %s ; plancher %.1f %% ; marge %.1f pp ; "+
			"[diag] permute %s\n",
			f, ac, 100*floors[f], 100*(ac.taux()-floors[f]), best.temoin[f])
	}
	stable := vipStabilite(emps, best.k, films)
	vipVerdict(col, best, nGate, stable, decaleOK, primaire)
}

// vipVerdict ecrit le verdict nomme d'une colonne sous le gate corrige.
func vipVerdict(col string, best vipEmplacement, nGate int, stable, decaleOK, primaire bool) {
	gate := nGate >= vipFilmsPourGate && stable && decaleOK
	label := "NE REPLIQUE PAS"
	if gate {
		label = "REPLIQUE"
	}
	fmt.Printf("VERDICT %s : le statborg %s %s — %s ; %d/3 au gate, stabilite 3/3 %v, "+
		"somme-film decale=0 %v\n",
		col, label, col, best.k, nGate, stable, decaleOK)
	if !primaire {
		fmt.Printf("   (%s : cible SECONDAIRE — anti-aliasing a verifier a la main : comp distinct de "+
			"2 A/3 A et valeur par slot <= comp generique)\n", col)
	}
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
// `shift` > 0 applique la permutation cyclique du TEMOIN DIAGNOSTIC (chaque xuid recoit
// l'oracle du suivant, ordre trie).
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
// decale = re-appariement cyclique film -> total, 0 faux candidat exige (§3.6). Rend le nombre
// de faux candidats du temoin decale (0 = clause du gate corrige tenue).
func vipSommeFilm(sw sweepData, oracle oracleData, col string, films []string) int {
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
		return 0
	}
	fmt.Printf("SOMME-FILM %s : candidat %s — signal %d/%d film(s), temoin decale %d (exige 0)\n",
		col, best.k, best.signal, best.disponib, best.temoin)
	return best.temoin
}

// vipSommeSlots somme la valeur finale d'un comp sur TOUS les slots joueurs pontes d'un film.
func vipSommeSlots(sw sweepData, film string, k slotKey) int64 {
	var s int64
	for _, v := range sw.finals[film][k] {
		s += v
	}
	return s
}
