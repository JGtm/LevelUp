//go:build cgo

package skill

// skill_v2_burst_failure_test.go — un cycle LUSR v2 dont l'écrivain partagé est
// indisponible, ou dont le contexte est annulé entre deux rafales, ne rend pas un succès :
// le recalcul d'une exclusion (RecomputeLUSRCanonicalWithAccess) disait « terminé » sur un
// état vide ou partiel (revue adversariale E10.1, constat C2).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"
)

var errWriterDown = errors.New("ecrivain partage indisponible")

// failingWriteAccess : Read sert le handle, Write échoue.
type failingWriteAccess struct{ db *sql.DB }

func (a failingWriteAccess) Read(context.Context) (*sql.DB, func(), error) {
	return a.db, func() {}, nil
}

func (a failingWriteAccess) Write(context.Context, string) (*sql.DB, func(), error) {
	return nil, nil, errWriterDown
}

func TestRunLUSRV2Shadow_EcrivainIndisponible_RemonteLErreur(t *testing.T) {
	db := openShadowTestDB(t)
	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		seedShadow2v2(t, db, fmt.Sprintf("mc%d", i), base.Add(time.Duration(i)*time.Hour))
	}

	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), nil, failingWriteAccess{db: db}, "owner")
	if !errors.Is(err, errWriterDown) {
		t.Fatalf("err = %v, attendu l'erreur de l'écrivain indisponible", err)
	}
	if processed != 0 {
		t.Errorf("processed = %d, attendu 0 (rien n'a pu être écrit)", processed)
	}
}

// cancelAfterFirstBurst : annule le contexte du cycle quand la première rafale est rendue.
type cancelAfterFirstBurst struct {
	pinned SharedAccessor
	cancel context.CancelFunc
}

func (a cancelAfterFirstBurst) Read(ctx context.Context) (*sql.DB, func(), error) {
	return a.pinned.Read(ctx)
}

func (a cancelAfterFirstBurst) Write(ctx context.Context, step string) (*sql.DB, func(), error) {
	db, release, err := a.pinned.Write(ctx, step)
	if err != nil {
		return nil, nil, err
	}
	return db, func() { release(); a.cancel() }, nil
}

func TestRunLUSRV2Shadow_ContexteAnnuleEntreDeuxRafales_RemonteLErreur(t *testing.T) {
	withBurstLimits(t, lusrBurstLimits{maxMatches: 1, maxHold: time.Hour, now: time.Now})
	db := openShadowTestDB(t)
	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		seedShadow2v2(t, db, fmt.Sprintf("mc%d", i), base.Add(time.Duration(i)*time.Hour))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processed, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, cancelAfterFirstBurst{newPinnedSharedAccessor(db), cancel}, "owner")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, attendu context.Canceled (2 matchs sur 3 restent à traiter)", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, attendu 1 (une rafale d'un match avant l'annulation)", processed)
	}
}
