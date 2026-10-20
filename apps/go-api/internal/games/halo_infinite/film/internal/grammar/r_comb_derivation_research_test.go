//go:build research && campagne_overlay

package grammar

// r_comb_derivation_research_test.go — compagnon de `r_comb_research_test.go` (RECHERCHE R-COMB,
// 2026-10-02) : l INTERACTION L1 x L9. Dans `TestRComb`, l oracle L1 de la combinaison est derive
// d une marche ou L9 est deja pose (ordre D-RI : la marche d image-cle avant les naissances). Ce
// test separe les deux causes possibles d une interaction :
//
//	derivation  l oracle L1 derive SANS L9, pose avec L9 (« L1(sans L9)+L9 ») ;
//	liaisons    la difference entre l oracle L1 derive avec et sans L9 : liaisons communes, propres
//	            a chaque derivation, meme cle sous un autre archetype ; et la marche
//	            « L9 + liaisons communes seulement ».
//
// Memes variables que `TestRComb`.
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 240m \
//	  -run '^TestRCombDerivation$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rcdDiff : la difference de deux oracles, liaison par liaison (cle de paquet, eid).
func rcdDiff(avec, sans map[[2]int][]cmLiaison) (communes map[[2]int][]cmLiaison, nC, nAvec, nSans, nAutreTi int, propres []string) {
	communes = map[[2]int][]cmLiaison{}
	type k3 struct {
		c   [2]int
		eid uint32
	}
	idx := map[k3]uint32{}
	for c, ls := range sans {
		for _, l := range ls {
			idx[k3{c, l.eid}] = l.ti
		}
	}
	vu := map[k3]bool{}
	for c, ls := range avec {
		for _, l := range ls {
			k := k3{c, l.eid}
			vu[k] = true
			ti, ok := idx[k]
			switch {
			case ok && ti == l.ti:
				nC++
				communes[c] = append(communes[c], l)
			case ok:
				nAutreTi++
				propres = append(propres, fmt.Sprintf("autre archetype : chunk %d paquet %d eid %#x ti %d (sans L9 : ti %d)", c[0], c[1], l.eid, l.ti, ti))
			default:
				nAvec++
				propres = append(propres, fmt.Sprintf("avec L9 seulement : chunk %d paquet %d eid %#x ti %d", c[0], c[1], l.eid, l.ti))
			}
		}
	}
	for k, ti := range idx {
		if !vu[k] {
			nSans++
			propres = append(propres, fmt.Sprintf("sans L9 seulement : chunk %d paquet %d eid %#x ti %d", k.c[0], k.c[1], k.eid, ti))
		}
	}
	sort.Strings(propres)
	return communes, nC, nAvec, nSans, nAutreTi, propres
}

// TestRCombDerivation joue l interaction L1 x L9 sur les films de CAMPAGNE_FILMS.
func TestRCombDerivation(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	var cat *profile.MapQuantCatalog
	if len(cartes) > 0 {
		var err error
		if cat, err = profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE")); err != nil {
			t.Fatalf("catalogue : %v", err)
		}
	}
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tcomposants\tconfig\tliaisons_l1\tliaisons_l9\tpaquets\tfermes\tfermes_sains\tutiles_fermes\t" +
		"utiles_fermes_sains\tutiles_lus\thors_cadre\tg_ref\tp_ref\tgs_ref\tps_ref"}
	diffs := []string{"film\tbuild\tcomposants\tcommunes\tavec_L9_seulement\tsans_L9_seulement\tautre_archetype"}
	detail := []string{"film\tbuild\tcomposants\tliaison"}
	defer rcLever()
	for _, id := range films {
		garde := filmproc.Arm("campagne/r-comb-derivation", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		x := &rcFilm{contexte: "instruments"}
		var f *cmFilm
		ok := false
		if carte := cartes[id]; carte != "" {
			entry, err := cat.Lookup(carte)
			if err == nil {
				f, ok = b2pOuvrirProduction(t, racine, id, entry, utiles)
				x.bornes = b2pBornes(entry.Module)
			}
		} else {
			f, ok = cmOuvrir(t, racine, id, utiles)
		}
		if !ok {
			garde.Disarm()
			continue
		}
		x.f, x.b = f, cmLireBlocs(f)
		var ref *rcObs
		for _, jeu := range []struct {
			nom string
			k   rcComps
		}{{"aucun", rcComps{}}, {"L8+L2+L6a+L6b", rcComps{l8: true, l2: true, l6a: len(x.bornes) > 0, l6b: true}}} {
			oA, colA, diagA := x.marcher(jeu.k, nil, true, true)
			if ref == nil {
				ref = oA
			}
			l9, l1Sans := diagA.imageCle, rcL1(colA)
			_, colB, _ := x.marcher(jeu.k, l9, true, false)
			l1Avec := rcL1(colB)
			communes, nC, nAvec, nSans, nAutre, propres := rcdDiff(l1Avec, l1Sans)
			diffs = append(diffs, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d", id, f.build, jeu.nom, nC, nAvec, nSans, nAutre))
			for _, p := range propres {
				detail = append(detail, fmt.Sprintf("%s\t%s\t%s\t%s", id, f.build, jeu.nom, p))
			}
			for _, v := range []struct {
				nom    string
				oracle map[[2]int][]cmLiaison
				nl1    int
			}{{"L1(avec L9)+L9", cmUnion(l9, l1Avec), b3Compter(l1Avec)}, {"L1(sans L9)+L9", cmUnion(l9, l1Sans), b3Compter(l1Sans)},
				{"L1(communes)+L9", cmUnion(l9, communes), nC}} {
				o, _, _ := x.marcher(jeu.k, v.oracle, false, false)
				c := rcComparer(o.statut, ref.statut)
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, jeu.nom, v.nom,
					v.nl1, b3Compter(l9), o.a.paquets, o.a.fermes, o.a.sains, o.a.utilesFermes, o.a.utilesSains, o.a.utilesLus, o.a.horsCadre,
					c.g, c.p, c.gs, c.ps))
			}
		}
		t.Logf("%s %s : pic %d Mio", id, f.build, garde.Peak()>>20)
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "r_comb_derivation.tsv", lignes)
	b2Ecrire(t, sortie, "r_comb_derivation_liaisons.tsv", diffs)
	b2Ecrire(t, sortie, "r_comb_derivation_detail.tsv", detail)
}
