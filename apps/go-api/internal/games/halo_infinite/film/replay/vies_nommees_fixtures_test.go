package replay

// vies_nommees_fixtures_test.go — LES VIES ANONYMES N'EXISTENT PAS, SUR CHAQUE FIXTURE.
//
// Decision utilisateur du 2026-09-07 : une vie est un humain ou un bot. Une vie publiee sans
// identite est un DEFAUT DE NOMMAGE ; la production le compte (`coverage.bridge.unnamedLives`) et
// l'alarme en ERREUR, et ces tests en font une erreur de test sur les fixtures d'entrees figees
// (`testdata/inputs_<id>.bin.gz`), assemblees avec les bots que la cuisson recoit du kill-feed
// (`testdata/bots_<id>.json`, porte d'ecriture `replaybuild.TestBotsDesFixturesDuRejeuRegenerate`) :
//
//	VN-CUISSON  le document publie de chaque fixture ;
//	VN-SYNC     le document interne que `PortagesAuSync` assemble, sous les options que ce chemin
//	            pose ([EntreePorteursAuSync.optionsDuRegistre]) sur les memes lectures.
//
// Aucun fixture n'est d'un mode que l'application ne gere pas (Firefight) : les deux tests
// n'exemptent rien.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
)

// botsDuFixture rend les bots figes du fixture `id`. Le fichier est obligatoire : un fixture sans
// ses bots ne dit pas ce que la cuisson publierait.
func botsDuFixture(t *testing.T, id string) []BotIdentity {
	t.Helper()
	f := lireBotsDuFixture(t, filepath.Join(goldenDir, "bots_"+id+".json"))
	if err := f.verifierRevision(); err != nil {
		t.Fatalf("bots du fixture %s : %v", id, err)
	}
	return f.Bots
}

// botsFiges : la forme d'un fichier `testdata/bots_<id>.json`. `KillsourceRev` est la revision de
// `killsource` sous laquelle la porte d'ecriture (`replaybuild.TestBotsDesFixturesDuRejeuRegenerate`)
// a decode le kill-feed : les bots viennent de BOT_METADATA, que cette couche lit, et un fichier
// fige sous une autre revision peut porter une equipe ou un roster que la cuisson ne rend plus
// (`bcb6d393`, quatre bots en equipe 0 au lieu de 1 avant `killsource-2026-10-16`).
type botsFiges struct {
	KillsourceRev string        `json:"killsourceRev"`
	Bots          []BotIdentity `json:"bots"`
}

// verifierRevision refuse un fichier fige sous une autre revision que [killsource.Rev].
func (f botsFiges) verifierRevision() error {
	if f.KillsourceRev == killsource.Rev {
		return nil
	}
	return fmt.Errorf("fige sous la revision %q, la revision courante est %q — regenerer par "+
		"REPLAY_FILM_CACHE=<repo>/data/cache go test ./internal/replaybuild/ "+
		"-run '^TestBotsDesFixturesDuRejeuRegenerate$' -update-bots-fixtures", f.KillsourceRev, killsource.Rev)
}

// lireBotsDuFixture lit un fichier de bots figes.
func lireBotsDuFixture(t *testing.T, chemin string) botsFiges {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis la table des fixtures
	if err != nil {
		t.Fatalf("bots du fixture absents (%s) : %v — regenerer par "+
			"replaybuild.TestBotsDesFixturesDuRejeuRegenerate", chemin, err)
	}
	var f botsFiges
	if err := json.Unmarshal(blob, &f); err != nil {
		t.Fatalf("bots du fixture illisibles (%s) : %v", chemin, err)
	}
	return f
}

// TestLesBotsDesFixturesPortentLaRevisionCourante : chaque `testdata/bots_<id>.json` porte la
// revision de `killsource` sous laquelle il a ete fige, et elle est la courante. Sans ce garde, une
// montee de la couche (equipe des bots lue autrement, roster change) laissait les fixtures assembler
// des bots d avant elle sans que rien ne rougisse. Mutation vue rouge : `killsourceRev` de
// `bots_bcb6d393.json` remis a `killsource-2026-10-15`.
func TestLesBotsDesFixturesPortentLaRevisionCourante(t *testing.T) {
	chemins, err := filepath.Glob(filepath.Join(goldenDir, "bots_*.json"))
	if err != nil || len(chemins) == 0 {
		t.Fatalf("aucun fichier de bots figes sous %s (%v)", goldenDir, err)
	}
	for _, chemin := range chemins {
		if err := lireBotsDuFixture(t, chemin).verifierRevision(); err != nil {
			t.Errorf("%s : %v", filepath.Base(chemin), err)
		}
	}
}

// exigerViesNommees fait echouer le test sur toute piste sans xuid ni bot, et sur un residu publie.
func exigerViesNommees(t *testing.T, quoi string, doc ReplayDocument) {
	t.Helper()
	for _, tr := range doc.Tracks {
		if tr.XUID == "" && tr.Bot == "" {
			t.Errorf("%s : piste du slot %d [%d-%d] sans identite — une vie est un humain ou un bot",
				quoi, tr.Slot, tr.StartFrame, tr.EndFrame)
		}
	}
	if doc.Coverage != nil && doc.Coverage.Bridge.UnnamedLives != 0 {
		t.Errorf("%s : coverage.bridge.unnamedLives = %d, attendu 0", quoi, doc.Coverage.Bridge.UnnamedLives)
	}
}

// TestChaqueVieDesFixturesEstNommeeALaCuisson (VN-CUISSON). Mutation vue rouge : assembler
// `bcb6d393` sans ses bots (deux vies de `343 Hollis` sans identite).
func TestChaqueVieDesFixturesEstNommeeALaCuisson(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			exigerViesNommees(t, b.Short8, assemblerGoldenBuild(t, b, g, entry))
		})
	}
}

// TestChaqueVieDesFixturesEstNommeeAuSync (VN-SYNC). Mutation vue rouge : assembler sans les bots
// (`bcb6d393`). Les occupants et la table du film ne nomment aucune vie de plus sur ces fixtures :
// leur pose est tenue par `TestLeDocumentInternePoseLesSiegesEtLesOccupants`, leur effet par la
// parite sur films reels (`sync/killcollector/porteurs_au_sync_parite_research_test.go`).
func TestChaqueVieDesFixturesEstNommeeAuSync(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			e := EntreePorteursAuSync{MatchID: b.Short8, Carte: entry, Libelles: goldenCatalog(t),
				Identite: IdentityInput{Positions: g.Positions, BipedCreations: g.BipedCreations, Deaths: g.Deaths,
					PlayerIndices: g.PlayerIndices, Bots: botsDuFixture(t, b.Short8)}}
			opt := e.optionsDuRegistre(nil, siegesEtOccupants{table: g.FilmTable, equipes: g.PlayerTeams,
				rapport: g.TeamScan, entites: g.PlayerEntities})
			opt.documentInterne = true
			exigerViesNommees(t, b.Short8+" (document interne du sync)",
				BuildFromPositions(context.Background(), b.Short8, "", g.Positions, nil, opt))
		})
	}
}
