package assets

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AssetConfig contient la configuration complète de la couche assets.
type AssetConfig struct {
	// CacheRootDir est le répertoire racine du cache FS (ex: "data/cache").
	CacheRootDir string
	// MetaDBPath est le chemin vers metadata.duckdb.
	MetaDBPath string
	// GameCMSBaseURL est l'URL de base de GameCMS (vide → défaut prod).
	GameCMSBaseURL string
	// TokenProvider est la fonction de récupération des tokens Halo.
	// Nil → seuls les assets publics (médailles, maps) peuvent être fetchés.
	TokenProvider TokenProvider
	// HTTPClient est le client HTTP à utiliser (nil → http.DefaultClient).
	HTTPClient *http.Client
	// ReconcileInterval est la périodicité du ReconcileWorker (0 → désactivé).
	ReconcileInterval time.Duration
}

// New crée et démarre un DefaultResolver avec la configuration fournie.
// Le caller doit appeler Close(ctx) lors du graceful shutdown.
//
// Note Phase 6 plan finition multi-titres : l'ancien override
// WithRootOverride(KindMapImage, StaticMapDir) a été retiré — il était
// partiellement cassé (le path résolu via LocalFSStore.Path() ajoutait
// {kind}/{titleID} sous le root override, ce qui ne correspondait pas à la
// réalité FS). La résolution d'URL d'image map passe désormais exclusivement
// par les helpers de internal/assets/static (couche 2 SRP) et l'adapter HI
// (internal/games/halo_infinite/adapter_asset_urls.go, couche 3).
func New(cfg AssetConfig) (*DefaultResolver, error) {
	if cfg.CacheRootDir == "" {
		return nil, fmt.Errorf("assets.New: CacheRootDir requis")
	}
	if cfg.MetaDBPath == "" {
		return nil, fmt.Errorf("assets.New: MetaDBPath requis")
	}

	// BinaryStore.
	fs := NewLocalFSStore(cfg.CacheRootDir)

	// IndexStore.
	idx := NewDuckDBIndexStore(cfg.MetaDBPath)

	// Ensure table (best-effort — si la DB est verrouillée au démarrage, on continue sans index).
	if idx.Available(context.Background()) {
		if err := idx.EnsureTable(context.Background()); err != nil {
			// Non-bloquant : l'index sera créé lors du premier accès ou par ReconcileWorker.
			_ = err
		}
	}

	// Fetcher.
	gameCMSFetcher := NewGameCMSFetcher(cfg.HTTPClient, cfg.TokenProvider, cfg.GameCMSBaseURL)
	medalFetcher := NewSpritesheetFallbackFetcher(gameCMSFetcher, cfg.GameCMSBaseURL)
	fetcher := &multiFetcher{fetchers: []Fetcher{medalFetcher, gameCMSFetcher}}

	// Metrics (no-op par défaut — remplacer par PrometheusMetrics en prod).
	metrics := Metrics(NoopMetrics{})

	// WriteQueue.
	queue := NewWriteQueue(idx, metrics)

	resolver := NewDefaultResolver(fs, idx, fetcher, queue, metrics)
	return resolver, nil
}

// multiFetcher est un dispatche vers le bon fetcher selon le Kind.
type multiFetcher struct {
	fetchers []Fetcher
}

func (m *multiFetcher) Supports(k Kind) bool {
	for _, f := range m.fetchers {
		if f.Supports(k) {
			return true
		}
	}
	return false
}

func (m *multiFetcher) Fetch(ctx context.Context, ref Ref) (Payload, error) {
	for _, f := range m.fetchers {
		if f.Supports(ref.Kind) {
			return f.Fetch(ctx, ref)
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, ref.Kind)
}
