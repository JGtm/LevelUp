//go:build research

package grammar

// campagne_bis3_research_test.go — MESURES BIS 3 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01) : LE
// PILOTE. Un film a la fois, sentinelle `filmproc` 4 Gio. Par film :
//
//	reference           la marche de la carte v2, avec la sonde M1 (`col`) et le diagnostic
//	                    des populations (`campagne_bis3_diag_research_test.go`) ;
//	oracle-NEW          l oracle des naissances de M1 (controle : MESURES_CIBLEES §2) ;
//	oracle-P1..P4       le meme oracle restreint aux eid d une population ;
//	oracle-P1-pont      les eid de P1 lies AU DEBUT de leur chunk avec l archetype du pont du bloc
//	                    du chunk (borne d une image-cle complete) ;
//	oracle-P6-<filtre>  les naissances trouvees a plus de 3 paquets, liees sous un filtre ;
//	largeur-<n>         la marche de reference sous `IDLowBits = n` (D-24), et la largeur que
//	                    `calibrateFrameConfig` (ScanMarchFacts) retiendrait.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 180m -run '^TestCampagneBis3Populations$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// b3Largeurs : les largeurs balayees (l espace de `calibrateFrameConfig`, 10..15).
var b3Largeurs = []int{10, 11, 12, 14, 15}

// TestCampagneBis3Populations joue les mesures bis 3 sur les films de CAMPAGNE_FILMS.
func TestCampagneBis3Populations(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := map[string][]string{
		"variantes": {"film\tbuild\tvariante\tliaisons\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\t" +
			"rejets_hors_datum\tgagnes\tperdus\tgagnes_contredits\tfermes_contredits"},
		"tables": {"film\tbuild\tmarche\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"},
		"cadre":  {"film\tbuild\tmesure\tvaleur"},
	}
	for _, id := range films {
		b3UnFilm(t, racine, id, utiles, out)
	}
	for nom, lignes := range out {
		b2Ecrire(t, sortie, "mb3_"+nom+".tsv", lignes)
	}
}

// b3Compter compte les liaisons d un oracle.
func b3Compter(m map[[2]int][]cmLiaison) int {
	n := 0
	for _, l := range m {
		n += len(l)
	}
	return n
}

// b3Restreindre garde les liaisons d un oracle dont l eid est dans la population.
func b3Restreindre(m map[[2]int][]cmLiaison, pop map[[2]uint32]bool) map[[2]int][]cmLiaison {
	out := map[[2]int][]cmLiaison{}
	for cle, ls := range m {
		for _, l := range ls {
			if pop[[2]uint32{uint32(cle[0]), l.eid}] { //nolint:gosec // numero de chunk
				out[cle] = append(out[cle], l)
			}
		}
	}
	return out
}

// b3UnFilm mesure un film.
func b3UnFilm(t *testing.T, racine, id string, utiles UsagesProduit, out map[string][]string) {
	garde := filmproc.Arm("campagne/bis3", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	f, ok := cmOuvrir(t, racine, id, utiles)
	if !ok {
		return
	}
	b := cmLireBlocs(f)
	wr := profile.QuantRangeCEBiped()
	cre, _, _ := ScanVehicleCreations(f.fc, &wr)
	col := nouveauCollecteur(f, b, cre)
	diag := b3NouveauDiag(f, b, true)
	jref := cmNouveauJuge(f, b, nil)
	rep, obs, _ := cmMarcher(f, cmVariante{}, cmMux{col, diag, jref})
	ligne := func(nom string, nl int, r FrameClosureReport, o *Observation, j *cmJuge) {
		out["variantes"] = append(out["variantes"], fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
			id, f.build, nom, nl, r.Paquets, r.PaquetsFermes, r.Utiles.RecordsFermes, r.Utiles.Records,
			r.Bloquants[CauseHorsCadre].Paquets, o.RejetsHorsDatum, j.gagnes, j.perdus, j.gagnesContredits, j.fermesContre))
	}
	tables := func(marche string, ts cmTables) {
		for nom, m := range ts {
			for _, cle := range cmCles(m) {
				x := m[cle]
				out["tables"] = append(out["tables"], fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d",
					id, f.build, marche, nom, cle, x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
			}
		}
	}
	ligne("reference", 0, rep, obs, jref)
	tables("reference", diag.t)
	type variante struct {
		nom string
		m   map[[2]int][]cmLiaison
	}
	vs := []variante{{"oracle-NEW", col.oracle}}
	for _, p := range []string{b3P1, b3P2, b3P3, b3P4} {
		vs = append(vs, variante{"oracle-" + p[:2], b3Restreindre(col.oracle, diag.pop[p])})
	}
	vs = append(vs, variante{"oracle-P1-pont", diag.imageCle})
	for _, fl := range b3Filtres {
		vs = append(vs, variante{"oracle-P6-" + fl, diag.etendu[fl]})
	}
	vs = append(vs, variante{"oracle-NEW+P1-pont", cmUnion(col.oracle, diag.imageCle)},
		variante{"oracle-NEW+P6-propre+pont+suivant", cmUnion(col.oracle, diag.etendu["propre+pont+suivant"])},
		variante{"oracle-NEW+P6-propre+alloc+suivant", cmUnion(col.oracle, diag.etendu["propre+alloc+suivant"])})
	for _, v := range vs {
		j := cmNouveauJuge(f, b, col.statut)
		j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
		r, o, _ := cmMarcher(f, cmVariante{oracle: v.m}, j)
		ligne(v.nom, b3Compter(v.m), r, o, j)
	}
	b3Largeur(f, b, col, jref, ligne, tables, out)
	t.Logf("%s %s : %d marches ; pic %d Mio, %s", id, f.build, len(vs)+len(b3Largeurs)+2, garde.Peak()>>20,
		time.Since(debut).Round(time.Second))
}

// b3Largeur : D-24 — le cardinal des blocs de type 1 (la largeur que le jeu lit,
// `FUN_1406d3140(_,_,7,_)` = ceil(log2(DAT_144706100))), la largeur que `calibrateFrameConfig`
// retiendrait, et la marche de reference sous chaque largeur balayee.
func b3Largeur(f *cmFilm, b *cmBlocs, col *cmCollecteur, jref *cmJuge,
	ligne func(string, int, FrameClosureReport, *Observation, *cmJuge), tables func(string, cmTables),
	out map[string][]string) {
	add := func(m, v string) {
		out["cadre"] = append(out["cadre"], fmt.Sprintf("%s\t%s\t%s\t%s", f.id, f.build, m, v))
	}
	card := map[int]int{}
	for _, e := range b.entrees {
		card[len(e)]++
	}
	var cs []string
	for n, k := range card {
		cs = append(cs, fmt.Sprintf("%d entrees x %d blocs (largeur %d)", n, k, b3Log2Haut(uint32(n)))) //nolint:gosec // cardinal
	}
	sort.Strings(cs)
	add("cardinal des blocs de type 1", strings.Join(cs, " ; "))
	add("plafond (plus grand slot alloue d un bloc)", fmt.Sprintf("%d", b.plafond))
	add("IDLowBits du cadre des instruments", fmt.Sprintf("%d", f.cfg.IDLowBits))
	kfs, deltas := marchPacketsOf(f.fc)
	cal, parDefaut, meilleur, dauphin := calibrateFrameConfig(f.reg, kfs, deltas, f.fc.CadreDeBalayage())
	add("calibrage ScanMarchFacts", fmt.Sprintf("retenu %d (profil plat : %v) ; meilleur %d (%d localises / %d evenements), dauphin %d (%d)",
		cal.IDLowBits, parDefaut, meilleur.cfg.IDLowBits, meilleur.located, meilleur.events, dauphin.cfg.IDLowBits, dauphin.located))
	largeurs := append([]int(nil), b3Largeurs...)
	for _, w := range largeurs {
		f2 := *f
		f2.cfg.IDLowBits = w
		d := b3NouveauDiag(&f2, b, false)
		j := cmNouveauJuge(&f2, b, col.statut)
		j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
		r, o, _ := cmMarcher(&f2, cmVariante{}, cmMux{d, j})
		nom := fmt.Sprintf("largeur-%d", w)
		if w == cal.IDLowBits {
			nom += " (calibree)"
		}
		ligne(nom, 0, r, o, j)
		tables(nom, d.t)
	}
}

// b3Log2Haut porte `FUN_1406d310c` : ceil(log2(n)) (0 pour n = 0 ou 1).
func b3Log2Haut(n uint32) int {
	if n == 0 {
		return 0
	}
	h := 31
	for n>>uint(h) == 0 {
		h--
	}
	if n&(1<<uint(h)-1) != 0 {
		h++
	}
	return h
}

// TestCampagneBis3Log2Haut : la largeur que le jeu tire du cardinal (`FUN_1406d310c`, relu dans
// Ghidra le 2026-10-01) — 8 191 slots (le bloc de 343 019 octets) donnent 13 bits.
func TestCampagneBis3Log2Haut(t *testing.T) {
	for n, w := range map[uint32]int{1: 0, 2: 1, 3: 2, 4096: 12, 7680: 13, 8191: 13, 8192: 13, 8193: 14, 16384: 14} {
		if got := b3Log2Haut(n); got != w {
			t.Errorf("b3Log2Haut(%d) = %d, attendu %d", n, got, w)
		}
	}
}
