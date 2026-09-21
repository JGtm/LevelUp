/**
 * stanceLogic.ts — L'ÉTAT DE MOUVEMENT D'UNE VIE À UNE IMAGE (schéma 65).
 *
 * LA SOURCE est le document : `stances[]`, un intervalle par (vie, genre) sur l'axe de frames.
 * Le film écrit ces états À L'INSTANT — un delta ne porte le composant que quand l'état CHANGE —
 * et le Go a replié ces transitions en intervalles bornés aux vies publiées. Ici on ne fait que
 * LIRE : quel genre couvre cette image, pour cette vie.
 *
 * TROIS GENRES, ET TROIS SEULEMENT : `crouch`, `slide`, `mobility`. Le SPRINT est réfuté comme
 * observable par la vitesse et le SAUT est lu mais pas prouvé (lot 5.3.5) — un quatrième genre
 * qui apparaîtrait dans un artefact futur serait une DONNÉE NEUVE, et ce module rend alors
 * `null` plutôt que le libellé d'un voisin.
 *
 * L'ORDRE DE PRIORITÉ EST ÉCRIT, et il n'est pas arbitraire : deux genres peuvent couvrir la
 * même image (glisser en étant accroupi est une transition de plus, pas une exclusion). La fiche
 * n'a qu'une ligne — elle montre donc le geste le plus SPÉCIFIQUE : une glissade dit plus
 * qu'un accroupissement, et une action de mobilité plus qu'une posture.
 */
import type { ReplayDocumentReady } from '@/lib/replay/replayReadyTypes'

/** Les trois genres publiés, dans l'ordre de PRIORITÉ d'affichage (le plus spécifique d'abord). */
export const STANCE_KINDS = ['mobility', 'slide', 'crouch'] as const

/** Le genre d'un état de mouvement, tel que le document l'écrit. */
export type StanceKind = (typeof STANCE_KINDS)[number]

/**
 * stanceAt rend le genre d'état qui couvre `frame` pour la vie `slot`, ou `null`.
 *
 * `null` veut dire, et seulement : aucun intervalle publié ne couvre cette image. Ce n'est PAS
 * « le joueur est debout et immobile » — un artefact antérieur au schéma 65 n'a pas de `stances`
 * du tout, et `coverage.stances` est le seul endroit qui dise si la marche a tourné.
 */
export function stanceAt(doc: ReplayDocumentReady, slot: number, frame: number): StanceKind | null {
  let trouve: StanceKind | null = null
  let rang: number = STANCE_KINDS.length
  for (const s of doc.stances) {
    if (s.slot !== slot || frame < s.t0 || frame > s.t1) continue
    const i = (STANCE_KINDS as readonly string[]).indexOf(s.kind)
    // UN GENRE INCONNU EST IGNORÉ, jamais rendu : le libellé d'un voisin serait un mensonge.
    if (i < 0 || i >= rang) continue
    rang = i
    trouve = STANCE_KINDS[i]
    if (rang === 0) break // le plus spécifique : rien ne peut le supplanter
  }
  return trouve
}
