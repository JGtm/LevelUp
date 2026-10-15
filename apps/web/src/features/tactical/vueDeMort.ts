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
export function sceneDeLaVue(
  contributions: readonly TacticalContribution[],
  vue: VueDeMort | null,
): SceneDuPlan {
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
export type FormeDeLegende = 'mort' | 'tueur' | 'victime' | 'plein' | 'creux' | 'nuage' | 'cercle'

/**
 * entreesDeLegende — LA légende du plan, une seule : une forme = une entrée, dans l'ordre Mort · Tueurs /
 * Victimes · Coéquipiers (dans le rayon, hors du rayon, ou le nuage de la zone) · Rayon du radar. `roles` sont les rôles que la
 * couche des tueurs et victimes peint ; pendant
 * une étape de la séquence cette couche se retire et le tueur est celui de la scène.
 */
export function entreesDeLegende(
  t: TacticalText,
  mode: ModeDeLaScene,
  scene: SceneDuPlan,
  locale: Locale,
  roles: readonly ('tueur' | 'victime')[],
): { forme: FormeDeLegende; texte: string }[] {
  const L = t.mortLegend
  const sceneMort = mode === 'instantane' || mode === 'sequence'
  const sorties: { forme: FormeDeLegende; texte: string }[] = []
  if (sceneMort) sorties.push({ forme: 'mort', texte: L.death })
  if (roles.includes('tueur')) sorties.push({ forme: 'tueur', texte: t.layerKillers })
  else if (mode === 'sequence') sorties.push({ forme: 'tueur', texte: L.killer })
  if (roles.includes('victime')) sorties.push({ forme: 'victime', texte: t.layerVictims })
  if (mode === 'zone') sorties.push({ forme: 'nuage', texte: L.cloud })
  if (sceneMort) {
    if (scene.marques.some((m) => m.role === 'coequipier' && m.plein)) sorties.push({ forme: 'plein', texte: L.allyIn })
    if (scene.marques.some((m) => m.role === 'coequipier' && !m.plein)) sorties.push({ forme: 'creux', texte: L.allyOut })
    if (scene.cercle) sorties.push({ forme: 'cercle', texte: L.radius(metresEntiers(scene.cercle.rayonM, locale)) })
  }
  return sorties
}
