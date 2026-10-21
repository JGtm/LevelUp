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
		bilan := pp.ramenerALaTaille()
		pp.bornerAuSuccesseur()
		pp.retirerLesAffichagesVides()
		apres := map[int]map[int]bool{}
		for i, e := range pp.roster {
			o := pp.occ.parEntree[i]
			for _, iv := range o.presence {
				if o.equipe == nil {
					continue
				}
				if apres[*o.equipe] == nil {
					apres[*o.equipe] = map[int]bool{}
				}
				apres[*o.equipe][placeAffichee(e, iv)] = true
			}
		}
		t.Logf("%s (taille %d) : places par equipe avant %s, apres %s ; ramenees %d, bots releves %d (dont en jeu %d), absences tenues %d ; occupants simultanes max apres %s",
			id, taille, tailleFormat(avant), tailleFormat(apres), bilan.ramenees, bilan.releves, bilan.relevesEnJeu,
			bilan.absencesTenues, tailleSimultanes(pp))
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

// tailleSimultanes rend, par equipe, le plus grand nombre d'intervalles AFFICHES a une meme frame :
// l'invariant « jamais plus d'occupants que de places ».
func tailleSimultanes(pp *poseDesPlaces) string {
	parEquipe := map[int][]borneDAffichage{}
	for i := range pp.roster {
		o := pp.occ.parEntree[i]
		if o.equipe == nil {
			continue
		}
		for _, iv := range o.presence {
			if iv.aMax >= iv.de {
				parEquipe[*o.equipe] = append(parEquipe[*o.equipe], borneDAffichage{iv.de, +1}, borneDAffichage{iv.aMax + 1, -1})
			}
		}
	}
	m := map[int]map[int]bool{}
	for t, b := range parEquipe {
		maxi, _ := balayerLesBornes(b, sansPlafond)
		m[t] = map[int]bool{}
		for k := range maxi {
			m[t][k] = true
		}
	}
	return tailleFormat(m)
}

// TestTailleSurToutLeParc rejoue la regle sur TOUS les artefacts du dossier (lecture seule, aucune
// cuisson) et compte ceux ou une equipe garde plus de places, ou plus d'occupants simultanes, que sa
// taille. La taille n'est pas dans l'artefact : elle se deduit ici de la table du debut (sieges `lu`
// repartis entre deux equipes, arrondis a 2, 4, 8 ou 12) — un instrument, pas la regle de production.
//
//	TAILLE_ARTEFACTS=<dossier> TAILLE_PARC=1 go test -tags=research ./internal/games/halo_infinite/film/replay -run TestTailleSurToutLeParc -v
func TestTailleSurToutLeParc(t *testing.T) {
	dir := os.Getenv("TAILLE_ARTEFACTS")
	if dir == "" || os.Getenv("TAILLE_PARC") == "" {
		t.Skip("TAILLE_ARTEFACTS ou TAILLE_PARC absent : instrument saute")
	}
	noms, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lus, horsDeuxEquipes, avantEnTrop, apresEnTrop, simultanesEnTrop, ramenes, enJeu int
	for _, nom := range noms {
		if strings.HasSuffix(nom, ".derived.json") {
			continue
		}
		raw, err := os.ReadFile(nom)
		if err != nil {
			t.Fatal(err)
		}
		var a tailleArtefact
		if json.Unmarshal(raw, &a) != nil || len(a.Roster) == 0 {
			continue
		}
		lus++
		taille, ok := tailleDeLaTable(a)
		if !ok {
			horsDeuxEquipes++
			continue
		}
		if depasse(taillePlacesParEquipe(a), taille) {
			avantEnTrop++
		}
		pp := tailleRefaireLaPose(a, taille)
		bilan := pp.ramenerALaTaille()
		pp.bornerAuSuccesseur()
		pp.retirerLesAffichagesVides()
		ramenes += bilan.ramenees
		enJeu += bilan.relevesEnJeu
		apres := map[int]map[int]bool{}
		for i, e := range pp.roster {
			o := pp.occ.parEntree[i]
			for _, iv := range o.presence {
				if o.equipe != nil {
					if apres[*o.equipe] == nil {
						apres[*o.equipe] = map[int]bool{}
					}
					apres[*o.equipe][placeAffichee(e, iv)] = true
				}
			}
		}
		id := strings.TrimSuffix(filepath.Base(nom), ".json")
		if depasse(apres, taille) {
			apresEnTrop++
			t.Logf("RESTE EN TROP %s (taille %d) : places %s", id, taille, tailleFormat(apres))
		}
		if simultanesAuDela(pp, taille) {
			simultanesEnTrop++
			t.Logf("OCCUPANTS EN TROP %s (taille %d) : %s", id, taille, tailleSimultanes(pp))
		}
	}
	t.Logf("PARC : %d artefacts a roster, %d hors deux equipes ; places en trop avant %d, apres %d ; "+
		"occupants simultanes en trop apres %d ; intervalles ramenes %d, bots releves encore en jeu %d",
		lus, horsDeuxEquipes, avantEnTrop, apresEnTrop, simultanesEnTrop, ramenes, enJeu)
}

// tailleDeLaTable deduit la taille d'equipe de la table du debut (cf. TestTailleSurToutLeParc).
func tailleDeLaTable(a tailleArtefact) (int, bool) {
	equipes, lus := map[int]bool{}, map[int]bool{}
	for _, e := range a.Roster {
		if e.Team != nil {
			equipes[*e.Team] = true
		}
		if e.SeatSrc == SeatSourceLu {
			lus[e.Seat] = true
		}
	}
	if len(equipes) != 2 {
		return 0, false
	}
	demi := (len(lus) + 1) / 2
	for _, s := range []int{2, 4, 8, 12} {
		if demi <= s {
			return s, true
		}
	}
	return 0, false
}

// depasse dit qu'une equipe tient plus de places que `taille`.
func depasse(m map[int]map[int]bool, taille int) bool {
	for _, places := range m {
		if len(places) > taille {
			return true
		}
	}
	return false
}

// simultanesAuDela dit qu'une equipe affiche plus de `taille` intervalles a une meme frame.
func simultanesAuDela(pp *poseDesPlaces, taille int) bool {
	parEquipe := map[int][]borneDAffichage{}
	for i := range pp.roster {
		o := pp.occ.parEntree[i]
		if o.equipe == nil {
			continue
		}
		for _, iv := range o.presence {
			if iv.aMax >= iv.de {
				parEquipe[*o.equipe] = append(parEquipe[*o.equipe], borneDAffichage{iv.de, +1}, borneDAffichage{iv.aMax + 1, -1})
			}
		}
	}
	for _, b := range parEquipe {
		if maxi, _ := balayerLesBornes(b, sansPlafond); maxi > taille {
			return true
		}
	}
	return false
}
