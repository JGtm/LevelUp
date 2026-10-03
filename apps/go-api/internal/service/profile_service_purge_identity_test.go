// Package service — profile_service_purge_identity_test.go : PurgeIdentityData,
// branche « dossier joueur non supprimable » (revue adversariale ronde 2,
// 2026-09-16 : le journal existait, aucun test ne l'exerçait).
package service

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

func newPurgeIdentityFixture(t *testing.T) (*ProfileService, *config.AppConfig, string) {
	t.Helper()
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {"Spartan": {"db_path": "x", "xuid": "111"}}
  }
}`), 0o644); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	dir := title.NewPathResolver(root).PlayerDir("halo_infinite", "Spartan")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("dossier joueur: %v", err)
	}
	return NewProfileService(profilesPath, root), &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}, dir
}

// captureLogs redirige le journal par défaut le temps du test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestPurgeIdentityData_DossierNonSupprimable_JournaliseEtRendFalse(t *testing.T) {
	svc, cfg, dir := newPurgeIdentityFixture(t)
	logs := captureLogs(t)
	boom := errors.New("EBUSY: dossier tenu par un autre process")
	svc.WithRemoveAll(func(string) error { return boom })

	removed, err := svc.PurgeIdentityData("Spartan", []string{"halo_infinite"})
	if err != nil {
		t.Fatalf("PurgeIdentityData: %v", err)
	}
	if removed["halo_infinite"] {
		t.Fatal("dossier non supprime : removed doit valoir false")
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatal("le faux RemoveAll ne supprime rien : le dossier doit toujours exister")
	}
	// L'entrée de profil, elle, est bien retirée : le refus disque n'annule pas
	// la mutation atomique du store.
	players, lerr := cfg.LoadPlayers()
	if lerr != nil {
		t.Fatalf("LoadPlayers: %v", lerr)
	}
	if len(players) != 0 {
		t.Fatalf("profil encore present : %+v", players)
	}
	// Règle 3 : la cause est journalisée AVANT la dégradation en booléen.
	out := logs.String()
	if !bytes.Contains([]byte(out), []byte("dossier joueur non supprim")) || !bytes.Contains([]byte(out), []byte("EBUSY")) {
		t.Fatalf("journal attendu avec la cause, obtenu :\n%s", out)
	}
}

func TestPurgeIdentityData_CheminNominal_SupprimeEtRendTrue(t *testing.T) {
	svc, _, dir := newPurgeIdentityFixture(t)
	removed, err := svc.PurgeIdentityData("Spartan", []string{"halo_infinite"})
	if err != nil {
		t.Fatalf("PurgeIdentityData: %v", err)
	}
	if !removed["halo_infinite"] {
		t.Fatal("dossier supprime : removed doit valoir true")
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Fatal("le dossier joueur devrait avoir disparu")
	}
}
