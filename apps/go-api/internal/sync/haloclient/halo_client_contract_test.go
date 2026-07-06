// halo_client_contract_test.go — contract test contre fixture Halo représentative.
//
// Validation simultanée du format URL produit par GetMatchHistory ET du parsing
// MatchHistoryEntry contre une réponse JSON conforme au shape réel Halo
// (testdata/halo/match_history_response.json). Garde-fou si :
//
//   1. Le call site engine.go régresse vers le format gamertag brut.
//   2. Le shape MatchInfo.StartTime ou MatchId change côté API.

package haloclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGetMatchHistory_ContractAgainstFixture rejoue une réponse Halo réaliste
// (ordre décroissant par StartTime) et vérifie URL + parsing.
func TestGetMatchHistory_ContractAgainstFixture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "halo", "match_history_response.json")
	fixtureBody, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("lecture fixture %s : %v", fixturePath, err)
	}

	var captured *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixtureBody)
	}))
	defer srv.Close()

	client, _ := newDryRunClient(srv) // helper défini dans halo_client_url_dryrun_test.go

	const xuid = "2535469190789936"
	playerID := "xuid(" + xuid + ")"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries, err := client.GetMatchHistory(ctx, playerID, "matchmaking", 0, 25)
	if err != nil {
		t.Fatalf("GetMatchHistory: %v", err)
	}

	// ─── Contract URL ──────────────────────────────────────────────────────
	if captured == nil {
		t.Fatal("aucune requête capturée par le httptest server")
	}
	decodedPath, _ := url.PathUnescape(captured.URL.Path)
	wantPath := "/hi/players/xuid(" + xuid + ")/matches"
	if decodedPath != wantPath {
		t.Errorf("path effectif incorrect\n  got  : %s\n  want : %s", decodedPath, wantPath)
	}
	q := captured.URL.Query()
	if q.Get("type") != "matchmaking" || q.Get("start") != "0" || q.Get("count") != "25" {
		t.Errorf("query params incorrects : %s", captured.URL.RawQuery)
	}

	// ─── Contract parsing ──────────────────────────────────────────────────
	if len(entries) != 4 {
		t.Fatalf("nombre d'entries attendu : 4, got %d", len(entries))
	}

	// Ordre Halo : plus récent en tête.
	if entries[0].MatchID != "11111111-1111-1111-1111-000000000001" {
		t.Errorf("1er match (le plus récent) attendu …000000000001, got %s", entries[0].MatchID)
	}
	if !strings.HasPrefix(entries[0].StartTime, "2026-05-19") {
		t.Errorf("StartTime du 1er match attendu 2026-05-19*, got %s", entries[0].StartTime)
	}

	// Le dernier (4ème) = b8c1b220, ancré au 6 mai (= last_sync_at de l'incident
	// 2026-05-20). Si jamais l'API renvoie un ordre croissant, ce test rouge.
	last := entries[len(entries)-1]
	if last.MatchID != "b8c1b220-5ef4-4dee-9e92-77d3ff55d6d3" {
		t.Errorf("dernier match attendu b8c1b220-..., got %s", last.MatchID)
	}
	if !strings.HasPrefix(last.StartTime, "2026-05-06") {
		t.Errorf("StartTime du dernier match attendu 2026-05-06*, got %s", last.StartTime)
	}
}
