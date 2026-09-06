package scheduler

// replay_purge_cron_runonce_test.go — LE CYCLE COMPLET DU CRON QUI SUPPRIME DES FICHIERS.
//
// POURQUOI CES TESTS EXISTENT (registre d'audit du 2026-09-05, constat G7). `RunOnce` était
// le SEUL cron du dépôt qui supprime des fichiers, et le seul sans aucun test — sa godoc
// disait pourtant « exporté pour les tests ». Les deux tests voisins
// (replay_purge_cron_test.go) ne couvrent que `purgeReplayArtifactsForTitle`, c'est-à-dire
// la purge d'UN titre avec un `cutoff` DÉJÀ calculé : ni la garde `months <= 0`, ni le
// calcul du seuil, ni la boucle sur les titres actifs, ni le PÉRIMÈTRE des suppressions
// n'étaient franchis. Inverser la garde faisait tomber le seuil sur l'instant courant et
// purgeait tout le parc au premier tick, sans qu'aucun test ne rougisse.
//
// L'HORLOGE EST INJECTÉE (champ `now`, nil en production) : la frontière de la fenêtre ne
// se prouve qu'avec un seuil connu à la seconde près. Sans elle, un test ne peut que dater
// des matchs très loin de part et d'autre du seuil — ce qui laisse passer un décalage d'un
// mois ou une comparaison rendue non stricte.

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// horlogeFixe : l'instant de référence des cycles testés. Une date FIXE, jamais
// `time.Now()` — sinon l'attendu bouge avec le calendrier.
var horlogeFixe = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

// runOnceFixture décrit le dépôt temporaire d'un cycle : où sont les artefacts, et tout ce
// qu'aucun cycle ne doit jamais toucher.
type runOnceFixture struct {
	repoRoot     string
	artifactsDir string
	leurres      []string // chemins ABSOLUS hors périmètre
}

// prepareRunOnceFixture monte un dépôt complet : la shared du titre par défaut avec trois
// matchs calés sur la FRONTIÈRE de la fenêtre, quatre artefacts (dont un indatable), et
// cinq leurres hors périmètre. Le seuil vaut `horlogeFixe` moins `mois`.
func prepareRunOnceFixture(t *testing.T, mois int) runOnceFixture {
	t.Helper()
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	cutoff := horlogeFixe.AddDate(0, -mois, 0)
	ecrireRegistre(t, sharedPath, map[string]time.Time{
		"aaaa0001-1111-4abc-9def-000000000001": cutoff.Add(-time.Second), // AVANT le seuil -> purgé
		"bbbb0002-1111-4abc-9def-000000000002": cutoff,                   // SUR le seuil -> gardé
		"cccc0003-1111-4abc-9def-000000000003": cutoff.Add(time.Second),  // APRÈS le seuil -> gardé
		"eeee0005-1111-4abc-9def-000000000005": cutoff.Add(-time.Second), // purgeable, sert de leurre
	})

	artifactsDir := pr.ReplayArtifactsDir(titlePkg.DefaultSlug)
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// dddd0004 n'a AUCUNE ligne de registre : indatable, donc jamais détruit.
	for _, n := range []string{"aaaa0001.json", "bbbb0002.json", "cccc0003.json", "dddd0004.json"} {
		ecrireFichier(t, filepath.Join(artifactsDir, n))
	}

	// LES LEURRES, ET POURQUOI CEUX-LÀ. Un leurre ne vaut que s'il tombe DANS le viseur :
	// nommé comme un artefact PURGEABLE (`aaaa0001` / `eeee0005`, tous deux datés avant le
	// seuil) et posé là où une garde du code l'épargne. Une première version de ce fichier
	// employait `aaaa0001.txt` et un sous-dossier `sous-dossier/` : retirer le filtre
	// d'extension ou le test `e.IsDir()` ne les emportait PAS (le nom tronqué, « aaaa0001.txt »
	// et « sous-dossier », n'est dans aucun registre) — deux leurres qui ne prouvaient rien.
	// Corrigés ici : un fichier SANS extension (le tronquage le laisse intact, donc datable
	// et purgeable — seul le filtre `.json` le sauve) et un RÉPERTOIRE VIDE nommé comme un
	// artefact purgeable (seul `e.IsDir()` le sauve ; vide, `os.Remove` réussirait).
	// Les trois autres couvrent le voisinage : les films, IRREMPLAÇABLES (en-tête du cron),
	// et le dossier PARENT du titre.
	cache := filepath.Join(repoRoot, "data", "cache")
	if err := os.MkdirAll(filepath.Join(artifactsDir, "eeee0005.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	leurres := []string{
		filepath.Join(cache, "film_chunks", "aaaa0001", "chunk_00.bin"),
		filepath.Join(cache, "film_manifests", "aaaa0001.json"),
		filepath.Join(cache, "replays", "aaaa0001.json"), // dossier PARENT du titre
		filepath.Join(artifactsDir, "aaaa0001"),          // sans extension : seul le filtre `.json` le sauve
		filepath.Join(artifactsDir, "eeee0005.json"),     // RÉPERTOIRE vide : seul `e.IsDir()` le sauve
	}
	for _, p := range leurres {
		if p == filepath.Join(artifactsDir, "eeee0005.json") {
			continue // déjà créé, et c'est un répertoire
		}
		ecrireFichier(t, p)
	}
	return runOnceFixture{repoRoot: repoRoot, artifactsDir: artifactsDir, leurres: leurres}
}

// ecrireRegistre crée la shared minimale (les deux colonnes du fragment temporel canonique)
// et y insère les matchs datés fournis.
func ecrireRegistre(t *testing.T, sharedPath string, matchs map[string]time.Time) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared RW: %v", err)
	}
	defer func() { _ = db.Close() }()
	sqlDB := db.SQLDb()
	if _, err := sqlDB.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	for id, at := range matchs {
		// RFC3339 AVEC son `Z`, jamais « 2006-01-02 15:04:05 » : une chaîne sans décalage
		// castée en TIMESTAMPTZ est lue dans le fuseau de SESSION de DuckDB (UTC+2 ici),
		// ce qui déplace l'instant stocké de deux heures — assez pour faire basculer du
		// mauvais côté du seuil les deux matchs posés dessus. Ce test l'a attrapé.
		if _, err := sqlDB.Exec(
			`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?::TIMESTAMPTZ)`,
			id, at.UTC().Format(time.RFC3339)); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
}

// ecrireFichier écrit un fichier témoin (et les dossiers qui lui manquent).
func ecrireFichier(t *testing.T, chemin string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(`{"schemaVersion":39}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cronDeTest monte le cron sur le dépôt temporaire, horloge fixée.
func cronDeTest(fx runOnceFixture, mois int) *ReplayPurgeCron {
	c := NewReplayPurgeCron(fx.repoRoot, func() int { return mois }, time.Hour)
	c.now = func() time.Time { return horlogeFixe }
	return c
}

// assertLeurresIntacts : aucun chemin hors périmètre n'a bougé.
func assertLeurresIntacts(t *testing.T, fx runOnceFixture) {
	t.Helper()
	for _, p := range fx.leurres {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("HORS PÉRIMÈTRE détruit par la purge : %s (%v)", p, err)
		}
	}
}

// presents rend les artefacts `.json` restants à la racine du dossier d'artefacts, triés.
func presents(t *testing.T, artifactsDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		t.Fatalf("lecture du dossier d'artefacts: %v", err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// TestRunOnce_SelectionParAge — LA FRONTIÈRE. Horloge fixée, fenêtre de 6 mois : seul
// l'artefact du match d'UNE SECONDE plus vieux que le seuil part. Celui qui tombe
// EXACTEMENT sur le seuil reste (la comparaison est stricte, `!at.Before(cutoff)`), celui
// d'une seconde plus récent aussi, et l'indatable n'est jamais détruit.
func TestRunOnce_SelectionParAge(t *testing.T) {
	fx := prepareRunOnceFixture(t, 6)
	cronDeTest(fx, 6).RunOnce(context.Background())

	got := presents(t, fx.artifactsDir)
	want := []string{"bbbb0002.json", "cccc0003.json", "dddd0004.json"}
	if !slices.Equal(got, want) {
		t.Errorf("artefacts restants %v, attendu %v", got, want)
	}
	assertLeurresIntacts(t, fx)
}

// TestRunOnce_FenetreIllimitee — LA GARDE `months <= 0`. Zéro et négatif veulent dire
// « rétention illimitée » : le cycle est un no-op NOMINAL. Inverser cette garde ferait
// tomber le seuil sur l'instant courant et purgerait TOUT le parc au premier tick — c'est
// le mode de panne que ce test interdit.
func TestRunOnce_FenetreIllimitee(t *testing.T) {
	for _, mois := range []int{0, -1, -12} {
		fx := prepareRunOnceFixture(t, 6) // dépôt daté comme au test précédent
		cronDeTest(fx, mois).RunOnce(context.Background())

		got := presents(t, fx.artifactsDir)
		want := []string{"aaaa0001.json", "bbbb0002.json", "cccc0003.json", "dddd0004.json"}
		if !slices.Equal(got, want) {
			t.Errorf("rétention %d mois : artefacts restants %v, attendu les 4 (aucune suppression)",
				mois, got)
		}
		assertLeurresIntacts(t, fx)
	}
}

// TestRunOnce_SansRetention — un cron sans lecteur de réglage ne supprime rien et ne
// panique pas : c'est le contrat écrit de `NewReplayPurgeCron(..., nil, ...)`.
func TestRunOnce_SansRetention(t *testing.T) {
	fx := prepareRunOnceFixture(t, 6)
	NewReplayPurgeCron(fx.repoRoot, nil, time.Hour).RunOnce(context.Background())

	if got := len(presents(t, fx.artifactsDir)); got != 4 {
		t.Errorf("%d artefacts restants, attendu 4 (retention nil = noop)", got)
	}
	assertLeurresIntacts(t, fx)
}
