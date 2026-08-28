package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCiblesDedoublonneEtGardeLOrdre — deux sources peuvent nommer la meme carte (le drapeau
// et le fichier), et une campagne doit rester reproductible : meme entree, meme ordre.
func TestCiblesDedoublonneEtGardeLOrdre(t *testing.T) {
	dir := t.TempDir()
	liste := filepath.Join(dir, "cartes.txt")
	contenu := "# les cartes du lot\nbbb\naaa  # commentaire en fin de ligne\n\nbbb\n"
	if err := os.WriteFile(liste, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := cibles(options{mapIDs: []string{"aaa", "ccc"}, depuis: liste})
	if err != nil {
		t.Fatalf("cibles: %v", err)
	}
	want := []string{"aaa", "ccc", "bbb"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("cibles = %v, attendu %v", got, want)
	}
}

// TestCiblesRefuseUneListeIntrouvable — un fichier de campagne absent doit ARRETER la
// commande, pas la faire tourner a vide sur les seules cartes du drapeau.
func TestCiblesRefuseUneListeIntrouvable(t *testing.T) {
	if _, err := cibles(options{depuis: filepath.Join(t.TempDir(), "absent.txt")}); err == nil {
		t.Fatal("liste introuvable acceptee")
	}
}

const pageAvecFichiers = `<html><head><script id="__NEXT_DATA__" type="application/json">
{"props":{"pageProps":{"asset":{"Files":{"Prefix":"https://blobs.example/ugcstorage/map/a/b/",
"FileRelativePaths":["map.mvar","navmesh.blob","lightprobes.blob"]}}}}}
</script></head><body>ignore</body></html>`

const pageSansNavmesh = `<script id="__NEXT_DATA__" type="application/json">
{"props":{"pageProps":{"asset":{"Files":{"Prefix":"https://blobs.example/x/","FileRelativePaths":["map.mvar"]}}}}}
</script>`

// TestRapatrieEcritLeBlobEnEntier — le chemin nominal, de bout en bout : la page publique
// resout le prefixe, le blob descend, le fichier final porte exactement les octets servis.
func TestRapatrieEcritLeBlobEnEntier(t *testing.T) {
	charge := strings.Repeat("N", 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/"+nomBlobNavmesh) {
			_, _ = w.Write([]byte(charge))
			return
		}
		_, _ = w.Write([]byte(strings.Replace(pageAvecFichiers, "https://blobs.example/ugcstorage/map/a/b/",
			srvURL(r)+"/blob/", 1)))
	}))
	defer srv.Close()

	c := &client{http: srv.Client()}
	dest := filepath.Join(t.TempDir(), "carte.blob")
	n, err := c.rapatrieDepuis(context.Background(), srv.URL+"/asset", dest, false)
	if err != nil {
		t.Fatalf("rapatrie: %v", err)
	}
	if n != int64(len(charge)) {
		t.Fatalf("octets ecrits = %d, attendu %d", n, len(charge))
	}
	lu, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(lu) != charge {
		t.Fatalf("contenu ecrit different de la charge servie (%d octets)", len(lu))
	}
	// Aucun fichier temporaire ne doit survivre : une reprise les prendrait pour du travail.
	restes, _ := filepath.Glob(filepath.Join(filepath.Dir(dest), ".navmesh-*"))
	if len(restes) != 0 {
		t.Fatalf("fichiers temporaires laisses derriere : %v", restes)
	}
}

// TestAssetSansNavmeshNEstPasUneErreur — sous ~1 000 objets l asset ne publie pas de maillage.
// C est le cas NOMINAL : il doit se distinguer d une panne, sinon une campagne entiere se
// declare en echec sur des cartes qui n ont simplement rien a rapatrier.
func TestAssetSansNavmeshNEstPasUneErreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageSansNavmesh))
	}))
	defer srv.Close()
	c := &client{http: srv.Client()}
	_, err := c.rapatrieDepuis(context.Background(), srv.URL+"/asset", filepath.Join(t.TempDir(), "x.blob"), false)
	if !errors.Is(err, ErrPasDeNavmesh) {
		t.Fatalf("err = %v, attendu ErrPasDeNavmesh", err)
	}
}

// TestPageSansBlocJSONEchoue — une page servie sans son bloc de donnees (maintenance, mur de
// connexion) ne doit pas passer pour un asset vide.
func TestPageSansBlocJSONEchoue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>rien ici</body></html>"))
	}))
	defer srv.Close()
	c := &client{http: srv.Client()}
	if _, _, err := c.resoutDepuis(context.Background(), srv.URL); err == nil {
		t.Fatal("page sans __NEXT_DATA__ acceptee")
	}
}

// srvURL reconstitue l origine du serveur de test depuis la requete recue.
func srvURL(r *http.Request) string { return "http://" + r.Host }
