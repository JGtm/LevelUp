package wire

import (
	"context"
	"os"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// Un profil sans fichier de base joueur n'a rien à recalculer : le rejeu le saute sans erreur et ne
// crée pas la base.
func TestScoringRevisionApplier_ProfilSansBaseJoueurSaute(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	a := scoringRevisionApplier{r: &ServiceRegistry{cfg: cfg}}
	path := config.PlayerDBPath(cfg, "halo_infinite", "SansBase")

	applied, err := a.ApplyForPlayer(context.Background(), "halo_infinite",
		domain.PlayerSummary{Gamertag: "SansBase", XUID: "1"})
	if err != nil || applied {
		t.Fatalf("applied=%v err=%v, attendu false, nil", applied, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("la base joueur %s existe après le saut (stat err = %v) : le rejeu ne doit pas la créer", path, err)
	}
}
