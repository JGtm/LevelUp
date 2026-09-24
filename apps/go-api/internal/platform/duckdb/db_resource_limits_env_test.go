// db_resource_limits_env_test.go — C.4 (campagne perf, 2026-09-23) : les bornes J2
// (memory_limit / threads) se lisent À L'OUVERTURE de chaque connexion, plus à l'init
// du paquet. Lues à l'init, elles précédaient config.BootstrapEnvLocal() : posées dans
// .env.local, elles étaient ignorées en silence. Le test pose les variables APRÈS
// l'init (t.Setenv) et rougit si l'on revient à des variables de paquet. Sans tag de
// build : il tourne dans la suite par défaut, comme pool_stats_test.go.
package duckdb

import (
	"context"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestResourceLimits_EnvSetAfterPackageInitIsHonored(t *testing.T) {
	t.Setenv("LEVELUP_DUCKDB_THREADS", "3")
	t.Setenv("LEVELUP_DUCKDB_MEMORY_LIMIT", "300MB")

	// (a) Observabilité : BudgetsSnapshot relit l'environnement à chaque appel.
	snap := BudgetsSnapshot()
	if got, ok := snap["threads"].(int); !ok || got != 3 {
		t.Errorf("BudgetsSnapshot threads = %v, attendu 3", snap["threads"])
	}
	if got := snap["memory_limit"]; got != "300MB" {
		t.Errorf("BudgetsSnapshot memory_limit = %v, attendu 300MB", got)
	}

	// (b) Connexion ouverte par le chemin normal du paquet. Fichier unique sous
	// t.TempDir → clé de cache neuve : la connexion naît APRÈS t.Setenv (un handle
	// ":memory:" resté en cache garderait les réglages de sa création).
	db, err := OpenReadWrite(filepath.Join(t.TempDir(), "c4_limits.duckdb"))
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	var threads string
	if err := db.QueryRow(ctx, "SELECT current_setting('threads')").Scan(&threads); err != nil {
		t.Fatalf("SELECT threads: %v", err)
	}
	if threads != "3" {
		t.Errorf("threads de la connexion = %q, attendu 3", threads)
	}

	// DuckDB renormalise l'affichage (300MB = 300e6 o → « 286.1 MiB ») : on compare
	// des octets à 1 % près (le défaut 512MB en est à 70 %, il ne peut pas passer).
	var mem string
	if err := db.QueryRow(ctx, "SELECT current_setting('memory_limit')").Scan(&mem); err != nil {
		t.Fatalf("SELECT memory_limit: %v", err)
	}
	t.Logf("DuckDB affiche memory_limit=%q et threads=%s pour 300MB / 3", mem, threads)
	if octets := octetsAffichesDuckDB(t, mem); math.Abs(octets-300e6) > 0.01*300e6 {
		t.Errorf("memory_limit de la connexion = %q (%.0f o), attendu 300MB à 1 %% près", mem, octets)
	}
}

// octetsAffichesDuckDB convertit une taille affichée par DuckDB (« 286.1 MiB »,
// « 300.0 MB », « 512 bytes ») en octets ; une unité inconnue fait échouer le test.
func octetsAffichesDuckDB(t *testing.T, s string) float64 {
	t.Helper()
	unites := map[string]float64{
		"bytes": 1, "kib": 1 << 10, "mib": 1 << 20, "gib": 1 << 30, "tib": 1 << 40,
		"kb": 1e3, "mb": 1e6, "gb": 1e9, "tb": 1e12,
	}
	fin := strings.IndexFunc(s, func(r rune) bool { return (r < '0' || r > '9') && r != '.' })
	if fin <= 0 {
		t.Fatalf("taille DuckDB illisible : %q", s)
	}
	n, err := strconv.ParseFloat(s[:fin], 64)
	mult, ok := unites[strings.ToLower(strings.TrimSpace(s[fin:]))]
	if err != nil || !ok {
		t.Fatalf("taille DuckDB illisible : %q (err=%v, unité connue=%v)", s, err, ok)
	}
	return n * mult
}
