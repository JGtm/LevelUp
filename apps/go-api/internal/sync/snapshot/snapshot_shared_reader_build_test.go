package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
)

// writeCurrentVersion pose CURRENT.json pour que la version courante du titre vaille `v`.
func writeCurrentVersion(t *testing.T, paths *title.PathResolver, v int) {
	t.Helper()
	p := paths.SnapshotCurrentManifestPath(title.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"version": ` + strconv.Itoa(v) + `}`)
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSnapshotReader_BuildOffRequestPath : pendant qu'une version se construit, Get rend le
// live sans attendre ; l'annulation de la requête déclenchante n'interrompt pas la
// construction ; une fois construite, la version est servie.
func TestSnapshotReader_BuildOffRequestPath(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir(), nil)
	writeCurrentVersion(t, paths, 1)
	live := &stubLiveReader{db: &sql.DB{}}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)
	defer r.Close()

	release := make(chan struct{})
	snap := &sql.DB{}
	var buildErr error
	r.open = func(ctx context.Context, _ *title.PathResolver, _ string) (*ops.SnapshotQuerier, error) {
		<-release
		buildErr = ctx.Err()
		return &ops.SnapshotQuerier{DB: snap, Version: 1}, nil
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	done := make(chan *sql.DB, 1)
	go func() {
		db, rel, _ := r.Get(reqCtx)
		rel()
		done <- db
	}()
	select {
	case db := <-done:
		if db != live.db {
			t.Fatal("pendant la construction, Get doit servir le live")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get a attendu la construction de la version")
	}
	cancel() // la requête déclenchante se termine avant la construction
	close(release)
	r.builds.Wait()
	if buildErr != nil {
		t.Fatalf("la construction a hérité de l'annulation de la requête : %v", buildErr)
	}

	db, rel, _ := r.Get(context.Background())
	rel()
	if db != snap {
		t.Fatal("version construite : Get doit servir le snapshot")
	}
}

// TestSnapshotReader_SingleBuildAndNegativeCache : une seule construction par version même
// sous Get concurrents, et un échec n'est pas retenté pour la même version.
func TestSnapshotReader_SingleBuildAndNegativeCache(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir(), nil)
	writeCurrentVersion(t, paths, 2)
	live := &stubLiveReader{db: &sql.DB{}}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)
	defer r.Close()

	release := make(chan struct{})
	calls := 0
	r.open = func(context.Context, *title.PathResolver, string) (*ops.SnapshotQuerier, error) {
		calls++
		<-release
		return nil, errors.New("schéma incomplet")
	}
	for range 5 {
		_, rel, _ := r.Get(context.Background())
		rel()
	}
	close(release)
	r.builds.Wait()
	for range 5 {
		db, rel, _ := r.Get(context.Background())
		rel()
		if db != live.db {
			t.Fatal("version en échec : Get doit servir le live")
		}
	}
	r.builds.Wait()
	if calls != 1 {
		t.Fatalf("constructions = %d, attendu 1 (une par version, échec mémorisé)", calls)
	}
}

// stubLiveReader : SharedReader live de repli, rend une DB sentinelle.
type stubLiveReader struct{ db *sql.DB }

func (s *stubLiveReader) Get(context.Context) (*sql.DB, func(), error) {
	return s.db, func() {}, nil
}
