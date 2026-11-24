//go:build integration

package killcollector

// film_absent_reconcile_integration_test.go — la marque « film absent » ne survit pas à l'arrivée du
// film : la réconciliation la retire des matchs dont le film est complet à l'archive, la passe ne la
// pose plus sur un match dont le film est complet, et rien d'autre ne bouge.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmarchive"
	"levelup/go-api/internal/sync/matchflags"
)

func TestReconcileFilmAbsent_RetireLaMarqueDesFilmsCompletsSeulement(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	absent := int64(matchflags.MBitFilmAbsent)
	autres := int64(matchflags.MBitWeaponKills | matchflags.MBitEvents)
	const (
		complet    = "a1b2c3d1-0000-4000-8000-000000000001"
		perdu      = "a1b2c3d2-0000-4000-8000-000000000002"
		sansMarque = "a1b2c3d3-0000-4000-8000-000000000003"
		illisible  = "a1b2c3d4-0000-4000-8000-000000000004"
	)
	inscrireMatch(t, db, complet, t0, absent|autres)
	inscrireMatch(t, db, perdu, t0, absent)
	inscrireMatch(t, db, sansMarque, t0, autres)
	inscrireMatch(t, db, illisible, t0, absent)

	acquired, released := 0, 0
	acquire := func(context.Context) (*sql.DB, func(), error) {
		acquired++
		return db, func() { released++ }, nil
	}
	deps := ReconcileFilmAbsentDeps{
		TitleSlug: "halo_infinite", ArchiveRoot: "archive",
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: acquire,
		Complete: func(root, short string) (bool, error) {
			switch short {
			case titlePkg.FilmShortMatchID(complet), titlePkg.FilmShortMatchID(sansMarque):
				return true, nil
			case titlePkg.FilmShortMatchID(illisible):
				return false, errors.New("manifeste illisible")
			}
			return false, nil
		},
	}

	n, err := ReconcileFilmAbsent(context.Background(), deps)
	if err != nil || n != 1 {
		t.Fatalf("ReconcileFilmAbsent = %d, %v ; attendu 1 marque retirée", n, err)
	}
	if got := lireBits(t, db, complet); got != autres {
		t.Errorf("film complet : bits = %d, attendu %d (marque retirée, autres bits intacts)", got, autres)
	}
	if got := lireBits(t, db, perdu); got != absent {
		t.Errorf("film perdu : bits = %d, attendu %d (marque gardée)", got, absent)
	}
	if got := lireBits(t, db, illisible); got != absent {
		t.Errorf("état de l'archive illisible : bits = %d, attendu %d (marque gardée)", got, absent)
	}
	if got := lireBits(t, db, sansMarque); got != autres {
		t.Errorf("match sans marque : bits = %d, attendu %d", got, autres)
	}
	if acquired != 1 || released != 1 {
		t.Errorf("segment d'écriture pris %d fois, rendu %d fois, attendu 1 et 1", acquired, released)
	}

	// Idempotente, et sans écrivain quand il n'y a plus rien à retirer.
	n, err = ReconcileFilmAbsent(context.Background(), deps)
	if err != nil || n != 0 || acquired != 1 {
		t.Errorf("second passage = %d, %v, écrivain pris %d fois ; attendu 0, nil, 1", n, err, acquired)
	}
}

func TestMarquerFilm_NePosepasLaMarqueSurUnFilmCompletALArchive(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	inscrireMatch(t, db, "m-complet", t0, 0)
	inscrireMatch(t, db, "m-perdu", t0, 0)
	acquire := func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }

	c := NewKillSourceCollector(nil, nil, acquire, nil, 0).
		AvecFilmComplet(func(id string) bool { return id == "m-complet" })
	c.marquerFilm(context.Background(), "m-complet", OutcomeNoFilm, 0)
	c.marquerFilm(context.Background(), "m-perdu", OutcomeNoFilm, 0)

	if got := lireBits(t, db, "m-complet"); got != 0 {
		t.Errorf("film complet à l'archive : bits = %d, attendu 0 (jamais marqué absent)", got)
	}
	if got := lireBits(t, db, "m-perdu"); got != int64(matchflags.MBitFilmAbsent) {
		t.Errorf("film réellement absent : bits = %d, attendu la marque", got)
	}
}

// Le câblage de la passe : la même lecture rend le backlog et les marques, la réconciliation retire la
// marque du match dont le film est RÉELLEMENT complet à l'archive (film écrit par filmarchive.Write), et
// ne se rejoue pas dans le même processus.
func TestRunPostSync_ReconcilieLesMarquesAvecLArchive(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	absent := int64(matchflags.MBitFilmAbsent)
	const (
		archive = "b2b2b2b2-0000-4000-8000-000000000001"
		perdu   = "b3b3b3b3-0000-4000-8000-000000000002"
	)
	inscrireMatch(t, db, archive, t0, absent)
	inscrireMatch(t, db, perdu, t0, absent)

	racine := t.TempDir()
	if err := filmarchive.Write(context.Background(), racine, titlePkg.FilmShortMatchID(archive), []filmarchive.WriteChunk{
		{Index: 0, ChunkType: 1, Data: []byte("entete")},
		{Index: 1, ChunkType: 2, Data: []byte("replication")},
		{Index: 2, ChunkType: 3, Data: []byte("temps forts")},
	}); err != nil {
		t.Fatalf("film archivé: %v", err)
	}

	h := NewPostSyncHook(racineDepot(t), 0)
	h.archiveRoot = func() string { return racine }
	d := PostSyncDeps{
		TitleSlug:     "titre_test_reconcile",
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
	}

	_, _, marques := h.lireArriere(context.Background(), d)
	if len(marques) != 2 {
		t.Fatalf("marques lues = %v, attendu les deux matchs marqués", marques)
	}
	reconcilierFilmsAbsents(context.Background(), d, h.racineVoulue(nil), marques)
	if got := lireBits(t, db, archive); got != 0 {
		t.Errorf("film complet à l'archive : bits = %d, attendu 0 (marque retirée)", got)
	}
	if got := lireBits(t, db, perdu); got != absent {
		t.Errorf("film absent : bits = %d, attendu la marque gardée", got)
	}
	if _, _, again := h.lireArriere(context.Background(), d); again != nil {
		t.Errorf("la réconciliation se rejoue dans le même processus : marques relues = %v", again)
	}
}
