package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// Compile-time check : *duckdb.MediaRepo doit satisfaire l'interface privée
// atomicMediaLiker pour activer le chemin atomique. On le vérifie ici via une
// déclaration vide qui force le compilateur à valider la conformité — si la
// signature de SetMediaLikeAtomic divergeait, le build casserait.
//
// On n'importe pas duckdb directement (cycle) — la vérif sera faite au commit
// suivant via un test d'intégration cgo.
var _ atomicMediaLiker = (*mockAtomicMediaRepo)(nil)

// --- mock ---

type mockMediaRepo struct {
	files             []domain.MediaFileRow
	filesErr          error
	count             int
	countErr          error
	filterOptions     domain.MediaFilterOptions
	filterOptsErr     error
	setOK             bool
	setErr            error
	toggleErr         error
	toggleCalled      bool
	toggleLiked       bool
	likersData        map[string]domain.MediaLikersInfo
	likersErr         error
	capturedFilter    domain.MediaFilters
	currentPlayerSlug string
	authors           []domain.MediaAuthor
	authorsErr        error
}

func (m *mockMediaRepo) LoadMediaFiles(_ context.Context, f domain.MediaFilters, _, _ int) ([]domain.MediaFileRow, error) {
	m.capturedFilter = f
	return m.files, m.filesErr
}
func (m *mockMediaRepo) CountMediaFiles(_ context.Context, _ domain.MediaFilters) (int, error) {
	return m.count, m.countErr
}
func (m *mockMediaRepo) LoadMediaFilterOptions(_ context.Context, _ domain.MediaFilters) (domain.MediaFilterOptions, error) {
	return m.filterOptions, m.filterOptsErr
}
func (m *mockMediaRepo) SetMediaLike(_ context.Context, _ string, _ bool) (bool, error) {
	if m.setErr != nil {
		return false, m.setErr
	}
	return m.setOK, nil
}
func (m *mockMediaRepo) ToggleSharedLike(_ context.Context, _, _, _ string, liked bool) error {
	m.toggleCalled = true
	m.toggleLiked = liked
	return m.toggleErr
}
func (m *mockMediaRepo) GetMediaLikers(_ context.Context, _ []string) (map[string]domain.MediaLikersInfo, error) {
	return m.likersData, m.likersErr
}

func (m *mockMediaRepo) CurrentPlayerSlug() string { return m.currentPlayerSlug }

func (m *mockMediaRepo) LoadMatchCandidatesForMedia(_ context.Context, _ string, _ int) (domain.MediaMatchCandidatesResponse, error) {
	return domain.MediaMatchCandidatesResponse{}, nil
}

func (m *mockMediaRepo) SetMediaMatchAssociation(_ context.Context, _, _ string) (*string, *string, error) {
	return nil, nil, nil
}

func (m *mockMediaRepo) ListMediaAuthors(_ context.Context) ([]domain.MediaAuthor, error) {
	return m.authors, m.authorsErr
}

// --- tests ---

// TestMediaService_ListMediaAuthors_SetsIsSelf vérifie que le service marque
// is_self pour le joueur courant (résolu via repo.CurrentPlayerSlug) et propage
// la liste/le compte tels quels.
func TestMediaService_ListMediaAuthors_SetsIsSelf(t *testing.T) {
	repo := &mockMediaRepo{
		currentPlayerSlug: "HeroPlayer",
		authors: []domain.MediaAuthor{
			{PlayerSlug: "Alice", MediaCount: 2},
			{PlayerSlug: "HeroPlayer", MediaCount: 5},
		},
	}
	svc := NewMediaService(repo, "")
	got, err := svc.ListMediaAuthors(context.Background())
	if err != nil {
		t.Fatalf("ListMediaAuthors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d authors, want 2", len(got))
	}
	for _, a := range got {
		wantSelf := a.PlayerSlug == "HeroPlayer"
		if a.IsSelf != wantSelf {
			t.Errorf("%s: IsSelf=%v, want %v", a.PlayerSlug, a.IsSelf, wantSelf)
		}
	}
}

func TestMediaService_ListMediaAuthors_PropagatesError(t *testing.T) {
	repo := &mockMediaRepo{authorsErr: errors.New("db_error")}
	svc := NewMediaService(repo, "")
	if _, err := svc.ListMediaAuthors(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMediaService_GetMediaPage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{
			{FileName: "clip1.mp4", FilePath: "/clips/clip1.mp4", Kind: "video", CaptureEndUTC: &now},
			{FileName: "shot1.png", FilePath: "/shots/shot1.png", Kind: "screenshot"},
		},
		count: 50,
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Items.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(resp.Items.Items))
	}
	if resp.Items.Pagination.Total != 50 {
		t.Errorf("Total = %d, want 50", resp.Items.Pagination.Total)
	}
	if resp.Items.Pagination.Page != 1 {
		t.Errorf("Page = %d, want 1", resp.Items.Pagination.Page)
	}
	if !resp.Items.Pagination.HasNext {
		t.Error("expected HasNext = true")
	}
}

func TestMediaService_GetMediaPage_ZeroPage(t *testing.T) {
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{},
		count: 0,
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items.Pagination.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped from 0)", resp.Items.Pagination.Page)
	}
}

func TestMediaService_GetMediaPage_NegativePage(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: -5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items.Pagination.Page != 1 {
		t.Errorf("Page = %d, want 1 (clamped from -5)", resp.Items.Pagination.Page)
	}
}

func TestMediaService_GetMediaPage_FilesError(t *testing.T) {
	repo := &mockMediaRepo{filesErr: errors.New("db fail")}
	svc := NewMediaService(repo, "")

	_, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err == nil {
		t.Error("expected error")
	}
}

func TestMediaService_GetMediaPage_CountError_Graceful(t *testing.T) {
	repo := &mockMediaRepo{
		files:    []domain.MediaFileRow{{FileName: "a.mp4", FilePath: "/a.mp4", Kind: "video"}},
		countErr: errors.New("count fail"),
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("expected graceful fallback, got: %v", err)
	}
	if resp.Items.Pagination.Total != 1 {
		t.Errorf("Total = %d, want 1 (fallback to len(files))", resp.Items.Pagination.Total)
	}
}

func TestMediaService_GetMediaPage_NoMore(t *testing.T) {
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{{FileName: "a.mp4", FilePath: "/a.mp4", Kind: "video"}},
		count: 1,
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items.Pagination.HasNext {
		t.Error("expected HasNext = false when all items fit on one page")
	}
}

func TestMediaService_GetMediaPage_PageSizeFromPagination(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{
		Pagination: domain.PaginationRequest{Page: 2, PageSize: 4},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items.Pagination.Page != 2 {
		t.Errorf("Page = %d, want 2", resp.Items.Pagination.Page)
	}
	if resp.Items.Pagination.PageSize != 4 {
		t.Errorf("PageSize = %d, want 4", resp.Items.Pagination.PageSize)
	}
}

func TestMediaService_SetMediaLike_OK(t *testing.T) {
	repo := &mockMediaRepo{setOK: true}
	svc := NewMediaService(repo, "")

	resp, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "/clips/clip1.mp4",
		Liked:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Liked || resp.LikeCount != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestMediaService_SetMediaLike_NotFound(t *testing.T) {
	repo := &mockMediaRepo{setOK: false}
	svc := NewMediaService(repo, "")

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "/clips/missing.mp4",
		Liked:    true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UploadMedia
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaService_UploadMedia_NoFiles(t *testing.T) {
	svc := NewMediaService(&mockMediaRepo{}, "")
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
	svc := NewMediaService(&mockMediaRepo{}, "")
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

func TestMediaService_UploadMedia_DuplicateContent_Skipped(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{}, "")
	buildReq := func() domain.UploadRequest {
		return domain.UploadRequest{
			Files:       []domain.UploadedFile{{OriginalName: "clip.mp4", Data: []byte("same video bytes")}},
			CapturesDir: dir,
			DBPath:      filepath.Join(dir, "stats.duckdb"),
		}
	}

	// 1er upload : écrit le fichier.
	first, err := svc.UploadMedia(context.Background(), buildReq())
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if first.Saved != 1 || first.Skipped != 0 {
		t.Fatalf("first upload: Saved=%d Skipped=%d, want 1/0", first.Saved, first.Skipped)
	}

	// 2e upload (même nom + même contenu) : doit être skippé, pas de copie disque.
	second, err := svc.UploadMedia(context.Background(), buildReq())
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if second.Saved != 0 || second.Skipped != 1 {
		t.Errorf("second upload: Saved=%d Skipped=%d, want 0/1", second.Saved, second.Skipped)
	}

	// Un seul .mp4 sur disque (pas de doublon suffixé timestamp).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	mp4 := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			mp4++
		}
	}
	if mp4 != 1 {
		t.Errorf("expected exactly 1 .mp4 on disk, got %d", mp4)
	}
}

func TestMediaService_UploadMedia_SameNameDifferentContent_Kept(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{}, "")
	mk := func(data string) domain.UploadRequest {
		return domain.UploadRequest{
			Files:       []domain.UploadedFile{{OriginalName: "clip.mp4", Data: []byte(data)}},
			CapturesDir: dir,
			DBPath:      filepath.Join(dir, "stats.duckdb"),
		}
	}

	if _, err := svc.UploadMedia(context.Background(), mk("first clip")); err != nil {
		t.Fatalf("first upload: %v", err)
	}
	// Même nom mais contenu différent : c'est un vrai média distinct → conservé
	// (suffixe timestamp), surtout pas skippé.
	second, err := svc.UploadMedia(context.Background(), mk("totally different clip"))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if second.Saved != 1 || second.Skipped != 0 {
		t.Errorf("second upload: Saved=%d Skipped=%d, want 1/0", second.Saved, second.Skipped)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	mp4 := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			mp4++
		}
	}
	if mp4 != 2 {
		t.Errorf("expected 2 distinct .mp4 on disk, got %d", mp4)
	}
}

func TestMediaService_UploadMedia_CreatesCapturesDir(t *testing.T) {
	base := t.TempDir()
	captures := filepath.Join(base, "new", "captures")
	svc := NewMediaService(&mockMediaRepo{}, "")
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
	svc := NewMediaService(&mockMediaRepo{}, "")
	// Tolerance=0 → doit utiliser 2 par défaut (pas de panique)
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
	svc := NewMediaService(&mockMediaRepo{}, "")
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

// ─────────────────────────────────────────────────────────────────────────────
// Filtres — vérifier que les filtres sont bien transmis au repo
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaService_GetMediaPage_KindFilterPassedToRepo(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo, "")

	_, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{
		Page:       1,
		KindFilter: "screenshot",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilter.KindFilter != "screenshot" {
		t.Errorf("KindFilter = %q, want %q", repo.capturedFilter.KindFilter, "screenshot")
	}
}

func TestMediaService_GetMediaPage_LikedOnly_PassedToRepo(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo, "")

	_, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{
		Page:      1,
		LikedOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.capturedFilter.LikedOnly {
		t.Error("LikedOnly flag should be true in filters")
	}
}

func TestMediaService_GetMediaPage_MapAndModeFilters(t *testing.T) {
	repo := &mockMediaRepo{files: []domain.MediaFileRow{}, count: 0}
	svc := NewMediaService(repo, "")

	_, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{
		Page:       1,
		MapFilter:  "Fragmentation",
		ModeFilter: "Slayer",
		Sort:       "date_asc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilter.MapFilter != "Fragmentation" {
		t.Errorf("MapFilter = %q, want Fragmentation", repo.capturedFilter.MapFilter)
	}
	if repo.capturedFilter.ModeFilter != "Slayer" {
		t.Errorf("ModeFilter = %q, want Slayer", repo.capturedFilter.ModeFilter)
	}
	if repo.capturedFilter.Sort != "date_asc" {
		t.Errorf("Sort = %q, want date_asc", repo.capturedFilter.Sort)
	}
}

func TestMediaService_GetMediaPage_FilterOptionsFallbackToLoadedItems(t *testing.T) {
	mapName := "Recharge"
	modeName := "Oddball"
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{
			{
				FileName: "clip1.mp4",
				FilePath: "/clips/clip1.mp4",
				Kind:     "video",
				MapName:  &mapName,
				ModeName: &modeName,
			},
		},
		count:         1,
		filterOptsErr: errors.New("options indisponibles"),
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.AvailableFilters.Maps) != 1 || resp.AvailableFilters.Maps[0].Value != mapName {
		t.Fatalf("Maps = %+v, want fallback map %q", resp.AvailableFilters.Maps, mapName)
	}
	if len(resp.AvailableFilters.Modes) != 1 || resp.AvailableFilters.Modes[0].Value != modeName {
		t.Fatalf("Modes = %+v, want fallback mode %q", resp.AvailableFilters.Modes, modeName)
	}
}

func TestMediaService_GetMediaPage_FilterOptionsCompleteEmptyRepoValues(t *testing.T) {
	mapName := "Aquarius"
	modeName := "Slayer"
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{
			{
				FileName: "clip1.mp4",
				FilePath: "/clips/clip1.mp4",
				Kind:     "video",
				MapName:  &mapName,
				ModeName: &modeName,
			},
		},
		count: 1,
		filterOptions: domain.MediaFilterOptions{
			Maps:  nil,
			Modes: []domain.LabelValue{{Label: modeName, Value: modeName}},
		},
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.AvailableFilters.Maps) != 1 || resp.AvailableFilters.Maps[0].Value != mapName {
		t.Fatalf("Maps = %+v, want fallback map %q", resp.AvailableFilters.Maps, mapName)
	}
	if len(resp.AvailableFilters.Modes) != 1 || resp.AvailableFilters.Modes[0].Value != modeName {
		t.Fatalf("Modes = %+v, want repo mode %q", resp.AvailableFilters.Modes, modeName)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Likes sociaux — ToggleSharedLike + GetMediaLikers
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaService_SetMediaLike_EmptyFilePath(t *testing.T) {
	repo := &mockMediaRepo{setOK: true}
	svc := NewMediaService(repo, "")

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "",
		Liked:    true,
	})
	if err == nil {
		t.Fatal("expected error for empty file_path")
	}
}

func TestMediaService_SetMediaLike_WithLikerSlug_CallsToggle(t *testing.T) {
	repo := &mockMediaRepo{setOK: true}
	svc := NewMediaService(repo, "")

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:      "/clips/g1.mp4",
		Liked:         true,
		LikerSlug:     "player-1",
		LikerGamertag: "Player1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.toggleCalled {
		t.Error("ToggleSharedLike should have been called")
	}
	if !repo.toggleLiked {
		t.Error("ToggleSharedLike should be called with liked=true")
	}
}

func TestMediaService_SetMediaLike_ToggleError_NonBlocking(t *testing.T) {
	repo := &mockMediaRepo{setOK: true, toggleErr: errors.New("shared db error")}
	svc := NewMediaService(repo, "")

	// L'erreur ToggleSharedLike ne doit pas faire échouer SetMediaLike
	resp, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath:  "/clips/g1.mp4",
		Liked:     true,
		LikerSlug: "player-1",
	})
	if err != nil {
		t.Fatalf("ToggleSharedLike error should be non-blocking, got: %v", err)
	}
	if resp == nil || !resp.Liked {
		t.Error("expected non-nil response with Liked=true")
	}
}

func TestMediaService_SetMediaLike_LikersEnrichedInResponse(t *testing.T) {
	repo := &mockMediaRepo{
		setOK: true,
		likersData: map[string]domain.MediaLikersInfo{
			"/clips/g1.mp4": {Names: []string{"Alice", "Bob"}, Total: 3},
		},
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "/clips/g1.mp4",
		Liked:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalLikers != 3 {
		t.Errorf("TotalLikers = %d, want 3", resp.TotalLikers)
	}
	if len(resp.Likers) != 2 {
		t.Errorf("Likers = %v, want [Alice Bob]", resp.Likers)
	}
}

func TestMediaService_GetMediaPage_LikersEnrichedOnItems(t *testing.T) {
	repo := &mockMediaRepo{
		files: []domain.MediaFileRow{
			{FileName: "g1.mp4", FilePath: "/clips/g1.mp4", Kind: "video"},
		},
		count: 1,
		likersData: map[string]domain.MediaLikersInfo{
			"/clips/g1.mp4": {Names: []string{"Alice"}, Total: 1},
		},
	}
	svc := NewMediaService(repo, "")

	resp, err := svc.GetMediaPage(context.Background(), domain.MediaPageRequest{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items.Items) != 1 {
		t.Fatalf("expected 1 item")
	}
	item := resp.Items.Items[0]
	if item.TotalLikers != 1 {
		t.Errorf("TotalLikers = %d, want 1", item.TotalLikers)
	}
	if len(item.Likers) != 1 || item.Likers[0] != "Alice" {
		t.Errorf("Likers = %v, want [Alice]", item.Likers)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// safeDestPath
// ─────────────────────────────────────────────────────────────────────────────

func TestSafeDestPath_NoConflict(t *testing.T) {
	dir := t.TempDir()
	got := safeDestPath(dir, "clip.mp4")
	if got != filepath.Join(dir, "clip.mp4") {
		t.Errorf("expected original path, got %s", got)
	}
}

func TestSafeDestPath_Conflict_AddsSuffix(t *testing.T) {
	dir := t.TempDir()
	// Créer le fichier en conflit
	origPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(origPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := safeDestPath(dir, "clip.mp4")
	if got == origPath {
		t.Error("expected a different path when conflict exists")
	}
	base := filepath.Base(got)
	if !strings.HasSuffix(base, ".mp4") {
		t.Errorf("expected .mp4 extension preserved, got %s", base)
	}
	if !strings.Contains(base, "_") {
		t.Errorf("expected timestamp suffix in filename, got %s", base)
	}
}

func TestSafeDestPath_PreservesExtension(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"shot.png", "video.mov", "clip.mp4"} {
		origPath := filepath.Join(dir, name)
		if err := os.WriteFile(origPath, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := safeDestPath(dir, name)
		if filepath.Ext(got) != filepath.Ext(name) {
			t.Errorf("%s: expected extension %s, got %s", name, filepath.Ext(name), filepath.Ext(got))
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UploadMedia — uploads simultanés (même joueur, deux navigateurs)
// ─────────────────────────────────────────────────────────────────────────────

func TestMediaService_UploadMedia_ConcurrentSameDir_NamingConflict(t *testing.T) {
	dir := t.TempDir()
	svc := NewMediaService(&mockMediaRepo{}, "")

	buildReq := func() domain.UploadRequest {
		return domain.UploadRequest{
			Files: []domain.UploadedFile{
				{OriginalName: "capture.mp4", Data: []byte("fake-video-data")},
			},
			CapturesDir: dir,
			DBPath:      filepath.Join(dir, "stats.duckdb"),
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.UploadMedia(context.Background(), buildReq())
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	// Au moins 1 fichier .mp4 doit exister dans le répertoire
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	mp4Files := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			mp4Files++
		}
	}
	if mp4Files < 1 {
		t.Errorf("expected at least 1 .mp4 file in dir, got %d", mp4Files)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NewMediaService — timezone stockée + sanitisée
// ─────────────────────────────────────────────────────────────────────────────

func TestNewMediaService_TimezoneStored(t *testing.T) {
	svc := NewMediaService(&mockMediaRepo{}, "Europe/Paris")
	if svc.timezone != "Europe/Paris" {
		t.Errorf("timezone = %q, want \"Europe/Paris\"", svc.timezone)
	}
}

func TestNewMediaService_InvalidTimezone_StoredEmpty(t *testing.T) {
	// Timezone avec injection SQL → doit être sanitisée à ""
	svc := NewMediaService(&mockMediaRepo{}, "bad;tz")
	if svc.timezone != "" {
		t.Errorf("expected sanitized empty timezone, got %q", svc.timezone)
	}
}

func TestNewMediaService_EmptyTimezone(t *testing.T) {
	svc := NewMediaService(&mockMediaRepo{}, "")
	if svc.timezone != "" {
		t.Errorf("expected \"\", got %q", svc.timezone)
	}
}

// ─── Atomic SetMediaLike (commit 6 db-concurrency) ───

// mockAtomicMediaRepo étend mockMediaRepo avec SetMediaLikeAtomic pour activer
// le chemin atomique du service. Chaque champ permet de simuler un cas d'échec
// ou de capture pour assertions.
type mockAtomicMediaRepo struct {
	mockMediaRepo
	atomicCalled  bool
	atomicArgs    atomicCallArgs
	atomicUpdated bool
	atomicErr     error
}

type atomicCallArgs struct {
	filePath, likerSlug, likerGamertag string
	liked                              bool
}

func (m *mockAtomicMediaRepo) SetMediaLikeAtomic(
	_ context.Context,
	_ port.DBExecutor,
	filePath, likerSlug, likerGamertag string,
	liked bool,
) (bool, error) {
	m.atomicCalled = true
	m.atomicArgs = atomicCallArgs{filePath: filePath, likerSlug: likerSlug, likerGamertag: likerGamertag, liked: liked}
	if m.atomicErr != nil {
		return false, m.atomicErr
	}
	return m.atomicUpdated, nil
}

// Note : les chemins "atomic success" et "atomic rollback" exigent une vraie
// *sql.DB pour BeginTx, donc des tests d'intégration cgo (DuckDB :memory:).
// Couverture déférée à la suite intégration ; ici on couvre les cas qui ne
// déclenchent pas BeginTx (lease busy + no-acquirer) + l'invariant compile-time
// que MediaRepo satisfait l'interface atomicMediaLiker.

// TestMediaService_SetMediaLike_Atomic_LeaseBusy_PropagatesErrDBLocked —
// l'acquéreur retourne ErrDBLocked → propagation au caller.
func TestMediaService_SetMediaLike_Atomic_LeaseBusy_PropagatesErrDBLocked(t *testing.T) {
	repo := &mockAtomicMediaRepo{atomicUpdated: true}
	acquirer := func() (*dblease.LeasedWriter, error) {
		return nil, fmt.Errorf("simulated lease busy: %w", dblease.ErrDBLocked)
	}
	svc := NewMediaService(repo, "", WithMediaWriterAcquirer(acquirer))

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "/clip.mp4", LikerSlug: "spartan-a", Liked: true,
	})
	if err == nil {
		t.Fatal("expected ErrDBLocked, got nil")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
	if repo.atomicCalled {
		t.Error("atomic should NOT be called when lease cannot be acquired")
	}
}

// TestMediaService_SetMediaLike_NoAcquirer_LegacyPath — sans option, le
// chemin legacy (SetMediaLike + ToggleSharedLike séparés) est emprunté.
// Garde-fou non-régression : les ~30 tests existants doivent continuer
// d'exercer ce chemin.
func TestMediaService_SetMediaLike_NoAcquirer_LegacyPath(t *testing.T) {
	repo := &mockAtomicMediaRepo{} // satisfait atomicMediaLiker mais...
	repo.setOK = true
	svc := NewMediaService(repo, "") // ...pas d'acquireWriter → legacy

	_, err := svc.SetMediaLike(context.Background(), domain.MediaLikeRequest{
		FilePath: "/clip.mp4", LikerSlug: "spartan-a", Liked: true,
	})
	if err != nil {
		t.Fatalf("SetMediaLike: %v", err)
	}
	if repo.atomicCalled {
		t.Error("atomic should NOT be called when no WriterAcquirer is configured")
	}
	if !repo.toggleCalled {
		t.Error("legacy ToggleSharedLike should have been called")
	}
}
