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
import { useMemo, useEffect, useState } from 'react'
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

    return {
      configured,
      frequent,
      remote,
      isRemoteLoading: remoteEnabled && remoteQuery.isFetching,
      hasAnyResult,
      remoteAttempted: remoteEnabled,
    }
  }, [
    query,
    availablePlayers,
    frequentOptions,
    excludeGamertags,
    remoteQuery.data,
    remoteQuery.isFetching,
    remoteEnabled,
  ])
}
