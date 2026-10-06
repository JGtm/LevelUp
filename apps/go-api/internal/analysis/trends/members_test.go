package trends

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestBuildSquad_MembresDansLOrdre(t *testing.T) {
	opts := SquadOptions{
		Options: Options{Now: testNow, Loc: time.UTC},
		Members: []SquadMember{{XUID: "px", Gamertag: "Moi"}, {XUID: "xa", Gamertag: "Allie"}, {XUID: "xb", Gamertag: "Bob"}},
	}
	resp := BuildSquad(nil, opts)
	want := []domain.TrendsMember{{XUID: "px", Gamertag: "Moi"}, {XUID: "xa", Gamertag: "Allie"}, {XUID: "xb", Gamertag: "Bob"}}
	if len(resp.Members) != len(want) {
		t.Fatalf("membres = %+v", resp.Members)
	}
	for i, w := range want {
		if resp.Members[i] != w {
			t.Errorf("membre %d = %+v, attendu %+v", i, resp.Members[i], w)
		}
	}
}

func TestBuildSquad_SansMembreTableauVide(t *testing.T) {
	resp := BuildSquad(nil, SquadOptions{Options: Options{Now: testNow, Loc: time.UTC}})
	if resp.Members == nil || len(resp.Members) != 0 {
		t.Fatalf("membres = %#v, attendu []", resp.Members)
	}
}

func TestBuildSolo_MembresVidesEnJSON(t *testing.T) {
	resp := BuildSolo(nil, Options{Now: testNow, Loc: time.UTC})
	if resp.Members == nil || len(resp.Members) != 0 {
		t.Fatalf("membres = %#v, attendu []", resp.Members)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"members":[]`) {
		t.Errorf("JSON sans \"members\":[] : %s", raw)
	}
}
