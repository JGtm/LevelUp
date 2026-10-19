//go:build integration

package sync

// citations_segments_lecture_integration_test.go — LE PAS DES CITATIONS DU POST-SYNC NE TIENT PAS
// LA BASE PARTAGÉE PENDANT TOUT LE LOT (D-15, 2026-10-19).
//
// Un lecteur de la base partagée en vol empêche toute prise d'écriture (le provider attend son
// relâchement, puis abandonne au bout de son délai de vidange). Sur le serveur local, juste après
// une recuisson, le recalcul des citations des matchs au film plus récent a porté sur 574 à 1 158
// matchs par joueur, tenus sous UN seul lecteur pendant environ 90 s : toutes les écritures de la
// base partagée échouaient pendant ce temps. Le pas relâche désormais la base entre deux matchs.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// espionLectureShared compte les lectures ouvertes sur la base partagée et le plus grand nombre de
// lectures tenues en même temps.
type espionLectureShared struct {
	db                   *sql.DB
	ouvertes, enVol, max int
}

func (e *espionLectureShared) lire(context.Context) (*sql.DB, func(), error) {
	e.ouvertes++
	e.enVol++
	if e.enVol > e.max {
		e.max = e.enVol
	}
	return e.db, func() { e.enVol-- }, nil
}

func TestRunCitations_RelacheLaBasePartageeEntreDeuxMatchs(t *testing.T) {
	f := buildPipelineFixture(t)
	ctx := context.Background()

	candidats, err := selectMatchesForCitations(ctx, f.player, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidats) < 3 {
		t.Fatalf("fixture : %d match(s) candidat(s), il en faut au moins 3 pour le test", len(candidats))
	}

	espion := &espionLectureShared{db: f.shared}
	shared := NewBurstSharedAccess(func(context.Context) (*sql.DB, func(), error) {
		return nil, nil, errors.New("le pas des citations n'écrit pas dans la base partagée")
	}, espion.lire, "test")
	e := &SyncEngine{metaDB: f.metadata, xuid: fixXUID, gamertag: fixGamertag, titleSlug: "halo_infinite"}

	var r domain.PostSyncResult
	e.runCitations(ctx, f.player, shared, &r)

	if r.CitationsComputed != len(candidats) {
		t.Fatalf("citations calculées pour %d match(s), attendu %d (erreurs : %v)",
			r.CitationsComputed, len(candidats), r.FatalErrors)
	}
	if realCitationValue(t, f.player, fixM1, "bulltrue") == 0 {
		t.Fatal("citation bulltrue absente pour le match au film : le calcul n'a pas eu lieu")
	}
	if espion.ouvertes <= len(candidats) {
		t.Errorf("%d lecture(s) de la base partagée pour %d matchs : la base reste tenue pendant "+
			"plusieurs matchs (attendu au moins une lecture par match, plus la sélection)",
			espion.ouvertes, len(candidats))
	}
	if espion.max != 1 || espion.enVol != 0 {
		t.Errorf("lectures tenues en même temps : %d (attendu 1), encore en vol à la fin : %d",
			espion.max, espion.enVol)
	}
}
