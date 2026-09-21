/**
 * shotFx.ts — CE QU'ON SAIT D'UN TIR AVANT DE LE DESSINER : où, quand, quelle forme,
 * quelle teinte, et dans quelle direction le tireur REGARDAIT.
 *
 * TIRS EN VÉHICULE (2026-09-03) : `doc.shots[i].v` marque un tir tiré depuis le véhicule de ce
 * slot (même clé que `VehicleTrack.slot`) — `x`/`y` valent alors la position INTERPOLÉE DU
 * VÉHICULE (centre), pas celle d'un tireur (le bipède ne réplique plus une fois embarqué,
 * `document.go`). Ce module résout ICI, une fois, ce qui NE DÉPEND QUE DU FILM (le véhicule
 * porteur et son cap à l'instant du tir, `vehicleChassisHeadingAt`) et le montage de l'arme
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
import { heldReading } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { vehicleChassisHeadingAt, vehicleShooterAimAt } from './vehiclesAim'
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
  /**
   * Le montage de l'arme sur le châssis, ou `null` quand le tag d'arme n'est pas documenté
   * (Wraith, Gungoose, Falcon, tourelle posée au sol…). `null` ne fait PLUS perdre la source :
   * l'éclair reste au CENTRE du véhicule, mais il garde sa DIRECTION (cf. `vehicleShotPlacement`).
   */
  mount: VehicleWeaponMount | null
  /** Famille du véhicule porteur (clé de `sizeOf`/`spriteOf`, cf. `VehicleStyle`). */
  family: string | undefined
  /**
   * Cap MONDE du véhicule à l'instant du tir — celui auquel le CHÂSSIS EST DESSINÉ
   * (`vehicleChassisHeadingAt`), degrés, convention `Point.h`. C'est le même que le sprite, et ce
   * n'est pas un détail : le montage d'arme est une ancre dans le repère LOCAL du sprite, donc un
   * éclair posé à un autre cap sortirait du châssis qu'il est censé quitter.
   */
  headingDeg: number
  /**
   * Visée MESURÉE de CELUI QUI A TIRÉ, degrés monde, ou `null` faute de lecture en vigueur
   * (`vehicleShooterAimAt`, apparié par SLOT — cf. son en-tête pour le négatif du lot 5.5.1 qui
   * rend cette valeur nécessaire). Elle n'oriente QUE les montages de classe `tourelle` : le
   * châssis, lui, garde son propre cap.
   */
  shooterHeadingDeg: number | null
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
  /** Tir d'une ARME DE VÉHICULE ; `null` pour un tir à pied ou une arme de joueur. */
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
      vehicleShot: vehicleShotSourceOf(doc, s, s.t),
    })
  }
  return out
}

/**
 * vehicleShotSourceOf — RÉSOUT UNE FOIS ce que le rendu aura besoin de savoir sur un tir en
 * véhicule : le véhicule porteur (par `v`, MÊME clé que `VehicleTrack.slot`), le montage de
 * l'arme (par `w`) quand il est documenté, et le cap du véhicule à l'instant `t`.
 *
 * # POURQUOI UN MONTAGE INCONNU NE FAIT PLUS PERDRE LA SOURCE (2026-09-20)
 *
 * LE DÉFAUT MESURÉ, et il explique à lui seul le constat utilisateur du 2026-09-19 (« toujours
 * pas d'effets de tir pour les véhicules »). Le cap de REGARD d'un tir vient de la trajectoire
 * du BIPÈDE (`heldReading` ci-dessous) — or un bipède EMBARQUÉ NE RÉPLIQUE PLUS. Mesure du
 * 2026-09-20 sur quatre documents cuits : sur les tirs qui portent `v`, le cap de regard est
 * lisible pour **1 sur 241** (`4f77afc1`), 3 sur 241 (`5676a9ba`), 0 sur 47 (`c259789d`) et
 * 0 sur 15 (`8a485699`). Sans cap, `drawMuzzleFlash` tombe sur la BOUFFÉE RONDE — sans
 * direction, centrée sur le châssis, et dans la teinte `neutral` (68 % de ces tirs portent une
 * arme absente de `weaponLabels`, donc sans famille ni teinte). Un halo gris pâle centré sur le
 * sprite ne se lit pas comme un tir : c'est ce que l'utilisateur ne voyait pas.
 *
 * CE QUE LE FILM DONNE, LUI, POUR 100 % DE CES TIRS : le CAP DU VÉHICULE. Garder la source même
 * sans montage rend donc une direction à l'éclair, et `vehicleShotPlacement` la traduit.
 *
 * LA GARDE EST LE REGISTRE D'ARMES, et c'est le MÊME discriminateur que le son
 * (`shotSoundStem` : « leurs identifiants sont ABSENTS de `weaponLabels` »). Une arme DE JOUEUR
 * tirée depuis un siège de passager reste sur son propre cap de regard, jamais sur celui du
 * châssis : le passager vise où il veut, et lui prêter la direction du véhicule serait une
 * invention. Mesure : `c259789d` ne porte QUE des tirs de ce genre (47 sur 47 dans le registre),
 * et ce chemin ne les touche pas.
 */
function vehicleShotSourceOf(
  doc: ReplayDocumentReady,
  shot: { v?: number; w?: string; slot: number },
  t: number,
): VehicleShotSource | null {
  if (shot.v === undefined) return null
  const mount = vehicleWeaponMountOf(shot.w)
  // ARME DE VÉHICULE = absente du registre d'armes de joueur (cf. l'en-tête de cette fonction).
  const armeDeVehicule = shot.w !== undefined && doc.weaponLabels?.[shot.w] === undefined
  if (!mount && !armeDeVehicule) return null
  const track = doc.vehicles.find((v) => v.slot === shot.v)
  if (!track) return null
  return {
    mount,
    family: track.family,
    headingDeg: vehicleChassisHeadingAt(track, t),
    // LE SLOT DU TIREUR, PAS SON SIÈGE (lot 5.5) : c'est la seule clé qui désigne l'occupant
    // qui a tiré, et elle vaut pour le tourelleur passager du Warthog comme pour le conducteur
    // artilleur du Scorpion.
    shooterHeadingDeg: vehicleShooterAimAt(track, shot.slot, t),
  }
}
