/**
 * Logique pure de `PageHeader` : type d'onglet, hauteurs fixes de l'en-tête, onglet actif. Séparée du
 * composant (le rafraîchissement à chaud ne s'applique qu'aux fichiers de composants).
 */
import type { RouteTo } from '@/components/shell/shellNavigation'
import type { TitleCapability } from '@/lib/capabilities/capabilities'
import { routeTemplateSuffix } from '@/lib/title-routing'

export interface PageHeaderTab {
  key: string
  /** Libellé déjà résolu dans la langue de l'interface. */
  label: string
  /** Template de route typé (`$titleSlug` et `$playerSlug` résolus par le composant). */
  to: RouteTo
  /** Capacité de titre requise : l'onglet n'est pas rendu quand le titre ne la déclare pas. */
  capability?: TitleCapability
}

/**
 * Hauteur minimale de la zone titre + chapeau : titre (28 px) + chapeau (20 px) = 48 px. Un chapeau
 * qui passe sur deux lignes (fenêtre étroite) agrandit la zone, jamais l'inverse. Constante partagée
 * avec les tests.
 */
export const PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS = 'min-h-12'
/** Hauteur de la ligne d'onglets, réservée même quand la section n'a pas d'onglets. */
export const PAGE_HEADER_TABS_HEIGHT_CLASS = 'h-11'
/**
 * Décalage `top` d'un bandeau collant posé SOUS la ligne d'onglets collante (même hauteur que
 * `PAGE_HEADER_TABS_HEIGHT_CLASS`) : sans lui, le bandeau collé à `top-0` recouvre les onglets.
 */
export const STICKY_BELOW_TABS_CLASS = 'top-11'
/** Idem, avec 8 px de marge, pour les bandeaux en carte (`top-2` quand rien ne les précède). */
export const STICKY_BELOW_TABS_GAP_CLASS = 'top-[3.25rem]'
/** Hauteur de la ligne d'onglets en rem, pour un `top` calculé (`calc`). */
export const PAGE_HEADER_TABS_HEIGHT_REM = 2.75

/**
 * L'onglet actif pour le suffixe de chemin relatif au joueur : celui dont le chemin est exactement
 * le suffixe ; à défaut, celui dont le chemin est le plus long préfixe de segments du suffixe (une
 * sous-page d'un onglet garde son onglet allumé). Pur.
 */
export function activeTabKey(suffix: string | null, tabs: readonly PageHeaderTab[]): string | null {
  if (suffix === null) return null
  let best: { key: string; length: number } | null = null
  for (const tab of tabs) {
    const tabSuffix = routeTemplateSuffix(tab.to)
    const matches = suffix === tabSuffix || (tabSuffix !== '' && suffix.startsWith(`${tabSuffix}/`))
    if (matches && (best === null || tabSuffix.length > best.length)) best = { key: tab.key, length: tabSuffix.length }
  }
  return best?.key ?? null
}

