package resourcepack

import (
	"archive/zip"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Chaque test de ce fichier échoue si le contrôle de sécurité qu'il nomme est retiré : un contrôle que rien ne
// teste est un contrôle qu'un refactor retire sans bruit.

// zipPackWithout écrit l'archive du paquet de packDir sans l'entrée skip.
func zipPackWithout(t *testing.T, packDir, skip string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "pack.zip")
	fh, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	zw := zip.NewWriter(fh)
	err = filepath.WalkDir(packDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(packDir, p)
		rel = filepath.ToSlash(rel)
		if rel == skip {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		_, err = w.Write(raw)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestExtractRefuseUneSommeDifferenteATailleEgale : un fichier altéré qui garde SA TAILLE passe le contrôle de
// taille ; seule la somme SHA-256 du manifeste le refuse.
func TestExtractRefuseUneSommeDifferenteATailleEgale(t *testing.T) {
	pack, m := buildPack(t, map[string]string{"static/a": "AAAA", "static/b": "BBBB"})
	if err := os.WriteFile(filepath.Join(pack, "static", "a"), []byte("CCCC"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	_, err := StageFromZip(root, zipPack(t, pack, nil), m.Fingerprint)
	if err == nil || !strings.Contains(err.Error(), "somme") {
		t.Fatalf("contenu de même taille mais autre somme : %v", err)
	}
	if Staged(root, m.Fingerprint) {
		t.Fatal("paquet marqué prêt malgré une somme différente")
	}
	if _, statErr := os.Stat(incomingDir(root, m.Fingerprint)); statErr == nil {
		t.Fatal("le dossier d'extraction d'un fichier refusé doit être retiré")
	}
}

// TestExtractRefuseUnFichierDuManifesteAbsentDeLArchive : une archive tronquée (un fichier annoncé n'y est pas) ne
// donne jamais un paquet « prêt » incomplet.
func TestExtractRefuseUnFichierDuManifesteAbsentDeLArchive(t *testing.T) {
	pack, m := buildPack(t, map[string]string{"static/a": "A", "static/b": "B", "ffmpeg.exe": "FF"})
	root := t.TempDir()
	_, err := StageFromZip(root, zipPackWithout(t, pack, "static/b"), m.Fingerprint)
	if err == nil || !strings.Contains(err.Error(), "absents de l'archive") {
		t.Fatalf("archive sans un fichier du manifeste : %v", err)
	}
	if Staged(root, m.Fingerprint) {
		t.Fatal("paquet incomplet marqué prêt")
	}
}

// TestSignatureRefuseUnCommentaireDeConfianceAltere : le commentaire de confiance est signé (signature globale) ; un
// commentaire changé avec la signature du contenu intacte est refusé.
func TestSignatureRefuseUnCommentaireDeConfianceAltere(t *testing.T) {
	k := newTestKey(t)
	f := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(f, []byte("contenu"), 0o644); err != nil {
		t.Fatal(err)
	}
	sig := k.Sign(t, f)
	pk, err := ParsePublicKey(k.Public())
	if err != nil {
		t.Fatal(err)
	}
	if err := pk.VerifyFile(f, sig); err != nil {
		t.Fatalf("signature valide refusée : %v", err)
	}
	text, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(text)), "\n")
	lines[2] = trustedPrefix + "timestamp:2\tfile:autre"
	altered := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n") + "\n"))
	err = pk.VerifyFile(f, altered)
	if !errors.Is(err, ErrBadSignature) || !strings.Contains(err.Error(), "commentaire de confiance") {
		t.Fatalf("commentaire de confiance altéré : %v", err)
	}
}

// TestBasculeEchoueeDefaitLesUnitesDejaBasculees : un renommage qui échoue au milieu de la bascule laisse l'ancien
// paquet entier, le paquet prêt intact, aucun journal ; la bascule se rejoue ensuite.
func TestBasculeEchoueeDefaitLesUnitesDejaBasculees(t *testing.T) {
	root := installedRoot(t)
	m := stagedNew(t, root)
	before, after := wantTrees(t, root)
	oldFP := ActiveFingerprint(root)
	plan := noFault()
	plan.failIf = func(kind, p string) bool {
		return kind == "rename" && strings.HasPrefix(filepath.Base(filepath.Dir(p)), incomingPre) && filepath.Base(p) == "ffmpeg.exe"
	}
	err := activateWith(t.Context(), plan.ops(), root, m.Fingerprint)
	if err == nil {
		t.Fatal("la bascule dont un renommage échoue devait échouer")
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, before) {
		t.Fatalf("arbre après retour arrière\n%v", got)
	}
	if v := Check(root, oldFP); v.State != StateMatch {
		t.Fatalf("l'ancien paquet doit être entier : %+v", v)
	}
	if _, statErr := os.Stat(journalPath(root)); statErr == nil {
		t.Fatal("un retour arrière complet ne laisse pas de journal")
	}
	if !Staged(root, m.Fingerprint) {
		t.Fatal("le paquet prêt doit rester prêt")
	}
	if err := Activate(t.Context(), root, m.Fingerprint); err != nil {
		t.Fatalf("la bascule rejouée : %v", err)
	}
	if got := readTree(t, root); !reflect.DeepEqual(got, after) {
		t.Fatalf("arbre après la bascule rejouée\n%v", got)
	}
}
