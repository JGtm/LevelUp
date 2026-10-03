/**
 * padSpawnSound.ts — L'APPARITION D'UNE ARME SUR SON SOCLE, datée à la frame.
 *
 * ## Pourquoi ce fichier existe, et pourquoi son voisin n'existe plus
 *
 * Le socle PUBLIE ses instants d'apparition (`WeaponPad.spawns`, en frames, sur l'axe de
 * `Point.T`) : il n'y a ni joueur ni déduction, on joue ce que le calque date.
 *
 * SON SYMÉTRIQUE, LE RAMASSAGE, A ÉTÉ RETIRÉ LE 2026-08-30 (`padSound.ts`, supprimé) et c'est
 * la même frontière qui l'explique. Aucun canal ne date le ramassage — `padPickups` donne un
 * intervalle dont la médiane est 20,00 s, et les loadouts vivent sur la même grille
 * d'images-clés. La règle d'alors le déduisait du premier TIR d'une famille d'arme de socle :
 * mesures justes, conclusion fausse. La doctrine de cette chaîne est que le rejeu SE TAIT
 * plutôt que de deviner, et déplacer un son sur un autre geste n'est pas se taire.
 * **Ici, le calque date ; là-bas, il ne datait pas. C'est toute la différence.**
 *
 * ## Ce que le son est, et ce qu'il n'est pas
 *
 * Le geste vient de `sb_004_mod_mp_shared_weaponpad` (`play_004_mod_mp_shared_weaponpad_appear`,
 * `54bd9e43`), nom cassé par hachage le 2026-08-30 et DÉSIGNÉ à l'oreille par l'utilisateur le
 * même jour.
 *
 * IL NE VAUT QUE POUR LES ARMES DE PUISSANCE, et ce n'est pas une prudence de notre part : le
 * Lua du jeu garde la table `MPItemSpawnerAudioAssets` par `MP_WEAPON_TIER.Power`. L'utilisateur
 * l'a dit dans les mêmes termes — « donc que les armes spéciales, pas les armes sur râteliers ».
 * Le calque, lui, ne publie QUE des socles : `doc.weaponPads` est la liste des socles du match,
 * les râteliers n'y sont pas. La restriction est donc tenue par la source, pas par un filtre.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { padEquipmentFamilyOf } from '../model/weaponPadFamilies'

/**
 * Le son de l'apparition d'une ARME sur socle. Banque `sb_004_mod_mp_shared_weaponpad`,
 * événement `54bd9e43` = `play_004_mod_mp_shared_weaponpad_appear`.
 */
export const PAD_SPAWN_SOUND_STEM = 'objective_pad_spawn'

/**
 * Le son de l'apparition d'un ÉQUIPEMENT sur socle — désigné à l'oreille par l'utilisateur le
 * 2026-08-30 (« Equip 7 »). Banque commune des équipements `sb_007_abl_shared`, événement
 * `4093f3c4`, nom Wwise non cassé.
 *
 * DEUX SONS ET PAS UN, parce que le calque distingue déjà les deux natures : un socle
 * d'équipement publie sa FAMILLE (`powerup_overshield` / `powerup_camo`) là où un socle d'arme
 * publie un identifiant de famille d'arme — c'est `padEquipmentFamilyOf` qui les sépare, la
 * même jointure que l'affichage des socles (`useReplayWeaponPads`). Le jeu, lui, n'a PAS de
 * banque de socle d'équipement (mesuré le 2026-08-30 : gabarit `sb_004_mod_mp_shared_%s` sur
 * le dictionnaire complet, rien pour l'équipement) — le son vient donc de la banque commune,
 * et la désignation est celle de l'oreille.
 */
export const EQUIPMENT_PAD_SPAWN_SOUND_STEM = 'equipment_pad_spawn'

/**
 * PAD_SPAWN_MAX_PAR_MATCH — plafond de sûreté, et il dit ce qu'il coupe.
 *
 * Un match porte 57 socles au plus haut relevé, et un socle réapparaît toutes les 30 à 180 s :
 * l'ordre de grandeur attendu est la centaine d'apparitions. Le plafond est là pour qu'un
 * artefact aberrant (un socle dont le cycle s'emballe) ne noie pas la piste, pas pour rogner un
 * match normal. Il est LOGGÉ par le test, jamais silencieux.
 */
export const PAD_SPAWN_MAX_PAR_MATCH = 300

/**
 * padSpawnSoundEvents — un son par apparition d'arme sur un socle, triés.
 *
 * Aucune notion de camp : un socle n'appartient à personne. Aucune dépendance au camp allié,
 * donc — contrairement aux sons d'état de zone, celui-ci sonne même quand la ligne « moi » du
 * tableau de score n'est pas résolue.
 */
export function padSpawnSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const pads = doc.weaponPads ?? []
  if (pads.length === 0) return []
  const spawns: { t: number; stem: string }[] = []
  for (const p of pads) {
    const stem = padEquipmentFamilyOf(p.weapon)
      ? EQUIPMENT_PAD_SPAWN_SOUND_STEM
      : PAD_SPAWN_SOUND_STEM
    for (const t of p.spawns ?? []) spawns.push({ t, stem })
  }
  spawns.sort((a, b) => a.t - b.t)
  return spawns
    .slice(0, PAD_SPAWN_MAX_PAR_MATCH)
    .map((s) => soundEvent(frameToMs(s.t, doc), s.stem))
}
