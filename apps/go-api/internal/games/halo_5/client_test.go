package halo_5

import (
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClient_RequestRecipe verrouille la recette de requete Halo 5 confirmee par
// la sonde live : header Spartan v4, User-Agent cpprestsdk, query ?auth=st, et
// SURTOUT pas de 343-clearance (Halo 5 ne l'utilise pas).
func TestClient_RequestRecipe(t *testing.T) {
	const token = "v4=abc.def.ghi"
	var gotPath, gotSpartan, gotUA, gotClearance, gotAuthQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSpartan = r.Header.Get("X-343-Authorization-Spartan")
		gotUA = r.Header.Get("User-Agent")
		gotClearance = r.Header.Get("343-clearance")
		gotAuthQuery = r.URL.Query().Get("auth")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixtureServiceRecord))
	}))
	defer srv.Close()

	c := NewClient(token, 0).WithBaseURLs(srv.URL, srv.URL)
	resp, err := c.GetServiceRecords(context.Background(), "JGtm", "arena")
	if err != nil {
		t.Fatalf("GetServiceRecords: %v", err)
	}
	if gotPath != "/h5/servicerecords/arena" {
		t.Errorf("path = %q, want /h5/servicerecords/arena (segment h5, gamertag en query)", gotPath)
	}
	if gotSpartan != token {
		t.Errorf("header Spartan = %q, want %q", gotSpartan, token)
	}
	if gotUA != "cpprestsdk/2.4.0" {
		t.Errorf("User-Agent = %q, want cpprestsdk/2.4.0", gotUA)
	}
	if gotClearance != "" {
		t.Errorf("343-clearance = %q, NE DOIT PAS etre envoye en Halo 5", gotClearance)
	}
	if gotAuthQuery != "st" {
		t.Errorf("query auth = %q, want st", gotAuthQuery)
	}
	if firstArenaResult(resp) == nil {
		t.Errorf("reponse mal parsee (ArenaStats absent)")
	}
}

// TestClient_GzipTransparent verifie qu'on ne force PAS Accept-Encoding (sinon
// net/http laisse le corps gzip compresse). Le serveur renvoie du gzip ; le client
// doit le decompresser de maniere transparente.
func TestClient_GzipTransparent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ae := r.Header.Get("Accept-Encoding"); strings.Contains(ae, "identity") {
			t.Errorf("Accept-Encoding=%q : le client ne doit pas desactiver gzip", ae)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(fixtureMatches))
		_ = gz.Close()
	}))
	defer srv.Close()

	c := NewClient("v4=x", 0).WithBaseURLs(srv.URL, srv.URL)
	resp, err := c.GetPlayerMatches(context.Background(), "JGtm", 0, 5)
	if err != nil {
		t.Fatalf("GetPlayerMatches (gzip): %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Id.MatchId == "" {
		t.Errorf("corps gzip mal decompresse/parse : %+v", resp)
	}
}

// TestClient_Unauthorized verifie qu'un 401 remonte un *HTTPError terminal (pas de retry).
func TestClient_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("v4=expired", 0).WithBaseURLs(srv.URL, srv.URL)
	_, err := c.GetServiceRecords(context.Background(), "JGtm", "arena")
	if err == nil {
		t.Fatal("401 doit produire une erreur")
	}
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusUnauthorized {
		t.Errorf("attendu *HTTPError 401, got %v", err)
	}
}
