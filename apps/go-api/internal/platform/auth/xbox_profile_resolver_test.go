package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXboxProfileResolver_ResolveXUID(t *testing.T) {
	const body = `{"profileUsers":[{"id":"2533274895653213","settings":[{"id":"Gamertag","value":"Neo"}]}]}`
	var gotAuth, gotPath, gotContract string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotContract = r.Header.Get("x-xbl-contract-version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	r := NewXboxProfileResolver(srv.Client(), staticHeader)
	r.baseURL = srv.URL

	xuid, err := r.ResolveXUID(context.Background(), "Neo")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if xuid != "2533274895653213" {
		t.Errorf("xuid = %q, want 2533274895653213", xuid)
	}
	if gotAuth != "XBL3.0 x=hash;token" {
		t.Errorf("Authorization header = %q", gotAuth)
	}
	if gotContract != "2" {
		t.Errorf("x-xbl-contract-version = %q, want 2", gotContract)
	}
	// Le gamertag voyage dans le PATH sous la forme gt(<Gamertag>).
	if !strings.Contains(gotPath, "/gt(Neo)/profile/settings") {
		t.Errorf("path = %q, want .../gt(Neo)/profile/settings", gotPath)
	}
}

func TestXboxProfileResolver_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"profileUsers":[]}`))
	}))
	defer srv.Close()

	r := NewXboxProfileResolver(srv.Client(), staticHeader)
	r.baseURL = srv.URL
	if _, err := r.ResolveXUID(context.Background(), "Ghost"); err == nil {
		t.Error("attendu une erreur quand aucun profil n'est retourné")
	}
}

func TestXboxProfileResolver_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	r := NewXboxProfileResolver(srv.Client(), staticHeader)
	r.baseURL = srv.URL
	if _, err := r.ResolveXUID(context.Background(), "Neo"); err == nil {
		t.Error("attendu une erreur sur HTTP 401")
	}
}

func TestXboxProfileResolver_EmptyGamertag(t *testing.T) {
	r := NewXboxProfileResolver(http.DefaultClient, staticHeader)
	if _, err := r.ResolveXUID(context.Background(), "  "); err == nil {
		t.Error("attendu une erreur pour un gamertag vide")
	}
}
