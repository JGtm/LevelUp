// Test de référence de la vue Escouade : compare BuildSquad aux compositions du
// bloc "squads" de testdata/maquette_attendu.json (mêmes données et même instant
// que TestReferenceMaquette). Le xuid d'un membre y est son gamertag.
package trends

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

const refPrincipal = "JGtm"

// refSquad est une composition de référence.
type refSquad struct {
	Members []string    `json:"members"`
	Mask    int         `json:"mask"`
	Strict  bool        `json:"strict"`
	Matches int         `json:"matches"`
	Matrix  []refRow    `json:"matrix"`
	Series  []refSeries `json:"series"`
}

// refSquadMates : colonne de la maquette, gamertag et bit de présence de chaque coéquipier.
var refSquadMates = []struct {
	col, gamertag string
	bit           int
}{
	{"m", "Madina97294", 1},
	{"c", "Chocoboflor", 2},
	{"x", "XxDaemonGamerxX", 4},
}

// refSquadSample construit l'échantillon d'escouade d'une ligne ; nil sans frags d'équipe.
func refSquadSample(g func(string) string) *SquadSample {
	if g("tk") == "" {
		return nil
	}
	s := &SquadSample{TeamKills: tsvInt(g("tk")), Members: map[string]MemberStat{
		refPrincipal: {Kills: tsvInt(g("k")), Deaths: tsvInt(g("d")), Assists: tsvInt(g("a"))},
	}}
	for _, mate := range refSquadMates {
		if g(mate.col+"_k") != "" {
			s.Members[mate.gamertag] = MemberStat{
				Kills: tsvInt(g(mate.col + "_k")), Deaths: tsvInt(g(mate.col + "_d")), Assists: tsvInt(g(mate.col + "_a")),
			}
		}
	}
	return s
}

// loadRefSquadMatches relit le TSV : chaque match avec son échantillon d'escouade et son masque.
func loadRefSquadMatches(t *testing.T) ([]Match, []int) {
	t.Helper()
	raw, err := os.ReadFile("testdata/maquette_matchs.tsv")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\r\n"), "\n")
	col := map[string]int{}
	for i, name := range strings.Split(strings.TrimRight(lines[0], "\r"), "\t") {
		col[name] = i
	}
	var matches []Match
	var masks []int
	for _, ln := range lines[1:] {
		f := strings.Split(strings.TrimRight(ln, "\r"), "\t")
		for len(f) < len(col) {
			f = append(f, "")
		}
		g := func(name string) string { return f[col[name]] }
		m := refMatch(g)
		m.Squad = refSquadSample(g)
		matches = append(matches, m)
		masks = append(masks, tsvInt(g("mask")))
	}
	return matches, masks
}

func loadRefSquads(t *testing.T) []refSquad {
	t.Helper()
	raw, err := os.ReadFile("testdata/maquette_attendu.json")
	if err != nil {
		t.Fatal(err)
	}
	var e struct {
		Squads []refSquad `json:"squads"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	return e.Squads
}

// TestReferenceMaquetteEscouade rejoue chaque composition de la maquette.
func TestReferenceMaquetteEscouade(t *testing.T) {
	matches, masks := loadRefSquadMatches(t)
	squads := loadRefSquads(t)
	if len(squads) == 0 {
		t.Fatal("aucune composition de référence")
	}
	now := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	var c refCounts
	for _, sq := range squads {
		label := "escouade " + strings.Join(sq.Members, "+")
		if sq.Strict {
			label += " stricte"
		}
		var compo, alone []Match
		for i, m := range matches {
			if masks[i] == 0 {
				alone = append(alone, m)
			}
			if (sq.Strict && masks[i] == sq.Mask) || (!sq.Strict && masks[i]&sq.Mask == sq.Mask) {
				compo = append(compo, m)
			}
		}
		if len(compo) != sq.Matches {
			t.Fatalf("%s : %d matchs retenus par le test, attendu %d", label, len(compo), sq.Matches)
		}
		members := make([]SquadMember, 0, len(sq.Members))
		for _, gt := range sq.Members {
			members = append(members, SquadMember{XUID: gt, Gamertag: gt})
		}
		resp := BuildSquad(compo, SquadOptions{
			Options: Options{Now: now, Loc: time.UTC, HpToKill: refHpToKill}, Members: members, Alone: alone,
		})
		if len(resp.Indicators) != len(sq.Matrix) {
			t.Errorf("%s : %d indicateurs Go, attendu %d", label, len(resp.Indicators), len(sq.Matrix))
		}
		for i, w := range sq.Matrix {
			if i < len(resp.Indicators) && (resp.Indicators[i].Key != w.Key || resp.Indicators[i].Variant != w.Variant) {
				t.Errorf("%s : ligne %d = %s/%s, attendu %s/%s", label, i,
					resp.Indicators[i].Key, resp.Indicators[i].Variant, w.Key, w.Variant)
			}
		}
		compareMatrix(t, label, sq.Matrix, resp, &c)
		compareSeries(t, sq.Series, resp, &c)
	}
	t.Logf("%d compositions ; comparaisons : %+v", len(squads), c)
}
