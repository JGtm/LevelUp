package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cacheBackend écrit un 200 minimal — cible aval des middlewares testés.
func cacheBackend() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCacheMaxAge_SetsHeaderOnGet(t *testing.T) {
	h := CacheMaxAge(600)(cacheBackend())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	got := rec.Header().Get("Cache-Control")
	want := "public, max-age=600, stale-while-revalidate=300"
	if got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestCacheMaxAge_NoHeaderOnNonGet(t *testing.T) {
	h := CacheMaxAge(600)(cacheBackend())
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/x", nil))
		if got := rec.Header().Get("Cache-Control"); got != "" {
			t.Errorf("%s: Cache-Control = %q, want vide (pas de cache sur mutations)", method, got)
		}
	}
}

func TestNoStore_AllMethods(t *testing.T) {
	h := NoStore(cacheBackend())
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/live", nil))
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", method, got)
		}
	}
}

func TestETagFromBytes_DeterministicDistinctFormat(t *testing.T) {
	a1 := ETagFromBytes([]byte("hello"))
	a2 := ETagFromBytes([]byte("hello"))
	b := ETagFromBytes([]byte("world"))

	if a1 != a2 {
		t.Errorf("ETag non déterministe : %q != %q", a1, a2)
	}
	if a1 == b {
		t.Errorf("ETags de corps distincts identiques : %q", a1)
	}
	// Format : 8 octets SHA-256 → 16 hex, entre guillemets → longueur 18.
	if len(a1) != 18 || !strings.HasPrefix(a1, `"`) || !strings.HasSuffix(a1, `"`) {
		t.Errorf("format ETag inattendu : %q (attendu \"<16 hex>\")", a1)
	}
}

func TestWriteJSONCached_200WithoutIfNoneMatch(t *testing.T) {
	body := []byte(`{"a":1}`)
	rec := httptest.NewRecorder()
	WriteJSONCached(rec, httptest.NewRequest(http.MethodGet, "/x", nil), http.StatusOK, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("ETag") != ETagFromBytes(body) {
		t.Errorf("ETag manquant ou incohérent : %q", rec.Header().Get("ETag"))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if rec.Body.String() != string(body) {
		t.Errorf("body = %q, want %q", rec.Body.String(), body)
	}
}

func TestWriteJSONCached_304OnMatchingETag(t *testing.T) {
	body := []byte(`{"a":1}`)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("If-None-Match", ETagFromBytes(body))

	rec := httptest.NewRecorder()
	WriteJSONCached(rec, req, http.StatusOK, body)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 ne doit pas porter de corps, got %q", rec.Body.String())
	}
}

func TestWriteJSONCached_200OnStaleETag(t *testing.T) {
	body := []byte(`{"a":1}`)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("If-None-Match", `"deadbeefdeadbeef"`) // version périmée

	rec := httptest.NewRecorder()
	WriteJSONCached(rec, req, http.StatusOK, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (ETag périmé → nouvelle version)", rec.Code)
	}
	if rec.Body.String() != string(body) {
		t.Errorf("body = %q, want %q", rec.Body.String(), body)
	}
}

// TestWriteJSONCached_NoTitleLeak — MT-25 : l'ETag dérive du corps, donc deux
// titres (corps différents) produisent des ETags différents ; un client porteur
// de l'ETag du titre A ne reçoit JAMAIS un 304 pour le corps du titre B.
func TestWriteJSONCached_NoTitleLeak(t *testing.T) {
	bodyTitleA := []byte(`{"title":"halo_infinite","kills":10}`)
	bodyTitleB := []byte(`{"title":"halo_5","kills":10}`)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	req.Header.Set("If-None-Match", ETagFromBytes(bodyTitleA)) // client a le titre A

	rec := httptest.NewRecorder()
	WriteJSONCached(rec, req, http.StatusOK, bodyTitleB) // serveur répond titre B

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — un 304 ici serait une fuite de cache inter-titre", rec.Code)
	}
	if rec.Body.String() != string(bodyTitleB) {
		t.Errorf("body = %q, want le corps du titre B", rec.Body.String())
	}
}
