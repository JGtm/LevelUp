package replayview_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/service/replayview"
)

func ptr(n int) *int { return &n }

// TestProjection_ServeLesLignesDUnBotParPlace : une élimination et une action de score dont l'auteur est un bot (sans
// identifiant, désigné par sa place dans le film) sont servies avec leur place et SANS `xuid`, au lieu d'être écartées ;
// la ligne d'un humain garde son `xuid` et n'a pas de place.
func TestProjection_ServeLesLignesDUnBotParPlace(t *testing.T) {
	doc := replay.ReplayDocument{
		Eliminations: []replay.Elimination{
			{T: 10, XUID: "2533274800000001", Victim: "2533274800000002"},
			{T: 20, Place: ptr(5), Victim: "2533274800000001"},
			{T: 30, XUID: "2533274800000001", VictimPlace: ptr(5)},
		},
		ScoreActions: []replay.ScoreAction{
			{T: 10, XUID: "2533274800000001", Name: "killPersonalScore", Points: 100},
			{T: 20, Place: ptr(5), Name: "killPersonalScore", Points: 100, Target: "2533274800000001"},
		},
	}
	out := replayview.FromArtifact(doc)
	if len(out.Eliminations) != 3 || len(out.ScoreActions) != 2 {
		t.Fatalf("lignes servies : %d éliminations, %d actions, attendu 3 et 2 (aucune ligne de bot écartée)", len(out.Eliminations), len(out.ScoreActions))
	}
	if e := out.Eliminations[1]; e.XUID != "" || e.Place == nil || *e.Place != 5 || e.Victim != "2533274800000001" {
		t.Errorf("élimination d'un bot = %+v, attendu place 5, sans xuid, victime humaine", e)
	}
	if e := out.Eliminations[2]; e.VictimPlace == nil || *e.VictimPlace != 5 || e.Victim != "" {
		t.Errorf("élimination d'un bot par un humain = %+v, attendu victimPlace 5, sans victime humaine", e)
	}
	if a := out.ScoreActions[1]; a.XUID != "" || a.Place == nil || *a.Place != 5 {
		t.Errorf("action d'un bot = %+v, attendu place 5, sans xuid", a)
	}
	raw, err := json.Marshal(out.Eliminations[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"place"`) {
		t.Errorf("une ligne d'humain ne porte pas de place : %s", raw)
	}
	rawBot, err := json.Marshal(out.Eliminations[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawBot), `"xuid"`) {
		t.Errorf("une ligne de bot ne porte pas de xuid vide : %s", rawBot)
	}
}
