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
 *   perf → perfTiers · skill → skillTiers · outcome → outcomeFilter
 *
 * Le tri du tableau (mode Matchs) est CLIENT et éphémère (état interne du tableau,
 * non persisté dans l'URL) — cf. thought_log 2026-07-17.
 *
 * Cf. plan nav-context-unification (Phase 1) et `@/lib/page-scope`.
 */
import { z } from 'zod'

import { csvToSet, setToCsv } from '@/lib/page-scope/serialize'
import type { MatchFilterOutcome, MatchFilterSpec } from '@/lib/match-nav/navContext'

export type SquadScope = '' | 'solo' | 'squad'

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
  // xuid optionnel transmis par le Classement : permet à l'Explorer de servir le
  // profil live d'un joueur absent des données locales (cf. goToExplorer).
  targetXuid: z.string().optional(),
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
})

/** Code outcome (DuckDB) → label canonique MatchFilterSpec. */
const OUTCOME_CODE_TO_LABEL: Record<string, MatchFilterOutcome> = {
  '2': 'win',
  '3': 'loss',
  '1': 'draw',
  '4': 'dnf',
}

/**
 * Construit un `MatchFilterSpec` (nav match-view) depuis le scope Explorer —
 * Phase 4 (connexion des filtres Explorer à la navigation contextuelle).
 *
 * Avant : ExplorerMatchesTable dérivait son filterSpec du `soloFilterStore`
 * (vide en contexte Explorer) → la nav prev/next tombait en chronologie globale
 * dès qu'on ouvrait un match depuis Explorer. Désormais le scope local pilote
 * directement le filterSpec (multi-playlist/mode supportés par le backend Q25
 * depuis la Phase 3).
 *
 * Mapping :
 *   - playlists    → playlist_names (égalité exacte, IN (...) backend)
 *   - modeNames    → mode_categories (cohérent avec le contextDescriptor qui
 *     traite déjà modeNames comme catégorie ; catégorie non résolue = ignorée
 *     côté backend, dégradation gracieuse)
 *   - dates        → date_from/date_to (T00:00:00Z / T23:59:59Z inclusif)
 *   - outcome      → seulement si exactement 1 sélectionné (MatchFilterSpec.outcome
 *     est mono-valeur ; multi-outcome → on n'applique pas le filtre)
 *
 * Note : la nav in-liste reste portée par `matchIds` (exact) ; ce filterSpec
 * est la couche de durabilité F5/lien partagé. Retourne `undefined` si aucun
 * axe mappable (URL non polluée, fallback Q25 global).
 */
export function explorerScopeToFilterSpec(s: ExplorerScope): MatchFilterSpec | undefined {
  const spec: MatchFilterSpec = {}
  if (s.playlists.size > 0) spec.playlist_names = [...s.playlists]
  if (s.modeNames.size > 0) spec.mode_categories = [...s.modeNames]
  if (s.startDate) spec.date_from = `${s.startDate}T00:00:00Z`
  if (s.endDate) spec.date_to = `${s.endDate}T23:59:59Z`
  if (s.outcomeFilter.size === 1) {
    const [code] = [...s.outcomeFilter]
    const label = code ? OUTCOME_CODE_TO_LABEL[code] : undefined
    if (label) spec.outcome = label
  }
  if (
    !spec.playlist_names &&
    !spec.mode_categories &&
    !spec.date_from &&
    !spec.date_to &&
    !spec.outcome
  ) {
    return undefined
  }
  return spec
}
