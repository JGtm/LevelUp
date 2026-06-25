package ops

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadCurrent_VirginDir : un répertoire snapshots sans CURRENT.json → version 0
// (le premier cut produira la v1).
func TestReadCurrent_VirginDir(t *testing.T) {
	dir := t.TempDir()
	v, err := readCurrent(filepath.Join(dir, "CURRENT.json"))
	if err != nil {
		t.Fatalf("readCurrent vierge: %v", err)
	}
	if v != 0 {
		t.Fatalf("version = %d, attendu 0 sur dir vierge", v)
	}
}

// TestFlipCurrent_Atomic : après flip, readCurrent voit la nouvelle version ; aucun
// fichier .tmp résiduel (le rename atomique l'a consommé).
func TestFlipCurrent_Atomic(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "CURRENT.json")

	if err := flipCurrent(cur, 1, "2026-06-25T10:00:00Z"); err != nil {
		t.Fatalf("flip v1: %v", err)
	}
	if v, _ := readCurrent(cur); v != 1 {
		t.Fatalf("après flip v1: version = %d, attendu 1", v)
	}
	if err := flipCurrent(cur, 2, "2026-06-25T10:05:00Z"); err != nil {
		t.Fatalf("flip v2: %v", err)
	}
	if v, _ := readCurrent(cur); v != 2 {
		t.Fatalf("après flip v2: version = %d, attendu 2", v)
	}
	if _, err := os.Stat(cur + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("fichier .tmp résiduel après flip (rename non consommé)")
	}
}

// TestApplyRetention_ProtectsCurrentAndKeepN : garde les `keep` plus récentes + la
// version active, supprime le reste.
func TestApplyRetention_ProtectsCurrentAndKeepN(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []int64{1, 2, 3, 4, 5} {
		if err := os.MkdirAll(filepath.Join(dir, snapshotVersionName(v)), 0o755); err != nil {
			t.Fatalf("mkdir v%d: %v", v, err)
		}
	}
	// keep=2 → garde v4,v5 ; current=1 protégée explicitement → v1 survit ; v2,v3 supprimées.
	removed, err := applyRetention(dir, 2, 1)
	if err != nil {
		t.Fatalf("applyRetention: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("supprimées = %v, attendu [2 3]", removed)
	}
	for _, v := range []int64{1, 4, 5} {
		if _, err := os.Stat(filepath.Join(dir, snapshotVersionName(v))); err != nil {
			t.Errorf("v%d devait survivre: %v", v, err)
		}
	}
	for _, v := range []int64{2, 3} {
		if _, err := os.Stat(filepath.Join(dir, snapshotVersionName(v))); !os.IsNotExist(err) {
			t.Errorf("v%d devait être supprimée", v)
		}
	}
}

// TestApplyRetention_NoopWhenUnderKeep : sous le seuil → rien supprimé.
func TestApplyRetention_NoopWhenUnderKeep(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []int64{1, 2} {
		_ = os.MkdirAll(filepath.Join(dir, snapshotVersionName(v)), 0o755)
	}
	removed, err := applyRetention(dir, 5, 2)
	if err != nil {
		t.Fatalf("applyRetention: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("supprimées = %v, attendu aucune", removed)
	}
}

// TestSHA256File : checksum stable et distinct selon le contenu.
func TestSHA256File(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.bin")
	b := filepath.Join(dir, "b.bin")
	if err := os.WriteFile(a, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, err := sha256File(a)
	if err != nil {
		t.Fatalf("sha a: %v", err)
	}
	ha2, _ := sha256File(a)
	hb, _ := sha256File(b)
	if ha != ha2 {
		t.Errorf("checksum non déterministe: %s != %s", ha, ha2)
	}
	if ha == hb {
		t.Errorf("checksums identiques pour contenus distincts")
	}
	// SHA-256 de "hello" (référence connue).
	const wantHello = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if ha != wantHello {
		t.Errorf("sha256(hello) = %s, attendu %s", ha, wantHello)
	}
}
