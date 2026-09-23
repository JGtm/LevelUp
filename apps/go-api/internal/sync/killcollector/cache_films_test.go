package killcollector

// cache_films_test.go — UN MANIFESTE DU CACHE NON FINALISE N'EST PAS SERVI (lot L3, 2026-09-23).
//
// Le temoin est la forme d'`ab526724` : un manifeste valide le 2026-09-22 sur 34 morceaux, sans
// le morceau des temps forts. Le cache le servait tel quel, le decodeur rendait `ErrNoKillFeed`, et
// le killsource retentait « sans kill-feed » a chaque cycle (276 fois en 13 h) sans que le reseau,
// qui avait finalise le film onze secondes plus tard, soit jamais consulte.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/sync/haloclient"
)

// cacheAvecManifestePartiel archive `chunksTemoins` puis REMPLACE le manifeste par ses deux
// premieres entrees — sans temps forts —, ce qu'ecrivait le writer d'avant le lot quand l'API
// servait un film pas encore finalise. Les fichiers passent par `filmcache.Write` : leur nom
// n'est declare que la.
func cacheAvecManifestePartiel(t *testing.T) string {
	t.Helper()
	racine := cacheVide(t)
	court := titlePkg.FilmShortMatchID(matchRemote)
	var aEcrire []filmcache.WriteChunk
	var entrees []haloclient.CachedChunk
	for _, c := range chunksTemoins() {
		aEcrire = append(aEcrire, filmcache.WriteChunk{Index: c.Index, ChunkType: c.ChunkType,
			StartMS: c.StartMS, DurationMS: c.DurationMS, Data: c.Data})
		entrees = append(entrees, haloclient.CachedChunk{Index: c.Index, ChunkType: c.ChunkType,
			StartMS: c.StartMS, DurationMS: c.DurationMS})
	}
	if err := filmcache.Write(racine, court, aEcrire); err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(haloclient.CachedManifest{Chunks: entrees[:2]})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(racine, court), blob, 0o644); err != nil {
		t.Fatal(err)
	}
	return racine
}

// TestLocalCacheFilms_ManifestePartiel_NonServi : le film n'est pas servi — et pas en silence.
//
// L'ERREUR EST TYPEE ET NON « ABSENT » (`found = false`), et c'est delibere : un absent pose,
// dans la passe HORS LIGNE (`backfill-killsource` sans reseau), le marqueur TERMINAL
// `MBitFilmAbsent`, qui retirerait a vie des rattrapages un film parfaitement recuperable.
// L'erreur ne pose aucun marqueur ; en ligne, `RemoteFilms` retombe sur le reseau (test suivant).
func TestLocalCacheFilms_ManifestePartiel_NonServi(t *testing.T) {
	racine := cacheAvecManifestePartiel(t)
	src := NewLocalCacheFilms(haloclient.NewLocalFilmCache(racine))

	chunks, found, err := src.GetFilmChunks(context.Background(), matchRemote)
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) {
		t.Fatalf("err = %v, attendu filmcache.ErrFilmNonFinalise", err)
	}
	if found || chunks != nil {
		t.Errorf("manifeste partiel servi : found=%v, %d morceaux", found, len(chunks))
	}
}

// TestRemoteFilms_ManifesteLocalPartiel_SeRepareParLeReseau : LE CAS DU TEMOIN, EN LIGNE. Le
// reseau sert le film finalise, l'archivage COMPLETE le manifeste partiel, et la lecture
// suivante vient du disque.
func TestRemoteFilms_ManifesteLocalPartiel_SeRepareParLeReseau(t *testing.T) {
	racine := cacheAvecManifestePartiel(t)
	distant := &filmsEnMemoire{chunks: map[string][]haloclient.FilmChunk{matchRemote: chunksTemoins()}}
	src := NewRemoteFilms(NewLocalCacheFilms(haloclient.NewLocalFilmCache(racine)), distant, racine)

	got, found, err := src.GetFilmChunks(context.Background(), matchRemote)
	if err != nil || !found || len(got) != 3 {
		t.Fatalf("GetFilmChunks = (%d, %v, %v), attendu le film finalise (3, true, nil)", len(got), found, err)
	}
	if distant.appels != 1 {
		t.Fatalf("appels reseau = %d, attendu 1 — le manifeste partiel a masque le reseau", distant.appels)
	}
	cm, err := haloclient.NewLocalFilmCache(racine).LoadManifest(matchRemote)
	if err != nil || cm == nil || len(cm.Chunks) != 3 {
		t.Fatalf("manifeste apres archivage : %+v (err %v), attendu 3 entrees", cm, err)
	}
	if _, found, err := src.GetFilmChunks(context.Background(), matchRemote); err != nil || !found {
		t.Fatalf("seconde lecture = (%v, %v)", found, err)
	}
	if distant.appels != 1 {
		t.Errorf("appels reseau = %d apres reparation, attendu 1 — le disque doit servir", distant.appels)
	}
}
