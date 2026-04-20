// Package duckdb — queries_media_test.go : tests unitaires des query builders médias.
package duckdb

import (
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// ─────────────────────────────────────────────────────────────────────────────
// BuildQ37MediaQuery
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildQ37MediaQuery_NoFilters(t *testing.T) {
	q, args := BuildQ37MediaQuery(domain.MediaFilters{}, 24, 0)

	if !strings.Contains(q, "WHERE") {
		t.Error("expected WHERE clause")
	}
	if !strings.Contains(q, "mf.status = 'active'") {
		t.Error("expected status = active filter")
	}
	if !strings.Contains(q, "ORDER BY mf.mtime DESC") {
		t.Errorf("expected default ORDER BY mf.mtime DESC, got: %s", q)
	}
	if !strings.Contains(q, "LIMIT ? OFFSET ?") {
		t.Error("expected LIMIT ? OFFSET ?")
	}
	// Les 2 derniers args doivent être limit=24, offset=0
	if len(args) != 2 {
		t.Fatalf("args = %v (len=%d), want [24 0]", args, len(args))
	}
	if args[0] != 24 || args[1] != 0 {
		t.Errorf("args = %v, want [24 0]", args)
	}
}

func TestBuildQ37MediaQuery_KindFilter(t *testing.T) {
	q, args := BuildQ37MediaQuery(domain.MediaFilters{KindFilter: "screenshot"}, 10, 0)

	if !strings.Contains(q, "mf.kind = ?") {
		t.Error("expected kind = ? clause")
	}
	// args: kind, limit, offset
	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
	if args[0] != "screenshot" {
		t.Errorf("args[0] = %v, want screenshot", args[0])
	}
	if args[1] != 10 || args[2] != 0 {
		t.Errorf("args[1:] = %v, want [10 0]", args[1:])
	}
}

func TestBuildQ37MediaQuery_LikedOnly(t *testing.T) {
	q, args := BuildQ37MediaQuery(domain.MediaFilters{LikedOnly: true}, 24, 0)

	if !strings.Contains(q, "COALESCE(mf.liked, FALSE) = TRUE") {
		t.Errorf("expected liked filter, got: %s", q)
	}
	// LikedOnly n'ajoute pas d'arg — args: limit, offset uniquement
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (LikedOnly n'ajoute pas d'arg)", len(args))
	}
}

func TestBuildQ37MediaQuery_MapFilter(t *testing.T) {
	q, args := BuildQ37MediaQuery(domain.MediaFilters{MapFilter: "Fragmentation"}, 24, 0)

	if !strings.Contains(q, "mr.map_name ILIKE ?") {
		t.Error("expected map_name ILIKE clause")
	}
	// Vérifier que le % wrapping est appliqué
	if len(args) < 1 {
		t.Fatal("expected at least 1 filter arg")
	}
	mapArg, ok := args[0].(string)
	if !ok || !strings.Contains(mapArg, "Fragmentation") {
		t.Errorf("map arg = %v, want %%Fragmentation%%", args[0])
	}
}

func TestBuildQ37MediaQuery_ModeFilter(t *testing.T) {
	q, args := BuildQ37MediaQuery(domain.MediaFilters{ModeFilter: "Slayer"}, 24, 0)

	if !strings.Contains(q, "mr.pair_name ILIKE ?") {
		t.Error("expected pair_name ILIKE clause")
	}
	if len(args) < 1 {
		t.Fatal("expected at least 1 filter arg")
	}
	modeArg, ok := args[0].(string)
	if !ok || !strings.Contains(modeArg, "Slayer") {
		t.Errorf("mode arg = %v, want %%Slayer%%", args[0])
	}
}

func TestBuildQ37MediaQuery_AllFilters_ArgOrder(t *testing.T) {
	f := domain.MediaFilters{
		KindFilter: "video",
		MapFilter:  "Recharge",
		ModeFilter: "CTF",
	}
	_, args := BuildQ37MediaQuery(f, 5, 10)

	// Ordre attendu : kind, map, mode, limit, offset
	if len(args) != 5 {
		t.Fatalf("args len = %d, want 5", len(args))
	}
	if args[0] != "video" {
		t.Errorf("args[0] = %v, want video", args[0])
	}
	// map et mode doivent être wrappés avec %
	mapArg, _ := args[1].(string)
	modeArg, _ := args[2].(string)
	if !strings.Contains(mapArg, "Recharge") {
		t.Errorf("args[1] = %v, want %%Recharge%%", args[1])
	}
	if !strings.Contains(modeArg, "CTF") {
		t.Errorf("args[2] = %v, want %%CTF%%", args[2])
	}
	if args[3] != 5 || args[4] != 10 {
		t.Errorf("args[3:] = %v, want [5 10]", args[3:])
	}
}

func TestBuildQ37MediaQuery_Sort_DateAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "date_asc"}, 24, 0)
	if !strings.Contains(q, "ORDER BY mf.mtime ASC") {
		t.Errorf("expected mtime ASC order, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_MapAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "map_asc"}, 24, 0)
	if !strings.Contains(q, "COALESCE(mr.map_name") {
		t.Errorf("expected map_name sort, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_ModeAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "mode_asc"}, 24, 0)
	if !strings.Contains(q, "COALESCE(mr.pair_name") {
		t.Errorf("expected pair_name sort, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_Unknown_FallsBackToDefault(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "random_invalid"}, 24, 0)
	if !strings.Contains(q, "ORDER BY mf.mtime DESC") {
		t.Errorf("expected fallback to mtime DESC, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_HasJoins(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{}, 24, 0)

	if !strings.Contains(q, "LEFT JOIN media_match_associations") {
		t.Error("expected JOIN media_match_associations")
	}
	if !strings.Contains(q, "LEFT JOIN shared.match_registry") {
		t.Error("expected JOIN shared.match_registry for map/mode enrichment")
	}
}

func TestBuildQ37MediaQuery_Pagination(t *testing.T) {
	_, args := BuildQ37MediaQuery(domain.MediaFilters{}, 12, 48)
	last2 := args[len(args)-2:]
	if last2[0] != 12 || last2[1] != 48 {
		t.Errorf("last args = %v, want [12 48]", last2)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildQ37MediaCountQuery
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildQ37MediaCountQuery_NoFilters(t *testing.T) {
	q, args := BuildQ37MediaCountQuery(domain.MediaFilters{})

	if !strings.Contains(q, "SELECT COUNT(*)") {
		t.Error("expected COUNT(*)")
	}
	if !strings.Contains(q, "mf.status = 'active'") {
		t.Error("expected status = active filter")
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want []", args)
	}
}

func TestBuildQ37MediaCountQuery_KindFilter(t *testing.T) {
	q, args := BuildQ37MediaCountQuery(domain.MediaFilters{KindFilter: "video"})

	if !strings.Contains(q, "mf.kind = ?") {
		t.Error("expected kind = ? clause")
	}
	if len(args) != 1 || args[0] != "video" {
		t.Errorf("args = %v, want [video]", args)
	}
}

func TestBuildQ37MediaCountQuery_LikedOnly(t *testing.T) {
	q, args := BuildQ37MediaCountQuery(domain.MediaFilters{LikedOnly: true})

	if !strings.Contains(q, "COALESCE(mf.liked, FALSE) = TRUE") {
		t.Error("expected liked filter")
	}
	if len(args) != 0 {
		t.Errorf("args = %v (len=%d), want []", args, len(args))
	}
}

func TestBuildQ37MediaCountQuery_MultipleFilters(t *testing.T) {
	f := domain.MediaFilters{
		KindFilter: "screenshot",
		MapFilter:  "Aquarius",
	}
	q, args := BuildQ37MediaCountQuery(f)

	if !strings.Contains(q, "mf.kind = ?") {
		t.Error("expected kind filter")
	}
	if !strings.Contains(q, "mr.map_name ILIKE ?") {
		t.Error("expected map filter")
	}
	// kind arg + map arg
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
}

func TestBuildQ37MediaCountQuery_HasJoins(t *testing.T) {
	q, _ := BuildQ37MediaCountQuery(domain.MediaFilters{MapFilter: "X"})
	if !strings.Contains(q, "LEFT JOIN media_match_associations") {
		t.Error("expected JOIN media_match_associations in count query")
	}
	if !strings.Contains(q, "LEFT JOIN shared.match_registry") {
		t.Error("expected JOIN shared.match_registry in count query")
	}
}
