/**
 * killSourceIcons — le RÉFÉRENTIEL des pictogrammes d'arme, rangé par weapon_key.
 *
 * D'OÙ IL VIENT. Le serveur sert, par titre, une entrée par arme (`GET /assets/{titre}/kill-sources` :
 * clé, nom dans la langue de la requête, image, teinte). C'est la même résolution que les lignes
 * d'élimination du fil : le dessin d'une arme est celui qu'une élimination par cette arme affiche. Une
 * ligne qui ne connaît que la clé de l'arme (la cause d'une destruction de véhicule, `causeKey`)
 * y retrouve son pictogramme, que l'arme ait éliminé quelqu'un dans le match ou non.
 *
 * Logique pure : la requête est dans `hooks/useKillSourceIcons`.
 */
import type { KillSourceIconEntry } from '@/lib/api/types'

import type { FeedWeaponIcon } from './feedEventLines'

/** Les pictogrammes d'arme du titre, par weapon_key. */
export type WeaponIconsByKey = ReadonlyMap<string, FeedWeaponIcon>

/** Le référentiel vide : en chargement, en erreur ou pour un titre sans pictogrammes, aucune ligne n'en porte. */
export const NO_WEAPON_ICONS: WeaponIconsByKey = new Map()

/**
 * weaponIconsOf range les entrées du serveur par clé. Une entrée sans clé ou sans image n'est
 * retrouvable (ou dessinable) par personne : elle n'entre pas.
 */
export function weaponIconsOf(entries: readonly KillSourceIconEntry[] | null | undefined): WeaponIconsByKey {
  const icons = new Map<string, FeedWeaponIcon>()
  for (const e of entries ?? []) {
    if (!e.key || !e.image_url) continue
    icons.set(e.key, { imageUrl: e.image_url, tinted: e.tinted, label: e.label })
  }
  return icons
}
