package replayartifacts

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/replaychild"
)

// remplacerLanceur pose un faux lanceur d'enfant qui capture la requete envoyee, sans demarrer de processus.
func remplacerLanceur(t *testing.T) *[]replaychild.Request {
	t.Helper()
	var vues []replaychild.Request
	avant := spawnEnfant
	spawnEnfant = func(_ context.Context, req replaychild.Request) (replaychild.Result, error) {
		vues = append(vues, req)
		return replaychild.Result{}, nil
	}
	t.Cleanup(func() { spawnEnfant = avant })
	return &vues
}

// TestSpawnBuildOne_RecopieLeNombreDeFilsVersLEnfant : la requete de l'enfant porte les fils de la requete de
// construction (maillon « parent -> enfant »).
func TestSpawnBuildOne_RecopieLeNombreDeFilsVersLEnfant(t *testing.T) {
	vues := remplacerLanceur(t)
	if _, err := SpawnBuildOne(context.Background(), BuildOneRequest{MatchID: "m", DecoderThreads: 3}); err != nil {
		t.Fatal(err)
	}
	if len(*vues) != 1 || (*vues)[0].DecoderThreads != 3 {
		t.Fatalf("requetes de l'enfant %+v, attendu une requete a 3 fils", *vues)
	}
}

// TestNewHook_LaConstructionPoseLeNombreDeFilsDeLInstallation : le crochet de synchronisation construit ses
// rejeux avec les fils de la configuration, quelle que soit la requete recue (maillon « crochet »).
func TestNewHook_LaConstructionPoseLeNombreDeFilsDeLInstallation(t *testing.T) {
	vues := remplacerLanceur(t)
	cfg := &config.AppConfig{}
	cfg.Capabilities.DecoderThreads = 5
	hook := NewHook(cfg, nil, nil)
	if hook.BuildOne == nil {
		t.Fatal("le crochet n'a pas de fonction de construction")
	}
	if _, err := hook.BuildOne(context.Background(), BuildOneRequest{MatchID: "m"}); err != nil {
		t.Fatal(err)
	}
	if len(*vues) != 1 || (*vues)[0].DecoderThreads != 5 {
		t.Fatalf("requetes de l'enfant %+v, attendu une requete a 5 fils", *vues)
	}
}
