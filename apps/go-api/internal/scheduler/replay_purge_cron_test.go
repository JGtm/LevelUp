package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// prepareReplayPurgeFixture construit une shared DB minimale (match_registry avec les
// deux colonnes de temps du fragment canonique) et un dossier d'artefacts.
func prepareReplayPurgeFixture(t *testing.T) (sharedPath, artifactsDir string) {
	t.Helper()
	root := t.TempDir()
	sharedPath = filepath.Join(root, "shared_matches_v2.duckdb")
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
	vieux := time.Now().UTC().AddDate(0, -8, 0).Format("2006-01-02 15:04:05")
	recent := time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	for _, row := range [][2]string{
		{"aaaa0001-1111-4abc-9def-000000000001", vieux},
		{"bbbb0002-1111-4abc-9def-000000000002", recent},
	} {
		if _, err := sqlDB.Exec(
			`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?::TIMESTAMPTZ)`,
			row[0], row[1]); err != nil {
			t.Fatalf("insert %s: %v", row[0], err)
		}
	}

	artifactsDir = filepath.Join(root, "replays", "halo_infinite")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"aaaa0001.json", "bbbb0002.json", "cccc0003.json"} {
		if err := os.WriteFile(filepath.Join(artifactsDir, name), []byte(`{"schemaVersion":3}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return sharedPath, artifactsDir
}

// TestPurgeReplayArtifacts — le contrat de la purge : SEUL l'artefact d'un match plus
// vieux que la fenêtre part ; le récent reste ; l'indatable (sans ligne de registre)
// n'est JAMAIS détruit.
func TestPurgeReplayArtifacts(t *testing.T) {
	sharedPath, artifactsDir := prepareReplayPurgeFixture(t)
	cutoff := time.Now().UTC().AddDate(0, -6, 0)

	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, artifactsDir, cutoff)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 || kept != 1 || unknown != 1 {
		t.Errorf("purge = (purged %d, kept %d, unknown %d), attendu (1, 1, 1)", purged, kept, unknown)
	}
	if _, err := os.Stat(filepath.Join(artifactsDir, "aaaa0001.json")); !os.IsNotExist(err) {
		t.Error("l'artefact du match de 8 mois devait être purgé")
	}
	for _, garde := range []string{"bbbb0002.json", "cccc0003.json"} {
		if _, err := os.Stat(filepath.Join(artifactsDir, garde)); err != nil {
			t.Errorf("%s ne devait PAS être purgé : %v", garde, err)
		}
	}
}

// TestPurgeReplayArtifacts_DossierAbsent — aucun artefact construit = cycle nominal.
func TestPurgeReplayArtifacts_DossierAbsent(t *testing.T) {
	sharedPath, _ := prepareReplayPurgeFixture(t)
	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, filepath.Join(t.TempDir(), "inexistant"), time.Now())
	if err != nil || purged != 0 || kept != 0 || unknown != 0 {
		t.Errorf("dossier absent = (%d, %d, %d, %v), attendu (0, 0, 0, nil)", purged, kept, unknown, err)
	}
}

// TestPurgeReplayArtifacts_MarquesDeDerivation — constat C6 de la revue A-R1.
//
// Les dérivations posent `<short8>.derived.json` À CÔTÉ de l'artefact, dans le MÊME dossier.
// Le cron balayait tout ce qui finit par `.json` : chaque marque donnait
// `short = "<short8>.derived"`, absent du registre, donc `unknown++` — une ligne INFO à CHAQUE
// passage dès qu'une dérivation existait, et une marque qui SURVIVAIT indéfiniment à la purge
// de son propre artefact.
//
// Les deux moitiés du contrat : la marque ne compte dans aucune catégorie, et elle part AVEC
// l'artefact qu'elle décrit.
func TestPurgeReplayArtifacts_MarquesDeDerivation(t *testing.T) {
	sharedPath, artifactsDir := prepareReplayPurgeFixture(t)
	for _, court := range []string{"aaaa0001", "bbbb0002", "cccc0003"} {
		if err := replaybuild.WriteDerivationsMark(
			filepath.Join(artifactsDir, court+".json"), 39, 42); err != nil {
			t.Fatalf("marque %s: %v", court, err)
		}
	}
	cutoff := time.Now().UTC().AddDate(0, -6, 0)

	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, artifactsDir, cutoff)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	// EXACTEMENT le même bilan que sans marque : elles ne sont ni purgées, ni conservées, ni
	// indatables — elles ne sont pas des artefacts.
	if purged != 1 || kept != 1 || unknown != 1 {
		t.Errorf("purge = (purged %d, kept %d, unknown %d), attendu (1, 1, 1) — les marques de "+
			"dérivation sont comptées comme des artefacts indatables (constat C6)",
			purged, kept, unknown)
	}
	// La marque de l'artefact purgé part avec lui.
	if _, err := os.Stat(filepath.Join(artifactsDir, "aaaa0001.derived.json")); !os.IsNotExist(err) {
		t.Error("la marque de l'artefact purgé a survécu — elle décrirait les dérivations d'un " +
			"fichier qui n'existe plus")
	}
	// Celles des artefacts conservés restent : sinon le rattrapage rejouerait leurs dérivations.
	for _, garde := range []string{"bbbb0002.derived.json", "cccc0003.derived.json"} {
		if _, err := os.Stat(filepath.Join(artifactsDir, garde)); err != nil {
			t.Errorf("%s ne devait PAS être supprimée : %v", garde, err)
		}
	}
}

// TestPurgeReplayArtifacts_MarqueOrpheline — constat N4 de la revue A-R2.
//
// Une marque dont l'artefact n'existe plus n'était ramassée par personne : le `continue` posé
// pour C6 intervient AVANT toute datation, donc une marque n'est plus jamais examinée POUR
// ELLE-MÊME. Le cas est réel sur toute instance dont le cron a purgé des artefacts avant que la
// suppression de la marque n'existe, et après toute suppression manuelle par un opérateur.
//
// Litière disque sans effet fonctionnel (`DerivationsUpToDate` fait un `os.Stat` de l'artefact
// d'abord), mais que rien n'aurait jamais enlevée.
func TestPurgeReplayArtifacts_MarqueOrpheline(t *testing.T) {
	sharedPath, artifactsDir := prepareReplayPurgeFixture(t)
	// Une marque SEULE : son artefact a disparu avant ce passage.
	orpheline := filepath.Join(artifactsDir, "dddd0004"+replaybuild.SuffixeMarqueDerivations)
	if err := os.WriteFile(orpheline, []byte(`{"rev":"derivations-2026-09-06"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Et une marque ACCOMPAGNÉE, sur un artefact conservé : elle, doit rester.
	if err := replaybuild.WriteDerivationsMark(
		filepath.Join(artifactsDir, "bbbb0002.json"), 39, 42); err != nil {
		t.Fatalf("marque bbbb0002: %v", err)
	}
	cutoff := time.Now().UTC().AddDate(0, -6, 0)

	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, artifactsDir, cutoff)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	// Le bilan des ARTEFACTS ne bouge pas : une marque n'en est pas un, orpheline ou non.
	if purged != 1 || kept != 1 || unknown != 1 {
		t.Errorf("purge = (purged %d, kept %d, unknown %d), attendu (1, 1, 1)", purged, kept, unknown)
	}
	if _, err := os.Stat(orpheline); !os.IsNotExist(err) {
		t.Error("la marque orpheline est toujours là — aucune reprise ne l'enlèvera jamais " +
			"(constat N4)")
	}
	if _, err := os.Stat(filepath.Join(artifactsDir, "bbbb0002"+replaybuild.SuffixeMarqueDerivations)); err != nil {
		t.Errorf("la marque d'un artefact CONSERVÉ a été supprimée : %v — le rattrapage "+
			"rejouerait ses dérivations", err)
	}
}
