package external

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestDiscordWebhookNotifier_PayloadCorrect vérifie qu'un embed coach bien formé
// est POSTé au webhook et que l'envoi réussi retourne nil.
func TestDiscordWebhookNotifier_PayloadCorrect(t *testing.T) {
	var body []byte
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := DiscordWebhookNotifier{}.Notify(context.Background(), srv.URL, ExternalNotification{
		Category: "milestone_unlocked",
		Severity: "success",
		Player:   "JGtm",
		Params:   map[string]any{"metric": "kills"},
		Lang:     "fr",
	})
	if err != nil {
		t.Fatalf("Notify err = %v ; want nil", err)
	}
	if gotCT == "" {
		t.Errorf("Content-Type manquant")
	}
	var payload struct {
		Embeds []struct {
			Title  string `json:"title"`
			Fields []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"embeds"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("payload illisible : %v (%s)", err, body)
	}
	if len(payload.Embeds) != 1 {
		t.Fatalf("embeds = %d ; want 1", len(payload.Embeds))
	}
	if payload.Embeds[0].Title == "" {
		t.Errorf("embed sans titre")
	}
	var playerSeen bool
	for _, f := range payload.Embeds[0].Fields {
		if f.Value == "JGtm" {
			playerSeen = true
		}
	}
	if !playerSeen {
		t.Errorf("joueur absent des champs de l'embed")
	}
}

// TestDiscordWebhookNotifier_TimeoutRespected vérifie qu'un webhook lent au-delà
// du timeout court retourne une erreur (jamais de blocage/panic).
func TestDiscordWebhookNotifier_TimeoutRespected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	start := time.Now()
	err := DiscordWebhookNotifier{Timeout: 20 * time.Millisecond}.Notify(
		context.Background(), srv.URL, ExternalNotification{Category: "personal_record"})
	elapsed := time.Since(start)
	if err == nil {
		t.Errorf("Notify err = nil ; want erreur (timeout)")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Notify a bloqué %v ; le timeout court n'a pas été respecté", elapsed)
	}
}

// TestDiscordWebhookNotifier_EmptyURL : webhook vide → erreur, aucun POST.
func TestDiscordWebhookNotifier_EmptyURL(t *testing.T) {
	if err := (DiscordWebhookNotifier{}).Notify(context.Background(), "", ExternalNotification{Category: "personal_record"}); err == nil {
		t.Errorf("Notify(url vide) err = nil ; want erreur")
	}
}

// TestDiscordWebhookNotifier_ServerErrorNoPanic : statut non-2xx → erreur sans panic.
func TestDiscordWebhookNotifier_ServerErrorNoPanic(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	err := DiscordWebhookNotifier{}.Notify(context.Background(), srv.URL, ExternalNotification{Category: "personal_record"})
	if err == nil {
		t.Errorf("Notify sur 500 err = nil ; want erreur")
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("hits = %d ; want 1", hits)
	}
}
