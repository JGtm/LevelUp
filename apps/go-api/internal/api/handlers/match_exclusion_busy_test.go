package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
)

// Une base momentanément indisponible (verrou d'écriture du joueur, bascule de la base
// partagée pendant une synchronisation) répond 503 db_busy + Retry-After, jamais 500 :
// l'exclusion est idempotente et peut être retentée.
func TestMatchExclusionHandler_Set_BusyDatabaseAnswers503(t *testing.T) {
	for name, cause := range map[string]error{
		"verrou d'écriture du joueur": dblease.ErrDBLocked,
		"bascule de la base partagée": sharedprovider.ErrSwapTimeout,
		"bascule échouée":             sharedprovider.ErrSwapFailed,
		"fournisseur partagé fermé":   sharedprovider.ErrProviderClosed,
	} {
		t.Run(name, func(t *testing.T) {
			mock := &mockMatchExclusionService{setErr: fmt.Errorf("MatchExclusionService.SetExclusion recompute: %w", cause)}
			r := newMatchExclusionRouter(func(context.Context, string) (port.MatchExclusionService, error) {
				return mock, nil
			})
			body, _ := json.Marshal(domain.SetMatchExclusionRequest{Excluded: true})
			req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/exclusion", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("statut %d, attendu 503 : %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Retry-After") == "" {
				t.Error("le 503 doit porter Retry-After")
			}
			if !strings.Contains(w.Body.String(), "db_busy") {
				t.Errorf("code attendu db_busy : %s", w.Body.String())
			}
		})
	}
}
