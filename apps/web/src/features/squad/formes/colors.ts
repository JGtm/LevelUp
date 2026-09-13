/**
 * colors.ts — LES ENCRES du bloc « formes retenues », toutes en jetons
 * sémantiques (aucun hex, aucune classe Tailwind de couleur).
 *
 * LA CORRESPONDANCE AVEC L'ARTEFACT, une fois pour toutes :
 *
 *   `--moi`      -> `squad-player-1`  (le joueur de la page, la MÊME encre que
 *                                      sa pastille et ses autres graphes)
 *   `--j2/--j3`  -> SQUAD_TEAMMATE_COLOR_TOKENS (source unique de la feature)
 *   `--j4`       -> `team-ally`       (coéquipier hors escouade : mon camp, sans
 *                                      identité de joueur)
 *   `--plus`     -> `divergent-pos`   (au-dessus de la référence)
 *   `--moins`    -> `divergent-neg`   (en dessous)
 *   `--parite`   -> `warning`         (le REPÈRE, jamais une donnée)
 *   `--fam-*`    -> USAGE_METRIC_TOKENS (features/_shared/usage) : les gestes
 *                                      gardent l'encre qu'ils ont sur la vue
 *                                      match et la page Sessions
 *   hachure adverse -> motif neutre (l'adversaire est compté, jamais coloré)
 */
import type { CSSProperties } from 'react'

import { USAGE_METRIC_TOKENS } from '@/features/_shared/usage/usageMetricKinds'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from '../colors'
import type { EquipmentAxis } from './model/access'

/** Le trait de parité — un jeton DISTINCT, jamais une teinte de donnée. */
export const PARITY_INK = tokenCssVar('warning')
/** Au-dessus / en dessous de la référence : comparer, pas juger. */
export const PLUS_INK = tokenCssVar('divergent-pos')
export const MINUS_INK = tokenCssVar('divergent-neg')
/** Le fond d'une piste vide, et l'encre d'une étendue (non mesurée ≠ donnée). */
export const TRACK_INK = 'var(--muted)'
export const SPREAD_INK = 'var(--muted-foreground)'
/** Mon camp sans identité de joueur (coéquipier hors escouade). */
export const TEAM_REST_INK = tokenCssVar('team-ally')

/**
 * L'encre d'un joueur de l'escouade par son RANG (0 = le joueur de la page).
 * Même convention que toute l'app : la couleur d'un joueur ne change pas d'un
 * écran à l'autre.
 */
export function squadPlayerInk(index: number): string {
  if (index <= 0) return tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN)
  const token = SQUAD_TEAMMATE_COLOR_TOKENS[(index - 1) % SQUAD_TEAMMATE_COLOR_TOKENS.length]
  return tokenCssVar(token)
}

/** Le jeton d'un geste d'équipement — celui de la vue match et de Sessions. */
const AXIS_TOKENS: Record<EquipmentAxis, SemanticToken> = {
  camo: USAGE_METRIC_TOKENS.camo,
  wall: USAGE_METRIC_TOKENS.wall,
  overshield: USAGE_METRIC_TOKENS.overshield,
  grapple: USAGE_METRIC_TOKENS.grapple,
  dropped: USAGE_METRIC_TOKENS.dropped,
}

/** L'encre d'un geste d'équipement. */
export function axisInk(axis: EquipmentAxis): string {
  return tokenCssVar(AXIS_TOKENS[axis])
}

/**
 * LA HACHURE DE L'ADVERSAIRE — deuxième copie ASSUMÉE du motif de
 * `features/_shared/usage/UsageForms.tsx` (l'import croisé entre features est
 * interdit par le ratchet). À la troisième copie : centraliser dans
 * `components/` et poser le garde-rail (CLAUDE.md n°6).
 *
 * L'adversaire n'a ni couleur d'équipe ni nom : il est compté, jamais affiché.
 */
export const ENEMY_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 4px, var(--muted-foreground) 4px, var(--muted-foreground) 6px)',
  opacity: 0.45,
}

/**
 * LA HACHURE DU NON MESURÉ — le match sans film décodé. Même grammaire que
 * celle de l'adversaire (une hachure, jamais un aplat), mais plus pâle : ce
 * n'est pas une donnée, c'est une absence.
 */
export const UNMEASURED_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 3px, var(--muted-foreground) 3px, var(--muted-foreground) 4px)',
  opacity: 0.3,
}

/**
 * L'encre d'une case de la bande : au-dessus ou en dessous de la parité, avec
 * une intensité qui SATURE À TRENTE POINTS d'écart (règle de l'artefact — une
 * échelle sans plafond ferait disparaître les écarts ordinaires).
 */
export function bandCellInk(gapPoints: number): string {
  const intensity = Math.min(1, Math.abs(gapPoints) / 30)
  const ink = gapPoints >= 0 ? PLUS_INK : MINUS_INK
  return `color-mix(in oklab, ${ink} ${Math.round(28 + intensity * 72)}%, var(--muted))`
}
