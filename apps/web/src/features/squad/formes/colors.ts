/**
 * colors.ts — LES ENCRES partagées par l'Emprise et les cartes d'objectif (Escouade, Séries
 * temporelles), toutes en jetons sémantiques (aucun hex, aucune classe Tailwind de couleur).
 *
 *   - le joueur de la page -> `squad-player-1` (`SQUAD_MAIN_PLAYER_INK`) ; les coéquipiers de
 *     l'Escouade prennent la palette de la page (`useSquadPlayerPalette`, ordre de la sélection) :
 *     la couleur d'un joueur ne change pas d'un écran à l'autre ;
 *   - le reste de mon camp (sans identité de joueur) -> `team-ally` à demi-opacité ;
 *   - au-dessus / en dessous de la référence -> `divergent-pos` / `divergent-neg` ;
 *   - la hachure du non mesuré -> motif neutre (le match sans film, jamais une donnée).
 */
import type { CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

import { SQUAD_MAIN_PLAYER_TOKEN } from '../colors'

/** Au-dessus / en dessous de la référence : comparer, pas juger. */
export const PLUS_INK = tokenCssVar('divergent-pos')
export const MINUS_INK = tokenCssVar('divergent-neg')
/** Le fond d'une piste vide (non mesurée ≠ donnée). */
export const TRACK_INK = 'var(--muted)'
/**
 * Mon camp SANS identité de joueur (coéquipier hors escouade) : l'encre de camp, à demi-opacité.
 * À pleine opacité, elle se confondait avec celle du joueur de la page (deux bleus voisins dans
 * la même barre).
 */
export const TEAM_REST_INK = `color-mix(in oklab, ${tokenCssVar('team-ally')} 55%, var(--muted))`

/** L'encre du joueur de la page (`squad-player-1`). */
export const SQUAD_MAIN_PLAYER_INK = tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN)

/**
 * LA HACHURE DU NON MESURÉ — le match sans film décodé (une hachure, jamais un aplat,
 * pâle : ce n'est pas une donnée, c'est une absence).
 */
export const UNMEASURED_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 3px, color-mix(in oklab, var(--muted-foreground) 30%, transparent) 3px, color-mix(in oklab, var(--muted-foreground) 30%, transparent) 4px)',
}

/**
 * L'encre d'une case de la bande : au-dessus ou en dessous de la parité, avec
 * une intensité qui SATURE À TRENTE POINTS d'écart (une échelle sans plafond ferait
 * disparaître les écarts ordinaires).
 */
export function bandCellInk(gapPoints: number): string {
  const intensity = Math.min(1, Math.abs(gapPoints) / 30)
  const ink = gapPoints >= 0 ? PLUS_INK : MINUS_INK
  return `color-mix(in oklab, ${ink} ${Math.round(28 + intensity * 72)}%, var(--muted))`
}
