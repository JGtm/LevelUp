/**
 * groundWeaponAmmo.ts — LES MUNITIONS D'UNE ARME LÂCHÉE, ET POURQUOI ELLES SONT DATÉES.
 *
 * # CE QUE LA MESURE A RÉFUTÉ, ET CE QUI RESTE
 *
 * La question posée au lot 6.6 (2026-09-10) était : « les munitions sont-elles SUR L'OBJET ? ».
 * Le record de création d'une arme au sol (`ti=42`) porte trois feuilles candidates, toutes
 * instrumentées et confrontées sur 61 films d'arène et 9 219 objets lâchés — verdict au
 * rapport `.ai/V7.5/RAPPORT_MUNITIONS_OBJET_2026-09-10.md` :
 *
 *   - la « liste de chargeurs » (point 5) porte des mots de 32 bits dont 6 609 valeurs
 *     DISTINCTES sur 10 613, minimum 98, DEUX sous 1024 — et zéro égal au chargeur du lâcheur
 *     sur 1 524 comparaisons. C'est un HANDLE, pas un compte ;
 *   - le R(7) du point 4 : 10 égalités sur 6 477 (0,2 %) ; le R(12) du point 3 : zéro.
 *
 * IL N'Y A DONC AUCUN CHIFFRE EXACT À AFFICHER, et ce module est le REPLI que le plan avait
 * prévu : la dernière lecture d'inventaire du lâcheur AVANT le lâcher. Elle est vraie, elle
 * est déjà servie dans l'artefact (aucune recuisson), et elle est EN RETARD — 9,0 s en médiane,
 * 18,0 s au neuvième décile (mesure du rapport 6.3, §4.2). C'est pourquoi rien ici ne rend un
 * nombre nu : l'appelant reçoit l'âge AVEC la valeur, et l'infobulle l'écrit.
 *
 * # LES CINQ REFUS
 *
 * Une arme `spawned` n'a pas de lâcheur mesuré ; une arme absente du relevé de loadout ne peut
 * pas être indexée dans `am` sans servir les munitions d'une AUTRE arme ; un emplacement sans
 * chargeur est une arme à JAUGE (publier 0 affirmerait « chargeur vide », cf. `AmmoSlot`) ; une
 * lecture plus vieille que [GROUND_WEAPON_AMMO_MAX_AGE_MS] ne dit plus rien du lâcher ; une
 * lecture À VENIR n'est pas une information passée (même doctrine que `grenadeBoxAt`).
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas.
 */
import type { ReplayGroundWeapon } from '@/lib/api/types'

import type { REPLAY_TEXT, ReplayLocale } from '../i18n/i18n'
import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'
import { formatSeconds, frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { inventoryAt } from './inventoryReading'

/**
 * GROUND_WEAPON_AMMO_MAX_AGE_MS — au-delà de cet âge, la lecture n'est pas montrée.
 *
 * LA VALEUR EST UN INTERVALLE D'IMAGE-CLÉ (~20 s), et c'est ce qui la justifie : l'inventaire
 * n'est lu QUE là (cadence mesurée 9 à 18 s, `inventory.go`). Une lecture plus vieille n'est
 * donc pas « l'image-clé précédente », c'est un TROU dans le recensement — le porteur a pu
 * ramasser, vider et reprendre l'arme entre les deux. Afficher ce chiffre-là, même daté,
 * donnerait à lire un état qui n'a plus de rapport avec le lâcher.
 */
export const GROUND_WEAPON_AMMO_MAX_AGE_MS = 20_000

/**
 * GroundWeaponAmmoReading — le chargeur (et la réserve quand elle est lue) du lâcheur, avec
 * l'ÂGE de la lecture au moment du lâcher. Les trois voyagent ensemble : sans l'âge, la valeur
 * ment par omission.
 */
export interface GroundWeaponAmmoReading {
  /** Balles au chargeur à la dernière image-clé avant le lâcher. */
  mag: number
  /** Réserve à la même lecture, ou `null` quand le film n'en écrit pas. */
  res: number | null
  /** Écart entre le lâcher et la lecture, en millisecondes. Toujours >= 0. */
  ageMs: number
}

/**
 * groundWeaponAmmoAt rend les munitions du lâcheur pour CETTE arme au sol, ou `null` — et
 * `null` est le cas nominal dans une bonne part des situations (cf. les cinq refus en tête).
 */
export function groundWeaponAmmoAt(
  doc: ReplayDocumentReady,
  item: ReplayGroundWeapon,
): GroundWeaponAmmoReading | null {
  if (item.origin !== 'dropped' || item.dropper < 0) return null
  const read = inventoryAt(doc, item.dropper, item.t0)
  // ÂGE NÉGATIF = LECTURE À VENIR : elle décrit un inventaire que le lâcheur n'a pas encore
  // au moment du lâcher. La rendre reviendrait à dater le futur au passé.
  if (!read || read.age < 0) return null
  const ageMs = frameToMs(read.age, doc)
  if (ageMs > GROUND_WEAPON_AMMO_MAX_AGE_MS) return null
  const slot = ammoSlotOf(doc, item, read.state.t, item.dropper, read.state.am)
  if (!slot || slot.mag === undefined || slot.mag === null) return null
  return { mag: slot.mag, res: slot.res ?? null, ageMs }
}

/**
 * ammoSlotOf trouve l'emplacement de munitions de l'arme lâchée DANS LE RELEVÉ DE CET INSTANT.
 *
 * LE LOADOUT DOIT DATER DU MÊME INSTANT QUE LA LECTURE, et c'est la règle du document :
 * « `Am` est l'état de munitions des emplacements portant une arme, DANS L'ORDRE de
 * `Loadout.W` » (`inventory.go`). Un loadout d'un autre instant peut porter d'autres armes,
 * dans un autre ordre — l'index serait alors celui d'une autre arme. On prend donc le relevé
 * BRUT (`doc.loadouts`), jamais `loadoutAt`, qui affine la lecture avec les changements d'arme
 * datés et ne correspond plus, par construction, à l'ordre des `am`.
 */
function ammoSlotOf(
  doc: ReplayDocumentReady,
  item: ReplayGroundWeapon,
  t: number,
  slot: number,
  am: ReadonlyArray<{ mag?: number | null; res?: number | null }>,
): { mag?: number | null; res?: number | null } | null {
  const load = (doc.loadouts ?? []).find((l) => l.slot === slot && l.t === t)
  if (!load) return null
  const key = weaponLabelKeyOf(item.w)
  const index = load.w.findIndex((w) => weaponLabelKeyOf(w) === key)
  if (index < 0 || index >= am.length) return null
  return am[index]
}

/**
 * groundWeaponAmmoLine — LE FRAGMENT D'INFOBULLE, composé ici et non dans le composant.
 *
 * MÊME RÈGLE QUE `inventoryEmptyHint` : une phrase bâtie dans un composant ne se teste qu'au
 * travers d'un rendu. Et surtout, la phrase ne se sépare JAMAIS de sa datation — c'est
 * l'unique garde-fou contre le nombre nu que la mesure interdit d'écrire.
 */
export function groundWeaponAmmoLine(
  t: (typeof REPLAY_TEXT)[ReplayLocale],
  reading: GroundWeaponAmmoReading,
): string {
  const age = formatSeconds(reading.ageMs)
  if (reading.res === null) return t.groundWeaponAmmoFmt(reading.mag, age)
  return t.groundWeaponAmmoResFmt(reading.mag, reading.res, age)
}
