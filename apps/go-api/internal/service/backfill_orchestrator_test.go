package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	go_sync "levelup/go-api/internal/sync"
)

// warnUnimplemented a été extrait du handler backfill vers l'orchestrateur (K1f).
// Ces tests (ex-handlers/backfill_test.go) valident qu'il liste les types non
// implémentés (medals/skill/aliases) et PAS ceux désormais implémentés (events).

// TestWarnUnimplemented_EventsNotListed (Phase 2 PLAN_HIGHLIGHT_EVENTS_BACKFILL) :
// scope.Events=true ne doit PLUS produire une warning listant 'events'.
func TestWarnUnimplemented_EventsNotListed(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "player-test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Events: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if got == nil {
		t.Fatalf("Get(%s) returned nil", job.JobID)
	}
	for _, w := range got.Warnings {
		if strings.Contains(w, "events") {
			t.Errorf("warning ne doit PAS lister 'events' (Phase 2 livrée) : %q", w)
		}
	}
}

// TestWarnUnimplemented_StillListsOthers : les types non implémentés (medals, skill)
// doivent rester listés.
func TestWarnUnimplemented_StillListsOthers(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "player-test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Medals: true, Skill: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if got == nil {
		t.Fatalf("Get(%s) returned nil", job.JobID)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("attendu une warning pour medals+skill non-implémentés")
	}
	hasMedals := false
	hasSkill := false
	for _, w := range got.Warnings {
		if strings.Contains(w, "medals") {
			hasMedals = true
		}
		if strings.Contains(w, "skill") {
			hasSkill = true
		}
	}
	if !hasMedals {
		t.Error("warning devrait mentionner medals")
	}
	if !hasSkill {
		t.Error("warning devrait mentionner skill")
	}
}

func TestWarnUnimplemented_NoTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{}} // no flags

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d", len(got.Warnings))
	}
}

func TestWarnUnimplemented_WithTypes(t *testing.T) {
	store := jobs.NewStore(t.TempDir())
	job := store.Create(domain.JobTypeBackfill, "test")
	o := &BackfillOrchestrator{jobStore: store, scope: &go_sync.SyncScope{Medals: true, Events: true}}

	o.warnUnimplemented(job.JobID)

	got := store.Get(job.JobID)
	if len(got.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(got.Warnings))
	}
}

// repoRootAvecManifestes : une racine de dépôt temporaire qui ne porte que les manifestes des
// titres (config/titles). Un moteur qui tournerait à tort y créerait ses bases dans le dossier
// temporaire, jamais dans le dépôt.
func repoRootAvecManifestes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("..", "..", "..", "..", "config", "titles")
	if err := os.CopyFS(filepath.Join(root, "config", "titles"), os.DirFS(src)); err != nil {
		t.Fatalf("copie des manifestes de titres: %v", err)
	}
	return root
}

// TestRunCitationsComeback_TitreSansMoteurDeCitations : sous le scope « tout » (AllData inclut les
// citations), le job d'un titre qui ne déclare pas citations.engine (Halo 5 : citations natives)
// avertit « non supporté pour ce titre » et ne lance pas le moteur ; celui d'Infinite l'est.
func TestRunCitationsComeback_TitreSansMoteurDeCitations(t *testing.T) {
	root := repoRootAvecManifestes(t)
	cas := []struct {
		titre       string
		servi       bool
		attendJoint string
	}{
		{"halo_5", false, "non supporte pour ce titre"},
		{"halo_infinite", true, ""},
	}
	for _, c := range cas {
		t.Run(c.titre, func(t *testing.T) {
			store := jobs.NewStore(t.TempDir())
			job := store.Create(domain.JobTypeBackfill, "player-test")
			engine := go_sync.NewSyncEngineForTitle(root, c.titre, "zz_citations", "1", nil, nil)
			o := &BackfillOrchestrator{engine: engine, jobStore: store, scope: &go_sync.SyncScope{Citations: true}}

			if got := o.citationsServed(job.JobID); got != c.servi {
				t.Fatalf("citationsServed(%s) = %v, attendu %v", c.titre, got, c.servi)
			}
			warnings := store.Get(job.JobID).Warnings
			if c.servi {
				if len(warnings) != 0 {
					t.Fatalf("titre servi : aucun avertissement attendu, got %v", warnings)
				}
				return
			}
			if len(warnings) != 1 || !strings.Contains(warnings[0], c.attendJoint) {
				t.Fatalf("avertissement attendu %q, got %v", c.attendJoint, warnings)
			}

			// Le pipeline ne touche à aucune base du titre : ni passe ni création de fichier.
			var counts backfillCounts
			o.runCitationsComeback(job.JobID, &counts)
			if counts.citationsUpdated != 0 {
				t.Errorf("citationsUpdated = %d, attendu 0", counts.citationsUpdated)
			}
			if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
				t.Errorf("le moteur de citations a créé des bases pour %s (stat data: %v)", c.titre, err)
			}
		})
	}
}
