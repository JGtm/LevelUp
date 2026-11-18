/**
 * frames.i18n.ts — LES TEXTES DES TROIS CADRES DU LECTEUR (Standard, Cinéma, Plein cadre) : le
 * contrat (`FramesText`) et ses deux langues. Le choix du cadre, la sortie du Plein cadre, le retour, le lien vers la fiche et l'état
 * « plan tourné » de la vue, la position « Auto » de l'orientation et le commutateur 2D / 3D.
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de
 * taille du dépôt. Servi par `REPLAY_TEXT[locale].frames` ; la parité FR/EN reste tenue par le typage
 * (`satisfies FramesText` dans chaque langue, `satisfies ReplayText` au-dessus).
 */
import type { ReplayFrame } from '../model/replayFrame'

export interface FramesText {
  /** Le nom accessible du groupe de boutons qui choisit le cadre. */
  groupLabel: string
  /**
   * Le nom accessible et l'infobulle de l'icône d'un cadre : le nom du cadre suivi de la touche qui
   * y mène, entre parenthèses (« Cinéma (T) ») ; le nom seul pour un cadre sans touche.
   */
  frameTip: (frame: ReplayFrame) => string
  /** Le bouton qui sort du Plein cadre et rend le cadre d'où l'on vient. */
  exit: string
  /** Le nom de la touche qui sort du Plein cadre, écrit dans la capsule du bouton « Quitter ». */
  escapeKey: string
  /** Le nom accessible du bouton qui revient en arrière (en-têtes du Cinéma et du Plein cadre). */
  back: string
  /** Le lien vers la fiche du match (en-tête du Cinéma), tel que son texte l'écrit. */
  sheet: string
  /** L'état de la vue quand le plan est tourné d'un quart de tour pour mieux remplir la zone. */
  rotatedPlan: string
  /** L'infobulle du bouton « Plan tourné » de la barre d'une ligne : ce que l'Auto a fait, et ce que le clic rend. */
  rotatedPlanTip: string
  /** La position « Auto » de l'orientation dans le panneau Vue, et sa phrase d'aide (infobulle). */
  orientationAuto: string
  orientationAutoHint: string
  /** Le nom accessible du groupe 2D / 3D posé à gauche du bouton Vue (les libellés « 2D » et « 3D » ne se traduisent pas). */
  dimensionGroup: string
  /** L'option du panneau Vue qui pose le fil des éliminations sur le plan, et sa phrase d'aide (infobulle). */
  planFeed: string
  planFeedHint: string
}

/** La touche de chaque cadre qui en a une : celles que `useReplayShortcuts` câble sur T et F. */
const FRAME_KEYS: Partial<Record<ReplayFrame, string>> = { cinema: 'T', 'plein-cadre': 'F' }

/** Compose le nom d'un cadre et le rappel de sa touche, dans une langue. */
function tipOf(names: Record<ReplayFrame, string>) {
  return (frame: ReplayFrame): string => {
    const key = FRAME_KEYS[frame]
    return key ? `${names[frame]} (${key})` : names[frame]
  }
}

const FR_FRAME_NAMES: Record<ReplayFrame, string> = { standard: 'Standard', cinema: 'Cinéma', 'plein-cadre': 'Plein cadre' }
const EN_FRAME_NAMES: Record<ReplayFrame, string> = { standard: 'Standard', cinema: 'Cinema', 'plein-cadre': 'Full frame' }

export const FR_FRAMES = {
  groupLabel: 'Cadre du lecteur',
  frameTip: tipOf(FR_FRAME_NAMES),
  exit: 'Quitter',
  escapeKey: 'Échap',
  back: 'Retour',
  sheet: 'Fiche',
  rotatedPlan: 'Plan tourné',
  rotatedPlanTip: "Plan tourné automatiquement pour mieux remplir la zone. Cliquer pour revenir à l'orientation d'origine.",
  orientationAuto: 'Auto',
  orientationAutoHint: "Le plan se tourne tout seul quand il y gagne de la place à l'écran. Aucun choix n'est retenu pour cette carte.",
  dimensionGroup: 'Vue du rejeu',
  planFeed: 'Fil des éliminations',
  planFeedHint: 'Les cinq dernières éliminations, posées en bas à gauche du plan.',
} satisfies FramesText

export const EN_FRAMES = {
  groupLabel: 'Player layout',
  frameTip: tipOf(EN_FRAME_NAMES),
  exit: 'Exit',
  escapeKey: 'Esc',
  back: 'Back',
  sheet: 'Details',
  rotatedPlan: 'Rotated map',
  rotatedPlanTip: 'Map rotated automatically to fill the area better. Click to return to the original orientation.',
  orientationAuto: 'Auto',
  orientationAutoHint: 'The map turns by itself when it gains room on screen. No choice is remembered for this map.',
  dimensionGroup: 'Replay view',
  planFeed: 'Elimination feed',
  planFeedHint: 'The last five eliminations, laid over the bottom left of the map.',
} satisfies FramesText
