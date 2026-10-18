package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
)

// backfillTitleFixture : un db_profiles v3 où JGtm est suivi sur les deux titres et Solo sur
// Infinite seulement, dans une racine de dépôt temporaire.
func backfillTitleFixture(t *testing.T) (*BackfillHandler, string) {
	t.Helper()
	root := t.TempDir()
	profiles := `{"version":"3.0","admin":"JGtm","profiles":{
		"halo_infinite":{"JGtm":{"xuid":"1001"},"Solo":{"xuid":"1002"}},
		"halo_5":{"JGtm":{"xuid":"1001"}}}}`
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	return NewBackfillHandler(cfg, jobs.NewStore(t.TempDir())), root
}

func startBackfill(h *BackfillHandler, body map[string]any) (*backfillStartOutput, error) {
	raw, _ := json.Marshal(body)
	return h.handleStartBackfill(context.Background(), &backfillStartInput{RawBody: raw})
}

// TestStartBackfill_TitreRefuseUnJoueurNonSuiviSurCeTitre : le champ title restreint la
// recherche aux profils de CE titre ; un titre inconnu ou un joueur absent du titre répond 404
// et ne lance aucun job.
func TestStartBackfill_TitreRefuseUnJoueurNonSuiviSurCeTitre(t *testing.T) {
	h, _ := backfillTitleFixture(t)
	for name, body := range map[string]map[string]any{
		"joueur d'Infinite seulement, titre halo_5": {"player_slug": "Solo", "title": "halo_5", "dry_run": true},
		"titre inconnu": {"player_slug": "JGtm", "title": "halo_9", "dry_run": true},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := startBackfill(h, body)
			var se huma.StatusError
			if !errors.As(err, &se) || se.GetStatus() != http.StatusNotFound {
				t.Fatalf("attendu 404, got %v", err)
			}
		})
	}
}

// TestStartBackfill_TitreRoute : sous title=halo_5 le moteur travaille sur les bases de Halo 5
// (la base joueur du titre apparaît), jamais sur celles du titre par défaut ; sans title, le
// comportement d'avant (titre par défaut) est inchangé.
func TestStartBackfill_TitreRoute(t *testing.T) {
	resolver := func(root string) *titlePkg.PathResolver { return titlePkg.NewPathResolver(root) }
	for _, c := range []struct {
		name, title, vers, pasVers string
	}{
		{"halo_5", "halo_5", "halo_5", titlePkg.DefaultSlug},
		{"sans titre", "", titlePkg.DefaultSlug, "halo_5"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h, root := backfillTitleFixture(t)
			out, err := startBackfill(h, map[string]any{"player_slug": "JGtm", "title": c.title, "dry_run": true})
			if err != nil {
				t.Fatalf("startBackfill: %v", err)
			}
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				if job := h.jobStore.Get(out.Body.JobID); job != nil && job.IsTerminal() {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if job := h.jobStore.Get(out.Body.JobID); job == nil || !job.IsTerminal() {
				t.Fatalf("le job n'est pas terminé: %+v", job)
			}
			pr := resolver(root)
			if _, err := os.Stat(pr.PlayerDBPath(c.vers, "JGtm")); err != nil {
				t.Errorf("la base joueur de %s devait être ouverte par le moteur: %v", c.vers, err)
			}
			if _, err := os.Stat(pr.PlayerDBPath(c.pasVers, "JGtm")); !os.IsNotExist(err) {
				t.Errorf("la base joueur de %s ne devait pas être touchée (stat: %v)", c.pasVers, err)
			}
		})
	}
}
