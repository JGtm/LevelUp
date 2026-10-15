/**
 * startingWeaponsOnGround — QUELLES ARMES AU SOL LE REJEU MONTRE PAR DÉFAUT.
 *
 * Tout le monde apparaît avec les mêmes armes : celles qu'un joueur lâche en mourant n'apportent
 * rien à la lecture tactique. Par défaut, une arme LÂCHÉE (`origin = "dropped"`) dont le modèle
 * est une arme de départ du match est donc masquée ; une option les réaffiche.
 *
 * - Une arme POSÉE PAR LA CARTE (`origin = "spawned"` : râtelier, socle) reste TOUJOURS
 *   affichée, même si son modèle est celui d'une arme de départ : c'est un lieu, pas un reliquat.
 * - Une arme de départ = un modèle présent dans une dotation `loadouts[]` de `src = "birth"`,
 *   lue DANS LE DOCUMENT, match par match. Aucune liste d'armes ni nom de mode en dur.
 * - Un document sans dotation de naissance ne masque rien.
 * - Les identifiants sont normalisés : `loadouts[].w` s'écrit « 0xABCD1234 », `groundWeapons[].w`
 *   « abcd1234 » (`weaponLabelKeyOf`).
 */
import type { ReplayGroundWeapon } from '@/lib/api/types'

import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'

/** La source des dotations d'apparition dans `loadouts[].src`. */
const LOADOUT_SOURCE_BIRTH = 'birth'

interface BirthLoadout {
  src?: string
  w?: readonly string[]
}

/** startingWeaponKeys — les modèles (clé canonique) portés à l'apparition, par n'importe quel slot. */
export function startingWeaponKeys(loadouts: readonly BirthLoadout[] | undefined): ReadonlySet<string> {
  const keys = new Set<string>()
  for (const lo of loadouts ?? []) {
    if (lo.src !== LOADOUT_SOURCE_BIRTH) continue
    for (const w of lo.w ?? []) keys.add(weaponLabelKeyOf(w))
  }
  return keys
}

/**
 * visibleGroundWeapons — filtre les armes au sol. `includeStarting` vrai : tout est gardé.
 * Rend le tableau d'entrée tel quel quand rien n'est retiré (référence stable pour les memos).
 */
export function visibleGroundWeapons<T extends Pick<ReplayGroundWeapon, 'w' | 'origin'>>(
  items: readonly T[],
  loadouts: readonly BirthLoadout[] | undefined,
  includeStarting: boolean,
): readonly T[] {
  if (includeStarting) return items
  const starting = startingWeaponKeys(loadouts)
  if (starting.size === 0) return items
  return items.filter((item) => item.origin === 'spawned' || !starting.has(weaponLabelKeyOf(item.w)))
}
