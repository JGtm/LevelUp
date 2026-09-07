// Tests — LA CLÉ DE L'ISSUE, dite par le titre (décision D5, 2026-09-07).
//
// CE QU'ILS PROTÈGENT. Le libellé sortait d'une map Go écrite en français (ou d'un couple
// FR/EN localisé serveur, option transitoire depuis abandonnée) : sous UI anglaise l'en-tête
// de la Match View annonçait « Victoire », pendant que l'écran de fin du rejeu, vu depuis un
// adversaire, prenait son titre dans `outcomes.toml` et disait « Loss ». Deux vocabulaires
// sur un seul panneau. La règle désormais : le Go sert une CLÉ canonique (win|loss|tie|dnf),
// jamais un texte — le web localise (useOutcomeLabel/useOutcomeMapping). Seule exception :
// l'export CSV, fichier rendu serveur sans JS pour localiser, qui a besoin d'un texte —
// résolu depuis l'adapter sémantique du titre, jamais une map Go (outcomeText).
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

// LA CLÉ : le chokepoint unique servi à tout DTO d'issue.
func TestOutcomeKey_TitreCable(t *testing.T) {
	outcomes := outcomesDeTest()
	cas := map[int]string{
		domain.OutcomeWin:  "win",
		domain.OutcomeLoss: "loss",
		domain.OutcomeDraw: "tie",
		domain.OutcomeDNF:  "dnf",
	}
	for code, want := range cas {
		if got := outcomeKey(outcomes, code); got != want {
			t.Errorf("outcomeKey(%d) = %q, attendu %q", code, got, want)
		}
	}
}

// Titre non câblé (adapter nil) : la clé est vide, ce n'est pas un repli — il n'y a rien à
// traduire, jamais de panic.
func TestOutcomeKey_AdapterNil(t *testing.T) {
	if got := outcomeKey(nil, domain.OutcomeWin); got != "" {
		t.Errorf("outcomeKey(nil, WIN) = %q, attendu vide", got)
	}
}

// Code brut non mappé par le titre (0, ou valeur aberrante) : vide également.
func TestOutcomeKey_CodeInconnu(t *testing.T) {
	if got := outcomeKey(outcomesDeTest(), 0); got != "" {
		t.Errorf("outcomeKey(0) = %q, attendu vide", got)
	}
	if got := outcomeKey(outcomesDeTest(), 99); got != "" {
		t.Errorf("outcomeKey(99) = %q, attendu vide", got)
	}
}

// outcomeKeyFromHaloCode : le repli Halo-only pour les DTO sans adapter câblé
// (CareerService, ExplorerService).
func TestOutcomeKeyFromHaloCode(t *testing.T) {
	cas := map[int]string{
		domain.OutcomeWin:  "win",
		domain.OutcomeLoss: "loss",
		domain.OutcomeDraw: "tie",
		domain.OutcomeDNF:  "dnf",
		0:                  "",
		99:                 "",
	}
	for code, want := range cas {
		if got := outcomeKeyFromHaloCode(code); got != want {
			t.Errorf("outcomeKeyFromHaloCode(%d) = %q, attendu %q", code, got, want)
		}
	}
}

// LE TEXTE : réservé à l'export CSV. Dépend de la locale, contrairement à la clé.
func TestOutcomeText_LocaleDeLaRequete(t *testing.T) {
	outcomes := outcomesDeTest()
	cas := []struct {
		locale string
		code   int
		want   string
	}{
		{"fr", domain.OutcomeWin, "Victoire"},
		{"fr", domain.OutcomeLoss, "Défaite"},
		{"en", domain.OutcomeWin, "Victory"},
		{"en", domain.OutcomeLoss, "Defeat"},
	}
	for _, c := range cas {
		if got := outcomeText(outcomes, c.locale, c.code); got != c.want {
			t.Errorf("outcomeText(%s, %d) = %q, attendu %q", c.locale, c.code, got, c.want)
		}
	}
}

// Sans adapter câblé, outcomeText dégrade sur "" (jamais un mot français fabriqué) — la
// dégradation gracieuse du titre remplace l'ancien repli FR.
func TestOutcomeText_AdapterNilRendVide(t *testing.T) {
	if got := outcomeText(nil, "en", domain.OutcomeWin); got != "" {
		t.Errorf("outcomeText(nil, en, WIN) = %q, attendu vide", got)
	}
}

func TestOutcomeTextByKey(t *testing.T) {
	outcomes := outcomesDeTest()
	if got := outcomeTextByKey(outcomes, "fr", "win"); got != "Victoire" {
		t.Errorf("outcomeTextByKey(fr, win) = %q, attendu Victoire", got)
	}
	if got := outcomeTextByKey(outcomes, "en", "win"); got != "Victory" {
		t.Errorf("outcomeTextByKey(en, win) = %q, attendu Victory", got)
	}
	if got := outcomeTextByKey(outcomes, "en", ""); got != "" {
		t.Errorf("outcomeTextByKey(en, clé vide) = %q, attendu vide", got)
	}
	if got := outcomeTextByKey(outcomes, "en", "clé_inconnue"); got != "clé_inconnue" {
		t.Errorf("outcomeTextByKey(en, clé inconnue) = %q, attendu la clé telle quelle", got)
	}
}

func TestOutcomesOf_AdapterNil(t *testing.T) {
	if got := outcomesOf(nil); got != nil {
		t.Errorf("outcomesOf(nil) = %v, attendu nil", got)
	}
}

// semantiqueDeTest — un adapter sémantique réduit à ce que la clé d'issue lui demande.
type semantiqueDeTest struct{ outcomes *mappings.OutcomeMappingSet }

func (s semantiqueDeTest) TitleSlug() string                     { return "titre_de_test" }
func (s semantiqueDeTest) SchemaVersion() int                    { return 1 }
func (s semantiqueDeTest) Fields() *mappings.FieldMappingSet     { return nil }
func (s semantiqueDeTest) Ranks() *mappings.RankCatalog          { return nil }
func (s semantiqueDeTest) Assets() *mappings.AssetMappingSet     { return nil }
func (s semantiqueDeTest) Outcomes() *mappings.OutcomeMappingSet { return s.outcomes }

// LES LIGNES DE L'HISTORIQUE / DE L'EXPLORER : même chokepoint que l'en-tête — la clé, pas un
// texte, et elle ne dépend PAS de la locale (contrairement à l'ancien résolveur figé sur fr).
func TestRowFormatters_CleDIssue(t *testing.T) {
	svc := NewMatchHistoryService(nil, "").WithSemantic(semantiqueDeTest{outcomes: outcomesDeTest()})
	f := svc.rowFormatters(nil)
	if got := f.outcomeKeyFor(domain.OutcomeWin); got != "win" {
		t.Errorf("ligne d'historique : %q, attendu %q", got, "win")
	}
	if got := f.outcomeKeyFor(domain.OutcomeLoss); got != "loss" {
		t.Errorf("ligne d'historique : %q, attendu %q", got, "loss")
	}
}

// Service sans adapter sémantique : repli Halo-only (outcomeKeyFromHaloCode), jamais de champ
// vide alors qu'on connaît le code brut Halo.
func TestRowFormatters_SansAdapterSemantique(t *testing.T) {
	f := NewMatchHistoryService(nil, "").rowFormatters(nil)
	if got := f.outcomeKeyFor(domain.OutcomeLoss); got != "loss" {
		t.Errorf("ligne sans adapter : %q, attendu %q", got, "loss")
	}
}

// L'export CSV : MatchHistoryService.OutcomeText, seule surface qui rend du texte, localisé
// à la locale de ctx.
func TestMatchHistoryService_OutcomeText(t *testing.T) {
	svc := NewMatchHistoryService(nil, "").WithSemantic(semantiqueDeTest{outcomes: outcomesDeTest()})
	if got := svc.OutcomeText(ctxLocale("fr"), domain.OutcomeWin); got != "Victoire" {
		t.Errorf("OutcomeText(fr, WIN) = %q, attendu Victoire", got)
	}
	if got := svc.OutcomeText(ctxLocale("en"), domain.OutcomeWin); got != "Victory" {
		t.Errorf("OutcomeText(en, WIN) = %q, attendu Victory", got)
	}
}

// Sans adapter câblé, OutcomeText rend "" — dégradation propre, jamais de mot français en dur.
func TestMatchHistoryService_OutcomeText_SansAdapter(t *testing.T) {
	svc := NewMatchHistoryService(nil, "")
	if got := svc.OutcomeText(ctxLocale("en"), domain.OutcomeWin); got != "" {
		t.Errorf("OutcomeText sans adapter = %q, attendu vide", got)
	}
}

// L'EN-TÊTE DE LA MATCH VIEW, bout en bout : c'est le champ que lit la carte d'en-tête ET
// l'écran de fin du rejeu 2D. Il porte désormais la clé, pas un mot d'une langue donnée.
func TestApplyMatchHeaderOutcomeKey_EnTeteLocalise(t *testing.T) {
	outcomes := outcomesDeTest()
	h := domain.MatchViewHeader{}
	applyMatchHeaderOutcome(&h, &domain.PlayerMatchStatsRaw{OutcomeCode: domain.OutcomeLoss})
	applyMatchHeaderOutcomeKey(&h, outcomes)
	if h.Outcome != "loss" {
		t.Errorf("en-tête : %q, attendu %q", h.Outcome, "loss")
	}
}

// Titre non câblé : la clé reste vide — jamais un mot fabriqué, jamais un panic.
func TestApplyMatchHeaderOutcomeKey_SansJeuDOutcomes(t *testing.T) {
	h := domain.MatchViewHeader{}
	applyMatchHeaderOutcome(&h, &domain.PlayerMatchStatsRaw{OutcomeCode: domain.OutcomeWin})
	applyMatchHeaderOutcomeKey(&h, nil)
	if h.Outcome != "" {
		t.Errorf("en-tête sans jeu d'outcomes : %q, attendu vide", h.Outcome)
	}
}

// Match sans issue enregistrée : le builder n'a rien posé (OutcomeCode nil) et la passe ne
// touche à rien.
func TestApplyMatchHeaderOutcomeKey_SansCodeDIssue(t *testing.T) {
	h := domain.MatchViewHeader{}
	applyMatchHeaderOutcomeKey(&h, outcomesDeTest())
	if h.Outcome != "" {
		t.Errorf("en-tête sans code : %q, attendu vide", h.Outcome)
	}
}
