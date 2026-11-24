package replayartifacts

// film_non_archive_test.go — « ne pas archiver » (ADR 0045 D-12) : les chunks d'un film cuit sont
// retirés, et seulement quand l'archive est désactivée.

import (
	"testing"

	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/replaybuild"

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

// TestHookArchiveFrags : la source des frags écrit le film quand le réglage l'y autorise, ou
// quand la cuisson locale du même cycle va le relire puis le retirer (sans quoi il serait
// téléchargé deux fois). Un site qui ne cuit pas (ouvrier, rien) ne garde rien.
func TestHookArchiveFrags(t *testing.T) {
	for _, tc := range []struct {
		nom      string
		archive  bool
		lieu     string
		env      replaybuild.PlacementEnv
		attendu  bool
		nilHook  bool
		nilArchi bool
	}{
		{nom: "archive active", archive: true, env: replaybuild.PlacementEnv{Production: true}, attendu: true},
		{nom: "archive coupee, cuisson locale : le film sert de relais", lieu: "", attendu: true},
		{nom: "archive coupee, hebergé sans ouvrier : rien n est garde",
			env: replaybuild.PlacementEnv{Production: true}, attendu: false},
		{nom: "archive coupee, hebergé avec ouvrier : rien n est garde",
			env: replaybuild.PlacementEnv{Production: true, WorkerConfigured: true}, attendu: false},
		{nom: "archive coupee, cuisson coupee", lieu: "off", attendu: false},
		{nom: "archive coupee, cuisson locale refusee en production : rien n est garde",
			lieu: "local", env: replaybuild.PlacementEnv{Production: true}, attendu: false},
		{nom: "hook nil : defaut oui", nilHook: true, attendu: true},
		{nom: "reglage non cable : defaut oui", nilArchi: true, attendu: true},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			h := &Hook{
				ArchiveFilms: func() bool { return tc.archive },
				Location:     func() string { return tc.lieu },
				Env:          tc.env,
			}
			if tc.nilHook {
				h = nil
			}
			if tc.nilArchi {
				h.ArchiveFilms = nil
			}
			if got := h.ArchiveFrags(t.Context()); got != tc.attendu {
				t.Fatalf("ArchiveFrags = %v, attendu %v", got, tc.attendu)
			}
		})
	}
}
