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
	"os"
	"path/filepath"
	"testing"
)

// botsDuFixture rend les bots figes du fixture `id`. Le fichier est obligatoire : un fixture sans
// ses bots ne dit pas ce que la cuisson publierait.
func botsDuFixture(t *testing.T, id string) []BotIdentity {
	t.Helper()
	chemin := filepath.Join(goldenDir, "bots_"+id+".json")
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis la table des fixtures
	if err != nil {
		t.Fatalf("bots du fixture absents (%s) : %v — regenerer par "+
			"replaybuild.TestBotsDesFixturesDuRejeuRegenerate", chemin, err)
	}
	var bots []BotIdentity
	if err := json.Unmarshal(blob, &bots); err != nil {
		t.Fatalf("bots du fixture illisibles (%s) : %v", chemin, err)
	}
	return bots
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
