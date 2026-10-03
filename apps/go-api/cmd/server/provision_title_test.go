//go:build integration

package main

// provision_title_test.go — preuve end-to-end « day-one 2e titre » (MT-16) : un
// titre additionnel découvert en config voit ses warehouses CRÉÉES + MIGRÉES,
// isolées sous data/titles/<slug>/, sans toucher aux DB Halo. Complète l'oracle
// migration (synthetic_title_b/migration_isolation_test.go) au niveau du boot
// glue (provisionAdditionalTitle : résolution de chemins + création de fichiers).

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const provisionTestSlug = "provision_test_title"

// provisionFailMetaSlug — titre de test dont la migration METADATA échoue, comme halo_5
// entre le 2026-09-12 et le 2026-09-20 (name_en NOT NULL violé par le seed du registre
// d'armes).
const provisionFailMetaSlug = "provision_fail_meta_title"

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

// registerProvisionFailMetaSet — même jeu minimal, mais la migration METADATA échoue.
// Reproduit l'anomalie de boot du 2026-09-12 : la metadata de halo_5 tombait sur une
// contrainte NOT NULL, et la boucle de provisioning sortait à cet échec.
func registerProvisionFailMetaSet() {
	step := func(name string, tgt migration.TargetDB) migration.Migration {
		return migration.Migration{
			Name:        name,
			TargetDB:    tgt,
			Description: "titre de test (metadata fautive) — table de base " + string(tgt),
			ApplySchema: func(db *sql.DB) error {
				if tgt == migration.TargetMetadata {
					return errors.New("NOT NULL constraint failed: weapons.name_en")
				}
				_, err := db.Exec("CREATE TABLE IF NOT EXISTS provisiontest_marker (k VARCHAR PRIMARY KEY)")
				return err
			},
		}
	}
	migration.RegisterMigrationSet(migration.TitleMigrationSet{
		Slug:           provisionFailMetaSlug,
		CanonicalOrder: []string{"pfm_shared", "pfm_meta", "pfm_social", "pfm_pve"},
		Steps: func(target migration.TargetDB) []migration.Migration {
			switch target {
			case migration.TargetShared:
				return []migration.Migration{step("pfm_shared", migration.TargetShared)}
			case migration.TargetMetadata:
				return []migration.Migration{step("pfm_meta", migration.TargetMetadata)}
			case migration.TargetSharedSocial:
				return []migration.Migration{step("pfm_social", migration.TargetSharedSocial)}
			case migration.TargetSharedPvE:
				return []migration.Migration{step("pfm_pve", migration.TargetSharedPvE)}
			default:
				return nil
			}
		},
	})
}

// TestProvisionAdditionalTitle_MetadataEnEchecNEmportePasLesAutresBases — LA régression
// du 2026-09-12 : la metadata échoue, et shared + social doivent malgré tout recevoir
// leurs migrations. L'erreur retournée nomme la SEULE base fautive.
func TestProvisionAdditionalTitle_MetadataEnEchecNEmportePasLesAutresBases(t *testing.T) {
	registerProvisionFailMetaSet()

	repoRoot := t.TempDir()
	pr := title.NewPathResolver(repoRoot)
	desc := &title.TitleDescriptor{
		Slug:         provisionFailMetaSlug,
		Name:         "Provision Fail Meta",
		Provider:     "test",
		Status:       title.StatusActive,
		Capabilities: []title.Capability{title.CapMatchmaking},
	}

	err := provisionAdditionalTitle(pr, desc)
	if err == nil {
		t.Fatal("provisionAdditionalTitle: erreur attendue (la metadata échoue)")
	}

	// 1. Les bases NON fautives ont bien été créées ET migrées.
	if !markerTableExists(t, pr.SharedDBPath(provisionFailMetaSlug)) {
		t.Error("marqueur absent de shared — la base a été ABANDONNÉE après l'échec de la metadata")
	}
	if !markerTableExists(t, pr.SharedSocialDBPath(provisionFailMetaSlug)) {
		t.Error("marqueur absent de shared_social — la base a été ABANDONNÉE après l'échec de la metadata")
	}

	// 2. L'erreur nomme la base fautive, et elle seule.
	if got := failedProvisionTargets(err); len(got) != 1 || got[0] != string(migration.TargetMetadata) {
		t.Errorf("failedProvisionTargets = %v, want [%s]", got, migration.TargetMetadata)
	}
	if !strings.Contains(err.Error(), "name_en") {
		t.Errorf("l'erreur ne porte pas la cause de l'échec metadata : %v", err)
	}
}
