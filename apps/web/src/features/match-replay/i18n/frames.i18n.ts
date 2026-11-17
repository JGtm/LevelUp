/**
 * frames.i18n.ts — LES TEXTES DES TROIS CADRES DU LECTEUR (Standard, Cinéma, Plein cadre) : le
 * contrat (`FramesText`) et ses deux langues. Le choix du cadre, la sortie du Plein cadre et l'état
 * « plan tourné » de la vue.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servi par `REPLAY_TEXT[locale].frames` ; la parité FR/EN reste tenue par le typage
 * (`satisfies FramesText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */
import type { ReplayFrame } from '../model/replayFrame'

export interface FramesText {
  /** Le nom accessible du groupe de boutons qui choisit le cadre. */
  groupLabel: string
  /** Le nom de chaque cadre, tel que son bouton l'écrit. */
  frame: Record<ReplayFrame, string>
  /** Le bouton qui sort du Plein cadre et rend le cadre d'où l'on vient. */
  exit: string
  /** L'état de la vue quand le plan est tourné d'un quart de tour pour mieux remplir la zone. */
  rotatedPlan: string
}

export const FR_FRAMES = {
  groupLabel: 'Cadre du lecteur',
  frame: { standard: 'Standard', cinema: 'Cinéma', 'plein-cadre': 'Plein cadre' },
  exit: 'Quitter',
  rotatedPlan: 'Plan tourné',
} satisfies FramesText

export const EN_FRAMES = {
  groupLabel: 'Player layout',
  frame: { standard: 'Standard', cinema: 'Cinema', 'plein-cadre': 'Full frame' },
  exit: 'Exit',
  rotatedPlan: 'Rotated map',
} satisfies FramesText
