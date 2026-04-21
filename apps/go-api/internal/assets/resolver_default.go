package assets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultResolver implémente Resolver avec le flux :
//
//  1. Lookup binaire FS                 → hit ? return SourceLocalFile.
//  2. Lookup index DuckDB (best-effort) → si Available()=false, traiter comme miss.
//  3. singleflight.Do(ref.String())     → fetcher.Fetch.
//  4. Persist FS (synchrone, atomique)  → échec = ErrPersistFailed.
//  5. Enqueue write index (async)       → le return HTTP n'attend jamais la DB.
//  6. return SourceRemote.
type DefaultResolver struct {
	binary  BinaryStore
	index   IndexStore
	fetcher Fetcher
	queue   *WriteQueue
	metrics Metrics
	sf      singleflight.Group
}

// NewDefaultResolver crée un resolver avec les composants injectés.
func NewDefaultResolver(binary BinaryStore, index IndexStore, fetcher Fetcher, queue *WriteQueue, m Metrics) *DefaultResolver {
	if m == nil {
		m = NoopMetrics{}
	}
	return &DefaultResolver{
		binary:  binary,
		index:   index,
		fetcher: fetcher,
		queue:   queue,
		metrics: m,
	}
}

// Get retourne l'asset identifié par ref (séquence 6 étapes).
func (r *DefaultResolver) Get(ctx context.Context, ref Ref) (Resolved, error) {
	start := time.Now()
	slog.Debug("assets: lookup", ref.LogAttrs()...)

	// Étape 1 : binaire local.
	if r.binary != nil {
		bin, err := r.binary.LookupBinary(ctx, ref)
		if err != nil {
			slog.Warn("assets: localfs read error", append(ref.LogAttrs(), "err", err)...)
		} else if bin != nil {
			r.metrics.IncHit(ref.Kind, SourceLocalFile)
			r.metrics.ObserveLatency(ref.Kind, SourceLocalFile, time.Since(start))
			slog.Debug("assets: cache_hit_fs", append(ref.LogAttrs(), "latency_ms", time.Since(start).Milliseconds())...)
			return Resolved{Payload: *bin, Source: SourceLocalFile, FetchedAt: time.Now(), ETag: bin.ETag}, nil
		}
	}

	// Étape 2 : index DuckDB (best-effort).
	if r.index != nil {
		if !r.index.Available(ctx) {
			slog.Warn("assets: index_unavailable", "path", "metadata.duckdb")
			r.metrics.IncIndexUnavailable()
		} else {
			entry, err := r.index.LookupIndex(ctx, ref)
			if err != nil {
				slog.Debug("assets: index lookup error", append(ref.LogAttrs(), "err", err)...)
			} else if entry != nil && entry.URL != "" {
				r.metrics.IncHit(ref.Kind, SourceLocalDB)
				r.metrics.ObserveLatency(ref.Kind, SourceLocalDB, time.Since(start))
				slog.Debug("assets: cache_hit_index", append(ref.LogAttrs(), "latency_ms", time.Since(start).Milliseconds())...)
				payload := r.entryToPayload(ref, entry)
				return Resolved{Payload: payload, Source: SourceLocalDB, FetchedAt: entry.FetchedAt, ETag: entry.ContentHash}, nil
			}
		}
	}

	// Étape 3 : cache miss → fetch distant via singleflight.
	r.metrics.IncMiss(ref.Kind)
	slog.Info("assets: cache_miss", ref.LogAttrs()...)

	result, err, _ := r.sf.Do(ref.String(), func() (any, error) {
		return r.fetchAndPersist(ctx, ref)
	})
	if err != nil {
		r.metrics.IncFetchError(ref.Kind)
		slog.Warn("assets: fetch_error", append(ref.LogAttrs(), "err", err)...)
		return Resolved{}, err
	}

	resolved := result.(Resolved)
	r.metrics.ObserveLatency(ref.Kind, SourceRemote, time.Since(start))
	return resolved, nil
}

// Refresh force un re-fetch depuis la source distante.
func (r *DefaultResolver) Refresh(ctx context.Context, ref Ref) (Resolved, error) {
	result, err, _ := r.sf.Do("refresh:"+ref.String(), func() (any, error) {
		return r.fetchAndPersist(ctx, ref)
	})
	if err != nil {
		return Resolved{}, err
	}
	return result.(Resolved), nil
}

// Warm pré-cache les refs de façon asynchrone.
func (r *DefaultResolver) Warm(ctx context.Context, refs ...Ref) {
	for _, ref := range refs {
		refCopy := ref
		go func() {
			if _, err := r.Get(ctx, refCopy); err != nil && !errors.Is(err, ErrNotFound) {
				slog.Debug("assets: warm failed", append(refCopy.LogAttrs(), "err", err)...)
			}
		}()
	}
}

// RegisterLocalFile enregistre un fichier FS existant dans l'index DuckDB.
func (r *DefaultResolver) RegisterLocalFile(ctx context.Context, ref Ref, path string) error {
	if r.index == nil {
		return nil
	}
	data, err := readFileBytes(path)
	if err != nil {
		return fmt.Errorf("registerLocalFile: read %s: %w", path, err)
	}
	entry := IndexEntry{
		Ref:         ref,
		LocalPath:   path,
		ContentHash: contentHash(data),
		FetchedAt:   time.Now(),
	}
	r.queue.Enqueue(ref, entry)
	return nil
}

// Close flush la WriteQueue et libère les ressources.
func (r *DefaultResolver) Close(ctx context.Context) error {
	if r.queue != nil {
		r.queue.Shutdown(ctx)
	}
	return nil
}

// fetchAndPersist fetche l'asset depuis la source distante et le persiste.
func (r *DefaultResolver) fetchAndPersist(ctx context.Context, ref Ref) (Resolved, error) {
	if r.fetcher == nil || !r.fetcher.Supports(ref.Kind) {
		return Resolved{}, fmt.Errorf("%w: %s", ErrUnsupportedKind, ref.Kind)
	}

	start := time.Now()
	payload, err := r.fetcher.Fetch(ctx, ref)
	if err != nil {
		return Resolved{}, err
	}

	latency := time.Since(start)
	fetchedAt := time.Now()
	var etag string
	var byteLen int

	// Étape 4 : persist FS si binaire.
	if bin, ok := payload.(BinaryPayload); ok && r.binary != nil {
		etag = bin.ETag
		byteLen = len(bin.Bytes)
		if persistErr := r.binary.PersistBinary(ctx, ref, bin); persistErr != nil {
			slog.Error("assets: persist_fs_failed",
				append(ref.LogAttrs(), "path", r.binary.Path(ref), "err", persistErr)...)
			return Resolved{}, persistErr
		}
	}
	if urlp, ok := payload.(URLPayload); ok {
		etag = contentHash([]byte(urlp.URL))
	}
	if jsonp, ok := payload.(JSONPayload); ok {
		etag = contentHash(jsonp.RawJSON)
		byteLen = len(jsonp.RawJSON)
	}

	slog.Info("assets: fetch_ok",
		append(ref.LogAttrs(), "bytes", byteLen, "latency_ms", latency.Milliseconds())...)

	// Étape 5 : enqueue write index (async).
	if r.queue != nil {
		entry := IndexEntry{
			Ref:         ref,
			ContentHash: etag,
			FetchedAt:   fetchedAt,
		}
		switch p := payload.(type) {
		case URLPayload:
			entry.URL = p.URL
		case BinaryPayload:
			if r.binary != nil {
				entry.LocalPath = r.binary.Path(ref)
			}
		case JSONPayload:
			entry.RawJSON = p.RawJSON
		}
		r.queue.Enqueue(ref, entry)
	}

	return Resolved{Payload: payload, Source: SourceRemote, FetchedAt: fetchedAt, ETag: etag}, nil
}

// entryToPayload convertit une IndexEntry en Payload selon le Kind.
func (r *DefaultResolver) entryToPayload(ref Ref, entry *IndexEntry) Payload {
	if ref.Kind.IsBinary() {
		if entry.LocalPath != "" {
			// Lire depuis FS si le chemin local est connu.
			if bin, err := r.binary.LookupBinary(context.Background(), ref); err == nil && bin != nil {
				return *bin
			}
		}
		if entry.URL != "" {
			return URLPayload{URL: entry.URL, ContentType: "image/png"}
		}
	}
	if len(entry.RawJSON) > 0 {
		return JSONPayload{RawJSON: entry.RawJSON}
	}
	if entry.URL != "" {
		return URLPayload{URL: entry.URL}
	}
	return URLPayload{URL: entry.URL}
}

// readFileBytes lit un fichier en entier via os.ReadFile.
func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// osReadFile est un alias pour les tests (peut être surchargé).
var osReadFile = os.ReadFile
