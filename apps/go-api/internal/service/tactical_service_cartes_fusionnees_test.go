package service

// tactical_service_cartes_fusionnees_test.go — LE PLANCHER DE MATCHS S'APPLIQUE AU COMPTE DE LA
// CARTE, pas a une part de ce compte. Bout en bout service + depot DuckDB `:memory:` : le
// registre porte, pour une meme carte, des matchs au vrai nom, au nom NULL et au map_id
// recopie ; scindee par nom, la carte tombait en trois vignettes toutes sous le plancher.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

func TestMapsPlayed_PlancherSurLeCompteFusionne(t *testing.T) {
	const (
		xuid  = "2533274000000901"
		carte = "map_illusion"
	)
	shared, err := duckdb.OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	_ = migration.All()
	if err := migration.RunForDB(shared.SQLDb(), migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	// 12 matchs d'une carte : 6 au vrai nom, 3 a NULL, 3 au map_id recopie. Aucune part
	// n'atteint le plancher (10) ; la carte, si.
	ctx := context.Background()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	ids := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		var nom any = "Illusion"
		switch {
		case i >= 9:
			nom = carte
		case i >= 6:
			nom = nil
		}
		id := "f" + string(rune('a'+i))
		ids = append(ids, id)
		start := base.Add(time.Duration(i) * time.Hour)
		if _, err := shared.Exec(ctx, `INSERT INTO match_registry
			(match_id, map_id, map_name, start_time, start_time_utc, playlist_name, pair_name)
			VALUES (?, ?, ?, ?, ?, 'Ranked Arena', 'Arena:Slayer')`, id, carte, nom, start, start); err != nil {
			t.Fatalf("registre %s: %v", id, err)
		}
		if _, err := shared.Exec(ctx, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
			VALUES (?, ?, ?, 0, ?)`, id, xuid, xuid, domain.OutcomeWin); err != nil {
			t.Fatalf("participant %s: %v", id, err)
		}
	}

	pdb := &duckdb.PlayerDB{
		Shared:       shared,
		SharedReader: duckdb.LegacySharedReader(shared),
		XUID:         xuid,
		TitleSlug:    titlepkg.DefaultSlug,
	}
	page, err := NewTacticalService(duckdb.NewTacticalRepo(pdb), capsPositionsSeules(), xuid).
		MapsPlayed(ctx, domain.TacticalScope{MatchIDs: ids})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(page.Cartes) != 1 {
		t.Fatalf("vignettes = %d, want 1 (une carte) : %+v", len(page.Cartes), page.Cartes)
	}
	got := page.Cartes[0]
	if got.Matchs != 12 || got.SousPlancher {
		t.Errorf("vignette = %+v, want 12 matchs et ouvrable (plancher %d sur le compte fusionne)",
			got, domain.PlancherMatchsParCarte)
	}
	if got.MapName != "Illusion" {
		t.Errorf("nom = %q, want %q", got.MapName, "Illusion")
	}
}
