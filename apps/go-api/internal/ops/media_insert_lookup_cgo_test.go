//go:build cgo

package ops

// media_insert_lookup_cgo_test.go — la fiche de même nom qu'insertMediaFile cherche :
// même dossier, même nom sans extension ; une fiche qui désigne déjà le fichier n'est pas
// un changement de format.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func countStem(t *testing.T, s *ensureSetup, stem string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM media_files WHERE file_stem = ?`, stem).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (s *ensureSetup) insert(t *testing.T, rel, hash string) {
	t.Helper()
	abs := filepath.Join(s.capDir, filepath.FromSlash(rel))
	writeFile(t, abs, hash)
	if err := insertMediaFile(context.Background(), s.db, abs, hash, indexPlayer, nil, nil, s.store); err != nil {
		t.Fatalf("insertMediaFile(%s): %v", rel, err)
	}
}

// TestInsertMediaFile_SameNameInOtherFolderIsAnotherMedia : deux dossiers, un même nom de
// fichier : deux médias, chacun avec son original.
func TestInsertMediaFile_SameNameInOtherFolderIsAnotherMedia(t *testing.T) {
	s := newEnsureSetup(t)
	s.insert(t, "a/clip.mp4", "contenu a")
	s.insert(t, "b/clip.mp4", "contenu b")

	if n := countStem(t, s, "clip"); n != 2 {
		t.Fatalf("%d lignes de nom clip, want 2", n)
	}
	var srcs int
	if err := s.db.QueryRow(`SELECT count(DISTINCT source_path) FROM media_files WHERE source_path IS NOT NULL`).Scan(&srcs); err != nil || srcs != 2 {
		t.Errorf("originaux distincts = %d (err %v), want 2", srcs, err)
	}
}

// TestInsertMediaFile_SameFolderConversionStillMatches : changement de format dans le même
// dossier, ancien fichier parti : la fiche est reprise (comportement conservé), et son
// original suit le nouveau fichier.
func TestInsertMediaFile_SameFolderConversionStillMatches(t *testing.T) {
	s := newEnsureSetup(t)
	s.insert(t, "a/clip.mp4", "contenu mp4")
	if err := os.Remove(filepath.Join(s.capDir, "a", "clip.mp4")); err != nil {
		t.Fatal(err)
	}
	s.insert(t, "a/clip.webm", "contenu webm")

	if n := countStem(t, s, "clip"); n != 1 {
		t.Fatalf("%d lignes de nom clip, want 1 (conversion)", n)
	}
	var fp, sp string
	if err := s.db.QueryRow(`SELECT file_path, source_path FROM media_files WHERE file_stem = 'clip'`).Scan(&fp, &sp); err != nil {
		t.Fatal(err)
	}
	if want := indexPlayer + "/a/clip.webm"; fp != want || sp != want {
		t.Errorf("file_path=%q source_path=%q, want %q", fp, sp, want)
	}
}

// TestInsertMediaFile_DeletedRowComesBackAtSamePath : un média supprimé dont le fichier
// revient au même endroit reprend sa fiche.
func TestInsertMediaFile_DeletedRowComesBackAtSamePath(t *testing.T) {
	s := newEnsureSetup(t)
	s.insert(t, "clip.mp4", "contenu")
	if _, err := s.db.Exec(`UPDATE media_files SET status = 'deleted'`); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.capDir, "clip.mp4")); err != nil {
		t.Fatal(err)
	}
	s.insert(t, "clip.mp4", "contenu")

	var n int
	var status *string
	if err := s.db.QueryRow(`SELECT count(*), max(status) FROM media_files`).Scan(&n, &status); err != nil {
		t.Fatal(err)
	}
	if n != 1 || status != nil {
		t.Errorf("%d ligne(s), status %v, want une ligne visible (status NULL)", n, status)
	}
}

// TestInsertMediaFile_DeletedConvertedRowComesBackByItsOriginal : la fiche d'une vidéo
// convertie supprimée, dont l'original revient, est rendue visible par le chemin de son
// original.
func TestInsertMediaFile_DeletedConvertedRowComesBackByItsOriginal(t *testing.T) {
	s := newEnsureSetup(t)
	s.insert(t, "clip.mkv", "contenu")
	if _, err := s.db.Exec(`UPDATE media_files SET file_path = ?, hls_path = ?, status = 'deleted'`,
		indexPlayer+"/hls/clip/master.m3u8", indexPlayer+"/hls/clip/master.m3u8"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.capDir, "clip.mkv")); err != nil {
		t.Fatal(err)
	}
	s.insert(t, "clip.mkv", "contenu")

	var n int
	var status *string
	if err := s.db.QueryRow(`SELECT count(*), max(status) FROM media_files`).Scan(&n, &status); err != nil {
		t.Fatal(err)
	}
	if n != 1 || status != nil {
		t.Errorf("%d ligne(s), status %v, want une ligne visible (status NULL)", n, status)
	}
}
