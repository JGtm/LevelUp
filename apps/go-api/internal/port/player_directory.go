// Package port — player_directory.go : l'annuaire des joueurs (ADR 0035 D2).
//
// Quatre registres décrivent un joueur, avec quatre cycles de vie et quatre
// niveaux de sensibilité : le compte (`data/auth/users.json`), les credentials
// (`data/auth/watcher_tokens/{xuid}.json`), le profil de suivi
// (`db_profiles.json`) et le suivi live (le daemon watcher). Ils restent
// séparés — ce port est ce qui manque : UNE frontière derrière laquelle ils se
// lisent ensemble, par xuid.
//
// L'implémentation vit dans `internal/service/playerdirectory/`. Elle compose
// les stores existants par petites interfaces de lecture et ne touche JAMAIS à
// l'entrepôt partagé (ADR 0035 D6).
package port

import (
	"context"
	"errors"

	"levelup/go-api/internal/domain"
)

// ErrIdentityNotFound : aucun des registres ne connaît ce xuid.
var ErrIdentityNotFound = errors.New("player_directory: identité inconnue")

// PlayerDirectory est la lecture unifiée des registres d'identité.
//
// Périmètre de l'étape 3 du plan `.ai/PLAN_ANNUAIRE_JOUEURS_2026-09-15.md` :
// LECTURE seule. Les deux écritures prévues par l'ADR 0035 — `Onboard` (D4,
// chemin unique de création de profil) et `Purge` (D6) — rejoindront cette
// interface avec leur implémentation, jamais avant : un port qui déclare une
// méthode que personne n'implémente vraiment est un mensonge de compilation.
type PlayerDirectory interface {
	// List rend toutes les identités connues d'au moins un registre, avec leurs
	// anomalies et les compteurs d'en-tête.
	List(ctx context.Context) (domain.AdminIdentitiesResponse, error)
	// Get rend l'identité d'un xuid, ou ErrIdentityNotFound.
	Get(ctx context.Context, xuid string) (domain.IdentityRecord, error)
	// HasTrackedProfile dit si le couple (titre, xuid) est un profil SUIVI —
	// la question que posent les portes de l'ADR 0035 D3.
	HasTrackedProfile(ctx context.Context, titleSlug, xuid string) (bool, error)
}
