// Package handlers_test — media_upload_content_test.go : à l'envoi, le contenu d'un fichier
// doit concorder avec la famille de son extension (media.ContentMatchesExtension), sinon 415
// upload_content_mismatch avant toute écriture.
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// En-têtes réels minimaux, pour les tests d'envoi de ce paquet.
var (
	uploadHeadMP4  = []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")
	uploadHeadMOV  = []byte("\x00\x00\x00\x14ftypqt  \x00\x00\x02\x00qt  ")
	uploadHeadWebM = []byte("\x1a\x45\xdf\xa3\x9f\x42\x86\x81\x01\x42\xf7\x81\x01webm")
	uploadHeadAVI  = []byte("RIFF\x00\x00\x00\x00AVI LIST")
	uploadHeadPNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	uploadHeadJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	uploadHeadGIF  = []byte("GIF89a\x01\x00\x01\x00")
	uploadHeadBMP  = []byte("BM\x36\x00\x00\x00\x00\x00\x00\x00\x36\x00\x00\x00")
)

// uploadSpy compte les appels au service d'envoi.
type uploadSpy struct {
	mockMediaService
	calls int
}

func (s *uploadSpy) UploadMedia(context.Context, domain.UploadRequest) (*domain.UploadResult, error) {
	s.calls++
	return &domain.UploadResult{Saved: 1}, nil
}

func newUploadSpyRouter(spy *uploadSpy) http.Handler {
	return newUploadRouter(
		func(_ context.Context, _ string) (port.MediaService, error) { return spy, nil },
		func(_ context.Context, _ string) (port.MediaService, string, string, string, string, string, error) {
			return spy, "TestPlayer", testTitleSlug, "/data/players/TestPlayer/stats.duckdb", "", "", nil
		},
	)
}

func TestUploadHandler_ContentMustMatchExtension(t *testing.T) {
	html := []byte("<!DOCTYPE html><html><script>alert(document.cookie)</script></html>")
	refused := map[string][]byte{
		"page.png":  html,
		"clip.mp4":  []byte("not a video, just text"),
		"image.gif": []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"clip.mkv":  uploadHeadPNG,
		"shot.jpg":  uploadHeadMP4,
	}
	for name, data := range refused {
		t.Run(name, func(t *testing.T) {
			spy := &uploadSpy{}
			w := httptest.NewRecorder()
			newUploadSpyRouter(spy).ServeHTTP(w,
				buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{name: data}))
			if w.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status = %d, attendu 415 (%s)", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "upload_content_mismatch") {
				t.Fatalf("corps = %s, attendu le code upload_content_mismatch", w.Body.String())
			}
			if spy.calls != 0 {
				t.Fatalf("service appelé %d fois pour un contenu refusé", spy.calls)
			}
		})
	}
}

// TestUploadHandler_MismatchRefusesTheWholeRequest : un seul fichier non concordant refuse
// l'envoi entier, rien n'est écrit.
func TestUploadHandler_MismatchRefusesTheWholeRequest(t *testing.T) {
	spy := &uploadSpy{}
	w := httptest.NewRecorder()
	newUploadSpyRouter(spy).ServeHTTP(w, buildMultipartRequest(t, "/players/test-player/media/upload",
		map[string][]byte{"clip.mp4": uploadHeadMP4, "evil.png": []byte("<html></html>")}))
	if w.Code != http.StatusUnsupportedMediaType || spy.calls != 0 {
		t.Fatalf("status = %d, appels = %d : attendu 415 sans appel", w.Code, spy.calls)
	}
}

// TestUploadHandler_LegitimateFormatsAccepted : chaque format accepté à l'envoi, avec un
// en-tête réel, atteint le service (vidéo QuickTime et MP4 compris).
func TestUploadHandler_LegitimateFormatsAccepted(t *testing.T) {
	accepted := map[string][]byte{
		"clip.mp4": uploadHeadMP4, "clip.mov": uploadHeadMOV, "clip.webm": uploadHeadWebM,
		"clip.mkv": uploadHeadWebM, "clip.avi": uploadHeadAVI, "CLIP.MP4": uploadHeadMP4,
		"shot.png": uploadHeadPNG, "shot.jpg": uploadHeadJPEG, "shot.jpeg": uploadHeadJPEG,
		"shot.gif": uploadHeadGIF, "shot.bmp": uploadHeadBMP,
	}
	for name, data := range accepted {
		t.Run(name, func(t *testing.T) {
			spy := &uploadSpy{}
			w := httptest.NewRecorder()
			newUploadSpyRouter(spy).ServeHTTP(w,
				buildMultipartRequest(t, "/players/test-player/media/upload", map[string][]byte{name: data}))
			if w.Code != http.StatusOK || spy.calls != 1 {
				t.Fatalf("status = %d, appels = %d : attendu 200 et 1 appel (%s)", w.Code, spy.calls, w.Body.String())
			}
		})
	}
}
