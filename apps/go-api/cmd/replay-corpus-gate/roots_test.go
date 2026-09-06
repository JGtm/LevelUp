package main

import (
	"path/filepath"
	"testing"
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
	got, err := resolveSourceRoot("/explicite/source")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if filepath.ToSlash(got) != "/explicite/source" {
		t.Fatalf("sourceRoot = %q, attendu le flag explicite", got)
	}
}

// TestResolveParcRootAutoDetectionValideeParLaBase — LA MESURE DU 2026-09-06 : sur CE depot,
// le `.git` commun ne pointe PAS vers le vrai parc (topologie ou `LevelUp-go-migration` est
// lui-meme un worktree d'un ancetre `LevelUp` renomme, sans le parc courant). Sans validation,
// l'auto-detection rendrait un chemin qui COMPILE mais fait echouer toute la suite plus loin
// avec un message DuckDB opaque — ce test verrouille le REFUS EXPLICITE, sur un titre dont la
// base n'existe nulle part sous le `.git` commun de ce processus de test.
func TestResolveParcRootAutoDetectionValideeParLaBase(t *testing.T) {
	if _, err := resolveParcRoot("", "titre-sans-base-nulle-part-6f3a1c"); err == nil {
		t.Fatal("une auto-detection sans base partagee pour ce titre doit etre refusee, pas silencieuse")
	}
}

// TestResolveParcRootFlagGagne — le flag explicite ne consulte jamais git ni la base.
func TestResolveParcRootFlagGagne(t *testing.T) {
	got, err := resolveParcRoot("/explicite/parc", "halo_infinite")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if filepath.ToSlash(got) != "/explicite/parc" {
		t.Fatalf("parcRoot = %q, attendu le flag explicite", got)
	}
}
