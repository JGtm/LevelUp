/**
 * groundPickupFeedLogic.ts — « ARMES ET BONUS » : les armes RAMASSÉES AU SOL, tirées de
 * `doc.groundWeapons[]`. Logique pure. Elles complètent les prises sur socle (`pickupFeedLogic`).
 *
 * UNE LIGNE = UN RAMASSAGE QUE LE DOCUMENT DATE, NOMME ET IDENTIFIE, D'UNE ARME QUI N'EST PAS DE
 * BASE :
 *  - DATÉ : une arme au sol dont la fin est `pickup` porte l'instant de sa reprise dans `t1`, exact
 *    (flux delta, `t1max == t1`). Les fins `seen` (disparition bornée par un intervalle) et `open`
 *    (jamais reprise) ne sont pas des ramassages datés : aucune ligne ;
 *  - NOMMÉ : `picker` est le slot de la vie ramasseuse ; son xuid est celui de la vie de ce slot
 *    qui couvre l'instant. Un slot sans vie à cet instant ou une vie sans xuid (bot) n'a pas de
 *    ramasseur à écrire : aucune ligne, jamais un nom deviné ;
 *  - IDENTIFIÉ : le libellé du document nomme la famille. Une famille inconnue ne s'écrit pas
 *    sous son identifiant brut ici : aucune ligne ;
 *  - PAS DE BASE : la règle est celle du réglage « armes de départ au sol » du calque — une arme
 *    est de base quand son modèle figure dans une dotation de naissance du match
 *    (`startingWeaponKeys`), aucune table d'armes en dur. Sans dotation de naissance, ou en
 *    départs aléatoires (`weaponTiers.randomStarts` : le niveau de base n'est pas publié), la
 *    base n'est pas établie : aucune ligne plutôt qu'un classement deviné.
 *
 * PAS DE DOUBLON. Les armes de socle ne sont pas dans `groundWeapons` (le document les publie
 * ailleurs), mais une ligne qui coïncide avec une prise de socle du même joueur, de la même
 * arme, à moins de `SAME_PICKUP_FRAMES` images, serait la même prise écrite deux fois : elle
 * est écartée au profit de la ligne du socle.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs, trackWindow } from '../../../lib/replay/replayLogic'
import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import type { PadNameOf } from './pickupFeedLogic'
import { startingWeaponKeys } from './startingWeaponsOnGround'

/** La fin d'une arme au sol qui porte un ramassage daté (`ReplayGroundWeapon.end`). */
const END_PICKUP = 'pickup'

/**
 * La fenêtre qui décide qu'une ligne du sol et une prise de socle sont LA MÊME prise : cinq images,
 * soit 500 ms sur la grille de 100 ms — la tolérance d'appariement des deux canaux de ramassage
 * (cf. `sound/weaponChangeSound.ts`).
 */
const SAME_PICKUP_FRAMES = 5

/** Le xuid de la vie du slot qui couvre l'image, ou null (aucune vie, ou vie sans xuid). */
function xuidOfSlotAt(doc: ReplayDocumentReady, slot: number, frame: number): string | null {
  for (const track of doc.tracks) {
    if (track.slot !== slot) continue
    const w = trackWindow(track)
    if (frame < w.start || frame > w.end) continue
    return track.xuid || null
  }
  return null
}

/** Faux quand la base n'est pas établie : aucune dotation de naissance, ou départs aléatoires. */
function baseIsEstablished(doc: ReplayDocumentReady, starting: ReadonlySet<string>): boolean {
  return starting.size > 0 && !doc.weaponTiers?.randomStarts
}

/**
 * groundPickupFeedEntries — une ligne par arme non de base ramassée au sol, à son instant sur l'axe
 * du rejeu (`frameToMs`), triée. `padLines` : les lignes de prises de socle déjà bâties, contre
 * lesquelles les doublons sont écartés.
 */
export function groundPickupFeedEntries(
  doc: ReplayDocumentReady,
  nameOf: PadNameOf,
  padLines: readonly ReplayFeedEntry[],
): ReplayFeedEntry[] {
  const starting = startingWeaponKeys(doc.loadouts)
  if (!baseIsEstablished(doc, starting)) return []
  const windowMs = frameToMs(SAME_PICKUP_FRAMES, doc)
  const entries: ReplayFeedEntry[] = []
  doc.groundWeapons.forEach((g, i) => {
    if (g.end !== END_PICKUP || g.picker < 0) return
    const key = weaponLabelKeyOf(g.w)
    if (starting.has(key) || !doc.weaponLabels?.[key]) return
    const xuid = xuidOfSlotAt(doc, g.picker, g.t1)
    if (!xuid) return
    const name = nameOf(g.w)
    const replayMs = frameToMs(g.t1, doc)
    const duplicate = padLines.some(
      (p) =>
        p.event?.family === 'pickups' &&
        p.event.xuid === xuid &&
        p.event.name === name &&
        Math.abs(p.replayMs - replayMs) <= windowMs,
    )
    if (duplicate) return
    entries.push(
      eventFeedEntry(`g-${xuid}-${g.w}-${g.t1}-${i}`, replayMs, { family: 'pickups', xuid, name, ground: true }),
    )
  })
  entries.sort((a, b) => a.replayMs - b.replayMs)
  return entries
}
