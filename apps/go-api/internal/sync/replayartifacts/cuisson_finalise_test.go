package replayartifacts

// cuisson_finalise_test.go — UN FILM NON FINALISE N'EST NI ARCHIVE NI CUIT : LA CUISSON EST
// REPORTEE AU CYCLE SUIVANT, COMPTEE, JOURNALISEE EN INFO (lot L3, 2026-09-23).
//
// 23 matchs sur 96 sont detectes moins de 60 s apres leur fin (rapport ctf_ab526724 §2.3) : le
// serveur n'a pas encore publie leur morceau des temps forts. Ce n'est ni un film absent ni une
// panne — c'est un film qu'il faut reprendre une minute plus tard.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
)

// fetcherNonFinalise : le client Halo face a un film en cours de publication. `liste` choisit la
// forme du refus : l'erreur typee du client reel (`haloclient.fetchFilmChunks`), ou une liste
// sans temps forts rendue par un client qui ne l'aurait pas verifiee — le writer la refuse alors.
type fetcherNonFinalise struct{ liste bool }

func (f fetcherNonFinalise) GetFilmChunks(context.Context, string) ([]haloclient.FilmChunk, bool, error) {
	if f.liste {
		return []haloclient.FilmChunk{
			{Index: 0, ChunkType: haloclient.FilmChunkTypeHeader, Data: []byte("h")},
			{Index: 1, ChunkType: haloclient.FilmChunkTypeReplicationData, Data: []byte("r")},
		}, true, nil
	}
	return nil, false, fmt.Errorf("GetFilmChunks(m) : %w", filmcache.ErrFilmNonFinalise)
}

// TestPersistFilmToCache_NonFinalise_ReporteEtCompte : les deux formes du refus reportent le
// film (ni persiste ni disponible, donc pas de cuisson), et chacune se compte.
func TestPersistFilmToCache_NonFinalise_ReporteEtCompte(t *testing.T) {
	ctx := context.Background()
	for _, liste := range []bool{false, true} {
		t.Run(fmt.Sprintf("liste=%v", liste), func(t *testing.T) {
			racine := t.TempDir()
			d := Deps{Gamertag: "g", TitleSlug: titlePkg.DefaultSlug, CacheRoot: racine,
				Fetcher: fetcherNonFinalise{liste: liste}}
			avant := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), CompteurFilmsNonFinalises)

			sauve, dispo := persistFilmToCache(ctx, d, "m1")
			if sauve || dispo {
				t.Fatalf("film non finalise : (sauve=%v, dispo=%v), attendu (false, false)", sauve, dispo)
			}
			if n := observability.LoadCounterT(ctxkeys.TitleSlug(ctx), CompteurFilmsNonFinalises) - avant; n != 1 {
				t.Errorf("%s : +%d, attendu +1 — le report est invisible", CompteurFilmsNonFinalises, n)
			}
			if _, found, err := filmcache.Open(racine, titlePkg.FilmShortMatchID("m1")); found || err != nil {
				t.Errorf("un manifeste a ete valide pour un film non finalise (found=%v, err=%v)", found, err)
			}
		})
	}
}

// TestBuildAll_NonFinalise_AucuneCuisson : le cycle ne cuit pas un film non finalise, ne le
// compte pas en echec, et passe au suivant.
func TestBuildAll_NonFinalise_AucuneCuisson(t *testing.T) {
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: fetcherNonFinalise{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			appels++
			return BuildOneResult{}, errors.New("aucune cuisson ne doit partir")
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})
	if appels != 0 || b.echecs != 0 || b.construits != 0 || b.filmsSauves != 0 {
		t.Errorf("bilan = %+v (%d cuisson(s)), attendu aucune cuisson, aucun echec", b, appels)
	}
}

// TestCuireUnMatch_RefusNonFinaliseDeLEnfant_CompteEcarte : L3.5 vu du cycle. La cuisson (dans un
// ENFANT) refuse un film non finalise ; le parent ne recoit que le TEXTE de l'erreur, et le classe
// en ECARTE — jamais en echec, qui ferait chercher une panne.
func TestCuireUnMatch_RefusNonFinaliseDeLEnfant_CompteEcarte(t *testing.T) {
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			// La frontiere de processus : seul le texte survit.
			return BuildOneResult{}, errors.New("enfant : cuisson m1 : " + filmcache.ErrFilmNonFinalise.Error())
		},
	}
	var b bilanCuisson
	cuireUnMatch(context.Background(), d, buildWork{matchID: "m1"}, &b, time.Minute)
	if b.ecartes != 1 || b.echecs != 0 {
		t.Errorf("bilan = %+v, attendu 1 ecarte et 0 echec", b)
	}
}
