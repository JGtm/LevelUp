/**
 * translocationCuts — UNE TRANSLOCATION EST UNE COUPURE DE LA PISTE, PAS UN DÉPLACEMENT.
 *
 * LE DÉFAUT CONSTATÉ À L'ÉCRAN (aj14-b, B0, film `1b2d9e08`, lecture à 1×). La piste d'une vie
 * est échantillonnée à chaque image ; l'échantillon d'ARRIVÉE d'une translocation se trouve à
 * 3 à 20 m du précédent. Interpolé comme un pas ordinaire, il donnait deux défauts : le pion
 * GLISSAIT du départ à l'arrivée pendant une image, et la traînée traçait un trait droit entre
 * les deux points, tenu pendant toute sa fenêtre — un déplacement que personne n'a fait.
 *
 * CE QUE LE MODULE DIT. `translocations[]` (l'événement 117 du film) porte les deux positions
 * du saut (`fx/fy` -> `tx/ty`). La coupure est posée sur le premier échantillon d'arrivée : celui
 * dont le PRÉCÉDENT est au départ et dont lui-même est à l'arrivée. `replayLogic` la lit comme une
 * lacune de réplication pour la position, l'altitude et la traînée (rien ne s'interpole au
 * travers) mais pas pour le dessin (`inGapAt` ne la voit pas : le joueur est là, il n'est pas
 * pâli).
 *
 * POURQUOI ON CHERCHE LE COUPLE D'ÉCHANTILLONS ET ON NE LE DEVINE PAS. L'image de l'événement
 * est tantôt la dernière image au départ, tantôt la première à l'arrivée (mesuré sur le parc du
 * 2026-10-14 : 30 événements sur 122 dans le premier cas, 92 dans le second). Deux candidats
 * seulement sont essayés — le premier échantillon à l'image de l'événement ou après, et le
 * suivant —, chacun vérifié par les DEUX positions publiées. Aucun seuil de distance de saut :
 * une translocation peut faire 50 cm. Un événement sans positions, ou dont aucun couple ne
 * correspond (1 sur 123 au parc : la piste s'arrête avant), ne coupe rien — on n'invente pas
 * de coupure.
 */
import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayCutPoint } from './replayReadyTypes'

/**
 * Écart maximal, en mètres, entre une extrémité publiée du saut et l'échantillon qui la porte.
 * Mesuré sur les 122 événements appariés du parc : 0,00 à 0,27 m au départ, 0,00 à 1,39 m à
 * l'arrivée (le joueur bouge pendant l'image) ; 2 m laisse la marge d'un échantillon décalé. Un
 * saut très court (0,48 m au plus petit) peut apparier deux couples voisins : la coupure tombe
 * alors entre deux échantillons presque confondus, et ne déplace rien.
 */
export const CUT_TOLERANCE_M = 2

type Translocation = NonNullable<ReplayDocument['translocations']>[number]

/**
 * cutSampleIndex — l'index de l'échantillon d'arrivée de CETTE translocation dans CETTE piste,
 * ou -1 quand elle n'y est pas lisible.
 */
export function cutSampleIndex(points: readonly ReplayCutPoint[], ev: Translocation): number {
  const { fx, fy, tx, ty } = ev
  if (fx === undefined || fy === undefined || tx === undefined || ty === undefined) return -1
  let k0 = -1
  for (let i = 0; i < points.length; i++) {
    if (points[i].t >= ev.t) {
      k0 = i
      break
    }
  }
  if (k0 < 0) return -1
  for (const k of [k0, k0 + 1]) {
    if (k <= 0 || k >= points.length) continue
    const a = points[k - 1]
    const b = points[k]
    if (Math.hypot(a.x - fx, a.y - fy) <= CUT_TOLERANCE_M && Math.hypot(b.x - tx, b.y - ty) <= CUT_TOLERANCE_M) {
      return k
    }
  }
  return -1
}

/**
 * markTranslocationCuts — pose `cut` sur l'échantillon d'arrivée de chaque translocation.
 *
 * Une vie par piste : un slot peut en porter plusieurs (réapparitions), on cherche donc parmi
 * celles du slot. Les pistes sans coupure sont rendues TELLES QUELLES (même référence), les
 * autres reçoivent un tableau neuf — les échantillons du contrat ne sont jamais mutés.
 */
export function markTranslocationCuts<T extends { slot: number; points: ReplayCutPoint[] }>(
  tracks: readonly T[],
  translocations: readonly Translocation[],
): T[] {
  if (translocations.length === 0) return [...tracks]
  const cuts = new Map<T, Set<number>>()
  for (const ev of translocations) {
    for (const track of tracks) {
      if (track.slot !== ev.slot) continue
      const k = cutSampleIndex(track.points, ev)
      if (k < 0) continue
      const set = cuts.get(track) ?? new Set<number>()
      set.add(k)
      cuts.set(track, set)
      break
    }
  }
  return tracks.map((track) => {
    const set = cuts.get(track)
    if (!set) return track
    return { ...track, points: track.points.map((p, i) => (set.has(i) ? { ...p, cut: true as const } : p)) }
  })
}
