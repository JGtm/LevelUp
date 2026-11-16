package replayview_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/service/replayview"
)

// Couverture de TOUS les champs du document servi par l'aller-retour de l'encodage en colonnes.
//
// L'encodeur est générique (aucune table par champ) : un champ ajouté au document est encodé sans
// qu'on le déclare. Ce fichier le prouve, sans liste de champs à tenir :
//
//   - un document « plein » est fabriqué par réflexion sur `replaydoc.ReplayDocument` : chaque
//     champ exporté est rempli, chaque liste porte assez d'éléments pour être mise en colonnes,
//     et un élément sur trois est laissé à zéro, ce qui fait manquer les champs `omitempty` sur
//     une partie des lignes (l'absence). Un champ neuf du document y entre donc tout seul ;
//   - ce document traverse `EncodeColumns` puis le décodeur de test, et doit revenir égal, clés et
//     ordre compris ;
//   - les fixtures réelles du web (schéma courant) sont comparées à ce document plein : les
//     chemins que les fixtures ne portent pas sont les seuls à ne reposer que sur lui, et leur
//     liste est journalisée.

const (
	champsLongueurListe = 16 // seuil de mise en colonnes (columnsMinRows)
	champsProfondeurMax = 12
	champsBudgetNoeuds  = 400000
)

// fabriquePleine fabrique un document plein de façon déterministe.
type fabriquePleine struct {
	n       int // compteur de valeurs distinctes
	noeuds  int // budget : coupe l'explosion des listes imbriquées
	locales []i18n.Locale
}

func (r *fabriquePleine) peupler(v reflect.Value, profondeur int) {
	if profondeur > champsProfondeurMax || r.noeuds > champsBudgetNoeuds {
		return
	}
	r.noeuds++
	switch v.Kind() {
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		r.peupler(v.Elem(), profondeur+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !f.IsExported() || f.Tag.Get("json") == "-" {
				continue
			}
			r.peupler(v.Field(i), profondeur+1)
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 { // []byte : base64, une chaîne
			v.SetBytes([]byte{byte(r.n), byte(r.n >> 8), 7})
			r.n++
			return
		}
		longueur := champsLongueurListe + 1
		if r.noeuds > champsBudgetNoeuds/2 {
			longueur = 3 // au-delà du budget, les listes profondes restent des listes
		}
		s := reflect.MakeSlice(v.Type(), longueur, longueur)
		for i := 0; i < longueur; i++ {
			if i%3 == 2 {
				continue // élément à zéro : les champs omitempty manquent sur cette ligne
			}
			r.peupler(s.Index(i), profondeur+1)
		}
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			r.peupler(v.Index(i), profondeur+1)
		}
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		for _, k := range r.clesDeCarte(v.Type().Key()) {
			e := reflect.New(v.Type().Elem()).Elem()
			r.peupler(e, profondeur+1)
			m.SetMapIndex(k, e)
		}
		v.Set(m)
	case reflect.String:
		r.n++
		v.SetString("s" + itoa(r.n))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		r.n++
		v.SetInt(int64(r.n*37%1000 + 1))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		r.n++
		v.SetUint(uint64(r.n*37%1000 + 1))
	case reflect.Float32, reflect.Float64:
		r.n++
		v.SetFloat(float64(r.n*37%100000+1) / 100) // deux décimales, valeurs sans bruit flottant
	}
}

func (r *fabriquePleine) clesDeCarte(t reflect.Type) []reflect.Value {
	if t == reflect.TypeOf(i18n.Locale("")) {
		out := make([]reflect.Value, 0, len(r.locales))
		for _, l := range r.locales {
			out = append(out, reflect.ValueOf(l))
		}
		return out
	}
	out := make([]reflect.Value, 0, champsLongueurListe)
	for i := 0; i < champsLongueurListe; i++ {
		k := reflect.New(t).Elem()
		if k.Kind() == reflect.String {
			k.SetString("k" + itoa(i))
		} else {
			k.SetInt(int64(i + 1))
		}
		out = append(out, k)
	}
	return out
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func documentPlein(t *testing.T) []byte {
	t.Helper()
	r := &fabriquePleine{locales: []i18n.Locale{"fr", "en"}}
	var d replaydoc.ReplayDocument
	r.peupler(reflect.ValueOf(&d).Elem(), 0)
	d.SchemaVersion = 125
	body, err := json.Marshal(&d)
	if err != nil {
		t.Fatalf("document plein : %v", err)
	}
	return body
}

// cheminsDeCles liste les chemins de clés d'un document JSON, listes repliées en `[]`. Les clés des
// cartes dont l'identifiant est libre (`mapPaths`) sont repliées en `{}`.
func cheminsDeCles(v any, chemin string, mapPaths map[string]bool, out map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			seg := k
			if mapPaths[chemin] {
				seg = "{}"
			}
			c := chemin + "." + seg
			out[c] = true
			cheminsDeCles(e, c, mapPaths, out)
		}
	case []any:
		for _, e := range x {
			cheminsDeCles(e, chemin+"[]", mapPaths, out)
		}
	}
}

// cartesDuType rend les chemins (même notation) des champs de type carte du document.
func cartesDuType(t reflect.Type, chemin string, vus map[reflect.Type]bool, out map[string]bool) {
	switch t.Kind() {
	case reflect.Ptr:
		cartesDuType(t.Elem(), chemin, vus, out)
	case reflect.Slice, reflect.Array:
		cartesDuType(t.Elem(), chemin+"[]", vus, out)
	case reflect.Map:
		if t.Key() != reflect.TypeOf(i18n.Locale("")) {
			out[chemin] = true
		}
		cartesDuType(t.Elem(), chemin+".{}", vus, out)
	case reflect.Struct:
		if vus[t] {
			return
		}
		vus[t] = true
		defer delete(vus, t)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			if !f.IsExported() || tag == "-" {
				continue
			}
			if tag == "" {
				tag = f.Name
			}
			cartesDuType(f.Type, chemin+"."+tag, vus, out)
		}
	}
}

func TestEncodeColumns_DocumentPleinAllerRetour(t *testing.T) {
	body := documentPlein(t)
	enc, err := replayview.EncodeColumns(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) >= len(body) {
		t.Fatalf("l'encodage en colonnes ne réduit rien (%d -> %d octets)", len(body), len(enc))
	}
	if p, ok := equalOrdered(parseOrdered(t, body), decodeColumnsDoc(t, parseOrdered(t, enc)), "$"); !ok {
		t.Fatalf("aller-retour différent en %s", p)
	}
	t.Logf("document plein : %d octets -> %d octets en colonnes", len(body), len(enc))
}

func TestEncodeColumns_FixturesCouvrentLeDocument(t *testing.T) {
	mapPaths := map[string]bool{}
	cartesDuType(reflect.TypeOf(replaydoc.ReplayDocument{}), "", map[reflect.Type]bool{}, mapPaths)

	var plein any
	if err := json.Unmarshal(documentPlein(t), &plein); err != nil {
		t.Fatal(err)
	}
	attendus := map[string]bool{}
	cheminsDeCles(plein, "", mapPaths, attendus)

	vus := map[string]bool{}
	for film, path := range webFixtureFiles(t) {
		var doc any
		if err := json.Unmarshal(readGzipFile(t, path), &doc); err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		cheminsDeCles(doc, "", mapPaths, vus)
	}

	var seuls []string
	for c := range attendus {
		if !vus[c] {
			seuls = append(seuls, c)
		}
	}
	sort.Strings(seuls)
	// Un chemin des fixtures que le document plein ignore serait un champ hors du type : l'inverse
	// du défaut recherché, mais il dirait que le type et le document servi divergent.
	var hors []string
	for c := range vus {
		if !attendus[c] {
			hors = append(hors, c)
		}
	}
	sort.Strings(hors)
	t.Logf("chemins du document : %d ; portés par les fixtures du web : %d ; portés par le seul document plein : %d", len(attendus), len(attendus)-len(seuls), len(seuls))
	if len(seuls) > 0 {
		t.Logf("sans fixture réelle (couverts par le document plein) : %s", strings.Join(seuls, ", "))
	}
	if len(hors) > 0 {
		t.Fatalf("chemins des fixtures absents du type du document : %s", strings.Join(hors, ", "))
	}
}

// Le document plein est aussi publié pour le web (`fixtures/go_columns/`) : la source en forme
// normale et sa forme en colonnes, écrite par l'encodeur. Le web les compare après décodage, ce
// qui lui fait lire, dans l'ordre exact du serveur, chaque champ du document courant, y compris
// ceux qu'aucun film réel de `fixtures/go/` ne porte (objectifs, horloges, compteurs à rebours,
// bombes). Régénérer avec `LEVELUP_REGEN_COLUMNS_FIXTURES=1` (voir columns_fixtures_test.go).
func TestColumnsFixturePleineMatchCommitted(t *testing.T) {
	source := documentPlein(t)
	enc, err := replayview.EncodeColumns(source)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(webFixturesRoot(t), "go_columns")
	cibles := map[string][]byte{"plein.json.gz": source, "columns_plein.json.gz": enc}
	for nom, contenu := range cibles {
		dst := filepath.Join(dir, nom)
		if os.Getenv(envRegenColumnsFixtures) == "1" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, gzipBytes(t, contenu), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if !bytes.Equal(readGzipFile(t, dst), contenu) {
			t.Fatalf("%s n'est plus celui que produit le document plein ; régénérer avec %s=1", nom, envRegenColumnsFixtures)
		}
	}
}
