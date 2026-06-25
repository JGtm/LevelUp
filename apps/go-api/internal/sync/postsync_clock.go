// Package sync — postsync_clock.go : chronométrage séquentiel des étapes du
// pipeline post-sync (dashboard monitoring P4).
//
// Principe « lap » : le pipeline est séquentiel, chaque lap(step) enregistre
// le temps écoulé depuis le lap précédent — zéro restructuration du pipeline
// (une ligne après chaque bloc), zéro closure autour des étapes existantes.
// Observability-only : aucune influence sur la logique de sync.
//
// Chaque lap alimente aussi l'agrégat expvar `postsync_step_ms_{step}`
// (count/sum/avg/max depuis le boot) → classement des étapes les plus
// lentes sans dépendre du dernier cycle.
package sync

import (
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// postSyncStepNames : étapes publiées (ordre du pipeline). Réexporté pour le
// registry (lecture des agrégats expvar par nom).
var postSyncStepNames = []string{
	"enrichment_rows",
	"scoring",
	"convergence_events",
	"weapon_kills",
	"convergence_psa",
	"citations",
	"dominance",
	"skill_rating",
	"csr_snapshots",
	"friends",
	"aggregates",
	"media_scan",
	"achievements",
	"snapshot_readiness",
}

// PostSyncStepNames retourne la liste ordonnée des étapes chronométrées.
func PostSyncStepNames() []string {
	out := make([]string, len(postSyncStepNames))
	copy(out, postSyncStepNames)
	return out
}

// postSyncClock chronomètre les étapes séquentielles d'un pipeline.
type postSyncClock struct {
	r         *domain.PostSyncResult
	titleSlug string // MT-05 : titre courant pour les agrégats expvar titrés
	start     time.Time
	last      time.Time
}

// newPostSyncClock démarre l'horloge sur le résultat donné, pour `titleSlug`.
func newPostSyncClock(r *domain.PostSyncResult, titleSlug string) *postSyncClock {
	now := time.Now()
	return &postSyncClock{r: r, titleSlug: titleSlug, start: now, last: now}
}

// lap enregistre l'étape écoulée depuis le lap précédent (durée + items) et
// alimente l'agrégat expvar titré de l'étape.
func (c *postSyncClock) lap(step string, items int) {
	now := time.Now()
	ms := now.Sub(c.last).Milliseconds()
	c.last = now
	c.r.StepTimings = append(c.r.StepTimings, domain.PostSyncStepTiming{
		Step: step, DurationMs: ms, Items: items,
	})
	observability.RecordDurationMST(c.titleSlug, "postsync_step_ms_"+step, ms)
}

// finish fige la durée totale du pipeline. À appeler en defer (couvre aussi
// le retour partiel après panic recover).
func (c *postSyncClock) finish() {
	c.r.DurationMs = time.Since(c.start).Milliseconds()
	observability.RecordDurationMST(c.titleSlug, "postsync_total_ms", c.r.DurationMs)
}
