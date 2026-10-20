package service

// solo_lives_block_test.go — « ISOLEMENT » DE CHAQUE JOUEUR DE L'ÉQUIPE (Vue match) : une seule
// lecture pour tous les joueurs (ADR 0036 I4), un bilan par joueur demandé — à zéro pour un joueur
// sans vie lue —, et les mêmes dégradations que la lecture d'un joueur.

import (
	"context"
	"errors"
	"slices"
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// viesCampFixes — le dépôt des vies de l'équipe : une lecture par joueur ou une erreur.
type viesCampFixes struct {
	parJoueur map[string]domain.ViesLues
	err       error
	appels    int
	xuids     []string
}

func (v *viesCampFixes) LoadLivesNearTeammateForPlayers(_ context.Context, _ []string, xuids []string) (map[string]domain.ViesLues, error) {
	v.appels++
	v.xuids = xuids
	out := map[string]domain.ViesLues{}
	for _, x := range xuids {
		out[x] = v.parJoueur[x]
	}
	return out, v.err
}

// viesDeA : A meurt une fois loin de tout coéquipier (40 m, portée 18 m) après un frag.
func viesDeA() domain.ViesLues {
	zero, un := 0, 1
	return domain.ViesLues{
		Vies:      []domain.VieLue{{MatchID: "m1", StartMS: 0, EndCause: coordination.CauseVieMort}},
		Morts:     []domain.MortSituee{{MatchID: "m1", TimeMS: 30_000, PlusProcheM: distanceDeTest(40)}},
		Frags:     []domain.FragLu{{MatchID: "m1", TimeMS: 10_000, CampTueur: &zero, CampVictime: &un}},
		Variantes: map[string]string{"m1": "Slayer:Arena"},
	}
}

func TestLireViesDuCamp_UneLecturePourTouteLEquipe(t *testing.T) {
	p := viesDeTest()
	p.Variantes = map[string]string{"m1": "Slayer:Arena"}
	repo := &viesCampFixes{parJoueur: map[string]domain.ViesLues{"P": p, "A": viesDeA()}}
	got, lu := lireViesDuCamp(context.Background(), viesCampQuery{
		Page: "match_view", Player: "Papa", XUIDs: []string{"P", "A", "Z"}, Repo: repo,
		Radar: map[string]int{"Slayer:Arena": 18}, MatchIDs: []string{"m1"},
	})
	if repo.appels != 1 || !slices.Equal(repo.xuids, []string{"P", "A", "Z"}) {
		t.Fatalf("lectures : %d pour %v, attendu une seule lecture des trois joueurs", repo.appels, repo.xuids)
	}
	if !lu || len(got) != 3 {
		t.Fatalf("bilans = %+v (lu %v), attendu trois entrées", got, lu)
	}
	if b := got["P"]; b.Near.Lives != 1 || b.Near.Kills != 1 {
		t.Errorf("P = %+v, attendu une vie près avec un frag", b)
	}
	if b := got["A"]; b.Alone.Lives != 1 || b.Alone.Kills != 1 {
		t.Errorf("A = %+v, attendu une vie seule avec un frag", b)
	}
	if b := got["Z"]; b != (domain.TimeseriesLivesNearTeammate{}) {
		t.Errorf("Z = %+v, attendu un bilan vide (aucune vie lue)", b)
	}
}

func TestLireViesDuCamp_Degradations(t *testing.T) {
	q := viesCampQuery{Page: "match_view", XUIDs: []string{"P"}, MatchIDs: []string{"m1"}}
	if got, lu := lireViesDuCamp(context.Background(), q); got != nil || lu {
		t.Errorf("repo nil : %+v %v, attendu rien", got, lu)
	}
	for _, err := range []error{games.ErrCapabilityNotSupported, errors.New("panne")} {
		q.Repo = &viesCampFixes{err: err}
		if got, lu := lireViesDuCamp(context.Background(), q); got != nil || lu {
			t.Errorf("%v : %+v %v, attendu rien", err, got, lu)
		}
	}
	q.Repo = &viesCampFixes{}
	if _, lu := lireViesDuCamp(context.Background(), q); lu {
		t.Errorf("aucune vie lue : attendu faux")
	}
}
