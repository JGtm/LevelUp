package worldenrich

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// rtFunc adapte une fonction en http.RoundTripper : intercepte AVANT toute
// résolution réseau, donc l'URL PeopleHub hardcodée (host xboxlive) est sans
// importance — on sert des réponses canned déterministes.
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func staticHeader(_ context.Context) (string, error) { return "XBL3.0 x=h;t", nil }

// peopleHubFor construit un résolveur dont le transport répond toujours `body`
// avec `status`, et enregistre le nombre d'appels via le compteur fourni.
func peopleHubFor(status int, body string, calls *int) *auth.PeopleHubResolver {
	client := &http.Client{Transport: rtFunc(func(_ *http.Request) (*http.Response, error) {
		*calls++
		return jsonResp(status, body), nil
	})}
	return auth.NewPeopleHubResolver(client, staticHeader)
}

// TestCachingResolver_CacheHitFromSeed : un gamertag présent dans le seed
// (n'importe quelle casse) est servi depuis le cache, SANS toucher au résolveur
// PeopleHub (zéro appel HTTP), et incrémente le compteur de hits.
func TestCachingResolver_CacheHitFromSeed(t *testing.T) {
	calls := 0
	r := peopleHubFor(http.StatusOK, `{"people":[]}`, &calls)
	c := NewCachingResolver([]*auth.PeopleHubResolver{r}, map[string]string{"Neo": "111"}, nil)

	// Casse différente du seed → doit quand même résoudre depuis le cache.
	got, err := c.ResolveXUID(context.Background(), "neo")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if got != "111" {
		t.Errorf("xuid = %q, want 111", got)
	}
	if calls != 0 {
		t.Errorf("appels HTTP = %d, want 0 (cache hit)", calls)
	}
	hits, misses := c.Stats()
	if hits != 1 || misses != 0 {
		t.Errorf("Stats = (%d,%d), want (1,0)", hits, misses)
	}
}

// TestCachingResolver_MissResolvesAndPersistsAndCaches : un miss résout via
// PeopleHub, met en cache (2e appel = hit, pas de 2e HTTP) et déclenche le
// callback persist UNE fois avec la nouvelle association.
func TestCachingResolver_MissResolvesAndPersistsAndCaches(t *testing.T) {
	calls := 0
	r := peopleHubFor(http.StatusOK, `{"people":[{"gamertag":"Trinity","xuid":"222"}]}`, &calls)

	var persisted [][2]string
	c := NewCachingResolver([]*auth.PeopleHubResolver{r}, nil, func(gt, x string) {
		persisted = append(persisted, [2]string{gt, x})
	})

	got, err := c.ResolveXUID(context.Background(), "Trinity")
	if err != nil {
		t.Fatalf("ResolveXUID(miss): %v", err)
	}
	if got != "222" {
		t.Errorf("xuid = %q, want 222", got)
	}
	// 2e appel : doit venir du cache (pas de 2e requête).
	if _, err := c.ResolveXUID(context.Background(), "trinity"); err != nil {
		t.Fatalf("ResolveXUID(2e): %v", err)
	}
	if calls != 1 {
		t.Errorf("appels HTTP = %d, want 1 (2e résolution = cache)", calls)
	}
	if len(persisted) != 1 || persisted[0] != [2]string{"Trinity", "222"} {
		t.Errorf("persist = %v, want une seule association (Trinity,222)", persisted)
	}
	hits, misses := c.Stats()
	if hits != 1 || misses != 1 {
		t.Errorf("Stats = (%d,%d), want (1,1)", hits, misses)
	}
}

// TestCachingResolver_429RotatesToNextToken : un 429 sur le 1er résolveur fait
// passer (round-robin) au 2e, qui répond. Anti rate-limit : on étale les
// résolutions sur N comptes.
func TestCachingResolver_429RotatesToNextToken(t *testing.T) {
	calls429, callsOK := 0, 0
	r429 := peopleHubFor(http.StatusTooManyRequests, `{"error":"rate"}`, &calls429)
	rOK := peopleHubFor(http.StatusOK, `{"people":[{"gamertag":"Morpheus","xuid":"333"}]}`, &callsOK)

	c := NewCachingResolver([]*auth.PeopleHubResolver{r429, rOK}, nil, nil)
	got, err := c.ResolveXUID(context.Background(), "Morpheus")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if got != "333" {
		t.Errorf("xuid = %q, want 333", got)
	}
	if calls429 != 1 || callsOK != 1 {
		t.Errorf("appels = (429:%d, ok:%d), want (1,1) — rotation après 429", calls429, callsOK)
	}
}

// TestCachingResolver_NonRateLimitErrorDoesNotRotate : une erreur autre que 429
// (ici gamertag absent des résultats) est renvoyée immédiatement, SANS essayer
// le résolveur suivant — inutile de roter si le compte n'est pas en throttle.
func TestCachingResolver_NonRateLimitErrorDoesNotRotate(t *testing.T) {
	callsA, callsB := 0, 0
	// 200 mais people sans correspondance exacte → erreur "absent des résultats".
	rA := peopleHubFor(http.StatusOK, `{"people":[{"gamertag":"Autre","xuid":"999"}]}`, &callsA)
	rB := peopleHubFor(http.StatusOK, `{"people":[{"gamertag":"Cible","xuid":"444"}]}`, &callsB)

	c := NewCachingResolver([]*auth.PeopleHubResolver{rA, rB}, nil, nil)
	if _, err := c.ResolveXUID(context.Background(), "Cible"); err == nil {
		t.Fatal("attendu une erreur (pas de correspondance sur le 1er résolveur)")
	}
	if callsA != 1 || callsB != 0 {
		t.Errorf("appels = (A:%d, B:%d), want (1,0) — pas de rotation sur erreur non-429", callsA, callsB)
	}
}

// TestCachingResolver_AllRateLimitedReturnsLastErr : si TOUS les résolveurs sont
// en 429, la boucle round-robin essaie chacun exactement une fois puis renvoie la
// dernière erreur (et ne met RIEN en cache). Invariant : 1 essai par résolveur,
// pas de boucle infinie, et le miss est compté une seule fois.
func TestCachingResolver_AllRateLimitedReturnsLastErr(t *testing.T) {
	callsA, callsB := 0, 0
	rA := peopleHubFor(http.StatusTooManyRequests, `{"error":"rate-A"}`, &callsA)
	rB := peopleHubFor(http.StatusTooManyRequests, `{"error":"rate-B"}`, &callsB)

	c := NewCachingResolver([]*auth.PeopleHubResolver{rA, rB}, nil, func(string, string) {
		t.Fatal("persist ne doit pas être appelé quand la résolution échoue")
	})
	_, err := c.ResolveXUID(context.Background(), "Smith")
	if err == nil {
		t.Fatal("attendu une erreur (tous les résolveurs en 429)")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v, want une erreur 429 (dernière rencontrée)", err)
	}
	// Chaque résolveur essayé exactement une fois (round-robin borné par len).
	if callsA != 1 || callsB != 1 {
		t.Errorf("appels = (A:%d, B:%d), want (1,1) — un essai par résolveur", callsA, callsB)
	}
	// Échec → pas de mise en cache : un 2e appel re-tente le réseau (2 essais de plus).
	if _, err := c.ResolveXUID(context.Background(), "Smith"); err == nil {
		t.Fatal("attendu une erreur au 2e appel aussi (rien mis en cache)")
	}
	if callsA != 2 || callsB != 2 {
		t.Errorf("appels après 2e essai = (A:%d, B:%d), want (2,2) — pas de cache sur échec", callsA, callsB)
	}
	if hits, misses := c.Stats(); hits != 0 || misses != 2 {
		t.Errorf("Stats = (%d,%d), want (0,2)", hits, misses)
	}
}

// TestNewCachingResolver_SeedNormalizationAndEmptyFiltering : le seed est
// normalisé (clé lower+trim) et les valeurs xuid vides/espaces sont ignorées —
// une association vide ne doit pas masquer une vraie résolution ultérieure.
func TestNewCachingResolver_SeedNormalizationAndEmptyFiltering(t *testing.T) {
	calls := 0
	r := peopleHubFor(http.StatusOK, `{"people":[{"gamertag":"Cypher","xuid":"555"}]}`, &calls)
	seed := map[string]string{
		"  Neo  ":  "111",   // gamertag avec espaces → clé "neo"
		"TRINITY":  "  ",    // xuid blanc → ignoré (pas mis en cache)
		"Morpheus": "",      // xuid vide → ignoré
		"Cypher":   "  9  ", // wins-by-seed mais sera ignoré ? non: trim → "9" reste
	}
	c := NewCachingResolver([]*auth.PeopleHubResolver{r}, seed, nil)

	// "Neo" (espaces dans le seed) résolu via cache, casse/espaces normalisés.
	if got, err := c.ResolveXUID(context.Background(), "NEO"); err != nil || got != "111" {
		t.Fatalf("Neo: got=%q err=%v, want 111", got, err)
	}
	// "Trinity" : xuid blanc dans le seed → NON caché → doit partir en résolution
	// réseau (ici PeopleHub ne renvoie pas Trinity → erreur, mais le point est
	// qu'on a TENTÉ le réseau, prouvant que le seed vide a bien été filtré).
	if _, err := c.ResolveXUID(context.Background(), "Trinity"); err == nil {
		t.Fatal("Trinity: attendu une erreur (seed vide filtré → résolution réseau qui échoue)")
	}
	if calls != 1 {
		t.Errorf("appels HTTP = %d, want 1 (seul Trinity sort en réseau)", calls)
	}
	// Cypher : trim laisse "9" → cache hit, pas de réseau.
	if got, err := c.ResolveXUID(context.Background(), "cypher"); err != nil || got != "9" {
		t.Fatalf("Cypher: got=%q err=%v, want 9 (seed trimé)", got, err)
	}
	if calls != 1 {
		t.Errorf("appels HTTP après Cypher = %d, want toujours 1 (cache hit)", calls)
	}
}
