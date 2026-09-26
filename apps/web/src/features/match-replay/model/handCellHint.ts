/**
 * handCellHint.ts — L'INFOBULLE DE LA CELLULE D'ARME QUAND ELLE EST SEULE : ce que la fiche
 * normale montre dans sa cellule de MUNITIONS et dans ses MARQUES SOUPLES, dit ici en toutes
 * lettres, avec ses valeurs et ses âges.
 *
 * POURQUOI CE MODULE (plan fiches compactes 2026-09-06, étape 3, décision D5). Sur la tuile
 * compacte la rangée d'inventaire n'a plus de cellule de munitions (`showAmmo: false`) et
 * n'écrit plus aucune marque en texte (`showInventoryMarks: false`) : les munitions de la main,
 * « dégainée ? », et l'état vide (« Mort » / « Inventaire indisponible », avec ses DEUX âges)
 * passent dans l'infobulle de l'arme en main — la seule cellule qui reste pour les porter. Or
 * cette cellule vit dans `ReplayWeaponsRow`, qui ne lit pas l'inventaire : c'est la FICHE qui
 * compose le texte ici et le lui confie (`handHint`).
 *
 * LES RÈGLES SONT CELLES DE `AmmoCell` ET DE `InventoryEmptyMark`, à l'identique et dans le
 * même ordre — ce module ne décide rien de nouveau, il DIT ce que la cellule montrait :
 *   - arme À CHARGE en main (familles plasma / mêlée / énergie) : le pour-cent RESTANT, jamais
 *     le « 1/0 » parasite du film ; à chargeur : `mag / res` ; emplacement jamais écrit :
 *     « pleines » ;
 *   - sélecteur NON LU (et lecture non vide) : « dégainée ? » — une lacune dite ;
 *   - armes rangées (D=2) : RIEN — aucune arme en main, aucune munition à décrire ;
 *   - lecture VIDE : le libellé de l'état, puis `inventoryEmptyHint` (l'âge de la lecture vide,
 *     et celui de l'équipement substitué quand il y en a un).
 *
 * Tout ce fichier est PUR : aucun React, donc testable sans rendu.
 */
import { familyOf } from '../layers/shotEffects'
import type { AmmoHint, ReplayText } from '../i18n/i18nContract'
import type { EquippedReading } from './equippedLogic'
import { inventoryAt, inventoryEmptyHint, type InventoryReading } from './inventoryReading'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/**
 * Les familles d'arme À CHARGE : pas de chargeur, une jauge (mesure 2026-08-24 : 46 cellules
 * « 1/0 » sur le témoin, toutes sur ces familles). Une seule définition, partagée par la
 * cellule de munitions (`ReplayInventoryRow`) et par cette infobulle.
 */
const CHARGE_FX = new Set(['plasma', 'melee', 'light'])

/** L'arme `id` est-elle à charge ? (identifiant de famille → effet de tir → famille de charge) */
export function isChargeWeapon(doc: ReplayDocumentReady, id: string | undefined): boolean {
  return CHARGE_FX.has(familyOf(id ? doc.weaponLabels?.[id]?.fx : undefined))
}

/** Ce que la main a à dire : ses munitions, ou la lacune du sélecteur — ou rien (D=2). */
export type HandAmmo = AmmoHint | { kind: 'unread' }

/** Le pour-cent RESTANT d'une charge : le film compte le CONSOMMÉ. */
function chargeLeftPct(consumed: number): number {
  return Math.round(Math.max(0, 1 - consumed) * 100)
}

/**
 * ammoOfHand — les munitions de l'arme EN MAIN, dans les mêmes branches que `AmmoCell` : null
 * sans compteurs lus, ou quand les armes sont rangées (D=2) ; `unread` quand le sélecteur n'a
 * pas été lu sur une lecture pleine.
 */
export function ammoOfHand(
  doc: ReplayDocumentReady,
  equipped: EquippedReading,
  read: InventoryReading | null,
): HandAmmo | null {
  const ammo = read?.state.am ?? []
  if (ammo.length === 0) return null
  if (equipped.drawn === null) {
    // SAUF SUR UNE LECTURE VIDE : `equippedWeapons` y rend volontairement `drawnUnread`, et
    // l'état vide dit POURQUOI — écrire « dégainée ? » à côté poserait une lacune là où la
    // cause est connue (même règle que la cellule).
    return !equipped.holstered && !read?.empty ? { kind: 'unread' } : null
  }
  const a = ammo[equipped.drawn] ?? {}
  if (isChargeWeapon(doc, equipped.weapons[0]?.id)) return { kind: 'charge', pct: chargeLeftPct(a.gauge ?? 0) }
  if (a.mag !== undefined) return { kind: 'count', mag: a.mag, res: a.res }
  if (a.gauge !== undefined) return { kind: 'charge', pct: chargeLeftPct(a.gauge) }
  return { kind: 'full' }
}

/**
 * handCellHint — l'infobulle composée pour la cellule d'arme seule : munitions (ou lacune du
 * sélecteur), puis l'état vide avec ses âges. `null` quand il n'y a rien à dire.
 */
export function handCellHint(
  t: ReplayText,
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
  equipped: EquippedReading | null,
): string | null {
  const read = inventoryAt(doc, slot, frame)
  const parts: string[] = []
  const ammo = equipped ? ammoOfHand(doc, equipped, read) : null
  if (ammo) parts.push(ammo.kind === 'unread' ? t.drawnUnknown : t.ammoHintFmt(ammo))
  if (read?.empty) {
    const label = read.empty.kind === 'dead' ? t.inventoryDeadLabel : t.inventoryEmptyLabel
    parts.push(`${label} — ${inventoryEmptyHint(t, read, read.empty, doc)}`)
  }
  return parts.length > 0 ? parts.join(' · ') : null
}
