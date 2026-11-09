package resourcepack

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// writeTree écrit les fichiers (chemin « / » -> contenu) sous dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// readTree rend les fichiers sous dir (hors dossier de travail et manifeste), chemin « / » -> contenu.
func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel == ManifestFileName || len(rel) >= len(workDirName) && rel[:len(workDirName)] == workDirName {
			return nil
		}
		raw, err := os.ReadFile(p)
		out[rel] = string(raw)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// buildPack range files sous un dossier neuf, y écrit le manifeste et rend (dossier, manifeste).
func buildPack(t *testing.T, files map[string]string) (string, Manifest) {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, files)
	m, err := Build(dir, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	return dir, m
}

// zipPack écrit l'archive d'un paquet (manifeste compris) et rend son chemin.
func zipPack(t *testing.T, packDir string, mutate func(w *zip.Writer)) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "pack.zip")
	fh, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(fh)
	err = filepath.WalkDir(packDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(packDir, p)
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		zw, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		_, err = zw.Write(raw)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(w)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fh.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

// testKey : une paire de clés minisign de test, au format de `tauri signer`.
type testKey struct {
	priv  ed25519.PrivateKey
	keyID [8]byte
}

func newTestKey(t *testing.T) testKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k := testKey{priv: priv}
	if _, err := rand.Read(k.keyID[:]); err != nil {
		t.Fatal(err)
	}
	return k
}

// Public rend la clé publique telle que tauri.conf.json la porte.
func (k testKey) Public() string {
	bin := append(append([]byte("ED"), k.keyID[:]...), k.priv.Public().(ed25519.PublicKey)...)
	file := "untrusted comment: minisign public key: TEST\n" + base64.StdEncoding.EncodeToString(bin) + "\n"
	return base64.StdEncoding.EncodeToString([]byte(file))
}

// Sign signe le fichier path et rend la signature telle que latest.json la porte.
func (k testKey) Sign(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := blake2b.Sum512(raw)
	sig := ed25519.Sign(k.priv, sum[:])
	trusted := "timestamp:1\tfile:test"
	global := ed25519.Sign(k.priv, append(append([]byte(nil), sig...), []byte(trusted)...))
	bin := append(append([]byte("ED"), k.keyID[:]...), sig...)
	file := "untrusted comment: test\n" + base64.StdEncoding.EncodeToString(bin) +
		"\ntrusted comment: " + trusted + "\n" + base64.StdEncoding.EncodeToString(global) + "\n"
	return base64.StdEncoding.EncodeToString([]byte(file))
}
