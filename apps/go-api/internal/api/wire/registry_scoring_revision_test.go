package wire

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Un profil sans fichier de base joueur n'a rien à recalculer : le rejeu le saute sans erreur ni issue
// d'étape et ne crée pas la base.
func TestScoringRevisionApplier_ProfilSansBaseJoueurSaute(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	a := scoringRevisionApplier{r: &ServiceRegistry{cfg: cfg}}
	path := config.PlayerDBPath(cfg, "halo_infinite", "SansBase")

	outcomes := a.ApplyForPlayer(context.Background(), "halo_infinite",
		domain.PlayerSummary{Gamertag: "SansBase", XUID: "1"})
	if len(outcomes) != 0 {
		t.Fatalf("issues = %+v, attendu aucune", outcomes)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("la base joueur %s existe après le saut (stat err = %v) : le rejeu ne doit pas la créer", path, err)
	}
}

// applierAvecBase : un applicateur dont la base joueur existe (fichier vide : les étapes sont des faux) et
// dont les deux étapes sont remplacées.
func applierAvecBase(
	t *testing.T,
	kpi func() (bool, error),
	campagnes func() (bool, error),
) (scoringRevisionApplier, domain.PlayerSummary, *[]string) {
	t.Helper()
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	path := config.PlayerDBPath(cfg, "halo_infinite", "Joueur")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var appels []string
	a := scoringRevisionApplier{
		r: &ServiceRegistry{cfg: cfg},
		kpiStep: func(context.Context, string, domain.PlayerSummary, string) (bool, error) {
			appels = append(appels, "kpi")
			return kpi()
		},
		campaignStep: func(context.Context, string, domain.PlayerSummary) (bool, error) {
			appels = append(appels, "campagnes")
			return campagnes()
		},
	}
	return a, domain.PlayerSummary{Gamertag: "Joueur", XUID: "1"}, &appels
}

func attendreIssues(t *testing.T, got []port.RevisionOutcome, want []port.RevisionOutcome) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("issues = %+v, attendu %+v", got, want)
	}
	for i := range want {
		if got[i].Step != want[i].Step || got[i].Applied != want[i].Applied || !errors.Is(got[i].Err, want[i].Err) {
			t.Fatalf("issue %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
}

// L'étape des notes échoue : celle des campagnes tourne quand même et son issue est rendue à part.
func TestScoringRevisionApplier_EchecDesNotesNeRetientPasLesCampagnes(t *testing.T) {
	boom := errors.New("rejeu refusé")
	a, p, appels := applierAvecBase(t,
		func() (bool, error) { return false, boom },
		func() (bool, error) { return true, nil })

	got := a.ApplyForPlayer(context.Background(), "halo_infinite", p)

	if len(*appels) != 2 {
		t.Fatalf("étapes jouées = %v, attendu les deux", *appels)
	}
	attendreIssues(t, got, []port.RevisionOutcome{
		{Step: port.RevisionStepKPI, Err: boom},
		{Step: port.RevisionStepCampaigns, Applied: true},
	})
}

// L'étape des campagnes échoue : celle des notes a tourné et reste appliquée.
func TestScoringRevisionApplier_EchecDesCampagnesNeDefaitPasLesNotes(t *testing.T) {
	boom := errors.New("lease refusé")
	a, p, appels := applierAvecBase(t,
		func() (bool, error) { return true, nil },
		func() (bool, error) { return false, boom })

	got := a.ApplyForPlayer(context.Background(), "halo_infinite", p)

	if len(*appels) != 2 || (*appels)[0] != "kpi" || (*appels)[1] != "campagnes" {
		t.Fatalf("étapes jouées = %v, attendu kpi puis campagnes", *appels)
	}
	attendreIssues(t, got, []port.RevisionOutcome{
		{Step: port.RevisionStepKPI, Applied: true},
		{Step: port.RevisionStepCampaigns, Err: boom},
	})
}
