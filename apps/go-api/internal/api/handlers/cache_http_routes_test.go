package handlers_test

// cache_http_routes_test.go — preuve de bout en bout que les trois routes blob du
// plan (replay background.png, tactical background.png, assets medal image)
// passent bien par servirBlobAvecETag (D8) : 200 + ETag au 1er appel, 304 sans
// corps au 2e avec If-None-Match égal, 304 sur une liste "W/"x", <etag>"", 200 sur
// un en-tête illisible.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/port"
)

// verifieCycleETag exécute le scénario commun aux trois routes blob : 200+ETag,
// 304 sur match exact, 304 sur liste avec préfixe faible, 200 sur en-tête illisible.
func verifieCycleETag(t *testing.T, faireRequete func(ifNoneMatch string) *httptest.ResponseRecorder) {
	t.Helper()

	w1 := faireRequete("")
	if w1.Code != http.StatusOK {
		t.Fatalf("1re requête : status = %d, attendu 200 — body=%s", w1.Code, w1.Body.String())
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("1re requête : ETag absent")
	}

	w2 := faireRequete(etag)
	if w2.Code != http.StatusNotModified {
		t.Fatalf("2e requête (If-None-Match exact) : status = %d, attendu 304 — body=%s",
			w2.Code, w2.Body.String())
	}
	if w2.Body.Len() != 0 {
		t.Errorf("2e requête : corps non vide sur 304 (%d octets)", w2.Body.Len())
	}
	if cl := w2.Header().Get("Content-Length"); cl != "" && cl != "0" {
		t.Errorf("2e requête : Content-Length = %q, attendu absent/0 sur 304", cl)
	}

	w3 := faireRequete(`W/"autre", ` + etag)
	if w3.Code != http.StatusNotModified {
		t.Errorf("3e requête (liste W/ + etag) : status = %d, attendu 304", w3.Code)
	}

	w4 := faireRequete("valeur-qui-ne-correspond-a-rien")
	if w4.Code != http.StatusOK {
		t.Errorf("4e requête (en-tête illisible) : status = %d, attendu 200", w4.Code)
	}
}

func TestReplayBackgroundImage_CycleETag(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return fondMock(), nil }
	r := newReplayRouter(factory)
	verifieCycleETag(t, func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet,
			"/players/"+testPlayerSlug+"/matches/000d5950/replay/background.png", nil)
		req.RemoteAddr = "127.0.0.1:5432"
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	})
}

func TestTacticalBackgroundImage_CycleETag(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x42}
	r := routeurFond(&mockReplayService{imageMap: png})
	verifieCycleETag(t, func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/players/JGtm/tactical/streets/background.png", nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	})
}

func TestAssetMedalImage_CycleETag(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.BinaryPayload{
			ContentType: "image/png",
			Bytes:       []byte{0x89, 0x50, 0x4e, 0x47},
			ETag:        "abc123", // provenance amont : ignoré par le helper, qui recalcule le sien (D7)
		}},
	}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	verifieCycleETag(t, func(ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/12345/image", nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	})
}
