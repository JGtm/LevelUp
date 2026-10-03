/**
 * vehicleWeaponRegistry.ts — LE LECTEUR DU REGISTRE DES ARMES DE VÉHICULE (schéma 69, retours du
 * rejeu du 2026-09-23, lot M4a). LE SEUL fichier du rejeu qui lise `doc.vehicleWeapons`.
 *
 * CE QU'IL REMPLACE. Trois tables CLIENT — le style (`vehicleShotFx.ts`), le son
 * (`vehicleShotSound.ts`) et le montage (`vehicleWeaponMounts.ts`) — étaient clées par des tags
 * `weap` lus dans le module du jeu (lot V3F) et recopiés ici sous le gabarit `0x<tag>00000000` :
 * mesure du 2026-09-23 sur les 111 documents du parc, 7 des 11 clés n'apparaissaient dans AUCUN
 * document, 3 tags réellement publiés en étaient absents, et le Rockethog était muet. Le TITRE
 * déclare désormais ces armes (`config/titles/{slug}/mappings/vehicle_weapons.toml`, clé = tag
 * OBSERVÉ dans un film, preuve à l'appui, garde-rail Go sur une fixture datée du parc) et le
 * serveur en publie, à la requête, les seules armes que les tirs du document emploient — keyées
 * par la clé MÊME du tir (`Shot.w`). Le client n'a plus de tag à connaître, ni de gabarit à
 * recomposer : un garde-rail (`vehicleWeaponRegistry.guard.test.ts`) interdit tout littéral de
 * tag d'arme de véhicule dans le code du rejeu.
 *
 * UNE ARME ABSENTE DU REGISTRE garde le rendu neutre et le silence — jamais le style, le son ou
 * le montage d'une voisine. Un SON absent sur une arme présente est un silence DÉCIDÉ (le
 * registre en porte la raison côté serveur).
 */
import type { ReplayVehicleWeapon } from '@/lib/api/types'

import type { FxTint } from '../layers/fxInk'
import type { ShotFamily } from '../layers/shotEffects'
import type { VehicleMountClass, VehicleWeaponMount } from './vehicleWeaponMounts'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Le style d'un tir : la FORME de l'éclair et la NATURE de la décharge (listes du titre). */
export interface VehicleShotStyle {
  fx: ShotFamily
  tint: FxTint
}

/**
 * VEHICLE_WEAPON_AIM_CLASS — la classe de visée publiée (`fixed` / `turret`, anglaise comme toute
 * énumération du contrat) vers celle de la géométrie du calque. Une valeur inconnue n'a pas de
 * classe : pas de montage, repli au centre du véhicule.
 */
const VEHICLE_WEAPON_AIM_CLASS: Readonly<Record<string, VehicleMountClass>> = {
  fixed: 'fixe',
  turret: 'tourelle',
}

/** L'entrée du registre d'une arme de tir, ou `null`. */
export function vehicleWeaponOf(
  doc: Pick<ReplayDocumentReady, 'vehicleWeapons'>,
  w: string | undefined,
): ReplayVehicleWeapon | null {
  if (!w) return null
  return doc.vehicleWeapons?.[w] ?? null
}

/**
 * vehicleShotStyleOf — le style d'une arme de véhicule, ou `null`. APPELÉ EN SECOND par
 * `buildShotFx` : une arme de JOUEUR garde le style de son registre (`weaponLabels`).
 */
export function vehicleShotStyleOf(
  doc: Pick<ReplayDocumentReady, 'vehicleWeapons'>,
  w: string | undefined,
): VehicleShotStyle | null {
  const e = vehicleWeaponOf(doc, w)
  if (!e) return null
  return { fx: e.fx as ShotFamily, tint: e.tint as FxTint }
}

/**
 * vehicleShotSoundStem — le stem de la PREMIÈRE variante du son de tir, ou `undefined` (arme
 * absente du registre, ou silence décidé). Les variantes s'attachent par `SOUND_VARIANTS`.
 */
export function vehicleShotSoundStem(
  doc: Pick<ReplayDocumentReady, 'vehicleWeapons'>,
  w: string | undefined,
): string | undefined {
  const stem = vehicleWeaponOf(doc, w)?.sound
  return stem ? stem : undefined
}

/**
 * vehicleWeaponMountOf — l'ancre de l'arme sur le sprite de son véhicule, ou `null` (tir à pied,
 * arme de joueur tirée d'un siège, arme sans montage documenté) : l'éclair part alors du centre
 * du véhicule — jamais d'une position inventée.
 */
export function vehicleWeaponMountOf(
  doc: Pick<ReplayDocumentReady, 'vehicleWeapons'>,
  w: string | undefined,
): VehicleWeaponMount | null {
  const m = vehicleWeaponOf(doc, w)?.mount
  if (!m) return null
  const classe = VEHICLE_WEAPON_AIM_CLASS[m.aim]
  if (!classe) return null
  return { classe, ax: m.ax, ay: m.ay }
}
