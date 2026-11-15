package leadshare

import (
	"encoding/json"
	"os"
	"testing"
)

// jeuDeCas : le fichier PARTAGÉ avec le test web de `leaderStates` / `leadChanges`.
const jeuDeCas = "testdata/meneur_cas.json"

type casMeneur struct {
	Nom       string `json:"nom"`
	CampCount int    `json:"camp_count"`
	Teams     []struct {
		TeamID *int `json:"team_id"`
		Total  []struct {
			T int `json:"t"`
			V int `json:"v"`
		} `json:"total"`
	} `json:"teams"`
	Etats []struct {
		Frame  int  `json:"frame"`
		TeamID *int `json:"team_id"`
	} `json:"etats"`
	Retournements []struct {
		Frame  int `json:"frame"`
		TeamID int `json:"team_id"`
	} `json:"retournements"`
}

func (c casMeneur) equipes() []Team {
	out := make([]Team, 0, len(c.Teams))
	for _, t := range c.Teams {
		ticks := make([]Tick, 0, len(t.Total))
		for _, p := range t.Total {
			ticks = append(ticks, Tick{Frame: p.T, Value: p.V})
		}
		out = append(out, Team{ID: t.TeamID, Totals: ticks})
	}
	return out
}

func lireJeuDeCas(t *testing.T) []casMeneur {
	t.Helper()
	data, err := os.ReadFile(jeuDeCas)
	if err != nil {
		t.Fatalf("lecture du jeu de cas : %v", err)
	}
	var jeu struct {
		Cas []casMeneur `json:"cas"`
	}
	if err := json.Unmarshal(data, &jeu); err != nil {
		t.Fatalf("décodage du jeu de cas : %v", err)
	}
	if len(jeu.Cas) == 0 {
		t.Fatal("le jeu de cas est vide : un test sur zéro cas serait vert et inerte")
	}
	return jeu.Cas
}

func libelleMeneur(p *int) string {
	if p == nil {
		return "égalité"
	}
	return string(rune('0' + *p))
}

// Les deux côtés (Go et web) lisent le même fichier : c'est lui qui prouve qu'ils rendent les
// mêmes états et les mêmes retournements.
func TestLeaderStates_JeuDeCasPartage(t *testing.T) {
	for _, cas := range lireJeuDeCas(t) {
		t.Run(cas.Nom, func(t *testing.T) {
			etats := LeaderStates(cas.equipes(), cas.CampCount)
			if len(etats) != len(cas.Etats) {
				t.Fatalf("%d états, attendu %d : %+v", len(etats), len(cas.Etats), etats)
			}
			for i, e := range cas.Etats {
				if etats[i].Frame != e.Frame || !sameLeader(etats[i].Leader, e.TeamID) {
					t.Errorf("état %d : frame %d meneur %s, attendu frame %d meneur %s", i,
						etats[i].Frame, libelleMeneur(etats[i].Leader), e.Frame, libelleMeneur(e.TeamID))
				}
			}
			retournements := Changes(etats)
			if len(retournements) != len(cas.Retournements) {
				t.Fatalf("%d retournements, attendu %d : %+v", len(retournements), len(cas.Retournements), retournements)
			}
			for i, r := range cas.Retournements {
				if retournements[i].Frame != r.Frame || retournements[i].Leader != r.TeamID {
					t.Errorf("retournement %d : %+v, attendu %+v", i, retournements[i], r)
				}
			}
		})
	}
}

func TestScoreAt_ZeroAvantLePremierPalier(t *testing.T) {
	totals := []Tick{{Frame: 10, Value: 1}, {Frame: 20, Value: 3}}
	for frame, want := range map[int]int{0: 0, 9: 0, 10: 1, 19: 1, 20: 3, 500: 3} {
		if got := ScoreAt(totals, frame); got != want {
			t.Errorf("frame %d : %d, attendu %d", frame, got, want)
		}
	}
	if got := ScoreAt(nil, 5); got != 0 {
		t.Errorf("série vide : %d, attendu 0", got)
	}
}
