//go:build cgo

package ops

// media_thumbnails_ensure_cgo_test.go — EnsureVideoThumbnails : une miniature par vidéo,
// jamais partagée entre deux fiches, régénérée quand elle manque, retrouvée quand elle
// existe déjà dans le dossier.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

type ensureSetup struct {
	db     *sql.DB
	base   string
	capDir string
	store  MediaPathStore
	opts   MediaIndexOptions
}

func newEnsureSetup(t *testing.T) *ensureSetup {
	t.Helper()
	_, db := openDiagDB(t)
	if err := ensureMediaTables(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(reconcileLikesDDL); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	capDir := filepath.Join(base, indexPlayer)
	if err := os.MkdirAll(filepath.Join(capDir, "thumbs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &ensureSetup{
		db: db, base: base, capDir: capDir, store: MediaPathStore{CapturesBase: base},
		opts: MediaIndexOptions{CapturesDir: capDir, CapturesBase: base, Gamertag: indexPlayer},
	}
}

// addVideo insère une vidéo visible dont le fichier source est rel (relatif au dossier
// du joueur) ; thumb vide = aucune miniature liée.
func (s *ensureSetup) addVideo(t *testing.T, rel, hash, thumb string) int64 {
	t.Helper()
	name := filepath.Base(rel)
	ext := filepath.Ext(name)
	var th any
	if thumb != "" {
		th = indexPlayer + "/thumbs/" + thumb
	}
	var id int64
	if err := s.db.QueryRow(`INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind,
		file_hash, source_path, thumbnail_path) VALUES (?, ?, ?, ?, ?, 'video', ?, ?, ?) RETURNING id`,
		indexPlayer, indexPlayer+"/"+rel, name, name[:len(name)-len(ext)], ext, hash,
		indexPlayer+"/"+rel, th).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (s *ensureSetup) thumbOf(t *testing.T, id int64) string {
	t.Helper()
	var th sql.NullString
	if err := s.db.QueryRow(`SELECT thumbnail_path FROM media_files WHERE id = ?`, id).Scan(&th); err != nil {
		t.Fatal(err)
	}
	return th.String
}

func (s *ensureSetup) abs(stored string) string { return s.store.ToAbs(stored) }

func (s *ensureSetup) ensure() (int, []string) {
	return EnsureVideoThumbnails(context.Background(), s.db, s.opts, s.store)
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEnsureVideoThumbnails_RenamedThenNewSameName : clip.mp4 renommé clip_old.mp4, puis
// un autre clip.mp4 déposé : la nouvelle vidéo ne reprend pas la miniature de l'ancienne.
func TestEnsureVideoThumbnails_RenamedThenNewSameName(t *testing.T) {
	requireFFmpeg(t)
	s := newEnsureSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "clip_old.mp4"), 440)
	genMP4Tone(t, filepath.Join(s.capDir, "clip.mp4"), 880)
	writeFile(t, filepath.Join(s.capDir, "thumbs", "clip.webp"), "miniature de l'ancienne")
	oldID := s.addVideo(t, "clip_old.mp4", "aaaaaaaaaaaaaaaaaaaa", "clip.webp")
	newID := s.addVideo(t, "clip.mp4", "bbbbbbbbbbbbbbbbbbbb", "")

	n, errs := s.ensure()
	if n != 1 || len(errs) != 0 {
		t.Fatalf("ensure = (%d, %v), want (1, [])", n, errs)
	}
	if got := s.thumbOf(t, oldID); got != indexPlayer+"/thumbs/clip.webp" {
		t.Errorf("miniature de l'ancienne = %q, want inchangée", got)
	}
	got := s.thumbOf(t, newID)
	if got != indexPlayer+"/thumbs/clip_bbbbbbbbbbbb.webp" {
		t.Errorf("miniature de la nouvelle = %q, want clip_bbbbbbbbbbbb.webp", got)
	}
	if !fileExists(s.abs(got)) {
		t.Errorf("miniature %q non produite", got)
	}
	if b, _ := os.ReadFile(filepath.Join(s.capDir, "thumbs", "clip.webp")); string(b) != "miniature de l'ancienne" {
		t.Error("la miniature de l'ancienne a été écrasée")
	}
}

// TestEnsureVideoThumbnails_RootClipTakesItsOwnName : un original à la racine dont le nom
// est libre prend {nom}.webp.
func TestEnsureVideoThumbnails_RootClipTakesItsOwnName(t *testing.T) {
	requireFFmpeg(t)
	s := newEnsureSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "clip.mp4"), 440)
	id := s.addVideo(t, "clip.mp4", "cccccccccccccccccccc", "")

	if n, errs := s.ensure(); n != 1 || len(errs) != 0 {
		t.Fatalf("ensure = (%d, %v), want (1, [])", n, errs)
	}
	if got := s.thumbOf(t, id); got != indexPlayer+"/thumbs/clip.webp" || !fileExists(s.abs(got)) {
		t.Errorf("miniature = %q, want spartan/thumbs/clip.webp produite", got)
	}
}

// TestEnsureVideoThumbnails_ManuallyDeletedIsRegenerated : miniature liée mais supprimée
// à la main, original présent : regénérée au même chemin.
func TestEnsureVideoThumbnails_ManuallyDeletedIsRegenerated(t *testing.T) {
	requireFFmpeg(t)
	s := newEnsureSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "clip.mp4"), 440)
	id := s.addVideo(t, "clip.mp4", "dddddddddddddddddddd", "clip.webp")
	target := filepath.Join(s.capDir, "thumbs", "clip.webp")

	if n, errs := s.ensure(); n != 1 || len(errs) != 0 {
		t.Fatalf("ensure = (%d, %v), want (1, [])", n, errs)
	}
	if !fileExists(target) || s.thumbOf(t, id) != indexPlayer+"/thumbs/clip.webp" {
		t.Errorf("miniature %q, lien %q, want regénérée au même chemin", target, s.thumbOf(t, id))
	}
	if n, _ := s.ensure(); n != 0 {
		t.Errorf("seconde passe : %d miniatures, want 0 (idempotent)", n)
	}
}

// TestEnsureVideoThumbnails_LinksHistoricThumbnail : une miniature déjà dans thumbs/, à
// suffixe d'empreinte, est liée sans nouvelle génération (aucun ffmpeg requis).
func TestEnsureVideoThumbnails_LinksHistoricThumbnail(t *testing.T) {
	s := newEnsureSetup(t)
	writeFile(t, filepath.Join(s.capDir, "clip.mp4"), "pas une vraie vidéo")
	writeFile(t, filepath.Join(s.capDir, "thumbs", "clip_9430c6551833.webp"), "historique")
	id := s.addVideo(t, "clip.mp4", "eeeeeeeeeeeeeeeeeeee", "")

	n, errs := s.ensure()
	if n != 1 || len(errs) != 0 {
		t.Fatalf("ensure = (%d, %v), want (1, [])", n, errs)
	}
	if got := s.thumbOf(t, id); got != indexPlayer+"/thumbs/clip_9430c6551833.webp" {
		t.Errorf("miniature = %q, want la miniature historique", got)
	}
	if b, _ := os.ReadFile(filepath.Join(s.capDir, "thumbs", "clip_9430c6551833.webp")); string(b) != "historique" {
		t.Error("la miniature historique a été réécrite")
	}
}

// TestEnsureVideoThumbnails_LegacyGifIsLinked : un GIF historique au nom exact est lié.
func TestEnsureVideoThumbnails_LegacyGifIsLinked(t *testing.T) {
	s := newEnsureSetup(t)
	writeFile(t, filepath.Join(s.capDir, "clip.mp4"), "x")
	writeFile(t, filepath.Join(s.capDir, "thumbs", "clip.gif"), "gif")
	id := s.addVideo(t, "clip.mp4", "ffffffffffffffffffff", "")

	if n, errs := s.ensure(); n != 1 || len(errs) != 0 {
		t.Fatalf("ensure = (%d, %v), want (1, [])", n, errs)
	}
	if got := s.thumbOf(t, id); got != indexPlayer+"/thumbs/clip.gif" {
		t.Errorf("miniature = %q, want clip.gif", got)
	}
}

// TestEnsureVideoThumbnails_ThumbnailOfAnotherRowIsNotLinked : une miniature que
// référence une autre fiche n'est pas reprise.
func TestEnsureVideoThumbnails_ThumbnailOfAnotherRowIsNotLinked(t *testing.T) {
	s := newEnsureSetup(t)
	writeFile(t, filepath.Join(s.capDir, "clip.mp4"), "x")
	writeFile(t, filepath.Join(s.capDir, "thumbs", "clip.webp"), "d'une autre fiche")
	s.addVideo(t, "ailleurs.mp4", "1111111111111111", "clip.webp")
	id := s.addVideo(t, "clip.mp4", "2222222222222222", "")

	s.ensure() // la génération échoue sur une fausse vidéo ; seul compte le lien
	if got := s.thumbOf(t, id); got != "" {
		t.Errorf("miniature = %q, want aucune (celle du dossier appartient à une autre fiche)", got)
	}
}

// TestEnsureVideoThumbnails_MissingSourceLeftAlone : sans original, rien n'est tenté.
func TestEnsureVideoThumbnails_MissingSourceLeftAlone(t *testing.T) {
	s := newEnsureSetup(t)
	id := s.addVideo(t, "absent.mp4", "3333333333333333", "")
	if n, errs := s.ensure(); n != 0 || len(errs) != 0 {
		t.Errorf("ensure = (%d, %v), want (0, [])", n, errs)
	}
	if got := s.thumbOf(t, id); got != "" {
		t.Errorf("miniature = %q, want aucune", got)
	}
}
