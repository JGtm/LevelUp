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
 */
import { type ChangeEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { getSeriesColors } from '@/lib/accessibility/plotlyColorscale'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { ReplayMapBackgroundCalibration } from '@/lib/api/types'

import type { KillEvent } from '@/features/match-view/_momentum'

import { resolveTeamColorFromID } from '@/lib/halo/teamNames'

import { drawCalloutsLayer, type CalloutZoneReady } from './calloutsLayer'
import { readInk } from './canvasInk'
import { readFxInk } from './fxInk'
import { buildShotFx } from './shotFx'
import {
  buildObjectivePulses,
  drawObjectivePulses,
  drawObjectivesLayer,
  normalizeMapObjectives,
} from './objectivesLayer'
import {
  buildGrenadeRestFx,
  DYNAMO_REST_HOLD_MS,
  GRENADE_REST_HOLD_MS,
} from './grenadeFx'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { buildKillFx } from './killFx'
import { ReplaySettingsDrawer } from './ReplaySettingsDrawer'
import { useReplaySettings } from './useReplaySettings'
import { useReplaySound } from './useReplaySound'
import { backgroundRect, coversPlayedArea } from './mapBackground'
import { buildFloorGrid } from './mapFloor'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  drawFloorLayer,
  drawGeometryLayer,
  drawGrenadeRestLayer,
  drawGrenadesLayer,
  drawKillFxLayer,
  drawShotsLayer,
  tintedIconCanvas,
} from './replayDraw'
import {
  fitWidth,
  formatClock,
  frameToMs,
  framesPerSecond,
  isAliveAt,
  msToFrames,
  sceneBounds,
} from './replayLogic'
import { drawProjectilesLayer, drawTracksLayer, type MarkerTiming } from './replayMarkers'

// 8 tokens de série = un par entité (cyclés au-delà de 8 via getSeriesColors).
const TRACK_TOKENS: SemanticToken[] = [
  'chart-series-1', 'chart-series-2', 'chart-series-3', 'chart-series-4',
  'chart-series-5', 'chart-series-6', 'chart-series-7', 'chart-series-8',
]
// Fond de carte : token neutre, sans connotation directionnelle (le sujet = les joueurs).
const GEOMETRY_TOKEN: SemanticToken = 'divergent-neutral'
// Événements ponctuels. Le LANCER emprunte un token d'information ; le TIR, lui, ne prend
// plus aucun token de données : sa couleur dit la NATURE DE LA DÉCHARGE et vient des teintes
// diégétiques du thème (fxInk.ts, décision utilisateur du 2026-08-15). Le token d'alerte
// reste employé par les effets de MORT, qui n'ont pas changé.
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'
// Rémanences des événements ponctuels, en temps réel — celles du POC, et elles DIFFÈRENT :
// un TIR est un éclat bref (0,6 s — c'est sa brièveté qui le rend lisible : à 1,4 s le trait
// traînait dim et se fondait dans la carte, mesure du recalage 2.2), un LANCER et une MORT
// tiennent 1,4 s parce qu'ils portent plus de sens qu'une détonation.
const SHOT_HOLD_MS = 600
const EVENT_HOLD_MS = 1_400

const CANVAS_HEIGHT = 480
const CANVAS_PAD = 24

/** Référence STABLE pour « pas de zones » : un `?? []` inline recuirait le calque à chaque rendu. */
const EMPTY_ZONES: CalloutZoneReady[] = []

/**
 * Réglages temporels du calque des joueurs, en TEMPS RÉEL — jamais en nombre de frames : la
 * cadence d'échantillonnage est choisie au build et peut changer sans que la lecture change.
 * Valeurs reprises du POC, où elles ont été réglées à l'écran ; leur justification mesurée est
 * en tête de replayMarkers.ts.
 */
const TIMING_MS = {
  trail: 7_000,
  aimHold: 5_000,
  death: 1_500,
  spawn: 800,
} as const

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
}

export function ReplayCanvas({ doc, locale, kills, t0Ms, onFrameChange, background, callouts }: ReplayCanvasProps) {
  const t = REPLAY_TEXT[locale]
  const canvasRef = useRef<HTMLCanvasElement>(null)
  // Fond de carte peint UNE fois puis recopié : il ne dépend ni de la frame ni de la lecture.
  const floorRef = useRef<HTMLCanvasElement | null>(null)
  // Zones nommées : même règle que le sol — calque statique cuit hors écran, recopié.
  const zonesRef = useRef<HTMLCanvasElement | null>(null)
  // Objectifs statiques du mode (lot 4) : même règle — le calque ne bouge jamais.
  const objectivesRef = useRef<HTMLCanvasElement | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const sliderRef = useRef<HTMLInputElement>(null)
  const clockRef = useRef<HTMLSpanElement>(null)
  const aliveRef = useRef<HTMLSpanElement>(null)
  const frameRef = useRef(0)
  const publishedAtRef = useRef(0)
  // Vignettes de TYPE de grenade, teintées à l'encre du thème (masques HUD blanc/gris +
  // alpha). Rempli hors rendu, par rang — un rang sans visuel garde l'anneau seul.
  const grenadeIconsRef = useRef<Map<number, HTMLCanvasElement>>(new Map())

  const [playing, setPlaying] = useState(true)
  const [width, setWidth] = useState(0)
  // TIROIR DE RÉGLAGES (décision utilisateur du 16/08) : fermé par défaut, ouvert par un
  // bouton unique dans la barre. Calques et vitesse persistés comme le son — même
  // mécanisme (replayPreferences.ts), trois réglages distincts.
  const [settingsOpen, setSettingsOpen] = useState(false)
  const {
    showAim, toggleAim, showZones, toggleZones, speed: multiplier, setSpeed: setMultiplier,
  } = useReplaySettings()
  // SON : coupé par défaut, préférence, volume et filtre par catégorie persistés, tout le
  // câblage dans le hook (règles dans replaySound.ts, lecture Web Audio dans replayAudio.ts).
  const sound = useReplaySound(doc, kills, t0Ms, multiplier)

  const paletteVersion = useColorPaletteVersion()
  const colors = useMemo(() => {
    void paletteVersion // re-résoudre au changement de thème (getSeriesColors lit le DOM)
    return getSeriesColors(doc.tracks.length, TRACK_TOKENS)
  }, [doc.tracks.length, paletteVersion])
  const geometryColor = useMemo(() => {
    void paletteVersion
    return resolveToken(GEOMETRY_TOKEN)
  }, [paletteVersion])
  const shotColor = useMemo(() => {
    void paletteVersion
    return resolveToken(SHOT_TOKEN)
  }, [paletteVersion])
  const grenadeColor = useMemo(() => {
    void paletteVersion
    return resolveToken(GRENADE_TOKEN)
  }, [paletteVersion])
  // Encres de mise en page du sol : elles suivent le thème, pas la palette d'accessibilité.
  const floorStyle = useMemo(() => {
    void paletteVersion
    return { fill: resolveToken(GEOMETRY_TOKEN), edge: readInk('--muted-foreground') }
  }, [paletteVersion])
  // Teintes des ÉCLAIRS DE BOUCHE : lues UNE fois par thème, jamais par image — un
  // getComputedStyle par effet et par frame coûterait plus que le dessin lui-même.
  const fxInk = useMemo(() => {
    void paletteVersion
    return readFxInk()
  }, [paletteVersion])

  // Couleur PAR SLOT : un tir se dessine dans la teinte de son tireur, et c'est elle qui permet
  // de suivre un joueur des yeux. Le slot d'une trace est unique dans le document.
  const colorBySlot = useMemo(() => {
    const m = new Map<number, string>()
    doc.tracks.forEach((tr, i) => {
      const c = colors[i] ?? colors[0]
      if (c) m.set(tr.slot, c)
    })
    return m
  }, [doc.tracks, colors])

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
    return nBig > 0 ? getSeriesColors(nBig, TRACK_TOKENS) : []
  }, [calloutZones, paletteVersion])

  // Les OBJECTIFS STATIQUES du mode arrivent AVEC le document (`mapObjectives`, servi à
  // la requête) : normalisés une fois, comme les callouts. Absents = pas de calque.
  const mapObjectives = useMemo(() => normalizeMapObjectives(doc.mapObjectives), [doc.mapObjectives])
  // Couleur d'un index d'équipe : le référentiel d'identité du jeu (lib/halo — donnée de
  // domaine, pas un choix d'UI), encre neutre du thème pour -1 ou un index hors
  // référentiel. `team` est DÉJÀ arbitré côté serveur (Bastion/Extraction = neutres).
  const objectiveColorOfTeam = useCallback(
    (team: number) => (team >= 0 ? resolveTeamColorFromID(team) : null) ?? floorStyle.edge,
    [floorStyle.edge],
  )
  // Les PULSES d'action d'objectif (doc.objectives, rendu nulle part avant le lot 4.4) :
  // précalculés en monde, comme les effets de mort.
  const objectivePulses = useMemo(
    () => buildObjectivePulses(doc, mapObjectives),
    [doc, mapObjectives],
  )

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
  const baseFps = useMemo(() => framesPerSecond(doc), [doc])
  const timing = useMemo<MarkerTiming>(
    () => ({
      trail: msToFrames(TIMING_MS.trail, doc),
      aimHold: msToFrames(TIMING_MS.aimHold, doc),
      death: msToFrames(TIMING_MS.death, doc),
      spawn: msToFrames(TIMING_MS.spawn, doc),
    }),
    [doc],
  )
  const eventHoldFrames = useMemo(() => msToFrames(EVENT_HOLD_MS, doc), [doc])
  const shotHoldFrames = useMemo(() => msToFrames(SHOT_HOLD_MS, doc), [doc])
  // Les tirs sont PRÉCALCULÉS comme les morts : famille, teinte et REGARD du tireur résolus
  // une fois au chargement (mesure : la couverture d'orientation passe de 18,6 % à 100 % sur
  // le film témoin en relisant le regard plutôt que le champ de l'événement).
  const shotFx = useMemo(() => buildShotFx(doc, timing.aimHold), [doc, timing.aimHold])
  // Les effets de mort sont PRÉCALCULÉS en monde (positions relues une fois, patron POC) :
  // pendant la lecture, seul le passage monde -> pixels reste à faire.
  const killFx = useMemo(
    () => buildKillFx(doc, kills ?? [], t0Ms ?? 0),
    [doc, kills, t0Ms],
  )
  // Fins de vol de grenade : le lien lancer -> projectile est dans l'artefact (v3).
  const grenadeRestFx = useMemo(() => buildGrenadeRestFx(doc), [doc])
  const restWindow = useMemo(
    () => ({
      holdHalo: msToFrames(GRENADE_REST_HOLD_MS, doc),
      holdDynamo: msToFrames(DYNAMO_REST_HOLD_MS, doc),
    }),
    [doc],
  )
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

    const view = { bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD }
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
    drawTracksLayer(ctx, doc.tracks, view, {
      colors,
      ink: floorStyle.edge,
      frame,
      timing,
      z: zRange,
      k: dpr,
      showAim,
    })

    // Les événements passent APRÈS les trajectoires : ils se lisent sur elles.
    const win = { frame, hold: eventHoldFrames }
    if (shotFx.length > 0) {
      drawShotsLayer(ctx, shotFx, view, { frame, hold: shotHoldFrames }, {
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
    // Le PULSE D'ACTION D'OBJECTIF (capture, retour, prise de zone) : un anneau qui
    // s'ouvre depuis la zone/le marqueur concerné à l'instant de l'action (lot 4.4).
    if (objectivePulses.length > 0) {
      drawObjectivePulses(ctx, objectivePulses, view, win,
        { colorOfTeam: objectiveColorOfTeam }, reducedMotion)
    }
    // Les MORTS par-dessus les tirs : c'est l'événement le plus lourd de sens du calque,
    // et le seul dont l'extrémité pointe une vraie victime (couple complet, règle 89/93).
    if (killFx.length > 0) {
      drawKillFxLayer(ctx, killFx, view, win, {
        colorOfSlot: (slot) => colorBySlot.get(slot) ?? null,
        fallback: shotColor,
        reducedMotion,
      })
    }

    if (clockRef.current) {
      clockRef.current.textContent = `${formatClock(frameToMs(frame, doc))} / ${totalLabel}`
    }
    if (aliveRef.current) {
      const alive = doc.tracks.reduce((n, tr) => n + (isAliveAt(tr, frame) ? 1 : 0), 0)
      aliveRef.current.textContent = `${alive} ${t.aliveSuffix}`
    }
    if (onFrameChange) {
      const now = performance.now()
      if (now - publishedAtRef.current >= FRAME_PUBLISH_MS) {
        publishedAtRef.current = now
        onFrameChange(Math.floor(frame))
      }
    }
  }, [
    doc, colors, geometryColor, bounds, zRange, timing, totalLabel,
    t.aliveSuffix, renderWidth,
    shotColor,
    grenadeColor,
    eventHoldFrames,
    shotHoldFrames,
    shotFx,
    fxInk,
    killFx,
    grenadeRestFx,
    restWindow,
    objectivePulses,
    objectiveColorOfTeam,
    floorStyle.edge,
    colorBySlot,
    reducedMotion,
    showAim,
    showZones,
    onFrameChange,
    mapImage,
  ])

  // Les vignettes de type de grenade se chargent et se TEIGNENT une fois par document et
  // par thème (l'encre suit floorStyle.edge) : le canvas ne sait pas teindre un masque au
  // moment du dessin, et re-teindre 60 fois par seconde coûterait pour rien.
  useEffect(() => {
    const map = new Map<number, HTMLCanvasElement>()
    grenadeIconsRef.current = map
    doc.grenadeLabels.forEach((lbl, rank) => {
      if (!lbl.img) return
      const im = new Image()
      im.onload = () => {
        map.set(rank, tintedIconCanvas(im, floorStyle.edge))
        draw()
      }
      im.src = lbl.img
    })
  }, [doc.grenadeLabels, floorStyle.edge, draw])

  // Le sol est repeint quand SA géométrie, SON cadrage ou SES encres changent — jamais à
  // l'image. C'est la condition pour que 45 000 cellules ne coûtent rien à l'animation.
  useEffect(() => {
    if (!floorGrid || renderWidth === 0) {
      floorRef.current = null
      return
    }
    const dpr = window.devicePixelRatio || 1
    const off = document.createElement('canvas')
    off.width = Math.round(renderWidth * dpr)
    off.height = Math.round(CANVAS_HEIGHT * dpr)
    const octx = off.getContext('2d')
    if (!octx) return
    octx.setTransform(dpr, 0, 0, dpr, 0, 0)
    drawFloorLayer(
      octx,
      floorGrid,
      { bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD },
      floorStyle,
    )
    floorRef.current = off
    draw()
  }, [floorGrid, renderWidth, bounds, floorStyle, draw])

  // Les zones nommées se cuisent quand LEURS données, LEUR cadrage ou LEURS encres
  // changent — jamais à l'image (28 polygones + libellés cernés repeints 60 fois par
  // seconde coûteraient pour rien, même règle que le sol).
  useEffect(() => {
    if (calloutZones.length === 0 || renderWidth === 0) {
      zonesRef.current = null
      return
    }
    const dpr = window.devicePixelRatio || 1
    const off = document.createElement('canvas')
    off.width = Math.round(renderWidth * dpr)
    off.height = Math.round(CANVAS_HEIGHT * dpr)
    const octx = off.getContext('2d')
    if (!octx) return
    octx.setTransform(dpr, 0, 0, dpr, 0, 0)
    drawCalloutsLayer(
      octx,
      calloutZones,
      { bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD },
      { bigColors: zoneColors, fineInk: floorStyle.edge, locale },
    )
    zonesRef.current = off
    draw()
  }, [calloutZones, zoneColors, renderWidth, bounds, floorStyle.edge, locale, draw])

  // Les objectifs statiques se cuisent quand LEURS données, LEUR cadrage ou LEURS encres
  // changent — jamais à l'image (même règle que le sol et les zones nommées).
  useEffect(() => {
    if (mapObjectives.length === 0 || renderWidth === 0) {
      objectivesRef.current = null
      return
    }
    const dpr = window.devicePixelRatio || 1
    const off = document.createElement('canvas')
    off.width = Math.round(renderWidth * dpr)
    off.height = Math.round(CANVAS_HEIGHT * dpr)
    const octx = off.getContext('2d')
    if (!octx) return
    octx.setTransform(dpr, 0, 0, dpr, 0, 0)
    drawObjectivesLayer(
      octx,
      mapObjectives,
      { bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD },
      { colorOfTeam: objectiveColorOfTeam },
    )
    objectivesRef.current = off
    draw()
  }, [mapObjectives, objectiveColorOfTeam, renderWidth, bounds, draw])

  // Redraw hors animation (thème, resize, données, pause).
  useEffect(() => {
    draw()
  }, [draw])

  // Boucle de lecture (requestAnimationFrame) uniquement quand `playing`.
  //
  // LE SON BAT AVEC ELLE, et nulle part ailleurs : hors lecture (pause, onglet en
  // arrière-plan, redessin au changement de thème) il n'y a pas de battement, donc pas un
  // son. C'est ce qui rend le silence d'un lecteur à l'arrêt structurel, pas conditionnel.
  const soundTick = sound.tick
  useEffect(() => {
    if (!playing || renderWidth === 0) return
    const fps = baseFps * multiplier
    let raf = 0
    let last = 0
    const step = (ts: number) => {
      if (last === 0) last = ts
      const dtSec = (ts - last) / 1000
      last = ts
      let next = frameRef.current + dtSec * fps
      if (next >= doc.frameCount - 1) next = 0
      frameRef.current = next
      if (sliderRef.current) sliderRef.current.value = String(Math.round(next))
      soundTick(frameToMs(next, doc))
      draw()
      raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [playing, baseFps, multiplier, doc, renderWidth, draw, soundTick])

  const onScrub = (e: ChangeEvent<HTMLInputElement>) => {
    frameRef.current = Number(e.currentTarget.value)
    if (!playing) draw()
  }

  const restart = () => {
    frameRef.current = 0
    if (sliderRef.current) sliderRef.current.value = '0'
    setPlaying(true)
  }

  return (
    // OVERFLOW-HIDDEN sur la carte : le tiroir (frère du canvas, ci-dessous) partage le
    // même fond que ce cadre — sans lui, ses coins carrés dépasseraient du rayon arrondi.
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex">
        {/* Colonne canvas : c'est ELLE que le ResizeObserver mesure, pas le cadre entier —
            sans quoi ouvrir le tiroir ne rétrécirait jamais le rendu (il resterait sous le
            panneau plutôt que de lui céder la place). */}
        <div ref={containerRef} className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
            <div className="flex items-baseline gap-2 text-sm">
              <span className="font-medium">
                {doc.tracks.length} {t.livesSuffix}
              </span>
              <span ref={aliveRef} className="text-xs text-muted-foreground" />
            </div>
            {/* TIROIR DE RÉGLAGES (décision utilisateur du 16/08) : un bouton unique, plutôt
                que la rangée de calques/son/vitesse qui vivait ici — ils sont tous dans le
                panneau maintenant, avec les nouveaux filtres de son par catégorie. */}
            <Button
              variant={settingsOpen ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setSettingsOpen((v) => !v)}
              className="h-7 px-2 text-xs"
              aria-expanded={settingsOpen}
            >
              {t.settingsButton}
            </Button>
          </div>
          <div className="p-3">
            <canvas
              ref={canvasRef}
              className="mx-auto block"
              style={{ width: renderWidth || '100%', height: CANVAS_HEIGHT }}
            />
            <div className="mt-2 flex items-center gap-3">
              <Button
                variant="default"
                size="sm"
                onClick={() => setPlaying((p) => !p)}
                className="h-8 w-24"
              >
                {playing ? t.pause : t.play}
              </Button>
              <Button variant="ghost" size="sm" onClick={restart} className="h-8">
                {t.restart}
              </Button>
              <span
                ref={clockRef}
                className="min-w-[5.5rem] font-mono text-xs tabular-nums text-muted-foreground"
                aria-label={t.time}
              />
              <input
                ref={sliderRef}
                type="range"
                min={0}
                max={Math.max(doc.frameCount - 1, 0)}
                defaultValue={0}
                onChange={onScrub}
                className="flex-1"
                aria-label={t.time}
              />
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {t.note}
              {doc.geometry?.length ? ` ${doc.geometry.length} ${t.propsSuffix}.` : ''}
              {/* CE QUI EST SOUS LES JOUEURS EST DIT. Une carte du jeu et un sol reconstruit ne
                  se lisent pas de la même façon, et rien à l'écran ne les distingue. */}
              {` ${mapImage ? t.mapBackgroundNote : t.mapBackgroundFallback}`}
              {/* L'ÉCRAN DIT « dernière position connue », JAMAIS « impact » : aucun événement
                  de détonation n'existe dans le film (item 2.3). */}
              {grenadeRestFx.length > 0 ? ` ${t.grenadeRestNote}` : ''}
            </p>
          </div>
        </div>
        {/* Le panneau POUSSE le canvas (rangée flex), il ne le recouvre jamais : exigence
            explicite du 16/08, « n'obstrue pas la carte pendant la lecture ». */}
        {settingsOpen && (
          <ReplaySettingsDrawer
            locale={locale}
            onClose={() => setSettingsOpen(false)}
            showAim={showAim}
            onToggleAim={toggleAim}
            showZones={showZones}
            onToggleZones={toggleZones}
            zonesAvailable={calloutZones.length > 0}
            sound={sound}
            speed={multiplier}
            onSetSpeed={setMultiplier}
          />
        )}
      </div>
    </div>
  )
}
