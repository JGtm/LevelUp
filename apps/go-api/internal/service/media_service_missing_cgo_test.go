//go:build cgo

// media_service_missing_cgo_test.go — fusion et oubli d'un média introuvable sur une vraie
// shared_social DuckDB et de vrais fichiers : durabilité (CHECKPOINT), fichiers retirés (ceux de
// l'application seulement), original de l'utilisateur intact, refus sans effet.
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// markFixtureMissing passe la fiche de la fixture en « introuvable » (son original reste sur le
// disque, pour prouver qu'aucune action ne le touche).
func markFixtureMissing(t *testing.T, f *deleteFixture) {
	t.Helper()
	if _, err := f.db.SQLDb().Exec(`UPDATE media_files SET status = 'missing' WHERE file_path = 'JGtm/clip.mp4'`); err != nil {
		t.Fatalf("marquage introuvable: %v", err)
	}
	if _, err := f.db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
}

// addFixtureTarget ajoute la fiche présente dans laquelle fondre, avec sa miniature sur le disque.
func addFixtureTarget(t *testing.T, f *deleteFixture, thumbnail string) {
	t.Helper()
	if _, err := f.db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, thumbnail_path)
		VALUES ('JGtm', 'JGtm/new.mp4', 'new.mp4', ?)`, thumbnail); err != nil {
		t.Fatalf("seed cible: %v", err)
	}
	abs := filepath.Join(f.capturesBase, filepath.FromSlash(thumbnail))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("thumb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func fixtureStatus(t *testing.T, f *deleteFixture, path string) string {
	t.Helper()
	var status string
	if err := f.db.SQLDb().QueryRow(`SELECT COALESCE(status, '') FROM media_files WHERE file_path = ?`, path).Scan(&status); err != nil {
		t.Fatalf("statut de %s: %v", path, err)
	}
	return status
}

func ownerMerge() domain.MediaMergeRequest {
	return domain.MediaMergeRequest{
		MediaRequester: enforced("JGtm"), FilePath: "JGtm/clip.mp4", TargetFilePath: "JGtm/new.mp4",
	}
}

func assertNoWAL(t *testing.T, f *deleteFixture, what string) {
	t.Helper()
	if st, err := os.Stat(f.dbPath + ".wal"); err == nil && st.Size() > 0 {
		t.Errorf("WAL non flushé (%d octets) après %s : écrire via CommitWithCheckpoint, pas tx.Commit", st.Size(), what)
	}
}

func TestMergeMissingMedia_MovesLikesRemovesAppFilesKeepsOriginalAndTargetFiles(t *testing.T) {
	f := newDeleteFixture(t)
	markFixtureMissing(t, f)
	addFixtureTarget(t, f, "JGtm/thumbs/new.webp")
	newThumb := filepath.Join(f.capturesBase, "JGtm", "thumbs", "new.webp")

	resp, err := f.svc.MergeMissingMedia(context.Background(), ownerMerge())
	if err != nil {
		t.Fatalf("MergeMissingMedia: %v", err)
	}
	if !resp.Merged || resp.FilesRemoved != 1 {
		t.Errorf("resp = %+v, attendu fusion et 1 fichier retiré (la miniature de la fiche fondue)", resp)
	}
	if exists(f.thumbAbs) {
		t.Error("la miniature de la fiche fondue est restée sur le disque")
	}
	if !exists(f.clipAbs) {
		t.Error("l'original a été retiré : une fusion ne touche jamais au fichier de l'utilisateur")
	}
	if !exists(newThumb) {
		t.Error("la miniature de la cible a été retirée")
	}
	if got := fixtureStatus(t, f, "JGtm/clip.mp4"); got != domain.MediaStatusDeleted {
		t.Errorf("statut de la fiche fondue = %q, attendu deleted", got)
	}
	var likes int
	if err := f.db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_latest WHERE media_path = 'JGtm/new.mp4' AND liker_slug = 'Chocoboflor' AND is_liked`,
	).Scan(&likes); err != nil || likes != 1 {
		t.Errorf("like transféré à la cible = %d (err=%v), attendu 1", likes, err)
	}
	assertNoWAL(t, f, "la fusion")
}

// La cible réutilise la miniature de la fiche fondue : elle est conservée.
func TestMergeMissingMedia_KeepsFilesTheTargetUses(t *testing.T) {
	f := newDeleteFixture(t)
	markFixtureMissing(t, f)
	addFixtureTarget(t, f, "JGtm/thumbs/clip.webp")

	if _, err := f.svc.MergeMissingMedia(context.Background(), ownerMerge()); err != nil {
		t.Fatalf("MergeMissingMedia: %v", err)
	}
	if !exists(f.thumbAbs) {
		t.Error("une miniature utilisée par la cible a été retirée")
	}
}

func TestMergeMissingMedia_ForeignRequesterChangesNothing(t *testing.T) {
	f := newDeleteFixture(t)
	markFixtureMissing(t, f)
	addFixtureTarget(t, f, "JGtm/thumbs/new.webp")
	req := ownerMerge()
	req.MediaRequester = enforced("Mallory")

	if _, err := f.svc.MergeMissingMedia(context.Background(), req); apiCode(err) != "forbidden" {
		t.Fatalf("err = %v, attendu forbidden", err)
	}
	if got := fixtureStatus(t, f, "JGtm/clip.mp4"); got != domain.MediaStatusMissing {
		t.Errorf("statut = %q, attendu missing (inchangé)", got)
	}
	if !exists(f.thumbAbs) || !exists(f.clipAbs) {
		t.Error("un fichier a été retiré malgré le refus")
	}
}

func TestForgetMissingMedia_RemovesAppFilesNeverTheOriginal(t *testing.T) {
	f := newDeleteFixture(t)
	markFixtureMissing(t, f)

	resp, err := f.svc.ForgetMissingMedia(context.Background(), domain.MediaForgetRequest{
		MediaRequester: enforced("JGtm"), FilePath: "JGtm/clip.mp4",
	})
	if err != nil {
		t.Fatalf("ForgetMissingMedia: %v", err)
	}
	if !resp.Forgotten || resp.FilesRemoved != 1 {
		t.Errorf("resp = %+v, attendu oubli et 1 fichier retiré (la miniature)", resp)
	}
	if exists(f.thumbAbs) {
		t.Error("la miniature est restée sur le disque")
	}
	if !exists(f.clipAbs) {
		t.Error("l'original a été retiré : oublier ne touche jamais au fichier de l'utilisateur")
	}
	if got := fixtureStatus(t, f, "JGtm/clip.mp4"); got != domain.MediaStatusDeleted {
		t.Errorf("statut = %q, attendu deleted", got)
	}
	assertNoWAL(t, f, "l'oubli")
}

// Une fiche présente n'est pas « oubliable » : 404, rien n'est retiré.
func TestForgetMissingMedia_PresentMediaIsNotFound(t *testing.T) {
	f := newDeleteFixture(t)

	_, err := f.svc.ForgetMissingMedia(context.Background(), domain.MediaForgetRequest{
		MediaRequester: enforced("JGtm"), FilePath: "JGtm/clip.mp4",
	})
	if apiCode(err) != "not_found" {
		t.Fatalf("err = %v, attendu not_found", err)
	}
	if !exists(f.thumbAbs) || !exists(f.clipAbs) || fixtureStatus(t, f, "JGtm/clip.mp4") != "" {
		t.Error("l'oubli d'une fiche présente a eu un effet")
	}
}

func TestForgetMissingMedia_ForeignRequesterChangesNothing(t *testing.T) {
	f := newDeleteFixture(t)
	markFixtureMissing(t, f)

	_, err := f.svc.ForgetMissingMedia(context.Background(), domain.MediaForgetRequest{
		MediaRequester: enforced("Mallory"), FilePath: "JGtm/clip.mp4",
	})
	if apiCode(err) != "forbidden" {
		t.Fatalf("err = %v, attendu forbidden", err)
	}
	if got := fixtureStatus(t, f, "JGtm/clip.mp4"); got != domain.MediaStatusMissing || !exists(f.thumbAbs) {
		t.Errorf("effet malgré le refus : statut=%q miniature présente=%v", got, exists(f.thumbAbs))
	}
}

// La liste réelle ne montre que les fiches introuvables du joueur.
func TestListMissingMedia_RealRepo(t *testing.T) {
	f := newDeleteFixture(t)
	ctx := context.Background()
	who := domain.MediaMissingListRequest{MediaRequester: enforced("JGtm")}

	if resp, err := f.svc.ListMissingMedia(ctx, who); err != nil || len(resp.Items) != 0 {
		t.Fatalf("avant : resp=%+v err=%v, attendu aucune fiche", resp, err)
	}
	markFixtureMissing(t, f)
	resp, err := f.svc.ListMissingMedia(ctx, who)
	if err != nil || len(resp.Items) != 1 || resp.Items[0].FilePath != "JGtm/clip.mp4" {
		t.Fatalf("après : resp=%+v err=%v, attendu la fiche marquée", resp, err)
	}
}
