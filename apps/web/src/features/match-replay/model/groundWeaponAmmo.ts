/**
 * groundWeaponAmmo.ts — LES MUNITIONS D'UNE ARME LÂCHÉE : L'EXACTE QUAND LE FILM LA DONNE,
 * LA DATÉE SINON.
 *
 * # DEUX SOURCES, ET ELLES NE DISENT PAS LA MÊME CHOSE
 *
 * 1. **EXACTE** (`groundWeapons[].ammo`, lot 6.10 du 2026-09-11). Le record de CRÉATION de
 *    l'objet porte un composant `weapon-ammo-component` dont deux champs sont prouvés : le
 *    chargeur et la réserve. Ce record est daté à l'INSTANT DU LÂCHER — il n'y a donc rien à
 *    dater, et la phrase n'écrit pas de « ≈ ». Preuve et chiffres :
 *    `.ai/V7.5/RAPPORT_MUNITIONS_EXACTES_2026-09-11.md`. Elle est ABSENTE sur la majorité des
 *    objets : c'est une réserve de LECTURE du décodeur, chiffrée par
 *    `coverage.groundWeaponItems.ammoRead`, pas une absence de l'objet.
 *
 * 2. **DATÉE** (le repli du lot 6.6, conservé). La dernière lecture d'inventaire du LÂCHEUR
 *    avant le lâcher, prise aux images-clés, EN RETARD — 9,0 s en médiane, 18,0 s au neuvième
 *    décile (rapport 6.3, §4.2). Elle ne se publie donc JAMAIS sans son âge.
 *
 * CE QUE LE LOT 6.6 AVAIT RÉFUTÉ RESTE RÉFUTÉ, et il faut le savoir pour ne pas y revenir : les
 * trois feuilles de l'ÉTAT PAR DÉFAUT de `ti=42` (liste de chargeurs, R(7), R(12)) ne portent
 * pas les munitions — 0 égalité sur 1 524 comparaisons pour la première. La valeur exacte ne
 * vient pas de là, mais d'un COMPOSANT du même record, que le balayage n'atteignait pas.
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
 * GroundWeaponAmmoReading — les munitions de l'arme, AVEC LEUR NATURE.
 *
 * C'EST UNE UNION DISCRIMINÉE, ET C'EST LE GARDE-FOU : une lecture `dated` ne peut pas
 * s'afficher sans son âge, parce que le type ne permet pas de l'oublier. Aplatir les deux en
 * un seul objet à `ageMs` optionnel rendrait l'oubli silencieux.
 */
export type GroundWeaponAmmoReading =
  | {
      /** `exact` : lue SUR L'OBJET, à l'instant du lâcher. Rien à dater. */
      kind: 'exact'
      mag: number
      res: number
    }
  | {
      /** `dated` : dernière lecture d'inventaire du lâcheur, en retard de `ageMs`. */
      kind: 'dated'
      mag: number
      /** Réserve à la même lecture, ou `null` quand le film n'en écrit pas. */
      res: number | null
      /** Écart entre le lâcher et la lecture, en millisecondes. Toujours >= 0. */
      ageMs: number
    }

/**
 * groundWeaponAmmoAt rend les munitions de CETTE arme au sol, ou `null`.
 *
 * L'EXACTE PASSE AVANT, TOUJOURS : quand l'objet la porte, la dernière lecture d'inventaire du
 * lâcheur n'a plus rien à ajouter — elle est plus vieille et moins sûre. Et elle vaut pour les
 * armes `spawned` aussi, que le repli ne peut pas servir faute de lâcheur mesuré.
 */
export function groundWeaponAmmoAt(
  doc: ReplayDocumentReady,
  item: ReplayGroundWeapon,
): GroundWeaponAmmoReading | null {
  if (item.ammo) return { kind: 'exact', mag: item.ammo.mag, res: item.ammo.res }
  if (item.origin !== 'dropped' || item.dropper < 0) return null
  const read = inventoryAt(doc, item.dropper, item.t0)
  // ÂGE NÉGATIF = LECTURE À VENIR : elle décrit un inventaire que le lâcheur n'a pas encore
  // au moment du lâcher. La rendre reviendrait à dater le futur au passé.
  if (!read || read.age < 0) return null
  const ageMs = frameToMs(read.age, doc)
  if (ageMs > GROUND_WEAPON_AMMO_MAX_AGE_MS) return null
  const slot = ammoSlotOf(doc, item, read.state.t, item.dropper, read.state.am)
  if (!slot || slot.mag === undefined || slot.mag === null) return null
  return { kind: 'dated', mag: slot.mag, res: slot.res ?? null, ageMs }
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
 * travers d'un rendu. Et surtout, une lecture DATÉE ne se sépare jamais de sa datation — c'est
 * l'unique garde-fou contre le nombre nu que la mesure interdit d'écrire là.
 *
 * LA LECTURE EXACTE, ELLE, S'ÉCRIT SANS « ≈ » ET SANS ÂGE, et c'est le point : elle est lue sur
 * l'objet à l'instant du lâcher. Lui coller le vocabulaire de l'approximation effacerait la
 * seule différence qui compte entre les deux sources.
 */
export function groundWeaponAmmoLine(
  t: (typeof REPLAY_TEXT)[ReplayLocale],
  reading: GroundWeaponAmmoReading,
): string {
  if (reading.kind === 'exact') return t.groundWeaponAmmoExactFmt(reading.mag, reading.res)
  const age = formatSeconds(reading.ageMs)
  if (reading.res === null) return t.groundWeaponAmmoFmt(reading.mag, age)
  return t.groundWeaponAmmoResFmt(reading.mag, reading.res, age)
}
