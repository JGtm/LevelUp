package skill

// skill_v2_options.go — options d'un cycle du shadow LUSR v2. Variadiques : les appelants
// qui n'en ont pas (replay shadow-only, tests) restent inchangés.

import (
	"context"
	"database/sql"
	"sync/atomic"
)

// MedalExploitLoader rend le score brut d'exploit médailles par match_id du joueur traité,
// lu sur un handle de LECTURE shared. Il vit dans internal/sync (metadata DB, difficulté des
// médailles) ; le paquet skill n'importe pas sync, d'où l'injection.
type MedalExploitLoader func(ctx context.Context, sharedDB *sql.DB) map[string]float64

// ShadowOption règle un cycle du shadow LUSR v2.
type ShadowOption func(*shadowOptions)

type shadowOptions struct {
	// ownerOnly : ne persister que l'état du joueur traité (RunLUSRV2ShadowOwnerOnly).
	ownerOnly bool
	// medalExploit : nil = pas de score d'exploit (composante absente des 8).
	medalExploit MedalExploitLoader
	// matchFailures : compte les matchs dont le calcul ou l'écriture a échoué (nil : pas de comptage).
	matchFailures *atomic.Int64
}

// WithMedalExploit fournit le chargeur du score d'exploit médailles, 6e composante des 8
// écrites dans lusr_component_history. Sans lui, cette composante est absente.
func WithMedalExploit(l MedalExploitLoader) ShadowOption {
	return func(o *shadowOptions) { o.medalExploit = l }
}

// WithMatchFailureCounter fait compter à n chaque match dont le chargement de l'état, le calcul,
// l'écriture de la note ou le persist de l'état échoue. Ces échecs sont journalisés sans arrêter
// la boucle : un appelant qui doit savoir que le rejeu est complet (rejeu après changement de
// formule) lit n à la fin. Sans l'option, le comportement est inchangé.
func WithMatchFailureCounter(n *atomic.Int64) ShadowOption {
	return func(o *shadowOptions) { o.matchFailures = n }
}

// noteMatchFailure compte un match en échec quand un compteur est fourni.
func (c shadowRunContext) noteMatchFailure() {
	if c.matchFailures != nil {
		c.matchFailures.Add(1)
	}
}

func buildShadowOptions(ownerOnly bool, opts []ShadowOption) shadowOptions {
	o := shadowOptions{ownerOnly: ownerOnly}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
