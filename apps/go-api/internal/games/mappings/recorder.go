package mappings

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

// LookupRecorder rate-limit les logs `field_lookup_missing` pour éviter le
// flooding quand un caller demande un FieldKey inconnu en boucle (ex: un
// scoreboard de 16 joueurs × 40 frames/s).
//
// Politique du plan §8.3 :
//   - max 1024 entrées (title, key, locale)
//   - 1 log Warn par couple unique
//   - au-delà du seuil, suppress + compteur global incrémenté
//   - tous les ~5 minutes (caller-driven), un log `mappings_lookup_throttled`
//     peut être émis avec le dropped_count et le compteur reset
type LookupRecorder struct {
	logger  *slog.Logger
	seen    sync.Map // (titleSlug|key|locale) → struct{}
	bound   int      // max entrées avant suppress
	stored  atomic.Int64
	dropped atomic.Int64
}

// NewLookupRecorder construit un recorder avec une borne par défaut de 1024
// entrées. logger nil → slog.Default.
func NewLookupRecorder(logger *slog.Logger) *LookupRecorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &LookupRecorder{logger: logger, bound: 1024}
}

// WithBound permet de configurer la borne (utile en tests).
func (r *LookupRecorder) WithBound(b int) *LookupRecorder {
	r.bound = b
	return r
}

// Record loggue un Warn `field_lookup_missing` la première fois pour un
// couple (titleSlug, key, locale). Les occurrences suivantes sont suppress
// et incrémentent le compteur dropped.
//
// Au-delà de la borne, plus aucun nouveau log n'est émis (suppress total)
// pour éviter une explosion mémoire si un attaquant ou un bug demande des
// milliers de keys différentes.
func (r *LookupRecorder) Record(titleSlug, key, locale string) {
	cacheKey := titleSlug + "|" + key + "|" + locale
	if _, alreadySeen := r.seen.Load(cacheKey); alreadySeen {
		r.dropped.Add(1)
		return
	}
	if r.stored.Load() >= int64(r.bound) {
		// Borne atteinte : on n'enregistre plus de nouveaux couples mais on
		// continue d'incrémenter le compteur global pour observer le volume.
		r.dropped.Add(1)
		return
	}
	if _, loaded := r.seen.LoadOrStore(cacheKey, struct{}{}); loaded {
		// Race : un autre goroutine a stocké entre Load et LoadOrStore.
		r.dropped.Add(1)
		return
	}
	r.stored.Add(1)
	r.logger.Warn("field_lookup_missing",
		"title_slug", titleSlug,
		"field_key", key,
		"locale", locale,
	)
}

// Stats retourne le nombre d'entrées uniques loggées et le nombre de drops.
func (r *LookupRecorder) Stats() (stored, dropped int64) {
	return r.stored.Load(), r.dropped.Load()
}

// FlushDropped émet un log `mappings_lookup_throttled` avec le compteur
// dropped courant et le reset à zéro. À appeler périodiquement (ex: ticker
// 5min) par l'appelant pour avoir une visibilité sur le volume de lookups
// suppress.
//
// Retourne le nombre de drops avant reset (utile pour les tests).
func (r *LookupRecorder) FlushDropped() int64 {
	d := r.dropped.Swap(0)
	if d > 0 {
		r.logger.Info("mappings_lookup_throttled",
			"dropped_count", d,
			"unique_keys_seen", r.stored.Load(),
		)
	}
	return d
}
