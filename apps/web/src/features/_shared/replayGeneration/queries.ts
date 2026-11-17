/**
 * TanStack Query — génération à la demande du rejeu d'un match qui n'en a pas (placement local : application
 * de bureau et poste de développement, capacité `can_generate_replays`) : état du match
 * (GET /players/{slug}/matches/{id}/replay/generation) et demande de génération
 * (POST .../replay/generate).
 *
 * Types écrits à la main : `openapi-gen` décrit le routeur du profil démo, qui ne monte aucune route sous
 * capacité ; ils entreront dans `generated.ts` avec le routeur de contrat (liste MANUAL_CAPABILITY_TYPES de
 * lib/api/response-types.guard.test.ts, qui le rappelle dès qu'ils y entrent).
 */

/** États servis par le serveur : des codes, les textes sont écrits ici (`text.ts`). */
export type ReplayGenerationState =
  | 'generatable'
  | 'queued'
  | 'running'
  | 'failed'
  | 'film_unavailable'
  | 'not_supported'
  | 'ready'

/** Raisons d'un échec (`failed`) ou d'un refus (`not_supported`). */
export type ReplayGenerationReason =
  | 'film_not_ready'
  | 'fetch_failed'
  | 'token_unavailable'
  | 'build_failed'
  | 'map_not_in_catalog'
  | 'unknown_film_key'
  | 'mode_unmanaged'
  | 'match_unknown'
  | 'no_token'

/** Étape d'une demande en cours : le film se télécharge, puis le rejeu se construit. */
export type ReplayGenerationPhase = 'downloading' | 'building'

export interface ReplayGenerationStatus {
  state: ReplayGenerationState
  reason?: ReplayGenerationReason
  /** Seulement en `running`. */
  phase?: ReplayGenerationPhase
  /** Seulement en `queued` : 1 = le prochain. */
  position?: number
  /** La file attend la fin de la partie (mode jeu). */
  waiting_for_game?: boolean
  /** La file est en pause à la demande de l'utilisateur. */
  paused?: boolean
  /** Seulement en `failed` : un nouvel essai a une chance d'aboutir. */
  retryable?: boolean
}

/** Corps de la réponse de POST .../replay/generate : `created` est faux quand la demande existait déjà. */
export interface ReplayGenerationOutcome {
  created: boolean
  status: ReplayGenerationStatus
}
