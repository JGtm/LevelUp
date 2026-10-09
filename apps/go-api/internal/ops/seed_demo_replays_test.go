package ops

// seed_demo_replays_test.go — rejeux figés de la démo : choix d'un match par famille de mode,
// installation des artefacts (copie, conservation, périmé, absent), films embarqués, index et
// élagage. La recuisson elle-même (enfant borné) est couverte par replaychild/filmproc ; ici
// aucun film n'est décodé.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func TestDemoReplayFamilyOf(t *testing.T) {
	cases := map[string]string{
		"CTF:Arena":                       "ctf",
		"CTF:Arena Neutral Flag":          "ctf",
		"BTB:Fiesta CTF":                  "fiesta",
		"Slayer:Arena Super Fiesta":       "fiesta",
		"Team Slayer:Arena":               "slayer",
		"Arena:Team Snipers":              "slayer",
		"Strongholds:Arena":               "strongholds",
		"Ranked:King of the Hill":         "koth",
		"KOTH:Arena":                      "koth",
		"Oddball:Arena":                   "oddball",
		"Assault:Neutral Bomb":            "assault",
		"Husky Raid:Assault":              "assault",
		"BTB:Total Control":               "total_control",
		"BTB:Extraction":                  "extraction",
		"Arena:VIP":                       "vip",
		"BTB:Escalation Slayer":           "escalation",
		"Firefight:Battle of the Academy": "firefight",
		"TFF | Survive The Undead":        "",
	}
	for variant, want := range cases {
		if got := demoReplayFamilyOf(variant); got != want {
			t.Errorf("%q : famille %q, attendu %q", variant, got, want)
		}
	}
}

func TestPickDemoReplays_UnParFamilleDeterministe(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	cands := []demoReplayCandidate{
		{matchID: "c-recent", variant: "CTF:Arena", mapName: "Aquarius", start: t0.Add(48 * time.Hour)},
		{matchID: "c-ajour", variant: "CTF:Arena", mapName: "Catalyst", start: t0, upToDate: true},
		{matchID: "s-b", variant: "Slayer:Arena", start: t0},
		{matchID: "s-a", variant: "Team Slayer:Arena", start: t0},
		{matchID: "x", variant: "CASTLE WARS", start: t0},
	}
	picks, unclassified := pickDemoReplays(cands)
	want := []DemoReplayPick{
		{MatchID: "c-ajour", Mode: "ctf", Map: "Catalyst"}, // artefact à jour d'abord
		{MatchID: "s-a", Mode: "slayer"},                   // égalité de date : plus petit match_id
	}
	if len(picks) != len(want) {
		t.Fatalf("choix %+v, attendu %+v", picks, want)
	}
	for i := range want {
		if picks[i] != want[i] {
			t.Errorf("choix %d = %+v, attendu %+v", i, picks[i], want[i])
		}
	}
	if len(unclassified) != 1 || unclassified[0] != "CASTLE WARS" {
		t.Errorf("variantes écartées %v, attendu [CASTLE WARS]", unclassified)
	}
}

func TestValidateDemoManifest_RejeuxFiges(t *testing.T) {
	base := DemoManifest{Version: demoManifestVersion, Corpus: DemoManifestCorpus{SoloMatchIDs: []string{"m1"}}}
	ok := base
	ok.Corpus.ReplayMatches = []DemoReplayPick{{MatchID: "r1", Mode: "ctf"}}
	if err := validateDemoManifest(&ok); err != nil {
		t.Errorf("manifeste valide refusé : %v", err)
	}
	if got := ok.CorpusMatchIDs(); len(got) != 2 || got[1] != "r1" {
		t.Errorf("le rejeu figé doit entrer dans le corpus : %v", got)
	}
	for _, bad := range [][]DemoReplayPick{
		{{MatchID: "r1"}},
		{{MatchID: "r1", Mode: "ctf"}, {MatchID: "r2", Mode: "ctf"}},
	} {
		m := base
		m.Corpus.ReplayMatches = bad
		if err := validateDemoManifest(&m); err == nil {
			t.Errorf("manifeste invalide accepté : %+v", bad)
		}
	}
}

// artefactJSON : un artefact minimal au schéma donné, décodé par les révisions courantes.
func artefactJSON(t *testing.T, matchID string, schema int) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"schemaVersion": schema, "matchId": matchID, "titleSlug": titlePkg.DefaultSlug,
		"tracks": []map[string]any{{"slot": 1, "points": []any{}}},
		"layers": replay.RevisionsCourantesDesCouches(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func ecrireTest(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSeedDemoReplays_InstalleIndexeEtElague(t *testing.T) {
	src, demo := t.TempDir(), t.TempDir()
	slug := titlePkg.DefaultSlug
	pr := titlePkg.NewPathResolver(src)
	layout := titlePkg.NewDemoLayout(demo)
	const ajour, conserve, perime, absent, orphelin = "aaaa0001", "bbbb0002", "cccc0003", "dddd0004", "eeee0005"

	ecrireTest(t, pr.ReplayArtifactPath(slug, ajour), artefactJSON(t, ajour, replay.SchemaVersion))
	ecrireTest(t, layout.ReplayArtifactPath(slug, conserve), artefactJSON(t, conserve, replay.SchemaVersion))
	// La démo s'écrit EN PLACE ici (PreviousOutDir vide) : sa propre copie est la démo
	// précédente, d'où « conserve ».
	ecrireTest(t, pr.ReplayArtifactPath(slug, perime), artefactJSON(t, perime, replay.SchemaVersion-1))
	ecrireTest(t, layout.ReplayArtifactPath(slug, orphelin), artefactJSON(t, orphelin, replay.SchemaVersion))
	// Le film du rejeu à jour, dans le cache source : il doit être embarqué.
	ecrireTest(t, filmcache.ManifestPath(pr.CacheRootDir(), ajour), []byte(`{"chunks":[{"index":0,"chunk_type":1}]}`))
	ecrireTest(t, filepath.Join(filmcache.ChunkDir(pr.CacheRootDir(), ajour), "chunk_00.bin"), []byte("film"))

	opts := SeedDemoOptions{RepoRoot: src, TitleSlug: slug}
	picks := []DemoReplayPick{
		{MatchID: ajour, Mode: "ctf", Map: "Aquarius"}, {MatchID: conserve, Mode: "slayer"},
		{MatchID: perime, Mode: "koth"}, {MatchID: absent, Mode: "oddball"},
	}
	roster := []demoRosterEntry{{SourceXUID: "2533274800000001", DemoXUID: "0000000000000000", DemoGamertag: "DemoPlayer"}}
	rep := seedDemoReplays(context.Background(), opts, layout, picks, roster)

	want := map[string]string{ajour: DemoReplayCopied, conserve: DemoReplayKept, perime: DemoReplayStale, absent: DemoReplayMissing}
	for id, w := range want {
		if rep.Outcomes[id] != w {
			t.Errorf("%s : issue %q, attendu %q", id, rep.Outcomes[id], w)
		}
	}
	if rep.Films != 1 {
		t.Errorf("films embarqués : %d, attendu 1", rep.Films)
	}
	if _, found, err := filmcache.Open(pr.DemoFilmsCacheRoot(slug), ajour); !found || err != nil {
		t.Errorf("film du rejeu non embarqué (found=%v, err=%v)", found, err)
	}
	if fileExists(layout.ReplayArtifactPath(slug, orphelin)) {
		t.Error("artefact hors manifeste non retiré")
	}

	var index domain.DemoReplayIndex
	raw, err := os.ReadFile(layout.ReplayIndexPath(slug))
	if err != nil || json.Unmarshal(raw, &index) != nil {
		t.Fatalf("index illisible : %v", err)
	}
	servis := map[string]bool{}
	for _, m := range index.Matches {
		servis[m.MatchID] = true
	}
	if !servis[ajour] || !servis[conserve] || !servis[perime] || servis[absent] {
		t.Errorf("matchs servis %v : attendu les trois rejeux avec artefact, pas l'absent", servis)
	}
	if len(index.Identities) != 1 || index.Identities[0].DemoGamertag != "DemoPlayer" {
		t.Errorf("identités de l'index : %+v", index.Identities)
	}
}

func TestSeedDemoReplays_SansRejeuNiDossierNeCreeRien(t *testing.T) {
	demo := t.TempDir()
	layout := titlePkg.NewDemoLayout(demo)
	seedDemoReplays(context.Background(), SeedDemoOptions{RepoRoot: t.TempDir(), TitleSlug: "halo_5"}, layout, nil, nil)
	if fileExists(layout.ReplaysDir("halo_5")) {
		t.Error("un titre sans rejeu figé ne doit rien créer")
	}
}

// TestInstallDemoArtifact_RecuitQuandRienNEstAJour — revue R1 (P2-6) : la branche de recuisson
// est EXERCÉE (couture cookDemoReplayFunc à la place de l'enfant de décodage) : artefact
// périmé partout et film présent → recuit ; recuisson en échec → l'artefact périmé reste
// servi ; pas de film → aucune recuisson tentée.
func TestInstallDemoArtifact_RecuitQuandRienNEstAJour(t *testing.T) {
	ctx := context.Background()
	slug := titlePkg.DefaultSlug
	const id = "ffff0001"
	for _, tc := range []struct {
		nom      string
		filmOK   bool
		cookErr  error
		attendu  string
		appelles int
	}{
		{"film_present_recuit", true, nil, DemoReplayCooked, 1},
		{"recuisson_en_echec", true, errCuissonTest, DemoReplayStale, 1},
		{"sans_film", false, nil, DemoReplayStale, 0},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			src, demo := t.TempDir(), t.TempDir()
			layout := titlePkg.NewDemoLayout(demo)
			ecrireTest(t, titlePkg.NewPathResolver(src).ReplayArtifactPath(slug, id), artefactJSON(t, id, replay.SchemaVersion-1))
			appels := 0
			ancien := cookDemoReplayFunc
			t.Cleanup(func() { cookDemoReplayFunc = ancien })
			cookDemoReplayFunc = func(_ context.Context, _ SeedDemoOptions, l titlePkg.DemoLayout, m string) error {
				appels++
				if tc.cookErr != nil {
					return tc.cookErr
				}
				ecrireTest(t, l.ReplayArtifactPath(slug, m), artefactJSON(t, m, replay.SchemaVersion))
				return nil
			}
			got := installDemoArtifact(ctx, SeedDemoOptions{RepoRoot: src, TitleSlug: slug}, layout, id, tc.filmOK)
			if got != tc.attendu || appels != tc.appelles {
				t.Errorf("issue %q après %d recuisson(s), attendu %q après %d", got, appels, tc.attendu, tc.appelles)
			}
		})
	}
}

var errCuissonTest = errors.New("cuisson en échec (test)")
