package skill

// skill_v2_options.go — options d'un cycle du shadow LUSR v2. Variadiques : les appelants
// qui n'en ont pas (replay shadow-only, tests) restent inchangés.

import (
	"context"
	"database/sql"
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
}

// WithMedalExploit fournit le chargeur du score d'exploit médailles, 6e composante des 8
// écrites dans lusr_component_history. Sans lui, cette composante est absente.
func WithMedalExploit(l MedalExploitLoader) ShadowOption {
	return func(o *shadowOptions) { o.medalExploit = l }
}

func buildShadowOptions(ownerOnly bool, opts []ShadowOption) shadowOptions {
	o := shadowOptions{ownerOnly: ownerOnly}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
