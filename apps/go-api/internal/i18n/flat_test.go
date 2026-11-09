package i18n_test

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// Trois parents qui reprennent les trois places où le document de rejeu écrit ses langues
// (au début, au milieu, à la fin), chacun face à son jumeau « d'avant » à deux champs En / Fr :
// la forme de fil doit être la même octet pour octet.

type startNew struct {
	Text i18n.Text `json:"i18n_text,omitempty" required:"true"`
	Img  string    `json:"img,omitempty"`
	Fx   string    `json:"fx,omitempty"`
}
type startOld struct {
	En  string `json:"en"`
	Fr  string `json:"fr"`
	Img string `json:"img,omitempty"`
	Fx  string `json:"fx,omitempty"`
}
type startWire startNew

func (s startNew) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(startWire(s), i18n.FlatLayout{})
}
func (s *startNew) UnmarshalJSON(b []byte) error { return i18n.UnmarshalFlat(b, (*startWire)(s)) }

type midNew struct {
	Index int       `json:"index"`
	Name  string    `json:"name"`
	Text  i18n.Text `json:"i18n_text,omitempty" required:"true"`
	X     float64   `json:"x"`
	Big   bool      `json:"big,omitempty"`
}
type midOld struct {
	Index int     `json:"index"`
	Name  string  `json:"name"`
	En    string  `json:"en"`
	Fr    string  `json:"fr"`
	X     float64 `json:"x"`
	Big   bool    `json:"big,omitempty"`
}
type midWire midNew

func (m midNew) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(midWire(m), i18n.FlatLayout{After: "name"})
}
func (m *midNew) UnmarshalJSON(b []byte) error { return i18n.UnmarshalFlat(b, (*midWire)(m)) }

type endNew struct {
	Img  string    `json:"img,omitempty"`
	Kind string    `json:"kind,omitempty"`
	Text i18n.Text `json:"i18n_text,omitempty"`
}
type endOld struct {
	Img  string `json:"img,omitempty"`
	Kind string `json:"kind,omitempty"`
	En   string `json:"en,omitempty"`
	Fr   string `json:"fr,omitempty"`
}
type endWire endNew

func (e endNew) MarshalJSON() ([]byte, error) {
	return i18n.MarshalFlat(endWire(e), i18n.FlatLayout{After: i18n.FlatEnd, OmitEmpty: true})
}
func (e *endNew) UnmarshalJSON(b []byte) error { return i18n.UnmarshalFlat(b, (*endWire)(e)) }

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T : %v", v, err)
	}
	return string(b)
}

// langueHorsEnFr écrit les langues de la liste autres que en et fr : le jumeau « d'avant » n'en a pas, et
// MarshalFlat émet toutes les langues de la liste (required:"true") ; la comparaison porte sur en et fr.
var langueHorsEnFr = func() []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, c := range i18n.FlatCodes() {
		if c == i18n.LocaleEN || c == i18n.LocaleFR {
			continue
		}
		q := regexp.QuoteMeta(string(c))
		membre := `"` + q + `":"(?:[^"\\]|\\.)*"`
		out = append(out, regexp.MustCompile(`,`+membre), regexp.MustCompile(membre+`,`))
	}
	return out
}()

// enFrSeulement retire d'un fil les langues autres que en et fr.
func enFrSeulement(raw string) string {
	for _, re := range langueHorsEnFr {
		raw = re.ReplaceAllString(raw, "")
	}
	return raw
}

// enFrDe garde les langues en et fr non vides d'un Text.
func enFrDe(tx i18n.Text) map[i18n.Locale]string {
	out := map[i18n.Locale]string{}
	for _, l := range []i18n.Locale{i18n.LocaleEN, i18n.LocaleFR} {
		if v := tx[l]; v != "" {
			out[l] = v
		}
	}
	return out
}

func TestFlatMemeFilQueLAncienneForme(t *testing.T) {
	texts := []struct {
		name   string
		en, fr string
	}{
		{"deux langues", "Rocket Launcher", "Lance-roquettes"},
		{"fr vide", "Sentinel", ""},
		{"en vide", "", "Sentinelle"},
		{"aucune", "", ""},
		{"echappement", "A <b> & \"c\"", "é x"},
	}
	for _, c := range texts {
		t.Run(c.name, func(t *testing.T) {
			tx := i18n.Text{}
			if c.en != "" {
				tx[i18n.LocaleEN] = c.en
			}
			if c.fr != "" {
				tx[i18n.LocaleFR] = c.fr
			}
			cases := []struct {
				nom    string
				neuf   any
				ancien any
			}{
				{"debut", startNew{Text: tx, Img: "i.png", Fx: "plasma"}, startOld{c.en, c.fr, "i.png", "plasma"}},
				{"debut sans reste", startNew{Text: tx}, startOld{En: c.en, Fr: c.fr}},
				{"milieu", midNew{Index: 3, Name: "n", Text: tx, X: 1.5, Big: true}, midOld{3, "n", c.en, c.fr, 1.5, true}},
				{"milieu sans suite", midNew{Index: 3, Name: "n", Text: tx}, midOld{Index: 3, Name: "n", En: c.en, Fr: c.fr}},
				{"fin", endNew{Img: "i", Kind: "k", Text: tx}, endOld{"i", "k", c.en, c.fr}},
				{"fin sans reste", endNew{Text: tx}, endOld{En: c.en, Fr: c.fr}},
			}
			for _, k := range cases {
				if got, want := enFrSeulement(mustJSON(t, k.neuf)), mustJSON(t, k.ancien); got != want {
					t.Errorf("%s : fil %s, attendu %s", k.nom, got, want)
				}
			}
		})
	}
}

func TestFlatAllerRetour(t *testing.T) {
	in := midNew{Index: 7, Name: "Dome", Text: i18n.Text{i18n.LocaleEN: "Dome", i18n.LocaleFR: "Dôme"}, X: 2, Big: true}
	raw := mustJSON(t, in)
	var out midNew
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal : %v", err)
	}
	if out.Index != 7 || out.Name != "Dome" || out.X != 2 || !out.Big || !reflect.DeepEqual(enFrDe(out.Text), enFrDe(in.Text)) {
		t.Errorf("relecture = %+v", out)
	}
	if again := mustJSON(t, out); again != raw {
		t.Errorf("aller-retour : %s puis %s", raw, again)
	}
}

// Une relecture dans une variable déjà remplie repart de zéro : pas de langue qui survit.
func TestFlatRelectureRepartDeZero(t *testing.T) {
	out := startNew{Text: i18n.Text{i18n.LocaleEN: "old", i18n.LocaleFR: "ancien"}}
	if err := json.Unmarshal([]byte(`{"en":"new"}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.Text.Has(i18n.LocaleFR) || out.Text[i18n.LocaleEN] != "new" {
		t.Errorf("Text = %v, attendu seulement en=new", out.Text)
	}
}

func TestFlatMembreHorsListeIgnore(t *testing.T) {
	var out startNew
	if err := json.Unmarshal([]byte(`{"en":"a","zz":"b","img":"i"}`), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Text) != 1 || out.Text[i18n.LocaleEN] != "a" || out.Img != "i" {
		t.Errorf("relecture = %+v", out)
	}
}

func TestFlatLangueVideEstDistincteDAbsente(t *testing.T) {
	var out startNew
	if err := json.Unmarshal([]byte(`{"en":"a","fr":""}`), &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.Text[i18n.LocaleFR]; !ok || v != "" {
		t.Errorf("fr vide : Text = %v", out.Text)
	}
}

func TestFlatMembreAbsentEstUneErreur(t *testing.T) {
	// `Name` est omitempty ici : absent du corps quand vide, le positionnement échoue
	// bruyamment plutôt que d'écrire les langues au mauvais endroit.
	type wire struct {
		Name string    `json:"name,omitempty"`
		Text i18n.Text `json:"i18n_text,omitempty"`
	}
	_, err := i18n.MarshalFlat(wire{Text: i18n.Text{i18n.LocaleEN: "x"}}, i18n.FlatLayout{After: "name"})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("erreur = %v, attendu un refus nommant le membre", err)
	}
}

func TestFlatSansChampTextEstUneErreur(t *testing.T) {
	type wire struct{ A string }
	if _, err := i18n.MarshalFlat(wire{}, i18n.FlatLayout{}); err == nil {
		t.Error("un struct sans champ Text doit être refusé")
	}
}

func TestFlatNeModifiePasLEntree(t *testing.T) {
	in := startNew{Text: i18n.Text{i18n.LocaleEN: "a"}}
	_ = mustJSON(t, in)
	if !in.Text.Has(i18n.LocaleEN) {
		t.Error("MarshalFlat a vidé le Text de l'appelant")
	}
}

func TestFlatCodesAlphabetiques(t *testing.T) {
	codes := i18n.FlatCodes()
	for i := 1; i < len(codes); i++ {
		if codes[i-1] >= codes[i] {
			t.Errorf("FlatCodes() = %v : pas alphabétique", codes)
		}
	}
	if len(codes) != len(i18n.Locales()) {
		t.Errorf("FlatCodes() = %v, Locales() = %v", codes, i18n.Locales())
	}
}
