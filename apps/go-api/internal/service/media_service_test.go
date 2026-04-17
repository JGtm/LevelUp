package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockMediaRepo struct {
	files    []domain.MediaFileRow
	filesErr error
	count    int
	countErr error
}

func (m *mockMediaRepo) LoadMediaFiles(_ context.Context, _, _ int) ([]domain.MediaFileRow, error) {
	return m.files, m.filesErr
}
func (m *mockMediaRepo) CountMediaFiles(_ context.Context) (int, error) {
	return m.count, m.countErr
}

// --- tests ---

func TestMediaService_GetMediaPage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{
			{FileName: "clip1.mp4", FilePath: "/clips/clip1.mp4", Kind: "video", CaptureEndUTC: &now},
			{FileName: "shot1.png", FilePath: "/shots/shot1.png", Kind: "screenshot"},
		},
		count: 50,
	}
	svc := NewMediaService(repo)

	resp, err := svc.GetMediaPage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(resp.Items))
	}
	if resp.TotalCount != 50 {
		t.Errorf("TotalCount = %d, want 50", resp.TotalCount)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, want 1", resp.Page)
	}
	if !resp.HasMore {
		t.Error("expected HasMore = true")
	}
}

func TestMediaService_GetMediaPage_ZeroPage(t *testing.T) {
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{},
		count: 0,
	}
	svc := NewMediaService(repo)

	resp, err := svc.GetMediaPage(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped from 0)", resp.Page)
	}
}

func TestMediaService_GetMediaPage_NegativePage(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo)

	resp, err := svc.GetMediaPage(context.Background(), -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped from -5)", resp.Page)
	}
}

func TestMediaService_GetMediaPage_FilesError(t *testing.T) {
	repo := &mockMediaRepo{filesErr: errors.New("db fail")}
	svc := NewMediaService(repo)

	_, err := svc.GetMediaPage(context.Background(), 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMediaService_GetMediaPage_CountError_Graceful(t *testing.T) {
	repo := &mockMediaRepo{
		files:    []domain.MediaFileRow{{FileName: "a.mp4", FilePath: "/a.mp4", Kind: "video"}},
		countErr: errors.New("count fail"),
	}
	svc := NewMediaService(repo)

	resp, err := svc.GetMediaPage(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected graceful fallback, got: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 (fallback to len(files))", resp.TotalCount)
	}
}

func TestMediaService_GetMediaPage_NoMore(t *testing.T) {
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{{FileName: "a.mp4", FilePath: "/a.mp4", Kind: "video"}},
		count: 1,
	}
	svc := NewMediaService(repo)

	resp, err := svc.GetMediaPage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HasMore {
		t.Error("expected HasMore = false when all items fit on one page")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UploadMedia
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaService_UploadMedia_NoFiles(t *testing.T) {
	svc := NewMediaService(&mockMediaRepo{})
	result, err := svc.UploadMedia(context.Background(), domain.UploadRequest{
		Files:       nil,
		CapturesDir: t.TempDir(),
		DBPath:      "/nonexistent.duckdb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Saved != 0 {
		t.Errorf("Saved = %d, want 0", result.Saved)
	}
}

func TestMediaService_UploadMedia_SavesToDisk(t *testing.T) {
	dir := t.TempDir()
	// DBPath pointe vers un fichier inexistant → IndexMedia échouera mais les
	// fichiers doivent quand même être écrits sur disque avant l'indexation.
	svc := NewMediaService(&mockMediaRepo{})
	req := domain.UploadRequest{
		Files: []domain.UploadedFile{
			{OriginalName: "clip.mp4", Data: []byte("fake video bytes")},
			{OriginalName: "shot.png", Data: []byte("fake image bytes")},
		},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
		Tolerance:   5,
	}
	result, err := svc.UploadMedia(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Saved != 2 {
		t.Errorf("Saved = %d, want 2", result.Saved)
	}
	// Vérifier que les fichiers existent bien sur disque
	for _, name := range []string{"clip.mp4", "shot.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("file %s not found on disk: %v", name, err)
		}
	}
}

func TestMediaService_UploadMedia_CreatesCapturesDir(t *testing.T) {
	base := t.TempDir()
	captures := filepath.Join(base, "new", "captures")
	svc := NewMediaService(&mockMediaRepo{})
	req := domain.UploadRequest{
		Files:       []domain.UploadedFile{{OriginalName: "a.mp4", Data: []byte("x")}},
		CapturesDir: captures,
		DBPath:      filepath.Join(base, "stats.duckdb"),
	}
	if _, err := svc.UploadMedia(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(captures); err != nil {
		t.Errorf("captures dir not created: %v", err)
	}
}

func TestMediaService_UploadMedia_ToleranceDefault(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{})
	// Tolerance=0 → doit utiliser 5 par défaut (pas de panique)
	req := domain.UploadRequest{
		Files:       []domain.UploadedFile{{OriginalName: "a.mp4", Data: []byte("x")}},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
		Tolerance:   0,
	}
	result, err := svc.UploadMedia(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Saved=1 même si IndexMedia échoue (db inexistante)
	if result.Saved != 1 {
		t.Errorf("Saved = %d, want 1", result.Saved)
	}
}

func TestMediaService_UploadMedia_PathTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{})
	// Un nom de fichier avec path traversal doit être nettoyé par filepath.Base
	req := domain.UploadRequest{
		Files: []domain.UploadedFile{
			{OriginalName: "../../evil.sh", Data: []byte("x")},
		},
		CapturesDir: dir,
		DBPath:      filepath.Join(dir, "stats.duckdb"),
	}
	result, err := svc.UploadMedia(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Saved != 1 {
		t.Fatalf("Saved = %d, want 1", result.Saved)
	}
	// Le fichier doit être dans dir, pas dans un parent
	expected := filepath.Join(dir, "evil.sh")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file at %s, not found: %v", expected, err)
	}
	// Vérifier que rien n'a été écrit hors de dir
	parent := filepath.Dir(dir)
	if _, err := os.Stat(filepath.Join(parent, "evil.sh")); err == nil {
		t.Error("path traversal succeeded — file written outside captures dir!")
	}
}
