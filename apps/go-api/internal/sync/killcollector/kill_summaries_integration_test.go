//go:build integration

package killcollector

// kill_summaries_integration_test.go — LE COLLECTEUR ÉCRIT LES RÉSUMÉS DE SA PASSE (ADR 0039).
//
// La passe du collecteur passe par `persist.KillSourcePersister.PersistPass`, donc par l'écrivain
// unique du journal et de ses résumés. Ce test le vérifie depuis le collecteur, sur les
// migrations réelles : la tête résume la passe que la vue sert, et une erreur base pendant
// l'écriture des résumés n'en laisse rien (le match reste à collecter).

import (
	"context"
	"testing"

	"levelup/go-api/internal/sync/testutil"
)

func TestCollecteurCredit_EcritLesResumesDeSaPasse(t *testing.T) {
	ctx := context.Background()
	db := openSharedTestDB(t)
	insererEvenements(t, db, "m-resume", duelSimple())
	if _, morts, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(ctx, "m-resume"); err != nil || morts != 3 {
		t.Fatalf("CollectMatch : morts=%d err=%v", morts, err)
	}
	testutil.ExigerResumeDeLaPasseCourante(t, db, "m-resume")

	// Seconde passe sur le même match : la tête suit la nouvelle passe.
	if _, _, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(ctx, "m-resume"); err != nil {
		t.Fatalf("seconde passe : %v", err)
	}
	testutil.ExigerResumeDeLaPasseCourante(t, db, "m-resume")
}

func TestCollecteurCredit_ErreurDesResumesAnnuleLaPasse(t *testing.T) {
	ctx := context.Background()
	db := openSharedTestDB(t)
	insererEvenements(t, db, "m-resume-ko", duelSimple())
	testutil.CasserLaTableDesTetes(t, db)
	if _, _, err := NewCreditCollector(db, sharedWriter(db)).CollectMatch(ctx, "m-resume-ko"); err == nil {
		t.Fatal("CollectMatch sans erreur alors que la tête n'a pas pu s'écrire")
	}
	testutil.ExigerAucunePasse(t, db, "m-resume-ko")
}
