package wire

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// Un profil sans fichier de base joueur n'a pas de base : faux sans erreur, et la base n'est pas créée.
func TestScoringRevisionPlayers_ProfilSansBaseJoueur(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	p := scoringRevisionPlayers{r: &ServiceRegistry{cfg: cfg}}
	path := config.PlayerDBPath(cfg, "halo_infinite", "SansBase")

	present, err := p.HasPlayerDB("halo_infinite", domain.PlayerSummary{Gamertag: "SansBase", XUID: "1"})
	if err != nil || present {
		t.Fatalf("HasPlayerDB = (%v, %v), attendu (false, nil)", present, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("la base joueur %s existe après la lecture (stat err = %v) : elle ne doit pas la créer", path, err)
	}
}

// Un profil dont le fichier de base joueur existe a une base.
func TestScoringRevisionPlayers_ProfilAvecBaseJoueur(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	path := config.PlayerDBPath(cfg, "halo_infinite", "Joueur")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	p := scoringRevisionPlayers{r: &ServiceRegistry{cfg: cfg}}

	present, err := p.HasPlayerDB("halo_infinite", domain.PlayerSummary{Gamertag: "Joueur", XUID: "1"})
	if err != nil || !present {
		t.Fatalf("HasPlayerDB = (%v, %v), attendu (true, nil)", present, err)
	}
}
