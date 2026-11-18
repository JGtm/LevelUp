/**
 * equipmentGestures.ts — LES GESTES DATÉS D'ÉQUIPEMENT que le document publie par vie : les
 * épisodes d'état actif, les poses déployées, les objets lâchés. La lecture de chacun, sa date et
 * l'écriture qu'il fait sur un compteur n'existent qu'ici.
 *
 * DEUX LECTEURS, UNE SEULE RÈGLE : le total du match (`buildEquipmentUsage`) verse tous ces gestes
 * dans le compteur de leur propriétaire ; la lecture à l'instant (`playerEquipmentStats`) les verse
 * dans le même compteur jusqu'à une image. Aucun des deux ne décide seul de ce qui compte pour
 * quoi : ni la famille, ni le filtre de la pose, ni la date.
 *
 * Un geste porte son propriétaire sous la forme `at` (le slot et l'image où le propriétaire de la
 * VIE qui occupe ce slot se résout, cf. `buildSlotOwnership`), ou `null` quand le film ne désigne
 * aucun poseur : le geste est réel, son auteur ne l'est pas.
 *
 * Pur : ni React, ni couleur, ni langue.
 */
import { PLACEMENT_RENDER, placementIsDeployedObject } from '../layers/equipmentPlacementsLayer'
import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import type { EquipmentUsageEpisode, EquipmentUsageTally } from './equipmentUsageLogic'
import { placementIsDroppedPower } from './placementDropped'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer, SlotOwnership } from '../../../lib/replay/rosterLogic'

/** Les deux familles dont l'ÉTAT ACTIF est mesuré (identifiants stables du document). */
export const EPISODE_FAMILIES = [EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD] as const
export type EquipmentEpisodeFamily = (typeof EPISODE_FAMILIES)[number]

/** Où se résout le propriétaire d'un geste : le slot du porteur et l'image du geste. */
export interface GestureAt {
  slot: number
  t: number
}

/** Un geste d'équipement daté (`at.t` est son image), avec la famille que le document lui donne. */
export type EquipmentGesture =
  | { kind: 'episode'; at: GestureAt; family: string; ms: number; kills: number }
  | { kind: 'deployed' | 'dropped'; at: GestureAt | null; family: string }

/** Incrémente une case de table, en la créant au besoin. */
export function bump<K extends string | number>(table: Record<K, number>, key: K, by = 1): void {
  table[key] = (table[key] ?? 0) + by
}

/**
 * LES FAMILLES DONT UN DÉPLOIEMENT EST UN GESTE, et pourquoi la liste n'est pas « toutes ».
 *
 * `PLACEMENT_RENDER` porte déjà la décision, mesurée et écrite : une famille à `null` y est une
 * famille CONNUE qu'on ne dessine pas parce que ce n'est pas un objet posé sur le terrain — les
 * quatre grenades (leurs poses `deployed` sont des LANCERS, déjà comptés par `grenades[]`, les
 * compter deux fois ferait deux colonnes d'un même geste) et les trois capacités `grapple`,
 * `thruster`, `repulsor` (l'appareil lui-même, pas un objet ; le grappin a sa colonne de
 * TRACTIONS ; le répulseur n'a aucun canal d'activation mesuré ; le propulseur en a un depuis
 * le 2026-09-03, schéma 38, mais son geste dure une demi-seconde et se lit sur la CARTE — le
 * dash du pion —, pas dans un compte de tableau : décision utilisateur du même jour).
 *
 * Réutiliser cette table plutôt qu'en écrire une deuxième est la règle des ≤ 2 copies : le jour
 * où une famille change de statut, elle change ici aussi, sans qu'on y pense.
 */
function isDeployableFamily(family: string): boolean {
  return PLACEMENT_RENDER[family] != null
}

/**
 * readEquipmentGestures — les épisodes, les poses déployées et les objets lâchés du document, dans
 * l'ordre de leur canal (épisodes puis poses).
 *
 * `owner` -1 = aucun bipède contemporain à moins de 3 m : la pose est réelle, son auteur ne l'est
 * pas (`at: null`). Une famille d'épisode hors des deux mesurées n'a ni libellé ni sens établi : elle
 * n'entre pas. Une pose qui n'est ni un déploiement d'objet de famille déployable, ni un lâcher de
 * puissance (origine inconnue, grenade, capacité) ne produit aucun geste.
 */
export function readEquipmentGestures(
  doc: Pick<ReplayDocumentReady, 'equipmentEpisodes' | 'equipmentPlacements' | 'frameIntervalMs'>,
): EquipmentGesture[] {
  const out: EquipmentGesture[] = []
  for (const e of doc.equipmentEpisodes) {
    if (!(EPISODE_FAMILIES as readonly string[]).includes(e.fam)) continue
    out.push({
      kind: 'episode',
      at: { slot: e.slot, t: e.t0 },
      family: e.fam,
      ms: frameToMs(e.t1 - e.t0, doc),
      kills: e.k ?? 0,
    })
  }
  for (const p of doc.equipmentPlacements) {
    const at = p.owner >= 0 ? { slot: p.owner, t: p.t0 } : null
    if (placementIsDeployedObject(p) && isDeployableFamily(p.family)) {
      out.push({ kind: 'deployed', at, family: p.family })
    } else if (placementIsDroppedPower(p)) {
      out.push({ kind: 'dropped', at, family: p.family })
    }
  }
  return out
}

/**
 * gestureOwner — LE JOUEUR À QUI REVIENT UN GESTE, la seule résolution de propriétaire de ces canaux : le
 * total du match et la lecture à l'instant la lisent. `null` = aucun propriétaire (pas de poseur mesuré, ou
 * slot sans propriétaire connu) : le geste ne revient à personne, il n'est jamais deviné.
 *
 * Un épisode ou une pose déployée se résout sur la vie qui occupe le slot à l'image du geste. UN LÂCHER SE
 * DATE À LA FIN DE LA VIE DU MORT PLUS UNE IMAGE (ou deux, selon l'échantillonnage) : le poseur n'occupe déjà
 * plus le slot, la résolution stricte à cette image rend le nouvel occupant, ou personne. Il revient donc au
 * joueur qui occupait le slot à l'image PRÉCÉDENTE (la vie qui vient de finir ; une mort à la dernière image
 * d'une manche, dont le slot est réattribué dès l'image suivante, reste au mort), et à défaut au dernier
 * propriétaire connu du slot (`ownerAtFrameOrLast`, borné à la vie qui précède l'image : jamais un occupant
 * plus tardif). Un lâcher à mi-vie (échange) tombe dans la vie du poseur, que les deux lectures rendent.
 */
export function gestureOwner(ownership: SlotOwnership, g: EquipmentGesture): ReplayPlayer | null {
  if (!g.at) return null
  const { slot, t } = g.at
  if (g.kind !== 'dropped') return ownership.ownerAtFrame(slot, t)
  return ownership.ownerAtFrame(slot, t - 1) ?? ownership.ownerAtFrameOrLast(slot, t)
}

/** Ajoute un épisode (un de plus, sa durée, ses frags) à une table par famille. */
function addEpisode(
  table: Record<string, EquipmentUsageEpisode>,
  fam: string,
  ms: number,
  kills: number,
): void {
  const cur = table[fam] ?? { count: 0, ms: 0, kills: 0 }
  table[fam] = { count: cur.count + 1, ms: cur.ms + Math.max(0, ms), kills: cur.kills + kills }
}

/** applyEquipmentGesture verse un geste dans le compteur de son propriétaire : la seule écriture de ces canaux. */
export function applyEquipmentGesture(tally: EquipmentUsageTally, g: EquipmentGesture): void {
  if (g.kind === 'episode') addEpisode(tally.episodes, g.family, g.ms, g.kills)
  else bump(g.kind === 'deployed' ? tally.deployed : tally.dropped, g.family)
}
