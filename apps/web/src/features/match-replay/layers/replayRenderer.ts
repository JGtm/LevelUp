/**
 * replayRenderer.ts — LE MOTEUR DE LA TOILE 2D DU REJEU : lier la scène à un état, puis la peindre.
 *
 * Deux entrées publiques, aucune ne dépend de React :
 *  - `buildReplayScene` LIE chaque calque à l'état d'une toile (`ReplaySceneInputs`) et rend la
 *    scène que `replayCompose` ordonne ;
 *  - `paintReplayFrame` dimensionne la toile, l'efface, compose la scène et dit s'il y avait
 *    quelque chose à peindre.
 *
 * Une toile qui a son propre cadrage (`view`) et ses propres cuissons appelle le même moteur avec
 * ses propres entrées : le composant du rejeu n'est que l'un de ses appelants.
 *
 * L'ordre des calques et leurs conditions d'ouverture vivent dans `replayCompose` ; ce module ne
 * porte que les liaisons. TOUT CE QUI DÉPEND DE L'IMAGE arrive en argument du peintre (`frame`,
 * `dpr`) : la table ne se relie qu'à ce qui change avec les données ou le thème.
 */
import { drawFireMarks } from './fireMark'
import { drawEquipmentPlacementsLayer } from './equipmentPlacementsLayer'
import { drawMapBackground, placeMapBackground } from './mapBackgroundDraw'
import { drawObjectivePulses } from './objectivesLayer'
import { drawRiftLayer } from './placementRift'
import { prepareStage } from './canvasStage'
import { drawGeometryLayer, drawKillFxLayer, drawShotsLayer } from './replayDraw'
import { drawProjectilesLayer, PROJECTILE_TAIL_MS } from './replayProjectiles'
import { drawTracksLayer } from './replayMarkers'
import { drawOpenPlayerRing } from './openPlayerRing'
import { drawZoneStates } from './zoneStatesLayer'
import {
  bindPainters,
  composeScene,
  sceneLayers,
  type LayerPaint,
  type ReplayLayerId,
  type ReplayScene,
} from './replayCompose'
import type { ReplaySceneInputs } from './replayRenderer.types'
import { REPLAY_TEXT } from '../i18n/i18n'
import { durationToFrames, frameToMs, layerOffset } from '../../../lib/replay/replayLogic'

export type { ReplaySceneInputs } from './replayRenderer.types'

type Painters<Id extends ReplayLayerId> = Pick<ReplayScene['paint'], Id>
type MapRect = ReturnType<typeof placeMapBackground>

/** La FENÊTRE D'ÉVÉNEMENT, commune aux tirs, aux pulses et aux morts. */
interface EventWindow {
  frame: number
  hold: number
  frameMs: number
}

/**
 * paintReplayFrame peint l'image `frame` sur `canvas` : dimensionne, efface, compose.
 *
 * Rend faux quand il n'y a rien à peindre (pas de toile, largeur pas encore mesurée, contexte
 * indisponible) : l'appelant n'avance alors ni horloge ni publication.
 */
export function paintReplayFrame(
  canvas: HTMLCanvasElement | null,
  frame: number,
  inputs: ReplaySceneInputs,
): boolean {
  const stage = prepareStage(canvas, inputs.size.width, inputs.size.height)
  if (!stage) return false
  composeScene(stage.ctx, sceneLayers(buildReplayScene(stage.ctx, frame, inputs)), frame, stage.dpr)
  return true
}

/**
 * buildReplayScene LIE chaque calque à l'état de la toile : les bascules, la matière à peindre et
 * le peintre de chacun des calques de `LAYER_ORDER`.
 */
export function buildReplayScene(
  ctx: CanvasRenderingContext2D,
  frame: number,
  s: ReplaySceneInputs,
): ReplayScene {
  const { weaponPads, groundWeapons, vehicles, abilityFx, grenadeRest, flags, objectiveObjects } = s.layers
  const { vipCrown, skullCarrier, bombCarrier, bombBlast, killFocus, lives, respawnForecast } = s.layers
  // Pendant un glisser, cuisson gelée : les calques cuits se recopient DÉCALÉS (layerOffset).
  const lo = layerOffset(s.cooked.origin.current, s.view)
  // LE FOND SUIT LA FENÊTRE, PAS LA SCÈNE : sinon il resterait cadré large pendant que les
  // joueurs zooment, et l'image cesserait de désigner le même endroit qu'eux.
  const bgRect = placeMapBackground(s.background, s.view)
  const win: EventWindow = { frame, hold: s.eventHoldFrames, frameMs: frameToMs(1, s.doc) }
  return {
    toggles: sceneToggles(s),
    has: sceneMatter(s, bgRect),
    paint: {
      // LES ONZE CALQUES CÂBLÉS PAR HOOK SE NOMMENT EUX-MÊMES : leur `id` vit dans le hook, à côté
      // de son `paint`, et `bindPainters` en dérive la liaison. Aucun d'eux n'est nommé ici.
      ...bindPainters(
        weaponPads,
        groundWeapons,
        vehicles,
        abilityFx,
        grenadeRest,
        flags,
        objectiveObjects,
        vipCrown,
        skullCarrier,
        bombCarrier,
        bombBlast,
        killFocus.veil,
        killFocus.labels,
        ...lives.layers,
        ...respawnForecast.layers,
      ),
      ...terrainPainters(ctx, s, bgRect, lo),
      ...trackPainters(ctx, s),
      ...effectPainters(ctx, s, win),
    },
  }
}

/** Les bascules du tiroir qui ferment un calque entier. */
function sceneToggles(s: ReplaySceneInputs): ReplayScene['toggles'] {
  const { show, layers } = s
  return {
    zones: show.zones,
    shotFx: show.shotFx,
    placements: show.placements,
    killFx: show.killFx,
    modeObjectives: show.modeObjectives,
    wallPlan: show.wallPlan,
    lives: layers.lives.show,
    respawnForecast: layers.respawnForecast.show,
  }
}

/** Ce que la scène a effectivement à peindre : un calque sans matière ne s'ouvre pas. */
function sceneMatter(s: ReplaySceneInputs, bgRect: MapRect): ReplayScene['has'] {
  const { doc, cooked, placements, fx, layers } = s
  return {
    background: !!s.background && !!bgRect,
    floor: !!doc.geometry?.length,
    wallPlan: !!cooked.wallPlan.current,
    heat: !!cooked.heat.current,
    zoneNames: !!cooked.zones.current,
    objectivesCooked: !!cooked.objectives.current,
    projectiles: !!doc.projectiles?.length,
    placements: placements.drawableCount > 0,
    rifts: placements.rift.rifts.length > 0 || placements.rift.teleports.length > 0,
    fireMarks: fx.fireMarks.length > 0,
    shotFx: fx.shotFx.length > 0,
    zoneStates: doc.zoneStates.length > 0,
    objectivePulses: fx.objectivePulses.length > 0,
    killFx: fx.killFx.length > 0,
    lives: layers.lives.available,
    respawnForecast: layers.respawnForecast.ready,
  }
}

/** Le sol, les calques cuits et les vols de projectile : ce qui se pose sous les pions. */
function terrainPainters(
  ctx: CanvasRenderingContext2D,
  s: ReplaySceneInputs,
  bgRect: MapRect,
  lo: ReturnType<typeof layerOffset>,
): Painters<
  'fond-carte' | 'sol-forge' | 'plan-murs' | 'chaleur' | 'zones-nommees' | 'objectifs-cuits' | 'projectiles'
> {
  const { doc, view, zRange, cooked } = s
  const { wallPlan: wallPlanRef, heat: heatRef, zones: zonesRef, objectives: objectivesRef } = cooked
  const { shownProjectile } = s.focus
  const { geometry: geometryColor, grenade: grenadeColor } = s.ink
  // Un calque CUIT se repose tel quel, décalé de `lo` : sa géométrie ne bouge pas.
  const cuit =
    (c: HTMLCanvasElement | null): LayerPaint =>
    () => {
      if (c) ctx.drawImage(c, lo.x, lo.y, s.size.width, s.size.height)
    }
  return {
    // L'image ENTIÈRE sur son emprise monde, le canvas rogne le débord (mapBackgroundDraw.ts).
    'fond-carte': () => drawMapBackground(ctx, s.background, bgRect, view),
    'sol-forge': () =>
      drawGeometryLayer(ctx, doc.geometry ?? [], view, { color: geometryColor, z: zRange }),
    'plan-murs': cuit(wallPlanRef.current),
    chaleur: cuit(heatRef.current),
    'zones-nommees': cuit(zonesRef.current),
    'objectifs-cuits': cuit(objectivesRef.current),
    projectiles: (_c, fr) =>
      drawProjectilesLayer(
        ctx,
        doc.projectiles ?? [],
        view,
        { frame: fr, tailFrames: durationToFrames(PROJECTILE_TAIL_MS, doc), color: grenadeColor, shown: shownProjectile },
      ),
  }
}

/** Les poses, les pions et les failles : ce qui se pose sur le terrain. */
function trackPainters(
  ctx: CanvasRenderingContext2D,
  s: ReplaySceneInputs,
): Painters<'poses-equipement' | 'trajectoires' | 'failles'> {
  const { doc, view, locale, reducedMotion, zRange, timing, placements } = s
  const { colorOfSlot, colorOfSlotOrLast, markOfSlot, nameOfSlot, campOfSlot, seatedAtSlot } = s.identity
  const { shownAtSlot, shown: focusShown, openPlayer } = s.focus
  const { vehicles, killFocus } = s.layers
  const { floorEdge, shot: shotColor, wall: wallInk, rift: riftInk, shroud: shroudInk } = s.ink
  const { self: selfInk, labelStroke, markerOutline } = s.ink
  const { aim: showAim, trail: showTrail } = s.show
  return {
    // La fenêtre d'une pose n'est PAS [t0, t1] : `t1` date la mise au repos, pas la
    // disparition (placementEndFrame) ; le ping bat en TEMPS de match, pas en images.
    'poses-equipement': (_c, fr, k) =>
      drawEquipmentPlacementsLayer(
        ctx,
        // Les VIES et leur CAMP voyagent avec les poses : le ping du capteur révèle les
        // adversaires du poseur, et « adversaire » est une relation entre deux vies. Le
        // camp est l'équipe du FILM de chacune, jamais le drapeau « allié » de la page.
        { placements: placements.all, lives: doc.tracks, campOfSlot, shown: focusShown },
        view,
        { frame: fr, ...placements.windowTime, k, reducedMotion, ...placements.toggles },
        // FRONTIÈRE : objet lâché à la mort, `t0 = finVie+1` — `colorOfSlotOrLast`.
        { colorOfSlot: colorOfSlotOrLast, neutral: floorEdge, wall: wallInk, rift: riftInk, shroud: shroudInk },
      ),
    trajectoires: (_c, fr, k) => {
      drawTracksLayer(ctx, doc.tracks, view, {
        colorOfSlot,
        outline: markerOutline,
        frame: fr,
        timing,
        z: zRange,
        k,
        showAim,
        markOfSlot,
        nameOfSlot,
        showTrail,
        selfInk,
        deathInk: shotColor,
        labelStroke, embarkedAtSlot: vehicles.isEmbarkedAt, seatedAtSlot, shownAtSlot, // PION EMBARQUÉ (C7) ; PION SANS FICHE ; SEUL / MASQUER.
        nameHiddenAtSlot: killFocus.nameHiddenAtSlot,
        // BORNAGE HORS CADRE : la flèche porte le nom ET la distance ; le texte est résolu ICI
        // (langue de la toile), le calque ne connaît aucune locale.
        offscreenLabelOf: (name, meters) => REPLAY_TEXT[locale].offscreenMarkerFmt(name, meters),
      })
      // LE PION CERCLÉ de la fiche ouverte, par-dessus les pions : l'anneau ne dit rien d'autre que « celui-ci ».
      if (openPlayer) drawOpenPlayerRing(ctx, openPlayer, view, fr, k)
    },
    // LES FAILLES, AU-DESSUS DES PISTES : la station et le passage du translocateur.
    failles: (_c, fr, k) =>
      drawRiftLayer(
        ctx,
        placements.rift,
        view,
        { frame: fr, ...placements.windowTime, k, reducedMotion },
        riftInk,
      ),
  }
}

/** Les marques et effets d'événement (tirs, zones, pulses, morts) : ce qui se pose sur les pions. */
function effectPainters(
  ctx: CanvasRenderingContext2D,
  s: ReplaySceneInputs,
  win: EventWindow,
): Painters<'marques-de-tir' | 'tirs' | 'etat-zones' | 'pulses-objectif' | 'morts'> {
  const { doc, view, reducedMotion, zones, shotHoldFrames } = s
  const { colorOfSlot, colorOfSlotOrLast } = s.identity
  const { shownAtSlot, shown: focusShown } = s.focus
  const { shotFx, fireMarks, killFx, objectivePulses } = s.fx
  const { vehicles } = s.layers
  const { floorEdge, shot: shotColor, fx: fxInk, labelStroke } = s.ink
  return {
    'marques-de-tir': (_c, fr, k) =>
      drawFireMarks(ctx, fireMarks, view, {
        frame: fr, hold: shotHoldFrames, colorOfSlot, ink: labelStroke || floorEdge, k, embarkedAtSlot: vehicles.isEmbarkedAt, shownAtSlot,
      }),
    // vehicleSizeOf : origine des tirs en véhicule sur LA MÊME source de tailles que le
    // calque véhicules (`useReplayVehicles.sizeOf`), jamais un second chargement.
    tirs: (_c, _fr, k) =>
      drawShotsLayer(ctx, shotFx, view, { ...win, hold: shotHoldFrames }, {
        ink: fxInk, k, reducedMotion, vehicleSizeOf: vehicles.sizeOf, shownAtSlot,
      }),
    'etat-zones': (_c, fr) => drawZoneStates(ctx, zones, doc.zoneStates, view, fr),
    'pulses-objectif': () =>
      drawObjectivePulses(ctx, objectivePulses, view, win, { colorOfTeam: zones.colorOfTeam, shownAuthor: focusShown?.author }, reducedMotion),
    morts: (_c, _fr, k) =>
      drawKillFxLayer(ctx, killFx, view, win, {
        colorOfSlot: colorOfSlotOrLast, // FRONTIÈRE : kill posthume/échange après la fin de vie.
        fallback: shotColor, reducedMotion, k, shown: focusShown?.killFx,
      }),
  }
}
