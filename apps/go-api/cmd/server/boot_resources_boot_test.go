package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/blake2b"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/resourcepack"
)

// Ces tests jouent bootResourcePack tel que le démarrage l'appelle : le gestionnaire est créé AVANT l'activation du
// paquet prêt, puis Settle, puis (paquet absent) le téléchargement et un second Settle. L'état que Paramètres lit
// ensuite doit dire le paquet réellement en place.

// bootPackKey : une paire de clés minisign de test, au format de `tauri signer`.
type bootPackKey struct {
	priv  ed25519.PrivateKey
	keyID [8]byte
}

func newBootPackKey(t *testing.T) bootPackKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k := bootPackKey{priv: priv}
	if _, err := rand.Read(k.keyID[:]); err != nil {
		t.Fatal(err)
	}
	return k
}

func (k bootPackKey) public() string {
	bin := append(append([]byte("ED"), k.keyID[:]...), k.priv.Public().(ed25519.PublicKey)...)
	file := "untrusted comment: minisign public key: TEST\n" + base64.StdEncoding.EncodeToString(bin) + "\n"
	return base64.StdEncoding.EncodeToString([]byte(file))
}

func (k bootPackKey) sign(t *testing.T, path string) string {
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

// buildBootPack range files sous un dossier neuf, y écrit le manifeste et rend (dossier, manifeste).
func buildBootPack(t *testing.T, files map[string]string) (string, resourcepack.Manifest) {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := resourcepack.Build(dir, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := resourcepack.WriteManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	return dir, m
}

// zipBootPack écrit l'archive du paquet de dir et rend son chemin.
func zipBootPack(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "pack.zip")
	if err := resourcepack.WriteZip(dir, out); err != nil {
		t.Fatal(err)
	}
	return out
}

// rewriteDoer envoie chaque requête du paquet au serveur de test, quelle que soit l'adresse de publication.
type rewriteDoer struct{ srv *httptest.Server }

func (d rewriteDoer) Do(req *http.Request) (*http.Response, error) {
	target := d.srv.URL + "/pack.zip"
	if strings.HasSuffix(req.URL.Path, ".sig") {
		target += ".sig"
	}
	out, err := http.NewRequestWithContext(req.Context(), req.Method, target, nil)
	if err != nil {
		return nil, err
	}
	out.Header = req.Header.Clone()
	return http.DefaultClient.Do(out)
}

// bootStateFor : l'état de démarrage du profil desktop sur la racine de ressources root.
func bootStateFor(t *testing.T, root, pubKey, fingerprint string) (*bootState, *bytes.Buffer) {
	t.Helper()
	prev := resourcesFingerprint
	resourcesFingerprint = fingerprint
	t.Cleanup(func() { resourcesFingerprint = prev })
	var out bytes.Buffer
	cfg := &config.AppConfig{RepoRoot: root, AppVersion: "9.9.9"}
	cfg.UpdatePublicKey = pubKey
	cfg.Capabilities = config.DeploymentCapabilities{UpdateApp: true}
	return &bootState{ctx: t.Context(), cfg: cfg, events: newDesktopEvents(&out)}, &out
}

func lastResourcesEvent(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	var last map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err == nil && ev["event"] == "resources" {
			last = ev
		}
	}
	if last == nil {
		t.Fatalf("aucun événement resources : %q", out.String())
	}
	return last
}

// TestBootPaquetPretActiveDitLePaquetActif : un paquet téléchargé pendant l'exécution précédente est activé au
// démarrage ; l'état lu ensuite dit le paquet du moteur actif, sans « paquet différent » ni redémarrage demandé.
func TestBootPaquetPretActiveDitLePaquetActif(t *testing.T) {
	oldDir, oldM := buildBootPack(t, map[string]string{"static/old.png": "OLD", "ffmpeg.exe": "OLD-FF"})
	root := t.TempDir()
	if err := copyTree(oldDir, root); err != nil {
		t.Fatal(err)
	}
	newDir, newM := buildBootPack(t, map[string]string{"static/new.png": "NEW", "ffmpeg.exe": "NEW-FF"})
	if _, err := resourcepack.StageFromZip(root, zipBootPack(t, newDir), newM.Fingerprint); err != nil {
		t.Fatal(err)
	}

	st, out := bootStateFor(t, root, newBootPackKey(t).public(), newM.Fingerprint)
	bootResourcePack(st)

	if v := resourcepack.Check(root, newM.Fingerprint); v.State != resourcepack.StateMatch {
		t.Fatalf("le paquet prêt devait être activé : %+v (ancien %s)", v, oldM.Fingerprint)
	}
	got := st.resources.Status()
	if got.Active != newM.Fingerprint || got.RestartNeeded || got.Expected != newM.Fingerprint {
		t.Fatalf("état après démarrage : %+v", got)
	}
	_ = out
}

// TestBootPaquetAbsentTelechargeEtDitLePaquetActif : sans paquet, le démarrage le télécharge, l'active, et l'état dit
// le paquet en place (pas « actif : aucun » avec un redémarrage demandé).
func TestBootPaquetAbsentTelechargeEtDitLePaquetActif(t *testing.T) {
	key := newBootPackKey(t)
	packDir, pm := buildBootPack(t, map[string]string{"static/a.png": "A", "ffmpeg.exe": "FF"})
	zipPath := zipBootPack(t, packDir)
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sig := key.sign(t, zipPath)
	mux := http.NewServeMux()
	mux.HandleFunc("/pack.zip", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(raw) })
	mux.HandleFunc("/pack.zip.sig", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(sig)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	prev := newResourcesDownloader
	newResourcesDownloader = func() resourcepack.Downloader { return resourcepack.Downloader{Doer: rewriteDoer{srv}} }
	t.Cleanup(func() { newResourcesDownloader = prev })

	root := t.TempDir()
	st, out := bootStateFor(t, root, key.public(), pm.Fingerprint)
	bootResourcePack(st)

	if v := resourcepack.Check(root, pm.Fingerprint); v.State != resourcepack.StateMatch {
		t.Fatalf("le paquet téléchargé devait être activé : %+v", v)
	}
	got := st.resources.Status()
	if got.Active != pm.Fingerprint || got.RestartNeeded || got.Phase != resourcepack.PhaseCurrent {
		t.Fatalf("état après démarrage : %+v", got)
	}
	ev := lastResourcesEvent(t, out)
	if ev["phase"] != "current" || ev["active"] != pm.Fingerprint || ev["restart_needed"] != false {
		t.Fatalf("dernier événement vers la coquille : %v", ev)
	}
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
}
