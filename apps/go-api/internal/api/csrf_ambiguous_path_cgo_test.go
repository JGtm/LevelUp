//go:build cgo

// csrf_ambiguous_path_cgo_test.go — l'exemption CSRF du protocole ouvrier ne s'applique
// jamais à un chemin ambigu, prouvé sur le routeur complet (api.NewRouter assemblé par
// openapigen.BuildDemoRouter, pile transverse comprise).
//
// Le chemin ci-dessous se route vers `POST …/coach/proposals/{id}/dismiss` (chi lit le chemin
// brut, `{id}` = « ..%2F..%2F..%2F..%2Finternal%2Fx ») mais se nettoie en
// `/api/v1/internal/x/dismiss`, sous le préfixe exempté. Une page d'une autre origine qui
// l'émet doit recevoir le refus du CSRF, pas celui d'une garde plus loin dans la pile.

package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api"
)

// foreignOrigin : l'origine d'une page tierce.
const foreignOrigin = "https://evil.example"

// postFromForeignOrigin émet un POST porteur d'une Origin étrangère et rend statut et corps.
func postFromForeignOrigin(router http.Handler, target string) (int, string) {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", foreignOrigin)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr.Code, rr.Body.String()
}

// TestTransverseStack_CSRF_AmbiguousPathNeverTakesWorkerExemption : la requête du constat
// C10-1, puis toute la table des formes ambiguës, reçoivent 403 csrf_rejected.
func TestTransverseStack_CSRF_AmbiguousPathNeverTakesWorkerExemption(t *testing.T) {
	router := buildStackRouter(t, "")
	targets := []string{
		"/api/v1/players/demo-player/coach/proposals/..%2F..%2F..%2F..%2Finternal%2Fx/dismiss",
	}
	for _, f := range api.AmbiguousForms {
		targets = append(targets, "/api/v1/players/demo-player/coach/proposals/"+
			api.Smuggle(f, "..", "..", "..", "..", "internal", "x")+"/dismiss")
	}
	for _, target := range targets {
		code, body := postFromForeignOrigin(router, target)
		if code != http.StatusForbidden || !strings.Contains(body, "csrf_rejected") {
			t.Errorf("POST %s (Origin %s) : statut %d corps %s, attendu 403 csrf_rejected",
				target, foreignOrigin, code, body)
		}
	}
}
