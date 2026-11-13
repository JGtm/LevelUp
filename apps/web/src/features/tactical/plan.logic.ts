/**
 * plan.logic — la logique PURE de la carte du plan : l'aide ⓘ du titre, les bornes et le dégradé
 * de la légende verticale, la boîte du fond, et ce qui se pose sur le fond selon la carte affichée
 * et l'état de sa lecture.
 *
 * LA RAMPE DE LÉGENDE VIENT DE LA RAMPE PEINTE (`heatRamp` / `heatRampDivergent`, déjà résolues pour
 * le calque) : elle en suit les couleurs ET l'opacité, sans seconde source. Verticale, le bas de la
 * rampe (valeur basse, ou négative sur une lecture signée) en bas.
 */
import type { EchelleTactique, TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { QuarterTurns } from '@/lib/replay/orientation'

import { LEGENDE_MARGE_PX, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import type { TacticalEtatLecture } from './tacticalLecture.logic'
import {
  lectureDeRejeu,
  libelleRayons,
  planLegend,
  repereAspect,
  TACTICAL_CELL_FLOOR,
  type FenetreDuPlan,
  type TacticalQuestion,
} from './tacticalView.logic'

/** La marge réservée à la légende verticale, à droite du fond. */
export const MARGE_LEGENDE = `${LEGENDE_MARGE_PX}px`

/** Nombre d'arrêts prélevés dans la rampe peinte pour le dégradé de la légende. */
const ARRETS_DE_LEGENDE = 9

/**
 * infoDuPlan — l'aide ⓘ du titre : matchs mesurés sur filtrés, source, pas de la grille et plancher
 * par zone ; les deux dénominateurs de « victoires − défaites » ; la règle de « morts seul » avec la
 * ou les portées du radar, les matchs sans portée et les morts écartées ; les zones servies hors du
 * cadre du fond (`peintes` = celles posées sur le fond).
 */
export function infoDuPlan(
  t: TacticalText,
  locale: Locale,
  question: TacticalQuestion,
  lecture: TacticalRaster,
  peintes: number,
): string {
  const source = lectureDeRejeu(question) ? t.planInfoSourceReplay : t.planInfoSourceJournal
  const parts = [t.planInfo(lecture.matchs_retenus, source, lecture.pas_m, TACTICAL_CELL_FLOOR)]
  if (question === 'gagne') {
    parts.push(t.planInfoGagne(lecture.matchs_victoire, lecture.matchs_defaite, TACTICAL_CELL_FLOOR))
  }
  if (question === 'isole') {
    const rayons = lecture.rayons_radar_m ?? []
    if (rayons.length > 0) parts.push(t.planInfoIsole(libelleRayons(t, rayons, locale)))
    if ((lecture.morts_equipe_a_terre ?? 0) > 0) parts.push(t.planInfoTeamDown(lecture.morts_equipe_a_terre ?? 0))
  }
  const servies = (lecture.cellules ?? []).length
  if (servies > peintes) parts.push(t.planInfoOffFrame(servies - peintes, servies))
  return parts.join(' · ')
}

/** Les bornes de la légende verticale : la basse, la haute (nombre et unité à part), le mode de rampe. */
export interface BornesDeLegende {
  lo: string
  hiNombre: string
  unite: string
  /** La borne haute avec son unité, telle que la nomme l'échelle accessible. */
  hi: string
  mode: 'intensity' | 'divergent'
}

/** legendeDuPlan — les bornes de `planLegend`, la borne haute séparée de son unité. */
export function legendeDuPlan(
  echelle: EchelleTactique,
  unite: string,
  format: (n: number) => string,
): BornesDeLegende {
  const { lo, hi, mode } = planLegend(echelle, unite, format)
  const hiNombre = mode === 'divergent' ? `+ ${format(Math.abs(echelle.borne))}` : format(echelle.p95)
  return { lo, hiNombre, unite, hi, mode }
}

/** rampeVerticale — le dégradé CSS de la légende, prélevé dans la rampe peinte, du bas vers le haut. */
export function rampeVerticale(ramp: readonly string[]): string {
  if (ramp.length === 0) return 'none'
  const arrets: string[] = []
  for (let i = 0; i < ARRETS_DE_LEGENDE; i += 1) {
    const part = i / (ARRETS_DE_LEGENDE - 1)
    const couleur = ramp[Math.round(part * (ramp.length - 1))]
    arrets.push(`${couleur} ${Math.round(part * 10000) / 100}%`)
  }
  return `linear-gradient(0deg, ${arrets.join(', ')})`
}

/**
 * boiteDuPlan — le cadre du fond : toute la largeur disponible (la marge de la légende déjà
 * retirée par son conteneur), au rapport du fond, sans dépasser `hauteurMax` de haut (l'espace que
 * la fenêtre laisse au plan). LE RAPPORT N'EST JAMAIS DÉFORMÉ : la borne de hauteur passe par une
 * largeur maximale.
 */
export function boiteDuPlan(
  aspect: number,
  hauteurMax: number,
): { aspectRatio: number; width: string; maxWidth: string } {
  return { aspectRatio: aspect, width: '100%', maxWidth: `${Math.round(aspect * hauteurMax)}px` }
}

/** La place de l'image du fond dans le cadre, en pourcentages du cadre (et sa rotation, carte tournée). */
export interface CadrageDuFond {
  left: string
  top: string
  width: string
  height: string
  /** `rotate(<turns x 90>deg)` autour du centre de l'image : absent à 0 quart de tour. */
  transform?: string
}

/**
 * cadrageDuFond — la place de l'image du fond dans le cadre, quand le plan est grossi : l'image
 * couvre la SCÈNE (le calage du fond), le cadre montre la FENÊTRE ; l'image est donc agrandie du
 * rapport scène / fenêtre et décalée du coin de la fenêtre, en pourcentages du cadre. À 1x : le
 * cadre entier, exactement l'image d'avant le zoom. Le calque se projette sur la même fenêtre
 * (`vueDuPlan`), les deux ne peuvent donc pas se désaligner.
 *
 * LA CARTE TOURNÉE (`turns` quarts de tour horaires, l'orientation retenue pour la carte) : `scene`
 * et `fenetre` sont ceux de la carte APRÈS rotation, et le rectangle calculé est celui de l'image
 * tournée. L'`<img>` garde ses dimensions d'origine — échangées pour un quart ou trois quarts de
 * tour —, posées autour du MÊME centre, puis tournées par CSS autour de ce centre : le fond et le
 * calque (projection tournée) coïncident exactement. Le cadre prend le rapport de la scène tournée,
 * d'où le passage des pourcentages de hauteur aux pourcentages de largeur par ce rapport.
 */
export function cadrageDuFond(scene: FenetreDuPlan, fenetre: FenetreDuPlan, turns: QuarterTurns = 0): CadrageDuFond {
  const lf = fenetre.maxX - fenetre.minX
  const hf = fenetre.maxY - fenetre.minY
  if (!(lf > 0) || !(hf > 0)) return { left: '0%', top: '0%', width: '100%', height: '100%' }
  const pct = (v: number) => `${Math.round(v * 10000) / 100}%`
  const left = (scene.minX - fenetre.minX) / lf
  const top = (fenetre.maxY - scene.maxY) / hf
  const width = (scene.maxX - scene.minX) / lf
  const height = (scene.maxY - scene.minY) / hf
  if (turns === 0) return { left: pct(left), top: pct(top), width: pct(width), height: pct(height) }
  const aspect = repereAspect(fenetre)
  // Dimensions de l'image avant rotation, en fraction du cadre : échangées pour un nombre impair de quarts.
  const w0 = turns % 2 === 1 ? height / aspect : width
  const h0 = turns % 2 === 1 ? width * aspect : height
  return {
    left: pct(left + width / 2 - w0 / 2),
    top: pct(top + height / 2 - h0 / 2),
    width: pct(w0),
    height: pct(h0),
    transform: `rotate(${turns * 90}deg)`,
  }
}

/** Ce qui se pose sur le fond : l'état de la lecture, ou ce que dit la carte affichée. */
export type EtatDuPlan = TacticalEtatLecture | 'hors_filtre' | 'sans_carte'

/**
 * etatDuPlan — l'échec (lecture, périmètre, composition impossible) prime ; puis une carte d'URL hors
 * du filtre ou sous le plancher (la dire, sans lecture), aucune carte ouvrable (rien sur le fond :
 * la colonne des cartes le dit) ; sinon l'état de la lecture.
 */
export function etatDuPlan(origine: CarteEffective['origine'], etat: TacticalEtatLecture): EtatDuPlan {
  if (etat === 'echec') return 'echec'
  if (origine === 'hors_filtre') return 'hors_filtre'
  if (origine === 'aucune') return 'sans_carte'
  return etat
}
