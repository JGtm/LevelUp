// Package middleware — middleware_internal_test.go : tests internes des fonctions privées.
//
// Couvre : resolveTitleSlug.
//
// NOTE : tests contractResponseWriter / validateErrorShape SUPPRIMÉS avec le
// middleware ContractValidate (L4, 2026-07-05 — Huma dérive déjà le contrat).
// Tests errorTrackWriter / discordSimplePayload / … supprimés en revue
// 2026-04-29 P8.3 (ADR 0009 — error_tracker retiré).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestResolveTitleSlug_NoHeader_NoSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	reg := titlePkg.NewRegistry()
	slug := resolveTitleSlug(req, reg)
	// No header, no session → fallback to default slug.
	if slug == "" {
		t.Fatal("expected non-empty default slug")
	}
}
