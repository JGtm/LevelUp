package i18n

import (
	"expvar"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func counter(t *testing.T, name string) int64 {
	t.Helper()
	v, ok := expvar.Get(name).(*expvar.Int)
	if !ok {
		t.Fatalf("compteur expvar %q absent", name)
	}
	return v.Value()
}

func TestTextPickOrder(t *testing.T) {
	txt := Text{"fr": "Plasma", "en": "Plasma Rifle"}
	if got := txt.Pick("fr"); got != "Plasma" {
		t.Errorf("Pick(fr) = %q", got)
	}
	if got := txt.Pick("en"); got != "Plasma Rifle" {
		t.Errorf("Pick(en) = %q", got)
	}
}

func TestTextPickFallsBackToFallbackAndCounts(t *testing.T) {
	txt := Text{Fallback: "Plasma Rifle", "fr": ""}
	before := counter(t, "levelup.i18n.text_fallback_total")

	for _, l := range []Locale{"fr", "zz", "es"} {
		got, fell := txt.Resolve(l)
		if got != "Plasma Rifle" || !fell {
			t.Errorf("Resolve(%q) = %q, %v, want (Plasma Rifle, true)", l, got, fell)
		}
	}
	if got := txt.Pick("zz"); got != "Plasma Rifle" {
		t.Errorf("Pick(zz) = %q", got)
	}
	if delta := counter(t, "levelup.i18n.text_fallback_total") - before; delta != 4 {
		t.Errorf("repli compté %d fois, want 4", delta)
	}
}

func TestTextPickFallbackLanguageItselfIsNotAFallback(t *testing.T) {
	txt := Text{Fallback: "Plasma Rifle"}
	before := counter(t, "levelup.i18n.text_fallback_total")
	if got, fell := txt.Resolve(Fallback); got != "Plasma Rifle" || fell {
		t.Errorf("Resolve(Fallback) = %q, %v", got, fell)
	}
	if counter(t, "levelup.i18n.text_fallback_total") != before {
		t.Error("lire la langue de repli ne doit pas compter un repli")
	}
}

func TestTextPickEmptyCountsAndReturnsEmpty(t *testing.T) {
	before := counter(t, "levelup.i18n.text_empty_total")
	for _, txt := range []Text{nil, {}, {"fr": "Plasma"}} {
		if got, fell := txt.Resolve("es"); got != "" || fell {
			t.Errorf("Resolve(es) sur %v = %q, %v, want (\"\", false)", txt, got, fell)
		}
	}
	if delta := counter(t, "levelup.i18n.text_empty_total") - before; delta != 3 {
		t.Errorf("texte vide compté %d fois, want 3", delta)
	}
}

func TestTextHasDoesNotCount(t *testing.T) {
	txt := Text{"fr": "x", "en": ""}
	before := counter(t, "levelup.i18n.text_empty_total")
	if !txt.Has("fr") || txt.Has("en") || txt.Has("es") {
		t.Error("Has : fr renseigné, en vide, es absent")
	}
	if counter(t, "levelup.i18n.text_empty_total") != before {
		t.Error("Has ne compte rien")
	}
}

func TestTextCloneAndTrimmed(t *testing.T) {
	var nilText Text
	if nilText.Clone() != nil || nilText.Trimmed() != nil {
		t.Error("nil reste nil")
	}
	src := Text{"fr": " a ", "en": "b"}
	c := src.Clone()
	c["fr"] = "z"
	if src["fr"] != " a " {
		t.Error("Clone doit être indépendant")
	}
	if got := src.Trimmed(); got["fr"] != "a" || got["en"] != "b" || src["fr"] != " a " {
		t.Errorf("Trimmed = %v (source %v)", got, src)
	}
}

func TestTextValidate(t *testing.T) {
	if err := (Text{"fr": "a", "en": "b"}).Validate(); err != nil {
		t.Errorf("texte complet refusé : %v", err)
	}
	err := (Text{"en": "b"}).Validate()
	if err == nil || !strings.Contains(err.Error(), `"fr"`) || strings.Contains(err.Error(), `"en"`) {
		t.Errorf("fr manquant doit être nommé seul : %v", err)
	}
	err = (Text{"fr": " ", "en": ""}).Validate()
	if err == nil || !strings.Contains(err.Error(), `"fr"`) || !strings.Contains(err.Error(), `"en"`) {
		t.Errorf("valeurs vides refusées : %v", err)
	}
	if err = (Text(nil)).Validate(); err == nil {
		t.Error("un texte nil doit être refusé : toutes les langues required manquent")
	}
	err = (Text{"fr": "a", "en": "b", "zz": "c"}).Validate()
	if err == nil || !strings.Contains(err.Error(), `"zz"`) {
		t.Errorf("langue hors liste doit être nommée : %v", err)
	}
}

func TestTextDecodesFromTOMLTable(t *testing.T) {
	var doc struct {
		Title Text            `toml:"title"`
		Names map[string]Text `toml:"names"`
	}
	src := `
title = { en = "Hello", fr = "Bonjour" }
[names]
a = { en = "A", fr = "Ah" }
`
	if err := toml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("decode : %v", err)
	}
	if doc.Title["fr"] != "Bonjour" || doc.Title["en"] != "Hello" {
		t.Errorf("Title = %v", doc.Title)
	}
	if doc.Names["a"].Pick("fr") != "Ah" {
		t.Errorf("Names = %v", doc.Names)
	}
	if err := doc.Title.Validate(); err != nil {
		t.Errorf("Validate : %v", err)
	}
}

func TestTextCheckKnown(t *testing.T) {
	if err := (Text{"fr": "a"}).CheckKnown(); err != nil {
		t.Errorf("table partielle d'une langue connue refusée : %v", err)
	}
	if err := (Text(nil)).CheckKnown(); err != nil {
		t.Errorf("texte nil refusé : %v", err)
	}
	err := (Text{"fr": "a", "zz": "b", "yy": "c"}).CheckKnown()
	if err == nil || !strings.Contains(err.Error(), `"yy" "zz"`) {
		t.Errorf("langues inconnues attendues, triées : %v", err)
	}
}

func TestTextEmpty(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   Text
		want bool
	}{
		{"nil", nil, true},
		{"vide", Text{}, true},
		{"valeurs vides", Text{LocaleFR: "", LocaleEN: ""}, true},
		{"une langue", Text{LocaleFR: "Or"}, false},
	}
	for _, c := range cases {
		if got := c.in.Empty(); got != c.want {
			t.Errorf("%s : Empty() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFromColumns(t *testing.T) {
	t.Parallel()
	got := FromColumns("Gold", "")
	if len(got) != 1 || got[LocaleEN] != "Gold" {
		t.Errorf("FromColumns(Gold, \"\") = %v, want une seule entrée en", got)
	}
	if got := FromColumns("Gold", "Or"); got[LocaleFR] != "Or" || got[LocaleEN] != "Gold" {
		t.Errorf("FromColumns(Gold, Or) = %v", got)
	}
	if !FromColumns("", "").Empty() {
		t.Errorf("FromColumns vide doit être Empty")
	}
}

func TestTextPtr(t *testing.T) {
	t.Parallel()
	txt := Text{LocaleFR: "Or", LocaleEN: ""}
	if p := txt.Ptr(LocaleFR); p == nil || *p != "Or" {
		t.Errorf("Ptr(fr) = %v, want Or", p)
	}
	if p := txt.Ptr(LocaleEN); p != nil {
		t.Errorf("Ptr(en) = %v, want nil (valeur vide)", *p)
	}
	if p := Text(nil).Ptr(LocaleFR); p != nil {
		t.Errorf("Ptr sur nil = %v, want nil", *p)
	}
}

func TestTextSet(t *testing.T) {
	t.Parallel()
	var txt Text
	txt.Set(LocaleFR, "Or")
	if txt[LocaleFR] != "Or" {
		t.Errorf("Set sur un Text nil : %v", txt)
	}
	txt.Set(LocaleEN, "Gold")
	if len(txt) != 2 || txt[LocaleEN] != "Gold" {
		t.Errorf("Set d'une 2e langue : %v", txt)
	}
}

func TestTextEqual(t *testing.T) {
	t.Parallel()
	a := Text{LocaleFR: "Or", LocaleEN: "Gold"}
	if !a.Equal(Text{LocaleEN: "Gold", LocaleFR: "Or"}) {
		t.Errorf("deux textes identiques doivent être égaux")
	}
	if a.Equal(Text{LocaleFR: "Or"}) {
		t.Errorf("une langue en moins ne doit pas être égale")
	}
	if a.Equal(Text{LocaleFR: "Or", LocaleEN: ""}) {
		t.Errorf("une valeur vide n'est pas une valeur identique")
	}
	if !Text(nil).Equal(Text{}) {
		t.Errorf("nil et vide sont égaux (aucune langue)")
	}
}

func TestTextCompareOrdreAlphabetiqueDesCodes(t *testing.T) {
	a := Text{LocaleEN: "a", LocaleFR: "z"}
	b := Text{LocaleEN: "a", LocaleFR: "b"}
	if got := a.Compare(b); got <= 0 {
		t.Errorf("Compare(a, b) = %d, attendu > 0 (fr différencie après en)", got)
	}
	if got := b.Compare(a); got >= 0 {
		t.Errorf("Compare(b, a) = %d, attendu < 0", got)
	}
	if got := a.Compare(Text{LocaleEN: "a", LocaleFR: "z", Locale("zz"): "ignoré"}); got != 0 {
		t.Errorf("une clé hors liste ne se compare pas : %d", got)
	}
	// en prime sur fr : l'ordre des anciens champs En puis Fr.
	c := Text{LocaleEN: "b", LocaleFR: "a"}
	if got := c.Compare(a); got <= 0 {
		t.Errorf("Compare(c, a) = %d, attendu > 0 (en décide avant fr)", got)
	}
}

// PickOptional : un champ facultatif absent rend "" sans compter de texte vide ; présent, il se
// comporte comme Pick (langue, puis repli).
func TestTextPickOptional(t *testing.T) {
	emptyBefore := counter(t, "levelup.i18n.text_empty_total")
	if got := (Text(nil)).PickOptional("fr"); got != "" {
		t.Errorf("PickOptional(nil) = %q, want vide", got)
	}
	if got := (Text{"fr": ""}).PickOptional("fr"); got != "" {
		t.Errorf("PickOptional(vide) = %q, want vide", got)
	}
	if after := counter(t, "levelup.i18n.text_empty_total"); after != emptyBefore {
		t.Errorf("PickOptional d'un champ absent a compté un texte vide : %d -> %d", emptyBefore, after)
	}
	txt := Text{Fallback: "Hold accuracy"}
	if got := txt.PickOptional("fr"); got != "Hold accuracy" {
		t.Errorf("PickOptional(fr) = %q, want le repli en", got)
	}
}
