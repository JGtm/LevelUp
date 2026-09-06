/**
 * equipmentChangeSound.ts — LE RAMASSAGE D'ÉQUIPEMENT, daté par le film.
 *
 * Le son est identifié depuis le 2026-08-26 et confirmé à l'oreille par l'utilisateur
 * (« c'est bien pour le ramassage des armes sur socles de power up ») :
 * `play_007_abl_shared_pickup` (`c73036e4`, banque `sb_007_abl_shared`, 0,80 s). Ce qui
 * manquait était l'INSTANT — et le schéma 26 le publie : `doc.equipmentChanges` porte les
 * ramassages (`taken`) et les consommations (`spent`), datés à la frame, par vie. Rien à
 * déduire, on joue ce que le calque date.
 *
 * ## Le `spent` est MUET, et c'est un choix écrit
 *
 * Consommer un équipement, c'est l'UTILISER — et l'utilisation sonne déjà par sa propre
 * famille : le camouflage et le surbouclier par leurs épisodes (`equipmentEpisodes`), le mur
 * et le capteur par leur pose (`equipmentPlacements`), le grappin par sa traction
 * (`grappleLines`), le répulseur par son kill. Un jingle générique de « consommation »
 * DOUBLERAIT ces sons-là au même instant. Les familles dont l'usage n'est pas encore mesuré
 * restent muettes — même règle que partout : on ne devine pas.
 *
 * Le `spent` a servi d'éclat de fiche au TRANSLOCATEUR du 2026-09-02 au 2026-09-03 ; il n'est
 * plus, depuis le schéma 38, que le REPLI des artefacts anciens — l'usage est daté par
 * l'événement du film (`translocations[]`, cf. `placementTeleport.ts`). Dans les deux cas c'est
 * un effet VISUEL, pas un son : le muet de ce fichier ne concerne que l'audio, et il tient —
 * aucun stem n'est désigné pour la translocation (même règle que le crâne).
 */
import type { ReplayDocumentReady } from '../model/replayNormalize'
import { frameToMs } from '../model/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/**
 * Le son du ramassage d'équipement. Banque `sb_007_abl_shared`, événement `c73036e4` =
 * `play_007_abl_shared_pickup` — nom cassé par hachage ET confirmé à l'oreille.
 */
export const EQUIPMENT_PICKUP_SOUND_STEM = 'objective_pad_pickup'

/** equipmentChangeSoundEvents — un son par ramassage publié, à sa frame. */
export function equipmentChangeSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const c of doc.equipmentChanges) {
    if (c.kind !== 'taken') continue
    out.push(soundEvent(frameToMs(c.t, doc), EQUIPMENT_PICKUP_SOUND_STEM))
  }
  return out
}
