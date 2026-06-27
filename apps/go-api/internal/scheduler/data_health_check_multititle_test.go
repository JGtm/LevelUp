// Package scheduler_test — data_health_check_multititle_test.go : preuve que
// l'audit data-health couvre PLUS D'UN titre (registry.All()), pas seulement
// halo_infinite.
//
// Le HealthScheduler itère désormais sur DefaultRegistry().All() et agrège les
// compteurs par-titre dans le même DataHealthCheckResult. Ce test enregistre un
// 2e titre (halo_5) dans le registre partagé, sème une anomalie UNIQUEMENT dans
// sa shared DB, et vérifie qu'elle remonte dans le total — ce qui est impossible
// si la boucle restait mono-titre (DefaultSlug).

package scheduler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// seedTitleSharedDB crée la shared DB (schéma TargetShared) du titre donné sous
// repoRoot et retourne son chemin.
func seedTitleSharedDB(t *testing.T, repoRoot, slug string) string {
	t.Helper()
	sharedPath := title.NewPathResolver(repoRoot).SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir warehouse (%s): %v", slug, err)
	}
	shared, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared (%s): %v", slug, err)
	}
	if err := migration.RunForDB(shared.SQLDb(), migration.TargetShared); err != nil {
		shared.Close()
		t.Fatalf("RunForDB(Shared, %s): %v", slug, err)
	}
	shared.Close()
	return sharedPath
}

// useTwoTitleRegistry remplace le registre partagé par {halo_infinite + halo_5}
// le temps du test, puis le restaure (évite de polluer les autres tests qui
// s'appuient sur le built-in mono-titre).
func useTwoTitleRegistry(t *testing.T) {
	t.Helper()
	reg := title.NewRegistry() // built-in halo_infinite (active)
	reg.Register(&title.TitleDescriptor{
		Slug:   "halo_5",
		Name:   "Halo 5",
		Status: title.StatusActive,
	})
	title.SetDefaultRegistry(reg)
	t.Cleanup(func() { title.SetDefaultRegistry(title.NewRegistry()) })
}

// TestHealthScheduler_E2E_AuditsMultipleTitles vérifie que l'audit couvre le 2e
// titre : une anomalie UUID semée UNIQUEMENT dans la shared DB halo_5 doit être
// comptée. Si la boucle restait mono-titre (halo_infinite), UUIDsRawCount serait
// 0 (la shared infinite est vierge).
func TestHealthScheduler_E2E_AuditsMultipleTitles(t *testing.T) {
	useTwoTitleRegistry(t)

	repoRoot := t.TempDir()
	// Les deux titres ont une shared DB (sinon halo_5 serait skippé proprement).
	seedTitleSharedDB(t, repoRoot, "halo_infinite")
	h5Shared := seedTitleSharedDB(t, repoRoot, "halo_5")

	// Anomalie UUID brut UNIQUEMENT dans halo_5.
	seedRawUUIDMapName(t, h5Shared)

	sched := scheduler.NewDataHealthScheduler(repoRoot)
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	// L'anomalie semée dans halo_5 doit remonter ⇒ preuve que >1 titre est audité.
	if res.UUIDsRawCount < 1 {
		t.Errorf("UUIDsRawCount: attendu >= 1 (anomalie halo_5), obtenu %d — la boucle ne couvre pas halo_5", res.UUIDsRawCount)
	}
	if res.WarningsTotal < 1 {
		t.Errorf("WarningsTotal: attendu >= 1, obtenu %d", res.WarningsTotal)
	}
}

// TestHealthScheduler_E2E_TitleWithoutSharedDB_SkippedGracefully vérifie qu'un
// titre enregistré SANS shared DB (cas live-only pas encore backfillé) est skippé
// proprement : pas de panic, et l'audit du titre présent (halo_infinite) aboutit.
func TestHealthScheduler_E2E_TitleWithoutSharedDB_SkippedGracefully(t *testing.T) {
	useTwoTitleRegistry(t)

	repoRoot := t.TempDir()
	// Seul halo_infinite a une shared DB ; halo_5 n'en a pas → doit être skippé.
	seedTitleSharedDB(t, repoRoot, "halo_infinite")

	sched := scheduler.NewDataHealthScheduler(repoRoot)
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil (panic/avorté alors qu'halo_infinite est présent)")
	}
	// halo_infinite vierge ⇒ aucun warning, et le cycle a bien tourné (pas avorté).
	if res.WarningsTotal != 0 {
		t.Errorf("WarningsTotal: attendu 0 sur titres vierges, obtenu %d", res.WarningsTotal)
	}
}
