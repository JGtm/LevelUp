/**
 * vehiclesPaint.ts — COMMENT un véhicule se peint : son cône de conducteur, son sprite orienté,
 * son losange de repli et les noms empilés de ses occupants.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-02). `vehiclesLayer.ts` décide CE QU'IL FAUT peindre — la
 * famille est-elle du décor, où est le véhicule à cette image, quel est son cap, qui est à bord,
 * quelle taille lui donner. Ce fichier-ci décide COMMENT : quelles primitives, dans quel ordre,
 * avec quelles encres. Le calque a franchi le seuil de taille du dépôt (CLAUDE.md n°5) en
 * gagnant le CÔNE DE VISÉE DU CONDUCTEUR, et la règle est d'EXTRAIRE plutôt que d'exempter. La
 * couture suit la frontière naturelle (règles pures d'un côté, canvas de l'autre), exactement
 * comme `zoneStatesLayer.ts` / `zoneStatesPaint.ts`.
 *
 * AUCUNE RÈGLE NE DOIT ENTRER ICI : toute décision (visibilité, position, cap, taille, occupants,
 * refus du décor) vient de `vehiclesLayer.ts` et y reste testable sans canvas.
 */
import type { ReplayVehicleRide } from '@/lib/api/types'

import { drawExplosion, EXPLOSION_MS } from './explosionFx'
import type { FxInk } from './fxInk'
import { project, type PlacementView } from './placementShapes'
import { drawAimSector } from './replayAimCone'
import { drawRotatedSprite } from './replayDraw'
import { drawNameLabel, type LabelStyle } from './replayLabels'
import type { XY } from './replayLogic'
import type { ReplayVehicleTrackReady } from './replayNormalize'
import {
  vehicleActiveRides,
  vehicleAimAngle,
  vehicleColorAt,
  vehicleDestructionFrame,
  vehicleDriverAt,
  vehicleExplosionKindOf,
  vehicleHeadingAt,
  vehicleIsDecor,
  vehiclePositionAt,
  vehicleRideColor,
  vehicleScreenAngle,
  vehicleScreenLengthPx,
  vehicleSpriteScale,
  vehicleVisibleAt,
  VEHICLE_FLOOR_PX,
  VEHICLE_UNKNOWN_HALF_PX,
} from './vehiclesLayer'
import { traceDiamond } from './weaponPadsLayer'

export type { PlacementView as VehicleView } from './placementShapes'

/** Ce que le calque a besoin de savoir de l'instant courant. */
export interface VehicleTime {
  frame: number
  /** Densité de pixels : toutes les tailles d'écran de ce tracé la suivent. */
  k: number
  /**
   * Durée RÉELLE d'une frame, en ms de match (`frameToMs(1, doc)`, même source que
   * `RestWindow.frameMs` des grenades) — l'explosion de destruction a une timeline en TEMPS, pas
   * en frames, EXACTEMENT le même besoin que `explosionFx`/`grenadeRestLayer`. Un artefact sans
   * échelle temporelle reçoit déjà le repli de `frameToMs` en amont ; ce champ ne le redevine pas.
   */
  frameMs: number
}

/**
 * Dimensions natives + échelle manifeste d'UNE famille — indépendant de la teinte.
 *
 * `naturalWidthPx` a rejoint `naturalHeightPx` le 2026-09-03 : ce calque n'en avait besoin que
 * pour la longueur (l'axe de mise à l'échelle), mais `vehicleWeaponMounts.vehicleShotPlacement`
 * a besoin des DEUX pour placer une ancre latérale (`ax`) dans le repère du sprite — même
 * source (`img.naturalWidth`), zéro requête de plus.
 */
export interface VehicleSpriteSize {
  naturalWidthPx: number
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
   * Calque de la VISÉE (bouton « Visée », le MÊME que celui des pions) : le cône du conducteur
   * le suit. Un utilisateur qui éteint les cônes les éteint partout — un cône de véhicule qui
   * survivrait à ce geste serait un calque sans interrupteur.
   */
  showAim: boolean
  /**
   * Vignette DÉJÀ TEINTE (family × couleur résolue), ou `null` — l'image source ou sa teinte
   * n'ont pas encore fini de charger : RIEN NE LA REMPLACE, elle apparaît après coup (même
   * contrat que les vignettes de socle, `useReplayWeaponPads`).
   */
  spriteOf: (family: string, color: string) => CanvasImageSource | null
  /** Dimensions natives + mm/px du manifeste pour une famille, ou `null` si pas encore chargées. */
  sizeOf: (family: string) => VehicleSpriteSize | null
  colorOfSlot: (slot: number, frame: number) => string | null
  /**
   * Couleur d'équipe d'un joueur PAR XUID — la SOURCE PRIORITAIRE de la teinte d'un occupant,
   * pour la raison EXACTE de `nameOfXuid` ci-dessous et sur les mêmes films : pendant l'épisode
   * le bipède ne réplique plus, le pont `colorOfSlot` est donc muet là où le document, lui,
   * nomme l'occupant (`VehicleRide.xuid`). Le contrat serveur promettait déjà cette couleur
   * (`document_vehicles.go` : « c'est lui qui donne sa couleur au véhicule ») ; la jointure par
   * slot ne la tenait pas. Le pont reste le repli.
   */
  colorOfXuid: (xuid: string) => string | null
  nameOfSlot: (slot: number, frame: number) => string | null
  /**
   * Nom d'un joueur PAR XUID — la SOURCE PRIORITAIRE de l'étiquette d'un occupant.
   *
   * POURQUOI IL FALLAIT L'AJOUTER (diagnostic du 2026-09-02, sur le film que l'utilisateur a
   * réellement visionné). L'étiquette ne se résolvait que par le PONT SLOT->JOUEUR
   * (`nameOfSlot` -> `ownerAtFrame`), c'est-à-dire par la vie de BIPÈDE qui occupe le slot à
   * l'image. Or `rosterLogic.buildPlayers` écarte toute trace SANS xuid (`if (!track.xuid)
   * continue`) : quand le film n'a pas joint le bipède à un joueur, le pont ne rend rien — et
   * le nom disparaissait alors même que le DOCUMENT nomme l'occupant lui-même
   * (`VehicleRide.xuid`, posé côté serveur). On lit donc d'abord ce que le document affirme, et
   * le pont ne sert plus que de repli.
   */
  nameOfXuid: (xuid: string) => string | null
  /**
   * Teintes de nature des effets (fxInk.ts, MÊME source que les tirs/grenades) : l'explosion de
   * destruction (demande utilisateur, cf. `drawVehicleDestructionFx`) y puise sa couleur PLASMA
   * (Covenant/Bannis) ou NORMALE (UNSC/humains) — jamais la couleur d'équipe de l'occupant, la
   * déflagration dit ce qui explose, pas qui le conduisait (même doctrine que `fxInk.ts`).
   */
  explosionInk: FxInk
  /** Sous « mouvement réduit », l'explosion de destruction ne se joue pas (même contrat que
   *  `explosionFx.ExplosionShape.reduced` et `GrenadeRestStyle.reducedMotion`). */
  reducedMotion: boolean
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
  style: Pick<VehicleStyle,
    'nameOfSlot' | 'nameOfXuid' | 'colorOfSlot' | 'colorOfXuid' | 'labelStroke' | 'neutralInk'>,
  k: number,
): void {
  const label: LabelStyle = { k, labelStroke: style.labelStroke }
  let line = 0
  for (const ride of rides) {
    const name = occupantName(ride, frame, style)
    if (!name) continue
    // JAMAIS `labelStroke` EN REPLI (correction du 2026-09-02) : `drawNameLabel` CERNE les
    // lettres avec cette même encre — un nom rempli à la couleur de son propre contour est un
    // pâté illisible, c'est-à-dire un nom perdu. L'encre du « aucun camp connu » est faite pour
    // ça, et c'est déjà celle du véhicule sans occupant résolu.
    const color = vehicleRideColor(ride, frame, style) ?? style.neutralInk
    drawNameLabel(ctx, c, name, label, color, baseEdgePx + VEHICLE_NAME_LINE_STEP_PX * line * k)
    line++
  }
}

/**
 * occupantName — QUI EST À BORD, dans l'ordre des sources : le XUID que le DOCUMENT porte sur
 * l'épisode d'abord, le pont slot->joueur ensuite (cf. `VehicleStyle.nameOfXuid`).
 */
function occupantName(
  ride: ReplayVehicleRide,
  frame: number,
  style: Pick<VehicleStyle, 'nameOfSlot' | 'nameOfXuid'>,
): string | null {
  const byXuid = ride.xuid ? style.nameOfXuid(ride.xuid) : null
  return byXuid ?? style.nameOfSlot(ride.slot, frame)
}

/**
 * drawVehicleAimCone — LE CÔNE DE VISÉE DU CONDUCTEUR, posé sur le véhicule.
 *
 * POURQUOI IL EXISTE (retour utilisateur du 2026-09-02, après visionnage réel). Un joueur
 * EMBARQUÉ ne réplique plus son bipède : son cap de visée n'existe plus dans le film, et son
 * pion — donc son cône — a disparu de la carte (décision « PION EMBARQUÉ » du plan). Il ne
 * restait alors AUCUN indice de direction sur un véhicule pourtant conduit. La décision du
 * chantier comble ce trou sans rien inventer : « à l'arrêt on assume qu'il regarde devant lui ;
 * en mouvement, la direction du déplacement » — c'est-à-dire EXACTEMENT le cap du véhicule
 * (`vehicleHeadingAt`, qui tient déjà ces deux régimes).
 *
 * CE QU'IL N'AFFIRME PAS : la visée d'un PASSAGER ou d'un TOURELLEUR. Elle est indépendante du
 * véhicule et le film ne la porte pas — seul le siège 0 obtient un cône (`vehicleDriverAt`).
 *
 * MÊME GÉOMÉTRIE ET MÊMES OPACITÉS QUE LE CÔNE DES PIONS : c'est le module `replayAimCone` qui
 * trace (`drawAimSector`), pas une copie locale. Deux différences, toutes deux voulues : pas
 * d'ÉLÉVATION (un cap de déplacement est horizontal — le cône garde sa longueur de référence,
 * ce que ce module lit déjà comme « à plat ») et pas de FRAÎCHEUR (le cap du véhicule est celui
 * de l'image, pas une mesure de visée qu'on maintient en pâlissant).
 */
function drawVehicleAimCone(
  ctx: CanvasRenderingContext2D,
  track: ReplayVehicleTrackReady,
  time: VehicleTime,
  style: Pick<VehicleStyle, 'showAim' | 'colorOfSlot' | 'colorOfXuid'>,
  c: XY,
): void {
  if (!style.showAim) return
  const driver = vehicleDriverAt(track, time.frame)
  if (!driver) return
  // L'ENCRE EST CELLE DU PION DU MÊME JOUEUR, sans repli : un cône neutre dirait « quelqu'un
  // regarde par là » sans dire qui — le calque des pions applique la même règle (une vie sans
  // couleur ne se dessine pas).
  const color = vehicleRideColor(driver, time.frame, style)
  if (!color) return
  drawAimSector(ctx, c, vehicleAimAngle(vehicleHeadingAt(track, time.frame)), time.k, color)
}

/**
 * vehicleExplosionSeed — germe stable, dérivé du SLOT et de la frame de destruction : deux
 * véhicules détruits à la même image tirent des particules différentes, et un retour en arrière
 * rejoue TOUJOURS la même explosion (même contrat que `explosionFx`/`GrenadeRestFx.seed`).
 */
function vehicleExplosionSeed(track: ReplayVehicleTrackReady, destroyedFrame: number): number {
  return (track.slot * 2654435761 + destroyedFrame * 97) % 100003
}

/**
 * vehicleExplosionEdgePx — LA MÊME primitive de taille que le sprite (`sizeOf` +
 * `vehicleScreenLengthPx`), pour l'explosion : jamais un second calcul de gabarit. Le repli
 * (chassis non résolu, ou vignette/manifeste pas encore chargés) vaut `VEHICLE_UNKNOWN_HALF_PX`
 * — EXACTEMENT la moitié du plancher de lisibilité d'un véhicule (`VEHICLE_FLOOR_PX`) — pour que
 * l'explosion d'un châssis inconnu ait la même ampleur qu'une grenade, pas une taille inventée.
 */
function vehicleExplosionEdgePx(
  track: ReplayVehicleTrackReady,
  style: Pick<VehicleStyle, 'sizeOf'>,
  k: number,
): number {
  const size = track.family ? style.sizeOf(track.family) : null
  if (!size) return VEHICLE_UNKNOWN_HALF_PX * k
  return (vehicleScreenLengthPx(size.naturalHeightPx, size.mmPerPx) / 2) * k
}

/**
 * drawVehicleDestructionFx — LA DESTRUCTION D'UN VÉHICULE, demande utilisateur mot pour mot :
 * « il faut aussi la destruction et un effet UI », « explosion plasma et explosion "normale"
 * pour les véhicules humains ».
 *
 * ZÉRO SECOND MOTEUR : c'est `explosionFx.drawExplosion` (déjà éprouvé sur les grenades)
 * RÉUTILISÉ TEL QUEL. Deux réglages seulement sont posés ici : L'ENCRE — plasma Covenant/Bannis
 * vs feu UNSC, `vehicleExplosionKindOf` (vehiclesLayer.ts), sur les MÊMES tokens de thème que les
 * grenades (`--replay-fx-plasma-cool` / `--replay-fx-blast`, cf. fxInk.ts) puisque c'est la même
 * NATURE de déflagration — et L'ÉCHELLE (`vehicleExplosionEdgePx`, la primitive de taille du
 * sprite, jamais un second calcul).
 *
 * ANCRÉE SUR LA POSITION DU VÉHICULE À `destroyedFrame`, JAMAIS SUR SA POSITION COURANTE : le
 * véhicule a cessé d'exister à cet instant précis, l'effet ne doit pas dériver si le document
 * portait encore des échantillons après coup (`vehiclePositionAt` tiendrait sinon la dernière
 * position connue, potentiellement plus tardive).
 *
 * SILENCIEUSE TANT QUE LE CONTRAT N'EXISTE PAS : `drawVehiclesLayer` ne l'appelle QUE quand
 * `vehicleDestructionFrame` a répondu un entier — aujourd'hui (tout artefact publie encore
 * `end: "unknown"`), cette fonction n'est JAMAIS invoquée : zéro dessin, zéro changement visible.
 */
function drawVehicleDestructionFx(
  ctx: CanvasRenderingContext2D,
  track: ReplayVehicleTrackReady,
  destroyedFrame: number,
  time: VehicleTime,
  view: PlacementView,
  style: Pick<VehicleStyle, 'sizeOf' | 'explosionInk' | 'reducedMotion'>,
): void {
  const ageMs = (time.frame - destroyedFrame) * time.frameMs
  if (ageMs < 0 || ageMs > EXPLOSION_MS) return
  const world = vehiclePositionAt(track, destroyedFrame)
  if (!world) return
  const c = project(world, view)
  // RATIO SANS DENSITÉ : `vehicleExplosionEdgePx` porte déjà `time.k` (comme la variable
  // homonyme de `drawVehiclesLayer`) ; le diviser par SA propre valeur de repli (constante, sans
  // `k`) l'ANNULE et ne laisse que le facteur de taille relative — que `drawExplosion` reçoit
  // ensuite comme SON `k`, il multiplie déjà tout par ce facteur (cf. explosionFx.ts).
  const k = vehicleExplosionEdgePx(track, style, time.k) / VEHICLE_UNKNOWN_HALF_PX
  const kind = vehicleExplosionKindOf(track.family)
  const fire =
    (kind === 'plasma' ? style.explosionInk.tint.plasma_cool : style.explosionInk.tint.blast) ||
    style.explosionInk.tint.neutral
  drawExplosion(
    ctx,
    { x: c.x, y: c.y, ageMs, seed: vehicleExplosionSeed(track, destroyedFrame), k, reduced: style.reducedMotion },
    { fire, core: style.explosionInk.core, smoke: '' },
  )
}

/**
 * drawVehiclesLayer trace TOUS les véhicules VISIBLES à l'image courante, et LA DESTRUCTION de
 * ceux dont elle est établie (cf. `drawVehicleDestructionFx`) — CETTE DERNIÈRE INDÉPENDAMMENT DE
 * LA VISIBILITÉ COURANTE : elle continue de jouer après que le sprite a cessé (`tEnd` fait
 * autorité sur `vehicleVisibleAt`, cf. vehiclesLayer.ts), sur SA PROPRE fenêtre de temps.
 *
 * ORDRE PEINTRE : l'explosion se pose APRÈS le sprite (donc AU-DESSUS) — c'est sa fin. Elle est
 * dessinée en dernier dans la boucle plutôt qu'avant le corps de la vie pour cette seule raison :
 * à la frame exacte de la destruction, le sprite est encore dans sa fenêtre de visibilité
 * (`tEnd` inclus), et doit rester SOUS le tout premier instant de l'explosion.
 *
 * ORDRE DU DOCUMENT ENTRE VÉHICULES, comme les armes au sol : aucun arbitrage de recouvrement
 * n'est fait ici.
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
    // LE DÉCOR NE SE DESSINE PAS (verdict utilisateur 2026-09-02) : ni sprite, ni losange de
    // repli, ni nom, ni explosion — le refus est en tête de boucle pour qu'aucune branche n'y
    // échappe.
    if (vehicleIsDecor(track.family)) continue
    if (vehicleVisibleAt(track, time.frame)) {
      const world = vehiclePositionAt(track, time.frame)
      if (world) {
        const c = project(world, view)
        const color = vehicleColorAt(track, time.frame, style) ?? style.neutralInk
        drawVehicleAimCone(ctx, track, time, style, c)
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
    }
    // LA DESTRUCTION (schéma 30, EN AVANCE DE PHASE) : `null` tant que le document ne publie que
    // `end: "unknown"` — aujourd'hui, systématiquement. Voir l'en-tête de fonction pour l'ordre
    // peintre et l'indépendance vis-à-vis de la visibilité ci-dessus.
    const destroyedFrame = vehicleDestructionFrame(track)
    if (destroyedFrame !== null) {
      drawVehicleDestructionFx(ctx, track, destroyedFrame, time, view, style)
    }
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
