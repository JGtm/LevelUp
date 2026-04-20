// Package halo — provider_test.go : tests unitaires HaloProvider (boîte blanche).
package halo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// newTestProvider crée un provider avec des base URLs injectées depuis les serveurs de test.
func newTestProvider(battlePassURL, challengesURL string) *HaloProvider {
	p := NewHaloProvider()
	p.battlePassBaseURL = battlePassURL
	p.challengesBaseURL = challengesURL
	if challengesURL != "" {
		p.gameCMSBaseURL = challengesURL
	}
	p.limiter = newRateLimiter(600)
	p.maxRetries = 3
	return p
}

func testTokens() *domain.HaloTokens {
	return &domain.HaloTokens{SpartanToken: "spartan_test", ClearanceToken: "clear_test"}
}

func ctxWithAuth(tokens *domain.HaloTokens, xuid string) context.Context {
	return ctxkeys.WithHaloAuth(context.Background(), tokens, xuid)
}

// ---------------------------------------------------------------------------
// GetBattlePass — cas sans auth
// ---------------------------------------------------------------------------

func TestGetBattlePass_NoTokens(t *testing.T) {
	p := newTestProvider("", "")
	resp := p.GetBattlePass(context.Background())

	if resp.Available {
		t.Error("expected available=false when no tokens")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

func TestGetBattlePass_NoXUID(t *testing.T) {
	ctx := ctxkeys.WithHaloAuth(context.Background(), testTokens(), "")
	p := newTestProvider("", "")
	resp := p.GetBattlePass(ctx)

	if resp.Available {
		t.Error("expected available=false when xuid empty")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetBattlePass — appel live OK
// ---------------------------------------------------------------------------

func TestGetBattlePass_LiveOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") == "" {
			http.Error(w, "missing spartan", http.StatusUnauthorized)
			return
		}
		payload := map[string]any{
			"ActiveOperationRewardTrackPath": "RewardTracks/Operations/Season6-Op",
			"OperationRewardTracks": []any{
				map[string]any{
					"RewardTrackPath": "RewardTracks/Operations/Season6-Op",
					"CurrentProgress": map[string]any{
						"Rank":            45,
						"PartialProgress": 2500,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	ctx := ctxWithAuth(testTokens(), "xuid-test")
	resp := p.GetBattlePass(ctx)

	if !resp.Available {
		t.Errorf("expected available=true, error_hint=%v", resp.ErrorHint)
	}
	if resp.Rank == nil || *resp.Rank != 45 {
		t.Errorf("expected rank=45, got %v", resp.Rank)
	}
	if resp.Progress == nil || *resp.Progress != 2500 {
		t.Errorf("expected progress=2500, got %v", resp.Progress)
	}
	if resp.RewardTrack == nil || *resp.RewardTrack != "RewardTracks/Operations/Season6-Op" {
		t.Errorf("expected reward_track, got %v", resp.RewardTrack)
	}
}

// ---------------------------------------------------------------------------
// GetBattlePass — erreurs HTTP
// ---------------------------------------------------------------------------

func TestGetBattlePass_HTTP401(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on 401")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error, got %v", resp.ErrorHint)
	}
	// 401 : pas de retry — 1 seul appel.
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry on 401), got %d", callCount)
	}
}

func TestGetBattlePass_HTTP500ThenOK(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		payload := map[string]any{
			"ActiveOperationRewardTrackPath": "RewardTracks/Operations/S7",
			"OperationRewardTracks": []any{
				map[string]any{
					"RewardTrackPath": "RewardTracks/Operations/S7",
					"CurrentProgress": map[string]any{"Rank": 10, "PartialProgress": 0},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true after retry, error_hint=%v", resp.ErrorHint)
	}
	if resp.Rank == nil || *resp.Rank != 10 {
		t.Errorf("expected rank=10, got %v", resp.Rank)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Errorf("expected at least 2 calls (retry), got %d", callCount)
	}
}

func TestGetBattlePass_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	resp := p.GetBattlePass(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on malformed JSON")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetChallenges — cas sans auth
// ---------------------------------------------------------------------------

func TestGetChallenges_NoTokens(t *testing.T) {
	p := newTestProvider("", "")
	resp := p.GetChallenges(context.Background())

	if resp.Available {
		t.Error("expected available=false when no tokens")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "auth_required" {
		t.Errorf("expected error_hint=auth_required, got %v", resp.ErrorHint)
	}
}

// ---------------------------------------------------------------------------
// GetChallenges — appel live OK
// ---------------------------------------------------------------------------

func TestGetChallenges_LiveOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hi/players/xuid(xuid-test)/decks":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			payload := map[string]any{
				"AssignedDecks": []any{
					map[string]any{
						"Expiration": map[string]string{"ISO8601Date": "2024-11-19T17:00:00Z"},
						"ActiveChallenges": []any{
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json", "XPReward": 1000, "Progress": 7, "Threshold": 10, "TrackingId": "track-1"},
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/DailyChallenges/ch2.json", "XPReward": 2000, "Progress": 1, "Threshold": 3, "TrackingId": "track-2"},
							map[string]any{"Path": "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Weapon/ch3.json", "XPReward": 1500, "Progress": 0, "Threshold": 5, "TrackingId": "track-3"},
						},
						"CompletedChallenges": []any{
							map[string]any{"XPReward": 500},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		case "/hi/Progression/file/ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi avancé"}, "value": "Advanced Challenge"},
				"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Gagne des points rapidement"}, "value": "Gain points quickly"},
				"Category":            "Weekly",
				"Difficulty":          "Heroic",
				"ThresholdForSuccess": 10,
			})
		case "/hi/Progression/file/ChallengeContent/ClientChallengeDefinitions/DailyChallenges/ch2.json":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi en cours"}, "value": "In Progress Challenge"},
				"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Continue la progression"}, "value": "Keep going"},
				"Category":            "Daily",
				"Difficulty":          "Normal",
				"ThresholdForSuccess": 3,
			})
		case "/hi/Progression/file/ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Weapon/ch3.json":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Défi pas commencé"}, "value": "Not Started Challenge"},
				"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Commence ce défi"}, "value": "Start this challenge"},
				"Category":            "Weekly",
				"Difficulty":          "Legendary",
				"ThresholdForSuccess": 5,
			})
		case "/hi/waypoint/file/images/weekly-action-heroic.png":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-ch1"))
		case "/hi/waypoint/file/images/daily-normal.png":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-ch2"))
		case "/hi/waypoint/file/images/weekly-weapon-legendary.png":
			if r.Header.Get("x-343-authorization-spartan") == "" {
				http.Error(w, "missing spartan", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-ch3"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true, error_hint=%v", resp.ErrorHint)
	}
	if resp.Total == nil || *resp.Total != 4 {
		t.Errorf("expected total=4, got %v", resp.Total)
	}
	if resp.Completed == nil || *resp.Completed != 1 {
		t.Errorf("expected completed=1, got %v", resp.Completed)
	}
	if resp.XPAvailable == nil || *resp.XPAvailable != 4500 {
		t.Errorf("expected xp_available=4500, got %v", resp.XPAvailable)
	}
	if resp.NextExpiry == nil || *resp.NextExpiry != "2024-11-19T17:00:00Z" {
		t.Errorf("expected next_expiry, got %v", resp.NextExpiry)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 active items, got %d", len(resp.Items))
	}
	if resp.Items[0].Title != "Défi avancé" || resp.Items[1].Title != "Défi en cours" || resp.Items[2].Title != "Défi pas commencé" {
		t.Fatalf("unexpected item order: %#v", resp.Items)
	}
	if resp.Items[0].ProgressPercent == nil || int(*resp.Items[0].ProgressPercent) != 70 {
		t.Fatalf("expected first item progress 70%%, got %#v", resp.Items[0].ProgressPercent)
	}
	if resp.Items[2].ProgressPercent == nil || int(*resp.Items[2].ProgressPercent) != 0 {
		t.Fatalf("expected third item progress 0%%, got %#v", resp.Items[2].ProgressPercent)
	}
	if resp.Items[0].Description == nil || *resp.Items[0].Description == "" {
		t.Fatal("expected description on enriched challenge")
	}
	if resp.Items[0].ImageURL == nil || *resp.Items[0].ImageURL == "" {
		t.Fatal("expected image_url on enriched challenge")
	}
	if !strings.HasPrefix(*resp.Items[0].ImageURL, "data:image/png;base64,") {
		t.Fatalf("expected data URL image, got %q", *resp.Items[0].ImageURL)
	}
}

func TestGetChallenges_EmptyDecks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"AssignedDecks": []any{}})
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if !resp.Available {
		t.Errorf("expected available=true even with empty decks")
	}
	if resp.Total == nil || *resp.Total != 0 {
		t.Errorf("expected total=0, got %v", resp.Total)
	}
	if resp.Completed == nil || *resp.Completed != 0 {
		t.Errorf("expected completed=0, got %v", resp.Completed)
	}
	if resp.NextExpiry != nil {
		t.Errorf("expected nil next_expiry for empty decks, got %v", resp.NextExpiry)
	}
}

func TestGetChallenges_HTTP401(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL)
	resp := p.GetChallenges(ctxWithAuth(testTokens(), "xuid-test"))

	if resp.Available {
		t.Error("expected available=false on 401")
	}
	if resp.ErrorHint == nil || *resp.ErrorHint != "fetch_error" {
		t.Errorf("expected error_hint=fetch_error")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry on 401), got %d", callCount)
	}
}

func TestFetchChallengeDefinition_UsesMetadataCache(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(metadata): %v", err)
	}

	ctx := context.Background()
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS challenge_definitions (
			challenge_path VARCHAR NOT NULL,
			content_hash VARCHAR NOT NULL,
			category VARCHAR,
			difficulty VARCHAR,
			threshold_for_success INTEGER,
			reward_xp INTEGER DEFAULT 0,
			secondary_reward_xp INTEGER DEFAULT 0,
			first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_current BOOLEAN DEFAULT TRUE,
			PRIMARY KEY (challenge_path, content_hash)
		);
		CREATE TABLE IF NOT EXISTS challenge_translations (
			challenge_path VARCHAR NOT NULL,
			content_hash VARCHAR NOT NULL,
			lang VARCHAR NOT NULL,
			title VARCHAR,
			description VARCHAR,
			first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (challenge_path, content_hash, lang)
		);`)
	if err != nil {
		t.Fatalf("create challenge tables: %v", err)
	}

	challengePath := "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json"
	_, err = db.Exec(ctx, `
		INSERT INTO challenge_definitions
			(challenge_path, content_hash, category, difficulty, threshold_for_success, reward_xp, secondary_reward_xp, is_current)
		VALUES (?, 'hash-1', 'Weekly', 'Heroic', 10, 1000, 250, TRUE)`, challengePath)
	if err != nil {
		t.Fatalf("insert challenge definition: %v", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO challenge_translations
			(challenge_path, content_hash, lang, title, description)
		VALUES
			(?, 'hash-1', 'fr-FR', 'Defi cache', 'Description locale'),
			(?, 'hash-1', 'en-US', 'Cached challenge', 'Local description')`, challengePath, challengePath)
	if err != nil {
		t.Fatalf("insert challenge translations: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close metadata db: %v", err)
	}

	p := NewHaloProvider().WithChallengeCache(metaPath, "")
	def, err := p.fetchChallengeDefinition(ctx, testTokens(), challengePath)
	if err != nil {
		t.Fatalf("fetchChallengeDefinition(metadata): %v", err)
	}
	if def == nil {
		t.Fatal("expected metadata challenge definition")
	}
	if got := resolveChallengeLocalizedValue(def.Title, "fr-FR"); got != "Defi cache" {
		t.Fatalf("unexpected title from metadata: %q", got)
	}
	if got := resolveChallengeLocalizedValue(def.Description, "fr-FR"); got != "Description locale" {
		t.Fatalf("unexpected description from metadata: %q", got)
	}
	if def.Category != "Weekly" || def.Difficulty != "Heroic" {
		t.Fatalf("unexpected metadata fields: %#v", def)
	}
	if resolvedThreshold, ok := coerceChallengeInt(def.ThresholdForSuccess); !ok || resolvedThreshold != 10 {
		t.Fatalf("unexpected threshold from metadata: %#v", def.ThresholdForSuccess)
	}
	if def.Reward.SoftExperience != 1000 || def.SecondaryReward.SoftExperience != 250 {
		t.Fatalf("unexpected xp rewards: %#v", def)
	}
}

func TestFetchChallengeDefinition_PersistsMetadataAfterLiveFetch(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(metadata): %v", err)
	}
	ctx := context.Background()
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS challenge_definitions (
			challenge_path VARCHAR NOT NULL,
			content_hash VARCHAR NOT NULL,
			category VARCHAR,
			difficulty VARCHAR,
			threshold_for_success INTEGER,
			reward_xp INTEGER DEFAULT 0,
			secondary_reward_xp INTEGER DEFAULT 0,
			first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_current BOOLEAN DEFAULT TRUE,
			PRIMARY KEY (challenge_path, content_hash)
		);
		CREATE TABLE IF NOT EXISTS challenge_translations (
			challenge_path VARCHAR NOT NULL,
			content_hash VARCHAR NOT NULL,
			lang VARCHAR NOT NULL,
			title VARCHAR,
			description VARCHAR,
			first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (challenge_path, content_hash, lang)
		);`)
	if err != nil {
		t.Fatalf("create challenge tables: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close metadata db: %v", err)
	}

	challengePath := "ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Weapon/ch3.json"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hi/Progression/file/" + challengePath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Title":               map[string]any{"translations": map[string]any{"fr-FR": "Defi persiste"}, "value": "Persisted challenge"},
				"Description":         map[string]any{"translations": map[string]any{"fr-FR": "Description persistee"}, "value": "Persisted description"},
				"Category":            "Weekly",
				"Difficulty":          "Legendary",
				"ThresholdForSuccess": 5,
				"Reward":              map[string]any{"SoftExperience": 1500},
				"SecondaryReward":     map[string]any{"SoftExperience": 300},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL).WithChallengeCache(metaPath, "")
	def, err := p.fetchChallengeDefinition(ctx, testTokens(), challengePath)
	if err != nil {
		t.Fatalf("fetchChallengeDefinition(live): %v", err)
	}
	if def == nil {
		t.Fatal("expected live challenge definition")
	}

	metaRO, err := duckdb.OpenReadOnly(metaPath)
	if err != nil {
		t.Fatalf("OpenReadOnly(metadata): %v", err)
	}
	defer func() { _ = metaRO.Close() }()

	var title string
	var description string
	var category string
	var difficulty string
	if err := metaRO.QueryRow(ctx, `
		SELECT t.title, t.description, d.category, d.difficulty
		FROM challenge_definitions d
		JOIN challenge_translations t
		  ON t.challenge_path = d.challenge_path
		 AND t.content_hash = d.content_hash
		WHERE d.challenge_path = ?
		  AND d.is_current = TRUE
		  AND t.lang = 'fr-FR'
		LIMIT 1`, challengePath).Scan(&title, &description, &category, &difficulty); err != nil {
		t.Fatalf("query persisted challenge metadata: %v", err)
	}
	if title != "Defi persiste" || description != "Description persistee" {
		t.Fatalf("unexpected persisted translations: %q / %q", title, description)
	}
	if category != "Weekly" || difficulty != "Legendary" {
		t.Fatalf("unexpected persisted metadata: %q / %q", category, difficulty)
	}
}

func TestFetchChallengeBadgeDataURL_UsesLocalWeeklyCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "challenge_badges")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(cacheDir): %v", err)
	}
	pngBody := append(append([]byte{}, challengePNGSignature...), []byte("weekly-cache")...)
	cachePath := filepath.Join(cacheDir, "weekly-action-heroic.png")
	if err := os.WriteFile(cachePath, pngBody, 0o644); err != nil {
		t.Fatalf("WriteFile(cache badge): %v", err)
	}

	p := NewHaloProvider().WithChallengeCache("", cacheDir)
	imageURL := p.fetchChallengeBadgeDataURL(
		context.Background(),
		testTokens(),
		"ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json",
		"Weekly",
		"Heroic",
	)
	if imageURL == nil || !strings.HasPrefix(*imageURL, "data:image/png;base64,") {
		t.Fatalf("expected data URL from local cache, got %v", imageURL)
	}
}

func TestFetchChallengeBadgeDataURL_PersistsLiveBadgeToLocalCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "challenge_badges")
	pngBody := append(append([]byte{}, challengePNGSignature...), []byte("live-weekly-cache")...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hi/waypoint/file/images/weekly-action-heroic.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := newTestProvider("", srv.URL).WithChallengeCache("", cacheDir)
	imageURL := p.fetchChallengeBadgeDataURL(
		context.Background(),
		testTokens(),
		"ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json",
		"Weekly",
		"Heroic",
	)
	if imageURL == nil || !strings.HasPrefix(*imageURL, "data:image/png;base64,") {
		t.Fatalf("expected data URL from live badge fetch, got %v", imageURL)
	}

	cachePath := filepath.Join(cacheDir, "weekly-action-heroic.png")
	body, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected persisted badge in local cache: %v", err)
	}
	if string(body) != string(pngBody) {
		t.Fatalf("unexpected cached badge bytes")
	}
}

// ---------------------------------------------------------------------------
// parseBattlePassTrack — unité pure
// ---------------------------------------------------------------------------

func TestParseBattlePassTrack_MatchFound(t *testing.T) {
	tracks := []battlePassTrack{
		{RewardTrackPath: "op/Season5", CurrentProgress: battlePassProgress{Rank: 20, PartialProgress: 500}},
		{RewardTrackPath: "op/Season6", CurrentProgress: battlePassProgress{Rank: 45, PartialProgress: 2500}},
	}
	rank, prog, path := parseBattlePassTrack("op/Season6", tracks)
	if rank != 45 || prog != 2500 || path != "op/Season6" {
		t.Errorf("unexpected result: rank=%d prog=%d path=%q", rank, prog, path)
	}
}

func TestParseBattlePassTrack_NoMatch_FallbackFirst(t *testing.T) {
	tracks := []battlePassTrack{
		{RewardTrackPath: "op/Season5", CurrentProgress: battlePassProgress{Rank: 10, PartialProgress: 100}},
	}
	rank, prog, path := parseBattlePassTrack("op/UnknownSeason", tracks)
	if rank != 10 || prog != 100 || path != "op/Season5" {
		t.Errorf("unexpected fallback: rank=%d prog=%d path=%q", rank, prog, path)
	}
}

func TestParseBattlePassTrack_EmptyTracks(t *testing.T) {
	rank, prog, path := parseBattlePassTrack("op/Season6", nil)
	if rank != 0 || prog != 0 || path != "op/Season6" {
		t.Errorf("unexpected result on empty tracks: rank=%d prog=%d path=%q", rank, prog, path)
	}
}
