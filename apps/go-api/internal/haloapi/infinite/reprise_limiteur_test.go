package infinite

// reprise_limiteur_test.go — les tests du transport du client de synchronisation d'avant la brique
// (attente de reprise, limiteur, verdict « absent » typé, adresses des fragments), portés sous leur
// nom sur le code de la brique qui les remplace : `Retry.attendre`, `Client.attendreLimiteur`,
// `IsNotFound`, `BuildChunkURL`.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestBackoff_Short(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	RetryHaloClient().attendre(ctx, 0, 0) // 2^0 × délai de base, sans Retry-After
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("attente trop longue : %v", elapsed)
	}
}

func TestBackoff_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé
	start := time.Now()
	RetryHaloClient().attendre(ctx, 2, 0)
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("l'attente doit rendre la main aussitôt sur un contexte annulé")
	}
}

// TestRetryBaseDelay_BorneBasse : la garantie de compilation d'une constante est remplacée par
// ce plancher (la politique est une valeur) ; un délai plus court transformerait une reprise
// bornée en martèlement du CDN.
func TestRetryBaseDelay_BorneBasse(t *testing.T) {
	const plancher = 500 * time.Millisecond
	if d := RetryHaloClient().DelaiBase; d < plancher {
		t.Fatalf("RetryHaloClient().DelaiBase = %v, attendu au moins %v", d, plancher)
	}
}

// clientLimite : un client sans Doer dont seul le limiteur sert.
func clientLimite(l *rate.Limiter) *Client {
	return New(nil, RetryHaloClient()).WithLimiter(l)
}

func TestRateWait_FirstCall(t *testing.T) {
	// Le seau commence plein (burst=1) : le premier appel n'attend pas.
	c := clientLimite(rate.NewLimiter(rate.Every(50*time.Millisecond), 1))
	start := time.Now()
	_ = c.attendreLimiteur(context.Background())
	if time.Since(start) > 10*time.Millisecond {
		t.Fatalf("le premier appel ne doit pas attendre, %v", time.Since(start))
	}
}

func TestRateWait_SecondCall(t *testing.T) {
	c := clientLimite(rate.NewLimiter(rate.Every(50*time.Millisecond), 1))
	_ = c.attendreLimiteur(context.Background()) // consomme le burst
	start := time.Now()
	_ = c.attendreLimiteur(context.Background())
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("attente ~50ms attendue, %v", elapsed)
	}
}

// TestRateWait_ConcurrentRespectsRPS : le débit tient sous concurrence (à lancer avec -race).
func TestRateWait_ConcurrentRespectsRPS(t *testing.T) {
	const rps = 20 // intervalle 50 ms
	const n = 10
	c := clientLimite(rate.NewLimiter(rate.Limit(rps), 1))
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = c.attendreLimiteur(context.Background())
		}()
	}
	wg.Wait()
	// (n-1) jetons après le burst initial, tolérance 80 % pour le jitter de l'ordonnanceur.
	minExpected := time.Duration(n-1) * time.Second / time.Duration(rps) * 80 / 100
	if elapsed := time.Since(start); elapsed < minExpected {
		t.Fatalf("limiteur inopérant : %d appels à %d RPS en %v (attendu ≥ %v)", n, rps, elapsed, minExpected)
	}
}

// TestIsNotFoundErr_TypeUniquement : le verdict « absent » est typé, le texte n'est pas une API.
func TestIsNotFoundErr_TypeUniquement(t *testing.T) {
	cas := []struct {
		nom    string
		err    error
		attend bool
	}{
		{"blob 404", &BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}, true},
		{"blob 410", &BlobHTTPError{StatusCode: 410, URL: "u", Attempts: 1}, true},
		{"blob 304", &BlobHTTPError{StatusCode: 304, URL: "u", Attempts: RetryHaloClient().Essais}, false},
		{"api 404", &HTTPError{StatusCode: 404, URL: "u"}, true},
		{"api 410", &HTTPError{StatusCode: 410, URL: "u"}, true},
		{"api 503", &HTTPError{StatusCode: 503, URL: "u"}, false},
		{"blob 404 enveloppé (errgroup)", fmt.Errorf("GetMatchFilm chunk 3: %w",
			&BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}), true},
		{"erreur quelconque", errors.New("connection refused"), false},
		{"texte HTTP 404", errors.New("HTTP 404"), false},
		{"texte ressource absente", errors.New("ressource absente"), false},
		{"nil", nil, false},
	}
	for _, c := range cas {
		if got := IsNotFound(c.err); got != c.attend {
			t.Errorf("%s : IsNotFound = %v, attendu %v", c.nom, got, c.attend)
		}
	}
}

func TestBuildChunkURL_EmptyRelativePath(t *testing.T) {
	if got := BuildChunkURL("https://blob.example.com/prefix/", ""); got != "https://blob.example.com/prefix/" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildChunkURL_BasicConcatenation(t *testing.T) {
	if got, want := BuildChunkURL("https://blob.example.com", "chunks/c0.bin"), "https://blob.example.com/chunks/c0.bin"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_TrailingSlashOnPrefix(t *testing.T) {
	if got, want := BuildChunkURL("https://blob.example.com/prefix/", "chunks/c0.bin"), "https://blob.example.com/prefix/chunks/c0.bin"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_PathWithLeadingSlash(t *testing.T) {
	if got, want := BuildChunkURL("https://blob.example.com", "/chunks/c0.bin"), "https://blob.example.com/chunks/c0.bin"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildChunkURL_EmptyPrefix(t *testing.T) {
	if got := BuildChunkURL("", "chunks/c0.bin"); got != "chunks/c0.bin" {
		t.Fatalf("got %q", got)
	}
}
