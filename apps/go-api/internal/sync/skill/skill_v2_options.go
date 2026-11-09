package skill

// skill_v2_options.go — options d'un cycle du shadow LUSR v2. Variadiques : les appelants
// qui n'en ont pas (replay shadow-only, tests) restent inchangés.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// ErrCanonicalWriteHeld signale qu'au moins une écriture canonical (match_skill_rank) a
// échoué pendant un cycle : le filigrane de son groupe est tenu, le match n'a pas de note
// et ne sera retenté qu'au cycle suivant. Rendu seulement avec WithErrorOnHeldWrites.
var ErrCanonicalWriteHeld = errors.New("écriture LUSR canonical tenue")

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
	// errorOnHeldWrites : un cycle qui a tenu au moins une écriture canonical rend
	// ErrCanonicalWriteHeld au lieu de nil.
	errorOnHeldWrites bool
}

// WithErrorOnHeldWrites fait rendre ErrCanonicalWriteHeld (avec le nombre de matchs traités)
// à un cycle dont au moins une écriture canonical a échoué. Pour les appelants qui n'ont
// pas de cycle suivant (rejeu complet, import, CLI) : sans elle, ces matchs restent sans
// note et l'appelant croit le rejeu complet. Le post-sync ne la pose pas : son prochain
// cycle retente.
func WithErrorOnHeldWrites() ShadowOption {
	return func(o *shadowOptions) { o.errorOnHeldWrites = true }
}

// heldWritesError rend l'erreur de fin de cycle demandée par WithErrorOnHeldWrites, ou nil.
func heldWritesError(cfg shadowOptions, s shadowRunStats) error {
	if !cfg.errorOnHeldWrites || s.skippedWriteFailed == 0 {
		return nil
	}
	return fmt.Errorf("RunLUSRV2Shadow: %d écriture(s) canonical tenue(s), %d match(s) traité(s): %w",
		s.skippedWriteFailed, s.processed, ErrCanonicalWriteHeld)
}

// WithMedalExploit fournit le chargeur du score d'exploit médailles, 6e composante des 8
// écrites dans lusr_component_history. Sans lui, cette composante est absente.
func WithMedalExploit(l MedalExploitLoader) ShadowOption {
	return func(o *shadowOptions) { o.medalExploit = l }
}

// WithMatchFailureCounter fait compter à n chaque match dont le chargement de l'état, le calcul,
// l'écriture de la note ou le persist de l'état échoue. Ces échecs sont journalisés sans arrêter
// la boucle : un appelant qui doit savoir que le rejeu est complet (rejeu après changement de
// formule) lit n à la fin. Sans l'option, le comportement est inchangé. Un match dont le solveur
// ne converge pas, même sans les observations de frags et de morts, n'est pas compté
// (noteComputeFailure).
func WithMatchFailureCounter(n *atomic.Int64) ShadowOption {
	return func(o *shadowOptions) { o.matchFailures = n }
}

// noteMatchFailure compte un match en échec quand un compteur est fourni.
func (c shadowRunContext) noteMatchFailure() {
	if c.matchFailures != nil {
		c.matchFailures.Add(1)
	}
}

// noteComputeFailure classe l'échec de calcul d'un match. Une non-convergence qui arrive ici a
// déjà été retentée sans les observations de frags et de morts (solveWithCountFallback) : un
// nouvel essai échouerait pareil, le rejeu est donc complet pour ce match, qui ne reçoit pas de
// note neuve. Elle est comptée à part ; toute autre erreur est un échec du match.
func (c shadowRunContext) noteComputeFailure(err error, s *shadowRunStats) {
	if errors.Is(err, skillv2.ErrNotConverged) {
		s.skippedNotConverged++
		return
	}
	c.noteMatchFailure()
}

func buildShadowOptions(ownerOnly bool, opts []ShadowOption) shadowOptions {
	o := shadowOptions{ownerOnly: ownerOnly}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
