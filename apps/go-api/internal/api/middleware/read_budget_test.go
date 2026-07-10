package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Le middleware doit envelopper le contexte de la requête (budget d'attente de
// swap posé via sharedprovider.WithSwapWaitBudget) ET appeler next. La sémantique
// fail-fast du budget elle-même est couverte par
// sharedprovider/read_budget_integration_test.go — ici on vérifie le CÂBLAGE HTTP.
func TestUserFacingReadBudget_WrapsContextAndCallsNext(t *testing.T) {
	var gotCtx context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	mw := UserFacingReadBudget(2 * time.Second)(next)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	origCtx := req.Context()
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("next non appelé (status %d)", rec.Code)
	}
	if gotCtx == nil {
		t.Fatal("next a reçu un contexte nil")
	}
	// WithSwapWaitBudget(ctx, budget>0) retourne un nouveau contexte (WithValue) :
	// le contexte vu par next doit différer de l'original.
	if gotCtx == origCtx {
		t.Error("le contexte n'a pas été enveloppé avec le budget de swap")
	}
}

// budget <= 0 retombe sur DefaultUserFacingReadBudget (>0) : le middleware
// enveloppe quand même le contexte et n'échoue pas.
func TestUserFacingReadBudget_NonPositiveUsesDefault(t *testing.T) {
	for _, budget := range []time.Duration{0, -1 * time.Second} {
		var called bool
		var gotCtx context.Context
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			gotCtx = r.Context()
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		origCtx := req.Context()
		rec := httptest.NewRecorder()
		UserFacingReadBudget(budget)(next).ServeHTTP(rec, req)

		if !called {
			t.Fatalf("budget=%v : next non appelé", budget)
		}
		if gotCtx == origCtx {
			t.Errorf("budget=%v : défaut (>0) attendu → contexte devrait être enveloppé", budget)
		}
	}
}
