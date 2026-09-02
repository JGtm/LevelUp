/**
 * vehiclesLayer.ts — LES VÉHICULES sur la carte du rejeu (schéma 29) : géométrie pure et tracé.
 *
 * CE QUE CE CALQUE AFFIRME. Une entrée de `doc.vehicles` est LA VIE D'UN VÉHICULE : où il naît,
 * sa trajectoire échantillonnée avec son cap, ses épisodes d'occupation (qui est à bord et
 * quand), et jusqu'à quelle frame l'afficher. La fin est une BORNE DE RECENSEMENT — `end` vaut
 * toujours `unknown` côté document — et ce calque ne dessine JAMAIS d'effet de destruction :
 * le sprite s'efface NETTEMENT à `t1max`, rien de plus (décision de cadrage du plan).
 *
 * QUATRE RESPONSABILITÉS PURES, TESTABLES SANS CANVAS : l'ORIENTATION (`vehicleHeadingAt` +
 * `vehicleScreenAngle`), la TAILLE (`vehicleSpriteScale`, ancrée sur le pion), l'OCCUPATION
 * (`vehicleActiveRides`, `vehicleColorAt`) et le PRÉDICAT EMBARQUÉ (`buildEmbarkedPredicate`,
 * consommé par `replayMarkers.drawTracksLayer` pour supprimer le pion d'un occupant). Le tracé
 * canvas (`drawVehiclesLayer`) ne fait qu'assembler ces quatre réponses image par image.
 *
 * ORIENTATION — LA CONSTANTE D'ÉCART D'ÉCRAN (GATE C6). Les échantillons portent un cap MONDE
 * (`VehicleSample.h`, convention `Point.h` : 0° = +X, 90° = +Y, sens `atan2(y,x)`), mais les
 * sprites sont dessinés NEZ EN HAUT — c'est-à-dire nez vers l'axe -Y LOCAL de l'image, jamais
 * vers +X comme les formes directionnelles du calque de tirs (`muzzleFlash.ts`, qui n'a besoin
 * que d'inverser le signe : `-h`). Un sprite « nez en haut » a donc besoin d'UN QUART DE TOUR DE
 * PLUS que cette inversion : la projection écran inverse l'axe Y du monde, donc le cap-écran
 * pour ORIENTER UNE IMAGE NEZ-HAUT est `(90° - h) `, converti en radians pour `ctx.rotate`.
 * Vérifié par `vehiclesLayer.test.ts` sur les quatre points cardinaux (est/nord/ouest/sud)
 * et RECOUPÉ sur données réelles (artefact `0d76e8f1` : cap publié 272,7° vs 273,2° recalculé
 * sur le déplacement, écart 0,5°). La preuve VISUELLE en jeu reste le gate D1 (utilisateur).
 *
 * TAILLE — DÉCISION DE CADRAGE DU PLAN (2026-09-02, utilisateur) : proportionnelle au monde
 * ENTRE véhicules (longueur de sprite en pixels × mm/px du manifeste), en pixels ÉCRAN FIXES
 * vis-à-vis du zoom (comme les pions), ancrée sur le Mongoose (≈ 1,5-2 pions de long). LE
 * MANIFESTE (`index.json`, servi par le lot A) NE GARANTIT AUJOURD'HUI l'échelle QUE DES
 * FAMILLES WARTHOG/MONGOOSE — les 12 autres portent la même valeur en attendant un re-rendu
 * mesuré (note explicite de la tâche de ce lot) : la règle lit le manifeste tel quel, et
 * DEVIENDRA JUSTE POUR CHAQUE FAMILLE quand SA valeur le sera, sans changement de code ici.
 */
import type { ReplayVehicleRide } from '@/lib/api/types'

import { drawRotatedSprite } from './replayDraw'
import { drawNameLabel, type LabelStyle } from './replayLabels'
import { CORE_RADIUS } from './replayMarkers'
import { lastIndexAt, positionAt, type XY } from './replayLogic'
import type { ReplayVehicleTrackReady } from './replayNormalize'
import { project, type PlacementView } from './placementShapes'
import { traceDiamond } from './weaponPadsLayer'

export type { PlacementView as VehicleView } from './placementShapes'

// --- ORIENTATION ----------------------------------------------------------------------------

/**
 * VEHICLE_DEFAULT_HEADING_DEG — le cap MONDE d'un véhicule dont AUCUN échantillon ne porte de
 * cap : ni mort (la tourelle Shade et la tourelle montée ne translatent jamais — décision de
 * cadrage « TOURELLES »), ni pas-encore-mobile. Choisi à 90° (monde +Y) précisément parce que
 * `vehicleScreenAngle(90) === 0` : AUCUNE rotation supplémentaire n'est appliquée, et le sprite
 * reste NEZ VERS LE HAUT DE L'ÉCRAN tel qu'il est dessiné — exactement la règle demandée.
 */
export const VEHICLE_DEFAULT_HEADING_DEG = 90

/**
 * vehicleHeadingAt — le cap MONDE du véhicule à `frame`, degrés, MÊME convention que `Point.h`.
 *
 * TROIS RÉGIMES, DANS L'ORDRE DE LA DÉCISION DE CADRAGE :
 *  1. EN MOUVEMENT / À L'ARRÊT — le dernier échantillon connu AU PLUS TARD à `frame` qui porte
 *     un cap (`h !== undefined`) : le serveur y écrit déjà la direction de la vélocité en
 *     mouvement, et REPORTE ce même cap sur les échantillons immobiles qui suivent (cf.
 *     `VehicleSample.H`) — chercher EN ARRIÈRE depuis `frame` couvre donc les deux à la fois,
 *     sans jamais interpoler un angle (l'interpolation d'un cap ferait tourner le véhicule par
 *     le chemin le plus court à travers 0°/360°, un artefact que le film ne montre pas).
 *  2. AVANT LE PREMIER MOUVEMENT — aucun échantillon EN ARRIÈRE ne porte de cap (le véhicule est
 *     encore à quai) : le cap du PREMIER échantillon mobile À VENIR, pour que le véhicule
 *     s'oriente déjà dans le sens qu'il va prendre plutôt que de pivoter brutalement au premier
 *     mouvement.
 *  3. JAMAIS MOBILE — aucun échantillon, nulle part, ne porte de cap (véhicule jamais conduit,
 *     ou tourelle qui ne translate pas) : `VEHICLE_DEFAULT_HEADING_DEG`, nez vers le haut.
 */
export function vehicleHeadingAt(track: ReplayVehicleTrackReady, frame: number): number {
  const samples = track.samples
  if (samples.length === 0) return VEHICLE_DEFAULT_HEADING_DEG
  const idx = lastIndexAt(samples, frame)
  for (let i = idx; i >= 0; i--) {
    const h = samples[i].h
    if (h !== undefined) return h
  }
  for (let i = Math.max(idx + 1, 0); i < samples.length; i++) {
    const h = samples[i].h
    if (h !== undefined) return h
  }
  return VEHICLE_DEFAULT_HEADING_DEG
}

/**
 * vehicleScreenAngle — LA CONSTANTE D'ÉCART D'ÉCRAN (gate C6) : convertit un cap MONDE en
 * radians CANEVAS pour un sprite dessiné NEZ EN HAUT. Voir l'en-tête du fichier pour la dérivation
 * complète ; le résultat tient en une ligne : `(90° − cap) → radians`.
 */
export function vehicleScreenAngle(headingDeg: number): number {
  return ((90 - headingDeg) * Math.PI) / 180
}

// --- POSITION ET FENÊTRE D'AFFICHAGE ---------------------------------------------------------

/**
 * vehicleVisibleAt — `t1max` est une PREUVE D'ABSENCE (cf. `document_vehicles.go`) : la franchir
 * affirmerait une présence que le film réfute. Décision de cadrage : « disparition NETTE » —
 * aucun estompage entre `t1`/`t1max` (à la différence des armes au sol) : le véhicule est plein
 * jusqu'à `t1max` inclus, rien après.
 */
export function vehicleVisibleAt(track: ReplayVehicleTrackReady, frame: number): boolean {
  return frame >= track.t0 && frame <= track.t1max
}

/**
 * vehiclePositionAt — la position MONDE à `frame`, ou `null` (rien à dessiner).
 *
 * `positionAt` (replayLogic.ts) tient déjà l'interpolation entre échantillons et le maintien de
 * la dernière position connue une fois le dernier échantillon dépassé — LA MÊME logique que pour
 * une trajectoire de joueur, réutilisée telle quelle (`VehicleSample` a le même sous-ensemble de
 * champs que `Point`). Elle rend `null` AVANT le premier échantillon : c'est alors le SPAWN qui
 * répond (la naissance, seule chose connue tant que personne n'a conduit le véhicule). Ni l'un
 * ni l'autre : le record de création n'a pas été lu ET aucun échantillon n'existe — rien ne se
 * dessine, plutôt que d'inventer une position.
 */
export function vehiclePositionAt(track: ReplayVehicleTrackReady, frame: number): XY | null {
  const fromSamples = track.samples.length > 0 ? positionAt(track.samples, frame) : null
  if (fromSamples) return fromSamples
  return track.spawn ? { x: track.spawn.x, y: track.spawn.y } : null
}

// --- TAILLE (décision de cadrage : ancrée sur le pion) ---------------------------------------

/**
 * PION_REFERENCE_PX — le noyau du marqueur joueur (`CORE_RADIUS`, replayMarkers.ts), SEULE
 * partie TOUJOURS visible d'un pion quel que soit son étage. C'est l'ancre de la règle de
 * taille : « pion = pixels fixes, CORE 3,4 + RING 6,5 px » (décision de cadrage).
 */
const PION_REFERENCE_PX = CORE_RADIUS * 2

/** Milieu de la fourchette demandée (1,5-2 pions de long pour le Mongoose). */
const MONGOOSE_TO_PION_RATIO = 1.75

/**
 * MONGOOSE_REFERENCE_LENGTH_MM — la longueur RÉELLE (nez-en-haut) du sprite Mongoose VALIDÉ du
 * lot A (`V4_RAPPORT_SPRITES_2026-08-31.md`, statut "valide") : 128 px de sprite × 10 mm/px.
 * C'EST, AVEC LE WARTHOG, LA SEULE FAMILLE DONT L'ÉCHELLE EST GARANTIE (note de la tâche de ce
 * lot) : l'ancre de la conversion mm → pixel-écran se calibre donc sur elle, jamais sur un
 * sprite dont l'échelle n'est pas mesurée — ce chiffre n'a PAS besoin de charger le sprite
 * Mongoose au runtime, il est dérivé une fois, ici, de sa mesure connue.
 */
const MONGOOSE_REFERENCE_LENGTH_MM = 1280

/**
 * VEHICLE_PX_PER_MM — LA CONSTANTE NOMMÉE UNIQUE de la règle de taille (décision de cadrage) :
 * un millimètre-monde vaut CE NOMBRE de pixels-écran, POUR TOUTE FAMILLE. Parce qu'elle est
 * UNIQUE et appliquée à `naturalHeightPx × mmPerPx` (une propriété du COUPLE sprite×manifeste,
 * jamais couplée à une famille précise dans la formule), les tailles RELATIVES entre véhicules
 * suivent le manifeste : le jour où les 12 familles non garanties reçoivent leur propre mm/px
 * mesuré, leur taille à l'écran devient juste SANS toucher à cette constante.
 */
const VEHICLE_PX_PER_MM = (PION_REFERENCE_PX * MONGOOSE_TO_PION_RATIO) / MONGOOSE_REFERENCE_LENGTH_MM

/** Plancher de lisibilité : aucun véhicule ne descend sous le noyau d'un pion. */
export const VEHICLE_FLOOR_PX = PION_REFERENCE_PX

/**
 * Plafond DOUX : au-delà, la croissance ralentit (racine carrée) au lieu de s'arrêter net — un
 * véhicule très long reste visiblement plus grand qu'un plus petit, mais cesse de dominer
 * l'écran. Choisi à 4× la cible du Mongoose : rien du corpus mesuré aujourd'hui (Warthog,
 * Mongoose, mm/px identique) ne l'atteint — il protège la lecture pour le jour où une famille
 * hors gabarit (le Pelican, dropship, nommé par la décision de cadrage) recevra sa propre
 * mesure et non plus la valeur provisoire des 12 familles non garanties.
 */
export const VEHICLE_SOFT_CEIL_PX = 4 * MONGOOSE_TO_PION_RATIO * PION_REFERENCE_PX

/**
 * vehicleScreenLengthPx — la longueur d'écran (pixels CSS fixes, avant densité `k`) d'un
 * véhicule, plancher et plafond doux appliqués. `naturalHeightPx` est la hauteur NATIVE du
 * sprite chargé (nez-en-haut : la hauteur EST l'axe de longueur du véhicule) ; `mmPerPx` vient
 * du manifeste (`index.json`) pour SA famille.
 */
export function vehicleScreenLengthPx(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0 || mmPerPx <= 0) return 0
  const raw = naturalHeightPx * mmPerPx * VEHICLE_PX_PER_MM
  const floored = Math.max(raw, VEHICLE_FLOOR_PX)
  return floored <= VEHICLE_SOFT_CEIL_PX
    ? floored
    : VEHICLE_SOFT_CEIL_PX + Math.sqrt(floored - VEHICLE_SOFT_CEIL_PX)
}

/**
 * vehicleSpriteScale — le facteur multiplicatif à donner à `drawRotatedSprite` (avant densité
 * `k`, que l'appelant applique en plus) pour que le sprite atteigne `vehicleScreenLengthPx` sur
 * son axe de hauteur, aspect ratio préservé.
 */
export function vehicleSpriteScale(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0) return 0
  return vehicleScreenLengthPx(naturalHeightPx, mmPerPx) / naturalHeightPx
}

/** Demi-diagonale du petit losange neutre d'un châssis non résolu — le noyau d'un pion. */
const VEHICLE_UNKNOWN_HALF_PX = CORE_RADIUS

// --- OCCUPATION -------------------------------------------------------------------------------

/**
 * vehicleActiveRides — les épisodes d'occupation qui couvrent `frame`, TRIÉS conducteur
 * d'abord (siège 0) puis sièges croissants (décision de cadrage C7). Un siège NON LU (`seat`
 * absent) passe en dernier, à égalité entre eux dans l'ordre du document — jamais avant un
 * siège connu, dont la place est, elle, affirmée.
 */
export function vehicleActiveRides(
  track: ReplayVehicleTrackReady,
  frame: number,
): ReplayVehicleRide[] {
  return track.rides
    .filter((r) => frame >= r.t0 && frame <= r.t1)
    .sort((a, b) => (a.seat ?? Number.POSITIVE_INFINITY) - (b.seat ?? Number.POSITIVE_INFINITY))
}

/**
 * vehicleColorAt — la teinte du véhicule à `frame` (décision de cadrage C7) : la couleur du
 * CONDUCTEUR (siège 0) quand il est actif et son identité résolue ; À DÉFAUT, celle de
 * N'IMPORTE QUEL occupant actif dont l'identité est connue ; sinon `null` (neutre — l'appelant
 * y pose son encre de thème, ce fichier ne connaît aucun token, cf. règle color-tokens).
 */
export function vehicleColorAt(
  track: ReplayVehicleTrackReady,
  frame: number,
  colorOfSlot: (slot: number, frame: number) => string | null,
): string | null {
  const active = vehicleActiveRides(track, frame)
  const driver = active.find((r) => r.seat === 0)
  if (driver) {
    const c = colorOfSlot(driver.slot, frame)
    if (c) return c
  }
  for (const r of active) {
    const c = colorOfSlot(r.slot, frame)
    if (c) return c
  }
  return null
}

/**
 * buildEmbarkedPredicate — LE PRÉDICAT « EMBARQUÉ À T » (C7, rappel utilisateur explicite :
 * MULTI-PASSAGERS). Regroupe TOUS les épisodes d'occupation de TOUS les véhicules du document
 * par slot de bipède UNE SEULE FOIS, puis répond en O(occupations de ce slot) — quelques
 * unités par joueur sur un match entier, jamais un balayage de tous les véhicules par image.
 *
 * SORTIES INDÉPENDANTES PAR CONSTRUCTION : chaque `VehicleRide` porte SON PROPRE `[t0,t1]`, donc
 * deux occupants du MÊME véhicule (conducteur + passagers, Warthog 3 places, Razorback 4...)
 * reprennent leur pion chacun à SA frame de sortie, sans dépendre l'un de l'autre — c'est le sens
 * même de « plusieurs épisodes SIMULTANÉS », pas une propriété qu'il faut recoder ici.
 */
export function buildEmbarkedPredicate(
  tracks: readonly ReplayVehicleTrackReady[],
): (slot: number, frame: number) => boolean {
  const bySlot = new Map<number, ReplayVehicleRide[]>()
  for (const track of tracks) {
    for (const ride of track.rides) {
      const list = bySlot.get(ride.slot)
      if (list) list.push(ride)
      else bySlot.set(ride.slot, [ride])
    }
  }
  return (slot, frame) => {
    const list = bySlot.get(slot)
    if (!list) return false
    return list.some((r) => frame >= r.t0 && frame <= r.t1)
  }
}

// --- TRACÉ --------------------------------------------------------------------------------

/** Ce que le calque a besoin de savoir de l'instant courant. */
export interface VehicleTime {
  frame: number
  /** Densité de pixels : toutes les tailles d'écran de ce fichier la suivent. */
  k: number
}

/** Dimensions natives + échelle manifeste d'UNE famille — indépendant de la teinte. */
export interface VehicleSpriteSize {
  naturalHeightPx: number
  mmPerPx: number
}

/** Ce que le calque emprunte au thème, au document et aux vignettes déjà cuites. */
export interface VehicleStyle {
  /** Encre du « aucun occupant connu » (token sémantique, résolu par l'appelant). */
  neutralInk: string
  /** Encre du CONTOUR des noms — même contrat que `replayMarkers`/`replayLabels`. */
  labelStroke: string
  /** Calque des NOMS (bouton « Noms » partagé avec les pions, décision de cadrage). */
  showNames: boolean
  /**
   * Vignette DÉJÀ TEINTE (family × couleur résolue), ou `null` — l'image source ou sa teinte
   * n'ont pas encore fini de charger : RIEN NE LA REMPLACE, elle apparaît après coup (même
   * contrat que les vignettes de socle, `useReplayWeaponPads`).
   */
  spriteOf: (family: string, color: string) => CanvasImageSource | null
  /** Dimensions natives + mm/px du manifeste pour une famille, ou `null` si pas encore chargées. */
  sizeOf: (family: string) => VehicleSpriteSize | null
  colorOfSlot: (slot: number, frame: number) => string | null
  nameOfSlot: (slot: number, frame: number) => string | null
}

/** Écart entre deux noms empilés, en pixels d'écran (police partagée avec `replayLabels.ts`). */
const VEHICLE_NAME_LINE_STEP_PX = 10

/**
 * drawUnknownVehicleMarker — LE CHÂSSIS NON RÉSOLU : un petit losange neutre, JAMAIS le sprite
 * d'un véhicule voisin (décision de cadrage). Même vocabulaire que les socles (`weaponPadsLayer
 * .traceDiamond`, réutilisée) : un losange dit « objet de la carte, pas un joueur ».
 */
function drawUnknownVehicleMarker(ctx: CanvasRenderingContext2D, c: XY, color: string, k: number): void {
  ctx.globalAlpha = 1
  ctx.fillStyle = color
  traceDiamond(ctx, c, VEHICLE_UNKNOWN_HALF_PX * k)
  ctx.fill()
}

/**
 * drawVehicleOccupantNames — LES NOMS EMPILÉS d'un véhicule (C7) : conducteur en premier, puis
 * passagers par siège croissant (`vehicleActiveRides` a déjà trié). CHAQUE NOM PORTE SA PROPRE
 * couleur d'occupant (comme un pion) — ce sont, en pratique, presque toujours la même équipe,
 * mais rien ici ne force une teinte unique. Un occupant dont l'identité n'est pas résolue
 * (`nameOfSlot` rend `null`) SAUTE SA LIGNE plutôt que de laisser un blanc.
 */
function drawVehicleOccupantNames(
  ctx: CanvasRenderingContext2D,
  rides: readonly ReplayVehicleRide[],
  frame: number,
  c: XY,
  baseEdgePx: number,
  style: Pick<VehicleStyle, 'nameOfSlot' | 'colorOfSlot' | 'labelStroke'>,
  k: number,
): void {
  const label: LabelStyle = { k, labelStroke: style.labelStroke }
  let line = 0
  for (const ride of rides) {
    const name = style.nameOfSlot(ride.slot, frame)
    if (!name) continue
    const color = style.colorOfSlot(ride.slot, frame) ?? style.labelStroke
    drawNameLabel(ctx, c, name, label, color, baseEdgePx + VEHICLE_NAME_LINE_STEP_PX * line * k)
    line++
  }
}

/**
 * drawVehiclesLayer trace TOUS les véhicules VISIBLES à l'image courante.
 *
 * ORDRE DU DOCUMENT, comme les armes au sol : aucun arbitrage de recouvrement n'est fait ici.
 */
export function drawVehiclesLayer(
  ctx: CanvasRenderingContext2D,
  tracks: readonly ReplayVehicleTrackReady[],
  view: PlacementView,
  time: VehicleTime,
  style: VehicleStyle,
): void {
  if (tracks.length === 0 || view.width === 0) return
  ctx.save()
  for (const track of tracks) {
    if (!vehicleVisibleAt(track, time.frame)) continue
    const world = vehiclePositionAt(track, time.frame)
    if (!world) continue
    const c = project(world, view)
    const color = vehicleColorAt(track, time.frame, style.colorOfSlot) ?? style.neutralInk
    // ÉDGE PAR DÉFAUT (chassis non résolu, ou vignette pas encore chargée) : le plancher de
    // lisibilité, seule mesure disponible avant qu'une taille réelle ne soit connue. TOUJOURS
    // MULTIPLIÉ PAR `time.k`, comme la branche sprite juste en dessous — un pixel d'écran
    // déclaré ici n'a de sens qu'à la densité du périphérique (même règle que `replayMarkers`).
    let edgePx = (VEHICLE_FLOOR_PX / 2) * time.k
    if (!track.family) {
      drawUnknownVehicleMarker(ctx, c, color, time.k)
      edgePx = VEHICLE_UNKNOWN_HALF_PX * time.k
    } else {
      const size = style.sizeOf(track.family)
      const sprite = size ? style.spriteOf(track.family, color) : null
      if (size && sprite) {
        const angle = vehicleScreenAngle(vehicleHeadingAt(track, time.frame))
        const scaleRatio = vehicleSpriteScale(size.naturalHeightPx, size.mmPerPx)
        ctx.globalAlpha = 1
        drawRotatedSprite(ctx, sprite, c.x, c.y, angle, scaleRatio * time.k)
        edgePx = (vehicleScreenLengthPx(size.naturalHeightPx, size.mmPerPx) / 2) * time.k
      }
      // Sinon : image ou manifeste pas encore chargés — rien ne remplace le sprite (même
      // contrat que les vignettes de socle), le nom garde le repli de plancher ci-dessus.
    }
    if (style.showNames) {
      const rides = vehicleActiveRides(track, time.frame)
      if (rides.length > 0) drawVehicleOccupantNames(ctx, rides, time.frame, c, edgePx, style, time.k)
    }
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
