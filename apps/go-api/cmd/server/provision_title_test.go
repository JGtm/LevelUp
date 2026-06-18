//go:build integration

package main

// provision_title_test.go — preuve end-to-end « day-one 2e titre » (MT-16) : un
// titre additionnel découvert en config voit ses warehouses CRÉÉES + MIGRÉES,
// isolées sous data/titles/<slug>/, sans toucher aux DB Halo. Complète l'oracle
// migration (synthetic_title_b/migration_isolation_test.go) au niveau du boot
// glue (provisionAdditionalTitle : résolution de chemins + création de fichiers).

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const provisionTestSlug = "provision_test_title"

// registerProvisionTestSet enregistre un jeu de migrations minimal multi-target
// pour le titre de test (shared + metadata + shared_social + shared_pve), sans
// jamais combiner le registre global Halo → isolation totale.
func registerProvisionTestSet() {
	step := func(name string, tgt migration.TargetDB) migration.Migration {
		return migration.Migration{
			Name:        name,
			TargetDB:    tgt,
			Description: "titre de test — table de base " + string(tgt),
			ApplySchema: func(db *sql.DB) error {
				_, err := db.Exec("CREATE TABLE IF NOT EXISTS provisiontest_marker (k VARCHAR PRIMARY KEY)")
				return err
			},
		}
	}
	migration.RegisterMigrationSet(migration.TitleMigrationSet{
		Slug:           provisionTestSlug,
		CanonicalOrder: []string{"pt_shared", "pt_meta", "pt_social", "pt_pve"},
		Steps: func(target migration.TargetDB) []migration.Migration {
			switch target {
			case migration.TargetShared:
				return []migration.Migration{step("pt_shared", migration.TargetShared)}
			case migration.TargetMetadata:
				return []migration.Migration{step("pt_meta", migration.TargetMetadata)}
			case migration.TargetSharedSocial:
				return []migration.Migration{step("pt_social", migration.TargetSharedSocial)}
			case migration.TargetSharedPvE:
				return []migration.Migration{step("pt_pve", migration.TargetSharedPvE)}
			default:
				return nil
			}
		},
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func markerTableExists(t *testing.T, path string) bool {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'provisiontest_marker'",
	).Scan(&n); err != nil {
		t.Fatalf("query marker in %s: %v", path, err)
	}
	return n == 1
}

// TestProvisionAdditionalTitle_CreatesIsolatedDatabases : provisionAdditionalTitle
// crée les 4 warehouses du titre additionnel (PvE incluse car CapFirefight), avec
// SON marqueur, isolées sous data/titles/<slug>/ — et ne crée AUCUNE DB Halo.
func TestProvisionAdditionalTitle_CreatesIsolatedDatabases(t *testing.T) {
	registerProvisionTestSet()

	repoRoot := t.TempDir()
	pr := title.NewPathResolver(repoRoot)
	desc := &title.TitleDescriptor{
		Slug:         provisionTestSlug,
		Name:         "Provision Test",
		Provider:     "test",
		Status:       title.StatusActive,
		Capabilities: []title.Capability{title.CapMatchmaking, title.CapFirefight},
	}

	if err := provisionAdditionalTitle(pr, desc); err != nil {
		t.Fatalf("provisionAdditionalTitle: %v", err)
	}

	// 1. Les 4 warehouses du titre existent (PvE incluse via CapFirefight).
	for _, p := range []string{
		pr.MetadataDBPath(provisionTestSlug),
		pr.SharedDBPath(provisionTestSlug),
		pr.SharedSocialDBPath(provisionTestSlug),
		pr.SharedPVEDBPath(provisionTestSlug),
	} {
		if !fileExists(p) {
			t.Errorf("DB attendue absente: %s", p)
		}
	}

	// 2. Le marqueur du titre est présent dans shared + metadata (migrations routées).
	if !markerTableExists(t, pr.SharedDBPath(provisionTestSlug)) {
		t.Error("marqueur absent de shared — RunForTitleDB(shared) non appliqué")
	}
	if !markerTableExists(t, pr.MetadataDBPath(provisionTestSlug)) {
		t.Error("marqueur absent de metadata — RunForTitleDB(metadata) non appliqué")
	}

	// 3. ISOLATION : aucune DB Halo n'a été créée par ce provisioning.
	if fileExists(pr.SharedDBPath(title.DefaultSlug)) {
		t.Error("la DB shared Halo a été créée — fuite cross-titre (isolation cassée)")
	}
	if fileExists(pr.MetadataDBPath(title.DefaultSlug)) {
		t.Error("la DB metadata Halo a été créée — fuite cross-titre")
	}
}

// TestProvisionAdditionalTitle_NoFirefightSkipsPvE : sans CapFirefight, la DB PvE
// n'est PAS provisionnée (gating par capability, pas par slug).
func TestProvisionAdditionalTitle_NoFirefightSkipsPvE(t *testing.T) {
	registerProvisionTestSet()

	repoRoot := t.TempDir()
	pr := title.NewPathResolver(repoRoot)
	desc := &title.TitleDescriptor{
		Slug:         provisionTestSlug,
		Name:         "Provision Test",
		Status:       title.StatusActive,
		Capabilities: []title.Capability{title.CapMatchmaking}, // pas de Firefight
	}

	if err := provisionAdditionalTitle(pr, desc); err != nil {
		t.Fatalf("provisionAdditionalTitle: %v", err)
	}
	if fileExists(pr.SharedPVEDBPath(provisionTestSlug)) {
		t.Error("DB PvE provisionnée alors que CapFirefight est absente")
	}
	if !fileExists(pr.SharedDBPath(provisionTestSlug)) {
		t.Error("DB shared devrait être provisionnée")
	}
}
