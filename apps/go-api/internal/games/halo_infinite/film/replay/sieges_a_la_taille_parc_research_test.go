//go:build research

package replay

// sieges_a_la_taille_parc_research_test.go — INSTRUMENT : la regle « aucune equipe ne depasse sa
// taille » (`sieges_a_la_taille.go`) rejouee sur des artefacts DEJA CUITS, sans recuisson.
//
// Le roster publie porte la place, l'equipe et la presence de chaque entree, les pistes ses vies :
// l'instrument en refait une pose (une place par numero publie, l'equipe de ses occupants), y joue
// la seule regle de la taille, et compte les places affichees par equipe avant et apres. Ce n'est
// pas la cuisson : le chainage des arrivants a deja eu lieu, sous la capacite estimee.
//
// USAGE (lecture seule) :
//
//	TAILLE_ARTEFACTS=<dossier des artefacts> TAILLE_IDS='3d58eb37=4,4f77afc1=12' \
//	  go test -tags=research ./internal/games/halo_infinite/film/replay -run TestTailleSurLeParc -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

type tailleArtefact struct {
	Roster []struct {
		XUID     string             `json:"xuid"`
		Name     string             `json:"name"`
		Bot      bool               `json:"bot"`
		Team     *int               `json:"team"`
		Seat     int                `json:"seat"`
		SeatSrc  string             `json:"seatSource"`
		Presence []PresenceInterval `json:"presence"`
	} `json:"roster"`
	Tracks []struct {
		XUID   string `json:"xuid"`
		Bot    string `json:"bot"`
		Points []struct {
			T int `json:"t"`
		} `json:"points"`
	} `json:"tracks"`
	FrameCount int `json:"frameCount"`
}

func TestTailleSurLeParc(t *testing.T) {
	dir, liste := os.Getenv("TAILLE_ARTEFACTS"), os.Getenv("TAILLE_IDS")
	if dir == "" || liste == "" {
		t.Skip("TAILLE_ARTEFACTS ou TAILLE_IDS absent : instrument saute")
	}
	for _, couple := range strings.Split(liste, ",") {
		id, tailleTxt, _ := strings.Cut(couple, "=")
		taille, _ := strconv.Atoi(tailleTxt)
		raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		var a tailleArtefact
		if err := json.Unmarshal(raw, &a); err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		avant := taillePlacesParEquipe(a)
		pp := tailleRefaireLaPose(a, taille)
		ramenees, releves := pp.ramenerALaTaille()
		pp.retirerLesAffichagesVides()
		apres := map[int]map[int]bool{}
		for i, e := range pp.roster {
			if o := pp.occ.parEntree[i]; o.equipe != nil && len(o.presence) > 0 {
				if apres[*o.equipe] == nil {
					apres[*o.equipe] = map[int]bool{}
				}
				apres[*o.equipe][e.Seat] = true
			}
		}
		t.Logf("%s (taille %d) : places par equipe avant %s, apres %s ; ramenees %d, bots releves %d",
			id, taille, tailleFormat(avant), tailleFormat(apres), ramenees, releves)
		for i, e := range pp.roster {
			if e.SeatSource == SeatSourceTaille {
				t.Logf("    %s -> place %d", pp.roster[i].Name, e.Seat)
			}
		}
	}
}

// taillePlacesParEquipe rend les places affichees par equipe dans l'artefact publie.
func taillePlacesParEquipe(a tailleArtefact) map[int]map[int]bool {
	out := map[int]map[int]bool{}
	for _, e := range a.Roster {
		if e.Team != nil && len(e.Presence) > 0 {
			if out[*e.Team] == nil {
				out[*e.Team] = map[int]bool{}
			}
			out[*e.Team][e.Seat] = true
		}
	}
	return out
}

// tailleRefaireLaPose refait une pose depuis le roster publie : une place par numero publie.
func tailleRefaireLaPose(a tailleArtefact, taille int) *poseDesPlaces {
	vies := map[string][][2]int{}
	for _, tr := range a.Tracks {
		if len(tr.Points) == 0 {
			continue
		}
		cle := tr.XUID
		if cle == "" && tr.Bot != "" {
			cle = botIdentityKey(tr.Bot)
		}
		vies[cle] = append(vies[cle], [2]int{tr.Points[0].T, tr.Points[len(tr.Points)-1].T})
	}
	roster := make([]RosterEntry, len(a.Roster))
	occ := occupants{parEntree: make([]occupantDuRoster, len(a.Roster)), balaye: true}
	var table grammar.FilmPlayerTable
	vus := map[int]bool{}
	for i, e := range a.Roster {
		roster[i] = RosterEntry{XUID: e.XUID, Name: e.Name, Bot: e.Bot, FilmIndex: e.Seat}
		var ivs []intervalleDePresence
		for _, p := range e.Presence {
			m := p.To
			if p.ToMax != nil {
				m = *p.ToMax
			}
			ivs = append(ivs, intervalleDePresence{de: p.From, a: p.To, aMax: m})
		}
		occ.parEntree[i] = occupantDuRoster{presence: ivs, equipe: e.Team, lue: true,
			vies: vies[cleDeRoster(roster[i])], declaree: e.Bot}
		if !vus[e.Seat] {
			vus[e.Seat] = true
			table.Seats = append(table.Seats, grammar.FilmPlayerSeat{FilmIndex: e.Seat, XUID: uint64(i + 1)})
		}
	}
	table.Occupied, table.Vacant = len(table.Seats), 32-len(table.Seats)
	pp := nouvellePoseDesPlaces(roster, &occ, entreesDesPlaces{table: table, tailleDEquipe: taille,
		horloge: replayClock{frames: a.FrameCount}})
	for i := range roster {
		roster[i].Seat, roster[i].SeatSource = a.Roster[i].Seat, a.Roster[i].SeatSrc
		if p := pp.places[roster[i].Seat]; p != nil && len(occ.parEntree[i].presence) > 0 {
			pp.asseoir(i, p, a.Roster[i].SeatSrc)
		}
	}
	return pp
}

func tailleFormat(m map[int]map[int]bool) string {
	equipes := make([]int, 0, len(m))
	for t := range m {
		equipes = append(equipes, t)
	}
	sort.Ints(equipes)
	var b strings.Builder
	for _, t := range equipes {
		fmt.Fprintf(&b, "[%d:%d]", t, len(m[t]))
	}
	return b.String()
}
