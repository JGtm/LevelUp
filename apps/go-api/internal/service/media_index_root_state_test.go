package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/jobs"
)

const rootStateTitle = "halo_infinite"

// TestBuildMediaScanHook_UnavailableRootIsRecorded : dossier configuré absent, sans
// repli, et dossier du joueur absent : le bandeau en est informé.
func TestBuildMediaScanHook_UnavailableRootIsRecorded(t *testing.T) {
	root := t.TempDir()
	const gamertag = "HookRootGone"
	hook := BuildMediaScanHook(root, gamertag,
		func() string { return filepath.Join(root, "disque-debranche") }, nil, nil)
	hook(ctxkeys.WithTitleSlug(context.Background(), rootStateTitle))

	if st := ops.GetRootState(rootStateTitle, gamertag); !st.Unavailable {
		t.Errorf("état = %+v, want racine indisponible", st)
	}
}

// TestBuildMediaScanHook_NoConfiguredBaseIsNotUnavailable : sans dossier configuré, un
// dossier interne absent n'est pas une panne.
func TestBuildMediaScanHook_NoConfiguredBaseIsNotUnavailable(t *testing.T) {
	root := t.TempDir()
	const gamertag = "HookNoBase"
	hook := BuildMediaScanHook(root, gamertag, func() string { return "" }, nil, nil)
	hook(ctxkeys.WithTitleSlug(context.Background(), rootStateTitle))

	if st := ops.GetRootState(rootStateTitle, gamertag); st.Unavailable {
		t.Errorf("état = %+v, want aucun signal sans dossier configuré", st)
	}
}

// TestScanAllMedia_UnavailableRootIsRecorded : même règle pour le scan de tous les joueurs.
func TestScanAllMedia_UnavailableRootIsRecorded(t *testing.T) {
	root := t.TempDir()
	const gamertag = "ScanRootGone"
	pr := titlePkg.NewPathResolver(root)
	if err := os.MkdirAll(pr.PlayerDir(rootStateTitle, gamertag), 0o755); err != nil {
		t.Fatal(err)
	}
	store := jobs.NewStore(filepath.Join(root, "jobs.json"))
	err := NewDirMediaIndexer().ScanAllMedia(
		ctxkeys.WithTitleSlug(context.Background(), rootStateTitle),
		root, filepath.Join(root, "disque-debranche"), "", false, store, "job-root-state")
	if err != nil {
		t.Fatalf("ScanAllMedia: %v", err)
	}
	if st := ops.GetRootState(rootStateTitle, gamertag); !st.Unavailable {
		t.Errorf("état = %+v, want racine indisponible", st)
	}
}
