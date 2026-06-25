package halo_5

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fixtureAppearance = `{"Model":{"Gender":0},"Emblem":{"ColorPrimary":46,"ColorSecondary":46,"ColorTertiary":24,"EmblemId":264,"HarmonyGroupIndex":30,"HarmonyIndex":2},"StanceId":7024,"Gamertag":"JGtm","ServiceTag":"OKLM","StatusCode":0,"Company":{"Id":"abc","Name":"HaloFrance"}}`

// pngMagic est l'en-tête PNG minimal (8 octets de signature) suffisant pour valider
// le chemin binaire sans embarquer un vrai PNG.
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}

// TestClient_GetAppearance verrouille le parse du JSON appearance (service tag +
// emblème) et la recette de requête (header Spartan, UA, ?auth=st, host PROFILS).
func TestClient_GetAppearance(t *testing.T) {
	const token = "v4=abc"
	var gotPath, gotSpartan, gotAuth, gotClearance string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSpartan = r.Header.Get("X-343-Authorization-Spartan")
		gotAuth = r.URL.Query().Get("auth")
		gotClearance = r.Header.Get("343-clearance")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixtureAppearance))
	}))
	defer srv.Close()

	c := NewClient(token, 0).WithProfilesBaseURL(srv.URL)
	app, err := c.GetAppearance(context.Background(), "JGtm")
	if err != nil {
		t.Fatalf("GetAppearance: %v", err)
	}
	if gotPath != "/h5/profiles/JGtm/appearance" {
		t.Errorf("path = %q, want /h5/profiles/JGtm/appearance", gotPath)
	}
	if gotSpartan != token {
		t.Errorf("Spartan header = %q, want %q", gotSpartan, token)
	}
	if gotAuth != "st" {
		t.Errorf("auth query = %q, want st", gotAuth)
	}
	if gotClearance != "" {
		t.Errorf("343-clearance = %q, NE DOIT PAS etre envoye", gotClearance)
	}
	if app.ServiceTag != "OKLM" {
		t.Errorf("ServiceTag = %q, want OKLM", app.ServiceTag)
	}
	if app.Emblem.EmblemId != 264 || app.Emblem.ColorPrimary != 46 || app.Emblem.ColorTertiary != 24 {
		t.Errorf("Emblem mal parse : %+v", app.Emblem)
	}
}

// TestClient_GetEmblemPNG_FollowsRedirect verifie que GetEmblemPNG suit le 302 vers
// le CDN et telecharge les octets PNG (le shape live : haloplayer renvoie un 302
// signe vers image.halocdn.com).
func TestClient_GetEmblemPNG_FollowsRedirect(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngMagic)
	}))
	defer cdn.Close()

	var gotPath string
	profiles := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		http.Redirect(w, r, cdn.URL+"/h5/emblems/264_46_46_24", http.StatusFound)
	}))
	defer profiles.Close()

	c := NewClient("v4=x", 0).WithProfilesBaseURL(profiles.URL)
	b, ct, err := c.GetEmblemPNG(context.Background(), "JGtm")
	if err != nil {
		t.Fatalf("GetEmblemPNG: %v", err)
	}
	if gotPath != "/h5/profiles/JGtm/emblem" {
		t.Errorf("path = %q, want /h5/profiles/JGtm/emblem", gotPath)
	}
	if len(b) != len(pngMagic) || b[0] != 0x89 {
		t.Errorf("corps PNG mal recupere apres redirect : %d octets", len(b))
	}
	if ct != "image/png" {
		t.Errorf("content-type = %q, want image/png", ct)
	}
}

// TestClient_GetSpartanRenderPNG_FollowsRedirect : meme garde pour le rendu Spartan.
func TestClient_GetSpartanRenderPNG_FollowsRedirect(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngMagic)
	}))
	defer cdn.Close()

	profiles := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cdn.URL+"/h5/spartans/1014_0", http.StatusFound)
	}))
	defer profiles.Close()

	c := NewClient("v4=x", 0).WithProfilesBaseURL(profiles.URL)
	b, _, err := c.GetSpartanRenderPNG(context.Background(), "JGtm")
	if err != nil {
		t.Fatalf("GetSpartanRenderPNG: %v", err)
	}
	if len(b) != len(pngMagic) {
		t.Errorf("corps PNG mal recupere : %d octets", len(b))
	}
}

// TestClient_GetEmblemPNG_NotFound verifie qu'un 404 sur le profil remonte un
// *HTTPError terminal.
func TestClient_GetEmblemPNG_NotFound(t *testing.T) {
	profiles := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer profiles.Close()

	c := NewClient("v4=x", 0).WithProfilesBaseURL(profiles.URL)
	if _, _, err := c.GetEmblemPNG(context.Background(), "JGtm"); err == nil {
		t.Fatal("404 doit produire une erreur")
	}
}
