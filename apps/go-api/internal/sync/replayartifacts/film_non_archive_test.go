package replayartifacts

// film_non_archive_test.go — « ne pas archiver » (ADR 0045 D-12) : les chunks d'un film cuit sont
// retirés, et seulement quand l'archive est désactivée.

import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/sync/filmfetch"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

func ecrireFilmDeTest(t *testing.T, racine, court string) {
	t.Helper()
	err := filmarchive.Write(t.Context(), racine, court, []filmarchive.WriteChunk{
		{Index: 0, ChunkType: 1, Data: []byte("registre")},
		{Index: 1, ChunkType: finalise.ChunkTypeTempsForts, Data: []byte("temps forts")},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func TestRetirerFilmNonArchive(t *testing.T) {
	for _, tc := range []struct {
		nom          string
		sansArchive  bool
		resteAuCache bool
	}{
		{"archive active : le film reste", false, true},
		{"archive desactivee : le film est retire", true, false},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			racine := t.TempDir()
			ecrireFilmDeTest(t, racine, "0000a001")
			retirerFilmNonArchive(t.Context(), Deps{FilmRoot: racine, SansArchive: tc.sansArchive}, "0000a001")
			_, found, err := filmcache.Open(racine, "0000a001")
			if err != nil || found != tc.resteAuCache {
				t.Fatalf("film present = %v (err %v), attendu %v", found, err, tc.resteAuCache)
			}
		})
	}
}

func TestHookArchive_ParDefautOui(t *testing.T) {
	var h *Hook
	if !h.Archive() || !(&Hook{}).Archive() {
		t.Fatal("un hook sans reglage doit conserver les films")
	}
	non := &Hook{ArchiveFilms: func() bool { return false }}
	if non.Archive() {
		t.Fatal("le reglage « ne pas archiver » n'est pas relayé")
	}
}

// fetcherCompte : le client Halo du serveur, qui n'a PAS de cache de film (aucun SetLocalFilmCache ni
// WithLocalFilmCache hors de la ligne de commande) : il télécharge à chaque appel, que le film soit déjà
// dans l'archive ou non.
type fetcherCompte struct{ appels int }

func (f *fetcherCompte) GetFilmChunks(context.Context, string) ([]filmfetch.FilmChunk, bool, error) {
	f.appels++
	return []filmfetch.FilmChunk{
		{Index: 0, ChunkType: filmfetch.FilmChunkTypeHeader, Data: []byte("h")},
		{Index: 1, ChunkType: filmfetch.FilmChunkTypeReplicationData, Data: []byte("r")},
		{Index: 2, ChunkType: finalise.ChunkTypeTempsForts, Data: []byte("t")},
	}, true, nil
}

// TestPersistFilmToCache_RetelechargeUnFilmDejaArchive : la passe des rejeux ne consulte pas l'archive avant
// de télécharger (le cache-first est celui du client, absent sur le serveur). Un film écrit au préalable par la
// source des frags n'évite donc aucun second téléchargement : écrire « pour la cuisson » ne sert à rien.
func TestPersistFilmToCache_RetelechargeUnFilmDejaArchive(t *testing.T) {
	racine := t.TempDir()
	ecrireFilmDeTest(t, racine, titlePkg.FilmShortMatchID("0a0a0a0a-1111-2222-3333-444455556666"))
	f := &fetcherCompte{}
	r := persistFilmToCache(t.Context(), Deps{Fetcher: f, FilmRoot: racine}, "0a0a0a0a-1111-2222-3333-444455556666")
	if f.appels != 1 || !r.sauve {
		t.Fatalf("appels réseau = %d, sauvé = %v : la passe des rejeux télécharge malgré l'archive", f.appels, r.sauve)
	}
}
