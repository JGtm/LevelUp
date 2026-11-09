package i18n

import (
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// placeholdersOf retourne l'ensemble trié des variables `{nom}` d'un modèle.
func placeholdersOf(tpl string) []string {
	seen := map[string]bool{}
	for _, m := range placeholderRE.FindAllStringSubmatch(tpl, -1) {
		seen[m[1]] = true
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// TestCatalogueComplet : toute clé du catalogue a un texte non vide dans CHAQUE langue de
// l'application (D3 : complète ou refusée), avec les mêmes variables partout. Une langue ajoutée à
// config/locales.toml fait échouer ce test tant qu'un message n'est pas traduit.
func TestCatalogueComplet(t *testing.T) {
	keys := MessageKeys()
	if len(keys) == 0 {
		t.Fatal("catalogue vide")
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			text := catalog[key]
			ref := placeholdersOf(text[Default])
			for _, l := range Locales() {
				if strings.TrimSpace(text[l]) == "" {
					t.Errorf("langue %q manquante ou vide", l)
					continue
				}
				if got := placeholdersOf(text[l]); strings.Join(got, ",") != strings.Join(ref, ",") {
					t.Errorf("langue %q : variables %v, attendu %v (celles de %q)", l, got, ref, Default)
				}
			}
		})
	}
}

func TestMsg_RemplaceLesVariables(t *testing.T) {
	got := Msg(LocaleEN, "coach.window.last_n_matches", Params{"n": 20})
	if got != "the last 20 matches" {
		t.Errorf("en : %q", got)
	}
	got = Msg(LocaleFR, "coach.window.last_n_matches", Params{"n": "20"})
	if got != "les 20 derniers matchs" {
		t.Errorf("fr : %q", got)
	}
}

func TestMsg_VariableAbsenteResteVisibleEtCompte(t *testing.T) {
	before := messageParamMissingTotal.Value()
	got := Msg(LocaleEN, "coach.window.last_n_matches", nil)
	if got != "the last {n} matches" {
		t.Errorf("variable absente : %q", got)
	}
	if messageParamMissingTotal.Value() != before+1 {
		t.Errorf("compteur de variable absente non incrémenté")
	}
}

func TestMsg_CleInconnueRendLaCleEtCompte(t *testing.T) {
	before := messageMissingTotal.Value()
	if got := Msg(LocaleFR, "n.existe.pas", nil); got != "n.existe.pas" {
		t.Errorf("clé inconnue : %q", got)
	}
	if messageMissingTotal.Value() != before+1 {
		t.Errorf("compteur de clé absente non incrémenté")
	}
}

func TestMsg_LangueInconnueReplieSurLeRepli(t *testing.T) {
	// Une langue hors liste a une entrée vide dans chaque message : le repli est pris et compté.
	before := messageFallbackTotal.Value()
	got := Msg(Locale("zz"), "common.unknown_weapon", nil)
	if got != Msg(Fallback, "common.unknown_weapon", nil) {
		t.Errorf("langue inconnue : %q, attendu le texte de %q", got, Fallback)
	}
	if messageFallbackTotal.Value() != before+1 {
		t.Errorf("compteur de repli non incrémenté")
	}
}

func TestMsgText_ToutesLesLangues(t *testing.T) {
	text := MsgText("common.unknown_weapon", nil)
	for _, l := range Locales() {
		if text[l] == "" {
			t.Errorf("langue %q absente de MsgText", l)
		}
	}
	if text[LocaleFR] != "Inconnue" || text[LocaleEN] != "Unknown" {
		t.Errorf("textes : %v", text)
	}
}

func TestMsgTextWith_UneVariableParLangue(t *testing.T) {
	text := MsgTextWith("coach.arc.description", func(l Locale) Params {
		return Params{"steps": 3, "axis": Msg(l, "coach.axis.combat", nil)}
	})
	if text[LocaleEN] != "A 3-step arc to consolidate your combat progress." {
		t.Errorf("en : %q", text[LocaleEN])
	}
	if text[LocaleFR] != "Un arc en 3 étapes pour consolider ta progression en combat." {
		t.Errorf("fr : %q", text[LocaleFR])
	}
}

func TestMsgAll_FormesDistinctes(t *testing.T) {
	got := MsgAll("release_notes.heading")
	if len(got) < 2 {
		t.Errorf("deux formes attendues au moins : %v", got)
	}
	if MsgAll("n.existe.pas") != nil {
		t.Errorf("clé inconnue : nil attendu")
	}
}

func TestPickTemplate_RepliParOrdre(t *testing.T) {
	text := Text{Default: "défaut"}
	if got := pickTemplate(text, Fallback); got != "défaut" {
		t.Errorf("sans repli : %q", got)
	}
	text[Fallback] = "repli"
	if got := pickTemplate(text, Locale("zz")); got != "repli" {
		t.Errorf("langue absente : %q", got)
	}
	if got := pickTemplate(Text{}, Fallback); got != "" {
		t.Errorf("texte vide : %q", got)
	}
}

func TestLoadCatalog_Refus(t *testing.T) {
	cases := map[string]string{
		"langue inconnue":      "[a.b]\nfr = \"x\"\nzz = \"y\"\n",
		"feuille et branche":   "[a]\nfr = \"x\"\n[a.b]\nfr = \"y\"\n",
		"texte hors table":     "fr = \"x\"\n",
		"valeur non textuelle": "[a.b]\nfr = 3\n",
		"toml illisible":       "[a.b\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{"messages/x.toml": {Data: []byte(doc)}}
			if _, err := loadCatalog(fsys); err == nil {
				t.Errorf("refus attendu pour %q", doc)
			}
		})
	}
}

func TestLoadCatalog_CleEnDouble(t *testing.T) {
	fsys := fstest.MapFS{
		"messages/a.toml": {Data: []byte("[k.v]\nfr = \"a\"\nen = \"a\"\n")},
		"messages/b.toml": {Data: []byte("[k.v]\nfr = \"b\"\nen = \"b\"\n")},
	}
	if _, err := loadCatalog(fsys); err == nil {
		t.Error("clé déclarée dans deux fichiers : refus attendu")
	}
}

func TestLoadCatalog_Aplatit(t *testing.T) {
	fsys := fstest.MapFS{"messages/a.toml": {Data: []byte("[k.v]\nfr = \"a\"\nen = \"b\"\n")}}
	c, err := loadCatalog(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if c["k.v"][LocaleFR] != "a" || c["k.v"][LocaleEN] != "b" {
		t.Errorf("aplati : %v", c)
	}
}
