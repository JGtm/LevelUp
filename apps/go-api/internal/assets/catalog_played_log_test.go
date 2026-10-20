package assets

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/observability"
)

// lockedBuffer est un tampon de journal sûr en concurrence.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Une panne de lecture des versions jouées est journalisée une fois par fenêtre et par titre,
// pas une fois par requête ; un demandeur parti ne produit pas d'avertissement.
func TestMetadataCatalog_PlayedVersionsFailureIsLoggedOncePerTitle(t *testing.T) {
	observability.ResetThrottle()
	t.Cleanup(observability.ResetThrottle)
	logs := &lockedBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	store := NewDuckDBIndexStore(filepath.Join(t.TempDir(), "metadata.duckdb"))
	t.Cleanup(store.Close)
	cat := NewMetadataCatalog(store, func(string) bool { return true }).
		WithPlayedMapVersions(NewPlayedMapVersions((&countingLoader{err: errors.New("base partagée illisible")}).load))

	for _, title := range []string{"halo_infinite", "halo_5"} {
		for range 20 {
			ref := Ref{Kind: KindMapImage, TitleID: title, ID: "map-x", Variant: "v-x"}
			if got := cat.Classify(context.Background(), ref); got != VerdictUnavailable {
				t.Fatalf("Classify(%s) = %v, attendu VerdictUnavailable", ref, got)
			}
		}
	}
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	_ = cat.classifyPlayedVersion(gone, Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "map-y", Variant: "v-y"}, VerdictUnknown)

	if n := strings.Count(logs.String(), "versions de carte jouées illisibles"); n != 2 {
		t.Fatalf("%d avertissement(s) pour 40 requêtes sur 2 titres, attendu 2 (un par titre) :\n%s", n, logs.String())
	}
}
