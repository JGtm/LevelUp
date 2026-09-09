package handlers

// cache_http_test.go — garde-rail anti-duplication (CLAUDE.md règle n°6) et tests
// unitaires du helper servirBlobAvecETag / de sa délégation par writeJSONCached.
//
// Amendement S5 du plan PLAN_FONDS_CARTE_WEBP_ETAG_2026-09-09.md, étape 1 : la
// logique ETag/304 est centralisée dans cache_http.go. Ce fichier verrouille deux
// choses distinctes :
//  1. qu'aucun autre fichier de production du paquet ne pose "ETag" ou ne lit
//     "If-None-Match" en littéral (le motif serait alors dupliqué une 4e fois) ;
//  2. que le helper lui-même respecte le contrat D7/D9 (ETag fort, liste
//     séparée par virgules, préfixe faible W/, wildcard *, en-tête illisible).

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// cacheHTTPDefFile : SEUL fichier autorisé à poser "ETag" ou lire "If-None-Match"
// en littéral dans le code de production du paquet — celui qui définit
// servirBlobAvecETag. Allowlist VIDE et datée : aucune autre exemption au
// 2026-09-09.
const cacheHTTPDefFile = "cache_http.go"

// reETagHeaderSet : l'appel littéral `Header().Set("ETag"`, guillemets compris.
var reETagHeaderSet = regexp.MustCompile(`Header\(\)\.Set\("ETag"`)

// reIfNoneMatchGet : l'appel littéral `Header.Get("If-None-Match")`.
var reIfNoneMatchGet = regexp.MustCompile(`Header\.Get\("If-None-Match"\)`)

// TestNoRawETagHandlingOutsideCacheHTTP — garde-rail CLAUDE.md règle n°6. Avant ce
// lot, le motif ETag/If-None-Match existait déjà en 2 endroits indépendants
// (writeJSONCached dans helpers.go, et l'ETag inerte de assets.go:217 — sans
// gestion de If-None-Match, donc sans jamais rendre de 304). Une 3e copie aurait
// été introduite par l'étape 1 sur replay.go/tactical.go : la règle des 2 copies
// impose donc le helper ET ce garde-rail.
//
// Périmètre : fichiers de production du paquet handlers, hors *_test.go (les
// tests lisent l'en-tête par son NOM HTTP réel pour vérifier le contrat client —
// leur imposer la constante testerait le garde-rail contre lui-même) et hors
// cache_http.go (le fichier de définition).
func TestNoRawETagHandlingOutsideCacheHTTP(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") ||
			name == cacheHTTPDefFile {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		if reETagHeaderSet.Match(src) {
			offenders = append(offenders, name+" (Header().Set(\"ETag\")")
		}
		if reIfNoneMatchGet.Match(src) {
			offenders = append(offenders, name+" (Header.Get(\"If-None-Match\"))")
		}
	}
	if len(offenders) > 0 {
		t.Errorf("logique ETag/If-None-Match dupliquée hors %s — passer par "+
			"servirBlobAvecETag :\n  %s", cacheHTTPDefFile, strings.Join(offenders, "\n  "))
	}
}

// TestETagGuardIsDiscriminant prouve que le garde-rail ci-dessus MORD, et qu'il
// ne mord QUE sur le littéral — jumeau de TestRetryAfterGuardIsDiscriminant
// (no_retry_after_literal_test.go), même méthode pour le même diagnostic anti-
// pattern n°8 (« factorisation abandonnée sans garde-rail »).
func TestETagGuardIsDiscriminant(t *testing.T) {
	mustMatch := []string{
		`w.Header().Set("ETag", etag)`,
		`if r.Header.Get("If-None-Match") == etag {`,
	}
	for _, src := range mustMatch {
		if !reETagHeaderSet.MatchString(src) && !reIfNoneMatchGet.MatchString(src) {
			t.Errorf("littéral NON détecté (garde-rail aveugle) : %q", src)
		}
	}

	mustNotMatch := []string{
		`w.Header().Set("Content-Type", contentType)`,
		`// pose l'ETag puis vérifie If-None-Match`,
		`etag := calculerETagFort(blob)`,
	}
	for _, src := range mustNotMatch {
		if reETagHeaderSet.MatchString(src) || reIfNoneMatchGet.MatchString(src) {
			t.Errorf("FAUX POSITIF sur usage légitime : %q", src)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	defPath := filepath.Join(filepath.Dir(thisFile), cacheHTTPDefFile)
	src, err := os.ReadFile(defPath)
	if err != nil {
		t.Fatalf("lecture du fichier de définition %s : %v", cacheHTTPDefFile, err)
	}
	if !reETagHeaderSet.Match(src) || !reIfNoneMatchGet.Match(src) {
		t.Errorf("%s ne porte plus la logique ETag/If-None-Match : son exemption "+
			"dans TestNoRawETagHandlingOutsideCacheHTTP est devenue un trou",
			cacheHTTPDefFile)
	}
}

// ---------------------------------------------------------------------------
// servirBlobAvecETag — contrat D7/D9
// ---------------------------------------------------------------------------

func TestServirBlobAvecETag_PremiereRequete_200EtETagNonVide(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/fond.png", nil)

	servirBlobAvecETag(w, r, []byte("contenu-image"), "image/png", "private, max-age=3600")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200", w.Code)
	}
	if w.Header().Get("ETag") == "" {
		t.Error("ETag absent")
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if cl := w.Header().Get("Content-Length"); cl != "13" {
		t.Errorf("Content-Length = %q, attendu 13", cl)
	}
	if w.Body.String() != "contenu-image" {
		t.Errorf("corps = %q", w.Body.String())
	}
}

func TestServirBlobAvecETag_EtagFort_FormatSha256(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	servirBlobAvecETag(w, r, []byte("abc"), "image/png", "")

	etag := w.Header().Get("ETag")
	if !strings.HasPrefix(etag, `"sha256-`) || !strings.HasSuffix(etag, `"`) {
		t.Fatalf("ETag = %q, attendu format \"sha256-<hex>\"", etag)
	}
	hex := strings.TrimSuffix(strings.TrimPrefix(etag, `"sha256-`), `"`)
	if len(hex) != 12 {
		t.Errorf("longueur hex = %d, attendu 12 (D7)", len(hex))
	}
}

func TestServirBlobAvecETag_IfNoneMatchExact_304SansCorps(t *testing.T) {
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	servirBlobAvecETag(w1, r1, []byte("contenu"), "image/png", "private, max-age=3600")
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag absent au 1er appel")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	r2.Header.Set("If-None-Match", etag)
	servirBlobAvecETag(w2, r2, []byte("contenu"), "image/png", "private, max-age=3600")

	if w2.Code != http.StatusNotModified {
		t.Fatalf("status = %d, attendu 304", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Errorf("corps non vide sur 304 : %d octets", w2.Body.Len())
	}
	if cl := w2.Header().Get("Content-Length"); cl != "" && cl != "0" {
		t.Errorf("Content-Length = %q, attendu absent ou 0 sur 304", cl)
	}
}

func TestServirBlobAvecETag_ListeAvecPrefixeFaible_304(t *testing.T) {
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	servirBlobAvecETag(w1, r1, []byte("contenu"), "image/png", "")
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	r2.Header.Set("If-None-Match", `W/"x", `+etag)
	servirBlobAvecETag(w2, r2, []byte("contenu"), "image/png", "")

	if w2.Code != http.StatusNotModified {
		t.Fatalf("status = %d, attendu 304 (liste avec W/ + etag)", w2.Code)
	}
}

func TestServirBlobAvecETag_Wildcard_304(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	r.Header.Set("If-None-Match", "*")
	servirBlobAvecETag(w, r, []byte("contenu"), "image/png", "")

	if w.Code != http.StatusNotModified {
		t.Fatalf("status = %d, attendu 304 (wildcard *)", w.Code)
	}
}

func TestServirBlobAvecETag_EnTeteIllisible_200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/fond.png", nil)
	r.Header.Set("If-None-Match", "n'importe-quoi-de-non-correspondant")
	servirBlobAvecETag(w, r, []byte("contenu"), "image/png", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu 200 (en-tête illisible/non correspondant)", w.Code)
	}
}

// ---------------------------------------------------------------------------
// writeJSONCached — route JSON cachée qui délègue désormais à servirBlobAvecETag
// (seul appelant JSON du helper : les routes Huma posent leur ETag via un champ
// de sortie déclaratif et ne peuvent pas appeler un writer direct — cf. § Découvertes).
// ---------------------------------------------------------------------------

func TestWriteJSONCached_ListeAvecPrefixeFaible_304(t *testing.T) {
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	writeJSONCached(w1, r1, http.StatusOK, map[string]string{"key": "value"})
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag absent au 1er appel")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	r2.Header.Set("If-None-Match", `W/"x", `+etag)
	writeJSONCached(w2, r2, http.StatusOK, map[string]string{"key": "value"})

	if w2.Code != http.StatusNotModified {
		t.Errorf("status = %d, attendu 304 (liste avec W/ + etag)", w2.Code)
	}
}

func TestWriteJSONCached_EnTeteIllisible_200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("If-None-Match", "valeur-qui-ne-correspond-a-rien")
	writeJSONCached(w, r, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, attendu 200", w.Code)
	}
}
