package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// TestResolveLockRootFlagGagne — le flag explicite prime sur tout le reste.
func TestResolveLockRootFlagGagne(t *testing.T) {
	got := resolveLockRoot("/un/chemin/explicite", "/parc/quelconque")
	if filepath.ToSlash(got) != "/un/chemin/explicite" {
		t.Fatalf("lockRoot = %q, attendu le flag explicite", got)
	}
}

// TestResolveLockRootEnvAvantDefaut — la variable d'environnement prime sur le defaut derive
// du parc, mais pas sur un flag explicite (deja couvert ci-dessus).
func TestResolveLockRootEnvAvantDefaut(t *testing.T) {
	t.Setenv("REPLAY_CORPUS_GATE_LOCK_ROOT", "/via/env")
	got := resolveLockRoot("", "/parc/quelconque")
	if filepath.ToSlash(got) != "/via/env" {
		t.Fatalf("lockRoot = %q, attendu la variable d'environnement", got)
	}
}

// TestResolveLockRootDefautSousLeParc — sans flag ni variable, le verrou vit sous
// CacheRootDir() du PARC — le MEME chemin que cmd/replay-build et backfill-replay y posent
// deja le leur (cf. l'en-tete de roots.go) : c'est ce qui rend le verrou partage.
func TestResolveLockRootDefautSousLeParc(t *testing.T) {
	got := resolveLockRoot("", filepath.FromSlash("/un/parc"))
	attendu := filepath.Join("/un/parc", "data", "cache")
	if filepath.Clean(got) != filepath.Clean(attendu) {
		t.Fatalf("lockRoot = %q, attendu %q (CacheRootDir du parc)", got, attendu)
	}
}

// TestResolveSourceRootFlagGagne — meme regle que les autres racines : le flag explicite
// prime sur l'auto-detection.
func TestResolveSourceRootFlagGagne(t *testing.T) {
	got, err := resolveSourceRoot(t.Context(), "/explicite/source")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if filepath.ToSlash(got) != "/explicite/source" {
		t.Fatalf("sourceRoot = %q, attendu le flag explicite", got)
	}
}

// TestResolveSourceRootAutoDetectionParGit — CORPUS-R1 C6 (2026-09-07) : sans flag,
// resolveSourceRoot doit reussir DEPUIS CE WORKTREE MEME (LevelUp-wt-v2-corpus), qui ne porte
// PAS de db_profiles.json local — l'ancienne resolution (title.FindRepoRoot) y echouait. La
// preuve : le resultat doit etre EXACTEMENT celui de `git rev-parse --show-toplevel`, pas un
// chemin devine autrement, et porter apps/go-api.
func TestResolveSourceRootAutoDetectionParGit(t *testing.T) {
	got, err := resolveSourceRoot(t.Context(), "")
	if err != nil {
		t.Fatalf("err = %v (ce test suppose un checkout git ; sinon --source-root est obligatoire)", err)
	}
	out, gitErr := exec.Command("git", "rev-parse", "--path-format=absolute", "--show-toplevel").Output()
	if gitErr != nil {
		t.Skipf("git indisponible pour verifier : %v", gitErr)
	}
	attendu := filepath.Clean(strings.TrimSpace(string(out)))
	if filepath.Clean(got) != attendu {
		t.Fatalf("sourceRoot = %q, attendu %q (git rev-parse --show-toplevel — PAS db_profiles.json, "+
			"qu'un worktree dedie ne porte jamais en copie locale)", got, attendu)
	}
	if _, err := os.Stat(filepath.Join(got, "apps", "go-api")); err != nil {
		t.Errorf("sourceRoot = %q ne porte pas apps/go-api : %v", got, err)
	}
}

// TestResolveParcRootAutoDetectionValideeParLaBase — LA MESURE DU 2026-09-06 : sur CE depot,
// ni la racine source (ce worktree, sans base locale) ni le `.git` commun ne pointent vers le
// vrai parc (topologie ou `LevelUp-go-migration` est lui-meme un worktree d'un ancetre
// `LevelUp` renomme, sans le parc courant). Sans validation, l'auto-detection rendrait un
// chemin qui COMPILE mais fait echouer toute la suite plus loin avec un message DuckDB opaque —
// ce test verrouille le REFUS EXPLICITE, sur un titre dont la base n'existe nulle part.
func TestResolveParcRootAutoDetectionValideeParLaBase(t *testing.T) {
	sansBase := t.TempDir()
	if _, err := resolveParcRoot(t.Context(), "", sansBase, "titre-sans-base-nulle-part-6f3a1c"); err == nil {
		t.Fatal("une auto-detection sans base partagee pour ce titre doit etre refusee, pas silencieuse")
	}
}

// TestResolveParcRootSourceRootPorteLaBase — CORPUS-R1 C6 (2026-09-07) : le cas le plus
// courant, un gate lance depuis le depot de developpement qui EST le parc — `sourceRoot` doit
// etre accepte SANS consulter le `.git` commun (qui, sur ce depot, pointe vers un ancetre
// perime, cf. le test ci-dessus).
func TestResolveParcRootSourceRootPorteLaBase(t *testing.T) {
	tmp := t.TempDir()
	titleSlug := "titre-avec-base-au-sourceroot-a3f9"
	sharedDB := title.NewPathResolver(tmp).SharedDBPath(titleSlug)
	if err := os.MkdirAll(filepath.Dir(sharedDB), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedDB, []byte("stub"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveParcRoot(t.Context(), "", tmp, titleSlug)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(tmp) {
		t.Fatalf("parcRoot = %q, attendu sourceRoot %q (il porte deja la base du titre — pas besoin "+
			"de consulter le .git commun)", got, tmp)
	}
}

// TestResolveParcRootFlagGagne — le flag explicite ne consulte jamais git ni la base.
func TestResolveParcRootFlagGagne(t *testing.T) {
	got, err := resolveParcRoot(t.Context(), "/explicite/parc", "/source/quelconque", "halo_infinite")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if filepath.ToSlash(got) != "/explicite/parc" {
		t.Fatalf("parcRoot = %q, attendu le flag explicite", got)
	}
}
