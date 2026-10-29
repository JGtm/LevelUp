package flatjson

import (
	"encoding/json"
	"strings"
	"testing"
)

// flat_test.go — LES CAS LIMITES DU CODEC, repris de `internal/i18n/flat_test.go` (copie jumelle).
// L'égalité des octets avec l'original vit côté LevelUp (`replaylabels/fil_jumeau_test.go`).

type texte map[string]string

type milieu struct {
	Index int     `json:"index"`
	Name  string  `json:"name"`
	Text  texte   `json:"i18n_text,omitempty"`
	X     float64 `json:"x"`
}

type milieuWire milieu

func (m milieu) MarshalJSON() ([]byte, error) {
	return MarshalFlat[texte](milieuWire(m), FlatLayout{After: "name"})
}

func (m *milieu) UnmarshalJSON(b []byte) error { return UnmarshalFlat[texte](b, (*milieuWire)(m)) }

func TestAllerRetourAuMilieu(t *testing.T) {
	in := milieu{Index: 7, Name: "Dome", Text: texte{"en": "Dome", "fr": "Dôme"}, X: 2}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"index":7,"name":"Dome","en":"Dome","fr":"Dôme","x":2}`; string(raw) != want {
		t.Fatalf("fil %s, attendu %s", raw, want)
	}
	var out milieu
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	again, _ := json.Marshal(out)
	if string(again) != string(raw) {
		t.Errorf("aller-retour : %s puis %s", raw, again)
	}
}

func TestMembreAbsentEstUneErreur(t *testing.T) {
	type wire struct {
		Name string `json:"name,omitempty"`
		Text texte  `json:"i18n_text,omitempty"`
	}
	_, err := MarshalFlat[texte](wire{Text: texte{"en": "x"}}, FlatLayout{After: "name"})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("erreur = %v, attendu un refus nommant le membre", err)
	}
}

func TestSansChampTexteEstUneErreur(t *testing.T) {
	type wire struct{ A string }
	if _, err := MarshalFlat[texte](wire{}, FlatLayout{}); err == nil {
		t.Error("un struct sans champ texte doit être refusé")
	}
	if err := UnmarshalFlat[texte]([]byte(`{}`), &wire{}); err == nil {
		t.Error("un struct sans champ texte doit être refusé à la lecture")
	}
}

func TestNeModifiePasLEntree(t *testing.T) {
	in := milieu{Name: "n", Text: texte{"en": "a"}}
	if _, err := json.Marshal(in); err != nil {
		t.Fatal(err)
	}
	if in.Text["en"] != "a" {
		t.Error("MarshalFlat a vidé le texte de l'appelant")
	}
}

func TestFormeDUnCode(t *testing.T) {
	for membre, want := range map[string]bool{
		"en": true, "fil": true, "e": false, "engl": false, "En": false, "e1": false, "": false,
	} {
		if got := aLaFormeDUnCode(membre); got != want {
			t.Errorf("aLaFormeDUnCode(%q) = %v, attendu %v", membre, got, want)
		}
	}
}
