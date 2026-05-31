// Package handlers_test — media_hls_serve_test.go : sert un arbre HLS et vérifie
// les Content-Types (playlist + segment fMP4). Garantit aussi que le master
// n'est pas routé vers le remux WebM (sinon Content-Type video/webm).
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeMediaFile_HLSContentTypes(t *testing.T) {
	capturesBase := t.TempDir()
	hlsDir := filepath.Join(capturesBase, "JGtm", "hls", "clip")
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string][]byte{
		"master.m3u8":    []byte("#EXTM3U\n#EXT-X-VERSION:7\n"),
		"seg_a0_000.m4s": []byte("fakesegmentbytes"),
	}
	for name, data := range fixtures {
		if err := os.WriteFile(filepath.Join(hlsDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	cases := map[string]string{
		"JGtm/hls/clip/master.m3u8":    "application/vnd.apple.mpegurl",
		"JGtm/hls/clip/seg_a0_000.m4s": "video/mp4",
	}
	for rpath, wantCT := range cases {
		req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/"+rpath, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s: code %d, want 200", rpath, w.Code)
			continue
		}
		if ct := w.Header().Get("Content-Type"); ct != wantCT {
			t.Errorf("%s: Content-Type %q, want %q", rpath, ct, wantCT)
		}
	}
}
