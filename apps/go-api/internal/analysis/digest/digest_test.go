package digest

// digest_test.go — LES PROPRIETES QUE LE HARNAIS D'EQUIVALENCE ACHETE.
//
// Chacune correspond a un trou qu'un digest naif laisserait ouvert : champ invisible, ordre de
// map, flottant non fini, artefact reencode, comptage, structure auto-referente.

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
	"time"
)

// exemple porte un champ NON exporte : c'est la raison d'etre du paquet.
type exemple struct {
	Nom    string
	Points []int
	cache  int
}

func TestDeuxValeursEgalesRendentLaMemeEmpreinte(t *testing.T) {
	a := exemple{Nom: "film", Points: []int{1, 2, 3}, cache: 7}
	b := exemple{Nom: "film", Points: []int{1, 2, 3}, cache: 7}
	ca, sa := Of(a)
	cb, sb := Of(b)
	if sa != sb {
		t.Fatalf("deux valeurs egales rendent deux empreintes : %s vs %s", sa, sb)
	}
	if ca != cb || ca != 1 {
		t.Fatalf("compte d'un struct : attendu 1 pour les deux, obtenu %d et %d", ca, cb)
	}
}

func TestUnChampNonExporteChangeLEmpreinte(t *testing.T) {
	_, avec := Of(exemple{Nom: "film", cache: 7})
	_, sans := Of(exemple{Nom: "film", cache: 8})
	if avec == sans {
		t.Fatal("un champ NON exporte different rend la meme empreinte : le digest est aveugle " +
			"la ou un refacto peut casser en silence (cf. filmdec.BipedPosition.componentDirs)")
	}
}

func TestFlottantsNonFinisRendentUneEmpreinteStable(t *testing.T) {
	cas := []struct {
		nom string
		val float64
	}{
		{"NaN", math.NaN()},
		{"+Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
	}
	for _, c := range cas {
		_, un := Of(struct{ V float64 }{c.val})
		_, deux := Of(struct{ V float64 }{c.val})
		if un != deux {
			t.Fatalf("%s : deux empreintes pour la meme valeur (%s vs %s)", c.nom, un, deux)
		}
		if un == "" {
			t.Fatalf("%s : aucune empreinte rendue", c.nom)
		}
	}
	// Deux non-finis DIFFERENTS ne doivent pas se confondre.
	_, nan := Of(math.NaN())
	_, plus := Of(math.Inf(1))
	_, moins := Of(math.Inf(-1))
	if nan == plus || plus == moins || nan == moins {
		t.Fatalf("NaN, +Inf et -Inf se confondent : %s / %s / %s", nan, plus, moins)
	}
}

func TestOrdreDInsertionDUneMapNeChangeRien(t *testing.T) {
	un := map[string]int{}
	for _, k := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
		un[k] = len(k)
	}
	deux := map[string]int{}
	for _, k := range []string{"epsilon", "delta", "gamma", "beta", "alpha"} {
		deux[k] = len(k)
	}
	cu, su := Of(un)
	cd, sd := Of(deux)
	if su != sd {
		t.Fatalf("l'ordre d'insertion change l'empreinte : %s vs %s", su, sd)
	}
	if cu != 5 || cd != 5 {
		t.Fatalf("compte d'une map : attendu 5, obtenu %d et %d", cu, cd)
	}
	// Le rendu doit etre STABLE d'un appel a l'autre (l'iteration de map est aleatoire).
	for i := 0; i < 20; i++ {
		if _, s := Of(un); s != su {
			t.Fatalf("empreinte instable au tour %d : %s vs %s", i, s, su)
		}
	}
}

func TestUneTrancheDOctetsEstHacheeTelleQuelle(t *testing.T) {
	blob := []byte(`{"schema":36,"tracks":[]}`)
	brut := sha256.Sum256(blob)
	n, sum := Of(blob)
	if sum != hex.EncodeToString(brut[:]) {
		t.Fatalf("l'empreinte d'un artefact n'est pas le sha256 de ses octets :\n  digest %s\n  brut   %s",
			sum, hex.EncodeToString(brut[:]))
	}
	if n != len(blob) {
		t.Fatalf("compte d'une tranche d'octets : attendu %d, obtenu %d", len(blob), n)
	}
}

func TestComptesDePremierNiveau(t *testing.T) {
	trois := []int{1, 2, 3}
	var nulle []int
	var pointeurNul *exemple
	cas := []struct {
		nom    string
		val    any
		attend int
	}{
		{"tranche", trois, 3},
		{"tranche nulle", nulle, 0},
		{"pointeur traverse", &trois, 3},
		{"pointeur nul", pointeurNul, 0},
		{"nil", nil, 0},
		{"map nulle", map[string]int(nil), 0},
		{"chaine", "abcd", 4},
		{"tableau", [2]int{1, 2}, 2},
		{"entier", 42, 1},
		{"struct", exemple{}, 1},
	}
	for _, c := range cas {
		if got, _ := Of(c.val); got != c.attend {
			t.Errorf("%s : compte attendu %d, obtenu %d", c.nom, c.attend, got)
		}
	}
}

// noeud sert la propriete de terminaison : une structure qui se pointe elle-meme.
type noeud struct {
	Nom     string
	Suivant *noeud
}

func TestUnCycleTermine(t *testing.T) {
	a := &noeud{Nom: "a"}
	b := &noeud{Nom: "b", Suivant: a}
	a.Suivant = b
	fini := make(chan string, 1)
	go func() {
		_, sum := Of(a)
		fini <- sum
	}()
	select {
	case sum := <-fini:
		if sum == "" {
			t.Fatal("cycle : aucune empreinte rendue")
		}
		if _, deux := Of(a); deux != sum {
			t.Fatalf("cycle : empreinte instable (%s vs %s)", deux, sum)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("un cycle de pointeurs n'a pas termine — la marque <cycle> ne coupe pas la descente")
	}
}

// TestUnPointeurPartageNEstPasUnCycle : le meme noeud vu par DEUX branches se rend en entier ;
// seule une reference OUVERTE sur le chemin courant est un cycle.
func TestUnPointeurPartageNEstPasUnCycle(t *testing.T) {
	partage := &noeud{Nom: "commun"}
	_, deuxFois := Of([]*noeud{partage, partage})
	_, deuxCopies := Of([]*noeud{{Nom: "commun"}, {Nom: "commun"}})
	if deuxFois != deuxCopies {
		t.Fatalf("un noeud partage se rend autrement que sa copie : %s vs %s", deuxFois, deuxCopies)
	}
}

// TestLesQuatreCollisionsConstruites — LES QUATRE COLLISIONS MESUREES PAR LA REVUE DU LOT 0
// (2026-09-02), qui doivent maintenant DIFFERER. Chacune exploitait un delimiteur de structure
// que la donnee pouvait imiter : c'est le prefixe de longueur qui les ferme (cf. l'en-tete).
//
// Ce ne sont pas des cas d'ecole : le harnais d'equivalence hache des tranches de chaines
// (noms d'armes, identifiants de carte), des maps a cles textuelles et des tranches d'octets.
func TestLesQuatreCollisionsConstruites(t *testing.T) {
	cas := []struct {
		nom      string
		un, deux any
	}{
		{
			"une chaine qui porte le separateur de tranche",
			[]string{"a,b"},
			[]string{"a", "b"},
		},
		{
			"une cle de map qui porte le separateur cle/valeur",
			map[string]string{"a": "b=c"},
			map[string]string{"a=b": "c"},
		},
		{
			"des octets imbriques qui portent le code de la virgule",
			[][]byte{{1, 2}, {3}},
			[][]byte{{1, 2, 44, 3}},
		},
		{
			"des octets imbriques qui imitent le marqueur du nul",
			[][]byte{nil},
			[][]byte{[]byte("<nil>")},
		},
	}
	for _, c := range cas {
		_, un := Of(c.un)
		_, deux := Of(c.deux)
		if un == deux {
			t.Errorf("%s : COLLISION, les deux valeurs rendent %s", c.nom, un)
		}
	}
}

// TestUneTrancheDOctetsNulleALaRacineEstZeroOctet verrouille la CONTREPARTIE ecrite du cas
// racine : a la racine, une tranche d'octets EST l'artefact, donc `nil` vaut zero octet et non
// un marqueur — et le marqueur du nul, lui, ne peut plus etre imite des la profondeur 1
// (cf. TestLesQuatreCollisionsConstruites).
func TestUneTrancheDOctetsNulleALaRacineEstZeroOctet(t *testing.T) {
	vide := sha256.Sum256(nil)
	if _, sum := Of([]byte(nil)); sum != hex.EncodeToString(vide[:]) {
		t.Errorf("[]byte(nil) a la racine : attendu le sha256 du vide, obtenu %s", sum)
	}
	_, nulle := Of([]byte(nil))
	_, imitation := Of([]byte("<nil>"))
	if nulle == imitation {
		t.Error("[]byte(nil) et []byte(\"<nil>\") se confondent a la racine")
	}
}

// TestUneMapQuiSeContientTermine — le garde de cycle ne suit que les POINTEURS ; une map qui se
// contient n'en porte aucun et faisait DEBORDER LA PILE. La borne de profondeur rend un digest.
func TestUneMapQuiSeContientTermine(t *testing.T) {
	m := map[string]any{}
	m["soi"] = m
	fini := make(chan string, 1)
	go func() {
		_, sum := Of(m)
		fini <- sum
	}()
	select {
	case sum := <-fini:
		if sum == "" {
			t.Fatal("map auto-referente : aucune empreinte rendue")
		}
		if _, deux := Of(m); deux != sum {
			t.Fatalf("map auto-referente : empreinte instable (%s vs %s)", deux, sum)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("une map qui se contient n'a pas termine — la borne de profondeur ne coupe pas la marche")
	}
}

// TestLaBorneDeProfondeurNeCoupePasLesValeursLegitimes : une imbrication profonde mais FINIE se
// rend en entier. La borne est un dernier rempart, pas un filtre.
func TestLaBorneDeProfondeurNeCoupePasLesValeursLegitimes(t *testing.T) {
	var v any = "feuille"
	for i := 0; i < 100; i++ {
		v = []any{v}
	}
	var w any = "feuille"
	for i := 0; i < 100; i++ {
		w = []any{w}
	}
	_, a := Of(v)
	_, b := Of(w)
	if a != b {
		t.Fatalf("deux imbrications identiques de 100 niveaux rendent deux empreintes : %s vs %s", a, b)
	}
	var different any = "autre"
	for i := 0; i < 100; i++ {
		different = []any{different}
	}
	if _, d := Of(different); d == a {
		t.Fatal("la feuille d'une imbrication de 100 niveaux n'atteint plus l'empreinte")
	}
}

// empreintes rejoue Of N fois et rend l'ensemble des empreintes vues. Une seule suffit a
// prouver la stabilite ; deux prouvent que l'ordre d'iteration de la map fuit dans le rendu.
func empreintes(v any, tours int) map[string]int {
	vues := map[string]int{}
	for i := 0; i < tours; i++ {
		_, sum := Of(v)
		vues[sum]++
	}
	return vues
}

// TestDeuxClesDeRenduEGALNeRendentQuUneEmpreinte — LE DEFAUT MESURE LE 2026-09-02 : le tri des
// paires par le seul rendu de la CLE est non deterministe des que deux cles DISTINCTES rendent
// les memes octets. La grammaire ne porte pas les noms de type, donc `int32(1)` et `int64(1)`
// rendent tous deux `1` ; deux cles POINTEUR vers des valeurs egales rendent la valeur pointee.
// Sans departage par la valeur, les deux cas ci-dessous rendaient DEUX empreintes.
func TestDeuxClesDeRenduEGALNeRendentQuUneEmpreinte(t *testing.T) {
	cas := []struct {
		nom string
		val any
	}{
		{"deux entiers de largeurs differentes", map[any]int{int32(1): 100, int64(1): 200}},
		{"deux pointeurs vers des valeurs egales", map[*noeud]int{{Nom: "a"}: 1, {Nom: "a"}: 2}},
	}
	for _, c := range cas {
		if vues := empreintes(c.val, 200); len(vues) != 1 {
			t.Errorf("%s : %d empreintes sur 200 tours (%v) — l'ordre d'iteration de la map fuit",
				c.nom, len(vues), vues)
		}
	}
}

// TestDeuxEntreesIndISCERNABLESRendentUneEmpreinteStable : cle ET valeur de meme rendu. Les deux
// entrees sont indiscernables PAR CONSTRUCTION — l'ordre entre elles ne change pas le flux, et
// l'empreinte doit donc rester stable sans que le departage ait a les separer.
func TestDeuxEntreesIndiscernablesRendentUneEmpreinteStable(t *testing.T) {
	if vues := empreintes(map[any]int{int32(1): 100, int64(1): 100}, 200); len(vues) != 1 {
		t.Errorf("deux entrees indiscernables : %d empreintes sur 200 tours (%v)", len(vues), vues)
	}
}

// TestLigneDeGrammaireSeRelit : ce que GrammarLine ecrit, ParseGrammarLine le relit — et une
// ligne qui n'est pas un marqueur (fichier fige avant son introduction) se signale comme telle
// au lieu de passer pour la version 0.
func TestLigneDeGrammaireSeRelit(t *testing.T) {
	v, ok := ParseGrammarLine(GrammarLine())
	if !ok || v != GrammarVersion {
		t.Fatalf("relecture de %q : version %d, ok=%v — attendu %d", GrammarLine(), v, ok, GrammarVersion)
	}
	for _, ligne := range []string{"", "score\t1\tabc", "# digest-grammar:", "# digest-grammar: x"} {
		if _, ok := ParseGrammarLine(ligne); ok {
			t.Errorf("%q passe pour une ligne de grammaire", ligne)
		}
	}
}
