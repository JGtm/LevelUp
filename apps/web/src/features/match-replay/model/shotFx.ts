/**
 * shotFx.ts — CE QU'ON SAIT D'UN TIR AVANT DE LE DESSINER : où, quand, quelle forme,
 * quelle teinte, et dans quelle direction le tireur REGARDAIT.
 *
 * TIRS EN VÉHICULE (2026-09-03) : `doc.shots[i].v` marque un tir tiré depuis le véhicule de ce
 * slot (même clé que `VehicleTrack.slot`) — `x`/`y` valent alors la position INTERPOLÉE DU
 * VÉHICULE (centre), pas celle d'un tireur (le bipède ne réplique plus une fois embarqué,
 * `document.go`). Ce module résout ICI, une fois, ce qui NE DÉPEND QUE DU FILM (le véhicule
 * porteur et son cap à l'instant du tir, `vehicleHeadingAt`) et le montage de l'arme
 * (`vehicleWeaponMountOf`, table statique) ; ce qui dépend du SPRITE CHARGÉ (sa taille, donc le
 * décalage écran réel) reste au tracé (`drawShotsLayer`), qui seul connaît `sizeOf` — même
 * découpage précalcul/canevas que le reste du fichier.
 *
 * LA MESURE QUI A DÉBLOQUÉ CE CALQUE (2026-08-15). Le rendu orientait un tir par le champ
 * `h` de l'ÉVÉNEMENT de tir : présent sur 90 tirs sur 483 du film témoin (18,6 %), 16,1 %
 * sur les 23 artefacts locaux. D'où l'impression d'absence d'effet. Or le cap de REGARD
 * vit AUSSI dans les trajectoires — c'est lui qui alimente déjà le calque « Visée » — et il
 * est connu en continu. En le relisant à l'instant du tir (même `heldReading`, même fenêtre
 * de maintien que le cône), la couverture passe à **483/483 sur le témoin (100 %)** et
 * **99,1 % sur le corpus** ; l'âge médian de la lecture est de 0 ms, et 94,4 % des lectures
 * ont moins de 200 ms.
 *
 * POURQUOI LE REGARD SEUL, ET PAS LE `h` DE L'ÉVÉNEMENT QUAND IL EST LÀ. Parce que les deux
 * disent la même chose : sur les 2 866 tirs qui portent les DEUX, l'écart médian est de
 * 0,3° et 84,4 % sont sous 5°. Une seule règle vaut donc mieux que deux, et c'est celle qui
 * couvre tout.
 *
 * LA MÊLÉE N'ENTRE PAS : un coup de marteau n'est pas un tir, il n'a pas d'éclair de bouche
 * (règle établie au lot 3.2, et la référence Csstat exclut la mêlée explicitement). Mesure :
 * 0 événement de tir de famille `melee` sur les 17 904 du corpus — la règle ne coûte rien,
 * mais sans elle un film qui en porterait un afficherait un éclair faux.
 *
 * Pas de React, pas de canvas : logique pure, testée (shotFx.test.ts).
 */
import { fxTintOf, type FxTint } from '../layers/fxInk'
import { familyOf, type ShotFamily } from '../layers/shotEffects'
import { heldReading } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { vehicleHeadingAt } from './vehiclesLayer'
import { vehicleWeaponMountOf, type VehicleWeaponMount } from './vehicleWeaponMounts'
import { buildLivesBySlot, lifeOfSlotAt } from './livesPosition'

/**
 * VehicleShotSource — CE QU'IL FAUT, EN PLUS DU CENTRE, POUR PLACER L'EFFET AU BON MONTAGE
 * (`vehicleWeaponMounts.vehicleShotPlacement`, appelé au tracé une fois le sprite chargé).
 * `null` sur `ShotFxEntry.vehicleShot` = tir à pied, OU tir en véhicule dont l'arme n'a pas de
 * montage connu (`vehicleWeaponMountOf` rend `null`) : le repli reste alors le centre du
 * véhicule, exactement le comportement d'avant ce fichier.
 */
export interface VehicleShotSource {
  mount: VehicleWeaponMount
  /** Famille du véhicule porteur (clé de `sizeOf`/`spriteOf`, cf. `VehicleStyle`). */
  family: string | undefined
  /** Cap MONDE du véhicule à l'instant du tir (`vehicleHeadingAt`), degrés, convention `Point.h`. */
  headingDeg: number
}

/** Un tir prêt à dessiner : coordonnées MONDE, la conversion en pixels dépend du cadrage. */
export interface ShotFxEntry {
  /** Frame du tir sur la grille du rejeu. */
  frame: number
  x: number
  y: number
  /**
   * Cap de REGARD du tireur à cet instant, en degrés monde. null = aucune lecture dans la
   * fenêtre de maintien : l'éclair sera une bouffée ronde, jamais une direction inventée.
   */
  h: number | null
  fam: ShotFamily
  tint: FxTint
  /** Germe stable : deux lectures du même instant redonnent la même forme. */
  seed: number
  /** Tir en véhicule dont l'arme a un montage connu ; `null` sinon (repli sur le centre). */
  vehicleShot: VehicleShotSource | null
}

/**
 * buildShotFx précalcule les tirs dessinables d'un document — positions, familles, teintes
 * et regards résolus UNE fois au chargement (patron `buildKillFx`). Pendant la lecture, il
 * ne reste que le passage monde -> pixels.
 *
 * `aimHoldFrames` est la fenêtre de maintien du regard, en frames : la MÊME que celle du
 * cône de visée, parce que c'est la même lecture. Au-delà, on ne sait plus où le joueur
 * regardait, et une direction périmée affirmerait ce qu'on ignore.
 */
export function buildShotFx(doc: ReplayDocumentReady, aimHoldFrames: number): ShotFxEntry[] {
  if (doc.shots.length === 0) return []
  // UNE TRACE = UNE VIE, et le slot de biped est réattribué à chaque réapparition : on
  // groupe donc par slot, puis on retient la vie QUI COUVRE l'instant du tir. Prendre la
  // première venue lirait le regard d'une autre vie du même joueur.
  const bySlot = buildLivesBySlot(doc.tracks)
  const out: ShotFxEntry[] = []
  for (const s of doc.shots) {
    const label = s.w ? doc.weaponLabels?.[s.w] : undefined
    const fam = familyOf(label?.fx)
    if (fam === 'melee') continue
    const track = lifeOfSlotAt(bySlot, s.slot, s.t)
    const read = track ? heldReading(track.points, s.t, (p) => p.h, aimHoldFrames) : null
    out.push({
      frame: s.t,
      x: s.x,
      y: s.y,
      h: read ? read.value : null,
      fam,
      tint: fxTintOf(label?.tint),
      seed: s.t + s.slot,
      vehicleShot: vehicleShotSourceOf(doc, s.v, s.w, s.t),
    })
  }
  return out
}

/**
 * vehicleShotSourceOf — RÉSOUT UNE FOIS ce que le rendu aura besoin de savoir sur un tir en
 * véhicule : le véhicule porteur (par `v`, MÊME clé que `VehicleTrack.slot`), le montage de
 * l'arme (par `w`) et le cap du véhicule à l'instant `t`. `null` dès qu'un maillon manque —
 * pas de véhicule marqué, véhicule introuvable (film incohérent), ou arme sans montage connu —
 * et le tir garde alors sa position de centre déjà publiée par le document.
 */
function vehicleShotSourceOf(
  doc: ReplayDocumentReady,
  vehicleSlot: number | undefined,
  weaponTag: string | undefined,
  t: number,
): VehicleShotSource | null {
  if (vehicleSlot === undefined) return null
  const mount = vehicleWeaponMountOf(weaponTag)
  if (!mount) return null
  const track = doc.vehicles.find((v) => v.slot === vehicleSlot)
  if (!track) return null
  return { mount, family: track.family, headingDeg: vehicleHeadingAt(track, t) }
}
