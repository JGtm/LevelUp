//go:build cgo

package migrations

// boot_metadata_probe_test.go — LA SONDE « BOOT SUR UNE COPIE » de la metadata.
//
// # Pourquoi elle existe
//
// Le référentiel d'armes vit à DEUX endroits qui ne se propagent pas de la même façon :
//
//	le SEED Go (weaponRegistryWeapons)   pousse par `weapons.ReconcileRegistry` à chaque
//	                                     boot, mais en INSERT OR IGNORE : une clé NOUVELLE
//	                                     arrive, une valeur MODIFIÉE jamais ;
//	les STEPS de migration               seuls capables de corriger une ligne déjà semée
//	                                     (cf. metadata_reclass_sword_hammer_heavy_v1).
//
// Mesurer l'effet d'un changement de registre sur une base réelle exige donc de rejouer
// LES DEUX, exactement comme `cmd/server/main.go` le fait au démarrage. Sans cette sonde,
// il faudrait lancer un serveur — c'est-à-dire ouvrir la vraie base en écriture.
//
// C'est ce qu'exigent les étapes A6.6 et A6.9 du plan
// `.ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md` : relancer le témoin de bascule après
// l'ajout des véhicules/tourelles et le reclassement de l'épée et du marteau.
//
// # Elle ne s'arme que sur demande, et elle ÉCRIT
//
// Sans `BOOT_META_DB`, elle est sautée. La variable DOIT pointer une COPIE : la sonde
// ouvre en lecture/écriture. Ne jamais la pointer sur
// `data/titles/*/warehouse/metadata.duckdb` — le modèle mono-process l'interdit et le
// serveur tient ce fichier.
//
//	export PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1 CC=/c/msys64/ucrt64/bin/gcc.exe
//	BOOT_META_DB=<copie RW de metadata.duckdb> \
//	BOOT_META_TOML=<config/titles/halo_infinite/mappings/weapon_names.toml> \
//	go test ./internal/games/halo_infinite/migrations/ -run BootMetadataSurCopie -v -count=1

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/migration"
)

const (
	sondeBootEnvDB   = "BOOT_META_DB"
	sondeBootEnvTOML = "BOOT_META_TOML"
)

// TestBootMetadataSurCopie rejoue sur une copie ce que le démarrage du serveur fait sur la
// vraie base : les migrations metadata du titre, puis la réconciliation du registre, puis
// celle des libellés. Les trois, dans cet ordre — c'est celui de `cmd/server/main.go`.
func TestBootMetadataSurCopie(t *testing.T) {
	chemin := strings.TrimSpace(os.Getenv(sondeBootEnvDB))
	if chemin == "" {
		t.Skipf("sonde non armee : %s requis (COPIE de metadata.duckdb, ouverte en ECRITURE)", sondeBootEnvDB)
	}
	db, err := sql.Open("duckdb", chemin)
	if err != nil {
		t.Fatalf("ouverture %s : %v", chemin, err)
	}
	defer func() { _ = db.Close() }()

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForTitleDB(db, migration.DefaultSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("migrations metadata : %v", err)
	}
	n, err := weapons.ReconcileRegistry(db, migration.DefaultSlug)
	if err != nil {
		t.Fatalf("ReconcileRegistry : %v", err)
	}
	t.Logf("registre reconcilie : %d ligne(s) inseree(s)", n)

	if toml := strings.TrimSpace(os.Getenv(sondeBootEnvTOML)); toml != "" {
		m, err := weapons.ReconcileNameLabels(db, migration.DefaultSlug, toml)
		if err != nil {
			t.Fatalf("ReconcileNameLabels(%s) : %v", toml, err)
		}
		t.Logf("libelles reconcilies : %d ligne(s)", m)
	}

	// Contrôle de sortie : les trois populations que le lot a touchées.
	q := `SELECT COUNT(*) FROM weapons WHERE title_slug = ? AND class = ?`
	for _, classe := range []string{"vehicle", "turret", "heavy", "melee"} {
		var n int
		if err := db.QueryRow(q, migration.DefaultSlug, classe).Scan(&n); err != nil {
			t.Fatalf("controle classe %s : %v", classe, err)
		}
		t.Logf("classe %-11s : %d cle(s)", classe, n)
	}
}
