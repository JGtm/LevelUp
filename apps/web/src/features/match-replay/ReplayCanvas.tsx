/**
 * ReplayCanvas — rejeu 2D animé (vue du dessus) des trajectoires joueurs décodées du film.
 * Rendu canvas (animation fluide) ; toute la logique est pure dans replayLogic.ts et le
 * dessin dans replayDraw.ts (anti-pattern « logique dans le composant »).
 *
 * Trois traits du document sont exploités ici :
 * - frameIntervalMs : la vitesse « 1× » suit le TEMPS RÉEL du match (avant : 60 frames/s
 *   sur un axe qui n'était qu'un index de record → rejeu très court et figé) ;
 * - geometry : fond de carte (props Forge) dessiné SOUS les trajectoires ;
 * - bounds.minZ/maxZ + points[].z : indication d'étage (opacité du décor, anneaux du marqueur).
 *
 * Couleurs = tokens sémantiques résolus en valeurs concrètes (getSeriesColors/resolveToken),
 * re-résolus au changement de thème/palette via useColorPaletteVersion (règle color-tokens).
 *
 * LES JOUEURS PORTENT LA COULEUR DE LEUR ÉQUIPE, celle que l'utilisateur a choisie dans les
 * réglages d'accessibilité (`team-ally` / `team-enemy`) — décision D1 du plan d'habillage,
 * amendée par l'utilisateur le 2026-08-16. Elle est attribuée AU JOUEUR : jusque-là, une
 * couleur de série était indexée sur la TRACE, c'est-à-dire sur la VIE (99 traces pour 8
 * joueurs sur le film témoin), et un joueur changeait donc de teinte à chaque réapparition.
 * Les tokens de série ne servent plus qu'aux ZONES NOMMÉES, qui sont des lieux, pas des gens.
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { getSeriesColors } from '@/lib/accessibility/plotlyColorscale'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { XuidMeta } from '@/features/match-view/xuidMeta'

import type { CalloutZoneReady } from './calloutsLayer'

import { ReplayHeatmapLegend } from './ReplayHeatmapLegend'
import type { PlaybackStore } from './model/playbackStore'
import { ReplayTransport } from './ReplayTransport'
import { ReplayZoomControl } from './ReplayZoomControl'
import { useTeamCascades } from './useTeamCascades'
import { drawObjectivePulses, normalizeMapObjectives } from './objectivesLayer'
import { drawZoneStates } from './zoneStatesLayer'
import { drawFireMarks } from './fireMark'
import { useReplayAbilityFx } from './useReplayAbilityFx'
import { drawEquipmentPlacementsLayer } from './equipmentPlacementsLayer'
import { ReplayCanvasTips } from './ReplayCanvasTips'
import { useReplayPlacements } from './useReplayPlacements'
import { EMPTY_FEED, EMPTY_MEDIA, EMPTY_ZONES, SERIES_TOKENS } from './replayCanvasConfig'
import { useReplayObjectiveObjects } from './useReplayObjectiveObjects'
import { useReplayVipCrown } from './useReplayVipCrown'
import { useReplayBombCarrier } from './useReplayBombCarrier'
import { useReplaySkullCarrier } from './useReplaySkullCarrier'
import { useReplayBombBlast } from './useReplayBombBlast'
import { useReplayGrenadeRest } from './useReplayGrenadeRest'
import { useReplayFlagCarries } from './useReplayFlagCarries'
import { useGrenadeIcons } from './useGrenadeIcons'
import { useZoneStates } from './useZoneStates'
import { useReplayWeaponPads } from './useReplayWeaponPads'
import { useReplayGroundWeapons } from './useReplayGroundWeapons'
import { useReplayVehicles } from './useReplayVehicles'
import type { ReplayLocale } from './i18n/i18n'
import type { EndMatchSoundSpec } from './sound/endMatchSound'
import { killsOfFeed, type ReplayFeedEntry } from './killFeedLogic'
import type { ReplayMediaItem } from './replayTimelineTracksLogic'
import { useReplayFx } from './useReplayFx'
import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import { useReplayDrawer } from './settings/useReplayDrawer'
import { useReplayTimeline } from './useReplayTimeline'
import { useSlotIdentity } from './useSlotIdentity'
import { ReplaySettingsDrawer } from './settings/ReplaySettingsDrawer'
import { useReplayHeatmap } from './useReplayHeatmap'
import { useReplayInks } from './useReplayInks'
import { hoverHandlers } from './hoverLayers'
import type { ExportOutcome } from './export/exportOverlayPanels'
import { useReplayCapture } from './export/useReplayCapture'
import { useReplayClock } from './useReplayClock'
import { useReplayPlayback } from './useReplayPlayback'
import { useReplayStaticLayers } from './useReplayStaticLayers'
import { useReplaySettings } from './settings/useReplaySettings'
import { useReplaySound } from './sound/useReplaySound'
import { backgroundRect } from './mapBackground'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  drawGeometryLayer,
  drawGrenadesLayer,
  drawKillFxLayer,
  drawShotsLayer,
} from './replayDraw'
import {
  composeScene,
  sceneLayers,
  type LayerPaint,
  type ReplayScene,
} from './replayCompose'
import { frameToMs, layerOffset } from './replayLogic'
import type { ReplayWindowBounds } from './replayWindow'
import { drawProjectilesLayer } from './replayProjectiles'
import { drawTracksLayer } from './replayMarkers'
import { useReplayTiming } from './useReplayTiming'
import { CANVAS_PAD, exportRenderScale, useReplayView, type ReplayMapBackgroundLayer } from './useReplayView'
import { useReplayViewport } from './useReplayViewport'
import { useReplayWheelZoom } from './useReplayWheelZoom'
import { useReplayDrag } from './useReplayDrag'

interface ReplayCanvasProps {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  /**
   * LA FENÊTRE DE GAMEPLAY, calculée UNE fois par la page (`replayWindow.ts`) : elle borne la
   * lecture et la frise, et recale l'horloge affichée. `null` = pas de cadrage établi (artefact
   * sans origine, en-tête sans durée jouable) : le film se lit entier, comme avant.
   */
  playWindow: ReplayWindowBounds | null
  /**
   * LE MAGASIN DE LECTURE (`model/playbackStore`) : le canvas y ECRIT la position, tout le
   * reste de la page l'y LIT. C'est la seule representation de « quelle image on regarde ».
   */
  playbackStore: PlaybackStore
  /**
   * Fond de carte figé. Absent = pas d'image CALÉE pour cette carte, et le rejeu retombe sur
   * le sol reconstruit. C'est une LACUNE, pas un mode : `map_structure` ne contient que DEUX
   * fichiers, et 129 vues du dessus dessinées dorment dans `static/maps` sans le calage qui
   * les rendrait utilisables ici. (Il a longtemps dit « seules 21 en ont » : il y en a 106.)
   */
  background?: ReplayMapBackgroundLayer | null
  /**
   * Zones nommées de la carte (callouts officiels), déjà normalisées par la route —
   * la même liste sert les fiches (zone courante). Vide = la carte n'en a pas (cas
   * Forge, par construction) : pas de calque, pas de bouton.
   */
  callouts?: CalloutZoneReady[]
  /**
   * Le scoreboard du match : il donne aux vies du film le NOM et l'ÉQUIPE de leur
   * propriétaire (jointure par xuid, cf. rosterLogic). Absent = aucune identité connue,
   * la carte reste lisible (points à l'encre neutre, sans étiquette) — jamais une erreur.
   */
  scoreboard?: MatchScoreboardRow[]
  /** Camp de chaque xuid, du point de vue du joueur de la page (allié / adversaire). */
  xuidMeta?: XuidMeta
  /** Marques d'identité par xuid (« moi », « ami ») : elles décident de la FORME du point. */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /**
   * LE FIL ALIGNÉ ET LES MÉDIAS, assemblés une fois par la page (`buildFeedEntries`,
   * `buildReplayMedia`) : un second recalage ici divergerait de ce qu'on lit à côté.
   */
  feedEntries?: readonly ReplayFeedEntry[]
  media?: readonly ReplayMediaItem[]
  /**
   * LA FIN DE PARTIE SONORE (lot C) : l'issue du joueur de la page et la langue, lues UNE fois
   * par la page — la même lecture que l'écran de fin (`endMatchSound.ts`). Le canvas ne fait
   * que la relayer au lecteur, qui préchargera les prises et les jouera quand la lecture
   * atteindra la borne de fin. Absente = aucune conclusion sonore, le reste est inchangé.
   */
  endMatch?: EndMatchSoundSpec | null
  /**
   * LE VERDICT DU MATCH, pour l'EXPORT seul : c'est lui qui permet de peindre l'ecran de fin
   * DANS la toile, la ou l'ecran affiche (`ReplayVictoryOverlay`) est du DOM qu'aucun encodeur
   * ne voit. Absent = clip sans ecran de fin, exactement comme une page sans libelle d'issue.
   */
  outcome?: ExportOutcome | null
}

export function ReplayCanvas({
  doc, locale, playWindow, playbackStore, background, callouts, scoreboard, xuidMeta, marks,
  endMatch, outcome, feedEntries = EMPTY_FEED, media = EMPTY_MEDIA,
}: ReplayCanvasProps) {
  // LES KILLS VIENNENT DU FIL, DÉJÀ RECALÉS (2026-09-05, J2) : la carte et la piste sonore
  // lisaient les kills BRUTS et rejouaient chacune `alignFeed` — quatre exécutions du même
  // recalage par chargement, et deux chemins qui divergeraient le jour où le canvas ne
  // recevrait plus exactement les mêmes kills que le fil. Ils partagent désormais sa sortie.
  const feedKills = useMemo(() => killsOfFeed(feedEntries), [feedEntries])
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  // LA CELLULE DE DESSIN : la boucle l'avance a la cadence de l'ecran et les calques
  // survolables la lisent sans rendu. C'est la SOURCE de la position ; ce que la page qui
  // entoure le canvas en voit passe par `playbackStore.publish` (cf. model/playbackStore).
  const frameRef = useRef(0)
  // L'HORLOGE AFFICHÉE et la publication bridée vivent dans useReplayClock (dixième extraction).
  const { clockRef, tick: clockTick } = useReplayClock({ doc, playWindow, publish: playbackStore.publish })

  // CE QUE L'ÉCRAN OFFRE (useReplayViewport) ; ce que la carte en retient, c'est useReplayView.
  const { width, freeHeight } = useReplayViewport(containerRef, canvasRef)
  // L'OBJET DE RÉGLAGES RESTE ENTIER (2026-08-28) : le tiroir en consomme la quasi-totalité ;
  // le dessin ne lit que les valeurs — d'où cette destructuration-là (cf. useReplayDrawer).
  const settings = useReplaySettings()
  const {
    showAim, showZones, showTrail, showHeatmap, heatmapMode, heatmapSpan,
    showShotFx, showKillFx, showPlacements, showUnnamedPlacements, showDroppedPlacements,
    showWeaponPads, showGroundWeapons, showFlagCarries, showVipCrown, showSkullCarrier, showBombCarrier, showVehicles, speed: multiplier,
    markerColors,
  } = settings
  // SON : coupé par défaut, câblage dans le hook (replaySound.ts, lecture replayAudio.ts, camps
  // objectiveSound.ts, fin endMatch, « manche terminée » locale-aware — la `locale` ne sert qu'à lui).
  const sound = useReplaySound(doc, feedKills, multiplier, scoreboard, endMatch ?? null, locale)

  const paletteVersion = useColorPaletteVersion()
  // TOUTES LES ENCRES DU REJEU, résolues une fois par palette — voir l'en-tête d'useReplayInks.
  const {
    teamColorOf, geometry: geometryColor, shot: shotColor, grenade: grenadeColor, neutral: neutralInk, pad: padInk,
    floor: floorStyle, fx: fxInk, grapple: grappleInk, labelStroke, self: selfInk, wall: wallInk, rift: riftInk, mark: markInk,
  } = useReplayInks(paletteVersion)
  // COULEURS DISTINCTES PAR JOUEUR (option du tiroir, 2026-08-24) : une série stable par joueur
  // à la place de la couleur d'équipe — le camp reste dit par les fiches, le fil et le bandeau.
  const distinctColors = useMemo(() => {
    void paletteVersion
    return markerColors === 'player' ? getSeriesColors(doc.roster.length, SERIES_TOKENS) : null
  }, [markerColors, doc.roster.length, paletteVersion])
  // Identité PAR SLOT ET PAR IMAGE : strict pour marqueurs/vies, `OrLast` pour la frontière — cf. useSlotIdentity.
  const { colorOfSlot, colorOfSlotOrLast, colorOfXuid, markOfSlot, nameOfSlot, nameOfXuid, sideOfSlot } = useSlotIdentity({
    doc,
    scoreboard,
    xuidMeta,
    marks,
    teamColorOf,
    neutral: floorStyle.edge,
    distinctColors,
  })

  // PRÉFÉRENCE DE MOUVEMENT RÉDUIT. La feuille de style la respecte pour le DOM ; un canvas
  // dessiné en JS n'est atteint par aucune règle CSS, la préférence se lit donc ici.
  const reducedMotion = useMemo(
    () => typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    [],
  )

  // LE CADRAGE — fond retenu ou écarté, bornes de la scène, largeur de dessin, amplitude
  // verticale, projection partagée et trame d'altitudes : une seule chaîne de décision, qui
  // vit dans `useReplayView` (neuvième extraction imposée par le cliquet de taille). Les noms
  // sortent inchangés : le dessin en dessous lit exactement les mêmes valeurs qu'avant.
  const { mapImage, bounds, renderWidth, renderHeight: viewH, zRange, canvasView, zoom } = useReplayView({
    doc, background, width, freeHeight,
  })

  // Une couleur de série PAR grande zone : la rotation de teinte du POC, en tokens.
  const calloutZones = callouts ?? EMPTY_ZONES
  const zoneColors = useMemo(() => {
    void paletteVersion
    const nBig = calloutZones.reduce((n, z) => n + (z.big ? 1 : 0), 0)
    return nBig > 0 ? getSeriesColors(nBig, SERIES_TOKENS) : []
  }, [calloutZones, paletteVersion])

  // Les OBJECTIFS STATIQUES du mode arrivent AVEC le document (`mapObjectives`, servi à
  // la requête) : normalisés une fois, comme les callouts. Absents = pas de calque.
  const mapObjectives = useMemo(() => normalizeMapObjectives(doc.mapObjectives), [doc.mapObjectives])
  // L'ÉTAT VIVANT DES ZONES (schémas 16-18) : encres, jointure du catalogue, tenue de la jauge (useZoneStates).
  const zones = useZoneStates(mapObjectives, scoreboard, teamColorOf, neutralInk, doc)

  const teamCascades = useTeamCascades(scoreboard, xuidMeta, locale)

  // Traînée, cône, croix de mort, apparition, rémanences et fins de vol : toutes les durées
  // du rejeu, converties une fois pour ce document (useReplayTiming).
  const { baseFps, timing, eventHoldFrames, shotHoldFrames, restWindow } = useReplayTiming(doc)
  // LES EFFETS PRÉCALCULÉS DU FILM (tirs, « ! » du tireur, morts, fins de vol, pulses
  // d'objectif) : cinq listes en coordonnées monde, cuites une fois pour ce document. Elles
  // vivent dans `useReplayFx` — onzième extraction imposée par le cliquet de taille. LE
  // GRAPPIN N'Y EST PLUS depuis le 2026-09-03 : il a rejoint le dash du propulseur juste
  // au-dessous, où les deux gestes de capacité sont bâtis ET peints ensemble.
  const { shotFx, fireMarks, killFx, grenadeRestFx, objectivePulses } =
    useReplayFx(doc, feedKills, timing.aimHold, mapObjectives)
  // LES GESTES DE CAPACITÉ SUR LEUR PORTEUR (grappin, propulseur) : bâtis et peints par un
  // seul hook — ils ne posent rien au sol, ils se lisent sur le pion (cf. useReplayAbilityFx).
  const abilityFx = useReplayAbilityFx({ doc, view: canvasView, grappleInk, colorOfSlot, reducedMotion })
  // La CARTE DE CHALEUR : grille cuite, rampe du thème et lecture réellement servie —
  // toute la logique vit dans le hook, le canvas ne fait que poser le calque.
  useReplayWheelZoom(canvasRef, zoom, canvasView) // molette : memes paliers que les boutons
  const drag = useReplayDrag(zoom, canvasView)
  const heat = useReplayHeatmap(doc, bounds, killFx, {
    show: showHeatmap,
    mode: heatmapMode,
    span: heatmapSpan,
    frameRef,
  })

  // LE REDESSIN, PAR RÉFÉRENCE : les calques statiques doivent pouvoir repeindre la scène
  // apres cuisson, mais ils se declarent AVANT `draw` — ils lisent donc la version courante
  // au moment de l'appel, jamais une capture figee (l'assignation vit avec le redraw plus bas).
  const drawRef = useRef<() => void>(() => {})
  const redraw = useCallback(() => drawRef.current(), [])
  // Vignettes de TYPE de grenade, teintées à l'encre du thème (masques HUD blanc/gris + alpha)
  // et remplies hors rendu, par rang : un rang sans visuel garde l'anneau seul.
  const grenadeIconsRef = useGrenadeIcons(doc.grenadeLabels, floorStyle.edge, redraw)
  // LES CALQUES STATIQUES (sol, zones nommées, chaleur, objectifs), cuits hors écran et
  // recopiés par la boucle : quatre effets qui partageaient la même amorce et recopiaient
  // chacun le cadrage — ils vivent dans useReplayStaticLayers, qui lit `canvasView`.
  const { zonesRef, heatRef, objectivesRef, cookedRef } = useReplayStaticLayers({
    view: canvasView,
    redraw,
    frozen: drag.dragging,
    zones: { zones: calloutZones, bigColors: zoneColors, fineInk: floorStyle.edge, locale },
    heat: { grid: heat.grid, ramp: heat.ramp },
    objectives: { elements: mapObjectives, colorOfTeam: zones.colorOfTeam },
  })

  // LES EMPLACEMENTS D'ARME (schéma 11) : tracé, survol et infobulle dans un seul hook. Ils
  // NE SONT PAS un calque statique — leur position ne bouge pas, mais leur ÉTAT change avec
  // l'image (plein, incertain, vide), donc ils se peignent dans la boucle comme les poses.
  const weaponPads = useReplayWeaponPads({
    doc,
    view: canvasView,
    frameRef,
    enabled: showWeaponPads,
    ink: { neutral: floorStyle.edge, fill: markInk.fill, outline: markInk.outline, family: padInk },
    locale,
    redraw,
  })
  // LES ARMES AU SOL (schéma 27) : les armes ABANDONNÉES — un socle est un LIEU qui réapprovisionne,
  // une arme au sol un OBJET qui ne revient pas. Liseré à l'encre du « aucun camp » (cf. le hook).
  const groundWeapons = useReplayGroundWeapons({
    doc, view: canvasView, enabled: showGroundWeapons,
    ink: { fill: markInk.fill, outline: neutralInk }, redraw,
  })
  // `showNames: true` : le calque des noms a quitte le tiroir le 2026-09-02 (toujours allume).
  const vehicles = useReplayVehicles({ doc, view: canvasView, enabled: showVehicles, showNames: true, showAim, colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, neutralInk, labelStroke, explosionInk: fxInk, reducedMotion, redraw }) // schéma 39 ; prédicat embarqué C7 ; cône du conducteur + nom ET couleur par xuid (2026-09-02) ; explosion de destruction (2026-09-03, en avance de phase).
  // LES POSES D'ÉQUIPEMENT (schéma 10) : comptes, axe de temps, bascules et survol dans un
  // seul hook (useReplayPlacements). Les LÂCHÉS DE PUISSANCE suivent leur bascule, et rien
  // d'autre — plus de garde de mode par-dessus (2026-08-20).
  const placements = useReplayPlacements({
    doc, view: canvasView, frameRef, enabled: showPlacements,
    showUnnamed: showUnnamedPlacements,
    showDropped: showDroppedPlacements,
  })

  // LA VIE DES DRAPEAUX (schéma 15) : tracé, survol et infobulle dans un seul hook — et pas un
  // calque statique, le drapeau porté suit son porteur image par image.
  const flags = useReplayFlagCarries({
    doc, view: canvasView, frameRef, enabled: showFlagCarries,
    scoreboard, teamColorOf, neutral: floorStyle.edge, outline: markInk.outline, reducedMotion,
  })

  const objectiveObjects = useReplayObjectiveObjects({
    lives: doc.objectiveObjects, carries: doc.skullCarries, view: canvasView, ink: neutralInk, outline: markInk.outline,
  })
  // LA COURONNE VIP (schéma 22) : marqueur sur le VIP courant, relu image par image (useReplayVipCrown).
  const vipCrown = useReplayVipCrown({ doc, view: canvasView, enabled: showVipCrown, ink: neutralInk, reducedMotion })
  // LE PORTEUR DU CRÂNE d'Oddball (schéma 23) : crâne sur le porteur courant, relu image par image.
  const skullCarrier = useReplaySkullCarrier({ doc, view: canvasView, enabled: showSkullCarrier, ink: neutralInk, outline: markInk.outline, reducedMotion })
  // LA BOMBE d'Assaut (schéma 30) : portée sur son porteur, au sol au dernier point du lâcheur.
  const bombCarrier = useReplayBombCarrier({ doc, view: canvasView, enabled: showBombCarrier, ink: neutralInk, outline: markInk.outline, reducedMotion })
  // LA FIN DE VOL des grenades (dix-septième extraction — elle paie la déflagration ci-dessous).
  const grenadeRest = useReplayGrenadeRest({ doc, view: canvasView, fx: grenadeRestFx, window: restWindow, ink: fxInk, smoke: floorStyle.edge, halo: grenadeColor, reducedMotion })
  // LA DÉFLAGRATION D'ASSAUT, où et quand elle a eu lieu — seul un match d'Assaut publie la stat.
  const bombBlast = useReplayBombBlast({ doc, view: canvasView, scoreboard, teamColorOf, neutral: floorStyle.edge, reducedMotion })

  /**
   * buildScene LIE chaque calque a l'etat courant du canvas.
   *
   * C'EST UNE TABLE, PAS UN ALGORITHME (exemption R5 assumee) : une entree par calque, aucun
   * embranchement, aucun ordre — l'ordre et les conditions vivent dans `replayCompose`, ou
   * ils sont testes. La decouper par famille n'en ferait pas trois fonctions plus courtes
   * mais trois listes de vingt liaisons a tenir en phase, pour la meme table.
   *
   * TOUT CE QUI DEPEND DE L'IMAGE arrive en ARGUMENT du peintre (`frame`, `dpr`) : la table,
   * elle, ne se relie qu'a ce qui change avec les donnees ou le theme.
   */
  const buildScene = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number): ReplayScene => {
      const view = canvasView
      // LE FOND SUIT LA FENETRE, PAS LA SCENE : sinon il resterait cadre large pendant que les
      // joueurs zooment, et l'image cesserait de designer le meme endroit qu'eux.
      // Pendant un glisser, cuisson gelee : les calques se recopient DECALES (layerOffset).
      const lo = layerOffset(cookedRef.current, canvasView)
      const bgRect = mapImage
        ? backgroundRect(mapImage.calibration, canvasView.bounds, renderWidth, viewH, CANVAS_PAD)
        : null
      // La FENETRE D'EVENEMENT, commune aux tirs, aux grenades, aux pulses et aux morts.
      const win = { frame, hold: eventHoldFrames, frameMs: frameToMs(1, doc) }
      // Un calque CUIT se repose tel quel, decale de `lo` : sa geometrie ne bouge pas.
      const cuit = (c: HTMLCanvasElement | null): LayerPaint => () => {
        if (c) ctx.drawImage(c, lo.x, lo.y, renderWidth, viewH)
      }

      return {
        toggles: { zones: showZones, shotFx: showShotFx, placements: showPlacements, killFx: showKillFx },
        has: {
          background: !!mapImage && !!bgRect,
          floor: !!doc.geometry?.length,
          heat: !!heatRef.current,
          zoneNames: !!zonesRef.current,
          objectivesCooked: !!objectivesRef.current,
          projectiles: !!doc.projectiles?.length,
          placements: placements.counts.drawable > 0,
          fireMarks: fireMarks.length > 0,
          shotFx: shotFx.length > 0,
          grenades: !!doc.grenades?.length,
          zoneStates: doc.zoneStates.length > 0,
          objectivePulses: objectivePulses.length > 0,
          killFx: killFx.length > 0,
        },
        paint: {
          // L'image ENTIERE sur son emprise monde, le canvas rogne le debord : projection
          // affine sans rotation, donc deux coins suffisent (mapBackground.ts).
          'fond-carte': () => {
            if (mapImage && bgRect) {
              ctx.drawImage(mapImage.image, bgRect.x, bgRect.y, bgRect.width, bgRect.height)
            }
          },
          'sol-forge': () =>
            drawGeometryLayer(ctx, doc.geometry ?? [], view, { color: geometryColor, z: zRange }),
          chaleur: cuit(heatRef.current),
          'zones-nommees': cuit(zonesRef.current),
          'objectifs-cuits': cuit(objectivesRef.current),
          projectiles: (_c, fr) =>
            drawProjectilesLayer(ctx, doc.projectiles ?? [], view, fr, grenadeColor),
          'socles-armes': weaponPads.paint,
          'armes-au-sol': groundWeapons.paint,
          // La fenetre d'une pose n'est PAS [t0, t1] : `t1` date la mise au repos, pas la
          // disparition (placementEndFrame) ; le ping bat en TEMPS de match, pas en images.
          'poses-equipement': (_c, fr, k) =>
            drawEquipmentPlacementsLayer(
              ctx,
              // Les VIES et leur CAMP voyagent avec les poses : le ping du capteur revele les
              // adversaires du poseur, et « adversaire » est une relation entre deux vies. Le
              // camp est celui de la base (`team_side`), jamais le drapeau « allie » de la page.
              { placements: doc.equipmentPlacements, lives: doc.tracks, sideOfSlot, rift: placements.rift },
              view,
              { frame: fr, ...placements.windowTime, k, reducedMotion, ...placements.toggles },
              // FRONTIERE : objet lache a la mort, `t0 = finVie+1` — `colorOfSlotOrLast`.
              { colorOfSlot: colorOfSlotOrLast, neutral: floorStyle.edge, wall: wallInk, rift: riftInk },
            ),
          vehicules: vehicles.paint,
          trajectoires: (_c, fr, k) =>
            drawTracksLayer(ctx, doc.tracks, view, {
              colorOfSlot,
              ink: floorStyle.edge,
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
              labelStroke, embarkedAtSlot: vehicles.isEmbarkedAt, // PION EMBARQUE (C7).
            }),
          'marques-de-tir': (_c, fr, k) =>
            drawFireMarks(ctx, fireMarks, view, {
              frame: fr, hold: shotHoldFrames, colorOfSlot, ink: labelStroke || floorStyle.edge, k,
            }),
          'gestes-capacite': abilityFx.paint,
          // vehicleSizeOf : origine des tirs en vehicule sur LA MEME source de tailles que le
          // calque vehicules (`useReplayVehicles.sizeOf`), jamais un second chargement.
          tirs: (_c, _fr, k) =>
            drawShotsLayer(ctx, shotFx, view, { ...win, hold: shotHoldFrames }, {
              ink: fxInk, k, reducedMotion, vehicleSizeOf: vehicles.sizeOf,
            }),
          grenades: () =>
            drawGrenadesLayer(ctx, doc.grenades ?? [], view, win, {
              color: grenadeColor,
              iconOf: (rank) => grenadeIconsRef.current.get(rank) ?? null,
            }),
          'fin-de-vol': grenadeRest.paint,
          'etat-zones': (_c, fr) => drawZoneStates(ctx, zones, doc.zoneStates, view, fr),
          drapeaux: flags.paint,
          'objets-objectif': objectiveObjects.paint,
          'couronne-vip': vipCrown.paint,
          'crane-porte': skullCarrier.paint,
          'bombe-portee': bombCarrier.paint,
          deflagration: bombBlast.paint,
          'pulses-objectif': () =>
            drawObjectivePulses(ctx, objectivePulses, view, win, { colorOfTeam: zones.colorOfTeam }, reducedMotion),
          morts: (_c, _fr, k) =>
            drawKillFxLayer(ctx, killFx, view, win, {
              colorOfSlot: colorOfSlotOrLast, // FRONTIERE : kill posthume/echange apres la fin de vie.
              fallback: shotColor, reducedMotion, k,
            }),
        },
      }
    },
    [
      doc, geometryColor, zRange, timing, wallInk, riftInk,
    // Refs STABLES : la regle de dependances ne le sait pas d'un hook maison.
    zonesRef, heatRef, objectivesRef, grenadeIconsRef, cookedRef,
    renderWidth, viewH, canvasView,
    placements.counts.drawable,
    placements.windowTime,
    placements.toggles,
    showPlacements,
    // Le TRACÉ seul, jamais l'objet du hook : `hover` change à chaque mouvement de pointeur,
    // et le mettre ici ferait recuire `draw` (donc toute la scène) pour une infobulle.
    weaponPads,
    groundWeapons, vehicles,
    shotColor,
    grenadeColor,
    eventHoldFrames,
    shotHoldFrames,
    shotFx,
    fireMarks,
    fxInk,
    abilityFx,
    killFx,
    grenadeRest,
    objectivePulses,
    placements.rift,
    zones,
    flags,
    objectiveObjects,
    vipCrown,
    skullCarrier,
    bombCarrier,
    bombBlast,
    floorStyle.edge,
    colorOfSlot,
    colorOfSlotOrLast,
    sideOfSlot,
    markOfSlot,
    nameOfSlot,
    showTrail,
    selfInk,
    labelStroke,
    reducedMotion,
    showAim,
    showZones,
    showShotFx,
    showKillFx,
    mapImage,
    ],
  )

  /**
   * draw DIMENSIONNE, EFFACE, COMPOSE, TIQUE — et rien d'autre.
   *
   * La scene se lie juste avant (`buildScene`), l'ordre et les bascules se decident dans
   * `replayCompose` : cette fonction n'a plus a savoir ce qu'il y a a peindre.
   */
  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || renderWidth === 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const dpr = (window.devicePixelRatio || 1) * exportRenderScale.current
    const pw = Math.round(renderWidth * dpr)
    const ph = Math.round(viewH * dpr)
    if (canvas.width !== pw || canvas.height !== ph) {
      canvas.width = pw
      canvas.height = ph
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    ctx.clearRect(0, 0, renderWidth, viewH)

    const frame = frameRef.current
    composeScene(ctx, sceneLayers(buildScene(ctx, frame)), frame, dpr)

    // L'HORLOGE ET LA PUBLICATION DE L'IMAGE, en dernier : elles disent ou en est la scene
    // qu'on vient de peindre (cf. useReplayClock).
    clockTick(frame)
  }, [buildScene, clockTick, renderWidth, viewH])


  // Redraw hors animation (thème, resize, données, pause), et publication de la version
  // courante de `draw` aux calques statiques (cf. drawRef plus haut).
  useEffect(() => {
    drawRef.current = draw
    draw()
  }, [draw])

  // LA LECTURE (état lu/pause, boucle rAF, curseur de la frise, ARRÊT SUR L'ÉTAT FINAL) vit
  // dans useReplayPlayback : le canvas garde le DESSIN, le hook porte le TEMPS.
  const playback = useReplayPlayback({
    doc, playWindow, baseFps, speed: multiplier, renderWidth, frameRef, draw,
    soundTick: sound.tick, onEnded: sound.endMatch, onTransportGesture: sound.wake, onPlayingChange: sound.setTransportPlaying,
  })
  // LA FRISE ET SON CLAVIER (planche 2a) vivent dans useReplayTimeline — treizième extraction
  // imposée par le cliquet : pistes, dominance, médias, horloges et raccourcis sont LA FRISE.
  const timeline = useReplayTimeline({
    doc, playWindow, feedEntries, media, marks: marks ?? NO_MARKS, renderWidth, locale,
    lead: teamCascades, playback, toggleSound: sound.toggle, zoom,
  })
  // LE TIROIR, groupé de même (useReplayDrawer) : les disponibilités viennent des calques, les
  // bascules de `useReplaySettings`, et l'état d'ouverture du hook lui-même (2026-08-30).
  const drawer = useReplayDrawer({
    settings, sound, locale,
    heat: { mode: heat.mode, killsAvailable: heat.killsAvailable },
    available: {
      zones: calloutZones.length > 0,
      placements: {
        drawable: placements.counts.drawable > 0,
        unnamed: placements.counts.unnamed > 0,
        dropped: placements.counts.dropped > 0,
      },
      weaponPads: weaponPads.available,
      groundWeapons: groundWeapons.available,
      flagCarries: flags.available,
      vipCrown: vipCrown.available,
      skullCarrier: skullCarrier.available,
      bombCarrier: bombCarrier.available, vehicles: vehicles.available,
    },
  })
  // CE QUI SORT DU REJEU (image, vidéo) vit dans useReplayCapture : le canvas prête sa TOILE, son
  // HORLOGE et sa lecture. `play` reçoit `togglePlay` — appelé sur une PAUSE seule, il vaut lecture.
  const capture = useReplayCapture({
    canvasRef, doc, frameRef, playing: playback.playing, play: playback.togglePlay,
    audioTrack: sound.recordingTrack, soundTrack: sound.exportTrack, soundVolume: sound.volume,
    redraw, playWindow, scoreboard, xuidMeta, outcome, locale,
  })

  return (
    // RELATIVE + OVERFLOW-HIDDEN : le tiroir se pose EN SURIMPRESSION dans ce cadre (retour
    // de planche du 16/08). `relative` lui donne son repère, `overflow-hidden` retient ses
    // coins carrés dans le rayon arrondi de la carte.
    <div className="relative overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex">
        {/* Colonne canvas : c'est ELLE que le ResizeObserver mesure. Elle occupe DÉSORMAIS
            toute la largeur du cadre, ouvert ou fermé — le panneau la recouvre au lieu de
            lui prendre sa place, donc le canvas ne se retaille plus et le rendu ne saute
            plus à l'ouverture. */}
        <div ref={containerRef} className="min-w-0 flex-1">
          {/* Le bouton du TIROIR DE RÉGLAGES vit tout à droite de la barre de lecture
              (demande du 2026-08-24) : la barre du haut, qui ne portait plus que lui, est
              supprimée — le rejeu gagne sa hauteur. */}
          <div className="relative p-3 pb-0">
            {/* La légende se pose DANS le cadre du canvas (coin bas-gauche) : une échelle
                de couleur lue à côté de sa carte n'est plus une échelle. Le conteneur
                relatif n'existe que pour elle — sans carte de chaleur, rien n'y flotte. */}
            {/* TROIS CALQUES SURVOLABLES SUR UNE SEULE BALISE (poses, emplacements d'arme,
                drapeaux) : chacun rejoue le survol sur SA donnée, le canvas passe le geste. */}
            <div className="relative mx-auto" style={{ width: renderWidth || '100%' }}>
              <canvas
                ref={canvasRef}
                className="block"
                style={{ width: renderWidth || '100%', height: viewH }}
                {...hoverHandlers([placements.hover, weaponPads, flags], drag)}
              />
              {/* Les infobulles des trois calques survolables (cf. ReplayCanvasTips). */}
              <ReplayCanvasTips
                locale={locale} width={renderWidth} ownerNameOf={nameOfSlot} playWindow={playWindow}
                placement={placements.hover.hover} pad={weaponPads.hover} flag={flags.hover}
              />
            </div>
            {/* LA LÉGENDE S'ANCRE AU BLOC, PLUS À LA TOILE (2026-09-02, retour utilisateur :
                « il faut le mettre sur le côté gauche du bloc avec un léger padding »). Elle
                vivait DANS le conteneur `mx-auto` de la toile — qui est CENTRÉ et souvent plus
                étroit que la colonne : sur une carte allongée dans un écran large, la légende
                flottait donc au milieu du cadre, à gauche de la carte mais loin du bord.
                Remontée d'un cran, elle se cale sur le bord gauche du bloc, où on la cherche. */}
            {heat.grid && <ReplayHeatmapLegend locale={locale} mode={heat.grid.mode} />}
            <ReplayZoomControl zoom={zoom} locale={locale} />
          </div>
          {/* LA BARRE DE LECTURE, SORTIE DU `p-3` LE 2026-09-02 : dedans, 12 px de carte
              l'encadraient de trois côtés ; ici elle va bord à bord. Elle reste DANS
              `containerRef` — c'est ce qui permet à `useReplayViewport` de déduire le chrome
              par soustraction, donc de rendre au terrain ce que la barre économise. */}
          <ReplayTransport
            playing={playback.playing} onTogglePlay={playback.togglePlay}
            onRestart={playback.restart} onSeekBy={playback.seekBy}
            clockRef={clockRef} timeline={timeline}
            autoPlay={settings.autoPlay} onToggleAutoPlay={settings.toggleAutoPlay}
            speed={multiplier} onSetSpeed={settings.setSpeed}
            sound={sound} capture={capture} locale={locale}
            settingsOpen={drawer.open} onToggleSettings={drawer.toggle}
            settingsButtonRef={drawer.buttonRef}
          />
        </div>
        {/* Le panneau se pose SUR la carte, à droite (retour de planche du 16/08 : « je vois
            plus un panneau par dessus »). Il ne mange donc plus la largeur du canvas — et il
            laisse libre le coin BAS-GAUCHE, où vit la légende de la carte de chaleur. */}
        {drawer.open && <ReplaySettingsDrawer {...drawer.panel} />}
      </div>
    </div>
  )
}
