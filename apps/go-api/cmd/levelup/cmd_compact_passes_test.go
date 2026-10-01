//go:build cgo

package main

// cmd_compact_passes_test.go — la commande `compact-passes` (DC.4, DC.5) sur une base partagée
// au schéma réel (chaîne de migration complète) : dry-run sans écriture ni sauvegarde, refus
// quand le bail d'écrivain est tenu, sauvegarde préalable qui porte la base d'avant, compaction,
// idempotence, et `--rewrite-file` (fichier neuf à l'inventaire identique, ancien gardé, séquence
// qui continue).

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// baseDeTest : une base partagée au schéma réel, `match_bomb_stats` semée de 6 versions pour 3
// lignes servies. Fermée au retour (la commande l'ouvre elle-même).
func baseDeTest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared_matches_v2.duckdb")
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(h.SQLDb(), migration.TargetShared); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	for i, l := range [][2]string{{"m1", "x1"}, {"m1", "x1"}, {"m1", "x2"}, {"m2", "x1"}, {"m1", "x1"}, {"m2", "x1"}} {
		if _, err := h.SQLDb().Exec(`INSERT INTO match_bomb_stats (match_id, xuid, bomb_arms, written_at)
			VALUES (?, ?, ?, TIMESTAMP '2026-01-01 00:00:00' + to_seconds(?))`, l[0], l[1], i, i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func lignes(t *testing.T, path, table string) int64 {
	t.Helper()
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer h.Close() //nolint:errcheck
	var n int64
	if err := h.SQLDb().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func fichiers(t *testing.T, dir, motif string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, motif))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCompactPasses_DryRunPuisSauvegardeEtCompaction(t *testing.T) {
	path := baseDeTest(t)
	dir := filepath.Dir(path)
	ctx := context.Background()

	if err := compacterTitre(ctx, "halo_infinite", path, compactPassesOptions{dryRun: true}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if n := lignes(t, path, "match_bomb_stats"); n != 6 || len(fichiers(t, dir, "*avant-compaction*")) != 0 {
		t.Fatalf("le dry-run a écrit : %d lignes, sauvegardes %v", n, fichiers(t, dir, "*avant-compaction*"))
	}

	if err := compacterTitre(ctx, "halo_infinite", path, compactPassesOptions{}); err != nil {
		t.Fatalf("compaction: %v", err)
	}
	sauvegardes := fichiers(t, dir, "shared_matches_v2.halo_infinite.avant-compaction-*.duckdb")
	if len(sauvegardes) != 1 {
		t.Fatalf("sauvegardes = %v, attendu une", sauvegardes)
	}
	if n := lignes(t, sauvegardes[0], "match_bomb_stats"); n != 6 {
		t.Fatalf("la sauvegarde porte %d lignes, attendu la base d'avant (6)", n)
	}
	if n := lignes(t, path, "match_bomb_stats"); n != 3 {
		t.Fatalf("base compactée : %d lignes, attendu 3", n)
	}

	time.Sleep(1100 * time.Millisecond) // horodatage de sauvegarde à la seconde
	if err := compacterTitre(ctx, "halo_infinite", path, compactPassesOptions{}); err != nil {
		t.Fatalf("seconde passe: %v", err)
	}
	if n := lignes(t, path, "match_bomb_stats"); n != 3 {
		t.Fatalf("seconde passe : %d lignes, attendu 3 (idempotence)", n)
	}
}

// TestCompactPasses_RefusSiLeBailEstTenu : un écrivain tient la base (bail ADR 0013) — la commande
// refuse, n'écrit rien et ne sauvegarde rien.
func TestCompactPasses_RefusSiLeBailEstTenu(t *testing.T) {
	path := baseDeTest(t)
	release, err := dblease.AcquireLease(path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ancien := compactLeaseTimeout
	compactLeaseTimeout = 50 * time.Millisecond
	defer func() { compactLeaseTimeout = ancien }()

	err = compacterTitre(context.Background(), "halo_infinite", path, compactPassesOptions{})
	if err == nil || !strings.Contains(err.Error(), "bail") {
		t.Fatalf("attendu un refus sur le bail, got %v", err)
	}
	release()
	if n := lignes(t, path, "match_bomb_stats"); n != 6 {
		t.Fatalf("base touchée malgré le refus : %d lignes", n)
	}
	if s := fichiers(t, filepath.Dir(path), "*avant-compaction*"); len(s) != 0 {
		t.Fatalf("sauvegarde écrite malgré le refus : %v", s)
	}
}

// TestCompactPasses_RewriteFile : le fichier est remplacé par une recopie à l'inventaire
// identique, l'ancien est gardé, la séquence continue dans le fichier neuf.
func TestCompactPasses_RewriteFile(t *testing.T) {
	path := baseDeTest(t)
	dir := filepath.Dir(path)
	ctx := context.Background()
	if err := compacterTitre(ctx, "halo_infinite", path, compactPassesOptions{rewriteFile: true}); err != nil {
		t.Fatalf("compaction + réécriture: %v", err)
	}
	if a := fichiers(t, dir, "shared_matches_v2.halo_infinite.avant-reecriture-*.duckdb"); len(a) != 1 {
		t.Fatalf("ancien fichier gardé = %v, attendu un", a)
	}
	if _, err := os.Stat(path + ".reecriture"); !os.IsNotExist(err) {
		t.Fatalf("fichier de réécriture resté en place : %v", err)
	}
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close() //nolint:errcheck
	var n, id int64
	if err := h.SQLDb().QueryRow(`SELECT COUNT(*) FROM match_bomb_stats_latest`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("vue après réécriture : %d, %v", n, err)
	}
	if err := h.SQLDb().QueryRow(`INSERT INTO match_bomb_stats (match_id, xuid) VALUES ('m9', 'x9')
		RETURNING id`).Scan(&id); err != nil || id != 7 {
		t.Fatalf("id après réécriture = %d (attendu 7, la séquence continue), %v", id, err)
	}
}

func TestCompactPasses_OptionsIncompatibles(t *testing.T) {
	if err := runCompactPasses(nil, []string{"--dry-run", "--rewrite-file"}); err == nil {
		t.Fatal("--dry-run --rewrite-file accepté")
	}
}

func TestCopierFichier_RefuseUnWALNonVide(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.duckdb")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".wal", []byte("wal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := copierFichier(context.Background(), path, "", "halo_infinite", "avant-compaction"); err == nil {
		t.Fatal("copie acceptée avec un WAL non vide")
	}
}

// envTenirLaBase : chemin de la base que le processus auxiliaire doit tenir ouverte.
const envTenirLaBase = "LEVELUP_TEST_TENIR_BASE"

// TestAideTenirLaBase n'est pas un test : c'est le PROCESSUS AUXILIAIRE du test suivant. Lancé
// avec envTenirLaBase, il ouvre la base en écriture (comme le serveur), écrit « pret », et la tient
// jusqu'à la fermeture de son entrée standard.
func TestAideTenirLaBase(t *testing.T) {
	path := os.Getenv(envTenirLaBase)
	if path == "" {
		t.Skip("processus auxiliaire de TestCompactPasses_RefusSiUnAutreProcessusTientLaBase")
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	if _, err := db.Exec(`SELECT 1`); err != nil {
		t.Fatal(err)
	}
	fmt.Println("pret")
	_, _ = io.Copy(io.Discard, os.Stdin) // tient la base jusqu'à la fermeture de stdin
}

// TestCompactPasses_RefusSiUnAutreProcessusTientLaBase : un AUTRE PROCESSUS tient la base en
// écriture (ce que fait le serveur) — la commande refuse, en dry-run comme pour de bon, sans rien
// écrire ni sauvegarder.
func TestCompactPasses_RefusSiUnAutreProcessusTientLaBase(t *testing.T) {
	path := baseDeTest(t)
	liberer := tenirLaBase(t, path)
	for _, o := range []compactPassesOptions{{dryRun: true}, {}} {
		err := compacterTitre(context.Background(), "halo_infinite", path, o)
		if err == nil || !strings.Contains(err.Error(), "refus") {
			t.Errorf("options %+v : attendu un refus, got %v", o, err)
		}
	}
	liberer()
	if n := lignes(t, path, "match_bomb_stats"); n != 6 {
		t.Fatalf("base touchée malgré le refus : %d lignes", n)
	}
	if s := fichiers(t, filepath.Dir(path), "*avant-compaction*"); len(s) != 0 {
		t.Fatalf("sauvegarde écrite malgré le refus : %v", s)
	}
}

// tenirLaBase lance le processus auxiliaire (TestAideTenirLaBase) qui tient `path` ouverte en
// écriture ; la fonction rendue le libère et attend sa sortie.
func tenirLaBase(t *testing.T, path string) func() {
	t.Helper()
	aux := exec.Command(os.Args[0], "-test.run=^TestAideTenirLaBase$")
	aux.Env = append(os.Environ(), envTenirLaBase+"="+path)
	entree, err := aux.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	sortie, err := aux.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := aux.Start(); err != nil {
		t.Fatal(err)
	}
	if ligne, err := bufio.NewReader(sortie).ReadString('\n'); err != nil || strings.TrimSpace(ligne) != "pret" {
		t.Fatalf("processus auxiliaire : %q, %v", ligne, err)
	}
	return func() {
		if err := entree.Close(); err != nil {
			t.Errorf("processus auxiliaire, stdin : %v", err)
		}
		if err := aux.Wait(); err != nil {
			t.Errorf("processus auxiliaire : %v", err)
		}
	}
}

// TestCompactPasses_OrphelinRefuseAvantTouteSauvegarde : une table du registre absente et sa
// `__compact` orpheline — la commande refuse AVANT de sauvegarder (sinon elle copierait l'état
// déjà orphelin et masquerait la bonne sauvegarde), en dry-run comme pour de bon, et le message
// renvoie à la sauvegarde antérieure à l'interruption.
func TestCompactPasses_OrphelinRefuseAvantTouteSauvegarde(t *testing.T) {
	path := baseDeTest(t)
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE match_bomb_stats__compact AS SELECT * FROM match_bomb_stats`,
		`DROP TABLE match_bomb_stats`,
	} {
		if _, err := h.SQLDb().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	for _, o := range []compactPassesOptions{{}, {dryRun: true}, {rewriteFile: true}} {
		err := compacterTitre(context.Background(), "halo_infinite", path, o)
		if s := fichiers(t, filepath.Dir(path), "*avant-*"); len(s) != 0 {
			t.Fatalf("options %+v : sauvegarde prise malgré l'orphelin : %v", o, s)
		}
		if err == nil || !strings.Contains(err.Error(), "orpheline") ||
			!strings.Contains(err.Error(), "ANTÉRIEURE") {
			t.Fatalf("options %+v : attendu le refus de l'orpheline, got %v", o, err)
		}
	}
}
