//go:build integration

package killcollector

// film_absent_mark_integration_test.go — le marqueur « film absent » posé par la génération à la demande : même bit,
// même requête que la passe film, idempotent, sans effet sur les autres bits.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/sync/matchflags"
)

func lireBits(t *testing.T, db *sql.DB, id string) int64 {
	t.Helper()
	var bits int64
	if err := db.QueryRowContext(context.Background(),
		`SELECT COALESCE(backfill_completed, 0) FROM match_registry WHERE match_id = ?`, id).Scan(&bits); err != nil {
		t.Fatalf("lecture backfill_completed(%s): %v", id, err)
	}
	return bits
}

func TestMarquerFilmAbsent_PoseLeBitSansToucherLesAutres(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	autres := int64(matchflags.MBitWeaponKills | matchflags.MBitEvents)
	inscrireMatch(t, db, "avec-bits", t0, autres)
	inscrireMatch(t, db, "voisin", t0, 0)
	released := 0
	acquire := func(context.Context) (*sql.DB, func(), error) {
		return db, func() { released++ }, nil
	}

	for i := 0; i < 2; i++ { // second appel : idempotent
		if err := MarquerFilmAbsent(context.Background(), acquire, "avec-bits"); err != nil {
			t.Fatalf("appel %d: %v", i+1, err)
		}
	}
	if got, want := lireBits(t, db, "avec-bits"), autres|int64(matchflags.MBitFilmAbsent); got != want {
		t.Fatalf("bits = %d, attendu %d (bit film absent ajouté, les autres intacts)", got, want)
	}
	if got := lireBits(t, db, "voisin"); got != 0 {
		t.Fatalf("un autre match a été touché : bits = %d", got)
	}
	if released != 2 {
		t.Fatalf("le segment d'écriture doit être rendu à chaque appel : %d", released)
	}
}

func TestMarquerFilmAbsent_MatchInconnuEtEcrivainIndisponible(t *testing.T) {
	db := baseBacklog(t)
	acquire := func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
	if err := MarquerFilmAbsent(context.Background(), acquire, "inconnu"); err != nil {
		t.Fatalf("un match hors registre n'est pas une erreur : %v", err)
	}
	if err := MarquerFilmAbsent(context.Background(), nil, "x"); !errors.Is(err, ErrNoSharedWriter) {
		t.Fatalf("acquire nul : %v, attendu ErrNoSharedWriter", err)
	}
	boom := errors.New("lease refusé")
	failing := func(context.Context) (*sql.DB, func(), error) { return nil, nil, boom }
	if err := MarquerFilmAbsent(context.Background(), failing, "x"); !errors.Is(err, boom) {
		t.Fatalf("échec du lease : %v, attendu l'erreur enveloppée", err)
	}
}
