package service

import (
	"context"
	"errors"
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
