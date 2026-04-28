package games

import (
	"fmt"
	"sync"
)

// StaticResolver est l'implémentation in-memory du Resolver.
//
// Construction : NewStaticResolver(defaultSlug). Enregistrement : Register*().
// Aucune persistance, pas de hot-reload : le resolver est pur agnostique du
// stockage et reçoit ses adapters par injection au boot.
type StaticResolver struct {
	mu          sync.RWMutex
	defaultSlug string
	data        map[string]TitleDataAdapter
	semantic    map[string]TitleSemanticAdapter
	assetURL    map[string]TitleAssetURLAdapter
}

// NewStaticResolver crée un resolver vide pour un slug par défaut donné.
func NewStaticResolver(defaultSlug string) *StaticResolver {
	if defaultSlug == "" {
		defaultSlug = "halo_infinite"
	}
	return &StaticResolver{
		defaultSlug: defaultSlug,
		data:        make(map[string]TitleDataAdapter),
		semantic:    make(map[string]TitleSemanticAdapter),
		assetURL:    make(map[string]TitleAssetURLAdapter),
	}
}

// RegisterData enregistre un TitleDataAdapter pour son TitleSlug().
func (r *StaticResolver) RegisterData(a TitleDataAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[a.TitleSlug()] = a
}

// RegisterSemantic enregistre un TitleSemanticAdapter pour son TitleSlug().
func (r *StaticResolver) RegisterSemantic(a TitleSemanticAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.semantic[a.TitleSlug()] = a
}

// RegisterAssetURL enregistre un TitleAssetURLAdapter pour son TitleSlug().
func (r *StaticResolver) RegisterAssetURL(a TitleAssetURLAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assetURL[a.TitleSlug()] = a
}

// Data retourne le TitleDataAdapter d'un slug ou ErrTitleNotResolved.
func (r *StaticResolver) Data(slug string) (TitleDataAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.data[slug]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrTitleNotResolved, slug)
}

// Semantic retourne le TitleSemanticAdapter d'un slug ou ErrTitleNotResolved.
func (r *StaticResolver) Semantic(slug string) (TitleSemanticAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.semantic[slug]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrTitleNotResolved, slug)
}

// AssetURL retourne le TitleAssetURLAdapter d'un slug ou ErrTitleNotResolved.
func (r *StaticResolver) AssetURL(slug string) (TitleAssetURLAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.assetURL[slug]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrTitleNotResolved, slug)
}

// DefaultSlug retourne le slug par défaut configuré au boot.
func (r *StaticResolver) DefaultSlug() string { return r.defaultSlug }

// Slugs retourne les slugs enregistrés côté Data + Semantic + AssetURL (union).
func (r *StaticResolver) Slugs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for s := range r.data {
		seen[s] = struct{}{}
	}
	for s := range r.semantic {
		seen[s] = struct{}{}
	}
	for s := range r.assetURL {
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	return out
}
