import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Capabilities DATA-LEVEL d'un titre — le second système de capabilities, à ne pas
 * confondre avec les capabilities TITLE-LEVEL de `capabilities.ts` :
 *
 * | | title-level (`capabilities.ts`) | data-level (ce fichier) |
 * |---|---|---|
 * | Clés | plates : `replay`, `objective_stats` | pointées : `film.replay_artifact` |
 * | Source Go | `domain/title/registry.go` | `games/adapter.go` + `capabilities.toml` |
 * | Canal | bootstrap (`availableTitles[].capabilities`) | `GET /titles/{slug}/capabilities` |
 * | Question | « cette PAGE existe-t-elle pour ce titre ? » | « ce titre PRODUIT-il cette donnée ? » |
 *
 * Le canal existait — servi, sans auth, dans le contrat OpenAPI — mais AUCUN client
 * ne le consommait : le front ne pouvait donc pas distinguer « ce titre ne sait pas
 * produire ce calque » de « ce match-là n'en a pas », et rendait des cartes vides sur
 * des titres qui n'auraient jamais de données (registre
 * `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`, L1 réfuté : « il manque un client, pas
 * un canal » ; L3, L4, L5).
 *
 * LA RÈGLE DES DEUX PORTES. Une surface alimentée par l'artefact de rejeu ou par une
 * table `film.*` en porte DEUX :
 *   1. la capability (ce TITRE a-t-il un film ?) — ce fichier ;
 *   2. la présence de donnée (ce MATCH-là en a-t-il ?) — l'état vide existant.
 * Un état vide sur un titre sans film est un bloc mort : la porte 1 le supprime, la
 * porte 2 reste ce qu'elle était.
 */

/**
 * Clés data-level consommées par le front. Sous-ensemble VOLONTAIRE du vocabulaire Go
 * (`games.AllCapabilityKeys()`) : N'Y ENTRE QU'UNE CLÉ EFFECTIVEMENT GATÉE CÔTÉ UI — une
 * clé listée « pour plus tard » serait du vocabulaire mort (CLAUDE.md n°7).
 *
 * Pourquoi `film.usage_summary` n'y est PAS, alors que le bloc d'usage des Sessions est
 * bien gaté : sa porte de titre arrive DÉJÀ dans le payload, en clair
 * (`usage.unavailable_reason === 'unsupported'` = « ce titre ne déclare pas
 * film.usage_summary », domain/session_usage.go). La lire une seconde fois ici ferait
 * deux sources de vérité pour une seule question, plus une requête inutile.
 *
 * Garde-rail : `internal/games/capabilities_front_parity_test.go` (Go) échoue si un
 * littéral passé à {@link useDataCapability} — ou à la prop `dataCapability` d'un
 * `FeatureGate` — ne correspond à AUCUNE CapabilityKey déclarée côté serveur. Une
 * typo fermerait la porte pour toujours, sans la moindre erreur visible.
 */
export const DATA_CAPABILITIES = ['film.kill_positions'] as const

export type DataCapabilityKey = (typeof DATA_CAPABILITIES)[number]

/** Statuts servis par `capabilities.toml` (miroir de `games.CapabilityStatus`). */
type DataCapabilityStatus = 'supported' | 'degraded' | 'not_exposed'

interface TitleCapabilitiesResponse {
  title_slug: string
  schema_version: number
  capabilities: Record<string, DataCapabilityStatus>
}

/** Cinq minutes : le serveur sert lui-même `Cache-Control: public, max-age=300`. */
const STALE_MS = 5 * 60 * 1000

/**
 * useTitleDataCapabilities — les capabilities data-level du titre COURANT, ou `null`
 * tant qu'elles ne sont pas connues (chargement, erreur, pas de titre résolu).
 *
 * Une seule requête par titre pour toute l'application : la clé de cache est le slug,
 * et le résultat est partagé par tous les appelants de {@link useDataCapability}.
 */
export function useTitleDataCapabilities(): Record<string, DataCapabilityStatus> | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.titleDataCapabilities(titleSlug),
    queryFn: () =>
      api.get<TitleCapabilitiesResponse>(`/titles/${encodeURIComponent(titleSlug)}/capabilities`),
    enabled: !!titleSlug,
    staleTime: STALE_MS,
  })
  return data?.capabilities ?? null
}

/**
 * hasDataCapabilityIn — prédicat pur, miroir hors-hook de {@link useDataCapability}.
 *
 * `supported` et `degraded` valent OUI (même sémantique que `games.CapabilityMap.Has` :
 * dégradé = partiel mais utilisable) ; `not_exposed` et l'absence valent NON.
 *
 * `caps === null` (pas encore chargées / erreur) ⇒ `true` : FAIL-OPEN, comme
 * {@link useCapability} côté title-level. Masquer pendant le chargement ferait
 * clignoter les sections du titre courant à chaque montage, et une panne du seul
 * endpoint de capabilities ne doit pas amputer l'application. Le masquage n'intervient
 * que sur une réponse REÇUE où la clé est absente ou `not_exposed`.
 */
export function hasDataCapabilityIn(
  caps: Record<string, DataCapabilityStatus> | null,
  key: DataCapabilityKey,
): boolean {
  if (caps == null) return true
  const status = caps[key]
  return status === 'supported' || status === 'degraded'
}

/**
 * useDataCapability — jumeau data-level de `useCapability` : `true` si le titre courant
 * déclare `key` (`supported` ou `degraded`). Fail-open tant que la réponse n'est pas là
 * (cf. {@link hasDataCapabilityIn}).
 */
export function useDataCapability(key: DataCapabilityKey): boolean {
  return hasDataCapabilityIn(useTitleDataCapabilities(), key)
}
