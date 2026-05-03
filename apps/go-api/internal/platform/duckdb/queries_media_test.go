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
	if !strings.Contains(q, "ORDER BY COALESCE(mf.capture_end_utc, mf.mtime) DESC") {
		t.Errorf("expected default ORDER BY capture_end_utc/mtime DESC, got: %s", q)
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

	// Le filtre type doit accepter à la fois "screenshot" (legacy) et "image"
	// (nouveau schéma) pour matcher les médias quelle que soit leur origine.
	if !strings.Contains(q, "mf.kind IN (?,?)") {
		t.Errorf("expected kind IN (?,?) clause for legacy+new compat, got: %s", q)
	}
	// args: 2 valeurs équivalentes (screenshot, image), limit, offset
	if len(args) != 4 {
		t.Fatalf("args len = %d, want 4", len(args))
	}
	got := []any{args[0], args[1]}
	wantSet := map[string]bool{"screenshot": true, "image": true}
	for _, v := range got {
		s, ok := v.(string)
		if !ok || !wantSet[s] {
			t.Errorf("args[0:2] = %v, want set {screenshot, image}", got)
		}
	}
	if args[2] != 10 || args[3] != 0 {
		t.Errorf("args[2:] = %v, want [10 0]", args[2:])
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

	// Match exact (case-insensitive) — pas de substring (sinon "Recharge Annex"
	// matcherait "Recharge").
	if !strings.Contains(q, "LOWER("+q37MediaMapLabelExpr+") = LOWER(?)") {
		t.Errorf("expected exact match LOWER(map_label) = LOWER(?), got: %s", q)
	}
	if len(args) < 1 {
		t.Fatal("expected at least 1 filter arg")
	}
	if args[0] != "Fragmentation" {
		t.Errorf("map arg = %v, want exactly Fragmentation (no %% wrapping)", args[0])
	}
}

func TestBuildQ37MediaQuery_ModeFilter_FiestaCategory(t *testing.T) {
	// ModeFilter = catégorie custom Fiesta → reverse mapping vers les préfixes
	// pair_name (Fiesta, Super Fiesta, Husky Raid, Super Husky Raid, Castle Wars)
	// → OR sur (LIKE 'X:%' OR = 'X') pour chaque préfixe.
	q, args := BuildQ37MediaQuery(domain.MediaFilters{ModeFilter: "Fiesta"}, 24, 0)

	if !strings.Contains(q, "LOWER(mr.pair_name) LIKE LOWER(?)") {
		t.Errorf("expected pair_name LIKE for category reverse-mapping, got: %s", q)
	}
	if !strings.Contains(q, "LOWER(mr.pair_name) = LOWER(?)") {
		t.Errorf("expected pair_name = for parent-only modes (Husky Raid sans :), got: %s", q)
	}
	// Catégorie Fiesta = 2 préfixes restants après promotion (Fiesta, Castle Wars)
	// × 2 args (LIKE + =) + limit + offset = 6
	if len(args) != 6 {
		t.Fatalf("args len = %d, want 6 (2 préfixes × 2 + limit + offset)", len(args))
	}
}

func TestBuildQ37MediaQuery_AllFilters_ArgOrder(t *testing.T) {
	f := domain.MediaFilters{
		KindFilter: "video",
		MapFilter:  "Recharge",
		ModeFilter: "Ranked", // catégorie custom (1 préfixe : Ranked)
	}
	_, args := BuildQ37MediaQuery(f, 5, 10)

	// kind (×2) + map (×2 : map_id + label) + mode "Ranked" (1 préfixe × 2) + limit + offset = 8
	if len(args) != 8 {
		t.Fatalf("args len = %d, want 8", len(args))
	}
	kindArgs := []any{args[0], args[1]}
	wantKindSet := map[string]bool{"video": true, "clip": true}
	for _, v := range kindArgs {
		s, _ := v.(string)
		if !wantKindSet[s] {
			t.Errorf("args[0:2] = %v, want set {video, clip}", kindArgs)
		}
	}
	if args[2] != "Recharge" || args[3] != "Recharge" {
		t.Errorf("args[2:4] = %v, %v, want Recharge/Recharge", args[2], args[3])
	}
	// args[4]=Ranked:%, args[5]=Ranked
	if args[4] != "Ranked:%" || args[5] != "Ranked" {
		t.Errorf("args[4:6] = %v, %v, want Ranked:%% / Ranked", args[4], args[5])
	}
	if args[6] != 5 || args[7] != 10 {
		t.Errorf("args[6:] = %v, want [5 10]", args[6:])
	}
}

func TestBuildQ37MediaQuery_Sort_DateAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "date_asc"}, 24, 0)
	if !strings.Contains(q, "ORDER BY COALESCE(mf.capture_end_utc, mf.mtime) ASC") {
		t.Errorf("expected capture_end_utc/mtime ASC order, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_MapAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "map_asc"}, 24, 0)
	if !strings.Contains(q, "COALESCE("+q37MediaMapLabelExpr) {
		t.Errorf("expected map_name sort, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_ModeAsc(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "mode_asc"}, 24, 0)
	if !strings.Contains(q, "COALESCE("+q37MediaModeLabelExpr) {
		t.Errorf("expected pair_name sort, got: %s", q)
	}
}

func TestBuildQ37MediaQuery_Sort_Unknown_FallsBackToDefault(t *testing.T) {
	q, _ := BuildQ37MediaQuery(domain.MediaFilters{Sort: "random_invalid"}, 24, 0)
	if !strings.Contains(q, "ORDER BY COALESCE(mf.capture_end_utc, mf.mtime) DESC") {
		t.Errorf("expected fallback to capture_end_utc/mtime DESC, got: %s", q)
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

	if !strings.Contains(q, "SELECT COUNT(DISTINCT mf.file_path)") {
		t.Errorf("expected COUNT(DISTINCT mf.file_path) (dédup multi-assoc), got: %s", q)
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

	if !strings.Contains(q, "mf.kind IN (?,?)") {
		t.Errorf("expected kind IN (?,?) for legacy+new compat, got: %s", q)
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (video + clip equivalents)", len(args))
	}
	wantSet := map[string]bool{"video": true, "clip": true}
	for _, v := range args {
		s, _ := v.(string)
		if !wantSet[s] {
			t.Errorf("args = %v, want set {video, clip}", args)
		}
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

	if !strings.Contains(q, "mf.kind IN (?,?)") {
		t.Errorf("expected kind IN (?,?), got: %s", q)
	}
	if !strings.Contains(q, "LOWER("+q37MediaMapLabelExpr+") = LOWER(?)") {
		t.Error("expected exact map filter")
	}
	// 2 kind args (screenshot, image) + 2 map args (map_id + label fallback) = 4
	if len(args) != 4 {
		t.Fatalf("args len = %d, want 4", len(args))
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

func TestBuildQ37MediaMapOptionsQuery_IgnoresCurrentMapFilter(t *testing.T) {
	q, args := BuildQ37MediaMapOptionsQuery(domain.MediaFilters{
		KindFilter: "clip",
		MapFilter:  "Aquarius",
		ModeFilter: "Ranked", // catégorie custom
	})

	if strings.Contains(q, "LOWER("+q37MediaMapLabelExpr+") = LOWER(?)") {
		t.Error("expected current map filter to be ignored for map options")
	}
	if !strings.Contains(q, "LOWER(mr.pair_name) LIKE LOWER(?)") {
		t.Error("expected mode filter (catégorie reverse-mapping) to stay applied for map options")
	}
	if !strings.Contains(q, "AS map_id, "+q37MediaMapLabelExpr+" AS label") {
		t.Error("expected distinct (map_id, label) selection for FR enrichment")
	}
	// args : kind (×2) + mode "Ranked" (1 préfixe × 2 args) = 4
	if len(args) != 4 {
		t.Fatalf("args len = %d, want 4 (2 kind + 2 mode)", len(args))
	}
}

func TestBuildQ37MediaModeOptionsQuery_UsesNormalizedModeLabels(t *testing.T) {
	q, args := BuildQ37MediaModeOptionsQuery(domain.MediaFilters{
		MapFilter:  "Recharge",
		ModeFilter: "CTF",
	})

	if strings.Contains(q, "LOWER("+q37MediaModeLabelExpr+") = LOWER(?)") {
		t.Error("expected current mode filter to be ignored for mode options")
	}
	if !strings.Contains(q, "LOWER("+q37MediaMapLabelExpr+") = LOWER(?)") {
		t.Error("expected map filter to stay applied for mode options")
	}
	if !strings.Contains(q, "AS pair_name_raw, "+q37MediaModeLabelExpr+" AS label") {
		t.Error("expected distinct (pair_name_raw, label) selection for FR enrichment")
	}
	if !strings.Contains(q, "regexp_replace") {
		t.Error("expected normalized mode expression in options query")
	}
	// 2 args : map_id candidate + label fallback (Recharge/Recharge)
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (map_id + label)", len(args))
	}
}

func TestBuildQ37MediaQuery_SharedSocialSchemaUsesPlayerScopedJoin(t *testing.T) {
	q, args := buildQ37MediaQuery(domain.MediaFilters{}, 24, 0, mediaQueryConfig{playerSlug: "HeroPlayer"})

	if !strings.Contains(q, "mf.id = mma.media_file_id") {
		t.Errorf("expected shared_social join on media_file_id, got: %s", q)
	}
	if !strings.Contains(q, "mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') AS match_start_time") {
		t.Errorf("expected match_start_time via COALESCE(start_time_utc, start_time AT TIME ZONE UTC), got: %s", q)
	}
	if strings.Contains(q, "mf.status = 'active'") {
		t.Errorf("did not expect legacy status filter in shared_social query, got: %s", q)
	}
	// Default SectionFilter "" = "Tous auteurs" → pas de contrainte WHERE sur player_slug.
	if strings.Contains(q, "mf.player_slug =") || strings.Contains(q, "mf.player_slug <>") || strings.Contains(q, "mf.player_slug IN") {
		t.Errorf("default section filter should not constrain player_slug in WHERE, got: %s", q)
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (limit, offset)", len(args))
	}
	if args[0] != 24 || args[1] != 0 {
		t.Fatalf("args = %v, want [24 0]", args)
	}
}

func TestBuildQ37MediaQuery_SectionFilterMineScopesToPlayer(t *testing.T) {
	q, args := buildQ37MediaQuery(domain.MediaFilters{SectionFilter: "mine"}, 24, 0, mediaQueryConfig{playerSlug: "HeroPlayer"})

	if !strings.Contains(q, "mf.player_slug = ?") {
		t.Errorf("expected mine to filter on player_slug =, got: %s", q)
	}
	if len(args) != 3 || args[0] != "HeroPlayer" {
		t.Fatalf("args = %v, want [HeroPlayer 24 0]", args)
	}
}

func TestBuildQ37MediaQuery_SectionFilterTeammateExcludesPlayer(t *testing.T) {
	q, args := buildQ37MediaQuery(domain.MediaFilters{SectionFilter: "teammate"}, 24, 0, mediaQueryConfig{playerSlug: "HeroPlayer"})

	if !strings.Contains(q, "mf.player_slug <> ?") {
		t.Errorf("expected teammate to filter on player_slug <>, got: %s", q)
	}
	if len(args) != 3 || args[0] != "HeroPlayer" {
		t.Fatalf("args = %v, want [HeroPlayer 24 0]", args)
	}
}

func TestBuildQ37MediaQuery_AuthorSlugsBuildsInClause(t *testing.T) {
	q, args := buildQ37MediaQuery(
		domain.MediaFilters{AuthorSlugs: []string{"alice", "bob"}},
		24, 0, mediaQueryConfig{playerSlug: "HeroPlayer"},
	)

	if !strings.Contains(q, "mf.player_slug IN (?,?)") {
		t.Errorf("expected IN clause from AuthorSlugs, got: %s", q)
	}
	if len(args) != 4 || args[0] != "alice" || args[1] != "bob" {
		t.Fatalf("args = %v, want [alice bob 24 0]", args)
	}
}

func TestBuildQ37MediaQuery_GroupByMapPrefixesOrderBy(t *testing.T) {
	q, _ := buildQ37MediaQuery(domain.MediaFilters{GroupBy: "map"}, 24, 0, mediaQueryConfig{playerSlug: "HeroPlayer"})

	if !strings.Contains(q, "ORDER BY COALESCE("+q37MediaMapLabelExpr+", '~zzz') ASC,") {
		t.Errorf("expected GroupBy=map to prefix ORDER BY with map expr, got: %s", q)
	}
}

// ModeFilterCandidates a été retiré : le filtre est maintenant au niveau
// catégorie custom (Assassin/Fiesta/BTB/Ranked/Firefight/Other) qui se
// reverse-mappe directement vers les préfixes pair_name dans le WHERE.
// Cf. TestBuildQ37MediaQuery_ModeFilter_FiestaCategory pour la nouvelle forme.

func TestBuildQ37MediaMapOptionsQuery_SharedSocialSchemaDefaultNoPlayerScope(t *testing.T) {
	q, args := buildQ37MediaMapOptionsQuery(domain.MediaFilters{ModeFilter: "Ranked"}, mediaQueryConfig{playerSlug: "HeroPlayer"})

	if !strings.Contains(q, "mf.id = mma.media_file_id") {
		t.Errorf("expected shared_social join on media_file_id, got: %s", q)
	}
	if strings.Contains(q, "mf.player_slug") {
		t.Errorf("default section filter should not constrain player_slug, got: %s", q)
	}
	// args : "Ranked" catégorie (1 préfixe × 2 args : LIKE + =)
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2 (1 préfixe Ranked × 2)", len(args))
	}
}
