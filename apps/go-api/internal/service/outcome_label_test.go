// Tests — LE MOT DE L'ISSUE, dit par le titre et dans la langue de la requête (2026-09-07).
//
// CE QU'ILS PROTÈGENT. Le libellé sortait d'une map Go écrite en français : sous UI anglaise
// l'en-tête de la Match View annonçait « Victoire », pendant que l'écran de fin du rejeu, vu
// depuis un adversaire, prenait son titre dans `outcomes.toml` et disait « Loss ». Deux
// vocabulaires sur un seul panneau. Ces cas fixent la règle inverse : un seul mot, celui du
// TOML du titre, dans la locale demandée — et le repli FR seulement quand le titre se tait.
package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

// outcomesDeTest reproduit le `outcomes.toml` d'un titre : les quatre issues canoniques, leurs
// deux locales, et le `raw_code` qui fait le pont depuis l'entier stocké en base (2/3/1/4).
func outcomesDeTest() *mappings.OutcomeMappingSet {
	return mappings.NewOutcomeMappingSet("titre_de_test", 1, map[string]mappings.OutcomeMapping{
		"win":  {Key: "win", Labels: map[string]string{"en": "Victory", "fr": "Victoire"}, RawCode: domain.OutcomeWin},
		"loss": {Key: "loss", Labels: map[string]string{"en": "Defeat", "fr": "Défaite"}, RawCode: domain.OutcomeLoss},
		"tie":  {Key: "tie", Labels: map[string]string{"en": "Tie", "fr": "Égalité"}, RawCode: domain.OutcomeDraw},
		"dnf":  {Key: "dnf", Labels: map[string]string{"en": "DNF", "fr": "Abandon"}, RawCode: domain.OutcomeDNF},
	})
}

func ctxLocale(locale string) context.Context {
	return ctxkeys.WithTitleSlug(ctxkeys.WithLocale(context.Background(), locale), "titre_de_test")
}

func TestResolveOutcomeLabel_LocaleDeLaRequete(t *testing.T) {
	outcomes := outcomesDeTest()
	cas := []struct {
		locale string
		code   int
		want   string
	}{
		{"fr", domain.OutcomeWin, "Victoire"},
		{"fr", domain.OutcomeLoss, "Défaite"},
		{"fr", domain.OutcomeDraw, "Égalité"},
		{"fr", domain.OutcomeDNF, "Abandon"},
		{"en", domain.OutcomeWin, "Victory"},
		{"en", domain.OutcomeLoss, "Defeat"},
		{"en", domain.OutcomeDraw, "Tie"},
		{"en", domain.OutcomeDNF, "DNF"},
	}
	for _, c := range cas {
		if got := resolveOutcomeLabel(ctxLocale(c.locale), outcomes, c.code); got != c.want {
			t.Errorf("resolveOutcomeLabel(%s, %d) = %q, attendu %q", c.locale, c.code, got, c.want)
		}
	}
}

// Sans locale explicite, le contexte rend « fr » (ctxkeys) : le comportement d'avant le
// chantier tient à l'octet — c'est la garantie de non-régression de l'UI française.
func TestResolveOutcomeLabel_SansLocaleResteFrancais(t *testing.T) {
	got := resolveOutcomeLabel(context.Background(), outcomesDeTest(), domain.OutcomeWin)
	if got != "Victoire" {
		t.Errorf("locale absente : %q, attendu %q", got, "Victoire")
	}
}

// Le repli : titre sans jeu d'outcomes. Jamais de panic, jamais de chaîne vide — le mot FR
// d'avant. C'est le chemin instrumenté par le kill-switch daté d'outcome_label.go.
func TestResolveOutcomeLabel_RepliQuandLeTitreSeTait(t *testing.T) {
	for code, want := range map[int]string{
		domain.OutcomeWin:  "Victoire",
		domain.OutcomeLoss: "Défaite",
		domain.OutcomeDraw: "Égalité",
		domain.OutcomeDNF:  "Abandon",
	} {
		// Même sous UI anglaise : sans mapping il n'y a rien à traduire, et un panneau vide
		// serait pire qu'un mot français.
		if got := resolveOutcomeLabel(ctxLocale("en"), nil, code); got != want {
			t.Errorf("repli (jeu nil), code %d : %q, attendu %q", code, got, want)
		}
	}
}

// Titre qui expose un jeu d'outcomes SANS raw_code : le pont int→canonique n'existe pas, la
// résolution échoue proprement et le repli sert. Cas réel d'un titre à ajouter.
func TestResolveOutcomeLabel_RepliQuandLeCodeBrutNestPasMappe(t *testing.T) {
	sansCode := mappings.NewOutcomeMappingSet("titre_sans_raw_code", 1, map[string]mappings.OutcomeMapping{
		"win": {Key: "win", Labels: map[string]string{"en": "Victory", "fr": "Victoire"}},
	})
	if got := resolveOutcomeLabel(ctxLocale("en"), sansCode, domain.OutcomeWin); got != "Victoire" {
		t.Errorf("repli (raw_code absent) : %q, attendu %q", got, "Victoire")
	}
}

// Un code que NI le titre NI le repli ne connaissent (0 = pas d'issue enregistrée) : le tiret
// d'avant. Ce n'est pas un repli — rien n'a été perdu, et le log du kill-switch ne se déclenche
// pas sur ce chemin (sinon le critère « 0 repli sur 30 j » ne serait jamais atteignable).
func TestResolveOutcomeLabel_CodeInconnuRendLeTiret(t *testing.T) {
	if got := resolveOutcomeLabel(ctxLocale("fr"), outcomesDeTest(), 0); got != outcomeLabelUnknown {
		t.Errorf("code 0 : %q, attendu %q", got, outcomeLabelUnknown)
	}
	if got := resolveOutcomeLabel(ctxLocale("fr"), nil, 99); got != outcomeLabelUnknown {
		t.Errorf("code 99 sans jeu : %q, attendu %q", got, outcomeLabelUnknown)
	}
}

func TestOutcomesOf_AdapterNil(t *testing.T) {
	if got := outcomesOf(nil); got != nil {
		t.Errorf("outcomesOf(nil) = %v, attendu nil", got)
	}
}

// semantiqueDeTest — un adapter sémantique réduit à ce que le libellé d'issue lui demande.
type semantiqueDeTest struct{ outcomes *mappings.OutcomeMappingSet }

func (s semantiqueDeTest) TitleSlug() string                     { return "titre_de_test" }
func (s semantiqueDeTest) SchemaVersion() int                    { return 1 }
func (s semantiqueDeTest) Fields() *mappings.FieldMappingSet     { return nil }
func (s semantiqueDeTest) Ranks() *mappings.RankCatalog          { return nil }
func (s semantiqueDeTest) Assets() *mappings.AssetMappingSet     { return nil }
func (s semantiqueDeTest) Outcomes() *mappings.OutcomeMappingSet { return s.outcomes }

// LES LIGNES DE L'HISTORIQUE / DE L'EXPLORER (et l'export CSV, qui recopie le champ) : même
// chokepoint que l'en-tête. Le résolveur y était figé sur « fr », locale de la requête ignorée.
func TestRowFormatters_LibelleDIssueSuitLaLocale(t *testing.T) {
	svc := NewMatchHistoryService(nil, "").WithSemantic(semantiqueDeTest{outcomes: outcomesDeTest()})
	for locale, want := range map[string]string{"fr": "Victoire", "en": "Victory"} {
		f := svc.rowFormatters(ctxLocale(locale), nil)
		if got := f.outcomeLabelFor(domain.OutcomeWin); got != want {
			t.Errorf("ligne d'historique (%s) : %q, attendu %q", locale, got, want)
		}
	}
}

// Service sans adapter sémantique : le repli FR, comme avant — jamais de champ vide.
func TestRowFormatters_SansAdapterSemantique(t *testing.T) {
	f := NewMatchHistoryService(nil, "").rowFormatters(ctxLocale("en"), nil)
	if got := f.outcomeLabelFor(domain.OutcomeLoss); got != "Défaite" {
		t.Errorf("ligne sans adapter : %q, attendu %q", got, "Défaite")
	}
}

// L'EN-TÊTE DE LA MATCH VIEW, bout en bout : c'est le champ que lit la carte d'en-tête ET
// l'écran de fin du rejeu 2D. Sous UI anglaise il disait « Victoire » ; il dit « Victory ».
func TestApplyMatchHeaderOutcomeLabel_EnTeteLocalise(t *testing.T) {
	outcomes := outcomesDeTest()
	for locale, want := range map[string]string{"fr": "Défaite", "en": "Defeat"} {
		h := domain.MatchViewHeader{}
		applyMatchHeaderOutcome(&h, &domain.PlayerMatchStatsRaw{OutcomeCode: domain.OutcomeLoss})
		applyMatchHeaderOutcomeLabel(ctxLocale(locale), &h, outcomes)
		if h.OutcomeLabel != want {
			t.Errorf("en-tête (%s) : %q, attendu %q", locale, h.OutcomeLabel, want)
		}
	}
}

// Titre non câblé : l'en-tête garde EXACTEMENT le libellé posé par le builder — la passe de
// localisation ne peut jamais vider le champ ni le remplacer par une clé brute.
func TestApplyMatchHeaderOutcomeLabel_SansJeuDOutcomes(t *testing.T) {
	h := domain.MatchViewHeader{}
	applyMatchHeaderOutcome(&h, &domain.PlayerMatchStatsRaw{OutcomeCode: domain.OutcomeWin})
	applyMatchHeaderOutcomeLabel(ctxLocale("en"), &h, nil)
	if h.OutcomeLabel != "Victoire" {
		t.Errorf("en-tête sans jeu d'outcomes : %q, attendu %q", h.OutcomeLabel, "Victoire")
	}
}

// Match sans issue enregistrée : le builder n'a rien posé (OutcomeCode nil) et la passe ne
// touche à rien — le tiret initial de l'en-tête survit.
func TestApplyMatchHeaderOutcomeLabel_SansCodeDIssue(t *testing.T) {
	h := domain.MatchViewHeader{OutcomeLabel: outcomeLabelUnknown}
	applyMatchHeaderOutcomeLabel(ctxLocale("en"), &h, outcomesDeTest())
	if h.OutcomeLabel != outcomeLabelUnknown {
		t.Errorf("en-tête sans code : %q, attendu %q", h.OutcomeLabel, outcomeLabelUnknown)
	}
}
