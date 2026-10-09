//go:build research

package main

// v2_research_test.go — LA CARTE V2 DE BOUT EN BOUT, sur les bobines versionnees du depot : les
// trois TSV du mode `fermeture` sont IDENTIQUES sous `v2` (hors pic memoire et duree), et les sept
// TSV de la carte v2 sont ecrits. Puis les classements purs : vue C, reste, et la naissance d un
// eid rejete contre deux blocs de type 1 synthetiques.
//
//	go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// mesurerLesBobines mesure les deux bobines de test sous `md` et rend le repertoire du rapport.
func mesurerLesBobines(t *testing.T, md modes) string {
	t.Helper()
	dir := t.TempDir()
	tab, err := lireTable(cheminTable)
	if err != nil {
		t.Fatal(err)
	}
	rap, err := ouvrirRapport(dir, tab, md, optionsV2{paquets: md.v2})
	if err != nil {
		t.Fatal(err)
	}
	if err := mesurerUnFilm(racineKillsource, "minibobine_000d5950", 4, rap, nil); err != nil {
		t.Fatalf("bobine contigue : %v", err)
	}
	if err := mesurerUnFilm(racineRejeu, "minifilm_bcb6d393", 4, rap, nil); err != nil {
		t.Fatalf("mini-bobine du rejeu : %v", err)
	}
	if err := rap.terminer(10); err != nil {
		t.Fatal(err)
	}
	return dir
}

// lireSansMesureDeMachine relit un TSV ; pour `fermeture_films.tsv`, les deux dernieres colonnes
// (pic memoire, duree) dependent de la machine et sont retirees.
func lireSansMesureDeMachine(t *testing.T, dir, nom string) string {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // repertoire du test
	if err != nil {
		t.Fatalf("%s : %v", nom, err)
	}
	if nom != "fermeture_films.tsv" {
		return string(brut)
	}
	var out []string
	for _, l := range strings.Split(strings.TrimRight(string(brut), "\n"), "\n") {
		c := strings.Split(l, "\t")
		out = append(out, strings.Join(c[:len(c)-2], "\t"))
	}
	return strings.Join(out, "\n")
}

// TestV2RendLesSortiesDeFermetureALIdentique : les TSV du mode `fermeture` ne bougent pas sous
// `v2`, et la carte v2 ecrit ses sept TSV et ses sections.
func TestV2RendLesSortiesDeFermetureALIdentique(t *testing.T) {
	ref := mesurerLesBobines(t, modes{fermeture: true})
	v2 := mesurerLesBobines(t, modes{fermeture: true, v2: true})
	for _, nom := range []string{"fermeture_films.tsv", "fermeture_archetypes.tsv", "fermeture_bloquants.tsv"} {
		if a, b := lireSansMesureDeMachine(t, ref, nom), lireSansMesureDeMachine(t, v2, nom); a != b {
			t.Errorf("%s differe sous v2 :\n--- fermeture\n%s\n--- v2\n%s", nom, a, b)
		}
	}
	attendus := map[string]string{
		"fermeture_sorties_vueB.tsv":       "minibobine_000d5950\t",
		"fermeture_hors_cadre.tsv":         "sortie_vueB\tvue_c\treste",
		"fermeture_hors_cadre_dernier.tsv": "sortie_vueB\tdernier_lu",
		"fermeture_rejets.tsv":             "etat_bloc_type1\tnaissance",
		"fermeture_entrees.tsv":            "minibobine_000d5950\t",
		"fermeture_chunk3.tsv":             "minibobine_000d5950\t",
		"fermeture_borne.tsv":              "minibobine_000d5950\t",
		"fermeture_resume.md":              "## Lot L0",
		"fermeture_ecrivain.tsv":           "\tpaquets\tfermes au bit\t",
		"fermeture_denominateurs.tsv":      "utiles_lus\tutiles_fermes_au_bit",
		"fermeture_temoins_decales.tsv":    "minibobine_000d5950\t",
		"fermeture_paquets.tsv":            "minibobine_000d5950\t",
	}
	for nom, marque := range attendus {
		if !strings.Contains(lireSansMesureDeMachine(t, v2, nom), marque) {
			t.Errorf("%s ne porte pas %q", nom, marque)
		}
	}
	if !strings.Contains(lireSansMesureDeMachine(t, v2, "fermeture_sorties_vueB.tsv"), "\tterminateur\t") {
		t.Error("aucune sortie de vue B par terminateur sur la bobine contigue")
	}
}

// TestClassesDeVueCEtDeReste : les classes du resume.
func TestClassesDeVueCEtDeReste(t *testing.T) {
	entrees := func(n int) grammar.FluxVueC { return grammar.FluxVueC{Entrees: make([]grammar.EntreeDeControle, n)} }
	cas := map[string]grammar.FluxVueC{"vide": {Vide: true}, "0 entree (kind 3 seul)": {},
		"3 entree(s)": entrees(3), "9+ entrees": entrees(12)}
	for want, c := range cas {
		if got := classeDeVueC(c); got != want {
			t.Errorf("classeDeVueC = %q, attendu %q", got, want)
		}
	}
	for reste, want := range map[int]string{-1: "negatif", 0: "0-7 (non nuls)", 7: "0-7 (non nuls)",
		8: "8-63", 63: "8-63", 64: ">= 64"} {
		if got := classeDeReste(reste); got != want {
			t.Errorf("classeDeReste(%d) = %q, attendu %q", reste, got, want)
		}
	}
}

// collecteurDeDeuxBlocs rend un collecteur au chunk 1, dont le bloc et celui du chunk 2 portent
// le slot 7 dans les etats donnes.
func collecteurDeDeuxBlocs(avant, apres grammar.DatumEntry) *collecteurV2 {
	bloc := func(e grammar.DatumEntry) blocDuChunk {
		en := make([]grammar.DatumEntry, 8)
		en[7] = e
		return blocDuChunk{present: true, entrees: en}
	}
	return &collecteurV2{m: nouvelleMesureV2(), suivant: map[int]int{1: 2}, chunk: 1,
		neufs: map[uint32]bool{}, neufsDesync: map[uint32]bool{}, blocs: map[int]blocDuChunk{1: bloc(avant), 2: bloc(apres)}}
}

// TestNaissanceDUnEidRejete : les classes de naissance, sur l eid `1<<30 | 7` (generation 1).
func TestNaissanceDUnEidRejete(t *testing.T) {
	const eid = 1<<30 | 7
	vivant := grammar.DatumEntry{Drapeaux: grammar.DatumAlloue | grammar.DatumPublie, Gen: 1, Generation: 2}
	libere := grammar.DatumEntry{Gen: 1, Generation: 2}
	autre := grammar.DatumEntry{Drapeaux: grammar.DatumAlloue | grammar.DatumPublie, Gen: 2, Generation: 3}
	cas := []struct {
		nom           string
		avant, apres  grammar.DatumEntry
		want, etatDeb string
	}{
		{"nee et vivante", grammar.DatumEntry{}, vivant, "naissance non lue", "vide"},
		{"nee et morte", grammar.DatumEntry{}, libere, "naissance non lue", "vide"},
		{"deja vivante", vivant, vivant, "vivant au bloc du chunk", "vivant"},
		{"morte avant le chunk", libere, libere, "libere avant le chunk (meme generation)", "trace"},
		{"autre generation", grammar.DatumEntry{}, autre, "realloue sous une autre generation", "vide"},
		{"rien", grammar.DatumEntry{}, grammar.DatumEntry{}, "aucune allocation", "vide"},
	}
	for _, c := range cas {
		col := collecteurDeDeuxBlocs(c.avant, c.apres)
		if got := col.naissance(eid); got != c.want {
			t.Errorf("%s : naissance %q, attendu %q", c.nom, got, c.want)
		}
		if got := etatAuBloc(col.bloc(1), 7); got != c.etatDeb {
			t.Errorf("%s : etat au bloc %q, attendu %q", c.nom, got, c.etatDeb)
		}
	}
	col := collecteurDeDeuxBlocs(grammar.DatumEntry{}, vivant)
	col.neufs[7] = true
	if got := col.naissance(eid); got != "NEW lu dans le chunk" {
		t.Errorf("NEW lu : naissance %q", got)
	}
	col.chunk = 2
	col.neufs = map[uint32]bool{}
	if got := col.naissance(eid); got != "non mesurable" {
		t.Errorf("dernier chunk : naissance %q, attendu non mesurable", got)
	}
	if got := etatAuBloc(blocDuChunk{}, 7); got != "sans bloc" {
		t.Errorf("chunk sans bloc : %q", got)
	}
	if got := etatAuBloc(col.bloc(1), 900); got != "absent" {
		t.Errorf("slot hors table : %q", got)
	}
}

// TestNaissanceGenerationZeroEtNeufDesynchronise : les deux classes corrigees du lot L0 (L0.2).
// `FUN_142f2e598` pose `gen = (gen + 1) & 3` a chaque allocation : un slot trace sous la generation
// 3 au bloc du chunk et a la generation 0 sans drapeau au bloc suivant a ete alloue sous la
// generation 0 puis libere — une naissance, que la regle « generation ou drapeau poses » rangeait
// en « realloue » (D-43). Un NEW lu dont la traversee a desynchronise n est pas une naissance non
// lue (D-44).
func TestNaissanceGenerationZeroEtNeufDesynchronise(t *testing.T) {
	const eid0 = 0<<30 | 7
	trace3 := grammar.DatumEntry{Gen: 3, Generation: 4}
	if got := collecteurDeDeuxBlocs(trace3, grammar.DatumEntry{}).naissance(eid0); got != "naissance non lue, generation 0" {
		t.Errorf("generation 0 nee puis liberee : %q", got)
	}
	if got := collecteurDeDeuxBlocs(grammar.DatumEntry{}, grammar.DatumEntry{}).naissance(eid0); got != "aucune allocation" {
		t.Errorf("slot jamais alloue aux deux blocs : %q", got)
	}
	const eid1 = 1<<30 | 7
	vivant := grammar.DatumEntry{Drapeaux: grammar.DatumAlloue | grammar.DatumPublie, Gen: 1, Generation: 2}
	col := collecteurDeDeuxBlocs(grammar.DatumEntry{}, vivant)
	col.neufsDesync[7] = true
	if got := col.naissance(eid1); got != "NEW lu desynchronise · naissance non lue" {
		t.Errorf("NEW lu desynchronise : %q", got)
	}
}
