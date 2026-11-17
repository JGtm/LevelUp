/**
 * handCellHint.ts — L'INFOBULLE DE LA CELLULE D'ARME : les munitions de la main, les marques de
 * la lecture d'inventaire et l'état vide, dits en toutes lettres, avec leurs valeurs et leurs âges.
 *
 * POURQUOI CE MODULE. La rangée d'inventaire de la tuile n'a pas de cellule de munitions et
 * n'écrit aucune marque en texte : les munitions de la main, « dégainée ? », et l'état vide
 * (« Mort » / « Inventaire indisponible », avec ses DEUX âges) passent dans l'infobulle de l'arme
 * en main — la seule cellule qui reste pour les porter. Or cette cellule vit dans
 * `ReplayWeaponsRow`, qui ne lit pas l'inventaire : c'est la FICHE qui compose le texte ici et
 * le lui confie (`handHint`) ; la colonne des équipes le réemploie.
 *
 * LES RÈGLES, dans cet ordre :
 *   - arme À CHARGE en main (familles plasma / mêlée / énergie) : le pour-cent RESTANT, jamais
 *     le « 1/0 » parasite du film ; à chargeur : `mag / res` ; emplacement jamais écrit :
 *     « pleines » ;
 *   - sélecteur NON LU (et lecture non vide) : « dégainée ? » — une lacune dite ;
 *   - emplacement hors de la rangée (D ≥ 2 : mains nues, objet tenu) : RIEN — aucune arme de la
 *     rangée en main, aucune munition à décrire ;
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
 * « 1/0 » sur le témoin, toutes sur ces familles). Une seule définition, partagée par les
 * lectures de munitions (`ammoOfRecord`).
 */
const CHARGE_FX = new Set(['plasma', 'melee', 'light'])

/** L'arme `id` est-elle à charge ? (identifiant de famille → effet de tir → famille de charge) */
export function isChargeWeapon(doc: ReplayDocumentReady, id: string | undefined): boolean {
  return CHARGE_FX.has(familyOf(id ? doc.weaponLabels?.[id]?.fx : undefined))
}

/** Ce que la main a à dire : ses munitions, ou la lacune du sélecteur — ou rien (D ≥ 2). */
export type HandAmmo = AmmoHint | { kind: 'unread' }

/** Le pour-cent RESTANT d'une charge : le film compte le CONSOMMÉ. */
function chargeLeftPct(consumed: number): number {
  return Math.round(Math.max(0, 1 - consumed) * 100)
}

/**
 * ammoOfHand — les munitions de l'arme EN MAIN : null
 * sans compteurs lus, ou quand le sélecteur désigne un emplacement hors de la rangée (D ≥ 2) ;
 * `unread` quand le sélecteur n'a pas été lu sur une lecture pleine.
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
    // cause est connue.
    return !equipped.holstered && !read?.empty ? { kind: 'unread' } : null
  }
  return ammoOfRecord(doc, equipped.weapons[0]?.id, ammo[equipped.drawn] ?? {})
}

/**
 * ammoOfRecord — les munitions d'UN emplacement de l'inventaire, dans les trois natures de la
 * main : charge restante (arme à charge), compte `mag / res` (arme à chargeur), ou « pleines »
 * (emplacement jamais écrit). `weaponId` est l'arme qui occupe l'emplacement : sa famille dit si
 * c'est une charge. Une seule définition, pour la main et pour l'arme rangée.
 */
export function ammoOfRecord(
  doc: ReplayDocumentReady,
  weaponId: string | undefined,
  a: { mag?: number; res?: number; gauge?: number },
): AmmoHint {
  if (isChargeWeapon(doc, weaponId)) return { kind: 'charge', pct: chargeLeftPct(a.gauge ?? 0) }
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
