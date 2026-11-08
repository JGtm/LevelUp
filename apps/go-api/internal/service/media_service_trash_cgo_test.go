//go:build cgo

package service

// media_service_trash_cgo_test.go — DeleteMedia de bout en bout (vraie shared_social, vrais
// fichiers) avec une Corbeille factice : elle déplace le fichier dans un dossier de test au lieu de
// le confier au système.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// corbeilleDeTest déplace le fichier dans bin (réussite) ou échoue.
type corbeilleDeTest struct {
	base, bin string
	echec     error
}

func (c *corbeilleDeTest) TrashMediaFile(_ context.Context, stored string) (bool, error) {
	if c.echec != nil {
		return false, c.echec
	}
	src := filepath.Join(c.base, filepath.FromSlash(stored))
	dst := filepath.Join(c.bin, filepath.Base(src))
	if err := os.MkdirAll(c.bin, 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

func TestMediaService_DeleteMedia_OriginalALaCorbeille(t *testing.T) {
	f := newDeleteFixture(t)
	bin := t.TempDir()
	f.svc.fileTrasher = &corbeilleDeTest{base: f.capturesBase, bin: bin}

	resp, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest())
	if err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if !resp.Deleted || !resp.Trashed {
		t.Errorf("réponse %+v, attendu Deleted et Trashed", resp)
	}
	if resp.FilesRemoved != 2 {
		t.Errorf("FilesRemoved = %d, attendu 2 (original à la Corbeille + miniature)", resp.FilesRemoved)
	}
	if _, err := os.Stat(filepath.Join(bin, "clip.mp4")); err != nil {
		t.Errorf("l'original n'est pas dans la Corbeille : %v", err)
	}
	for _, p := range []string{f.clipAbs, f.thumbAbs} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("fichier toujours dans le dossier des captures : %s (%v)", p, err)
		}
	}
}

// Échec de la Corbeille : erreur, l'original ET la miniature restent en place.
func TestMediaService_DeleteMedia_EchecDeLaCorbeilleLaisseLOriginal(t *testing.T) {
	f := newDeleteFixture(t)
	f.svc.fileTrasher = &corbeilleDeTest{echec: errors.New("fichier verrouillé")}

	resp, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest())
	if err == nil {
		t.Fatalf("attendu une erreur, réponse %+v", resp)
	}
	for _, p := range []string{f.clipAbs, f.thumbAbs} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("fichier touché malgré l'échec de la Corbeille : %s (%v)", p, err)
		}
	}
}

// Échec de la Corbeille : aucune écriture en base (status inchangé, likes intacts), donc réessayer
// fonctionne : le second essai masque la fiche et envoie l'original à la Corbeille.
func TestMediaService_DeleteMedia_EchecDeLaCorbeillePuisReessai(t *testing.T) {
	f := newDeleteFixture(t)
	status := func() (st sql.NullString) {
		if err := f.db.SQLDb().QueryRow(`SELECT status FROM media_files WHERE file_path = 'JGtm/clip.mp4'`).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	f.svc.fileTrasher = &corbeilleDeTest{echec: errors.New("fichier verrouillé")}
	if _, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest()); err == nil {
		t.Fatal("attendu une erreur de la Corbeille")
	}
	if st := status(); st.Valid {
		t.Fatalf("status = %q après l'échec de la Corbeille, want NULL (aucune écriture)", st.String)
	}

	bin := t.TempDir()
	f.svc.fileTrasher = &corbeilleDeTest{base: f.capturesBase, bin: bin}
	resp, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest())
	if err != nil {
		t.Fatalf("second essai : %v", err)
	}
	if !resp.Deleted || !resp.Trashed {
		t.Errorf("réponse %+v, attendu Deleted et Trashed", resp)
	}
	if st := status(); st.String != "deleted" {
		t.Errorf("status = %q après le second essai, want deleted", st.String)
	}
	if _, err := os.Stat(filepath.Join(bin, "clip.mp4")); err != nil {
		t.Errorf("l'original n'est pas dans la Corbeille : %v", err)
	}
}

// Sans Corbeille configurée : comportement d'avant, tout est effacé définitivement, Trashed faux.
func TestMediaService_DeleteMedia_SansCorbeilleInchange(t *testing.T) {
	f := newDeleteFixture(t)
	resp, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest())
	if err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if resp.Trashed || resp.FilesRemoved != 2 {
		t.Errorf("réponse %+v, attendu Trashed=false et 2 fichiers", resp)
	}
}
