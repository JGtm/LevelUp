/**
 * usageInks.ts — LES ENCRES des formes partagées (jauge, bande de régularité et sa légende), en un
 * seul endroit : la légende doit peindre exactement les encres des cases qu'elle explique.
 *
 * COULEURS — jetons sémantiques uniquement.
 */
import { tokenCssVar } from '@/lib/accessibility'

/** L'encre du trait de parité — jeton distinct, jamais une teinte de donnée. */
export const PARITY_INK = tokenCssVar('warning')
/** L'encre de « nous » (le camp du joueur), surchargeable par l'accessibilité. */
export const ALLY_INK = tokenCssVar('team-ally')

/** Les quatre encres de la bande de régularité — source unique des cases ET de leur légende. */
export const BAND_TONE_INKS = {
  above: tokenCssVar('divergent-pos'),
  near: tokenCssVar('divergent-neutral'),
  below: tokenCssVar('divergent-neg'),
  /** Non mesuré : AUCUNE encre — la case reste sur le fond `bg-muted`, ce n'est pas une donnée. */
  unmeasured: undefined,
} as const
