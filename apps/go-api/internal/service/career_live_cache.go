// Package service — career_live_cache.go : cache TTL process-level pour les
// fetches live carrière. Deux sous-caches indépendants :
//   - progress  → TTL court (5 min) : XP + rang qui bougent à chaque match
//   - customs   → TTL long  (6 h)   : ServiceTag + images, changent rarement
//
// Une horloge injectable (now func() time.Time) rend les tests TTL
// déterministes (cf. career_live_cache_test.go).
//
// Singleflight (par kind+xuid) déduplique les fetches concurrents pour le
// même joueur — calque du pattern challengesFetchSFGroup côté provider Halo.
package service

import (
	"sync"
	"time"

	"levelup/go-api/internal/domain"

	"golang.org/x/sync/singleflight"
)

// Cadences par défaut. Surchageables via CareerLiveCacheConfig pour les tests.
const (
	DefaultCareerProgressTTL      = 5 * time.Minute
	DefaultCareerCustomizationTTL = 6 * time.Hour
)

type careerProgressEntry struct {
	data      *domain.CareerRankSnapshot
	fetchedAt time.Time
}

type careerCustomEntry struct {
	data      *domain.SpartanCustomizationData
	fetchedAt time.Time
}

// CareerLiveCache mémorise les dernières réponses live Halo pour les éviter
// de re-spammer Waypoint à chaque chargement de page. Thread-safe.
//
// Le cache est process-level (tous les joueurs partagent l'instance) ; la clé
// d'entrée est le couple (xuid, titleSlug). Le titre fait partie de la clé
// (défense en profondeur V72-29) : un token Spartan reste valide entre titres,
// donc sans cette dimension une entrée « carrière Infinite » pourrait être servie
// sous le même xuid pour un autre titre. Pas de borne sur le nombre d'entrées : on
// suppose un nombre modéré de joueurs actifs simultanément (cas multi-user typique
// du produit). Si le besoin se présente, ajouter une LRU.
type CareerLiveCache struct {
	progressTTL time.Duration
	customTTL   time.Duration
	now         func() time.Time

	muP      sync.RWMutex
	progress map[string]careerProgressEntry

	muC     sync.RWMutex
	customs map[string]careerCustomEntry

	sfProgress singleflight.Group
	sfCustom   singleflight.Group
}

// CareerLiveCacheConfig permet de surcharger les TTL et l'horloge.
// Tous les champs sont optionnels — les zéro-valeurs prennent les defaults.
type CareerLiveCacheConfig struct {
	ProgressTTL      time.Duration
	CustomizationTTL time.Duration
	Now              func() time.Time
}

// NewCareerLiveCache crée un cache avec les TTL spécifiés (ou defaults).
func NewCareerLiveCache(cfg CareerLiveCacheConfig) *CareerLiveCache {
	pTTL := cfg.ProgressTTL
	if pTTL <= 0 {
		pTTL = DefaultCareerProgressTTL
	}
	cTTL := cfg.CustomizationTTL
	if cTTL <= 0 {
		cTTL = DefaultCareerCustomizationTTL
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &CareerLiveCache{
		progressTTL: pTTL,
		customTTL:   cTTL,
		now:         now,
		progress:    make(map[string]careerProgressEntry),
		customs:     make(map[string]careerCustomEntry),
	}
}

// cacheKey compose la clé d'entrée (xuid, titleSlug). Le NUL comme séparateur évite
// toute collision entre un xuid et un slug (aucun des deux ne contient d'octet NUL).
func cacheKey(xuid, titleSlug string) string {
	return xuid + "\x00" + titleSlug
}

// GetProgress retourne le snapshot caché si frais, sinon (nil, false).
func (c *CareerLiveCache) GetProgress(xuid, titleSlug string) (*domain.CareerRankSnapshot, bool) {
	c.muP.RLock()
	entry, ok := c.progress[cacheKey(xuid, titleSlug)]
	c.muP.RUnlock()
	if !ok {
		return nil, false
	}
	if c.now().Sub(entry.fetchedAt) >= c.progressTTL {
		return nil, false
	}
	return entry.data, true
}

// PutProgress mémorise le snapshot avec un timestamp égal à `now()`.
func (c *CareerLiveCache) PutProgress(xuid, titleSlug string, data *domain.CareerRankSnapshot) {
	c.muP.Lock()
	c.progress[cacheKey(xuid, titleSlug)] = careerProgressEntry{data: data, fetchedAt: c.now()}
	c.muP.Unlock()
}

// GetCustomization retourne la customisation cachée si fraîche, sinon (nil, false).
func (c *CareerLiveCache) GetCustomization(xuid, titleSlug string) (*domain.SpartanCustomizationData, bool) {
	c.muC.RLock()
	entry, ok := c.customs[cacheKey(xuid, titleSlug)]
	c.muC.RUnlock()
	if !ok {
		return nil, false
	}
	if c.now().Sub(entry.fetchedAt) >= c.customTTL {
		return nil, false
	}
	return entry.data, true
}

// PutCustomization mémorise la customisation avec un timestamp égal à `now()`.
func (c *CareerLiveCache) PutCustomization(xuid, titleSlug string, data *domain.SpartanCustomizationData) {
	c.muC.Lock()
	c.customs[cacheKey(xuid, titleSlug)] = careerCustomEntry{data: data, fetchedAt: c.now()}
	c.muC.Unlock()
}

// DoProgress exécute fn une seule fois pour un couple (xuid, titre) donné même en
// cas d'appels concurrents (pattern singleflight). Les erreurs ne sont pas
// cachées (chaque tentative suivante refait l'appel).
func (c *CareerLiveCache) DoProgress(xuid, titleSlug string, fn func() (*domain.CareerRankSnapshot, error)) (*domain.CareerRankSnapshot, error) {
	v, err, _ := c.sfProgress.Do(cacheKey(xuid, titleSlug), func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*domain.CareerRankSnapshot), nil
}

// DoCustomization exécute fn une seule fois pour un couple (xuid, titre) donné même
// en cas d'appels concurrents (pattern singleflight). Les erreurs ne sont pas
// cachées.
func (c *CareerLiveCache) DoCustomization(xuid, titleSlug string, fn func() (*domain.SpartanCustomizationData, error)) (*domain.SpartanCustomizationData, error) {
	v, err, _ := c.sfCustom.Do(cacheKey(xuid, titleSlug), func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*domain.SpartanCustomizationData), nil
}
