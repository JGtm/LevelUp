package coordination

// bloc_appui_golden_test.go — LE VERSANT APPUI DU BLOC, FIGÉ OCTET POUR OCTET.
//
// Écrit AVANT le retrait de la riposte du bloc (plan PLAN_SESSIONS_EMPRISE_2026-10-06, S5.4) : la
// coupe ne doit rien changer à ce que lisent la carte « Appui reçu » de Sessions et la frise des
// Séries temporelles — couverture, deux couvertures d'appui, parité pondérée, et le versant appui de
// chaque case. Le golden a été produit par le bloc d'AVANT la coupe ; toute dérive le fait rougir.
//
// Régénération (seulement pour un changement VOULU du versant appui) :
// COORDINATION_APPUI_GOLDEN_UPDATE=1 go test ./internal/analysis/coordination/ -run TestBloc_AppuiGolden

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// appuiCase — le versant appui d'une case de la bande de régularité.
type appuiCase struct {
	MatchID              string   `json:"match_id"`
	TeamSize             *int     `json:"team_size,omitempty"`
	ParityPct            *float64 `json:"parity_pct,omitempty"`
	MyKills              int      `json:"my_kills"`
	MyAssistedKills      int      `json:"my_assisted_kills"`
	TeamAssists          int      `json:"team_assists"`
	AssistsToMe          int      `json:"assists_to_me"`
	AssistShareOfTeamPct *float64 `json:"assist_share_of_team_pct,omitempty"`
	AssistedSharePct     *float64 `json:"assisted_share_pct,omitempty"`
}

// appuiProjection — ce que lisent les surfaces d'appui, et rien d'autre.
type appuiProjection struct {
	Available         bool                     `json:"available"`
	UnavailableReason string                   `json:"unavailable_reason,omitempty"`
	MatchesMeasured   int                      `json:"matches_measured"`
	MatchesTotal      int                      `json:"matches_total"`
	Appui             domain.CoordinationAppui `json:"appui"`
	PerMatch          []appuiCase              `json:"per_match"`
}

func projeterAppui(b domain.CoordinationBlock) appuiProjection {
	out := appuiProjection{
		Available:         b.Available,
		UnavailableReason: b.UnavailableReason,
		MatchesMeasured:   b.MatchesMeasured,
		MatchesTotal:      b.MatchesTotal,
		Appui:             b.Appui,
		PerMatch:          []appuiCase{},
	}
	for _, p := range b.PerMatch {
		out.PerMatch = append(out.PerMatch, appuiCase{
			MatchID: p.MatchID, TeamSize: p.TeamSize, ParityPct: p.ParityPct,
			MyKills: p.MyKills, MyAssistedKills: p.MyAssistedKills,
			TeamAssists: p.TeamAssists, AssistsToMe: p.AssistsToMe,
			AssistShareOfTeamPct: p.AssistShareOfTeamPct, AssistedSharePct: p.AssistedSharePct,
		})
	}
	return out
}

// scenarioAppuiLarge — le scénario de bloc_test.go, plus un match 8 contre 8 aux appuis mêlés,
// un match FFA (sans effectif) et un match non mesuré : parité pondérée, case sans parité, et un
// match qui n'entre nulle part.
func scenarioAppuiLarge() domain.CoordinationEntree {
	in := scenario()
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "m2", Mesure: true, TeamSize: entier(8)},
		domain.CoordinationMatch{MatchID: "m3", Mesure: true},
		domain.CoordinationMatch{MatchID: "m4", Mesure: false, TeamSize: entier(4)},
	)
	in.Equipes["m2"] = map[string]int{"P": 0, "A": 0, "B": 0, "E1": 1, "E2": 1}
	in.Equipes["m3"] = map[string]int{"P": 0, "E1": 1}
	in.Equipes["m4"] = map[string]int{"P": 0, "A": 0}
	in.Appuis = append(in.Appuis,
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "B", KillerXUID: "P", Nombre: 2},
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "A", KillerXUID: "B", Nombre: 3},
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "", KillerXUID: "P", Nombre: 4},
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "E2", KillerXUID: "E1", Nombre: 1},
		domain.CoordinationAppuiRow{MatchID: "m3", AssistXUID: "", KillerXUID: "P", Nombre: 5},
		domain.CoordinationAppuiRow{MatchID: "m4", AssistXUID: "A", KillerXUID: "P", Nombre: 9},
	)
	return in
}

func TestBloc_AppuiGolden(t *testing.T) {
	cas := map[string]domain.CoordinationEntree{
		"scenario":       scenario(),
		"scenario_large": scenarioAppuiLarge(),
		"sans_sujet":     {Matchs: scenario().Matchs},
	}
	got := map[string]appuiProjection{}
	for nom, in := range cas {
		got[nom] = projeterAppui(Bloc(in))
	}
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join("testdata", "bloc_appui.golden.json")
	if os.Getenv("COORDINATION_APPUI_GOLDEN_UPDATE") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden absent (%v) : le produire AVANT toute modification du bloc", err)
	}
	if !bytes.Equal(want, data) {
		t.Fatalf("versant appui du bloc modifié.\nattendu :\n%s\nobtenu :\n%s", want, data)
	}
}
