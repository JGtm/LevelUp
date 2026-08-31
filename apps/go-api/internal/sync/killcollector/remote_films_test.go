package killcollector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
)

const matchRemote = "aabbccdd-1111-2222-3333-444455556666"

// filmsEnMemoire : le CDN, en memoire. Volontairement local a ce fichier plutot que partage
// avec `fakeFilmClient` (collector_test.go), qui est sous `//go:build integration` : ces
// tests-ci doivent tourner dans la suite rapide.
type filmsEnMemoire struct {
	chunks map[string][]haloclient.FilmChunk
	appels int
}

func (f *filmsEnMemoire) GetFilmChunks(_ context.Context, matchID string) ([]haloclient.FilmChunk, bool, error) {
	f.appels++
	c, ok := f.chunks[matchID]
	if !ok {
		return nil, false, nil
	}
	return c, true, nil
}

// cacheVide : une racine de cache VIDE mais EXISTANTE — l etat que `preparerCacheFilms`
// garantit cote CLI. Sans les dossiers, `NewLocalFilmCache` rend nil pour tout le process et
// le film archive ne serait jamais relu.
func cacheVide(t *testing.T) string {
	t.Helper()
	racine := t.TempDir()
	for _, d := range []string{filmcache.ChunksRoot(racine), filepath.Dir(filmcache.ManifestPath(racine, "0"))} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return racine
}

func chunksTemoins() []haloclient.FilmChunk {
	return []haloclient.FilmChunk{
		{Index: 0, ChunkType: 1, Data: []byte("entete"), StartMS: 0, DurationMS: 100},
		{Index: 1, ChunkType: 2, Data: []byte("replication"), StartMS: 100, DurationMS: 900},
		{Index: 2, ChunkType: 3, Data: []byte("killfeed"), StartMS: 1000, DurationMS: 10},
	}
}

// TestRemoteFilms_TelechargeEtArchive : LE CŒUR DU CORRECTIF.
//
// Le film n est pas en cache — c est l etat de tous les matchs postérieurs au 2026-04-07
// (`.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`). La source doit alors le TELECHARGER, et
// l ECRIRE au cache : un film expire ne se retelecharge jamais, et le jeter apres decodage
// perdrait une donnee irremplacable qu on a eue en main.
func TestRemoteFilms_TelechargeEtArchive(t *testing.T) {
	racine := cacheVide(t)
	distant := &filmsEnMemoire{chunks: map[string][]haloclient.FilmChunk{matchRemote: chunksTemoins()}}
	src := NewRemoteFilms(NewLocalCacheFilms(haloclient.NewLocalFilmCache(racine)), distant, racine)

	got, found, err := src.GetFilmChunks(context.Background(), matchRemote)
	if err != nil || !found {
		t.Fatalf("GetFilmChunks = (found=%v, err=%v), attendu trouve", found, err)
	}
	if len(got) != 3 {
		t.Fatalf("chunks = %d, attendu 3", len(got))
	}

	court := titlePkg.FilmShortMatchID(matchRemote)
	if _, err := os.Stat(filmcache.ManifestPath(racine, court)); err != nil {
		t.Fatalf("manifeste non archive : %v — un film telecharge puis jete est une perte seche", err)
	}
	// Le NOM des fichiers de chunks n est declare que dans `filmcache` (garde-rail
	// filmcache_guard_test.go) : on compte les entrees, on ne recopie pas la convention.
	entrees, err := os.ReadDir(filmcache.ChunkDir(racine, court))
	if err != nil || len(entrees) != 3 {
		t.Errorf("chunks archives = %d (err=%v), attendu 3", len(entrees), err)
	}

	// Deuxieme lecture : le disque sert, le reseau n est PAS retouche. Le film est immuable ;
	// une passe --force sur 400 matchs qui retelechargerait tout serait une faute.
	avantCache := observability.LoadCounter(CompteurFilmsDepuisCache)
	if _, found, err := src.GetFilmChunks(context.Background(), matchRemote); err != nil || !found {
		t.Fatalf("seconde lecture = (found=%v, err=%v)", found, err)
	}
	if distant.appels != 1 {
		t.Errorf("appels reseau = %d, attendu 1 — le cache doit servir la seconde lecture", distant.appels)
	}
	if apres := observability.LoadCounter(CompteurFilmsDepuisCache); apres != avantCache+1 {
		t.Errorf("%s = %d, attendu %d", CompteurFilmsDepuisCache, apres, avantCache+1)
	}
}

// TestRemoteFilms_FilmExpire : 404/410 cote serveur = ETAT NORMAL, pas panne. Le collecteur
// le compte en `killsource_films_absents` et passe au match suivant.
func TestRemoteFilms_FilmExpire(t *testing.T) {
	racine := t.TempDir()
	src := NewRemoteFilms(
		NewLocalCacheFilms(haloclient.NewLocalFilmCache(racine)),
		&filmsEnMemoire{chunks: map[string][]haloclient.FilmChunk{}},
		racine,
	)
	got, found, err := src.GetFilmChunks(context.Background(), matchRemote)
	if err != nil {
		t.Fatalf("err = %v, attendu nil : un film expire n est pas une panne", err)
	}
	if found || got != nil {
		t.Errorf("(found=%v, chunks=%v), attendu (false, nil)", found, got)
	}
}

// TestRemoteFilms_SansClientDistant : source inerte, pas panne. C est le comportement d une
// passe hors ligne qui n a rien en cache — elle ne fait rien, proprement.
func TestRemoteFilms_SansClientDistant(t *testing.T) {
	src := NewRemoteFilms(NewLocalCacheFilms(haloclient.NewLocalFilmCache(t.TempDir())), nil, "")
	if _, found, err := src.GetFilmChunks(context.Background(), matchRemote); found || err != nil {
		t.Errorf("(found=%v, err=%v), attendu (false, nil)", found, err)
	}
}

// TestRemoteFilms_EchecArchivageNonFatal_MaisCOMPTE : les octets sont en memoire, le decodage
// peut se faire — donc l echec d ecriture ne doit pas perdre le match. Mais il est COMPTE :
// un disque plein qui ferait echouer l archivage en silence ferait repayer le reseau a chaque
// passe sans que personne ne le sache.
func TestRemoteFilms_EchecArchivageNonFatal_MaisCOMPTE(t *testing.T) {
	racine := t.TempDir()
	// Un FICHIER la ou le writer veut un repertoire : l ecriture echoue a coup sur.
	if err := os.WriteFile(filepath.Join(racine, "film_chunks"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	distant := &filmsEnMemoire{chunks: map[string][]haloclient.FilmChunk{matchRemote: chunksTemoins()}}
	src := NewRemoteFilms(NewLocalCacheFilms(haloclient.NewLocalFilmCache(racine)), distant, racine)

	avant := observability.LoadCounter(CompteurArchiveErreurs)
	got, found, err := src.GetFilmChunks(context.Background(), matchRemote)
	if err != nil || !found || len(got) != 3 {
		t.Fatalf("GetFilmChunks = (%d chunks, found=%v, err=%v), attendu les 3 chunks malgre "+
			"l echec d archivage", len(got), found, err)
	}
	if apres := observability.LoadCounter(CompteurArchiveErreurs); apres != avant+1 {
		t.Errorf("%s = %d, attendu %d — un echec d archivage avale est le defaut qu on corrige",
			CompteurArchiveErreurs, apres, avant+1)
	}
}
