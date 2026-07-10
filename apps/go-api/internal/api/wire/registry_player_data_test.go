package wire

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/platform/duckdb"
)

// haloPlayerBuilder reproduit la factory player-scoped Halo enregistrée au boot
// (server.go) : adapter avec CareerRepo lié → career.progression supported.
func haloPlayerBuilder() func(*duckdb.PlayerDB) games.TitleDataAdapter {
	return func(pdb *duckdb.PlayerDB) games.TitleDataAdapter {
		return halo_games.NewDataAdapter(duckdb.NewCareerRepo(pdb), slog.Default())
	}
}

// TestPlayerDataBuilders_Routing — MT-09 : la factory route par SLUG (lookup de
// map, pas de comparaison littérale). Halo conserve sa parité (career.progression
// supported = CareerRepo lié, vs l'adapter resolver à career=nil) ; un 2e titre
// enregistré route ; un titre sans builder et un pdb nil dégradent en nil.
func TestPlayerDataBuilders_Routing(t *testing.T) {
	reg := &ServiceRegistry{}
	reg.RegisterPlayerDataBuilder(titlePkg.DefaultSlug, haloPlayerBuilder())

	syntheticCalled := false
	reg.RegisterPlayerDataBuilder("synthetic_b", func(*duckdb.PlayerDB) games.TitleDataAdapter {
		syntheticCalled = true
		return nil
	})

	// Parité Halo : adapter non-nil + career.progression supported (CareerRepo lié).
	a := reg.dataAdapterForPDB(&duckdb.PlayerDB{TitleSlug: titlePkg.DefaultSlug})
	if a == nil {
		t.Fatal("halo : dataAdapterForPDB a retourné nil")
	}
	if got := a.Capabilities()[games.CapCareerProgression]; got != games.CapSupported {
		t.Errorf("parité halo : career.progression = %q, want supported (CareerRepo lié)", got)
	}

	// Un 2e titre enregistré route via lookup (pas de gate par slug).
	reg.dataAdapterForPDB(&duckdb.PlayerDB{TitleSlug: "synthetic_b"})
	if !syntheticCalled {
		t.Error("synthetic_b : le builder enregistré n'a pas été appelé (routing cassé)")
	}

	// Titre sans builder → nil (dégradation propre).
	if reg.dataAdapterForPDB(&duckdb.PlayerDB{TitleSlug: "inconnu_xyz"}) != nil {
		t.Error("titre sans builder : doit retourner nil")
	}
	// pdb nil → nil.
	if reg.dataAdapterForPDB(nil) != nil {
		t.Error("pdb nil : doit retourner nil")
	}
}

// TestTitleDataAdapter_LookupAndDegradation — MT-09 : la factory exportée résout
// le pdb puis route par builder ; titre sans builder → ErrTitleNotResolved.
func TestTitleDataAdapter_LookupAndDegradation(t *testing.T) {
	reg := &ServiceRegistry{
		resolve: func(_ context.Context, slug string) (*duckdb.PlayerDB, error) {
			return &duckdb.PlayerDB{TitleSlug: slug}, nil
		},
	}
	reg.RegisterPlayerDataBuilder(titlePkg.DefaultSlug, haloPlayerBuilder())

	a, err := reg.TitleDataAdapter(context.Background(), titlePkg.DefaultSlug)
	if err != nil || a == nil {
		t.Fatalf("halo : attendu adapter non-nil sans erreur, got a=%v err=%v", a, err)
	}

	_, err = reg.TitleDataAdapter(context.Background(), "titre_sans_builder")
	if !errors.Is(err, games.ErrTitleNotResolved) {
		t.Errorf("titre sans builder : attendu ErrTitleNotResolved, got %v", err)
	}
}
