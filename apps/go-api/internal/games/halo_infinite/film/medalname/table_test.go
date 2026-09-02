// Package medalname — garde-rails de la table (type_hint, medal_type) -> nom.
//
// Le corpus mesure (testdata/corpus_medailles_2026-09-02.tsv) est la PIECE : chaque
// ligne est un couple reellement observe dans un film, avec le nom que l ancien
// collecteur Python avait ecrit dans raw_json. Les tests rejouent ce corpus contre la
// table. Une entree retiree, renommee ou inventee fait rougir.
package medalname

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// cheminCorpus est la mesure figee du 2026-09-02 (44 568 events medal).
const cheminCorpus = "testdata/corpus_medailles_2026-09-02.tsv"

// ligneCorpus est une ligne du corpus mesure.
type ligneCorpus struct {
	typeHint    int
	medalType   int
	nom         string
	occurrences int
}

// lireCorpus lit le TSV mesure (en-tete + 1 ligne par couple).
func lireCorpus(t *testing.T) []ligneCorpus {
	t.Helper()
	f, err := os.Open(cheminCorpus)
	if err != nil {
		t.Fatalf("ouverture du corpus mesure: %v", err)
	}
	defer f.Close()

	var out []ligneCorpus
	sc := bufio.NewScanner(f)
	numero := 0
	for sc.Scan() {
		numero++
		texte := strings.TrimRight(sc.Text(), "\r")
		if texte == "" {
			continue
		}
		if numero == 1 {
			continue // en-tete
		}
		champs := strings.Split(texte, "\t")
		if len(champs) != 4 {
			t.Fatalf("ligne %d du corpus mal formee (%d champs): %q", numero, len(champs), texte)
		}
		th, err := strconv.Atoi(champs[0])
		if err != nil {
			t.Fatalf("ligne %d: type_hint illisible: %v", numero, err)
		}
		mt, err := strconv.Atoi(champs[1])
		if err != nil {
			t.Fatalf("ligne %d: medal_type illisible: %v", numero, err)
		}
		occ, err := strconv.Atoi(champs[3])
		if err != nil {
			t.Fatalf("ligne %d: occurrences illisibles: %v", numero, err)
		}
		out = append(out, ligneCorpus{typeHint: th, medalType: mt, nom: champs[2], occurrences: occ})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("lecture du corpus: %v", err)
	}
	return out
}

// TestTableCouvreToutLeCorpusMesure — LE garde-rail : chaque couple mesure doit
// rendre EXACTEMENT le nom mesure. C est ce test qui echoue si quelqu un retire ou
// renomme une entree de la table.
func TestTableCouvreToutLeCorpusMesure(t *testing.T) {
	corpus := lireCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("corpus mesure vide — le TSV de reference a disparu")
	}
	manquants := 0
	for _, l := range corpus {
		nom, ok := Lookup(l.typeHint, l.medalType)
		if !ok {
			manquants++
			t.Errorf("couple mesure absent de la table: type_hint=%d medal_type=%d (%q, %d occurrences)",
				l.typeHint, l.medalType, l.nom, l.occurrences)
			continue
		}
		if nom != l.nom {
			t.Errorf("type_hint=%d medal_type=%d: table=%q, corpus mesure=%q",
				l.typeHint, l.medalType, nom, l.nom)
		}
	}
	if manquants > 0 {
		t.Errorf("%d couples mesures non couverts sur %d", manquants, len(corpus))
	}
}

// TestTableNInventeAucuneEntree — reciproque : aucune entree de la table qui ne soit
// pas dans la mesure. Une entree devinee est un nom faux servi en production.
func TestTableNInventeAucuneEntree(t *testing.T) {
	corpus := lireCorpus(t)
	mesures := make(map[medalKey]bool, len(corpus))
	for _, l := range corpus {
		mesures[medalKey{typeHint: l.typeHint, medalType: l.medalType}] = true
	}
	for k := range table {
		if !mesures[k] {
			t.Errorf("entree hors mesure: type_hint=%d medal_type=%d -> %q (absente du corpus)",
				k.typeHint, k.medalType, table[k])
		}
	}
}

// TestCompletudeTable — la mesure du 2026-09-02 : 124 couples.
func TestCompletudeTable(t *testing.T) {
	const couplesMesures = 124
	if Len() != couplesMesures {
		t.Fatalf("table: %d couples, mesure du 2026-09-02 = %d", Len(), couplesMesures)
	}
	corpus := lireCorpus(t)
	if len(corpus) != couplesMesures {
		t.Fatalf("corpus TSV: %d lignes, mesure du 2026-09-02 = %d", len(corpus), couplesMesures)
	}
	const eventsMesures = 44568
	total := 0
	for _, l := range corpus {
		total += l.occurrences
	}
	if total != eventsMesures {
		t.Fatalf("corpus TSV: %d events cumules, mesure du 2026-09-02 = %d", total, eventsMesures)
	}
}

// TestBijectionNoms — la mesure est une bijection : 124 couples, 124 noms distincts,
// zero couple ambigu. Deux couples qui rendraient le meme nom signeraient une table
// recopiee de travers.
func TestBijectionNoms(t *testing.T) {
	vus := make(map[string]medalKey, len(table))
	for k, nom := range table {
		if nom == "" {
			t.Errorf("nom vide pour type_hint=%d medal_type=%d", k.typeHint, k.medalType)
			continue
		}
		if precedent, deja := vus[nom]; deja {
			t.Errorf("nom %q partage par (type_hint=%d,medal_type=%d) et (type_hint=%d,medal_type=%d)",
				nom, precedent.typeHint, precedent.medalType, k.typeHint, k.medalType)
			continue
		}
		vus[nom] = k
	}
	if len(vus) != Len() {
		t.Fatalf("%d noms distincts pour %d couples — la bijection est rompue", len(vus), Len())
	}
}

// TestTypeHintsSontDesPoidsDeMedaille — invariant du decodeur : un event medal a
// forcement un type_hint de la liste des poids de tri des medailles
// (analysis.medalSortingWeights, non exporte : la liste est recopiee ici et le test
// echoue des qu une entree sort de cet ensemble). Une entree a type_hint=20 (mort)
// ou 10 (mode) serait une ligne de table posee au mauvais endroit.
func TestTypeHintsSontDesPoidsDeMedaille(t *testing.T) {
	poids := map[int]bool{
		50: true, 51: true, 52: true, 100: true, 101: true,
		150: true, 200: true, 205: true, 210: true, 220: true,
		225: true, 230: true, 235: true, 240: true, 245: true, 250: true,
	}
	for k := range table {
		if !poids[k.typeHint] {
			t.Errorf("type_hint=%d hors des poids de medaille (medal_type=%d -> %q)",
				k.typeHint, k.medalType, table[k])
		}
	}
}

// TestLookupCoupleInconnu — degradation : pas de nom voisin, pas de panique.
func TestLookupCoupleInconnu(t *testing.T) {
	// medal_type 255 n a jamais ete observe sur aucun type_hint du corpus.
	if nom, ok := Lookup(50, 255); ok {
		t.Fatalf("couple inconnu (50,255) resolu en %q — la table doit rendre false", nom)
	}
	// type_hint 20 = mort : jamais une medaille.
	if nom, ok := Lookup(20, 26); ok {
		t.Fatalf("couple (20,26) resolu en %q — 20 est un type_hint de mort", nom)
	}
}
