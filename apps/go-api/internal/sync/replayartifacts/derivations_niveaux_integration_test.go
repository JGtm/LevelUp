//go:build integration

package replayartifacts

// derivations_niveaux_integration_test.go — LA PANNE DES NIVEAUX DE SOCLE, PAR LE POINT D'ENTREE
// REEL (lot L4.1 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// Du 2026-09-14 au 2026-09-27, `Deriver` ne transmettait aucun segment de lecture aux familles :
// les niveaux d'armes rendaient « identites illisibles » a chaque passe, n'ecrivaient rien, et
// la marque de derivation se posait quand meme. Les tests de `padtiers_integration_test.go`
// appelaient la famille avec des `Deps` completes, a la main : ils ne pouvaient pas voir que le
// point d'entree ne les lui donnait pas. Ceux-ci passent par `Deriver`.

import (
	"context"
	"database/sql"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

// cheminDuLot : l'artefact que `monterLeLot` a pose, a sa place canonique.
func cheminDuLot(d Deps, matchID string) string {
	return titlePkg.NewPathResolver(d.RepoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
}

// TestDeriver_NiveauxSansSegmentDeLecture_NeMarquePas — LA PANNE. Sans segment de lecture, rien
// n'est ecrit (on ne devine pas le mode d'un match) ; avant le correctif la marque se posait
// quand meme, et le rattrapage ne reprenait jamais le match.
func TestDeriver_NiveauxSansSegmentDeLecture_NeMarquePas(t *testing.T) {
	db, d, _ := monterLeLot(t, "panne1")
	chemin := cheminDuLot(d, "panne1")

	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, Gamertag: d.Gamertag,
		AcquireWriter: d.AcquireWriter,
	}, []ArtefactRange{{MatchID: "panne1", Path: chemin}})

	if n := lignesDeNiveauxEnBase(t, db); n != 0 {
		t.Errorf("%d ligne(s) de niveaux ecrite(s) sans identite de match", n)
	}
	if replaybuild.DerivationsUpToDate(chemin) {
		t.Fatal("marque de derivation posee alors que les niveaux d'armes n'ont pas ete ecrits : " +
			"le rattrapage ne reprendrait jamais ce match (panne du 2026-09-14 au 2026-09-27)")
	}
}

// TestDeriver_NiveauxProjetesAuFilDeLEau — le chemin nominal par le point d'entree : les niveaux
// sont ecrits, la passe est marquee, et l'identite est lue AVANT que le segment d'ecriture de la
// passe soit tenu (en B-swap, une lecture sous ce segment attend un swap qu'il empeche).
func TestDeriver_NiveauxProjetesAuFilDeLEau(t *testing.T) {
	db, d, _ := monterLeLot(t, "nominal1")
	chemin := cheminDuLot(d, "nominal1")

	tenu, lectures, lecturesSousWriter := false, 0, 0
	withRead := func(_ context.Context, _ string, fn func(*sql.DB)) {
		lectures++
		if tenu {
			lecturesSousWriter++
		}
		fn(db)
	}
	acquire := func(context.Context) (*sql.DB, func(), error) {
		tenu = true
		return db, func() { tenu = false }, nil
	}
	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, Gamertag: d.Gamertag,
		WithRead: withRead, AcquireWriter: acquire,
	}, []ArtefactRange{{MatchID: "nominal1", Path: chemin}})

	if n := lignesDeNiveauxEnBase(t, db); n == 0 {
		t.Fatal("aucune ligne de niveaux ecrite par Deriver alors que le segment de lecture est cable")
	}
	if lectures == 0 {
		t.Error("aucune lecture d'identite : le segment de lecture n'est pas transmis aux familles")
	}
	if lecturesSousWriter != 0 {
		t.Errorf("%d lecture(s) ouverte(s) pendant que le segment d'ecriture est tenu : en B-swap "+
			"elle(s) attendrai(en)t jusqu'a ErrSwapTimeout", lecturesSousWriter)
	}
	if !replaybuild.DerivationsUpToDate(chemin) {
		t.Error("marque absente alors que toutes les familles ont pu ecrire")
	}
}

// TestRattrapage_TransmetLeSegmentDeLecture — le RATTRAPAGE (troisième appelant de Deriver) doit
// transmettre son segment de lecture : sans lui, ses cinq matchs les plus récents restaient non
// marqués et re-dérivés à chaque cycle, sans niveaux (revue du lot L4, constat C-2).
func TestRattrapage_TransmetLeSegmentDeLecture(t *testing.T) {
	db, d, _ := monterLeLot(t, "rattrape1")
	chemin := cheminDuLot(d, "rattrape1")

	rattraperDerivations(context.Background(), d)

	if n := lignesDeNiveauxEnBase(t, db); n == 0 {
		t.Fatal("le rattrapage n'a écrit aucun niveau d'arme : son segment de lecture n'arrive pas aux familles")
	}
	if !replaybuild.DerivationsUpToDate(chemin) {
		t.Error("le rattrapage n'a pas marqué un match dont les familles ont toutes écrit")
	}
}
