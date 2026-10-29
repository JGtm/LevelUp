package ops

// seed_demo_replay_corpus_test.go — le match de chaque rejeu figé fait partie du corpus de la
// démo, même quand aucune des quatre voies (solo, escouade, classé, médias) ne le retient : ses
// données de match, son roster et son anonymisation en dépendent.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

func TestResolveDemoCorpusAndRoster_LeMatchDUnRejeuFigeEntreDansLeCorpus(t *testing.T) {
	opts := fixtureCorpusCampagne(t)
	opts.TitleSlug = titlePkg.DefaultSlug
	opts.SourceLabel = "JGtm"
	opts.RepoRoot = t.TempDir()
	const (
		replayOnly = "replayOnly" // hors de toute voie du corpus
		viaSolo    = "arena1"     // déjà au corpus par la voie solo : pas de doublon
		realMate   = "2533274000000099"
	)
	execAll(t, opts.SourceSharedDB, []string{
		`INSERT INTO match_registry VALUES ('` + replayOnly + `', '2026-09-20 10:00:00', '2026-09-20 10:00:00+00',
			'aaaaaaaa-0000-0000-0000-000000000001', FALSE, 'pl', 'pl')`,
		`INSERT INTO match_participants VALUES ('` + replayOnly + `', '` + campagneOpsXUID + `'),
			('` + replayOnly + `', '` + realMate + `')`,
	})
	mp := titlePkg.NewPathResolver(opts.RepoRoot).DemoManifestPath(opts.SourceLabel, opts.TitleSlug)
	if err := writeDemoManifest(mp, &DemoManifest{
		Version: demoManifestVersion, TitleSlug: opts.TitleSlug,
		Corpus: DemoManifestCorpus{
			SoloMatchIDs: []string{"arena1"},
			// Le même match sous deux familles : une seule fois au corpus.
			ReplayMatches: []DemoReplayPick{
				{MatchID: replayOnly, Mode: "ctf"}, {MatchID: replayOnly, Mode: demoModeVehicles},
				{MatchID: viaSolo, Mode: "slayer"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	sel, err := resolveDemoCorpusAndRoster(context.Background(), opts)
	if err != nil {
		t.Fatalf("resolveDemoCorpusAndRoster : %v", err)
	}
	if want := []string{"arena1", replayOnly}; len(sel.matchIDs) != 2 || sel.matchIDs[0] != want[0] || sel.matchIDs[1] != want[1] {
		t.Fatalf("corpus = %v, attendu %v (rejeux figés dedoublonnes)", sel.matchIDs, want)
	}
	// Le roster est dérivé du corpus : le coéquipier réel du match de rejeu est anonymisé.
	found := false
	for _, e := range sel.roster {
		found = found || e.SourceXUID == realMate
	}
	if !found {
		t.Errorf("le participant du match de rejeu n'est pas au roster (fuite d'identité) : %+v", sel.roster)
	}
}

// Un match qui représente deux familles (« véhicules » et son mode) est rangé, compté et indexé
// une seule fois.
func TestSeedDemoReplays_UnMatchSousDeuxFamillesEstIndexeUneFois(t *testing.T) {
	src, demo := t.TempDir(), t.TempDir()
	slug := titlePkg.DefaultSlug
	pr := titlePkg.NewPathResolver(src)
	layout := titlePkg.NewDemoLayout(demo)
	const id = "aaaa0001"
	ecrireTest(t, pr.ReplayArtifactPath(slug, id), artefactJSON(t, id, replay.SchemaVersion))
	ecrireTest(t, filmcache.ManifestPath(pr.CacheRootDir(), id), []byte(`{"chunks":[{"index":0,"chunk_type":1}]}`))
	ecrireTest(t, filepath.Join(filmcache.ChunkDir(pr.CacheRootDir(), id), "chunk_00.bin"), []byte("film"))

	rep := seedDemoReplays(context.Background(), SeedDemoOptions{RepoRoot: src, TitleSlug: slug}, layout,
		[]DemoReplayPick{{MatchID: id, Mode: "ctf"}, {MatchID: id, Mode: demoModeVehicles}}, nil)

	if rep.Films != 1 {
		t.Errorf("films embarqués : %d, attendu 1", rep.Films)
	}
	var index domain.DemoReplayIndex
	raw, err := os.ReadFile(layout.ReplayIndexPath(slug))
	if err != nil || json.Unmarshal(raw, &index) != nil {
		t.Fatalf("index illisible : %v", err)
	}
	if len(index.Matches) != 1 || index.Matches[0].MatchID != id {
		t.Errorf("index = %+v, attendu une seule entrée pour %s", index.Matches, id)
	}
}
