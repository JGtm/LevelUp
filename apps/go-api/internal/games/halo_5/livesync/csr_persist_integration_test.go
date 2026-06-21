//go:build integration

// Package livesync — csr_persist_integration_test.go : round-trip persist→read du
// CSR Halo 5 sur une DuckDB in-memory. Vérifie que les lignes mappées depuis le
// service record arena sont écrites dans player_csr_snapshots (append-only) puis
// relues via la vue _latest filtrée par la saison lifetime — la chaîne exacte que
// la page Carrière emprunte (GetCSRSnapshots WHERE season_id = h5-lifetime).
//
// Tag integration : importe DuckDB (CGO) — ne compile pas sur Windows par défaut.
package livesync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
)

func TestPersistAndReadH5CSR_RoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma player (crée player_csr_snapshots + vue _latest).
	if err := syncpkg.EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}

	// Service record arena : 2 playlists (une classée Onyx, une en placement).
	resp := arenaResp([]halo5.H5ArenaPlaylistStat{
		{
			PlaylistId: "pl-onyx",
			Csr:        csrPtr(5, 2, 1700), // Onyx (sub ignoré)
			HighestCsr: csrPtr(5, 1, 1800),
		},
		{
			PlaylistId:             "pl-placement",
			MeasurementMatchesLeft: 6,
			Csr:                    nil,
			HighestCsr:             nil,
		},
	})

	csrs := mapH5ArenaToPlaylistCSRs(resp)
	if len(csrs) != 2 {
		t.Fatalf("mapper: attendu 2 lignes, obtenu %d", len(csrs))
	}

	// Persistance append-only avec la saison lifetime.
	n, err := syncpkg.SaveCSRSnapshots(ctx, db, csrs, h5LifetimeSeasonID)
	if err != nil {
		t.Fatalf("SaveCSRSnapshots: %v", err)
	}
	if n != 2 {
		t.Fatalf("SaveCSRSnapshots: %d lignes écrites, want 2", n)
	}

	// Relecture via la vue _latest, filtrée par la saison lifetime (chaîne lecture).
	type row struct {
		tier      string
		subTier   int
		value     float64
		remaining int
	}
	got := map[string]row{}
	rows, err := db.QueryContext(ctx, `
		SELECT playlist_id, COALESCE(current_tier,''), COALESCE(current_sub_tier,0),
		       COALESCE(current_value,0), COALESCE(current_measurement_remaining,0)
		FROM player_csr_snapshots_latest
		WHERE season_id = ?`, h5LifetimeSeasonID)
	if err != nil {
		t.Fatalf("query latest: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var r row
		if err := rows.Scan(&id, &r.tier, &r.subTier, &r.value, &r.remaining); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("relu %d lignes, want 2 (season_id filtré)", len(got))
	}
	if onyx := got["pl-onyx"]; onyx.tier != "Onyx" || onyx.subTier != 0 || onyx.value != 1700 {
		t.Errorf("pl-onyx: got %+v, want tier=Onyx sub=0 val=1700", onyx)
	}
	if pl := got["pl-placement"]; pl.tier != "" || pl.remaining != 6 {
		t.Errorf("pl-placement: got %+v, want tier='' remaining=6", pl)
	}
}
