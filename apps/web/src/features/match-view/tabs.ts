/**
 * Source unique des ids d'onglets de la page match + rétro-compat des deep-links.
 *
 * Centralisé ici (plutôt que dans la route ou dans MatchViewPage) pour être partagé
 * par le schéma de recherche de la route ET la page sans créer de cycle d'import —
 * même patron que `features/settings/tabs.ts`.
 */
import { z } from 'zod'

/**
 * Quatre onglets depuis le 2026-09-19. Le troisième s'appelle « Armes et terrain »
 * (`arsenal`) depuis le 2026-09-22 : il était né « Contrôle » (`control`) avec les usages
 * d'équipement, le contrôle des socles d'armes et l'occupation du terrain, sortis de
 * Chronologie ; il a repris de Général la répartition des frags et la distance des frags.
 * Son axe de lecture est désormais AVEC QUOI et OÙ, ce que « Contrôle » ne disait pas.
 */
export const MATCH_VIEW_TABS = ['summary', 'chronology', 'arsenal', 'players'] as const
export type MatchViewTab = (typeof MATCH_VIEW_TABS)[number]

/** Schéma des ids canoniques — utilisé par le `validateSearch` de la route. */
export const matchViewTabSchema = z.enum(MATCH_VIEW_TABS)

/**
 * L'onglet « Détails » a été scindé en « Chronologie » + « Joueurs » (2026-08-24).
 * Les deep-links `?tab=details` déjà partagés (favoris, liens) tombent sur
 * Chronologie : résolution au DÉCODAGE, sans redirection ni réécriture d'URL.
 *
 * `control` -> `arsenal` (2026-09-22) : même mécanique, pour les liens partagés entre le
 * 2026-09-19 et le renommage de l'onglet en « Armes et terrain ».
 */
const TAB_ALIASES: Record<string, MatchViewTab> = {
  details: 'chronology',
  control: 'arsenal',
}

/** Résout une valeur brute d'URL en onglet valide (alias inclus) ; défaut `summary`. */
export function resolveMatchViewTab(raw: unknown): MatchViewTab {
  const parsed = matchViewTabSchema.safeParse(raw)
  if (parsed.success) return parsed.data
  if (typeof raw === 'string' && TAB_ALIASES[raw]) return TAB_ALIASES[raw]
  return 'summary'
}
