/**
 * carriedUseSound.ts — L'UTILISATION D'UN ÉQUIPEMENT PORTÉ, entendue à l'instant où le film la date.
 *
 * LA SOURCE est le document (schéma 95) : `carriedEquipment[].uses`, les images des activations de
 * l'objet porté (i21). Chacune est suivie, une ou deux images plus tard, de la baisse d'énergie d'un
 * coût de la famille : c'est le geste qui consomme une charge. La famille est celle du rang de l'objet
 * (`abilityLabels[r].family`).
 *
 * POURQUOI CES QUATRE FAMILLES, ET POURQUOI PAS LEUR POSE. Le capteur, le traqueur, le champ de
 * réparation et l'écran occultant sonnaient sur leur pose `deployed` (`equipmentPlacements`) ; le film
 * ne produit plus de pose `deployed` pour eux depuis la règle du 15/09, et ils étaient muets depuis.
 * Leur utilisation est datée par le film : décision utilisateur du 2026-10-15, les rebrancher dessus,
 * avec le même stem et les mêmes variantes qu'avant. Le mur garde sa pose `deployed`, qui suit son
 * utilisation de deux ou trois images : il n'entre pas ici, sans quoi il sonnerait deux fois.
 *
 * L'INSTANT est l'utilisation lue, sans décalage : aucun de ces quatre sons n'a de montée relevée
 * (contrairement à la pose de la faille, `riftPoseSound.ts`). Catégorie Équipement.
 *
 * LA FIN DU CHAMP DE RÉPARATION (décision utilisateur du 2026-10-15 : « si on a l'événement, on
 * rebranche le son de fin »). Le film l'écrit : la pile i28 de l'objet porté nomme l'objet que
 * l'équipement engendre, et la SUPPRESSION de cet objet est la fin du champ
 * (`carriedEquipment[].tracked[].t1`). Le son d'extinction (`play_007_abl_repairfield_deactivate`)
 * part à cette image ; un champ dont la suppression n'est pas lue ne sonne pas sa fin.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le stem de l'utilisation, par famille de la palette. */
export const CARRIED_USE_SOUND_STEMS: Readonly<Record<string, string>> = {
  sensor: 'sensor_activate',
  // Même appareil que le capteur, un mode près : même geste, donc même son.
  threat_seeker: 'sensor_activate',
  // L'ACTIVATION du champ (`play_007_abl_repairfield_activate`), pas le claquement de sa pose :
  // décision utilisateur du 2026-08-27, détail dans `replaySound.ts`.
  repair_field: 'repair_field_activate',
  // `play_007_abl_shroud_deploy_player`, banque `sb_007_abl_shroud` (nommé le 2026-08-27).
  shroud_screen: 'shroud_deploy',
}

/** carriedUseSoundEvents — un son par utilisation publiée d'une famille de la table, à son image. */
export function carriedUseSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const c of doc.carriedEquipment) {
    const family = doc.abilityLabels?.[String(c.r)]?.family
    const stem = family ? CARRIED_USE_SOUND_STEMS[family] : undefined
    if (!stem) continue
    for (const u of c.uses) out.push(soundEvent(frameToMs(u, doc), stem))
  }
  return out
}

/**
 * Le stem de la FIN d'un objet engendré, par famille : seul le champ de réparation a un son
 * d'extinction propre dans le jeu (décision produit du 2026-08-26, « y compris le champ de
 * réparation qui disparaît »).
 */
export const CARRIED_END_SOUND_STEMS: Readonly<Record<string, string>> = {
  repair_field: 'repair_field_end',
}

/** carriedEndSoundEvents — un son par fin lue d'un objet engendré d'une famille de la table. */
export function carriedEndSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const c of doc.carriedEquipment) {
    const family = doc.abilityLabels?.[String(c.r)]?.family
    const stem = family ? CARRIED_END_SOUND_STEMS[family] : undefined
    if (!stem) continue
    for (const s of c.tracked) if (s.t1 !== undefined) out.push(soundEvent(frameToMs(s.t1, doc), stem))
  }
  return out
}
