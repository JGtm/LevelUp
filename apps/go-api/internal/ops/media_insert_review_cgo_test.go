//go:build cgo

package ops

// media_insert_review_cgo_test.go — l'insertion face aux fiches introuvables et aux
// fiches supprimées : une fiche introuvable ambiguë n'est jamais reprise, une reprise
// emporte les likes, une résurrection rend une fiche lisible.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

const likeInsertSQL = `INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
	VALUES (?, ?, ?, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`

func likesOn(t *testing.T, db *sql.DB, path string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM media_likes_latest WHERE media_path = ? AND is_liked`, path).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestInsertMediaFile_ConversionTransfersLikes : la fiche reprise par le fichier d'un
// autre format change de file_path ; ses likes la suivent et quittent l'ancien chemin.
func TestInsertMediaFile_ConversionTransfersLikes(t *testing.T) {
	s := newEnsureSetup(t)
	s.insert(t, "a/clip.mp4", "contenu mp4")
	oldPath := indexPlayer + "/a/clip.mp4"
	if _, err := s.db.Exec(likeInsertSQL, oldPath, "bob", "Bob"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.capDir, "a", "clip.mp4")); err != nil {
		t.Fatal(err)
	}
	s.insert(t, "a/clip.webm", "contenu webm")

	newPath := indexPlayer + "/a/clip.webm"
	if n := countStem(t, s, "clip"); n != 1 {
		t.Fatalf("%d fiches, want 1 (conversion)", n)
	}
	if got := likesOn(t, s.db, newPath); got != 1 {
		t.Errorf("likes sur %s = %d, want 1", newPath, got)
	}
	if got := likesOn(t, s.db, oldPath); got != 0 {
		t.Errorf("likes sur %s = %d, want 0", oldPath, got)
	}
}

// TestIndexMedia_AmbiguousCandidatesAfterOriginalGone : l'original a disparu et deux
// fichiers de même nom apparaissent : aucun ne reprend la fiche (D5), le premier indexé
// devient une nouvelle fiche, et la fiche d'origine reste introuvable avec ses likes. Le
// second est écarté par la règle des fichiers de même nom qui coexistent (une fiche
// visible le désigne déjà comme variante), comme avant.
func TestIndexMedia_AmbiguousCandidatesAfterOriginalGone(t *testing.T) {
	s := newIndexSetup(t)
	writeFile(t, filepath.Join(s.capDir, "clip.png"), "contenu original")
	if res := s.index(t); res.NewFiles != 1 {
		t.Fatalf("première passe : NewFiles = %d, want 1 (erreurs %v)", res.NewFiles, res.Errors)
	}
	origPath := indexPlayer + "/clip.png"
	s.write(t, func(db *sql.DB) {
		if _, err := db.Exec(likeInsertSQL, origPath, "bob", "Bob"); err != nil {
			t.Fatal(err)
		}
	})

	if err := os.Remove(filepath.Join(s.capDir, "clip.png")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(s.capDir, "clip.jpg"), "premier candidat")
	writeFile(t, filepath.Join(s.capDir, "clip.jpeg"), "second candidat")
	if res := s.index(t); len(res.Errors) != 0 {
		t.Fatalf("erreurs %v, want aucune", res.Errors)
	}

	rows := s.rows(t)
	if len(rows) != 2 {
		t.Fatalf("%d fiches, want 2 (origine introuvable + une nouvelle) : %+v", len(rows), rows)
	}
	if rows[0].FilePath != origPath || rows[0].Status != "missing" {
		t.Errorf("fiche d'origine = %+v, want file_path %q et status missing", rows[0], origPath)
	}
	if fresh := rows[1]; fresh.Status != "" || fresh.FilePath == origPath ||
		(fresh.FilePath != indexPlayer+"/clip.jpg" && fresh.FilePath != indexPlayer+"/clip.jpeg") {
		t.Errorf("nouvelle fiche = %+v, want visible, sur l'un des deux candidats", fresh)
	}
	s.read(t, func(db *sql.DB) {
		if got := likesOn(t, db, origPath); got != 1 {
			t.Errorf("likes de la fiche d'origine = %d, want 1", got)
		}
		if got := likesOn(t, db, rows[1].FilePath); got != 0 {
			t.Errorf("likes de %s = %d, want 0", rows[1].FilePath, got)
		}
	})
}

// TestIndexMedia_RestoredDeletedConvertedVideoIsReadableAgain : une vidéo convertie
// supprimée dont l'original revient et dont la version convertie a disparu repasse en
// lecture directe en une seule passe, avec ses likes, et le balayage de conversion la
// reprend.
func TestIndexMedia_RestoredDeletedConvertedVideoIsReadableAgain(t *testing.T) {
	s := newIndexSetup(t)
	mkv := filepath.Join(s.capDir, "clip.mkv")
	writeFile(t, mkv, "contenu")
	if res := s.index(t); res.NewFiles != 1 {
		t.Fatalf("première passe : NewFiles = %d, want 1 (erreurs %v)", res.NewFiles, res.Errors)
	}
	origPath := indexPlayer + "/clip.mkv"
	hlsPath := indexPlayer + "/hls/clip/master.m3u8"
	s.write(t, func(db *sql.DB) {
		if _, err := db.Exec(`UPDATE media_files SET file_path = ?, hls_path = ?, transcode_status = 'ready',
			status = 'deleted'`, hlsPath, hlsPath); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(likeInsertSQL, hlsPath, "bob", "Bob"); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.Remove(mkv); err != nil {
		t.Fatal(err)
	}
	writeFile(t, mkv, "contenu")
	s.index(t)

	rows := s.rows(t)
	if len(rows) != 1 {
		t.Fatalf("%d fiches, want 1 : %+v", len(rows), rows)
	}
	r := rows[0]
	if r.Status != "" || r.FilePath != origPath || r.SourcePath != origPath || r.HLSPath != "" {
		t.Errorf("fiche = %+v, want visible, file_path et source_path %q, hls_path NULL", r, origPath)
	}
	var transcode sql.NullString
	s.read(t, func(db *sql.DB) {
		if err := db.QueryRow(`SELECT transcode_status FROM media_files WHERE id = ?`, r.ID).Scan(&transcode); err != nil {
			t.Fatal(err)
		}
		if got := likesOn(t, db, origPath); got != 1 {
			t.Errorf("likes sur %s = %d, want 1", origPath, got)
		}
		if got := likesOn(t, db, hlsPath); got != 0 {
			t.Errorf("likes sur %s = %d, want 0", hlsPath, got)
		}
	})
	if transcode.Valid {
		t.Errorf("transcode_status = %q, want NULL", transcode.String)
	}
	cands, err := selectPendingHLSCandidates(context.Background(), s.socialPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].filePath != origPath {
		t.Errorf("candidats du balayage = %+v, want la fiche restaurée (%q)", cands, origPath)
	}
}

// write exécute fn sur une connexion en écriture, puis vide le journal.
func (s *indexSetup) write(t *testing.T, fn func(db *sql.DB)) {
	t.Helper()
	db, err := sql.Open("duckdb", s.socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	fn(db)
	if _, err := db.Exec("CHECKPOINT"); err != nil {
		t.Fatal(err)
	}
}
