// Package ops - seed_demo_campaign_corpus_test.go : E6.2a-F2 / E6.2b-C2. Le corpus demo rendu par
// resolveDemoCorpus (manifeste figé ou sélection dynamique) ne porte aucun match de Campagne, quelle que soit la voie qui l'a
// retenu (solo, escouade, classe, media). Halo 5 masque la Campagne ; Halo Infinite
// (aucune variante masquee) reste neutre.
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
)

func execAll(t *testing.T, path string, stmts []string) {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture : %v\n%s", err, s)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

// Fixture : arena1 (solo), campSolo, campSquad (voie escouade), campClip (ancre par un clip),
// arenaClip (ancre par un clip) ; seuls campSolo/campSquad/campClip portent la variante Campagne.
func fixtureCorpusCampagne(t *testing.T) SeedDemoOptions {
	t.Helper()
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared_matches_v2.duckdb")
	player := filepath.Join(dir, "stats.duckdb")
	social := filepath.Join(dir, "shared_social.duckdb")
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	arena := "aaaaaaaa-0000-0000-0000-000000000001"
	reg := func(id, day, variant string) string {
		return `('` + id + `', '2026-09-` + day + ` 10:00:00', '2026-09-` + day + ` 10:00:00+00', '` + variant + `', FALSE, 'pl', 'pl')`
	}
	execAll(t, shared, []string{
		`CREATE TABLE match_registry (match_id VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			game_variant_id VARCHAR, is_ranked BOOLEAN, playlist_name VARCHAR, pair_name VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`,
		`INSERT INTO match_registry VALUES ` + reg("arena1", "01", arena) + `, ` + reg("campSolo", "02", camp) + `, ` +
			reg("campSquad", "03", camp) + `, ` + reg("campClip", "04", camp) + `, ` + reg("arenaClip", "05", arena),
		`INSERT INTO match_participants VALUES ('arena1', '` + campagneOpsXUID + `'), ('campSolo', '` + campagneOpsXUID + `')`,
	})
	execAll(t, player, []string{
		`CREATE TABLE player_match_enrichment_latest (match_id VARCHAR, session_id VARCHAR, is_with_friends BOOLEAN)`,
		`INSERT INTO player_match_enrichment_latest VALUES ('campSquad', 's1', TRUE)`,
	})
	execAll(t, social, []string{
		`CREATE TABLE media_files (id VARCHAR, file_name VARCHAR, file_path VARCHAR, thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ)`,
		`CREATE TABLE media_match_associations_latest (media_file_id VARCHAR, match_id VARCHAR)`,
		`INSERT INTO media_files VALUES ('f1', 'Halo_5_Guardians-a.mp4', 'a.mp4', '', '2026-09-04 10:00:00+00'),
			('f2', 'Halo_5_Guardians-b.mp4', 'b.mp4', '', '2026-09-05 10:00:00+00')`,
		`INSERT INTO media_match_associations_latest VALUES ('f1', 'campClip'), ('f2', 'arenaClip')`,
	})
	return SeedDemoOptions{
		SourcePlayerDB: player, SourceSharedDB: shared, SourceXUID: campagneOpsXUID,
		MaxMatches: 10, MaxMedia: 10, IncludeMedia: true,
	}
}

func assertCorpus(t *testing.T, label string, dc dynamicCorpus, want []string) {
	t.Helper()
	all := unionMatchIDs(dc.solo, dc.squad, dc.ranked, dc.media)
	sort.Strings(all)
	if len(all) != len(want) {
		t.Fatalf("%s : corpus %v, attendu %v", label, all, want)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Fatalf("%s : corpus %v, attendu %v", label, all, want)
		}
	}
}

func attenduCorpus(masquee bool) []string {
	if masquee {
		return []string{"arena1", "arenaClip"}
	}
	return []string{"arena1", "arenaClip", "campClip", "campSolo", "campSquad"}
}

// Chemin dynamique : la Campagne n'entre par aucune voie.
func TestResolveDemoCorpus_Dynamique_ExclutLaCampagneSurToutesLesVoies(t *testing.T) {
	for _, tc := range campagneOpsTitres {
		opts := fixtureCorpusCampagne(t)
		opts.TitleSlug = tc.slug
		dc, manifest, err := resolveDemoCorpus(context.Background(), opts)
		if err != nil || manifest != nil {
			t.Fatalf("%s : manifest=%v err=%v", tc.slug, manifest, err)
		}
		assertCorpus(t, tc.slug, dc, attenduCorpus(tc.masquee))
	}
}

// Chemin du manifeste figé (defaut de la demo) : un manifeste qui liste un match de Campagne
// n'en laisse rien passer, par le MEME point d'application que le chemin dynamique.
func TestResolveDemoCorpus_ManifesteFige_ExclutLaCampagne(t *testing.T) {
	for _, tc := range campagneOpsTitres {
		opts := fixtureCorpusCampagne(t)
		opts.TitleSlug = tc.slug
		opts.SourceLabel = "JGtm"
		opts.RepoRoot = t.TempDir()
		mp := titlePkg.NewPathResolver(opts.RepoRoot).DemoManifestPath(opts.SourceLabel, tc.slug)
		err := writeDemoManifest(mp, &DemoManifest{
			Version: demoManifestVersion, TitleSlug: tc.slug,
			Corpus: DemoManifestCorpus{
				SoloMatchIDs: []string{"arena1", "campSolo"}, SquadMatchIDs: []string{"campSquad"},
				MediaMatchIDs: []string{"campClip", "arenaClip"},
			},
		})
		if err != nil {
			t.Fatalf("%s : ecriture du manifeste : %v", tc.slug, err)
		}
		dc, manifest, err := resolveDemoCorpus(context.Background(), opts)
		if err != nil || manifest == nil {
			t.Fatalf("%s : manifeste attendu : manifest=%v err=%v", tc.slug, manifest, err)
		}
		assertCorpus(t, tc.slug, dc, attenduCorpus(tc.masquee))
	}
}
