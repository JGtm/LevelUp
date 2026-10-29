package replaybuild

// bots_des_fixtures_rejeu_test.go — LA PORTE D'ECRITURE DES BOTS DES FIXTURES DU REJEU.
//
// Les fixtures d'entrees du rejeu (`film/replay/testdata/inputs_<id>.bin.gz`) figent ce que l'etage
// de balayage du film rend ; les BOTS, eux, arrivent a la cuisson par le kill-feed (`killsource`,
// BOT_METADATA) projete par `replayidentity.BotIdentities`. Sans eux, l'assemblage d'un fixture
// laissait sans identite les corps que la cuisson nomme par leur bot (`bcb6d393` : deux vies de
// `343 Hollis`). Cette porte les fige a cote des entrees, par les fonctions de la cuisson
// (`decodeKillSource`, `BotIdentities`) : `replay` ne peut pas les appeler, `replayidentity`
// l'important.
//
// REGENERATION (jamais d'edition a la main) :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache go test ./internal/replaybuild/ \
//	  -run '^TestBotsDesFixturesDuRejeuRegenerate$' -update-bots-fixtures

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/replayidentity"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

var updateBotsFixtures = flag.Bool("update-bots-fixtures", false,
	"reecrire film/replay/testdata/bots_<id>.json depuis les films du cache (REPLAY_FILM_CACHE)")

// dossierDesFixturesDuRejeu : les fixtures du rejeu, relatives au paquet.
const dossierDesFixturesDuRejeu = "../../../../libs/film/replay/testdata"

func TestBotsDesFixturesDuRejeuRegenerate(t *testing.T) {
	cache := os.Getenv("REPLAY_FILM_CACHE")
	if !*updateBotsFixtures || cache == "" {
		t.Skip("regeneration des bots des fixtures : passer -update-bots-fixtures et REPLAY_FILM_CACHE")
	}
	ctx := context.Background()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	b, err := NewBuilder(ctx, repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("constructeur : %v", err)
	}
	entrees, err := filepath.Glob(filepath.Join(dossierDesFixturesDuRejeu, "inputs_*.bin.gz"))
	if err != nil || len(entrees) == 0 {
		t.Fatalf("aucun fixture d'entrees sous %s (%v)", dossierDesFixturesDuRejeu, err)
	}
	for _, chemin := range entrees {
		id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(chemin), "inputs_"), ".bin.gz")
		faits, err := ReadFactsFile(filepath.Join(dossierDesFixturesDuRejeu, "equivalence", id+".facts.json"))
		if err != nil {
			t.Fatalf("%s : faits : %v", id, err)
		}
		carte, err := b.ResolveMapEntry(faits.MapNames)
		if err != nil {
			t.Fatalf("%s : carte : %v", id, err)
		}
		film, ok, err := filmcache.LoadFilm(cache, id)
		if err != nil || !ok {
			t.Fatalf("%s : film absent du cache %s : %v", id, cache, err)
		}
		res := b.decodeKillSource(ctx, id, carte, film)
		if res == nil {
			t.Fatalf("%s : kill-feed non decode", id)
		}
		// `killsourceRev` : la revision sous laquelle le kill-feed est decode ; `replay` refuse un
		// fichier fige sous une autre (`TestLesBotsDesFixturesPortentLaRevisionCourante`).
		fige := struct {
			KillsourceRev string               `json:"killsourceRev"`
			Bots          []replay.BotIdentity `json:"bots"`
		}{KillsourceRev: decfilm.Rev, Bots: replayidentity.BotIdentities(res)}
		blob, err := json.MarshalIndent(fige, "", "  ")
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(dossierDesFixturesDuRejeu, "bots_"+id+".json"), append(blob, '\n'), 0o600); err != nil {
			t.Fatalf("%s : %v", id, err)
		}
	}
	t.Fatalf("bots des fixtures reecrits sous %s ; relancer sans -update-bots-fixtures", dossierDesFixturesDuRejeu)
}
