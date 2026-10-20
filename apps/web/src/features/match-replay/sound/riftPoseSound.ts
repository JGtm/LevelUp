/**
 * riftPoseSound.ts — LA POSE DE LA FAILLE DU TRANSLOCATEUR, entendue pour que son COUP tombe à la pose.
 *
 * LA SOURCE est le document (schéma 95) : `riftPoses`, une entrée par pose, datée par le film —
 * l'énergie du translocateur porté tombe à zéro — avec l'activation qui la précède (`a`, ~1,63 s plus
 * tôt sur tout le parc).
 *
 * LE SON (`translocator_deploy`) est le geste tel que le jeu le joue, recomposé en cinq couches natives
 * de la banque de l'appareil (`sb_007_abl_quantum`) sur l'enregistrement en jeu de l'utilisateur :
 * charge `55f75682`, foley `9520fe71`, COUP de pose `0431c88f`, foley `a9c26b1b`, boucle de la faille
 * `388207de` (coupée en fondu) ; 48 kHz, crête plafonnée à -1 dBTP. Le coup est à 1,045 s du début du
 * fichier : le son part 1,045 s avant la pose, jamais avant l'activation qui l'a commandée.
 * Catégorie Équipement.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le stem de la pose de la faille. */
export const RIFT_POSE_SOUND_STEM = 'translocator_deploy'

/** Position du COUP de pose dans le fichier, en millisecondes (relevée sur l'enregistrement en jeu). */
export const RIFT_POSE_HIT_OFFSET_MS = 1045

/** riftPoseSoundEvents — un son par pose publiée, son coup à l'instant de la pose. */
export function riftPoseSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  return doc.riftPoses.map((p) => {
    let depart = frameToMs(p.t, doc) - RIFT_POSE_HIT_OFFSET_MS
    if (p.a !== undefined) depart = Math.max(depart, frameToMs(p.a, doc))
    return soundEvent(Math.max(0, depart), RIFT_POSE_SOUND_STEM)
  })
}
