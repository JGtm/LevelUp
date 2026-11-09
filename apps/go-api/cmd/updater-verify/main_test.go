package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// pair : une paire de cles ed25519 de test et l'identifiant de cle minisign associe.
type pair struct {
	pub   ed25519.PublicKey
	priv  ed25519.PrivateKey
	keyID [keyIDLen]byte
}

func newPair(t *testing.T, id byte) pair {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	p := pair{pub: pub, priv: priv}
	for i := range p.keyID {
		p.keyID[i] = id + byte(i)
	}
	return p
}

func b64(parts ...[]byte) string { return base64.StdEncoding.EncodeToString(bytes.Join(parts, nil)) }

// pubText : le contenu d'un fichier .pub minisign.
func (p pair) pubText() string {
	return "untrusted comment: minisign public key: test\n" + b64([]byte(algRaw), p.keyID[:], p.pub) + "\n"
}

// sigText : le contenu d'un fichier .sig minisign ; la signature globale n'est pas verifiee, un remplissage suffit.
func (p pair) sigText(alg string, content []byte) string {
	message := content
	if alg == algPrehashed {
		sum := blake2b.Sum512(content)
		message = sum[:]
	}
	sig := ed25519.Sign(p.priv, message)
	return "untrusted comment: signature from test\n" + b64([]byte(alg), p.keyID[:], sig) +
		"\ntrusted comment: timestamp:1\nglobal\n"
}

func wrap(text string) string { return base64.StdEncoding.EncodeToString([]byte(text)) }

func TestVerify(t *testing.T) {
	content := []byte(strings.Repeat("installeur ", 100))
	good, other := newPair(t, 1), newPair(t, 100)
	sameID := newPair(t, 1) // meme identifiant de cle, autre paire

	for _, alg := range []string{algRaw, algPrehashed} {
		t.Run(alg, func(t *testing.T) {
			sig := good.sigText(alg, content)
			for name, wrapped := range map[string]bool{"tel quel": false, "encode comme Tauri": true} {
				pub, s := good.pubText(), sig
				if wrapped {
					pub, s = wrap(pub), wrap(s)
				}
				if err := verify(mustUnwrap(t, pub), s, content); err != nil {
					t.Errorf("%s : %v", name, err)
				}
			}
			cases := map[string]struct {
				pub     string
				sig     string
				content []byte
				want    string
			}{
				"cle d'un autre identifiant":    {other.pubText(), sig, content, "TAURI_SIGNING_PRIVATE_KEY"},
				"fichier modifie":               {good.pubText(), sig, append(bytes.Clone(content), '!'), "invalide"},
				"autre paire, meme identifiant": {sameID.pubText(), sig, content, "invalide"},
				"signature d'une autre paire":   {good.pubText(), other.sigText(alg, content), content, "TAURI_SIGNING_PRIVATE_KEY"},
				"texte sans signature":          {good.pubText(), "n'importe quoi", content, "signature"},
			}
			for name, c := range cases {
				err := verify(c.pub, c.sig, c.content)
				if err == nil || !strings.Contains(err.Error(), c.want) {
					t.Errorf("%s : erreur %v, attendu un message contenant %q", name, err, c.want)
				}
			}
		})
	}

	if err := verify(good.pubText(), good.sigText("XX", content), content); err == nil || !strings.Contains(err.Error(), "inconnu") {
		t.Errorf("algorithme inconnu : %v", err)
	}
}

func mustUnwrap(t *testing.T, text string) string {
	t.Helper()
	out, err := unwrap(text)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunCodesDeSortie(t *testing.T) {
	dir := t.TempDir()
	content := []byte("contenu de l'installeur")
	good, other := newPair(t, 1), newPair(t, 100)

	conf, _ := json.Marshal(map[string]any{"plugins": map[string]any{"updater": map[string]any{"pubkey": wrap(good.pubText())}}})
	confPath := writeFile(t, dir, "tauri.conf.json", conf)
	otherConf, _ := json.Marshal(map[string]any{"plugins": map[string]any{"updater": map[string]any{"pubkey": wrap(other.pubText())}}})
	otherConfPath := writeFile(t, dir, "autre.conf.json", otherConf)
	filePath := writeFile(t, dir, "setup.exe", content)
	sigPath := writeFile(t, dir, "setup.exe.sig", []byte(wrap(good.sigText(algPrehashed, content))))

	run1 := func(args ...string) (int, string) {
		var out, errOut bytes.Buffer
		code := run(args, &out, &errOut)
		return code, out.String() + errOut.String()
	}

	if code, msg := run1("-pubkey", confPath, "-sig", sigPath, "-file", filePath); code != 0 {
		t.Errorf("signature valide : code %d (%s)", code, msg)
	}
	if code, msg := run1("-pubkey", wrap(good.pubText()), "-sig", sigPath, "-file", filePath); code != 0 {
		t.Errorf("cle en base64 : code %d (%s)", code, msg)
	}
	if code, msg := run1("-pubkey", otherConfPath, "-sig", sigPath, "-file", filePath); code != 1 || !strings.Contains(msg, "ECHEC") {
		t.Errorf("cle publique d'une autre paire : code %d (%s)", code, msg)
	}
	changed := writeFile(t, dir, "modifie.exe", append(bytes.Clone(content), 0))
	if code, _ := run1("-pubkey", confPath, "-sig", sigPath, "-file", changed); code != 1 {
		t.Errorf("fichier modifie : code %d, attendu 1", code)
	}
	if code, _ := run1("-pubkey", confPath, "-sig", filepath.Join(dir, "absent.sig"), "-file", filePath); code != 1 {
		t.Errorf("signature absente : code %d, attendu 1", code)
	}
	if code, _ := run1("-pubkey", confPath); code != 2 {
		t.Errorf("arguments manquants : code %d, attendu 2", code)
	}
}
