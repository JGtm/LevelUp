package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mediapkg "levelup/go-api/internal/media"
)

// Un remux dont la place ffmpeg n'a pas été libérée dans le délai répond 503 + Retry-After,
// avant d'avoir écrit le moindre octet du flux.
func TestRespondRemuxBusy(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "video/webm")
	busy := fmt.Errorf("attente de la place ffmpeg: %w", mediapkg.ErrFFmpegBusy)
	if !respondRemuxBusy(context.Background(), rec, "clip.mkv", busy) {
		t.Fatal("ErrFFmpegBusy doit être traitée comme « occupé »")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("statut = %d, attendu 503", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After absent")
	}
	if ct := rec.Header().Get("Content-Type"); ct == "video/webm" {
		t.Fatalf("Content-Type %q laissé sur une réponse d'erreur", ct)
	}

	other := httptest.NewRecorder()
	if respondRemuxBusy(context.Background(), other, "clip.mkv", errors.New("ffmpeg remux: exit status 1")) {
		t.Fatal("un échec de ffmpeg n'est pas « occupé »")
	}
	if other.Code != http.StatusOK || other.Body.Len() != 0 {
		t.Fatal("rien ne doit être écrit pour une erreur qui n'est pas « occupé »")
	}
}
