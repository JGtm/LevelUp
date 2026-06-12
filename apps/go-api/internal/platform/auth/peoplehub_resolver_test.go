package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func staticHeader(_ context.Context) (string, error) { return "XBL3.0 x=hash;token", nil }

func TestPeopleHubResolver_ResolveXUID(t *testing.T) {
	const body = `{"people":[
		{"gamertag":"OtherGuy","xuid":"111"},
		{"gamertag":"Neo","xuid":"2533274895653213"}
	]}`
	var gotAuth, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	r := NewPeopleHubResolver(srv.Client(), staticHeader)
	r.baseURL = srv.URL

	// Correspondance exacte case-insensitive parmi plusieurs résultats.
	xuid, err := r.ResolveXUID(context.Background(), "neo")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if xuid != "2533274895653213" {
		t.Errorf("xuid = %q, want 2533274895653213", xuid)
	}
	if gotAuth != "XBL3.0 x=hash;token" {
		t.Errorf("Authorization header = %q", gotAuth)
	}
	if gotQuery != "neo" {
		t.Errorf("query q = %q, want neo", gotQuery)
	}

	// Aucune correspondance exacte → erreur (pas de faux positif fuzzy).
	if _, err := r.ResolveXUID(context.Background(), "Ne"); err == nil {
		t.Error("attendu une erreur pour un gamertag sans correspondance exacte")
	}
}

func TestPeopleHubResolver_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	r := NewPeopleHubResolver(srv.Client(), staticHeader)
	r.baseURL = srv.URL
	if _, err := r.ResolveXUID(context.Background(), "Neo"); err == nil {
		t.Error("attendu une erreur sur HTTP 401")
	}
}
