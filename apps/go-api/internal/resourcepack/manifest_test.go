package resourcepack

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEtEmpreinteStables(t *testing.T) {
	files := map[string]string{"static/a.png": "A", "web/titles/x/b.png": "BB", "ffmpeg.exe": "FF"}
	dir1, m1 := buildPack(t, files)
	_, m2 := buildPack(t, files)
	if m1.Fingerprint != m2.Fingerprint || len(m1.Fingerprint) != 64 {
		t.Fatalf("empreintes %q et %q", m1.Fingerprint, m2.Fingerprint)
	}
	for i := 1; i < len(m1.Files); i++ {
		if m1.Files[i-1].Path >= m1.Files[i].Path {
			t.Fatalf("fichiers non triés : %v", m1.Files)
		}
	}
	if Fingerprint([]File{m1.Files[2], m1.Files[0], m1.Files[1]}) != m1.Fingerprint {
		t.Fatal("l'empreinte dépend de l'ordre de la liste")
	}
	// Le manifeste écrit ne change pas l'empreinte au second calcul.
	again, err := Build(dir1, "autre version documentaire")
	if err != nil || again.Fingerprint != m1.Fingerprint {
		t.Fatalf("la version documentaire ne doit pas compter : %v %q", err, again.Fingerprint)
	}
	files["static/a.png"] = "A2"
	if _, m3 := buildPack(t, files); m3.Fingerprint == m1.Fingerprint {
		t.Fatal("un fichier modifié doit changer l'empreinte")
	}
}

func TestManifestAllerRetourEtValidation(t *testing.T) {
	dir, m := buildPack(t, map[string]string{"static/a": "A"})
	got, err := ReadManifest(dir)
	if err != nil || got.Fingerprint != m.Fingerprint {
		t.Fatalf("relecture : %v %+v", err, got)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, ManifestFileName))
	tampered := strings.Replace(string(raw), `"size": 1`, `"size": 2`, 1)
	if _, err := ParseManifest([]byte(tampered)); err == nil {
		t.Fatal("un manifeste altéré doit être refusé (empreinte recalculée)")
	}
	bad := m
	bad.Files = []File{{Path: "../x", Size: 1, SHA256: strings.Repeat("a", 64)}}
	bad.Fingerprint = Fingerprint(bad.Files)
	if err := bad.Validate(); err == nil {
		t.Fatal("un chemin qui sort de la racine doit être refusé")
	}
	if _, err := ReadManifest(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("racine sans manifeste : %v", err)
	}
}

func TestSafeRelPath(t *testing.T) {
	for _, ok := range []string{"a", "static/a.png", "data/titles/halo_5/reference/x.json"} {
		if !SafeRelPath(ok) {
			t.Errorf("%q devrait être sûr", ok)
		}
	}
	for _, bad := range []string{"", "/a", "../a", "a/../b", "a//b", "a\\b", "C:/x", "a/./b", "a:b"} {
		if SafeRelPath(bad) {
			t.Errorf("%q devrait être refusé", bad)
		}
	}
}

func TestCheckLesCinqVerdicts(t *testing.T) {
	root, m := buildPack(t, map[string]string{"static/a": "A"})
	if v := Check(root, ""); v.State != StateUnchecked {
		t.Fatalf("sans empreinte attendue : %v", v.State)
	}
	if v := Check(root, m.Fingerprint); v.State != StateMatch || v.Active != m.Fingerprint {
		t.Fatalf("paquet attendu : %+v", v)
	}
	if v := Check(root, strings.Repeat("0", 64)); v.State != StateMismatch || v.Active != m.Fingerprint {
		t.Fatalf("autre paquet : %+v", v)
	}
	if v := Check(t.TempDir(), m.Fingerprint); v.State != StateAbsent {
		t.Fatalf("racine vide : %+v", v)
	}
	if err := os.WriteFile(filepath.Join(root, ManifestFileName), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := Check(root, m.Fingerprint); v.State != StateInvalid || v.Err == nil {
		t.Fatalf("manifeste cassé : %+v", v)
	}
}
