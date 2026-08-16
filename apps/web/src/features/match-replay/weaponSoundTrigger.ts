/**
 * weaponSoundTrigger.ts — QUAND jouer un son, et SOUS QUELLE CLÉ. Fonctions pures, testées.
 *
 * LA CLÉ. `Shot.w` est un identifiant d'arme du film : soit une FAMILLE (8 chiffres hex),
 * soit un identifiant GLOBAL 64 bits dont la famille est la moitié haute. Le manifeste des
 * sons est indexé par FAMILLE (gid du tag `weap`, 8 hex minuscules) : une arme a plusieurs
 * identifiants globaux (variantes, skins) mais une seule famille de son. La normalisation
 * ici est le MIROIR EXACT de `buildWeaponLabels` (identity.go) — même seuil de longueur,
 * prefixe `0x` compris — pour que sons et libellés lisent le même identifiant de la même
 * façon.
 *
 * LE QUAND. Le film n'enregistre que les tirs qui INFLIGENT un dégât (contrat de
 * drawShotsLayer) : chaque tir listé est un coup réel, on le joue quand la tête de lecture
 * le franchit. Deux garde-fous, tous deux nommés :
 *
 * - MAX_AVANCE : un bond de plus d'une seconde de film en un seul pas n'est pas de la
 *   lecture (onglet suspendu, à-coup système, boucle de fin) — aucun son, sinon l'avalanche.
 * - MAX_SONS_PAR_PAS : un pas normal ne franchit qu'une poignée de tirs ; au-delà, on borne
 *   pour que le mixage reste un son de rejeu, pas un mur.
 */

/** Ce que le déclencheur lit d'un tir : l'instant (frame) et l'identifiant d'arme. */
export interface TirMinimal {
  t: number
  w?: string
}

/** Nombre maximal de sons déclenchés par pas d'animation. */
export const MAX_SONS_PAR_PAS = 8

/**
 * weaponFamilyKey normalise un identifiant d'arme du film vers la clé du manifeste.
 *
 * Rend `null` pour tout ce qui n'est pas un identifiant lisible — un tir sans arme connue
 * reste muet, il n'emprunte pas le son d'un voisin (même règle que les libellés).
 */
export function weaponFamilyKey(id: string | undefined | null): string | null {
  if (!id) return null
  const brut = id.trim()
  const hex = brut.replace(/^0x/i, '')
  if (!/^[0-9a-fA-F]{1,16}$/.test(hex)) return null
  const v = BigInt('0x' + hex)
  // Miroir d'identity.go : `len(id) > 10` porte sur l'identifiant TEL QUE REÇU, préfixe
  // compris. Un identifiant long est global (famille = moitié haute), un court est déjà
  // une famille.
  const famille = brut.length > 10 ? v >> 32n : v & 0xffffffffn
  return famille.toString(16).padStart(8, '0')
}

/**
 * tirsAJouer rend les clés de famille des tirs que la tête de lecture vient de franchir,
 * dans l'ordre du film — fenêtre (avant, courant], bornée par les deux garde-fous.
 */
export function tirsAJouer(
  tirs: readonly TirMinimal[],
  avant: number,
  courant: number,
  maxAvance: number,
): string[] {
  if (!(courant > avant)) return []
  if (courant - avant > maxAvance) return []
  const franchis = tirs
    .filter((s) => s.t > avant && s.t <= courant)
    .sort((a, b) => a.t - b.t)
  const cles: string[] = []
  for (const s of franchis) {
    if (cles.length >= MAX_SONS_PAR_PAS) break
    const cle = weaponFamilyKey(s.w)
    if (cle) cles.push(cle)
  }
  return cles
}
