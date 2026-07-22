import type { TitleSummary } from '@/lib/api/types'

/** Verdict de gate d'un slug de titre d'URL (D-6). */
export type TitleGate = 'wait' | 'valid' | 'unknown' | 'coming_soon' | 'archived'

/**
 * resolveTitleGate — projette un slug de titre d'URL contre la liste des titres
 * du bootstrap. Fonction PURE (D-6, D-8). Le composant layout `t/$titleSlug`
 * consomme ce verdict (déclaratif, gaté sur bootstrap) :
 *  - `wait` : store pas encore bootstrappé → ne rien décider (loader `__root`) ;
 *  - `unknown` / `coming_soon` / `archived` : gate (parité `RequireActiveTitle`) ;
 *  - `valid` : titre servable → appliquer si divergence avec le titre courant.
 *
 * `status` absent est traité comme `active` (parité `buildTitleSwitcherEntries`).
 */
export function resolveTitleGate(
  titleSlug: string,
  availableTitles: TitleSummary[],
  isBootstrapped: boolean,
): TitleGate {
  if (!isBootstrapped) return 'wait'
  const found = availableTitles.find((t) => t.slug === titleSlug)
  if (!found) return 'unknown'
  const status = found.status ?? 'active'
  if (status === 'coming_soon') return 'coming_soon'
  if (status === 'archived') return 'archived'
  return 'valid'
}
