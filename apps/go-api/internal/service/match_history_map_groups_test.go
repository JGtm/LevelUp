package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
)

// groupesDeTest : Thunderhead (base, deux identifiants) et sa variante Heavies forment un groupe.
func groupesDeTest() domain.GroupesDeCartes {
	return domain.NewGroupesDeCartes(map[string]string{
		"var-heavies": "base-a", "base-b": "base-a",
	}).AvecLibelles(map[string]i18n.Text{
		"base-a": {"fr": "Tonnerre", "en": "Thunderhead"},
	})
}

func rowCarteGroupe(id string, nom *string, outcome int) domain.MatchHistoryRawRow {
	var names i18n.Text
	if nom != nil {
		names = i18n.FromColumns(*nom, "")
	}
	return domain.MatchHistoryRawRow{MapID: &id, MapName: names, Outcome: outcome}
}

// Une variante et sa base donnent UN taux ; une ligne sans nom au registre est comptée par son
// identifiant ; une carte hors groupe reste seule.
func TestComputeMapWinRates_VarianteEtBaseUnSeulTaux(t *testing.T) {
	s := func(v string) *string { return &v }
	rows := []domain.MatchHistoryRawRow{
		rowCarteGroupe("base-a", s("Thunderhead"), domain.OutcomeWin),
		rowCarteGroupe("var-heavies", s("Thunderhead Heavies"), domain.OutcomeLoss),
		rowCarteGroupe("base-b", nil, domain.OutcomeWin), // nom NULL au registre
		rowCarteGroupe("seule", s("La Fosse"), domain.OutcomeWin),
	}
	rates := computeMapWinRates(rows, groupesDeTest())
	if got := rates["base-a"]; got != [2]int{2, 3} {
		t.Errorf("groupe Thunderhead = %v, attendu [2 3]", got)
	}
	if got := rates["seule"]; got != [2]int{1, 1} {
		t.Errorf("carte seule = %v, attendu [1 1]", got)
	}
	if _, ok := rates["var-heavies"]; ok {
		t.Error("la variante ne doit plus avoir sa propre entrée")
	}
}

// La lecture (enrichRow) et l'agrégation partagent la clé : la ligne d'une variante affiche le taux
// du groupe.
func TestEnrichRow_TauxDuGroupe(t *testing.T) {
	s := func(v string) *string { return &v }
	groups := groupesDeTest()
	rows := []domain.MatchHistoryRawRow{
		rowCarteGroupe("base-a", s("Thunderhead"), domain.OutcomeWin),
		rowCarteGroupe("var-heavies", s("Thunderhead Heavies"), domain.OutcomeLoss),
	}
	rates := computeMapWinRates(rows, groups)
	fmts := rowFormatters{mapGroups: groups}
	for _, r := range rows {
		got := enrichRow(r, rates, fmts, i18n.Default)
		if got.WinRateHist == nil || *got.WinRateHist != 50 || got.WinRateHistTotal == nil || *got.WinRateHistTotal != 2 {
			t.Errorf("ligne %s : taux = %v sur %v, attendu 50 %% sur 2", *r.MapID, got.WinRateHist, got.WinRateHistTotal)
		}
	}
}

// Briefing : la dimension « carte » compte base + variante comme UNE carte, au libellé de la base.
func TestBuildBriefingDimensions_CarteRegroupee(t *testing.T) {
	s := func(v string) *string { return &v }
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 6; i++ {
		r := rowCarteGroupe("base-a", s("Thunderhead"), domain.OutcomeWin)
		r.MapName[i18n.LocaleFR] = "Tonnerre"
		scope = append(scope, r)
	}
	for i := 0; i < 6; i++ {
		r := rowCarteGroupe("var-heavies", s("Thunderhead Heavies"), domain.OutcomeLoss)
		r.MapName[i18n.LocaleFR] = "Du lourd sur Thunderhead"
		scope = append(scope, r)
	}
	for i := 0; i < 12; i++ {
		r := rowCarteGroupe("seule", s("La Fosse"), domain.OutcomeLoss)
		r.MapName[i18n.LocaleFR] = "La Fosse"
		scope = append(scope, r)
	}
	dims := buildBriefingDimensions(scope, scope, groupesDeTest(), i18n.Default)
	for _, d := range dims {
		if d.Dimension != "map" {
			continue
		}
		groupe := 0
		for _, e := range d.Entries {
			if e.Label == "Du lourd sur Thunderhead" {
				t.Errorf("le libellé d'une variante ne doit jamais s'afficher : %+v", e)
			}
			if e.Label == "Tonnerre" {
				groupe++
				if e.Matches != 12 {
					t.Errorf("le groupe doit compter base + variante (12 parties) : %+v", e)
				}
			}
		}
		if groupe != 1 {
			t.Errorf("une seule entrée Tonnerre attendue, %d trouvée(s) : %+v", groupe, d.Entries)
		}
		return
	}
	t.Fatalf("dimension carte absente : %+v", dims)
}

func rowSynthese(mapID, en, fr string, outcome canonical.Outcome) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{Map: &canonical.AssetReference{
			Kind: "map", ID: mapID, DefaultLabel: en, Labels: map[string]string{"fr": fr},
		}},
		Self: canonical.MatchParticipant{Outcome: outcome},
	}
}

// Synthèse : « cartes les plus jouées » compte base + variante comme UNE entrée, au libellé de la base ;
// une carte hors groupe garde son libellé propre.
func TestBuildBreakdownsFromCanonical_UneEntreePourBaseEtVariante(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		rowSynthese("base-a", "Thunderhead", "Tonnerre", canonical.OutcomeWin),
		rowSynthese("var-heavies", "Thunderhead Heavies", "Du lourd sur Thunderhead", canonical.OutcomeLoss),
		rowSynthese("base-b", "Thunderhead", "Tonnerre", canonical.OutcomeWin),
		rowSynthese("seule", "The Pit", "La Fosse", canonical.OutcomeWin),
	}
	got := buildBreakdownsFromCanonical(rows, groupesDeTest(), i18n.Default)
	if len(got.TopMaps) != 2 {
		t.Fatalf("entrées de carte = %d, attendu 2 : %+v", len(got.TopMaps), got.TopMaps)
	}
	for _, e := range got.TopMaps {
		switch e.MapName {
		case "Tonnerre":
			if e.MatchCount != 3 || e.Wins != 2 || e.Losses != 1 {
				t.Errorf("groupe Tonnerre = %+v, attendu 3 parties, 2 V, 1 D", e)
			}
		case "La Fosse":
			if e.MatchCount != 1 {
				t.Errorf("carte seule = %+v", e)
			}
		default:
			t.Errorf("libellé inattendu %q (aucune mention de variante)", e.MapName)
		}
	}
	// Sans groupe : le comportement d'avant, une entrée par libellé.
	if avant := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{}, i18n.Default); len(avant.TopMaps) != 3 {
		t.Errorf("sans groupe : %d entrées, attendu 3", len(avant.TopMaps))
	}
}

func statRow(id, nom, fr string, outcome int) legacymatch.StatsMatchRow {
	return legacymatch.StatsMatchRow{MatchID: id + nom, MapID: id, MapName: i18n.FromColumns(nom, fr), Outcome: &outcome}
}

// Séries temporelles : le découpage par carte compte base + variante comme UNE carte, au libellé de la
// base, historique compris.
func TestBuildSoloMapBreakdown_CartesRegroupees(t *testing.T) {
	groups := groupesDeTest()
	win, loss := domain.OutcomeWin, domain.OutcomeLoss
	current := []legacymatch.StatsMatchRow{
		statRow("base-a", "Thunderhead", "Tonnerre", win),
		statRow("var-heavies", "Thunderhead Heavies", "Du lourd sur Thunderhead", loss),
		statRow("seule", "The Pit", "La Fosse", win),
	}
	got := buildSoloMapBreakdown(current, current, groups, i18n.Default)
	if len(got) != 2 {
		t.Fatalf("cartes = %d, attendu 2 : %+v", len(got), got)
	}
	for _, r := range got {
		if r.MapUI == "Du lourd sur Thunderhead" {
			t.Errorf("libellé de variante servi : %+v", r)
		}
		if r.MapUI == "Tonnerre" && (r.MatchCount != 2 || r.HistoricalWinRate == nil || *r.HistoricalWinRate != 0.5) {
			t.Errorf("groupe = %+v, attendu 2 parties, historique 50 %%", r)
		}
	}
}
