/**
 * vueDeMort — la vue D'UNE MORT ouverte par la sélection d'une tuile : l'instantané (A, B) et, quand la
 * séquence est servie, ses étapes (D). Partagée par le plan (scène, légende, barre d'étapes) et la
 * colonne de droite (fiche de la mort).
 */
import type { TacticalContribution, TacticalSequence } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'

import { metresEntiers, sceneDeMort, sceneDeSequence, sceneDeZone, type SceneDuPlan } from './mort.logic'

export interface VueDeMort {
  /** L'index de la contribution choisie dans la liste servie. */
  index: number
  contribution: TacticalContribution
  /** La séquence, quand la requête a réussi ; `null` sinon (la vue D n'est alors pas offerte). */
  sequence: TacticalSequence | null
  /** L'index de l'étape affichée, `null` = l'instantané de la mort. */
  etape: number | null
  onEtape: (etape: number | null) => void
}

/**
 * sceneDeLaVue — ce que le plan peint : l'étape de la séquence si elle est choisie, sinon l'instantané
 * de la mort choisie, sinon (aucune tuile choisie) le nuage des coéquipiers de la zone.
 */
export function sceneDeLaVue(contributions: readonly TacticalContribution[], vue: VueDeMort | null): SceneDuPlan {
  if (!vue) return sceneDeZone(contributions)
  if (vue.sequence && vue.etape !== null) return sceneDeSequence(vue.sequence, vue.etape)
  return sceneDeMort(vue.contribution)
}

/** Le mode de la légende : le nuage de la zone, l'instantané d'une mort, ou une étape de la séquence. */
export type ModeDeLaScene = 'zone' | 'instantane' | 'sequence' | 'aucun'

export function modeDeLaScene(scene: SceneDuPlan, vue: VueDeMort | null): ModeDeLaScene {
  if (scene.marques.length === 0 && scene.cercle === null) return 'aucun'
  if (!vue) return 'zone'
  return vue.sequence && vue.etape !== null ? 'sequence' : 'instantane'
}

/**
 * sequenceDemandee — le corps de la requête de séquence pour la contribution choisie, `null` quand elle
 * n'en a pas (ni mort, ni rejeu servi pour ce match, ni victime nommée).
 */
export function sequenceDemandee(
  c: TacticalContribution | undefined,
): { match_id: string; victim_xuid: string; time_ms: number } | null {
  if (!c || c.face !== 'mort' || !c.replay_available || !c.xuid) return null
  return { match_id: c.match_id, victim_xuid: c.xuid, time_ms: c.instant_ms }
}
/** La contribution choisie, s'il s'agit d'une mort ou d'un frag (les autres faces n'ont pas de vue). */
export function contributionDeLaVue(
  contributions: readonly TacticalContribution[],
  index: number | null,
): TacticalContribution | undefined {
  if (index === null) return undefined
  const c = contributions[index]
  return c && (c.face === 'mort' || c.face === 'frag') ? c : undefined
}

/** Une entrée de légende : sa forme, par le mot. */
export type FormeDeLegende = 'mort' | 'tueur' | 'plein' | 'creux' | 'nuage' | 'cercle' | 'trait'

/** La légende d'un mode : une entrée par forme présente dans la scène. */
export function entreesDeLegende(
  t: TacticalText,
  mode: ReturnType<typeof modeDeLaScene>,
  scene: SceneDuPlan,
  avecTrait: boolean,
  locale: Locale,
): { forme: FormeDeLegende; texte: string }[] {
  const L = t.mortLegend
  const rayon = scene.cercle ? L.radius(metresEntiers(scene.cercle.rayonM, locale)) : null
  const aDesCreux = scene.marques.some((m) => m.role === 'coequipier' && !m.plein)
  const aDesPleins = scene.marques.some((m) => m.role === 'coequipier' && m.plein)
  switch (mode) {
    case 'zone':
      return [{ forme: 'nuage', texte: L.cloud }]
    case 'instantane':
      return [
        { forme: 'mort', texte: L.death },
        ...(aDesPleins ? [{ forme: 'plein' as const, texte: L.allyIn }] : []),
        ...(aDesCreux ? [{ forme: 'creux' as const, texte: L.allyOut }] : []),
        ...(avecTrait ? [{ forme: 'trait' as const, texte: L.riposte }] : []),
        ...(rayon ? [{ forme: 'cercle' as const, texte: rayon }] : []),
      ]
    case 'sequence':
      return [
        { forme: 'mort', texte: L.death },
        { forme: 'tueur', texte: L.killer },
        { forme: 'plein', texte: t.mortDetailUnnamed },
        ...(rayon ? [{ forme: 'cercle' as const, texte: rayon }] : []),
      ]
    default:
      return []
  }
}

