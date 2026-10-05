/**
 * vehicleWeaponMounts.ts — D'OÙ PART UN TIR EN VÉHICULE, PAS DU CENTRE DU PION.
 *
 * DEMANDE UTILISATEUR (2026-09-03) : « si ça vient du passager, faut que le tir vienne du
 * siège passager qui a une tourelle ». Avant ce fichier, `doc.shots` marqué `v` (slot
 * véhicule) dessinait son éclair au CENTRE du véhicule (`ShotFxEntry.x/y` = position du
 * véhicule, cf. `document.go` : « X, Y sont l'origine INTERPOLÉE DU VÉHICULE »). Ce fichier
 * fournit l'ANCRE PHYSIQUE de l'arme du véhicule, en FRACTIONS DU SPRITE (repère nez en haut,
 * le même que `vehiclesPaint.drawRotatedSprite` : origine au centre, `ax` de -0,5 (bord
 * gauche) à +0,5 (bord droit), `ay` de -0,5 (nez, haut de l'image) à +0,5 (arrière, bas de
 * l'image)) — et la géométrie pure qui la tourne par le cap du véhicule.
 *
 * LES ANCRES NE VIVENT PLUS ICI (schéma 69, retours du rejeu du 2026-09-23, lot M4a). La table
 * `Shot.w` -> ancre de ce fichier était clée par des tags `weap` du module du jeu recopiés sous le
 * gabarit `0x<tag>00000000` — dont la majorité n'apparaissait dans aucun film. Elle est remplacée
 * par le REGISTRE DU TITRE (`config/titles/{slug}/mappings/vehicle_weapons.toml`, clé = tag
 * OBSERVÉ, preuve à l'appui), publié dans le document et lu par `vehicleWeaponRegistry.ts`. Chaque
 * arme du registre y porte son ancre, mesurée sur le sprite de son véhicule (provenance dans le
 * `proof` de l'entrée) : une arme sans ancre tirerait du centre du sprite, là où se trouve le pion
 * du conducteur.
 *
 * CE QUE CE FICHIER GARDE : le type d'une ancre et la GÉOMÉTRIE qui la tourne par le cap du
 * véhicule. CLASSE : `fixe` = solidaire du nez, ne vise qu'où le véhicule pointe (le pilote EST
 * le viseur) — l'ancre EST la bouche ; `tourelle` = visée manœuvrée INDÉPENDAMMENT du cap (canon
 * arrière du Warthog, canon du Scorpion, poste latéral du Falcon) — l'ancre est le PIVOT, et la
 * bouche est à `reach` (longueur du canon, fraction de la longueur du sprite) du pivot dans la
 * direction de la visée MESURÉE de son tireur ; sans visée lue, le tir part du pivot en bouffée
 * ronde, jamais d'une direction inventée.
 */
import { vehicleAimAngle, vehicleScreenAngle, vehicleSpriteScale } from './vehiclesLayer'
import type { XY } from '../../../lib/replay/replayLogic'

/** `fixe` = solidaire du nez, vise où pointe le véhicule. `tourelle` = visée indépendante, inconnue. */
export type VehicleMountClass = 'fixe' | 'tourelle'

/** Une ancre de montage : classe de visée + position en fractions du sprite (repère nez en haut). */
export interface VehicleWeaponMount {
  classe: VehicleMountClass
  /** Latéral : -0,5 (bord gauche du sprite) .. +0,5 (bord droit). */
  ax: number
  /** Longitudinal : -0,5 (nez) .. +0,5 (arrière). */
  ay: number
  /**
   * Longueur du canon d'une TOURELLE, en fraction de la LONGUEUR du sprite (sa hauteur, nez en
   * haut) : la bouche est à cette distance du pivot, le long de la visée du tireur. Absent = 0
   * (arme fixe, dont l'ancre est déjà la bouche).
   */
  reach?: number
}

// --- GÉOMÉTRIE : ancre -> point d'écran ---------------------------------------------------------

/** Ce que le tracé sait du sprite au moment de placer une ancre (mêmes champs que `vehiclesPaint`). */
export interface VehicleMountSpriteSize {
  naturalWidthPx: number
  naturalHeightPx: number
  mmPerPx: number
}

/** Le résultat : un décalage ÉCRAN (pixels, densité `k` déjà appliquée) et une direction. */
export interface VehicleShotPlacement {
  /** À ajouter au centre déjà projeté du véhicule (`project(world, view)`). */
  offset: XY
  /** Angle canevas de la décharge, ou `null` (tourelle dont le tireur n'a aucune visée lue). */
  angle: number | null
}

/**
 * VehicleShotCaps — LES DEUX CAPS D'UN TIR EN VÉHICULE, et ils ne servent pas à la même chose.
 *
 * `chassisDeg` est le cap auquel le SPRITE EST DESSINÉ (`vehicleChassisHeadingAt`) : le montage
 * est une ancre dans le repère LOCAL du sprite, donc c'est lui, et lui seul, qui place l'éclair.
 *
 * `tireurDeg` est la visée MESURÉE de celui qui a tiré (`vehicleShooterAimAt`, apparié par SLOT),
 * ou `null` quand aucune lecture n'est en vigueur. Il ne sert qu'à ORIENTER la décharge d'une
 * TOURELLE — là où le châssis ne dit rien, parce que la tourelle tourne indépendamment du corps.
 * Les deux restent distincts à dessein : confondre le placement et la direction sortirait l'éclair
 * du châssis qu'il est censé quitter.
 */
export interface VehicleShotCaps {
  chassisDeg: number
  tireurDeg: number | null
  /**
   * CE QUI A TIRÉ (lot 5.8.5) : une arme DE VÉHICULE est solidaire du châssis, donc son cap est un
   * repli légitime quand le montage manque ; une arme DE JOUEUR tirée d'un siège ne l'est pas —
   * elle n'accepte que la visée mesurée de son tireur (règle du lot 5.2a.5, qui survit).
   */
  arme: 'vehicule' | 'joueur'
}

/**
 * vehicleShotPlacement — LE calcul, et rien d'autre : ancre (fractions du sprite) + cap monde +
 * taille du sprite -> décalage écran + direction. RÉUTILISE LES PRIMITIVES DU CALQUE VÉHICULES,
 * NE RECOPIE AUCUNE TRIGONOMÉTRIE :
 *  - `vehicleScreenAngle` pour tourner le décalage LOCAL (repère du sprite, nez en haut) EXACTEMENT
 *    comme `drawRotatedSprite` tourne l'image elle-même (même angle, même sens) ;
 *  - `vehicleSpriteScale` pour la même mise à l'échelle physique->écran que le sprite ;
 *  - `vehicleAimAngle` pour la direction d'un tir `fixe` — LE MÊME angle que le cône du
 *    conducteur (`drawVehicleAimCone`), pas une variante recalculée : « cap du véhicule = viseur »
 *    est une hypothèse déjà posée ailleurs dans ce calque, pas une nouvelle invention.
 *
 * Repère LOCAL du décalage AVANT rotation : `(ax * naturalWidthPx, ay * naturalHeightPx)` — le
 * même repère que `drawRotatedSprite` dessine son image (`drawImage(img, -w/2, -h/2, w, h)`),
 * donc une ancre au nez (`ay = -0,5`) tombe bien sur le bord haut du sprite, PAS le bas.
 *
 * LE CANON D'UNE TOURELLE : le pivot tourné par le cap du CHÂSSIS (il appartient au sprite), puis
 * `reach × longueur du sprite à l'écran` le long de la DIRECTION DE LA DÉCHARGE (la visée du
 * tireur). Sans visée lue, pas de direction : le tir reste au pivot.
 */
export function vehicleShotPlacement(
  mount: VehicleWeaponMount | null,
  caps: VehicleShotCaps,
  size: VehicleMountSpriteSize,
  k: number,
  scalePxPerM: number,
): VehicleShotPlacement {
  // MONTAGE INCONNU : aucun décalage — l'éclair reste au CENTRE du châssis, la seule position
  // que le document donne —, mais la DIRECTION lui revient (cf. `vehicleMountAngle`).
  if (!mount) return { offset: { x: 0, y: 0 }, angle: vehicleSansMontageAngle(caps) }
  const scale = vehicleSpriteScale(size.naturalHeightPx, size.mmPerPx, scalePxPerM) * k
  const localX = mount.ax * size.naturalWidthPx
  const localY = mount.ay * size.naturalHeightPx
  const screenAngle = vehicleScreenAngle(caps.chassisDeg)
  const cosA = Math.cos(screenAngle)
  const sinA = Math.sin(screenAngle)
  const angle = vehicleMountAngle(mount, caps)
  // LA BOUCHE D'UNE TOURELLE VISÉE : le canon, de la longueur mesurée, dans la direction du tir.
  const barrel = tourelleBarrelOffset(mount, angle, size.naturalHeightPx * scale)
  const offset: XY = {
    x: (localX * cosA - localY * sinA) * scale + barrel.x,
    y: (localX * sinA + localY * cosA) * scale + barrel.y,
  }
  return { offset, angle }
}

/**
 * tourelleBarrelOffset — le canon d'une TOURELLE à l'écran : `reach × longueur du sprite` le long
 * de la direction de la décharge. Nul pour une arme fixe (son ancre est la bouche) et pour une
 * tourelle sans visée lue (aucune direction, le tir reste au pivot).
 */
function tourelleBarrelOffset(mount: VehicleWeaponMount, angle: number | null, spriteLengthPx: number): XY {
  if (mount.classe !== 'tourelle' || angle === null) return { x: 0, y: 0 }
  const length = (mount.reach ?? 0) * spriteLengthPx
  return { x: length * Math.cos(angle), y: length * Math.sin(angle) }
}

/**
 * vehicleMountAngle — LA DIRECTION D'UNE DÉCHARGE, montage par montage. UN SEUL FOYER pour les
 * deux appelants (`vehicleShotPlacement` avec le sprite, `vehicleShotOrigin` sans) : la règle y
 * était écrite DEUX FOIS, et c'est exactement ce qui la ferait re-diverger (CLAUDE.md n° 6).
 *
 *   - `fixe` : le cap du CHÂSSIS, inchangé — sur ces familles l'arme ne tourne pas par rapport au
 *     corps, et depuis 5.2a.6 ce cap EST déjà la visée du conducteur (écart mesuré 0,0 degré).
 *   - `tourelle` : la visée MESURÉE du tireur quand elle est en vigueur (lot 5.5), sinon `null` —
 *     le repli d'avant, la bouffée ronde, parce qu'une tourelle ne pointe pas là où le nez pointe.
 */
function vehicleMountAngle(mount: VehicleWeaponMount, caps: VehicleShotCaps): number | null {
  if (mount.classe !== 'tourelle') return vehicleAimAngle(caps.chassisDeg)
  return caps.tireurDeg === null ? null : vehicleAimAngle(caps.tireurDeg)
}

/**
 * vehicleSansMontageAngle — LA DIRECTION QUAND AUCUN MONTAGE N'EST DOCUMENTÉ, et elle dépend de
 * CE QUI A TIRÉ (lot 5.8.5).
 *
 *   - ARME DE VÉHICULE (le Shade, dont le tag `weap` manque) : le cap du CHÂSSIS, inchangé depuis
 *     5.2a.5 — l'arme est solidaire du corps, c'est la seule direction que le film lui donne.
 *   - ARME DE JOUEUR tirée d'un siège : la visée MESURÉE de son tireur, et RIEN d'autre. Le cap du
 *     châssis lui serait une invention (règle du lot 5.2a.5, qui survit entière) ; sans lecture,
 *     la source n'est pas même créée (`shotFx.vehicleShotSourceOf`), donc ce `null` ne se produit
 *     qu'en appel direct.
 */
function vehicleSansMontageAngle(caps: VehicleShotCaps): number | null {
  if (caps.arme === 'vehicule') return vehicleAimAngle(caps.chassisDeg)
  return caps.tireurDeg === null ? null : vehicleAimAngle(caps.tireurDeg)
}

/**
 * vehicleShotOrigin — LE POINT ET LA DIRECTION D'UN ÉCLAIR, tir en véhicule compris
 * (2026-09-03). Posé ICI (pas dans `replayDraw.ts`, déjà au plafond de taille du dépôt,
 * CLAUDE.md n°5) : la composition repli/montage appartient au même fichier que la géométrie
 * qu'elle appelle.
 *
 * REPLI PAR DÉFAUT (`vehicleShot === null`, l'immense majorité des tirs) : le centre déjà
 * projeté, la direction du REGARD relu dans la trajectoire (`h`, cf. shotFx.ts) — inchangé. C'est
 * là que retombe encore une arme de JOUEUR tirée d'un siège dont la visée n'est pas lue (5.8.5).
 *
 * TIR EN VÉHICULE AVEC MONTAGE CONNU : la demande utilisateur du 2026-09-03, mot pour mot —
 * « si ça vient du passager, faut que le tir vienne du siège passager qui a une tourelle » —
 * plutôt que du centre du véhicule (cf. `document.go` : `x`/`y` du tir sont l'origine
 * INTERPOLÉE DU VÉHICULE). `vehicleShotPlacement` fait le calcul ; cette fonction n'ajoute que
 * le repli SANS taille de sprite connue (chargement pas encore abouti, famille inconnue) :
 * le centre, mais avec la BONNE direction dès que la classe du montage est connue (`tourelle`
 * -> aucune, `fixe` -> le cap) — jamais un décalage inventé sans la taille réelle qui le
 * justifie.
 */
export function vehicleShotOrigin(args: {
  h: number | null
  vehicleShot: {
    mount: VehicleWeaponMount | null
    /** Ce qui a tiré (lot 5.8.5) — cf. `VehicleShotCaps.arme`. */
    arme: 'vehicule' | 'joueur'
    family: string | undefined
    headingDeg: number
    /** Visée MESURÉE du tireur (lot 5.5), `null` si aucune lecture en vigueur. */
    shooterHeadingDeg: number | null
  } | null
  center: XY
  sizeOf: ((family: string) => VehicleMountSpriteSize | null) | undefined
  k: number
  /** L'échelle du cadrage (pixels CSS par mètre) : le montage suit la taille du sprite. */
  scalePxPerM: number
}): { origin: XY; angle: number | null } {
  const { h, vehicleShot, center, sizeOf, k } = args
  if (!vehicleShot) return { origin: center, angle: h === null ? null : (-h * Math.PI) / 180 }
  const { mount, arme, family, headingDeg, shooterHeadingDeg } = vehicleShot
  const caps: VehicleShotCaps = { chassisDeg: headingDeg, tireurDeg: shooterHeadingDeg, arme }
  const size = family ? sizeOf?.(family) : null
  // LA DIRECTION SANS LE SPRITE : même règle qu'avec lui (`vehicleMountAngle`), seul le DÉCALAGE
  // manque — il exige la taille réelle et ne s'invente pas. Un montage INCONNU (2026-09-20) prend
  // le cap du véhicule ; une TOURELLE prend la visée de son tireur (lot 5.5) ou rien.
  if (!size) {
    return {
      origin: center,
      angle: mount ? vehicleMountAngle(mount, caps) : vehicleSansMontageAngle(caps),
    }
  }
  const { offset, angle } = vehicleShotPlacement(mount, caps, size, k, args.scalePxPerM)
  return { origin: { x: center.x + offset.x, y: center.y + offset.y }, angle }
}
