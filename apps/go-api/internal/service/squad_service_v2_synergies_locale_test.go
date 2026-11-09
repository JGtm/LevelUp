package service

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// carteLangues fabrique un match dont la carte porte un libelle par langue (le defaut est en anglais,
// la langue de repli) : le graphique par carte doit servir le libelle de la langue DEMANDEE.
func carteLangues(matchID, mapID, en, fr string, outcome canonical.Outcome, perf float64) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			Outcome:      outcome,
			Map: &canonical.AssetReference{
				Kind: "map", ID: mapID, DefaultLabel: en,
				Labels: map[string]string{"en": en, "fr": fr},
			},
		},
		Self:       canonical.MatchParticipant{Outcome: outcome},
		Enrichment: canonical.PlayerMatchEnrichment{PerformanceScore: &perf},
	}
}

func rowsCartesLangues() []canonical.PlayerMatchRow {
	return []canonical.PlayerMatchRow{
		carteLangues("m1", "bazaar", "Bazaar", "Bazar", canonical.OutcomeWin, 70),
		carteLangues("m2", "bazaar", "Bazaar", "Bazar", canonical.OutcomeLoss, 50),
		carteLangues("m3", "pit", "The Pit", "La Fosse", canonical.OutcomeWin, 60),
	}
}

func categories(pts []domain.ChartPointStacked) []string {
	out := make([]string, 0, len(pts))
	for _, p := range pts {
		out = append(out, p.Category)
	}
	return out
}

// Le lollipop sert les noms de carte dans la langue de la requete ; la cle de regroupement, l'ordre et les
// comptes ne dependent pas de la langue.
func TestBuildMapBreakdownLollipop_NomsDansLaLangueDeLaRequete(t *testing.T) {
	t.Parallel()
	rows := rowsCartesLangues()
	fr := BuildMapBreakdownLollipop(rows, 0, domain.GroupesDeCartes{}, "fr")
	en := BuildMapBreakdownLollipop(rows, 0, domain.GroupesDeCartes{}, "en")
	if got, want := categories(fr.Datapoints), []string{"Bazar", "La Fosse"}; !reflect.DeepEqual(got, want) {
		t.Errorf("fr : %v, attendu %v", got, want)
	}
	if got, want := categories(en.Datapoints), []string{"Bazaar", "The Pit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("en : %v, attendu %v", got, want)
	}
	for i := range fr.Datapoints {
		if !reflect.DeepEqual(fr.Datapoints[i].Components, en.Datapoints[i].Components) {
			t.Errorf("rang %d : les comptes dependent de la langue : fr %v, en %v", i, fr.Datapoints[i].Components, en.Datapoints[i].Components)
		}
	}
}

// Une carte regroupee sous sa base prend le nom de la base dans la langue de la requete.
func TestBuildMapBreakdownLollipop_BaseDeGroupeDansLaLangue(t *testing.T) {
	t.Parallel()
	groups := groupesSquadTest() // base "th" : en "Thunderhead", fr "Tonnerre" ; variante "th-heavies"
	rows := []canonical.PlayerMatchRow{
		carteLangues("m1", "th", "Thunderhead", "Tonnerre", canonical.OutcomeWin, 60),
		carteLangues("m2", "th-heavies", "Thunderhead Heavies", "Tonnerre Heavies", canonical.OutcomeLoss, 60),
	}
	for loc, want := range map[i18n.Locale]string{"fr": "Tonnerre", "en": "Thunderhead"} {
		got := BuildMapBreakdownLollipop(rows, 0, groups, loc)
		if len(got.Datapoints) != 1 || got.Datapoints[0].Category != want {
			t.Errorf("%s : %+v, attendu une carte %q", loc, got.Datapoints, want)
		}
	}
}

// Les trois autres graphiques par carte de la page (bullet, ecart de perf, heatmap) suivent la meme langue.
func TestSynergiesParCarte_NomsDansLaLangueDeLaRequete(t *testing.T) {
	t.Parallel()
	rows := rowsCartesLangues()
	historical := map[string]domain.MapSquadStats{
		"bazaar": {Wins: 1, Total: 4, PerfAvg: fptrSyn(40)}, "pit": {Wins: 1, Total: 2, PerfAvg: fptrSyn(50)},
	}
	for loc, want := range map[i18n.Locale][]string{"fr": {"Bazar", "La Fosse"}, "en": {"Bazaar", "The Pit"}} {
		bullet := BuildBulletWinrate(rows, historical, 0, domain.GroupesDeCartes{}, loc)
		if got := categories(bullet.Datapoints); !reflect.DeepEqual(got, want) {
			t.Errorf("bullet %s : %v, attendu %v", loc, got, want)
		}
		perf := BuildPerfVsHistorical(rows, historical, 0, domain.GroupesDeCartes{}, loc)
		var xs []string
		for _, p := range perf.Datapoints {
			xs = append(xs, fmt.Sprint(p.X))
		}
		if len(xs) != 2 || !(xs[0] == want[0] && xs[1] == want[1] || xs[0] == want[1] && xs[1] == want[0]) {
			t.Errorf("ecart de perf %s : %v, attendu %v", loc, xs, want)
		}
		heat := BuildHeatmapPlayerMap(map[string][]canonical.PlayerMatchRow{"moi": rows}, 0, domain.GroupesDeCartes{}, loc)
		seen := map[string]bool{}
		for _, p := range heat.Datapoints {
			seen[p.X] = true
		}
		if len(seen) != 2 || !seen[want[0]] || !seen[want[1]] {
			t.Errorf("heatmap %s : %v, attendu %v", loc, seen, want)
		}
	}
}

// La langue de la requete traverse l'assemblage de la page : buildSquadCharts la recoit de GetSquadPageV2.
func TestBuildSquadCharts_LaLangueDeLaRequeteArriveAuxGraphiquesParCarte(t *testing.T) {
	t.Parallel()
	rows := map[string][]canonical.PlayerMatchRow{"moi": rowsCartesLangues()}
	for loc, want := range map[i18n.Locale]string{"fr": "Bazar", "en": "Bazaar"} {
		charts := buildSquadCharts(buildSquadChartsInput{mainGT: "moi", rowsByPlayer: rows, locale: loc})
		if charts.MapBreakdownLollipop == nil || len(charts.MapBreakdownLollipop.Datapoints) == 0 {
			t.Fatalf("%s : lollipop absent", loc)
		}
		if got := charts.MapBreakdownLollipop.Datapoints[0].Category; got != want {
			t.Errorf("%s : premiere carte %q, attendu %q", loc, got, want)
		}
	}
}
