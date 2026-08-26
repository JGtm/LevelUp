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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { getSeriesColors } from '@/lib/accessibility/plotlyColorscale'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { MatchScoreboardRow, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { XuidMeta } from '@/features/match-view/xuidMeta'

import type { CalloutZoneReady } from './calloutsLayer'

import { ReplayHeatmapLegend } from './ReplayHeatmapLegend'
import { ReplayTransport } from './ReplayTransport'
import { useLeadMarks } from './useLeadMarks'
import { buildShotFx } from './shotFx'
import {
  buildObjectivePulses,
  drawObjectivePulses,
  normalizeMapObjectives,
} from './objectivesLayer'
import { drawZoneStates } from './zoneStatesLayer'
import { buildFireMarks, drawFireMarks } from './fireMark'
import { buildGrappleFx, drawGrappleLayer } from './grappleLayer'
import { drawEquipmentPlacementsLayer } from './equipmentPlacementsLayer'
import { ReplayCanvasTips } from './ReplayCanvasTips'
import { useReplayPlacements } from './useReplayPlacements'
import { useReplayFlagCarries } from './useReplayFlagCarries'
import { useGrenadeIcons } from './useGrenadeIcons'
import { useZoneStates } from './useZoneStates'
import { useReplayWeaponPads } from './useReplayWeaponPads'
import { buildGrenadeRestFx } from './grenadeFx'
import type { ReplayLocale } from './i18n'
import { buildKillFx } from './killFx'
import type { PlayerMarkKind } from './playerMarks'
import { useSlotIdentity } from './useSlotIdentity'
import { ReplaySettingsDrawer } from './ReplaySettingsDrawer'
import { useReplayHeatmap } from './useReplayHeatmap'
import { useReplayInks } from './useReplayInks'
import { useReplayPlayback } from './useReplayPlayback'
import { useReplayStaticLayers } from './useReplayStaticLayers'
import { useReplaySettings } from './useReplaySettings'
import { useReplaySound } from './useReplaySound'
import { backgroundRect, coversPlayedArea } from './mapBackground'
import { buildFloorGrid } from './mapFloor'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  drawGeometryLayer,
  drawGrenadesLayer,
  drawKillFxLayer,
  drawShotsLayer,
} from './replayDraw'
import { drawGrenadeRestLayer } from './grenadeRestLayer'
import { fitWidth, formatClock, frameToMs, sceneBounds } from './replayLogic'
import { drawProjectilesLayer } from './replayProjectiles'
import { drawTracksLayer } from './replayMarkers'
import { useReplayTiming } from './useReplayTiming'

// 8 tokens de série : une teinte par GRANDE ZONE NOMMÉE (cyclés au-delà de 8 via
// getSeriesColors), et — depuis le 2026-08-24 — la palette des COULEURS DISTINCTES PAR
// JOUEUR quand l'option du tiroir la choisit. Par défaut un joueur porte la couleur de
// son ÉQUIPE (D1).
const SERIES_TOKENS: SemanticToken[] = [
  'chart-series-1', 'chart-series-2', 'chart-series-3', 'chart-series-4',
  'chart-series-5', 'chart-series-6', 'chart-series-7', 'chart-series-8',
]
// Les TOKENS des encres du canvas vivent avec elles, dans useReplayInks ; les DURÉES et leur
// conversion en images, dans useReplayTiming.
const CANVAS_HEIGHT = 480
const CANVAS_PAD = 24

/** Référence STABLE pour « pas de zones » : un `?? []` inline recuirait le calque à chaque rendu. */
const EMPTY_ZONES: CalloutZoneReady[] = []

/**
 * Cadence de publication de l'image courante vers React, en millisecondes.
 *
 * POURQUOI PAS À CHAQUE IMAGE. Le canvas se redessine à la cadence de l'écran ; les fiches
 * joueur, elles, sont du DOM. Les re-rendre 60 fois par seconde coûterait tout le budget
 * d'animation pour un contenu qui change à peine. 150 ms reste bien en deçà de ce que l'œil
 * perçoit comme un retard sur un compteur, et divise le travail de React par dix.
 */
const FRAME_PUBLISH_MS = 150

/**
 * Le FOND DE CARTE : l'image cuite de la carte, et le calage qui la pose dans le repère
 * monde du rejeu. Les deux voyagent ensemble — une image sans calage ne se superpose à rien,
 * et l'appelant ne doit jamais pouvoir en fournir une seule.
 */
export interface ReplayMapBackgroundLayer {
  calibration: ReplayMapBackgroundCalibration
  image: HTMLImageElement
}

interface ReplayCanvasProps {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  /**
   * Kills du match (mêmes events que le fil, horloge gameplay) : la carte en tire les
   * EFFETS DE MORT orientés tueur -> victime. Absents = pas d'effet, jamais une erreur.
   */
  kills?: KillEvent[]
  /** Offset du countdown pré-match, en ms (`header.t0_ms`) — même recalage que le fil. */
  t0Ms?: number
  /** Appelé à cadence réduite avec l'image courante : sert aux panneaux hors canvas. */
  onFrameChange?: (frame: number) => void
  /**
   * Fond de carte figé. Absent = la carte n'a pas d'image (seules 21 en ont) : le rejeu
   * garde son sol reconstruit, qui reste lisible. Ce n'est pas un mode dégradé caché,
   * c'est le cas nominal des autres cartes.
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
}

export function ReplayCanvas({
  doc, locale, kills, t0Ms, onFrameChange, background, callouts, scoreboard, xuidMeta, marks,
}: ReplayCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const clockRef = useRef<HTMLSpanElement>(null)
  const frameRef = useRef(0)
  const publishedAtRef = useRef(0)

  const [width, setWidth] = useState(0)
  // TIROIR DE RÉGLAGES (décision utilisateur du 16/08) : fermé par défaut, ouvert par un
  // bouton unique dans la barre. Calques, effets et vitesse persistés comme le son — même
  // mécanisme (replayPreferences.ts), des réglages distincts.
  const [settingsOpen, setSettingsOpen] = useState(false)
  // Le bouton du tiroir : il est exclu du « clic dehors » et REPREND le focus à la
  // fermeture — sans quoi le focus retomberait au document et la navigation au clavier
  // repartirait du haut de la page.
  const settingsButtonRef = useRef<HTMLButtonElement>(null)
  const closeSettings = useCallback(() => {
    setSettingsOpen(false)
    settingsButtonRef.current?.focus({ preventScroll: true })
  }, [])
  const {
    showAim, toggleAim, showZones, toggleZones, showNames, toggleNames,
    showTrail, toggleTrail,
    showHeatmap, toggleHeatmap, heatmapMode, setHeatmapMode, heatmapSpan, setHeatmapSpan,
    showShotFx, toggleShotFx, showKillFx, toggleKillFx,
    showPlacements, togglePlacements,
    showUnnamedPlacements, toggleUnnamedPlacements,
    showDroppedPlacements, toggleDroppedPlacements,
    showWeaponPads, toggleWeaponPads, showFlagCarries, toggleFlagCarries,
    speed: multiplier, setSpeed: setMultiplier,
    markerColors, setMarkerColors,
  } = useReplaySettings()
  // SON : coupé par défaut, préférence, volume et filtre par catégorie persistés, tout le
  // câblage dans le hook (règles dans replaySound.ts, lecture Web Audio dans replayAudio.ts).
  const sound = useReplaySound(doc, kills, t0Ms, multiplier)

  const paletteVersion = useColorPaletteVersion()
  // TOUTES LES ENCRES DU REJEU, résolues une fois par palette (useReplayInks) : couleurs
  // d'équipe, fond de carte, lancers, sol, teintes d'éclair, grappin, contour des noms, et le
  // double contour du joueur de la page. Elles partageaient huit fois le même corps ici — voir
  // l'en-tête du hook.
  const {
    teamColorOf, geometry: geometryColor, shot: shotColor, grenade: grenadeColor, neutral: neutralInk, pad: padInk,
    floor: floorStyle, fx: fxInk, grapple: grappleInk, labelStroke, self: selfInk, wall: wallInk, mark: markInk,
  } = useReplayInks(paletteVersion)
  // Les tractions de grappin, jointes une fois aux points de leur vie (schéma 8).
  const grappleFx = useMemo(() => buildGrappleFx(doc), [doc])
  // COULEURS DISTINCTES PAR JOUEUR (option du tiroir, 2026-08-24) : une couleur de série
  // stable par joueur du roster, à la place de la couleur d'équipe — pour suivre quelqu'un
  // dans la mêlée. Le camp reste dit par les fiches, le fil et le bandeau.
  const distinctColors = useMemo(() => {
    void paletteVersion
    return markerColors === 'player' ? getSeriesColors(doc.roster.length, SERIES_TOKENS) : null
  }, [markerColors, doc.roster.length, paletteVersion])
  // Couleur, marque et nom PAR SLOT : un tir et une mort se dessinent dans la teinte de leur
  // auteur, et c'est elle qui permet de suivre un joueur des yeux. Le calcul (jointure au
  // scoreboard + descente sur les vies) vit dans useSlotIdentity.
  const { colorOfSlot, slotColors, markOfSlot, nameOfSlot, sideOfSlot } = useSlotIdentity({
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

  const bounds = useMemo(() => sceneBounds(doc), [doc])

  // LE FOND DE CARTE PREND LA PLACE DU SOL RECONSTRUIT, il ne s'y ajoute pas : l'image
  // porte la carte telle que le jeu la dessine, la trame d'altitudes n'en est que
  // l'approximation. Les superposer ne ferait que voiler la meilleure des deux.
  //
  // Il est ÉCARTÉ quand il ne recouvre pas la zone jouée : un fond qui ne contient pas le
  // terrain n'est pas un défaut d'affichage, c'est le signe que les deux repères ne sont
  // pas le même — mieux vaut alors le sol reconstruit qu'une carte posée à côté des joueurs.
  const mapImage = useMemo(() => {
    if (!background) return null
    return coversPlayedArea(background.calibration, doc.bounds) ? background : null
  }, [background, doc.bounds])

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
  // Les PULSES d'action d'objectif (doc.objectives, rendu nulle part avant le lot 4.4) :
  // précalculés en monde, comme les effets de mort.
  const objectivePulses = useMemo(
    () => buildObjectivePulses(doc, mapObjectives),
    [doc, mapObjectives],
  )
  // L'ÉTAT VIVANT DES ZONES (schémas 16-18) : encres, jointure du catalogue, tenue de la jauge (useZoneStates).
  const zones = useZoneStates(mapObjectives, scoreboard, teamColorOf, neutralInk, doc)

  const leadMarks = useLeadMarks(doc, scoreboard, xuidMeta, locale)

  // La trame d'altitudes ne dépend QUE du document : construite une fois, pas à chaque resize.
  const floorGrid = useMemo(
    () => (!mapImage && doc.structure?.length ? buildFloorGrid(doc.structure, doc.bounds) : null),
    [doc.structure, doc.bounds, mapImage],
  )
  // Largeur de dessin = ratio de la scène à hauteur fixée (évite les marges latérales).
  const renderWidth = useMemo(
    () => (width === 0 ? 0 : Math.floor(fitWidth(bounds, width, CANVAS_HEIGHT, CANVAS_PAD))),
    [bounds, width],
  )
  const zRange = useMemo(
    () => ({ min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 }),
    [doc.bounds.minZ, doc.bounds.maxZ],
  )
  // LE CADRAGE, une fois : le dessin ET le survol doivent lire la MÊME projection — un
  // pointeur qui viserait un autre cadre que celui peint ne toucherait rien.
  const canvasView = useMemo(
    () => ({ bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD }),
    [bounds, renderWidth],
  )
  // Traînée, cône, croix de mort, apparition, rémanences et fins de vol : toutes les durées
  // du rejeu, converties une fois pour ce document (useReplayTiming).
  const { baseFps, timing, eventHoldFrames, shotHoldFrames, restWindow } = useReplayTiming(doc)
  // Les tirs sont PRÉCALCULÉS comme les morts : famille, teinte et REGARD du tireur résolus
  // une fois au chargement (mesure : la couverture d'orientation passe de 18,6 % à 100 % sur
  // le film témoin en relisant le regard plutôt que le champ de l'événement).
  const shotFx = useMemo(() => buildShotFx(doc, timing.aimHold), [doc, timing.aimHold])
  // Le « ! » dans le point du tireur (demande du 2026-08-24) : mêmes tirs, même fenêtre
  // que l'éclair de bouche — deux effets du même événement (cf. fireMark.ts).
  const fireMarks = useMemo(() => buildFireMarks(doc), [doc])
  // Les effets de mort sont PRÉCALCULÉS en monde (positions relues une fois, patron POC) :
  // pendant la lecture, seul le passage monde -> pixels reste à faire.
  const killFx = useMemo(
    () => buildKillFx(doc, kills ?? [], t0Ms ?? 0),
    [doc, kills, t0Ms],
  )
  // La CARTE DE CHALEUR : grille cuite, rampe du thème et lecture réellement servie —
  // toute la logique vit dans le hook, le canvas ne fait que poser le calque.
  const heat = useReplayHeatmap(doc, bounds, killFx, {
    show: showHeatmap,
    mode: heatmapMode,
    span: heatmapSpan,
    frameRef,
  })
  // Fins de vol de grenade : le lien lancer -> projectile est dans l'artefact (v3).
  const grenadeRestFx = useMemo(() => buildGrenadeRestFx(doc), [doc])
  const totalLabel = formatClock(doc.durationMs ?? frameToMs(doc.frameCount, doc))

  // Largeur responsive (ResizeObserver du conteneur).
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const ro = new ResizeObserver((entries) => {
      setWidth(Math.max(Math.floor(entries[0]?.contentRect.width ?? 0), 0))
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

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
  const { floorRef, zonesRef, heatRef, objectivesRef } = useReplayStaticLayers({
    view: canvasView,
    redraw,
    floor: { grid: floorGrid, style: floorStyle },
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
    scoreboard, teamColorOf, neutral: floorStyle.edge, reducedMotion,
  })

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas || renderWidth === 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const dpr = window.devicePixelRatio || 1
    const pw = Math.round(renderWidth * dpr)
    const ph = Math.round(CANVAS_HEIGHT * dpr)
    if (canvas.width !== pw || canvas.height !== ph) {
      canvas.width = pw
      canvas.height = ph
    }
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    ctx.clearRect(0, 0, renderWidth, CANVAS_HEIGHT)

    const view = canvasView
    const frame = frameRef.current
    // ORDRE DES CALQUES, du fond vers le sujet : le sol porte les trajectoires, qui portent les
    // événements. Inverser noierait les joueurs.
    const bgRect = mapImage
      ? backgroundRect(mapImage.calibration, bounds, renderWidth, CANVAS_HEIGHT, CANVAS_PAD)
      : null
    if (mapImage && bgRect) {
      // L'image ENTIÈRE est posée sur son emprise monde ; le canvas rogne le débord. La
      // projection est affine et sans rotation, donc deux coins suffisent (mapBackground.ts).
      ctx.drawImage(mapImage.image, bgRect.x, bgRect.y, bgRect.width, bgRect.height)
    } else if (floorRef.current) {
      ctx.drawImage(floorRef.current, 0, 0, renderWidth, CANVAS_HEIGHT)
    } else if (doc.geometry?.length) {
      // REPLI, pas un doublon : sans fichier de structure figé, la carte n'a pas de sol
      // reconstruit et les props Forge redeviennent le seul repère disponible. Ils couvrent
      // 3,4 % du terrain — c'est peu, et c'est mieux qu'un fond vide.
      drawGeometryLayer(ctx, doc.geometry, view, { color: geometryColor, z: zRange })
    }
    // La CARTE DE CHALEUR juste au-dessus du fond : c'est une lecture du terrain, elle se
    // pose SUR la carte et SOUS tout ce qui la nomme ou s'y déplace. Elle ne masque rien —
    // son opacité est bornée, et elle laisse le décor transparaître (heatmapLayer.ts).
    if (heatRef.current) {
      ctx.drawImage(heatRef.current, 0, 0, renderWidth, CANVAS_HEIGHT)
    }
    // Les ZONES NOMMÉES par-dessus le fond, sous tout ce qui bouge : c'est le vocabulaire
    // du terrain, pas un événement. Calque statique recopié (cuit hors écran).
    if (showZones && zonesRef.current) {
      ctx.drawImage(zonesRef.current, 0, 0, renderWidth, CANVAS_HEIGHT)
    }
    // Les OBJECTIFS DU MODE par-dessus le vocabulaire : c'est l'enjeu du match (zones de
    // capture, apparitions de drapeau), il prime sur les noms de lieux. Statique aussi.
    if (objectivesRef.current) {
      ctx.drawImage(objectivesRef.current, 0, 0, renderWidth, CANVAS_HEIGHT)
    }
    // Les projectiles passent SOUS les joueurs : ce sont des objets du terrain, pas le sujet.
    if (doc.projectiles?.length) {
      drawProjectilesLayer(ctx, doc.projectiles, view, frame, grenadeColor)
    }
    // Les EMPLACEMENTS D'ARME juste au-dessus du terrain et SOUS les poses : un socle est un
    // MEUBLE de la carte, il précède ce qu'un joueur y dépose comme ce qui s'y déplace.
    weaponPads.paint(ctx, frame, dpr)
    // Les POSES D'ÉQUIPEMENT, au-dessus du terrain (fond, zones, chaleur, objectifs) et SOUS
    // les marqueurs de joueurs : un mur est un objet POSÉ sur la carte — il appartient au
    // décor du moment, pas au sujet. Sa fenêtre d'affichage n'est PAS [t0, t1] : `t1` date la
    // mise au repos de l'objet, pas sa disparition (cf. placementEndFrame).
    if (showPlacements && placements.counts.drawable > 0) {
      drawEquipmentPlacementsLayer(
        ctx,
        // Les VIES et leur CAMP voyagent avec les poses : le ping du capteur révèle les
        // adversaires du poseur, et « adversaire » est une relation entre deux vies. Le camp
        // est celui de la base (`team_side`), jamais le drapeau « allié » vu de la page.
        { placements: doc.equipmentPlacements, lives: doc.tracks, sideOfSlot },
        view,
        {
          frame,
          // Durée RÉELLE d'une frame : le ping du capteur bat en temps de match, pas en
          // nombre d'images (même règle que la fin de vol des grenades). Le même objet sert
          // au survol — une pose ne peut pas être dessinée et non survolable.
          ...placements.windowTime,
          k: dpr,
          reducedMotion,
          ...placements.toggles,
        },
        { colorOfSlot: (slot) => slotColors.get(slot) ?? null, neutral: floorStyle.edge, wall: wallInk },
      )
    }
    drawTracksLayer(ctx, doc.tracks, view, {
      colorOfSlot,
      ink: floorStyle.edge,
      frame,
      timing,
      z: zRange,
      k: dpr,
      showAim,
      markOfSlot,
      nameOfSlot,
      showNames,
      showTrail,
      selfInk,
      deathInk: shotColor,
      labelStroke,
    })
    // Le « ! » PAR-DESSUS le marqueur du tireur, centré dans le noyau : il se lit sur le
    // point, il se dessine donc juste après lui. Même interrupteur que l'éclair de bouche
    // (« Effets de tirs ») : c'est le même événement, un seul geste l'éteint.
    if (showShotFx && fireMarks.length > 0) {
      drawFireMarks(ctx, fireMarks, view, {
        frame, hold: shotHoldFrames, colorOfSlot, ink: labelStroke || floorStyle.edge, k: dpr,
      })
    }

    // La LIGNE DE GRAPPIN juste au-dessus des trajectoires et SOUS les effets de tir :
    // c'est un lien joueur -> point d'accroche, il se lit sur la trajectoire sans couvrir
    // les événements. Fenêtre MESURÉE [t0, t1] : la ligne suit le joueur qui se déplace
    // vers l'ancre, puis disparaît à l'arrivée. Statique par frame (reduced-motion par
    // construction).
    if (grappleFx.length > 0) {
      drawGrappleLayer(ctx, grappleFx, view, frame, grappleInk)
    }
    // Les événements passent APRÈS les trajectoires : ils se lisent sur elles. Les DEUX
    // effets d'événement sont ÉTEIGNABLES depuis le tiroir (décision du 16/08) : c'est le
    // DESSIN qui s'éteint, jamais la mesure — `killFx` continue d'alimenter la lecture
    // « éliminations » de la carte de chaleur, qui n'est pas un effet.
    const win = { frame, hold: eventHoldFrames, frameMs: frameToMs(1, doc) }
    if (showShotFx && shotFx.length > 0) {
      drawShotsLayer(ctx, shotFx, view, { ...win, hold: shotHoldFrames }, {
        ink: fxInk,
        k: dpr,
        reducedMotion,
      })
    }
    if (doc.grenades?.length) {
      drawGrenadesLayer(ctx, doc.grenades, view, win, {
        color: grenadeColor,
        iconOf: (rank) => grenadeIconsRef.current.get(rank) ?? null,
      })
    }
    // La FIN DE VOL après le lancer : halo « dernière position connue » (jamais un
    // impact — aucun événement de détonation dans le film), nappe électrique persistante
    // pour la Shock/Dynamo.
    if (grenadeRestFx.length > 0) {
      drawGrenadeRestLayer(
        ctx,
        grenadeRestFx,
        view,
        {
          frame,
          holdHalo: restWindow.holdHalo,
          holdDynamo: restWindow.holdDynamo,
          // Durée réelle d'UNE frame : `frameToMs` porte déjà le repli des artefacts sans
          // échelle temporelle. La lire ici plutôt que le champ brut évite qu'une explosion
          // reste figée à l'âge zéro sur un artefact ancien.
          frameMs: frameToMs(1, doc),
        },
        {
          ink: fxInk,
          smoke: floorStyle.edge,
          halo: grenadeColor,
          k: dpr,
          reducedMotion,
        },
      )
    }
    // L'ÉTAT DES ZONES à l'image courante (schémas 16-18) : teinte du camp qui la tient,
    // surbrillance de la colline ACTIVE, arc de la JAUGE EN DIRECT. Il se peint dans la
    // boucle et non dans un calque cuit : la géométrie ne bouge pas, l'état si. Le calque lui-même
    // refuse de peindre si le catalogue de l'artefact ne joint pas la liste servie.
    if (doc.zoneStates.length > 0) drawZoneStates(ctx, zones, doc.zoneStates, view, frame)
    // LES DRAPEAUX par-dessus les zones et SOUS les morts : l'enjeu du mode prime sur le
    // terrain, mais une élimination reste l'événement le plus lourd de sens du calque.
    flags.paint(ctx, frame)
    // Le PULSE D'ACTION D'OBJECTIF (capture, retour, prise de zone) : un anneau qui
    // s'ouvre depuis la zone/le marqueur concerné à l'instant de l'action (lot 4.4).
    if (objectivePulses.length > 0) {
      drawObjectivePulses(ctx, objectivePulses, view, win,
        { colorOfTeam: zones.colorOfTeam }, reducedMotion)
    }
    // Les MORTS par-dessus les tirs : c'est l'événement le plus lourd de sens du calque,
    // et le seul dont l'extrémité pointe une vraie victime (couple complet, règle 89/93).
    if (showKillFx && killFx.length > 0) {
      drawKillFxLayer(ctx, killFx, view, win, {
        colorOfSlot: (slot) => slotColors.get(slot) ?? null,
        fallback: shotColor,
        reducedMotion, k: dpr,
      })
    }

    if (clockRef.current) {
      clockRef.current.textContent = `${formatClock(frameToMs(frame, doc))} / ${totalLabel}`
    }
    if (onFrameChange) {
      const now = performance.now()
      if (now - publishedAtRef.current >= FRAME_PUBLISH_MS) {
        publishedAtRef.current = now
        onFrameChange(Math.floor(frame))
      }
    }
  }, [
    doc, geometryColor, bounds, zRange, timing, totalLabel, wallInk,
    // Refs STABLES : la regle de dependances ne le sait pas d'un hook maison.
    floorRef, zonesRef, heatRef, objectivesRef, grenadeIconsRef,
    renderWidth, canvasView,
    placements.counts.drawable,
    placements.windowTime,
    placements.toggles,
    showPlacements,
    // Le TRACÉ seul, jamais l'objet du hook : `hover` change à chaque mouvement de pointeur,
    // et le mettre ici ferait recuire `draw` (donc toute la scène) pour une infobulle.
    weaponPads,
    shotColor,
    grenadeColor,
    eventHoldFrames,
    shotHoldFrames,
    shotFx,
    fireMarks,
    fxInk,
    grappleFx,
    grappleInk,
    killFx,
    grenadeRestFx,
    restWindow,
    objectivePulses,
    zones,
    flags,
    floorStyle.edge,
    slotColors,
    colorOfSlot,
    sideOfSlot,
    markOfSlot,
    nameOfSlot,
    showNames,
    showTrail,
    selfInk,
    labelStroke,
    reducedMotion,
    showAim,
    showZones,
    showShotFx,
    showKillFx,
    onFrameChange,
    mapImage,
  ])

  // Redraw hors animation (thème, resize, données, pause), et publication de la version
  // courante de `draw` aux calques statiques (cf. drawRef plus haut).
  useEffect(() => {
    drawRef.current = draw
    draw()
  }, [draw])

  // LA LECTURE (état lu/pause, boucle rAF, curseur de la frise, ARRÊT SUR L'ÉTAT FINAL) vit
  // dans useReplayPlayback : le canvas garde le DESSIN, le hook porte le TEMPS.
  const { playing, endFrame, sliderRef, togglePlay, restart, onScrub } = useReplayPlayback({
    doc, baseFps, speed: multiplier, renderWidth, frameRef, draw, soundTick: sound.tick,
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
          <div className="p-3">
            {/* La légende se pose DANS le cadre du canvas (coin bas-gauche) : une échelle
                de couleur lue à côté de sa carte n'est plus une échelle. Le conteneur
                relatif n'existe que pour elle — sans carte de chaleur, rien n'y flotte. */}
            {/* TROIS CALQUES SURVOLABLES SUR UNE SEULE BALISE (poses, emplacements d'arme,
                drapeaux) : chacun rejoue le survol sur SA donnée, le canvas passe le geste. */}
            <div className="relative mx-auto" style={{ width: renderWidth || '100%' }}>
              <canvas
                ref={canvasRef}
                className="block"
                style={{ width: renderWidth || '100%', height: CANVAS_HEIGHT }}
                onPointerMove={(e) => {
                  placements.hover.onPointerMove(e)
                  weaponPads.onPointerMove(e)
                  flags.onPointerMove(e)
                }}
                onPointerLeave={() => {
                  placements.hover.onPointerLeave()
                  weaponPads.onPointerLeave()
                  flags.onPointerLeave()
                }}
              />
              {/* Les infobulles des trois calques survolables (cf. ReplayCanvasTips). */}
              <ReplayCanvasTips
                locale={locale} width={renderWidth} ownerNameOf={nameOfSlot}
                placement={placements.hover.hover} pad={weaponPads.hover} flag={flags.hover}
              />
              {heat.grid && <ReplayHeatmapLegend locale={locale} mode={heat.grid.mode} />}
            </div>
            {/* LA BARRE DE LECTURE : icônes, vitesse et son au niveau de la lecture
                (demandes du 2026-08-24) — extraite dans ReplayTransport. */}
            <ReplayTransport
              playing={playing}
              onTogglePlay={togglePlay}
              onRestart={restart}
              clockRef={clockRef}
              sliderRef={sliderRef}
              maxFrame={endFrame}
              onScrub={onScrub}
              speed={multiplier}
              onSetSpeed={setMultiplier}
              sound={sound}
              locale={locale}
              leadMarks={leadMarks}
              settingsOpen={settingsOpen}
              onToggleSettings={() => setSettingsOpen((v) => !v)}
              settingsButtonRef={settingsButtonRef}
            />
          </div>
        </div>
        {/* Le panneau se pose SUR la carte, à droite (retour de planche du 16/08 : « je vois
            plus un panneau par dessus »). Il ne mange donc plus la largeur du canvas — et il
            laisse libre le coin BAS-GAUCHE, où vit la légende de la carte de chaleur. */}
        {settingsOpen && (
          <ReplaySettingsDrawer
            locale={locale}
            onClose={closeSettings}
            showAim={showAim}
            onToggleAim={toggleAim}
            showZones={showZones}
            onToggleZones={toggleZones}
            showNames={showNames}
            onToggleNames={toggleNames}
            showTrail={showTrail}
            onToggleTrail={toggleTrail}
            zonesAvailable={calloutZones.length > 0}
            placements={{
              available: placements.counts.drawable > 0,
              show: showPlacements,
              onToggle: togglePlacements,
              unnamedAvailable: placements.counts.unnamed > 0,
              showUnnamed: showUnnamedPlacements,
              onToggleUnnamed: toggleUnnamedPlacements,
              droppedAvailable: placements.counts.dropped > 0,
              showDropped: showDroppedPlacements,
              onToggleDropped: toggleDroppedPlacements,
            }}
            weaponPads={{
              available: weaponPads.available,
              show: showWeaponPads,
              onToggle: toggleWeaponPads,
            }}
            flagCarries={{ available: flags.available, show: showFlagCarries, onToggle: toggleFlagCarries }}
            heatmap={{
              show: showHeatmap,
              onToggle: toggleHeatmap,
              mode: heat.mode,
              onSetMode: setHeatmapMode,
              span: heatmapSpan,
              onSetSpan: setHeatmapSpan,
              killsAvailable: heat.killsAvailable,
            }}
            showShotFx={showShotFx}
            onToggleShotFx={toggleShotFx}
            showKillFx={showKillFx}
            onToggleKillFx={toggleKillFx}
            sound={sound}
            markerColors={markerColors}
            onSetMarkerColors={setMarkerColors}
            triggerRef={settingsButtonRef}
          />
        )}
      </div>
    </div>
  )
}
