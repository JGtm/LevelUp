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

import { LEGENDE_MARGE_PX, PLAN_BOITE_HAUTEUR_MAX_PX, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import type { TacticalEtatLecture } from './tacticalLecture.logic'
import {
  lectureDeRejeu,
  libelleRayons,
  planLegend,
  TACTICAL_CELL_FLOOR,
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
  const parts = [t.planInfo(lecture.matchs_retenus, lecture.matchs_filtres, source, lecture.pas_m, TACTICAL_CELL_FLOOR)]
  if (question === 'gagne') {
    parts.push(t.planInfoGagne(lecture.matchs_victoire, lecture.matchs_defaite, TACTICAL_CELL_FLOOR))
  }
  if (question === 'isole') {
    const rayons = lecture.rayons_radar_m ?? []
    if (rayons.length > 0) parts.push(t.planInfoIsole(libelleRayons(t, rayons, locale)))
    if ((lecture.matchs_sans_rayon ?? 0) > 0) parts.push(t.planInfoNoRange(lecture.matchs_sans_rayon ?? 0))
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
 * retirée par son conteneur), au rapport du fond, sans dépasser `PLAN_BOITE_HAUTEUR_MAX_PX` de haut.
 * LE RAPPORT N'EST JAMAIS DÉFORMÉ : la borne de hauteur passe par une largeur maximale.
 */
export function boiteDuPlan(aspect: number): { aspectRatio: number; width: string; maxWidth: string } {
  return { aspectRatio: aspect, width: '100%', maxWidth: `${aspect * PLAN_BOITE_HAUTEUR_MAX_PX}px` }
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
