/**
 * Source unique des ids d'onglets de la page match + rétro-compat des deep-links.
 *
 * Centralisé ici (plutôt que dans la route ou dans MatchViewPage) pour être partagé
 * par le schéma de recherche de la route ET la page sans créer de cycle d'import —
 * même patron que `features/settings/tabs.ts`.
 */
import { z } from 'zod'

export const MATCH_VIEW_TABS = ['summary', 'chronology', 'players'] as const
export type MatchViewTab = (typeof MATCH_VIEW_TABS)[number]

/** Schéma des ids canoniques — utilisé par le `validateSearch` de la route. */
export const matchViewTabSchema = z.enum(MATCH_VIEW_TABS)

/**
 * L'onglet « Détails » a été scindé en « Chronologie » + « Joueurs » (2026-08-24).
 * Les deep-links `?tab=details` déjà partagés (favoris, liens) tombent sur
 * Chronologie : résolution au DÉCODAGE, sans redirection ni réécriture d'URL.
 */
const TAB_ALIASES: Record<string, MatchViewTab> = {
  details: 'chronology',
}

/** Résout une valeur brute d'URL en onglet valide (alias inclus) ; défaut `summary`. */
export function resolveMatchViewTab(raw: unknown): MatchViewTab {
  const parsed = matchViewTabSchema.safeParse(raw)
  if (parsed.success) return parsed.data
  if (typeof raw === 'string' && TAB_ALIASES[raw]) return TAB_ALIASES[raw]
  return 'summary'
}
