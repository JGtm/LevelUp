/**
 * weaponIconsByKey — le pictogramme d'une arme retrouvé par son weapon_key, à partir des lignes
 * d'élimination du fil.
 *
 * POURQUOI ICI. Une destruction de véhicule ne porte que le `causeKey` de sa cause (cf.
 * `vehicleDestructionSource`) : le document ne sert aucune URL d'image pour elle. Le pictogramme
 * d'une élimination, lui, est résolu par le serveur à partir de la source du dégât, et se trouve sur
 * la ligne (`weaponKey`, `weaponImageUrl`). La jointure se fait donc sur la CLÉ, le seul vocabulaire
 * commun : la destruction montre exactement le dessin que le fil montre déjà pour cette arme.
 *
 * LIMITE ASSUMÉE : une arme qui n'a éliminé personne dans ce match n'a pas de pictogramme ici, et la
 * ligne de destruction n'en montre aucun. Jamais le pictogramme d'une arme voisine.
 */
import type { FeedWeaponIcon } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'

/** Les pictogrammes d'arme connus du fil, par weapon_key. */
export type WeaponIconsByKey = ReadonlyMap<string, FeedWeaponIcon>

/** La table vide : aucune élimination connue, aucun pictogramme. */
export const NO_WEAPON_ICONS: WeaponIconsByKey = new Map()

/**
 * Relève, dans le fil, le premier pictogramme servi de chaque weapon_key. Une ligne sans clé ou sans
 * image (arme non identifiée sans ambiguïté) n'entre pas : elle ne nomme aucune arme.
 */
export function weaponIconsByKey(feed: readonly ReplayFeedEntry[]): WeaponIconsByKey {
  const icons = new Map<string, FeedWeaponIcon>()
  for (const { kill } of feed) {
    if (!kill?.weaponKey || !kill.weaponImageUrl || icons.has(kill.weaponKey)) continue
    icons.set(kill.weaponKey, {
      imageUrl: kill.weaponImageUrl,
      tinted: kill.weaponTinted,
      label: kill.weaponLabel,
    })
  }
  return icons
}
