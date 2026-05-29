/**
 * explorerScope — contrat d'état persistable de la page Explorer (mode Matchs).
 *
 * Définit les deux représentations manipulées par `usePageScope` :
 *   - `ExplorerScope`        : état riche consommé par ExplorerPage (Set<string>)
 *   - `EncodedExplorerScope` : forme plate sérialisée en query params d'URL
 *
 * + le schéma Zod `explorerSearchSchema` utilisé par `validateSearch` de la
 *   route (inclut `mode`/`target`, gérés séparément côté composant).
 *
 * Param URL ↔ champ applicatif :
 *   start → startDate · end → endDate · scope → squadScope · mid → matchIDSearch
 *   exp → expTypes · pl → playlists · maps → mapNames · modes → modeNames
 *   perf → perfTiers · skill → skillTiers · outcome → outcomeFilter · sort → sortKey
 *
 * Cf. plan nav-context-unification (Phase 1) et `@/lib/page-scope`.
 */
import { z } from 'zod'

import { csvToSet, setToCsv } from '@/lib/page-scope/serialize'

export type SquadScope = '' | 'solo' | 'squad'

/** Tri par défaut — omis de l'URL quand actif (URL propre). */
export const DEFAULT_SORT_KEY = 'start_time:desc'

/** État riche consommé par ExplorerPage. */
export interface ExplorerScope {
  startDate: string
  endDate: string
  squadScope: SquadScope
  matchIDSearch: string
  expTypes: Set<string>
  playlists: Set<string>
  mapNames: Set<string>
  modeNames: Set<string>
  perfTiers: Set<string>
  skillTiers: Set<string>
  outcomeFilter: Set<string>
  sortKey: string
}

/** Forme plate sérialisée en query params (toutes les clés optionnelles). */
export interface EncodedExplorerScope {
  start?: string
  end?: string
  scope?: SquadScope
  mid?: string
  exp?: string
  pl?: string
  maps?: string
  modes?: string
  perf?: string
  skill?: string
  outcome?: string
  sort?: string
}

/** Clés de scope dans l'URL (détection cold-start + reset). */
export const EXPLORER_URL_KEYS: readonly (keyof EncodedExplorerScope)[] = [
  'start',
  'end',
  'scope',
  'mid',
  'exp',
  'pl',
  'maps',
  'modes',
  'perf',
  'skill',
  'outcome',
  'sort',
]

/** App → URL. Toutes les clés présentes ; `undefined` pour les valeurs vides
 *  (param omis de l'URL et du miroir). */
export function encodeExplorerScope(s: ExplorerScope): EncodedExplorerScope {
  return {
    start: s.startDate || undefined,
    end: s.endDate || undefined,
    scope: s.squadScope || undefined,
    mid: s.matchIDSearch || undefined,
    exp: setToCsv(s.expTypes),
    pl: setToCsv(s.playlists),
    maps: setToCsv(s.mapNames),
    modes: setToCsv(s.modeNames),
    perf: setToCsv(s.perfTiers),
    skill: setToCsv(s.skillTiers),
    outcome: setToCsv(s.outcomeFilter),
    sort: s.sortKey && s.sortKey !== DEFAULT_SORT_KEY ? s.sortKey : undefined,
  }
}

/** URL (partielle) → App. Remplit les défauts pour les params absents. */
export function decodeExplorerScope(raw: Partial<EncodedExplorerScope>): ExplorerScope {
  const sc = raw.scope
  return {
    startDate: raw.start ?? '',
    endDate: raw.end ?? '',
    squadScope: sc === 'solo' || sc === 'squad' ? sc : '',
    matchIDSearch: raw.mid ?? '',
    expTypes: csvToSet(raw.exp),
    playlists: csvToSet(raw.pl),
    mapNames: csvToSet(raw.maps),
    modeNames: csvToSet(raw.modes),
    perfTiers: csvToSet(raw.perf),
    skillTiers: csvToSet(raw.skill),
    outcomeFilter: csvToSet(raw.outcome),
    sortKey: raw.sort || DEFAULT_SORT_KEY,
  }
}

/**
 * Schéma de validation du search de la route Explorer.
 * `mode`/`target` (existants, gérés par ExplorerPage) + tous les filtres de
 * scope. Tout est optionnel : un param invalide est ignoré (chaîne libre côté
 * filtres, décodage tolérant dans `decodeExplorerScope`).
 */
export const explorerSearchSchema = z.object({
  mode: z.enum(['matches', 'player']).optional(),
  target: z.string().optional(),
  start: z.string().optional(),
  end: z.string().optional(),
  scope: z.enum(['solo', 'squad']).optional(),
  mid: z.string().optional(),
  exp: z.string().optional(),
  pl: z.string().optional(),
  maps: z.string().optional(),
  modes: z.string().optional(),
  perf: z.string().optional(),
  skill: z.string().optional(),
  outcome: z.string().optional(),
  sort: z.string().optional(),
})
