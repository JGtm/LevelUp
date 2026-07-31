/**
 * ReplayCanvas — rejeu 2D animé (vue du dessus) des trajectoires joueurs décodées du film.
 * Rendu canvas (animation fluide) ; toute la logique est pure dans replayLogic.ts et le
 * dessin dans replayDraw.ts (anti-pattern « logique dans le composant »).
 *
 * Trois traits du document sont exploités ici :
 * - frameIntervalMs : la vitesse « 1× » suit le TEMPS RÉEL du match (avant : 60 frames/s
 *   sur un axe qui n'était qu'un index de record → rejeu très court et figé) ;
 * - geometry : fond de carte (props Forge) dessiné SOUS les trajectoires ;
 * - bounds.minZ/maxZ + points[].z : indication d'étage (opacité du décor, filtre de tranche).
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

import { readInk } from './canvasInk'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { buildFloorGrid } from './mapFloor'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  drawFloorLayer,
  drawGeometryLayer,
  drawGrenadesLayer,
  drawShotsLayer,
} from './replayDraw'
import {
  fitWidth,
  FLOOR_BANDS,
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
// Événements ponctuels. Le tir emprunte le token d'alerte (il a INFLIGÉ un dégât — le film
// n'enregistre que ceux-là), le lancer un token d'information : deux natures, deux lectures.
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'
// Rémanence des événements ponctuels, en temps réel. 1,4 s est la convention déjà retenue par
// le POC pour un lancer ; elle vaut aussi pour un tir, dont l'instant est ponctuel.
const EVENT_HOLD_MS = 1_400

const CANVAS_HEIGHT = 480
const CANVAS_PAD = 24
const SPEED_MULTIPLIERS = [0.5, 1, 2, 4]

/**
 * Réglages temporels du calque des joueurs, en TEMPS RÉEL — jamais en nombre de frames : la
 * cadence d'échantillonnage est choisie au build et peut changer sans que la lecture change.
 * Valeurs reprises du POC, où elles ont été réglées à l'écran ; leur justification mesurée est
 * en tête de replayMarkers.ts.
 */
const TIMING_MS = {
  trail: 7_000,
  aimHold: 5_000,
  shieldHold: 2_000,
  death: 1_500,
  spawn: 800,
} as const
// En dessous de ce dénivelé (mètres), la carte est considérée plate : pas de filtre d'étage.
const MIN_FLOOR_SPAN = 1

/**
 * Cadence de publication de l'image courante vers React, en millisecondes.
 *
 * POURQUOI PAS À CHAQUE IMAGE. Le canvas se redessine à la cadence de l'écran ; les fiches
 * joueur, elles, sont du DOM. Les re-rendre 60 fois par seconde coûterait tout le budget
 * d'animation pour un contenu qui change à peine. 150 ms reste bien en deçà de ce que l'œil
 * perçoit comme un retard sur un compteur, et divise le travail de React par dix.
 */
const FRAME_PUBLISH_MS = 150

interface ReplayCanvasProps {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  /** Appelé à cadence réduite avec l'image courante : sert aux panneaux hors canvas. */
  onFrameChange?: (frame: number) => void
}

export function ReplayCanvas({ doc, locale, onFrameChange }: ReplayCanvasProps) {
  const t = REPLAY_TEXT[locale]
  const canvasRef = useRef<HTMLCanvasElement>(null)
  // Fond de carte peint UNE fois puis recopié : il ne dépend ni de la frame ni de la lecture.
  const floorRef = useRef<HTMLCanvasElement | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const sliderRef = useRef<HTMLInputElement>(null)
  const clockRef = useRef<HTMLSpanElement>(null)
  const aliveRef = useRef<HTMLSpanElement>(null)
  const frameRef = useRef(0)
  const publishedAtRef = useRef(0)

  const [playing, setPlaying] = useState(true)
  const [multiplier, setMultiplier] = useState(1)
  const [floor, setFloor] = useState<number | null>(null)
  const [width, setWidth] = useState(0)
  const [showAim, setShowAim] = useState(true)
  const [showShield, setShowShield] = useState(true)

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

  // La trame d'altitudes ne dépend QUE du document : construite une fois, pas à chaque resize.
  const floorGrid = useMemo(
    () => (doc.structure?.length ? buildFloorGrid(doc.structure, doc.bounds) : null),
    [doc.structure, doc.bounds],
  )

  const bounds = useMemo(() => sceneBounds(doc), [doc])
  // Largeur de dessin = ratio de la scène à hauteur fixée (évite les marges latérales).
  const renderWidth = useMemo(
    () => (width === 0 ? 0 : Math.floor(fitWidth(bounds, width, CANVAS_HEIGHT, CANVAS_PAD))),
    [bounds, width],
  )
  const zRange = useMemo(
    () => ({ min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 }),
    [doc.bounds.minZ, doc.bounds.maxZ],
  )
  const hasFloors = zRange.max - zRange.min > MIN_FLOOR_SPAN
  const baseFps = useMemo(() => framesPerSecond(doc), [doc])
  const timing = useMemo<MarkerTiming>(
    () => ({
      trail: msToFrames(TIMING_MS.trail, doc),
      aimHold: msToFrames(TIMING_MS.aimHold, doc),
      shieldHold: msToFrames(TIMING_MS.shieldHold, doc),
      death: msToFrames(TIMING_MS.death, doc),
      spawn: msToFrames(TIMING_MS.spawn, doc),
    }),
    [doc],
  )
  const eventHoldFrames = useMemo(() => msToFrames(EVENT_HOLD_MS, doc), [doc])
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
    if (floorRef.current) {
      ctx.drawImage(floorRef.current, 0, 0, renderWidth, CANVAS_HEIGHT)
    } else if (doc.geometry?.length) {
      // REPLI, pas un doublon : sans fichier de structure figé, la carte n'a pas de sol
      // reconstruit et les props Forge redeviennent le seul repère disponible. Ils couvrent
      // 3,4 % du terrain — c'est peu, et c'est mieux qu'un fond vide.
      drawGeometryLayer(ctx, doc.geometry, view, { color: geometryColor, z: zRange })
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
      floor: hasFloors ? floor : null,
      z: zRange,
      k: dpr,
      showAim,
      showShield,
    })

    // Les événements passent APRÈS les trajectoires : ils se lisent sur elles.
    const win = { frame, hold: eventHoldFrames }
    if (doc.shots?.length) {
      drawShotsLayer(ctx, doc.shots, view, win, {
        colorOfSlot: (slot) => colorBySlot.get(slot) ?? null,
        fallback: shotColor,
        labelOf: (id) => (id ? doc.weaponLabels?.[id] : undefined),
        reducedMotion,
      })
    }
    if (doc.grenades?.length) {
      drawGrenadesLayer(ctx, doc.grenades, view, win, grenadeColor)
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
    doc, colors, geometryColor, bounds, zRange, hasFloors, floor, timing, totalLabel,
    t.aliveSuffix, renderWidth,
    shotColor,
    grenadeColor,
    eventHoldFrames,
    floorStyle.edge,
    colorBySlot,
    reducedMotion,
    showAim,
    showShield,
    onFrameChange,
  ])

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

  // Redraw hors animation (thème, resize, données, pause, filtre d'étage).
  useEffect(() => {
    draw()
  }, [draw])

  // Boucle de lecture (requestAnimationFrame) uniquement quand `playing`.
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
      draw()
      raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [playing, baseFps, multiplier, doc.frameCount, renderWidth, draw])

  const onScrub = (e: ChangeEvent<HTMLInputElement>) => {
    frameRef.current = Number(e.currentTarget.value)
    if (!playing) draw()
  }

  const restart = () => {
    frameRef.current = 0
    if (sliderRef.current) sliderRef.current.value = '0'
    setPlaying(true)
  }

  const floorLabels = [t.floorLow, t.floorMid, t.floorHigh].slice(0, FLOOR_BANDS)

  return (
    <div ref={containerRef} className="rounded-lg border border-border bg-card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
        <div className="flex items-baseline gap-2 text-sm">
          <span className="font-medium">
            {doc.tracks.length} {t.livesSuffix}
          </span>
          <span ref={aliveRef} className="text-xs text-muted-foreground" />
        </div>
        <div className="flex flex-wrap items-center gap-1">
          <span className="mr-1 text-xs text-muted-foreground">{t.layers}</span>
          <Button
            variant={showAim ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setShowAim((v) => !v)}
            className="h-7 px-2 text-xs"
            title={t.layerAimHint}
            aria-pressed={showAim}
          >
            {t.layerAim}
          </Button>
          <Button
            variant={showShield ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setShowShield((v) => !v)}
            className="h-7 px-2 text-xs"
            title={t.layerShieldHint}
            aria-pressed={showShield}
          >
            {t.layerShield}
          </Button>
          <span aria-hidden className="mx-2 h-4 w-px bg-border" />
          {hasFloors && (
            <>
              <span className="mr-1 text-xs text-muted-foreground">{t.floor}</span>
              <Button
                variant={floor === null ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setFloor(null)}
                className="h-7 px-2 text-xs"
              >
                {t.floorAll}
              </Button>
              {floorLabels.map((label, i) => (
                <Button
                  key={label}
                  variant={floor === i ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setFloor(i)}
                  className="h-7 px-2 text-xs"
                >
                  {label}
                </Button>
              ))}
              <span aria-hidden className="mx-2 h-4 w-px bg-border" />
            </>
          )}
          <span className="mr-1 text-xs text-muted-foreground">{t.speed}</span>
          {SPEED_MULTIPLIERS.map((m) => (
            <Button
              key={m}
              variant={multiplier === m ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setMultiplier(m)}
              className="h-7 px-2 text-xs"
            >
              {m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`}
            </Button>
          ))}
        </div>
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
        </p>
      </div>
    </div>
  )
}
