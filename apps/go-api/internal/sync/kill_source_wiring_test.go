package sync

// kill_source_wiring_test.go — LE GARDE-RAIL DU CABLAGE DE L ETAPE 1.57.
//
// POURQUOI IL EXISTE, ET IL FAUT LE LIRE AVANT DE LE TOUCHER. `assist_known` n a qu UNE
// origine — le kill-feed du film — et son seul producteur etait une commande manuelle. Le
// jour ou plus personne ne l a lancee (cache de films gele le 2026-04-07), la donnee s est
// arretee sans un log : cinq mois, deux blocs de l app disparus en silence
// (`.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`).
//
// La revue du meme jour avait par ailleurs montre qu un cablage pose sur UNE factory et pas
// sur sa jumelle ne prend jamais, et que rien ne le dit. Les deux lecons se rejoignent ici :
// l etape est installee DANS LE CONSTRUCTEUR, et ces tests interdisent qu on la debranche —
// que ce soit en retirant l installation par defaut ou en retirant l appel du pipeline.

import (
	"context"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/sync/killcollector"
)

// TestKillSourceInstalleParDefaut : TOUT moteur porte l etape, sans qu aucun site de wiring
// n ait a y penser. C est la seule protection contre « la factory oubliee ».
func TestKillSourceInstalleParDefaut(t *testing.T) {
	if e := NewSyncEngine(t.TempDir(), "GT", "123", nil, nil); e.killSource == nil {
		t.Error("NewSyncEngine n installe pas l etape 1.57 : un site de wiring devrait y penser, " +
			"et c est exactement comme ca que la donnee s est arretee cinq mois")
	}
	if e := NewSyncEngineForTitle(t.TempDir(), "halo_5", "GT", "123", nil, nil); e.killSource == nil {
		t.Error("NewSyncEngineForTitle n installe pas l etape 1.57 — la capability decide, " +
			"pas le constructeur")
	}
}

// LES CLIENTS DE PRODUCTION PORTENT LA CAPACITE FILM — VERIFIE A LA COMPILATION.
//
// C EST LE TEST QUI MANQUAIT, ET SON ABSENCE A COUTE LE LOT. L etape 1.57 obtient
// `GetFilmChunks` par assertion de type. Ni `*PooledHaloClient` (chemin serveur) ni
// `*cachedHaloClient` (pose SYSTEMATIQUEMENT sur le chemin V1) ne la portaient : l assertion
// echouait partout, l etape ne s executait nulle part, et les deux tests ci-dessous restaient
// verts parce qu ils ne verifiaient que « le hook est non nil » et « la ligne d appel existe
// dans le fichier ».
//
// Ces assertions-ci ne peuvent pas mentir : elles ne compilent pas si un client perd la
// methode. Toute NOUVELLE implementation de HaloClient qui traverse le post-sync doit etre
// ajoutee ici.
var (
	_ killcollector.FilmChunkFetcher = (*PooledHaloClient)(nil)
	_ killcollector.FilmChunkFetcher = (*cachedHaloClient)(nil)
	_ killcollector.FilmChunkFetcher = (*HaloAPIClient)(nil)
)

// TestKillSourceClientsDeProductionTraversentLAssertion : le pendant DYNAMIQUE des assertions
// ci-dessus — il refait l assertion exactement comme `runKillSource`, sur les clients tels que
// le moteur les construit (enveloppe de cache comprise), et non sur le type nu.
func TestKillSourceClientsDeProductionTraversentLAssertion(t *testing.T) {
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
		if _, ok := c.client.(killcollector.FilmChunkFetcher); !ok {
			t.Errorf("%s (%T) ne passe pas l assertion de runKillSource : l etape 1.57 sortirait "+
				"en silence et `assist_known` resterait FALSE", c.nom, c.client)
		}
	}
}

// TestKillSourceEnveloppeDeCacheDelegue : l enveloppe ne doit pas seulement COMPILER, elle doit
// DELEGUER. Une methode qui rendrait toujours `found = false` passerait les assertions
// ci-dessus tout en desarmant l etape aussi surement que son absence.
func TestKillSourceEnveloppeDeCacheDelegue(t *testing.T) {
	interne := &filmFetcherTemoin{chunks: []FilmChunk{{Index: 0, ChunkType: 1, Data: []byte("x")}}}
	env := &cachedHaloClient{inner: interne, cfg: FetchCacheConfig{CacheDir: t.TempDir()}}

	got, found, err := env.GetFilmChunks(context.Background(), "m1")
	if err != nil || !found || len(got) != 1 {
		t.Fatalf("GetFilmChunks = (%d chunks, found=%v, err=%v) ; l enveloppe n a pas delegue", len(got), found, err)
	}
	if interne.appels != 1 {
		t.Errorf("appels a l interne = %d, attendu 1", interne.appels)
	}
}

// filmFetcherTemoin : un HaloClient minimal qui porte la capacite film. Les methodes non
// utilisees paniquent volontairement — si l une d elles etait appelee ici, ce serait un bug de
// delegation, pas un cas a couvrir en silence.
type filmFetcherTemoin struct {
	HaloClient
	chunks []FilmChunk
	appels int
}

func (f *filmFetcherTemoin) GetFilmChunks(context.Context, string) ([]FilmChunk, bool, error) {
	f.appels++
	return f.chunks, true, nil
}

// TestKillSourceAppeleeParLePipeline : l installation ne sert a rien si le pipeline ne
// l appelle pas. Le test lit le pipeline plutot que de le jouer (il exige une base, des
// tokens et un reseau) — ce qu il verrouille, c est la PRESENCE de l appel.
func TestKillSourceAppeleeParLePipeline(t *testing.T) {
	src, err := os.ReadFile("engine_postsync.go")
	if err != nil {
		t.Fatalf("lecture du pipeline post-sync : %v", err)
	}
	if !strings.Contains(string(src), "films.runKillSource(ctx, insertedIDs)") {
		t.Error("runKillSource a disparu du pipeline post-sync : `assist_known` cessera d etre " +
			"produit, et rien ne le signalera (c est le defaut du 2026-04-07)")
	}
	pipeline := string(src)
	posKill := strings.Index(pipeline, "films.runKillSource(")
	posReplay := strings.Index(pipeline, "films.runReplayArtifacts(")
	if posKill < 0 || posReplay < 0 || posKill > posReplay {
		t.Error("l etape 1.57 doit passer AVANT les artefacts de rejeu : elle archive le film au " +
			"cache, que l etape 1.58 retrouve alors sur disque au lieu de le retelecharger")
	}
}
