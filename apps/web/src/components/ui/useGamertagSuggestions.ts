/**
 * useGamertagSuggestions — hook partagé pour la recherche de gamertag.
 *
 * Combine 3 sources, toujours dans cet ordre :
 *   1. Joueurs configurés     — local, store appShell.availablePlayers (DB)
 *   2. Coéquipiers fréquents  — local, prop frequentOptions (encounter_count)
 *   3. Recherche serveur      — remote, /directory/gamertags/search (xuid_aliases, fuzzy)
 *
 * Score local : exact (1000) > prefix (200) > substring (50). Pas de char-by-char ambigu.
 * Recherche serveur : déclenchée à partir de 2 caractères, debounce 250 ms,
 * dédupliquée des sources locales.
 */
import { useMemo, useEffect, useState, useCallback } from 'react'
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useAppShellStore } from '@/stores/appShellStore'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { TeammateOption, GamertagSearchResponse, GamertagSuggestion } from '@/lib/api/types'

// ─── Constantes ────────────────────────────────────────────────────────────────

const REMOTE_MIN_CHARS = 2
const REMOTE_DEBOUNCE_MS = 250
const REMOTE_LIMIT = 12

// ─── Types ─────────────────────────────────────────────────────────────────────

export interface ConfiguredItem {
  gamertag: string
  xuid: string
  score: number
  isConfigured: true
}

export interface FrequentItem {
  gamertag: string
  xuid: string | null | undefined
  score: number
  encounter_count: number
  isConfigured: false
}

export interface RemoteItem {
  gamertag: string
  xuid: string | null | undefined
  exact_match: boolean
  isConfigured: false
}

export interface UseGamertagSuggestionsResult {
  configured: ConfiguredItem[]
  frequent: FrequentItem[]
  remote: RemoteItem[]
  isRemoteLoading: boolean
  hasAnyResult: boolean
  /** True si la query est suffisamment longue pour avoir consulté le serveur. */
  remoteAttempted: boolean
  /**
   * Repli LIVE Xbox EXPLICITE (V72-24) : résultats de la résolution Xbox d'un joueur
   * jamais croisé localement, dédupliqués des autres sources. Vide tant que
   * l'utilisateur n'a pas déclenché triggerLiveSearch (bouton « Rechercher sur Xbox »).
   */
  liveResults: RemoteItem[]
  /** True pendant le round-trip du repli live (2-3 s). */
  isLiveLoading: boolean
  /** True quand un repli live a été déclenché pour la query courante. */
  liveAttempted: boolean
  /** Déclenche le repli live Xbox pour la query courante (intention explicite). */
  triggerLiveSearch: () => void
}

interface UseGamertagSuggestionsArgs {
  query: string
  frequentOptions?: TeammateOption[]
  excludeGamertags?: string[]
}

// ─── Scoring local ─────────────────────────────────────────────────────────────

function localScore(query: string, target: string): number {
  if (!query) return 1 // pas de query = tous visibles avec score uniforme
  const q = query.toLowerCase()
  const t = target.toLowerCase()
  if (t === q) return 1000
  if (t.startsWith(q)) return 200
  if (t.includes(q)) return 50
  return 0
}

// ─── Hook ──────────────────────────────────────────────────────────────────────

export function useGamertagSuggestions({
  query,
  frequentOptions = [],
  excludeGamertags = [],
}: UseGamertagSuggestionsArgs): UseGamertagSuggestionsResult {
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)

  // Debounced query for the remote call
  const [debouncedQuery, setDebouncedQuery] = useState(query)
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), REMOTE_DEBOUNCE_MS)
    return () => clearTimeout(t)
  }, [query])

  const trimmed = debouncedQuery.trim()
  const remoteEnabled = trimmed.length >= REMOTE_MIN_CHARS

  const remoteQuery = useQuery({
    queryKey: queryKeys.gamertagSearch(trimmed),
    queryFn: () =>
      api.get<GamertagSearchResponse>(
        `/directory/gamertags/search?q=${encodeURIComponent(trimmed)}&limit=${REMOTE_LIMIT}`,
      ),
    enabled: remoteEnabled,
    staleTime: 60 * 1000,
    placeholderData: keepPreviousData,
  })

  // Repli LIVE Xbox (V72-24) : désarmé par défaut (typeahead rapide, local seul).
  // Déclenché explicitement par triggerLiveSearch. Réinitialisé à chaque changement
  // de query — le repli est décidé par-query, jamais rémanent.
  const [liveArmedQuery, setLiveArmedQuery] = useState<string | null>(null)
  useEffect(() => {
    setLiveArmedQuery(null)
  }, [trimmed])
  const liveEnabled = liveArmedQuery !== null && liveArmedQuery === trimmed && remoteEnabled
  const liveQuery = useQuery({
    queryKey: queryKeys.gamertagSearch(trimmed, true),
    queryFn: () =>
      api.get<GamertagSearchResponse>(
        `/directory/gamertags/search?q=${encodeURIComponent(trimmed)}&limit=${REMOTE_LIMIT}&live=1`,
      ),
    enabled: liveEnabled,
    staleTime: 60 * 1000,
    placeholderData: keepPreviousData,
  })
  const triggerLiveSearch = useCallback(() => {
    if (trimmed.length >= REMOTE_MIN_CHARS) setLiveArmedQuery(trimmed)
  }, [trimmed])

  return useMemo<UseGamertagSuggestionsResult>(() => {
    const q = query.trim()
    const excluded = new Set(excludeGamertags)

    // Source 1 : joueurs configurés
    const configured: ConfiguredItem[] = availablePlayers
      .filter((p) => !excluded.has(p.gamertag))
      .map((p) => ({
        gamertag: p.gamertag,
        xuid: p.xuid,
        score: localScore(q, p.gamertag),
        isConfigured: true as const,
      }))
      .filter((item) => item.score > 0)
      .sort((a, b) => b.score - a.score || a.gamertag.localeCompare(b.gamertag))

    const configuredGts = new Set(configured.map((c) => c.gamertag))

    // Source 2 : coéquipiers fréquents (déduplique avec configured)
    const frequent: FrequentItem[] = frequentOptions
      .filter((o) => !excluded.has(o.gamertag) && !configuredGts.has(o.gamertag))
      .map((o) => ({
        gamertag: o.gamertag,
        xuid: o.xuid,
        score: localScore(q, o.gamertag),
        encounter_count: o.encounter_count,
        isConfigured: false as const,
      }))
      .filter((item) => item.score > 0)
      .sort(
        (a, b) =>
          b.score - a.score ||
          b.encounter_count - a.encounter_count ||
          a.gamertag.localeCompare(b.gamertag),
      )

    const frequentGts = new Set(frequent.map((f) => f.gamertag))

    // Source 3 : remote (déduplique avec configured + frequent)
    const remoteItems: GamertagSuggestion[] = remoteQuery.data?.items ?? []
    const remote: RemoteItem[] = remoteItems
      .filter(
        (r) =>
          !excluded.has(r.gamertag) &&
          !configuredGts.has(r.gamertag) &&
          !frequentGts.has(r.gamertag),
      )
      .map((r) => ({
        gamertag: r.gamertag,
        xuid: r.xuid,
        exact_match: r.exact_match,
        isConfigured: false as const,
      }))

    const hasAnyResult = configured.length + frequent.length + remote.length > 0

    // Repli live : items du fetch Xbox, dédupliqués de TOUTES les sources déjà affichées
    // (configuré / fréquent / remote local) → seuls les hits réellement nouveaux restent.
    const remoteGts = new Set(remote.map((r) => r.gamertag))
    const liveItems: GamertagSuggestion[] = liveQuery.data?.items ?? []
    const liveResults: RemoteItem[] = liveItems
      .filter(
        (r) =>
          !excluded.has(r.gamertag) &&
          !configuredGts.has(r.gamertag) &&
          !frequentGts.has(r.gamertag) &&
          !remoteGts.has(r.gamertag),
      )
      .map((r) => ({
        gamertag: r.gamertag,
        xuid: r.xuid,
        exact_match: r.exact_match,
        isConfigured: false as const,
      }))

    return {
      configured,
      frequent,
      remote,
      isRemoteLoading: remoteEnabled && remoteQuery.isFetching,
      hasAnyResult,
      remoteAttempted: remoteEnabled,
      liveResults,
      isLiveLoading: liveEnabled && liveQuery.isFetching,
      liveAttempted: liveArmedQuery !== null && liveArmedQuery === trimmed,
      triggerLiveSearch,
    }
  }, [
    query,
    trimmed,
    availablePlayers,
    frequentOptions,
    excludeGamertags,
    remoteQuery.data,
    remoteQuery.isFetching,
    remoteEnabled,
    liveQuery.data,
    liveQuery.isFetching,
    liveEnabled,
    liveArmedQuery,
    triggerLiveSearch,
  ])
}
