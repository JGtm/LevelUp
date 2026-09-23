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
 * CE QUI EN DÉCOULE, HONNÊTEMENT : SEULES LES ARMES DE VÉHICULE DONT LE TAG `weap` EST CONNU PEUVENT
 * ENTRER ICI — pas de tag `weap` retrouvé = pas d'entrée, JAMAIS un tag de dégâts `jpt!` réemployé
 * en devinant qu'il coïnciderait. Un tag est connu par DEUX voies, et deux seulement : un RAPPORT
 * DE RE (`V3F_TIRS_COVENANT_2026-09-02.md` en documente sept — Ghost, Banshee ×2, Chopper, Wasp ×2,
 * et `00015cfa` pour le Scorpion, jamais publié — ; `WARTHOG_FINAL_2026-09-02.md` §1 donne
 * `c7d50912`, le LANCE-ROQUETTES du Rockethog, la LAAG étant `0c6fd911` et le Gauss `8647925a`) ;
 * ou l'OBSERVATION datée du parc (retours du 2026-09-23, lot L1.5 : `49e40d17`, le canon du
 * Scorpion tel que le film le publie, et `0bb6976b`, le lance-grenades du Falcon), tenue par la
 * fixture `test/fixtures/vehicle_weapon_tags_observed.json` et son garde-rail
 * `vehicleWeaponTags.guard.test.ts`.
 *
 * DEPUIS LE LOT 5.8.3 (2026-09-21), TROIS AUTRES ENTRENT PAR UNE MESURE DE SPRITE et non par un
 * rapport de RE — leur TAG, lui, était déjà connu (il sert le SON depuis le 2026-09-04) : le
 * Wraith (`121b4009`), le Gungoose (`0042678e`) et la tourelle LMG du Falcon (`00015cd3`). La
 * position vient de l'IMAGE, qui est le repère même où l'ancre se lit (cf. leur bloc). Le SHADE
 * reste dehors, et pour une autre raison : c'est son TAG qui manque, pas sa position — il garde
 * donc le repli centre, et c'est noté, pas caché.
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
}

/**
 * vehicleWeapTag — LE GABARIT DE `Shot.w` POUR UNE ARME DE VÉHICULE : `0x` + le tag `weap` de
 * 8 chiffres hex (tel que publié par `V3F_TIRS_COVENANT_2026-09-02.md`), majuscules, complété
 * par 32 bits nuls — VÉRIFIÉ d'abord sur les deux occurrences observables en direct le 2026-09-04
 * (`c7d50912`, alors dit « Warthog » — ce sont les roquettes du Rockethog —, et Wasp `11725dc4`,
 * artefact `0d76e8f1`), puis retrouvé porté par un véhicule sur les sept tags de ce gabarit que
 * publie le parc (fixture du 2026-09-23). Centralisé ici pour que la table ci-dessous cite le tag
 * `weap` TEL QUE le rapport l'écrit, sans recopier le gabarit à la main N fois.
 * EXPORTÉ depuis le lot sons de tir (2026-09-04) : `vehicleShotSound.ts` indexe sa table par le
 * MÊME gabarit — une deuxième copie du littéral aurait re-divergé (CLAUDE.md n° 6).
 */
export function vehicleWeapTag(weap8hex: string): string {
  return `0x${weap8hex.toUpperCase()}00000000`
}

// --- ANCRES (une par arme de véhicule connue) ---------------------------------------------------

/**
 * WARTHOG_REAR_MOUNT — MESURÉ. `WARTHOG_FINAL_V2_2026-09-02.md` §§1-2 : chassis Warthog/
 * Rockethog/Razorback/Gauss-hog 2,10 m de long, **+X modèle = AVANT** (décision utilisateur,
 * §0). Rectangle du plateau arrière détecté à X local -0,585 m (centre, Y = 0,000 exact) —
 * soit ~26 % de la demi-longueur (1,05 m) VERS L'ARRIÈRE, donc `ay = +0,26` dans notre
 * convention (+0,5 = arrière). `ax = 0` : le rectangle est centré en Y (candidat 1 seul,
 * robuste à l'érosion 0/1/2 px). MÊME POINT pour les trois armes (LAAG, Gauss, roquettes) —
 * le rapport le dit explicitement : « Même (cx, cy) pour les trois armes ». Le tag `c7d50912`
 * est celui des ROQUETTES (`WARTHOG_FINAL_2026-09-02.md` §1, erratum du 2026-09-23) ; la LAAG
 * (`0c6fd911`) et le Gauss (`8647925a`) n'ont jamais été observés dans un document.
 */
const WARTHOG_REAR_MOUNT: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: 0.26 }

/** ESTIMATION — canons jumeaux de nez du Ghost (fourche avant, pas de rapport géométrique). */
const GHOST_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.22, ay: -0.35 }

/** ESTIMATION — canons à plasma jumeaux de la Banshee (mode M1, ailerons avant). */
const BANSHEE_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.3, ay: -0.3 }

/** ESTIMATION — tir lourd unique de la Banshee (mode M2, sous le nez, arme centrale). */
const BANSHEE_CENTER_HEAVY: VehicleWeaponMount = { classe: 'fixe', ax: 0, ay: -0.4 }

/**
 * LES DEUX MODES DU WASP — QUELLE ARME EST LAQUELLE N'EST PAS TRANCHÉ (écart signalé le 2026-09-23,
 * question Q11 des retours du rejeu, à trancher au lot M4a). Ce fichier a longtemps dit « M1 =
 * autocanon de menton, M2 = missiles d'aile » ; or `V3F_TIRS_COVENANT_2026-09-02.md` §4 donne
 * `11725dc4` = son AU COUP (`snd!`, 450/min) et `d3c407ed` = son EN BOUCLE (`lsnd`, 600/min) — une
 * boucle évoque plutôt une arme à tir continu (l'autocanon), un son au coup plutôt des missiles
 * (DÉDUIT, non mesuré). L'utilisateur a mené une recherche qui les distingue ; elle n'a pas été
 * retrouvée (`.ai/`, Notion). Les ancres ci-dessous sont nommées par leur PLACE (menton, aile),
 * pas par l'arme, et restent des estimations jusqu'à ce que l'attribution soit tranchée.
 */
/** ESTIMATION — menton du Wasp (mode M1, `11725dc4`, confirmé tiré dans l'artefact). */
const WASP_CHIN_MOUNT: VehicleWeaponMount = { classe: 'fixe', ax: 0.08, ay: -0.4 }

/** ESTIMATION — aile du Wasp (mode M2, `d3c407ed`, documenté, pas observé en direct). */
const WASP_WING_MOUNT: VehicleWeaponMount = { classe: 'fixe', ax: 0.3, ay: -0.05 }

/** ESTIMATION — canons jumeaux avant du Chopper, de part et d'autre de la grande roue. */
const CHOPPER_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.28, ay: -0.35 }

/**
 * SCORPION_TURRET — ESTIMATION DE POSITION. Le canon principal tourne sur un plateau
 * INDÉPENDANT des chenilles (comme un char réel) : sa visée diverge du cap du châssis, donc
 * `tourelle` (non-directionnel). Position : centre du plateau, léger biais avant (estimation).
 */
const SCORPION_TURRET: VehicleWeaponMount = { classe: 'tourelle', ax: 0, ay: -0.05 }

/**
 * LES TROIS MONTAGES MANQUANTS, MESURÉS SUR LE SPRITE (lot 5.8.3, 2026-09-21).
 *
 * POURQUOI ILS MANQUAIENT : aucun rapport de RE ne donne leur position 3D, et l'en-tête de ce
 * fichier l'écrivait en toutes lettres (« AUCUN pour le Gungoose, le Shade ou la tourelle LMG »).
 * Depuis 5.2a.5 un montage inconnu ne fait plus perdre la SOURCE, mais l'éclair part du CENTRE du
 * châssis — pour le Wraith, c'est-à-dire à un mètre et demi derrière la bouche de son mortier.
 *
 * LA MESURE EST CELLE DU SPRITE, ET C'EST EXACTEMENT LE BON REPÈRE : `vehicleShotPlacement` lit
 * `ax`/`ay` en fractions des dimensions PLEINES de l'image (`naturalWidthPx`/`naturalHeightPx`,
 * origine au centre, nez en haut). Mesurer sur l'image revient donc à mesurer sur ce que le
 * lecteur voit — il n'y a aucune conversion à croire. Les sprites sont ceux du manifeste du titre
 * (`static/vehicles-assets/halo_infinite/replay/`, 10 mm/px vérifiés, nez en haut depuis la
 * correction d'orientation du 2026-09-02) ; la détection est faite sur l'ENCRE du trait
 * (luminance < 110), l'alpha étant plein sur toute la silhouette.
 *
 * LA CLASSE NE SE MESURE PAS, ELLE EST DÉJÀ DÉCIDÉE : `FAMILLES_ARME_FIXE` (5.2a.6, décision
 * utilisateur du 2026-09-20) range le Wraith et le Gungoose parmi les châssis à arme FIXE — viser,
 * c'est tourner le véhicule — et le Falcon parmi les châssis à TOURELLE. Leur donner une autre
 * classe ici ferait dire deux choses au dépôt sur la même famille.
 */

/**
 * WRAITH_MORTAR_MUZZLE — MESURÉ sur `wraith.png` (304 × 313, centre 152,0 / 156,5). Le canon du
 * mortier à plasma est la paire de traits verticaux à x = 148 et 155 (donc un fût centré sur
 * x ≈ 151,5, soit l'axe du châssis à un demi-pixel près), qui court de y ≈ 96 à y ≈ 135 avant de
 * rejoindre l'ANNEAU de tourelle (le cercle mesuré à y ≈ 138..182). L'ancre est la BOUCHE, pas le
 * moyeu : c'est de là que la charge part. `ay = (95 - 156,5) / 313 = -0,196`.
 */
const WRAITH_MORTAR_MUZZLE: VehicleWeaponMount = { classe: 'fixe', ax: 0, ay: -0.2 }

/**
 * GUNGOOSE_NOSE_TWIN — MESURÉ sur `gungoose.png` (78 × 128, centre 39,0 / 64,0). Les canons
 * jumelés (l'objet `scen` 0x004164ea que le rapport `CONTACT_ARMES_GUNGOOSE_2026-09-02.md` pose
 * en zone AVANT) sont les deux amas d'encre séparés par un vide en x ∈ [36..41] : fût gauche
 * x ≈ 34, fût droit x ≈ 43,5, tous deux du haut y ≈ 13 jusqu'à y ≈ 36. L'ancre prend le fût DROIT,
 * comme le Ghost, la Banshee et le Chopper prennent un seul de leurs jumeaux — une paire dessinée
 * à son milieu n'aurait plus rien d'une paire. `ax = (43,5 - 39) / 78 = +0,058`,
 * `ay = (13 - 64) / 128 = -0,398`.
 */
const GUNGOOSE_NOSE_TWIN: VehicleWeaponMount = { classe: 'fixe', ax: 0.06, ay: -0.4 }

/**
 * FALCON_SIDE_POST — MESURÉ sur `falcon.png` (430 × 390, centre 215,0 / 195,0). Les deux postes
 * latéraux sont les caissons que l'encre ferme à y = 236 (arrêtes horizontales x 170..194 à gauche
 * et x 230..256 à droite) et qui descendent jusqu'à y ≈ 255 — de part et d'autre du fuselage, en
 * arrière du poste de pilotage, exactement là où le Falcon ouvre ses portes. L'ancre prend le
 * poste DROIT : `ax = (242 - 215) / 430 = +0,063`, `ay = (245 - 195) / 390 = +0,128`.
 *
 * LE LANCE-GRENADES (`0bb6976b`, décision Q10 du 2026-09-23) SORT DU MÊME POSTE DE PORTE : le
 * Falcon porte à cet endroit une LMG OU un lance-grenades, et c'est le tag du tir qui dit lequel.
 *
 * `tourelle`, ET C'EST LA DÉCISION DE 5.2a.6 : une mitrailleuse de porte est servie par un
 * PASSAGER et ne pointe pas où le nez pointe. Depuis 5.5.3 sa décharge prend donc la visée
 * MESURÉE de son tireur, et la bouffée ronde ne revient qu'à défaut de lecture.
 */
const FALCON_SIDE_POST: VehicleWeaponMount = { classe: 'tourelle', ax: 0.06, ay: 0.13 }

// --- TABLE : `Shot.w` -> montage ----------------------------------------------------------------

/**
 * VEHICLE_WEAPON_MOUNTS — un tag `weap` par arme de véhicule CONNUE (rapport de RE ou observation
 * datée du parc, cf. l'en-tête ; ancres : les blocs ci-dessus). EXPORTÉE pour le garde-rail des
 * tags observés (`vehicleWeaponTags.guard.test.ts` : toute clé est observée dans un document ou
 * attendue, retrait au lot M4a). AUCUNE entrée « partagée entre familles » ici : chaque tag
 * `weap` appartient à UNE seule arme d'UNE seule famille (à la différence des tags `jpt!` de
 * dégâts, qui peuvent être réutilisés entre variantes) — le risque d'ambiguïté qui justifiait des
 * exclusions dans la V1 de ce fichier ne se pose donc plus.
 */
export const VEHICLE_WEAPON_MOUNTS: ReadonlyMap<string, VehicleWeaponMount> = new Map([
  [vehicleWeapTag('c7d50912'), WARTHOG_REAR_MOUNT], // Rockethog — lance-roquettes (plateau arrière).
  [vehicleWeapTag('00015435'), GHOST_NOSE_TWIN], // Ghost — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa68'), BANSHEE_NOSE_TWIN], // Banshee M1 — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa69'), BANSHEE_CENTER_HEAVY], // Banshee M2 — tir lourd unique.
  [vehicleWeapTag('11725dc4'), WASP_CHIN_MOUNT], // Wasp M1 — confirmé tiré, artefact 0d76e8f1.
  [vehicleWeapTag('d3c407ed'), WASP_WING_MOUNT], // Wasp M2 — documenté, pas observé en direct.
  [vehicleWeapTag('b40e9618'), CHOPPER_NOSE_TWIN], // Chopper — canons jumeaux avant.
  [vehicleWeapTag('49e40d17'), SCORPION_TURRET], // Scorpion — canon principal (tag PUBLIÉ ; 00015cfa jamais vu).
  // LES TROIS DU LOT 5.8.3, mesurés sur le sprite faute de rapport de RE (cf. leur bloc).
  [vehicleWeapTag('121b4009'), WRAITH_MORTAR_MUZZLE], // Wraith — mortier à plasma.
  [vehicleWeapTag('0042678e'), GUNGOOSE_NOSE_TWIN], // Gungoose — mitrailleuses avant.
  [vehicleWeapTag('00015cd3'), FALCON_SIDE_POST], // Falcon — tourelle LMG de porte.
  [vehicleWeapTag('0bb6976b'), FALCON_SIDE_POST], // Falcon — lance-grenades de porte (Q10).
])

/**
 * vehicleWeaponMountOf — le montage d'un tag d'arme (`Shot.w`), ou `null` : tir à pied, tir
 * d'une arme de JOUEUR tirée depuis un véhicule (un passager qui utilise son arme personnelle —
 * aucun montage à affirmer, cf. en-tête), ou arme de véhicule dont le tag `weap` n'est pas
 * documenté (le Shade, dont aucun rapport ne donne le tag `weap`). `null` = repli sur le
 * centre du véhicule, le comportement d'avant ce fichier — JAMAIS une position inventée.
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
  const offset: XY = {
    x: (localX * cosA - localY * sinA) * scale,
    y: (localX * sinA + localY * cosA) * scale,
  }
  return { offset, angle: vehicleMountAngle(mount, caps) }
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
