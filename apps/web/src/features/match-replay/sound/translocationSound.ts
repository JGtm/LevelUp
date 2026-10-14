/**
 * translocationSound.ts — LA TÉLÉPORTATION DU TRANSLOCATEUR, entendue à son départ.
 *
 * LA SOURCE est le document (schéma 38) : `translocations`, une entrée par saut, datée par
 * l'événement du film `EquipmentTranslocatorTeleportEffects` (type 117), qui précède le saut de 5 à
 * 80 ms. Le son part à cet instant : c'est le DÉPART du geste.
 *
 * LE SON (`translocator_teleport`) est le geste tel que le jeu le joue, recomposé de la banque de
 * l'appareil (`sb_007_abl_quantum`) : l'événement de départ du porteur `10f62776`
 * (`play_007_abl_quantum_teleport_player_start`, `.wem` 123675313) puis, 0,20 s plus tard, l'effet
 * `efe84c70` (`.wem` 596614165), délais et gains relevés sur une capture du jeu ; 48 kHz stéréo,
 * crête plafonnée à -1 dBTP. Catégorie Équipement.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le stem de la téléportation. */
export const TRANSLOCATION_SOUND_STEM = 'translocator_teleport'

/** translocationSoundEvents — un son par téléportation publiée, à sa frame. */
export function translocationSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  return doc.translocations.map((t) => soundEvent(frameToMs(t.t, doc), TRANSLOCATION_SOUND_STEM))
}
