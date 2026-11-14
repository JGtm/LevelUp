package replayview_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/service/replayview"
)

// roundTrip encode `body`, décode le résultat avec le décodeur de test et exige le même arbre
// (mêmes clés dans le même ordre, mêmes valeurs). Rend le corps encodé.
func roundTrip(t *testing.T, body string) string {
	t.Helper()
	enc, err := replayview.EncodeColumns([]byte(body))
	if err != nil {
		t.Fatalf("encodage refusé : %v", err)
	}
	want := parseOrdered(t, []byte(body))
	got := decodeColumnsDoc(t, parseOrdered(t, enc))
	if path, ok := equalOrdered(want, got, "$"); !ok {
		t.Fatalf("aller-retour différent en %s\nentrée : %.400s\nsortie : %.400s", path, body, enc)
	}
	return string(enc)
}

// rows écrit `n` objets {"i":..., ...champs} ; `field` donne les champs de la ligne i.
func rows(n int, field func(i int) string) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "{" + field(i) + "}"
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func doc(list string) string { return `{"items":` + list + `}` }

func TestEncodeColumns_RacineEtListesCourtes(t *testing.T) {
	out := roundTrip(t, `{"a":1,"b":"x","c":[1,2,3],"d":{"e":null},"f":[]}`+"\n")
	if !strings.HasPrefix(out, `{"$enc":1,"a":1`) || !strings.HasSuffix(out, "\n") {
		t.Fatalf("racine inattendue : %s", out)
	}
	if got := roundTrip(t, `{}`); got != `{"$enc":1}` {
		t.Fatalf("racine vide : %s", got)
	}
}

func TestEncodeColumns_SeuilDeSeize(t *testing.T) {
	for _, n := range []int{15, 16, 17} {
		out := roundTrip(t, doc(rows(n, func(i int) string { return fmt.Sprintf(`"x":%d,"y":%d.5`, i, i) })))
		if got := strings.Contains(out, `"$c"`); got != (n >= 16) {
			t.Fatalf("%d éléments : table en colonnes = %v", n, got)
		}
	}
	for _, n := range []int{15, 16, 17} {
		out := roundTrip(t, doc("["+strings.Repeat("[1,2],", n-1)+"[3,4]]"))
		if got := strings.Contains(out, `"a":2`); got != (n >= 16) {
			t.Fatalf("%d n-uplets : table = %v", n, got)
		}
	}
}

func TestEncodeColumns_AbsentNullEtZeroSontTroisChoses(t *testing.T) {
	body := doc(rows(24, func(i int) string {
		switch i % 4 {
		case 0:
			return `"a":0`
		case 1:
			return `"a":null`
		case 2:
			return `"b":1`
		}
		return `"a":1,"b":2`
	}))
	out := roundTrip(t, body)
	if !strings.Contains(out, `"a":`) {
		t.Fatalf("la colonne creuse doit porter son absence : %s", out)
	}
	// première ligne absente, dernière absente, colonne entièrement présente
	roundTrip(t, doc(rows(20, func(i int) string {
		if i == 0 || i == 19 {
			return `"k":1`
		}
		return `"k":1,"h":0.25`
	})))
	roundTrip(t, doc(rows(20, func(i int) string { return `"k":1,"h":0.25` })))
}

func TestEncodeColumns_AbsenceEnChaineQuandElleEstPlusCourte(t *testing.T) {
	out := roundTrip(t, doc(rows(40, func(i int) string {
		if i%2 == 0 {
			return `"k":1,"h":5`
		}
		return `"k":1`
	})))
	if !strings.Contains(out, `"a":"`) {
		t.Fatalf("une alternance serrée doit s'écrire en chaîne de 0 et de 1 : %s", out)
	}
}

func TestEncodeColumns_Delta(t *testing.T) {
	out := roundTrip(t, doc(rows(30, func(i int) string { return fmt.Sprintf(`"t":%d`, 1000000+i*33) })))
	if !strings.Contains(out, `"d":1`) {
		t.Fatalf("une suite croissante régulière doit s'écrire en différences : %s", out)
	}
	out = roundTrip(t, doc(rows(30, func(i int) string { return fmt.Sprintf(`"t":%d`, (i*7919)%10) })))
	if strings.Contains(out, `"d":1`) {
		t.Fatalf("des valeurs courtes et désordonnées ne gagnent rien au delta : %s", out)
	}
}

func TestEncodeColumns_EchelleMixte(t *testing.T) {
	vals := []string{"1.5", "2", "3.25", "-0.125", "40", "0.5"}
	out := roundTrip(t, doc(rows(18, func(i int) string { return `"v":` + vals[i%len(vals)] })))
	if !strings.Contains(out, `"s":3`) {
		t.Fatalf("l'échelle est le plus grand nombre de décimales : %s", out)
	}
}

func TestEncodeColumns_ColonnesQuiResterBrutes(t *testing.T) {
	cases := map[string]func(i int) string{
		"exposant":     func(i int) string { return fmt.Sprintf(`"v":%de-7`, i+1) },
		"trop precis":  func(i int) string { return `"v":0.1234567891` },
		"melange":      func(i int) string { return fmt.Sprintf(`"v":%s`, []string{"1", `"x"`, "null", "true"}[i%4]) },
		"chaines":      func(i int) string { return fmt.Sprintf(`"v":"s%d"`, i) },
		"entier perdu": func(i int) string { return `"v":9007199254740993` },
		"tres grand":   func(i int) string { return `"v":12345678901234567890` },
	}
	for name, f := range cases {
		out := roundTrip(t, doc(rows(20, f)))
		if strings.Contains(out, `"s":`) || strings.Contains(out, `"d":1`) {
			t.Fatalf("%s : colonne mise à l'échelle alors qu'elle doit rester brute : %.300s", name, out)
		}
	}
	// le jeton brut est conservé caractère pour caractère
	out := roundTrip(t, doc(rows(16, func(i int) string { return `"v":9007199254740993` })))
	if !strings.Contains(out, "9007199254740993,9007199254740993") {
		t.Fatalf("le grand entier doit rester textuel : %s", out)
	}
}

func TestEncodeColumns_EntiersSurs(t *testing.T) {
	out := roundTrip(t, doc(rows(18, func(i int) string { return `"v":9007199254740991` })))
	if !strings.Contains(out, "9007199254740991") {
		t.Fatalf("2^53-1 est un entier sûr : %s", out)
	}
	// des différences qui dépasseraient 2^53 : la colonne reste en valeurs absolues
	out = roundTrip(t, doc(rows(20, func(i int) string {
		if i%2 == 0 {
			return `"v":9000000000000000`
		}
		return `"v":-9000000000000000`
	})))
	if strings.Contains(out, `"d":1`) {
		t.Fatalf("une différence hors des entiers sûrs ne doit pas être retenue : %s", out)
	}
}

func TestEncodeColumns_MoinsZeroDevientZero(t *testing.T) {
	out := roundTrip(t, doc(rows(20, func(i int) string {
		if i%3 == 0 {
			return `"v":-0`
		}
		return `"v":1.5`
	})))
	if strings.Contains(out, "-0,") || strings.Contains(out, "[-0") {
		t.Fatalf("-0 doit devenir 0 dans une colonne à échelle : %s", out)
	}
}

func TestEncodeColumns_Recursif(t *testing.T) {
	inner := rows(20, func(i int) string { return fmt.Sprintf(`"x":%d.25,"y":%d`, i, i*2) })
	body := doc(rows(18, func(i int) string {
		return fmt.Sprintf(`"id":%d,"pts":%s,"pair":[[1,2],[3,4]],"o":{"p":[1,2,3]},"e":{}`, i, inner)
	}))
	out := roundTrip(t, body)
	if strings.Count(out, `"$c"`) < 2 {
		t.Fatalf("la table imbriquée doit aussi être mise en colonnes : %.300s", out)
	}
}

func TestEncodeColumns_NUplets(t *testing.T) {
	pairs := make([]string, 30)
	for i := range pairs {
		pairs[i] = fmt.Sprintf("[%d,%d.5,%d]", i, i, 100-i)
	}
	out := roundTrip(t, doc("["+strings.Join(pairs, ",")+"]"))
	if !strings.Contains(out, `"a":3`) {
		t.Fatalf("n-uplets de trois nombres : %s", out)
	}
	// arité inégale, valeur non numérique, arité nulle : la liste reste une liste
	for _, mix := range []string{`[1,2,3]`, `[1,"a"]`, `[]`} {
		list := "[" + strings.Repeat("[1,2],", 20) + mix + "]"
		if out := roundTrip(t, doc(list)); strings.Contains(out, `"a":`) {
			t.Fatalf("liste de n-uplets irréguliers encodée en table : %s", out)
		}
	}
}

func TestEncodeColumns_ListesMelangees(t *testing.T) {
	roundTrip(t, doc("["+strings.Repeat(`{"a":1},`, 20)+`5]`))
	roundTrip(t, doc("["+strings.Repeat(`{"a":1},`, 20)+`null]`))
	roundTrip(t, doc("[1,"+strings.Repeat(`{"a":1},`, 20)+`{"a":2}]`))
	roundTrip(t, doc("["+strings.Repeat(`{"a":1},`, 17)+`[1,2]]`))
}

func TestEncodeColumns_OrdreDesCles(t *testing.T) {
	// l'ordre b, a est restitué même quand la première ligne ne porte que a
	out := roundTrip(t, doc(rows(20, func(i int) string {
		if i == 0 {
			return `"a":1`
		}
		return `"b":2,"a":1`
	})))
	if !strings.Contains(out, `"k":["b","a"]`) {
		t.Fatalf("ordre des clés : %s", out)
	}
	// ordres contradictoires : première apparition, jamais d'erreur
	enc, err := replayview.EncodeColumns([]byte(doc(rows(20, func(i int) string {
		if i%2 == 0 {
			return `"a":1,"b":2`
		}
		return `"b":2,"a":1`
	}))))
	if err != nil || !bytes.Contains(enc, []byte(`"k":["a","b"]`)) {
		t.Fatalf("ordre contradictoire : %v %s", err, enc)
	}
}

func TestEncodeColumns_Refus(t *testing.T) {
	collisions := []string{
		`{"$enc":1}`,
		`{"a":{"$c":1}}`,
		doc(rows(20, func(i int) string { return fmt.Sprintf(`"x":%d,"$c":1`, i) })),
		doc(rows(20, func(i int) string { return `"$k":1` })),
	}
	for _, body := range collisions {
		if _, err := replayview.EncodeColumns([]byte(body)); !errors.Is(err, replayview.ErrColumnsKeyCollision) {
			t.Fatalf("%.80s : attendu ErrColumnsKeyCollision, reçu %v", body, err)
		}
	}
	for _, body := range []string{``, `[1,2]`, `"x"`, `{"a":`, `{"a":1} x`, doc(rows(20, func(int) string { return `"a":1,"a":2` }))} {
		if _, err := replayview.EncodeColumns([]byte(body)); !errors.Is(err, replayview.ErrColumnsUnsupported) {
			t.Fatalf("%.80q : attendu ErrColumnsUnsupported, reçu %v", body, err)
		}
	}
}

func TestEncodeColumns_EspacesEtEchappements(t *testing.T) {
	body := "{ \"a\" : [ " + strings.Repeat(`{ "s" : "x\"y\\", "n" : 1 } , `, 20) + `{ "s" : "<&>", "n" : 2 } ] }` + "\n"
	roundTrip(t, body)
}

func TestEncodeColumns_NeModifiePasLEntree(t *testing.T) {
	in := []byte(doc(rows(20, func(i int) string { return fmt.Sprintf(`"x":%d`, i) })))
	cp := append([]byte(nil), in...)
	if _, err := replayview.EncodeColumns(in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, cp) {
		t.Fatal("le corps d'entrée a été modifié")
	}
}
