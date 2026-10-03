//go:build cgo

// csrf_transverse_stack_cgo_test.go — LA PREUVE QUI MANQUAIT (lot csrf-ouvrier,
// 2026-08-25).
//
// LE TROU QU'IL BOUCHE. Le protocole ouvrier avait déjà une preuve e2e complète
// (wire/build_queue_transport_e2e_cgo_test.go : vrai serveur HTTP, vraie DuckDB,
// vrai artefact rendu à l'octet près). Elle montait pourtant les routes par un
// `chi.NewRouter()` NU + MountBuildWorkerRoutes — c'est-à-dire SANS la pile
// transverse du routeur de production. Elle ne pouvait donc pas voir que le
// middleware CSRF, monté transversalement sur la racine, rejetait tout le
// protocole en 403 `csrf_rejected` avant même le contrôle de jeton. Le défaut n'a
// été trouvé qu'en conditions réelles (dry run superviseur du 2026-08-25 depuis le
// VPS de calcul, contre https://lvelup.info/api/v1/internal).
//
// La leçon est la raison d'être de ce fichier : ce qui traverse la pile doit être
// prouvé SUR la pile. Le routeur testé ici est celui d'`api.NewRouter` (assemblé
// par openapigen.BuildDemoRouter, le même harnais que les tests de contrat et que
// la génération d'api/openapi.yaml), middlewares transverses compris.
//
// CE QUI N'EST PAS COUVERT ICI, ET OÙ ÇA L'EST. Le chemin nominal « jeton valide →
// job pris → artefact → compte rendu » reste couvert par la preuve e2e du package
// wire, qui dispose d'un store DuckDB réel. Le routeur de démo n'en a pas : ce
// fichier s'arrête donc aux verdicts que le GARDE rend avant tout accès au store
// (503 / 401), qui sont exactement ceux que le runbook fait vérifier en production.
//
// Driver DuckDB requis (tag cgo, dépendance transitive du routeur assemblé).

package api_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/api/openapigen"
)

const (
	// workerClaimPath — la route exacte que `replay-worker --once` appelle en
	// premier, et celle du curl de pré-vol du runbook (§3).
	workerClaimPath = "/api/v1/internal/build-queue/claim"
	// mutatingPathOutsideWorker — route mutatrice HORS protocole ouvrier : le
	// témoin de non-régression du contrôle CSRF sur le reste de l'API.
	mutatingPathOutsideWorker = "/api/v1/session/context"
)

// buildStackRouter assemble le VRAI routeur (pile transverse comprise) en mode
// démo. workerToken vide = protocole fermé (défaut de production tant que
// LEVELUP_BUILD_WORKER_TOKEN n'est pas posé).
func buildStackRouter(t *testing.T, workerToken string) http.Handler {
	t.Helper()
	router, _, err := openapigen.BuildDemoRouter(t.Context(), openapigen.Options{
		Root:             t.TempDir(),
		GroupStorePath:   filepath.Join(t.TempDir(), "groups.json"),
		Prestige:         true,
		BuildWorkerToken: workerToken,
	})
	if err != nil {
		t.Fatalf("BuildDemoRouter: %v", err)
	}
	return router
}

// postNu émet le POST tel qu'un ouvrier l'émet : corps JSON, AUCUN en-tête Origin,
// AUCUN Referer, AUCUN cookie. C'est la forme exacte que produit un client
// net/http (cmd/replay-worker) — et exactement celle que le CSRF-par-origine
// refusait.
func postNu(t *testing.T, router http.Handler, target string, extraHeaders map[string]string) (int, http.Header, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{"worker_id":"ouvrier-de-test"}`))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	// Garde-fou du harnais lui-même : si ces trois en-têtes apparaissaient, le test
	// ne prouverait plus rien du cas ouvrier.
	if req.Header.Get("Origin") != "" || req.Header.Get("Referer") != "" || len(req.Cookies()) != 0 {
		t.Fatalf("le harnais doit émettre une requête NUE (ni Origin, ni Referer, ni cookie)")
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr.Code, rr.Result().Header, rr.Body.String()
}

// TestTransverseStack_ProtocoleOuvrier_TraverseLeCSRF — LE test du lot. Sans
// jeton configuré côté serveur, la requête nue doit atteindre le garde de jeton et
// recevoir SON verdict (503 build_queue_disabled), jamais un 403 csrf_rejected.
func TestTransverseStack_ProtocoleOuvrier_TraverseLeCSRF(t *testing.T) {
	code, _, body := postNu(t, buildStackRouter(t, ""), workerClaimPath, nil)

	if code == http.StatusForbidden || strings.Contains(body, "csrf_rejected") {
		t.Fatalf("RÉGRESSION du défaut mesuré le 2026-08-25 : le protocole ouvrier est "+
			"rejeté par le CSRF transverse avant tout contrôle de jeton (code=%d, body=%s)", code, body)
	}
	if code == http.StatusNotFound {
		t.Fatalf("route absente du routeur assemblé : le protocole ouvrier n'est plus monté (body=%s)", body)
	}
	if code != http.StatusServiceUnavailable {
		t.Errorf("sans jeton configuré, verdict attendu 503 (garde de jeton), got %d (body=%s)", code, body)
	}
	if !strings.Contains(body, "build_queue_disabled") {
		t.Errorf("le corps doit porter le verdict du garde de jeton (build_queue_disabled), got %s", body)
	}
}

// TestTransverseStack_ProtocoleOuvrier_JetonToujoursExige — l'exemption ne lève QUE
// le contrôle d'origine : avec un jeton configuré côté serveur, un jeton client
// faux est toujours refusé en 401. C'est la question que pose une revue de
// sécurité (« l'exemption ouvre-t-elle la porte ? ») : non, la porte est le jeton.
func TestTransverseStack_ProtocoleOuvrier_JetonToujoursExige(t *testing.T) {
	router := buildStackRouter(t, "jeton-serveur-de-test") // pragma: allowlist secret
	code, _, body := postNu(t, router, workerClaimPath, map[string]string{
		"Authorization": "Bearer mauvais-jeton",
	})

	if code == http.StatusForbidden || strings.Contains(body, "csrf_rejected") {
		t.Fatalf("le CSRF rejette encore le protocole ouvrier (code=%d, body=%s)", code, body)
	}
	if code != http.StatusUnauthorized {
		t.Errorf("jeton client faux : attendu 401, got %d (body=%s)", code, body)
	}
	if !strings.Contains(body, "invalid_worker_token") {
		t.Errorf("le corps doit porter invalid_worker_token, got %s", body)
	}
}

// TestTransverseStack_ProtocoleOuvrier_GardeLeResteDeLaPile — l'exemption n'est PAS
// un montage « bare » : la réponse du protocole ouvrier porte toujours les en-têtes
// posés par la pile transverse. Si les routes avaient été sorties de la pile pour
// contourner le CSRF, ces en-têtes auraient disparu (et avec eux le rate-limit et
// les logs d'audit, qui ne laissent pas de trace observable dans la réponse).
func TestTransverseStack_ProtocoleOuvrier_GardeLeResteDeLaPile(t *testing.T) {
	_, headers, _ := postNu(t, buildStackRouter(t, ""), workerClaimPath, nil)

	if headers.Get("X-Request-ID") == "" {
		t.Error("X-Request-ID absent : le middleware RequestID ne s'applique plus au protocole ouvrier")
	}
	if got := headers.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q : SecurityHeaders ne s'applique plus au protocole ouvrier", got)
	}
}

// TestTransverseStack_HorsProtocoleOuvrier_CSRFToujoursActif — non-régression : une
// route mutatrice hors préfixe ouvrier, sans Origin, reste rejetée en 403.
func TestTransverseStack_HorsProtocoleOuvrier_CSRFToujoursActif(t *testing.T) {
	code, _, body := postNu(t, buildStackRouter(t, ""), mutatingPathOutsideWorker, nil)

	if code != http.StatusForbidden {
		t.Errorf("POST mutateur hors protocole ouvrier sans Origin : attendu 403, got %d (body=%s)", code, body)
	}
	if !strings.Contains(body, "csrf_rejected") {
		t.Errorf("le corps doit porter csrf_rejected, got %s", body)
	}
}

// TestTransverseStack_TraverseeNeContournePasLeCSRF — un chemin qui PORTE le
// préfixe exempté mais DÉSIGNE une route protégée reste rejeté par le CSRF. Vérifié
// sur la vraie pile, pas seulement en unitaire.
func TestTransverseStack_TraverseeNeContournePasLeCSRF(t *testing.T) {
	router := buildStackRouter(t, "")
	for _, target := range []string{
		"/api/v1/internal/../session/context",
		"/api/v1/internal/build-queue/../../session/context",
	} {
		code, _, body := postNu(t, router, target, nil)
		if code != http.StatusForbidden || !strings.Contains(body, "csrf_rejected") {
			t.Errorf("POST %q (traversée) doit rester 403 csrf_rejected, got %d (body=%s)", target, code, body)
		}
	}
}
