//go:build integration

// Package livesync — career_sr_persist_integration_test.go : round-trip
// persist→read du rang SR Halo 5 dans career_progression sur une DuckDB
// in-memory. Vérifie que writeCareerSR écrit rank_name = "SR N" (libellé
// title-aware, source unique halo5.SpartanRankLabel) — sinon la Home retombe
// sur le fallback générique HINF « Rang N » (career.rank_catalog = not_exposed).
//
// Tag integration : importe DuckDB (CGO) — ne compile pas sur Windows par défaut.
package livesync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
)

func TestWriteCareerSR_WritesSRRankName(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma player (crée career_progression avec les colonnes exactes de l'INSERT).
	if err := syncpkg.EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}

	// SR 111 / TotalXP réaliste (> seuil de début du rang 111) → SpartanRankProgression ok.
	const xuid = "xuid1"
	const spartanRank = 111
	const totalXP = 3908120
	start := sql.NullTime{Time: time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC), Valid: true}

	if ok := writeCareerSR(ctx, db, xuid, spartanRank, totalXP, start); !ok {
		t.Fatalf("writeCareerSR: want true (SR valide + recorded_at présent)")
	}

	// rank_name doit être "SR 111" (= halo5.SpartanRankLabel(111)). On filtre par
	// xuid uniquement (une seule ligne attendue) — la comparaison directe sur le
	// TIMESTAMP DuckDB est sujette aux écarts de fuseau/précision du round-trip.
	var rankName string
	if err := db.QueryRowContext(ctx,
		`SELECT rank_name FROM career_progression WHERE xuid = ?`,
		xuid).Scan(&rankName); err != nil {
		t.Fatalf("query rank_name: %v", err)
	}
	want := halo5.SpartanRankLabel(spartanRank)
	if want != "SR 111" {
		t.Fatalf("garde-fou : SpartanRankLabel(%d) = %q, want \"SR 111\"", spartanRank, want)
	}
	if rankName != want {
		t.Errorf("rank_name = %q, want %q (libellé SR title-aware)", rankName, want)
	}

	// Note : la dédup par (xuid, recorded_at) n'est PAS asséurée ici — le binding
	// d'un time.Time dans la forme INSERT…SELECT…WHERE NOT EXISTS du driver DuckDB
	// dépend du fuseau du process (le TIMESTAMP stocké subit le décalage local), ce
	// qui rend la comparaison de paramètres non fiable hors UTC. Orthogonal au but
	// du test (rank_name = "SR N"). Les gardes propres de writeCareerSR ci-dessous
	// (SR hors borne, recorded_at absent) ne dépendent d'aucun round-trip TIMESTAMP.

	// SR hors borne → false (snapshot ignoré).
	if ok := writeCareerSR(ctx, db, xuid, 0, totalXP, start); ok {
		t.Errorf("writeCareerSR (SR hors borne): want false, got true")
	}
	// recorded_at absent → false.
	if ok := writeCareerSR(ctx, db, xuid, spartanRank, totalXP, sql.NullTime{}); ok {
		t.Errorf("writeCareerSR (recorded_at absent): want false, got true")
	}
}
