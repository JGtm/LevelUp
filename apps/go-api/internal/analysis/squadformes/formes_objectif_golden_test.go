package squadformes

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// formes_objectif_golden_test.go — LA NON-RÉGRESSION DE L'OBJECTIF PUBLIÉ (plan
// PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, lot L7, décision D7).
//
// Le bloc `formes_retenues` ne garde que ce que lisent les cartes d'objectif (Escouade ›
// Contributions, Séries temporelles › Usages) : disponibilité, comptes, joueur et escouade, et
// pour chaque match À OBJECTIF son identité, son camp et sa feuille d'objectif. Cette projection
// est figée octet pour octet dans `testdata/objectif_publie.golden.json` : réduire le contrat ne
// doit rien y changer. Régénération (changement VOULU de l'objectif seulement) :
// UPDATE_GOLDEN=1 go test ./internal/analysis/squadformes/ -run TestBuild_ObjectifPublieInchange

// vueObjectif — ce que lisent les cartes d'objectif, sous les noms du contrat.
type vueObjectif struct {
	Available         bool                             `json:"available"`
	UnavailableReason string                           `json:"unavailable_reason,omitempty"`
	MatchesTotal      int                              `json:"matches_total"`
	MatchesMeasured   int                              `json:"matches_measured"`
	MainXUID          string                           `json:"main_xuid,omitempty"`
	Squad             []domain.SessionUsageSquadPlayer `json:"squad,omitempty"`
	Matches           []vueMatchObjectif               `json:"matches"`
}

type vueMatchObjectif struct {
	MatchID    string                       `json:"match_id"`
	StartTime  string                       `json:"start_time,omitempty"`
	ModeLabel  string                       `json:"mode_label,omitempty"`
	MapLabel   string                       `json:"map_label,omitempty"`
	PlayerTeam *int                         `json:"player_team,omitempty"`
	Objective  *domain.SquadFormesObjective `json:"objective"`
}

func projectionObjectif(b domain.SquadFormesBlock) vueObjectif {
	v := vueObjectif{
		Available: b.Available, UnavailableReason: b.UnavailableReason,
		MatchesTotal: b.MatchesTotal, MatchesMeasured: b.MatchesMeasured,
		MainXUID: b.MainXUID, Squad: b.Squad, Matches: []vueMatchObjectif{},
	}
	for _, m := range b.Matches {
		if m.Objective == nil {
			continue
		}
		v.Matches = append(v.Matches, vueMatchObjectif{
			MatchID: m.MatchID, StartTime: m.StartTime, ModeLabel: m.ModeLabel, MapLabel: m.MapLabel,
			PlayerTeam: m.PlayerTeam, Objective: m.Objective,
		})
	}
	return v
}

// fixtureObjectif — la fixture du paquet, plus trois feuilles d'objectif : m1 (sans film), m2
// (mesuré, avec la grandeur optionnelle des prises nettes et sa fenêtre de jonglage) et un match à
// objectif seul, au mode écarté des parts de rôle.
func fixtureObjectif() Input {
	in := fixture()
	in.Metas = append(in.Metas, MatchMeta{MatchID: "obj-seul", StartTime: "2026-07-31T18:10:00Z",
		ModeLabel: "Drapeau neutre", MapLabel: "Aquarius", ObjectiveExcluded: true})
	in.Objectives = []ObjectiveColumnRow{
		{MatchID: "m1", XUID: "moi", Family: narrative.FamilyZonesStrongholds,
			Values: map[string]float64{"zone_secures": 3, "zone_captures": 1, "time_in_zones_seconds": 42.5}},
		{MatchID: "m1", XUID: "adv", Family: narrative.FamilyZonesStrongholds,
			Values: map[string]float64{"zone_secures": 1, "zone_captures": 4, "time_in_zones_seconds": 61}},
		{MatchID: "m2", XUID: "moi", Family: narrative.FamilyCTF, FlagJuggleWindowSeconds: 1.5,
			Values: map[string]float64{"flag_returns": 2, "flag_captures": 1, "flag_grabs_net": 3, "time_as_flag_carrier_seconds": 20}},
		{MatchID: "m2", XUID: "cop", Family: narrative.FamilyCTF, FlagJuggleWindowSeconds: 1.5,
			Values: map[string]float64{"flag_returns": 0, "flag_captures": 0, "time_as_flag_carrier_seconds": 7.25}},
		{MatchID: "m2", XUID: "adv", Family: narrative.FamilyCTF, FlagJuggleWindowSeconds: 1.5,
			Values: map[string]float64{"flag_returns": 1, "flag_captures": 2, "flag_grabs_net": 5, "time_as_flag_carrier_seconds": 33}},
		{MatchID: "obj-seul", XUID: "moi", Family: narrative.FamilyCTF,
			Values: map[string]float64{"flag_returns": 4}},
	}
	return in
}

func TestBuild_ObjectifPublieInchange(t *testing.T) {
	got, err := json.MarshalIndent(projectionObjectif(Build(fixtureObjectif())), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join("testdata", "objectif_publie.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden illisible (UPDATE_GOLDEN=1 pour le créer) : %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("l'objectif publié a changé :\n--- attendu\n%s\n--- obtenu\n%s", want, got)
	}
}
