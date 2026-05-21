//go:build integration

// progression_test.go — tests d'intégration des 3 endpoints progression V2.
// Stratégie : DuckDB temp + PlayerDB câblé + chi.Router minimal + httptest.

package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

const (
	progTestXUID  = "xuid-progression-handler"
	progTestGT    = "ProgressionHandlerTester"
	progTestSlug  = "progression-handler-tester"
	progTestTitle = "halo_infinite"
)

// setupHandlerEnv crée 3 DBs (Player, SharedSocial, Metadata) + PlayerDB.
// On omet Shared (pas utilisé par les handlers, contrairement au hook post-sync).
func setupHandlerEnv(t *testing.T) (*duckdb.PlayerDB, func()) {
	t.Helper()
	dir := t.TempDir()

	openAndMigrate := func(name string, target migration.TargetDB) *duckdb.DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		raw.Close()
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		return db
	}

	player := openAndMigrate("stats", migration.TargetPlayer)
	social := openAndMigrate("shared_social", migration.TargetSharedSocial)
	meta := openAndMigrate("metadata", migration.TargetMetadata)

	pdb := &duckdb.PlayerDB{
		Player: player, SharedSocial: social, Metadata: meta,
		XUID: progTestXUID, Gamertag: progTestGT, TitleSlug: progTestTitle,
	}
	cleanup := func() {
		player.Close()
		social.Close()
		meta.Close()
		_ = os.RemoveAll(dir)
	}
	t.Cleanup(cleanup)
	return pdb, cleanup
}

// mountRouter assemble un chi router minimal aligné sur la prod
// (subroute /players/{player_slug}).
func mountRouter(pdb *duckdb.PlayerDB) *chi.Mux {
	r := chi.NewRouter()
	resolver := func(_ context.Context, slug string) (*duckdb.PlayerDB, error) {
		if slug != progTestSlug {
			return nil, errNotFound
		}
		return pdb, nil
	}
	h := handlers.NewProgressionHandler(resolver, progTestTitle)
	r.Route("/api/v1/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

var errNotFound = simpleErr{"player not found"}

type simpleErr struct{ msg string }

func (e simpleErr) Error() string { return e.msg }

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestListStreaks_Empty_ReturnsEmptyArray(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)
	r := mountRouter(pdb)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+progTestSlug+"/streaks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 0 {
		t.Errorf("empty DB should return 0 streaks, got %d", len(body.Items))
	}
}

func TestListStreaks_WithSeededStreak(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)

	// Seed une streak active.
	ctx := context.Background()
	repo := duckdb.NewStreaksRepo(pdb.Player)
	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.Upsert(ctx, streaks.Streak{
		ID: "s1", UserID: progTestXUID, TitleSlug: progTestTitle,
		Type: streaks.StreakTypeDailyPlay, StartedAt: now.AddDate(0, 0, -7),
		CurrentLength: 8, BestLength: 10, LastIncrementAt: &now,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusActive,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := mountRouter(pdb)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+progTestSlug+"/streaks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Items []struct {
			ID            string  `json:"id"`
			Type          string  `json:"type"`
			CurrentLength int     `json:"current_length"`
			BestLength    int     `json:"best_length"`
			Status        string  `json:"status"`
			PPMultiplier  float64 `json:"pp_multiplier"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 streak, got %d", len(body.Items))
	}
	s := body.Items[0]
	if s.ID != "s1" || s.Type != "daily_play" || s.CurrentLength != 8 || s.BestLength != 10 {
		t.Errorf("unexpected streak fields: %+v", s)
	}
	if s.PPMultiplier != 1.25 { // longueur 8 → palier 1.25x
		t.Errorf("PPMultiplier = %.2f, want 1.25", s.PPMultiplier)
	}
}

func TestListRecords_WithPBAndHistory(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	pbRepo := duckdb.NewPersonalRecordsRepo(pdb)
	if err := pbRepo.Upsert(ctx, records.PersonalRecord{
		XUID: progTestXUID, Metric: "performance_score",
		Period: records.RecordPeriodAllTime, Value: 92,
		AchievedAt: &now, AchievedMatchID: "m_42", UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed PB: %v", err)
	}
	histRepo := duckdb.NewRecordHistoryRepo(pdb.Player)
	if err := histRepo.Append(ctx, records.RecordHistory{
		ID: "h1", UserID: progTestXUID, TitleSlug: progTestTitle,
		Metric: "performance_score", Period: records.RecordPeriodAllTime,
		Value: 92, AchievedAt: now,
	}); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	r := mountRouter(pdb)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+progTestSlug+"/records", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		PersonalBests []struct {
			Metric string  `json:"metric"`
			Period string  `json:"period"`
			Value  float64 `json:"value"`
		} `json:"personal_bests"`
		History []struct {
			ID    string  `json:"id"`
			Value float64 `json:"value"`
		} `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.PersonalBests) != 1 {
		t.Errorf("expected 1 PB, got %d", len(body.PersonalBests))
	}
	if len(body.History) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(body.History))
	}
	if body.PersonalBests[0].Value != 92 || body.History[0].ID != "h1" {
		t.Errorf("unexpected payload: %+v", body)
	}
}

func TestListMilestones_CatalogPlusEarned(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	catRepo := duckdb.NewMilestoneCatalogRepo(pdb.Metadata)
	entries := []milestones.CatalogEntry{
		{ID: "h.matches.10", TitleSlug: progTestTitle, Metric: "matches_played",
			Threshold: 10, TitleEN: "Decimus", TitleFR: "Decimus"},
		{ID: "h.matches.100", TitleSlug: progTestTitle, Metric: "matches_played",
			Threshold: 100, TitleEN: "Centurion", TitleFR: "Centurion"},
	}
	for _, e := range entries {
		if err := catRepo.Upsert(ctx, e); err != nil {
			t.Fatalf("seed catalog %s: %v", e.ID, err)
		}
	}

	earnedRepo := duckdb.NewMilestoneEarnedRepo(pdb.Player)
	if err := earnedRepo.Append(ctx, milestones.Earned{
		UserID: progTestXUID, TitleSlug: progTestTitle,
		MilestoneID: "h.matches.10", EarnedAt: now,
	}); err != nil {
		t.Fatalf("seed earned: %v", err)
	}

	r := mountRouter(pdb)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+progTestSlug+"/milestones", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []struct {
			ID       string     `json:"id"`
			Earned   bool       `json:"earned"`
			EarnedAt *time.Time `json:"earned_at"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 catalog items, got %d", len(body.Items))
	}
	// Validate join : matches.10 earned, matches.100 not.
	byID := map[string]bool{}
	for _, it := range body.Items {
		byID[it.ID] = it.Earned
	}
	if !byID["h.matches.10"] {
		t.Errorf("h.matches.10 should be earned")
	}
	if byID["h.matches.100"] {
		t.Errorf("h.matches.100 should NOT be earned")
	}
}

func TestProgressionHandler_PlayerNotFound_Returns404(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)
	r := mountRouter(pdb)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/unknown-slug/streaks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
