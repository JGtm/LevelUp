package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestCommendationHandler_RefuseLaSortieDuDossier : aucune forme de `..` (en clair ou
// encodée) ne fait lire un fichier hors de `<static>/commendations`, tandis qu'une
// image légitime, y compris à nom encodé, reste servie.
func TestCommendationHandler_RefuseLaSortieDuDossier(t *testing.T) {
	root := t.TempDir()
	static := filepath.Join(root, "static")
	commendations := filepath.Join(static, "commendations")
	if err := os.MkdirAll(commendations, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commendations, "Killing Spree's.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.json"), []byte(`{"token":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(static, "voisin.txt"), []byte("voisin"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newCommendationHandler(static)
	serve := func(target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for _, target := range []string{
		"/static/commendations/..%2f..%2fsecret.json",
		"/static/commendations/..%2F..%2Fsecret.json",
		"/static/commendations/%2e%2e%2f%2e%2e%2fsecret.json",
		"/static/commendations/..%5c..%5csecret.json",
		"/static/commendations/..%2fvoisin.txt",
		"/static/commendations/%2e%2e/voisin.txt",
		"/static/commendations/",
	} {
		if rec := serve(target); rec.Code != http.StatusNotFound {
			t.Errorf("%s : statut %d, attendu 404 (corps %q)", target, rec.Code, rec.Body.String())
		}
	}

	for _, target := range []string{
		"/static/commendations/Killing%20Spree's.png",
		"/static/commendations/Killing%20Spree%27s.png",
	} {
		rec := serve(target)
		if rec.Code != http.StatusOK || rec.Body.String() != "png" {
			t.Errorf("%s : statut %d corps %q, attendu 200 png", target, rec.Code, rec.Body.String())
		}
	}
}
