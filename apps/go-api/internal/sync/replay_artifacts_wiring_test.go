package sync

// replay_artifacts_wiring_test.go — LE GARDE-RAIL DU CABLAGE DE L ETAPE 1.58.
//
// POURQUOI IL EXISTE, ET IL FAUT LE LIRE AVANT DE LE TOUCHER.
//
// L etape 1.57 obtenait sa capacite film par assertion de type. Aucun client de production ne
// la portait : l assertion echouait partout, l etape ne s executait nulle part, et RIEN ne le
// disait — cinq mois de donnee arretee (`.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`). Le
// correctif fut `kill_source_wiring_test.go` : des assertions qui NE PEUVENT PAS MENTIR parce
// qu elles ne compilent pas quand un client perd la methode.
//
// L etape 1.58 obtenait la SIENNE par la meme assertion, et n avait AUCUN garde-rail. Mesure
// du 2026-09-01 : 1 seul des 222 matchs des 90 derniers jours porte un artefact, et le journal
// de synchronisation ne contient pas une ligne « rejeu 2D ». Ce fichier ferme le meme trou sur
// la meme etape voisine.
//
// Toute NOUVELLE implementation de HaloClient qui traverse le post-sync doit etre ajoutee ici.

import (
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/sync/replayartifacts"
)

// LES CLIENTS DE PRODUCTION PORTENT LA CAPACITE CHUNKS — VERIFIE A LA COMPILATION.
//
// `replayartifacts.ChunksFetcher` et `killcollector.FilmChunkFetcher` sont deux interfaces
// DISTINCTES de meme forme, declarees dans deux paquets : verifier l une ne verifie pas
// l autre. C est exactement le genre de jumelage qui derive en silence — d ou les deux jeux
// d assertions, cote a cote.
var (
	_ replayartifacts.ChunksFetcher = (*PooledHaloClient)(nil)
	_ replayartifacts.ChunksFetcher = (*cachedHaloClient)(nil)
	_ replayartifacts.ChunksFetcher = (*HaloAPIClient)(nil)
)

// TestReplayArtifactsClientsDeProductionTraversentLAssertion : le pendant DYNAMIQUE des
// assertions ci-dessus — il refait l assertion exactement comme `runReplayArtifacts`, sur les
// clients tels que le moteur les construit (enveloppe de cache comprise), et non sur le type
// nu. Une enveloppe qui oublierait de re-exposer la methode ne se verrait pas autrement.
func TestReplayArtifactsClientsDeProductionTraversentLAssertion(t *testing.T) {
	cas := []struct {
		nom    string
		client HaloClient
	}{
		{"HaloAPIClient nu", NewHaloAPIClient("spartan", "clearance", 4)},
		{"enveloppe de cache (chemin V1)", NewCachedHaloClient(
			NewHaloAPIClient("spartan", "clearance", 4),
			FetchCacheConfig{CacheDir: t.TempDir()})},
		{"client poole (chemin serveur)", &PooledHaloClient{}},
	}
	for _, c := range cas {
		if _, ok := c.client.(replayartifacts.ChunksFetcher); !ok {
			t.Errorf("%s (%T) ne passe pas l assertion de runReplayArtifacts : l etape 1.58 "+
				"n archiverait aucun film et ne construirait aucun artefact", c.nom, c.client)
		}
	}
}

// TestReplayArtifactsAppeleeParLePipeline : l installation ne sert a rien si le pipeline ne
// l appelle pas. Le test lit le pipeline plutot que de le jouer (il exige une base, des tokens
// et un reseau) — ce qu il verrouille, c est la PRESENCE de l appel.
func TestReplayArtifactsAppeleeParLePipeline(t *testing.T) {
	src, err := os.ReadFile("engine_postsync.go")
	if err != nil {
		t.Fatalf("lecture du pipeline post-sync : %v", err)
	}
	if !strings.Contains(string(src), "films.runReplayArtifacts(ctx, insertedIDs)") {
		t.Error("runReplayArtifacts a disparu du pipeline post-sync : plus aucun film ne serait " +
			"archive au fil de l eau, et les films EXPIRENT cote serveur Halo")
	}
}

// TestReplayArtifactsAssertionRateeNeSeTaitPas — LE DEFAUT EXACT QUE CE LOT CORRIGE.
//
// `fetcher, _ := s.client.(replayartifacts.ChunksFetcher)` jetait le booleen : un client sans
// la capacite donnait un fetcher nil et l etape sortait sans un mot. Le test lit le cablage et
// exige que le resultat de l assertion soit EXAMINE — une assertion muette est indetectable a
// l execution, donc elle se verrouille a la source.
func TestReplayArtifactsAssertionRateeNeSeTaitPas(t *testing.T) {
	src, err := os.ReadFile("convergence.go")
	if err != nil {
		t.Fatalf("lecture du cablage : %v", err)
	}
	cablage := string(src)
	if strings.Contains(cablage, "fetcher, _ := s.client.(replayartifacts.ChunksFetcher)") {
		t.Error("l assertion ChunksFetcher jette de nouveau son resultat : un client sans la " +
			"capacite desarmerait l etape 1.58 en silence (defaut du 2026-09-01)")
	}
	if !strings.Contains(cablage, "replayartifacts.SignalerClientSansChunks(") {
		t.Error("le cablage ne signale plus l echec de l assertion ChunksFetcher : sans ce " +
			"signal, « l etape ne peut rien faire » et « l etape n existe pas » s ecrivent pareil")
	}
}

// TestReplayArtifactsNeFiltrePasSurLesInseres : le cablage ne doit PAS sortir quand le cycle
// n a rien insere. Le film Theater se publie APRES le match ; un filtre sur `insertedIDs`
// interdit tout rattrapage, et c est la seconde moitie du defaut mesure le 2026-09-01.
func TestReplayArtifactsNeFiltrePasSurLesInseres(t *testing.T) {
	src, err := os.ReadFile("convergence.go")
	if err != nil {
		t.Fatalf("lecture du cablage : %v", err)
	}
	if strings.Contains(string(src), "e.replayArtifacts == nil || len(insertedIDs) == 0") {
		t.Error("runReplayArtifacts refiltre sur les matchs inseres : un cycle sans insertion " +
			"ne rattraperait jamais un film publie apres coup")
	}
}

// TestReplayArtifactsHookInstalleParLesTroisChemins — LA « FACTORY OUBLIEE », VERROUILLEE.
//
// Le hook n est PAS installe par le constructeur du moteur (il lui faut la configuration et le
// magasin de reglages) : il l est par les trois sites de cablage. Un site oublie ne se voit
// nulle part — c est la lecon de la revue du 2026-08-29. Le test lit les trois fichiers.
func TestReplayArtifactsHookInstalleParLesTroisChemins(t *testing.T) {
	sites := map[string]string{
		"scheduler (auto-sync)": "../scheduler/auto_sync_engine.go",
		"handler HTTP (legacy)": "../api/handlers/sync_handler.go",
		"cablage V2 (serveur)":  "../../cmd/server/sync_v2_wiring.go",
	}
	for nom, chemin := range sites {
		src, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatalf("%s : lecture de %s : %v", nom, chemin, err)
		}
		if !strings.Contains(string(src), "WithReplayArtifacts(replayartifacts.NewHook(") {
			t.Errorf("%s (%s) n installe plus le hook du rejeu 2D : ce chemin de sync "+
				"n archiverait aucun film, et rien ne le dirait", nom, chemin)
		}
	}
}
