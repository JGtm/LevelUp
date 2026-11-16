package leadshareload

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/games"
)

type depotFactice struct {
	rendu map[string]leadshare.PlayerShare
	err   error
	appel int
}

func (d *depotFactice) LoadLeadShare(_ context.Context, _ []string, _ string) (map[string]leadshare.PlayerShare, error) {
	d.appel++
	return d.rendu, d.err
}

func TestLoad_RienAPoser(t *testing.T) {
	ids := []string{"m1"}
	for nom, c := range map[string]struct {
		repo *depotFactice
		xuid string
		ids  []string
	}{
		"joueur vide":        {&depotFactice{}, "", ids},
		"liste vide":         {&depotFactice{}, "xu1", nil},
		"capability absente": {&depotFactice{err: games.ErrCapabilityNotSupported}, "xu1", ids},
		"lecture en échec":   {&depotFactice{err: errors.New("boom")}, "xu1", ids},
	} {
		if got, ok := Load(context.Background(), c.repo, c.ids, c.xuid, "test"); ok || got != nil {
			t.Errorf("%s : rien à poser attendu, obtenu ok=%v got=%v", nom, ok, got)
		}
	}
}

func TestLoad_DepotNilEtLecture(t *testing.T) {
	if _, ok := Load(context.Background(), nil, []string{"m1"}, "xu1", "test"); ok {
		t.Fatal("un lecteur nil ne lit rien")
	}
	d := &depotFactice{rendu: map[string]leadshare.PlayerShare{"m1": {Share: 0.4, Changes: 2}}}
	got, ok := Load(context.Background(), d, []string{"m1"}, "xu1", "test")
	if !ok || got["m1"].Changes != 2 || d.appel != 1 {
		t.Fatalf("une lecture attendue : ok=%v got=%v appels=%d", ok, got, d.appel)
	}
}

func TestPointers_NePartagentRien(t *testing.T) {
	v := leadshare.PlayerShare{Share: 0.25, Changes: 3}
	s, c := Pointers(v)
	*s, *c = 0.9, 9
	if v.Share != 0.25 || v.Changes != 3 {
		t.Fatalf("la valeur source a été modifiée : %+v", v)
	}
}
