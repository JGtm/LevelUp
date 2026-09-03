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
 * QUEL IDENTIFIANT INDEXE CETTE TABLE — CORRIGÉ EN COURS DE LOT, ET IL FAUT LE DIRE. La
 * première version indexait par les tags `jpt!` de dégâts de `labels.tsv` (8 chiffres hex,
 * "ARME DE VEHICULE (vehi ..., classe fixed/turret)"). Une VÉRIFICATION SUR PIÈCES contre le
 * document réel servi par le serveur local (`GET .../matches/0d76e8f1/replay`, 23 tirs `v`)
 * a montré que c'était le MAUVAIS espace : `Shot.w` (documenté `document.go` : « clé de
 * metadata.weapon_labels.weapon_id ») est un identifiant 64 BITS distinct du tag `jpt!` — sur
 * les 4 valeurs `w` réellement portées par les 23 tirs `v` de cet artefact, DEUX sont même des
 * ARMES DE JOUEUR (`Disrupteur`, `CQS48 Bulldog` : un passager qui tire sa PROPRE arme depuis
 * son siège, sans montage à affirmer — repli centre, correct), et les DEUX AUTRES
 * (`0xC7D5091200000000`, `0x11725DC400000000`) sont ABSENTES de `weaponLabels` — ce sont les
 * VRAIES armes de véhicule. Leur moitié haute (`c7d50912`, `11725dc4`) est EXACTEMENT les tags
 * `weap` que `V3F_TIRS_COVENANT_2026-09-02.md` identifie pour le Warthog et le Wasp (§4,
 * « témoin de méthode ») — un espace d'identifiants TOTALEMENT différent de `labels.tsv`
 * (lui-même construit pour le RE de l'arme du kill, jamais pour `Shot.w`). Cette table indexe
 * donc par `0x<weap 8 hex majuscules>00000000` — le gabarit observé sur les deux occurrences
 * vérifiées en direct, moitié basse nulle sur les deux (aucune arme de véhicule n'a de variante
 * de loadout, à la différence des armes de joueur qui portent un suffixe de variante non nul).
 *
 * CE QUI EN DÉCOULE, HONNÊTEMENT : SEULES LES ARMES DE VÉHICULE DONT LE TAG `weap` EST DOCUMENTÉ
 * PAR UN RAPPORT DE RE PEUVENT ENTRER ICI — pas de tag `weap` retrouvé = pas d'entrée, JAMAIS un
 * tag de dégâts `jpt!` réemployé en devinant qu'il coïnciderait. `V3F_TIRS_COVENANT_2026-09-02
 * .md` en documente SEPT (Ghost, Banshee ×2, Chopper, Scorpion, Wasp ×2) plus le Warthog
 * (témoin, un seul, non départagé LAAG/Gauss/Roquettes) ; AUCUN pour le Gungoose, le Shade ou
 * la tourelle LMG posée au sol — ces trois-là restent donc sur le repli centre (comportement
 * d'avant ce fichier), et c'est noté, pas caché.
 *
 * CLASSE, INDÉPENDANTE DE LA SOURCE DU TAG. `fixe` = solidaire du nez, ne peut viser qu'où le
 * véhicule pointe (Ghost, Banshee, Wasp, Chopper — véhicules MONOPLACES où le pilote EST le
 * viseur, l'approximation déjà faite par le cône du conducteur, `vehiclesLayer
 * .VEHICLE_DEFAULT_HEADING_DEG` : cap du véhicule = direction de tir, faute de mieux). `tourelle`
 * = visée manœuvrée INDÉPENDAMMENT du cap (canon arrière du Warthog, canon principal du
 * Scorpion — son plateau tourne séparément des chenilles, comme un char réel) : même principe
 * que `vehiclesLayer.vehicleDriverAt` — « la visée d'un passager ou d'un tourelleur est
 * INDÉPENDANTE de celle du véhicule, elle est inconnue, et rien ne la remplace » — d'où
 * `angle: null` pour ces montages (bouffée ronde, jamais une direction inventée).
 *
 * DEUX NIVEAUX DE PRÉCISION SUR LA POSITION, ET C'EST DIT PARTOUT CI-DESSOUS : le montage du
 * Warthog est MESURÉ (rectangle détecté sur le rendu du modèle, `WARTHOG_FINAL_V2_2026-09-02
 * .md`). Les montages Ghost / Banshee / Wasp / Chopper / Scorpion sont des ESTIMATIONS
 * VISUELLES (aucun rapport ne mesure leur position 3D à ce jour) — l'effet part du bon CÔTÉ du
 * véhicule (nez, canons jumeaux, plateau de tourelle) plutôt que du centre, sans prétendre au
 * millimètre.
 */
import { vehicleAimAngle, vehicleScreenAngle, vehicleSpriteScale } from './vehiclesLayer'
import type { XY } from './replayLogic'

/** `fixe` = solidaire du nez, vise où pointe le véhicule. `tourelle` = visée indépendante, inconnue. */
export type VehicleMountClass = 'fixe' | 'tourelle'

/** Une ancre de montage : classe de visée + position en fractions du sprite (repère nez en haut). */
export interface VehicleWeaponMount {
  classe: VehicleMountClass
  /** Latéral : -0,5 (bord gauche du sprite) .. +0,5 (bord droit). */
  ax: number
  /** Longitudinal : -0,5 (nez) .. +0,5 (arrière). */
  ay: number
}

/**
 * vehicleWeapTag — LE GABARIT DE `Shot.w` POUR UNE ARME DE VÉHICULE : `0x` + le tag `weap` de
 * 8 chiffres hex (tel que publié par `V3F_TIRS_COVENANT_2026-09-02.md`), majuscules, complété
 * par 32 bits nuls — VÉRIFIÉ sur les deux seules occurrences observables en direct (Warthog
 * `c7d50912`, Wasp `11725dc4`, artefact `0d76e8f1`). Centralisé ici pour que la table ci-dessous
 * cite le tag `weap` TEL QUE le rapport l'écrit, sans recopier le gabarit à la main N fois.
 */
function vehicleWeapTag(weap8hex: string): string {
  return `0x${weap8hex.toUpperCase()}00000000`
}

// --- ANCRES (une par arme de véhicule documentée) -----------------------------------------------

/**
 * WARTHOG_REAR_MOUNT — MESURÉ. `WARTHOG_FINAL_V2_2026-09-02.md` §§1-2 : chassis Warthog/
 * Rockethog/Razorback/Gauss-hog 2,10 m de long, **+X modèle = AVANT** (décision utilisateur,
 * §0). Rectangle du plateau arrière détecté à X local -0,585 m (centre, Y = 0,000 exact) —
 * soit ~26 % de la demi-longueur (1,05 m) VERS L'ARRIÈRE, donc `ay = +0,26` dans notre
 * convention (+0,5 = arrière). `ax = 0` : le rectangle est centré en Y (candidat 1 seul,
 * robuste à l'érosion 0/1/2 px). MÊME POINT pour les trois armes (LAAG, Gauss, roquettes) —
 * le rapport le dit explicitement : « Même (cx, cy) pour les trois armes ». Le seul tag
 * `weap` retrouvé (`c7d50912`, témoin V3F §4) ne départage pas laquelle des trois a tiré ;
 * sans conséquence ici puisque le point d'ancrage est identique pour les trois.
 */
const WARTHOG_REAR_MOUNT: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: 0.26 }

/** ESTIMATION — canons jumeaux de nez du Ghost (fourche avant, pas de rapport géométrique). */
const GHOST_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.22, ay: -0.35 }

/** ESTIMATION — canons à plasma jumeaux de la Banshee (mode M1, ailerons avant). */
const BANSHEE_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.3, ay: -0.3 }

/** ESTIMATION — tir lourd unique de la Banshee (mode M2, sous le nez, arme centrale). */
const BANSHEE_CENTER_HEAVY: VehicleWeaponMount = { classe: 'fixe', ax: 0, ay: -0.4 }

/** ESTIMATION — autocanon de menton du Wasp (mode M1, `11725dc4`, confirmé tiré dans l'artefact). */
const WASP_CHIN_AUTOCANNON: VehicleWeaponMount = { classe: 'fixe', ax: 0.08, ay: -0.4 }

/** ESTIMATION — second mode d'armement du Wasp (`d3c407ed`, documenté, pas observé en direct). */
const WASP_WING_ROCKETS: VehicleWeaponMount = { classe: 'fixe', ax: 0.3, ay: -0.05 }

/** ESTIMATION — canons jumeaux avant du Chopper, de part et d'autre de la grande roue. */
const CHOPPER_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.28, ay: -0.35 }

/**
 * SCORPION_TURRET — ESTIMATION DE POSITION. Le canon principal tourne sur un plateau
 * INDÉPENDANT des chenilles (comme un char réel) : sa visée diverge du cap du châssis, donc
 * `tourelle` (non-directionnel). Position : centre du plateau, léger biais avant (estimation).
 */
const SCORPION_TURRET: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: -0.05 }

// --- TABLE : `Shot.w` -> montage ----------------------------------------------------------------

/**
 * VEHICLE_WEAPON_MOUNTS — un tag `weap` par arme de véhicule DOCUMENTÉE (source : le tableau
 * ci-dessus). AUCUNE entrée « partagée entre familles » ici : chaque tag `weap` appartient à
 * UNE seule arme d'UNE seule famille (à la différence des tags `jpt!` de dégâts, qui peuvent
 * être réutilisés entre variantes) — le risque d'ambiguïté qui justifiait des exclusions dans
 * la V1 de ce fichier ne se pose donc plus.
 */
const VEHICLE_WEAPON_MOUNTS: ReadonlyMap<string, VehicleWeaponMount> = new Map([
  [vehicleWeapTag('c7d50912'), WARTHOG_REAR_MOUNT], // Warthog (témoin, LAAG/Gauss/Roquettes non départagés).
  [vehicleWeapTag('00015435'), GHOST_NOSE_TWIN], // Ghost — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa68'), BANSHEE_NOSE_TWIN], // Banshee M1 — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa69'), BANSHEE_CENTER_HEAVY], // Banshee M2 — tir lourd unique.
  [vehicleWeapTag('11725dc4'), WASP_CHIN_AUTOCANNON], // Wasp M1 — confirmé tiré, artefact 0d76e8f1.
  [vehicleWeapTag('d3c407ed'), WASP_WING_ROCKETS], // Wasp M2 — documenté, pas observé en direct.
  [vehicleWeapTag('b40e9618'), CHOPPER_NOSE_TWIN], // Chopper — canons jumeaux avant.
  [vehicleWeapTag('00015cfa'), SCORPION_TURRET], // Scorpion — canon principal.
])

/**
 * vehicleWeaponMountOf — le montage d'un tag d'arme (`Shot.w`), ou `null` : tir à pied, tir
 * d'une arme de JOUEUR tirée depuis un véhicule (un passager qui utilise son arme personnelle —
 * aucun montage à affirmer, cf. en-tête), ou arme de véhicule dont le tag `weap` n'est pas
 * documenté (Gungoose, Shade, tourelle LMG posée au sol). `null` = repli sur le centre du
 * véhicule, le comportement d'avant ce fichier — JAMAIS une position inventée.
 */
export function vehicleWeaponMountOf(tag: string | undefined): VehicleWeaponMount | null {
  if (!tag) return null
  return VEHICLE_WEAPON_MOUNTS.get(tag) ?? null
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
  /** Angle canevas de la décharge, ou `null` (tourelle : visée du tourelleur inconnue). */
  angle: number | null
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
 */
export function vehicleShotPlacement(
  mount: VehicleWeaponMount,
  headingDeg: number,
  size: VehicleMountSpriteSize,
  k: number,
): VehicleShotPlacement {
  const scale = vehicleSpriteScale(size.naturalHeightPx, size.mmPerPx) * k
  const localX = mount.ax * size.naturalWidthPx
  const localY = mount.ay * size.naturalHeightPx
  const screenAngle = vehicleScreenAngle(headingDeg)
  const cosA = Math.cos(screenAngle)
  const sinA = Math.sin(screenAngle)
  const offset: XY = {
    x: (localX * cosA - localY * sinA) * scale,
    y: (localX * sinA + localY * cosA) * scale,
  }
  const angle = mount.classe === 'tourelle' ? null : vehicleAimAngle(headingDeg)
  return { offset, angle }
}

/**
 * vehicleShotOrigin — LE POINT ET LA DIRECTION D'UN ÉCLAIR, tir en véhicule compris
 * (2026-09-03). Posé ICI (pas dans `replayDraw.ts`, déjà au plafond de taille du dépôt,
 * CLAUDE.md n°5) : la composition repli/montage appartient au même fichier que la géométrie
 * qu'elle appelle.
 *
 * REPLI PAR DÉFAUT (`vehicleShot === null`, l'immense majorité des tirs) : le centre déjà
 * projeté, la direction du REGARD relu dans la trajectoire (`h`, cf. shotFx.ts) — inchangé.
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
  vehicleShot: { mount: VehicleWeaponMount; family: string | undefined; headingDeg: number } | null
  center: XY
  sizeOf: ((family: string) => VehicleMountSpriteSize | null) | undefined
  k: number
}): { origin: XY; angle: number | null } {
  const { h, vehicleShot, center, sizeOf, k } = args
  if (!vehicleShot) return { origin: center, angle: h === null ? null : (-h * Math.PI) / 180 }
  const { mount, family, headingDeg } = vehicleShot
  const size = family ? sizeOf?.(family) : null
  if (!size) return { origin: center, angle: mount.classe === 'tourelle' ? null : vehicleAimAngle(headingDeg) }
  const { offset, angle } = vehicleShotPlacement(mount, headingDeg, size, k)
  return { origin: { x: center.x + offset.x, y: center.y + offset.y }, angle }
}
