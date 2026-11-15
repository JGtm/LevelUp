/**
 * pickupFeedLogic.ts — « ARMES ET BONUS » : les prises sur socle, tirées de `doc.padPickups[]`.
 * Logique pure.
 *
 * UNE LIGNE = UNE PRISE QUE LE SERVEUR A DATÉE (`t`) ET NOMMÉE (`xuid`) — arme ou bonus
 * (surbouclier, camouflage), la jointure par le ramassage natif étant la même pour les deux
 * (cf. `padPickupDating.ts`). Une prise sans `t` n'est connue qu'à l'intervalle des images-clés
 * près (≈ 20 s) et une prise sans `xuid` n'a pas de ramasseur : ni l'une ni l'autre ne fait une
 * ligne, parce que ce fil écrit des instants et des noms, pas des approximations.
 *
 * LE NOM DU SOCLE VIENT DE L'APPELANT (`padNameFor` : table des bonus locale, sinon libellé du
 * document, sinon l'identifiant) — ce fichier ne connaît aucune langue ni aucun catalogue.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import { padFamilyOf, type PadFamily } from './weaponPadFamilies'

/** Rend le nom affichable de ce qu'un socle porte (`weaponPads[].weapon`). */
export type PadNameOf = (weapon: string) => string

/**
 * pickupNatureOf — la nature de ce qu'une prise emporte (socle ou sol), par la même règle que les
 * socles de la carte : la famille d'équipement d'abord, puis la clé canonique du titre lue dans
 * `weaponLabels` (identifiant normalisé), le repli « râtelier » sinon.
 */
export function pickupNatureOf(doc: ReplayDocumentReady, weapon: string): PadFamily {
  return padFamilyOf(weapon, doc.weaponLabels?.[weaponLabelKeyOf(weapon)]?.key)
}

export function pickupFeedEntries(doc: ReplayDocumentReady, nameOf: PadNameOf): ReplayFeedEntry[] {
  const entries: ReplayFeedEntry[] = []
  doc.padPickups.forEach((p, i) => {
    if (p.t === undefined || p.t === null || !p.xuid) return
    const pad = doc.weaponPads[p.pad]
    if (!pad) return
    entries.push(
      eventFeedEntry(`u-${p.xuid}-${p.pad}-${p.t}-${i}`, frameToMs(p.t, doc), {
        family: 'pickups',
        xuid: p.xuid,
        name: nameOf(pad.weapon),
        nature: pickupNatureOf(doc, pad.weapon),
      }),
    )
  })
  entries.sort((a, b) => a.replayMs - b.replayMs)
  return entries
}
