package mapvar

// socles_root6_test.go — LE CHAMP RACINE 6, JAMAIS DECODE.
//
// POURQUOI IL A SA PLACE DANS LE LOT DES SOCLES. La grammaire (`mapvar.go`) range
// `root[6]` sous « tables de regroupement (non exploitees ici) » et n'en dit pas plus.
// Le premier inventaire de `catalyst_catalyst.mvar` (2026-08-19) y trouve une liste de
// ONZE entrees — et l'oracle des socles mesures sur cette meme carte en compte ONZE
// (10 socles d'armes + 1 socle de power-up). La coincidence de cardinalite ne prouve
// rien a elle seule ; ce fichier la met a l'epreuve en deroulant la structure entiere.
//
// LECTURE SEULE. Garde : `MVAR_FILE`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// soclesProfondeurMax borne le deroulement : au-dela, une structure Bond profonde noie la
// sortie sans rien apprendre.
const soclesProfondeurMax = 6

// TestSoclesRoot6 deroule integralement `root[6]` : chaque entree, chaque champ, chaque
// valeur scalaire. C'est la reponse a « le fichier de carte porte-t-il autre chose que
// des objectifs ? », et elle se lit dans les valeurs, pas dans les cardinalites.
func TestSoclesRoot6(t *testing.T) {
	root, nom := soclesRacine(t)
	f, ok := root.Field(6)
	if !ok {
		t.Logf("%s : root[6] ABSENT", nom)
		return
	}
	t.Logf("== %s : root[6] deroule ==", nom)
	soclesDump(t, "root[6]", f, 0)
}

// TestSoclesRoot11 fait de meme pour `root[11]` (« surcharges de proprietes indexees »),
// l'autre champ que la grammaire declare non exploite. Si l'identite d'un objet — quelle
// arme un socle fait apparaitre — etait ecrite quelque part, ce serait la.
func TestSoclesRoot11(t *testing.T) {
	root, nom := soclesRacine(t)
	f, ok := root.Field(11)
	if !ok {
		t.Logf("%s : root[11] ABSENT", nom)
		return
	}
	t.Logf("== %s : root[11] deroule (%d paires) ==", nom, len(f.Pairs))
	for i, kv := range f.Pairs {
		if i >= 40 {
			t.Logf("  ... %d paires de plus", len(f.Pairs)-40)
			break
		}
		t.Logf("  cle : %s", soclesScalaire(kv.Key))
		soclesDump(t, fmt.Sprintf("  valeur[%d]", i), kv.Val, 1)
	}
}

// TestSoclesRoot1 deroule l'en-tete de carte (13 champs sur Catalyst) : c'est la que
// vivrait une reference au variant de MODE, si le fichier en portait une.
func TestSoclesRoot1(t *testing.T) {
	root, nom := soclesRacine(t)
	f, ok := root.Field(1)
	if !ok {
		t.Logf("%s : root[1] ABSENT", nom)
		return
	}
	t.Logf("== %s : root[1] (en-tete) deroule ==", nom)
	soclesDump(t, "root[1]", f, 0)
}

// soclesRacine decode la racine Bond du fichier sous garde.
func soclesRacine(t *testing.T) (Value, string) {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(soclesFileEnv))
	if path == "" {
		t.Skipf("%s absent — instrument de mesure ignore", soclesFileEnv)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s: %v", path, err)
	}
	root, err := DecodeRoot(buf)
	if err != nil {
		t.Fatalf("DecodeRoot de %s: %v", path, err)
	}
	return root, path
}

// soclesDump deroule un noeud Bond, valeurs comprises.
func soclesDump(t *testing.T, prefixe string, v Value, prof int) {
	t.Helper()
	if prof > soclesProfondeurMax {
		t.Logf("%s ... (profondeur max)", prefixe)
		return
	}
	if len(v.Items) == 0 && len(v.Fields) == 0 && len(v.Pairs) == 0 {
		t.Logf("%s = %s", prefixe, soclesScalaire(v))
		return
	}
	t.Logf("%s : type %d, %d items, %d champs, %d paires",
		prefixe, v.Type, len(v.Items), len(v.Fields), len(v.Pairs))
	for i, it := range v.Items {
		soclesDump(t, fmt.Sprintf("%s[%d]", prefixe, i), it, prof+1)
	}
	cles := make([]int, 0, len(v.Fields))
	for id := range v.Fields {
		cles = append(cles, int(id))
	}
	sort.Ints(cles)
	for _, id := range cles {
		soclesDump(t, fmt.Sprintf("%s.%d", prefixe, id), v.Fields[uint16(id)], prof+1)
	}
	for i, kv := range v.Pairs {
		t.Logf("%s{%d} cle = %s", prefixe, i, soclesScalaire(kv.Key))
		soclesDump(t, fmt.Sprintf("%s{%d}", prefixe, i), kv.Val, prof+1)
	}
}

// soclesScalaire rend la valeur d'un noeud terminal, avec son type — un entier lu comme
// signe ou non signe ne raconte pas la meme histoire.
func soclesScalaire(v Value) string {
	switch v.Type {
	case btString, btWString:
		return fmt.Sprintf("(str) %q", v.Str)
	case btFloat, btDouble:
		return fmt.Sprintf("(f) %g", v.Float)
	case btBool:
		return fmt.Sprintf("(bool) %v", v.Uint != 0)
	case btUint8, btUint16, btUint32, btUint64:
		return fmt.Sprintf("(u%d) %d", v.Type, v.Uint)
	default:
		return fmt.Sprintf("(t%d) int=%d uint=%d f=%g", v.Type, v.Int, v.Uint, v.Float)
	}
}
