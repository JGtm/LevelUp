package replayartifacts

// cuisson_finalise_test.go — UN FILM NON FINALISE N'EST NI ARCHIVE NI CUIT : LA CUISSON EST
// REPORTEE AU CYCLE SUIVANT, COMPTEE, JOURNALISEE EN INFO — ET EN WARN, SOUS UN COMPTEUR DISTINCT,
// QUAND LE FILM RESTE NON FINALISE AU-DELA DU DELAI DE FINALISATION (lot L3, 2026-09-23 ; reprise
// apres revue adverse, constats L3-R1, L3-R3 et L3-R6).
//
// 23 matchs sur 96 sont detectes moins de 60 s apres leur fin (rapport ctf_ab526724 §2.3) : le
// serveur n'a pas encore publie leur morceau des temps forts. Ce n'est ni un film absent ni une
// panne — c'est un film qu'il faut reprendre une minute plus tard. Un film qui ne l'est TOUJOURS
// pas un quart d'heure apres, en revanche, est une derive a rendre visible.

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
	"levelup/go-api/internal/replaybuild"
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

// horlogeDeTest remplace la memoire des attentes par une memoire neuve a horloge tenue, et la
// restaure a la fin du test. Rend le pointeur sur l'instant courant, que le test avance.
func horlogeDeTest(t *testing.T) *time.Time {
	t.Helper()
	instant := time.Date(2026, 9, 23, 21, 35, 50, 0, time.UTC)
	avant := attentes
	attentes = nouvellesAttentes(func() time.Time { return instant })
	t.Cleanup(func() { attentes = avant })
	return &instant
}

// compteur lit un compteur du titre par defaut.
func compteur(nom string) int64 {
	return observability.LoadCounterT(ctxkeys.TitleSlug(context.Background()), nom)
}

// TestPersistFilmToCache_NonFinalise_ReporteEtCompte : les deux formes du refus reportent le
// film (ni persiste ni disponible, donc pas de cuisson, et REPORTE — pas « sans film »), et
// chacune se compte.
func TestPersistFilmToCache_NonFinalise_ReporteEtCompte(t *testing.T) {
	ctx := context.Background()
	horlogeDeTest(t)
	for _, liste := range []bool{false, true} {
		t.Run(fmt.Sprintf("liste=%v", liste), func(t *testing.T) {
			racine := t.TempDir()
			d := Deps{Gamertag: "g", TitleSlug: titlePkg.DefaultSlug, CacheRoot: racine,
				Fetcher: fetcherNonFinalise{liste: liste}}
			avant := compteur(CompteurFilmsNonFinalises)

			r := persistFilmToCache(ctx, d, "m1")
			if r != (resultatFilm{reporte: true}) {
				t.Fatalf("film non finalise : %+v, attendu {reporte: true} seul", r)
			}
			if n := compteur(CompteurFilmsNonFinalises) - avant; n != 1 {
				t.Errorf("%s : +%d, attendu +1 — le report est invisible", CompteurFilmsNonFinalises, n)
			}
			if _, found, err := filmcache.Open(racine, titlePkg.FilmShortMatchID("m1")); found || err != nil {
				t.Errorf("un manifeste a ete valide pour un film non finalise (found=%v, err=%v)", found, err)
			}
		})
	}
}

// TestBuildAll_NonFinalise_ReporteSansCuisson : le cycle ne cuit pas un film non finalise, ne le
// compte ni en echec ni en « sans film », et passe au suivant — le report est un POSTE DU BILAN.
//
// AVANT LA REPRISE DU LOT CE TEST ETAIT VERT SANS LE CORRECTIF (constat L3-R6 : un Fetcher en
// erreur donnait deja « ni cuisson ni echec »). Ce qui le rend discriminant est le poste
// `reportes` et le compteur : la mutation « reporterNonFinalise rend toujours faux » range les
// deux films en `sansFilm` et ne compte rien — rouge.
func TestBuildAll_NonFinalise_ReporteSansCuisson(t *testing.T) {
	horlogeDeTest(t)
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: fetcherNonFinalise{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			appels++
			return BuildOneResult{}, errors.New("aucune cuisson ne doit partir")
		},
	}
	avant := compteur(CompteurFilmsNonFinalises)
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})
	if appels != 0 || b.echecs != 0 || b.construits != 0 || b.filmsSauves != 0 || b.sansFilm != 0 {
		t.Errorf("bilan = %+v (%d cuisson(s)), attendu aucune cuisson, aucun echec, aucun « sans film »", b, appels)
	}
	if b.reportes != 2 || b.traites() != 2 {
		t.Errorf("reportes = %d, traites = %d, attendu 2 et 2", b.reportes, b.traites())
	}
	if n := compteur(CompteurFilmsNonFinalises) - avant; n != 2 {
		t.Errorf("%s : +%d, attendu +2", CompteurFilmsNonFinalises, n)
	}
}

// TestReporterNonFinalise_HorsDelai_SignaleUneFoisParFilm : LE CONSTAT L3-R1. Un film encore non
// finalise au-dela de [DelaiDeFinalisation] n'est plus un film frais : il est compte UNE fois
// dans le compteur distinct, et chaque refus suivant reste un WARN. Un film archive entre-temps
// clot son attente — refuse de nouveau, il repart d'une attente neuve.
func TestReporterNonFinalise_HorsDelai_SignaleUneFoisParFilm(t *testing.T) {
	ctx := context.Background()
	instant := horlogeDeTest(t)
	d := Deps{Gamertag: "g"}
	err := fmt.Errorf("GetFilmChunks(m) : %w", filmcache.ErrFilmNonFinalise)
	horsDelai := func() int64 { return compteur(CompteurFilmsNonFinalisesHorsDelai) }
	depart := horsDelai()

	if !reporterNonFinalise(ctx, d, "m1", err) {
		t.Fatal("un refus de finalisation n'est pas reporte")
	}
	*instant = instant.Add(DelaiDeFinalisation)
	reporterNonFinalise(ctx, d, "m1", err)
	if n := horsDelai() - depart; n != 0 {
		t.Fatalf("compte hors delai a l'echeance exacte : +%d, attendu 0 (un film frais)", n)
	}
	*instant = instant.Add(time.Second)
	reporterNonFinalise(ctx, d, "m1", err)
	if n := horsDelai() - depart; n != 1 {
		t.Fatalf("compte hors delai au-dela du delai : +%d, attendu +1", n)
	}
	reporterNonFinalise(ctx, d, "m1", err)
	if n := horsDelai() - depart; n != 1 {
		t.Fatalf("le meme film recompte : +%d, attendu +1 (une fois par film)", n)
	}
	// Un AUTRE film, refuse pour la premiere fois maintenant, est frais.
	reporterNonFinalise(ctx, d, "m2", err)
	if n := horsDelai() - depart; n != 1 {
		t.Fatalf("un film frais compte hors delai : +%d, attendu +1", n)
	}
	// Archive : l'attente de m1 est close.
	attentes.oublier(cleDAttente(ctx, "m1"))
	reporterNonFinalise(ctx, d, "m1", err)
	if n := horsDelai() - depart; n != 1 {
		t.Fatalf("un film archive puis refuse garde son ancienne attente : +%d, attendu +1", n)
	}
}

// TestPersistFilmToCache_FilmArchive_CloreLAttente : l'archivage d'un film finalise clot son
// attente — sans quoi un film refuse une fois, archive, puis rejoue plus tard partirait hors
// delai sur une attente perimee.
func TestPersistFilmToCache_FilmArchive_CloreLAttente(t *testing.T) {
	ctx := context.Background()
	instant := horlogeDeTest(t)
	cle := cleDAttente(ctx, "m1")
	reporterNonFinalise(ctx, Deps{}, "m1", filmcache.ErrFilmNonFinalise)
	*instant = instant.Add(time.Minute)

	d := Deps{Gamertag: "g", TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: fetcherFinalise{}}
	if r := persistFilmToCache(ctx, d, "m1"); r != (resultatFilm{sauve: true, dispo: true}) {
		t.Fatalf("film finalise : %+v, attendu {sauve, dispo}", r)
	}
	attentes.mu.Lock()
	_, reste := attentes.premier[cle]
	attentes.mu.Unlock()
	if reste {
		t.Error("l'attente d'un film archive n'est pas close")
	}
}

// fetcherFinalise : le client Halo face a un film finalise (un morceau de chaque type).
type fetcherFinalise struct{}

func (fetcherFinalise) GetFilmChunks(context.Context, string) ([]haloclient.FilmChunk, bool, error) {
	return []haloclient.FilmChunk{
		{Index: 0, ChunkType: haloclient.FilmChunkTypeHeader, Data: []byte("h")},
		{Index: 1, ChunkType: haloclient.FilmChunkTypeReplicationData, Data: []byte("r")},
		{Index: 2, ChunkType: haloclient.FilmChunkTypeHighlightEvents, Data: []byte("t")},
	}, true, nil
}

// refusDontLeTexteNeDitRien : une erreur que `errors.Is` reconnait comme `cible` mais dont le
// texte ne la cite pas. C'est la forme qui separe un classement par TYPE d'un classement par
// TEXTE : seul le premier la range correctement.
type refusDontLeTexteNeDitRien struct{ cible error }

func (r refusDontLeTexteNeDitRien) Error() string        { return "refus de l'enfant" }
func (r refusDontLeTexteNeDitRien) Is(target error) bool { return target == r.cible }

// TestReplayArtifacts_FilmNonFinaliseReporteSansEchec : L3.5 vu du cycle (lot J2.12, DT-5). La
// cuisson (dans un ENFANT) refuse un film non finalise ; la raison traverse le tube en JETON et
// `replaychild` rend l'erreur typee enveloppee. Le cycle la classe par `errors.Is` en ECARTE —
// jamais en echec, qui ferait chercher une panne. Il ne lit pas le texte : l'erreur injectee ne
// cite pas la sentinelle.
func TestReplayArtifacts_FilmNonFinaliseReporteSansEchec(t *testing.T) {
	for _, cible := range []error{filmcache.ErrFilmNonFinalise, replaybuild.ErrUnknownFilmKey} {
		d := Deps{
			RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
			BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
				return BuildOneResult{}, fmt.Errorf("cuisson m1 : %w", refusDontLeTexteNeDitRien{cible})
			},
		}
		var b bilanCuisson
		cuireUnMatch(context.Background(), d, buildWork{matchID: "m1"}, &b, time.Minute)
		if b.ecartes != 1 || b.echecs != 0 {
			t.Errorf("%v : bilan = %+v, attendu 1 ecarte et 0 echec", cible, b)
		}
	}
}
