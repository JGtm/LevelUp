/**
 * replayLogic.ts — logique PURE du rejeu 2D (interpolation, projection, lecture).
 * Sans React ni canvas → testable unitairement (anti-pattern « logique dans le composant »).
 */
import type { ReplayBounds, ReplayMapObject, ReplayPoint } from '@/lib/api/types'

import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

export interface XY {
  x: number
  y: number
}

/**
 * Cadence de lecture « 1× » quand l'artefact ne porte pas d'échelle temporelle
 * (frameIntervalMs absent, anciens artefacts) : l'axe T est alors un simple index.
 */
const FALLBACK_FPS = 60

/** Nombre de bandes d'altitude proposées comme « étages ». */
export const FLOOR_BANDS = 3

/**
 * positionAt renvoie la position d'une track au temps `t` (index de frame) :
 * - null avant le 1er point échantillonné ;
 * - le dernier point maintenu après le dernier échantillon ;
 * - interpolation linéaire entre deux échantillons connus (recherche binaire).
 */
export function positionAt(points: ReplayPoint[], t: number): XY | null {
  if (points.length === 0) return null
  const first = points[0]
  if (t <= first.t) return t < first.t ? null : { x: first.x, y: first.y }
  const last = points[points.length - 1]
  if (t >= last.t) return { x: last.x, y: last.y }

  let lo = 0
  let hi = points.length - 1
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1
    if (points[mid].t <= t) lo = mid
    else hi = mid
  }
  const a = points[lo]
  const b = points[hi]
  const span = b.t - a.t
  const f = span === 0 ? 0 : (t - a.t) / span
  return { x: a.x + (b.x - a.x) * f, y: a.y + (b.y - a.y) * f }
}

/**
 * trailAt renvoie les points de la traînée d'une track dans la fenêtre [t-window, t]
 * (échantillons bruts) plus la tête interpolée à `t`.
 */
export function trailAt(points: ReplayPoint[], t: number, windowFrames: number): XY[] {
  const start = t - windowFrames
  const out: XY[] = []
  for (const p of points) {
    if (p.t < start) continue
    if (p.t > t) break
    out.push({ x: p.x, y: p.y })
  }
  const head = positionAt(points, t)
  if (head) {
    const tail = out[out.length - 1]
    if (!tail || tail.x !== head.x || tail.y !== head.y) out.push(head)
  }
  return out
}

/** altitudeAt renvoie le Z interpolé d'une track au temps `t` (0 si l'artefact n'a pas de Z). */
export function altitudeAt(points: ReplayPoint[], t: number): number | null {
  if (points.length === 0) return null
  const first = points[0]
  if (t <= first.t) return t < first.t ? null : (first.z ?? 0)
  const last = points[points.length - 1]
  if (t >= last.t) return last.z ?? 0

  let lo = 0
  let hi = points.length - 1
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1
    if (points[mid].t <= t) lo = mid
    else hi = mid
  }
  const a = points[lo]
  const b = points[hi]
  const span = b.t - a.t
  const f = span === 0 ? 0 : (t - a.t) / span
  return (a.z ?? 0) + ((b.z ?? 0) - (a.z ?? 0)) * f
}

/**
 * advanceFrame calcule le prochain temps de lecture ; boucle à 0 en fin de rejeu.
 * `deltaFrames` peut être fractionnaire (dt réel × vitesse).
 */
export function advanceFrame(t: number, deltaFrames: number, frameCount: number): number {
  if (frameCount <= 1) return 0
  const next = t + deltaFrames
  if (next >= frameCount - 1) return 0
  return next < 0 ? 0 : next
}

/**
 * worldToCanvas projette une position monde (x,y) dans un rectangle canvas (fit en
 * préservant le ratio, centré, avec marge `pad`). Y est INVERSÉ : le monde a +Y vers le
 * haut, le canvas +Y vers le bas.
 */
export function worldToCanvas(
  p: XY,
  bounds: ReplayBounds,
  width: number,
  height: number,
  pad: number,
): XY {
  const bw = Math.max(bounds.maxX - bounds.minX, 1e-6)
  const bh = Math.max(bounds.maxY - bounds.minY, 1e-6)
  const scale = Math.min((width - 2 * pad) / bw, (height - 2 * pad) / bh)
  const drawW = bw * scale
  const drawH = bh * scale
  const offsetX = (width - drawW) / 2
  const offsetY = (height - drawH) / 2
  return {
    x: offsetX + (p.x - bounds.minX) * scale,
    y: offsetY + (bounds.maxY - p.y) * scale,
  }
}

/**
 * fitWidth donne la largeur de canvas qui épouse le ratio de la scène à hauteur fixée :
 * une carte quasi carrée dans un conteneur large laisserait sinon d'immenses marges
 * latérales (la scène est cadrée sur la hauteur). Bornée par la largeur disponible.
 */
export function fitWidth(
  bounds: ReplayBounds,
  available: number,
  height: number,
  pad: number,
): number {
  const bw = Math.max(bounds.maxX - bounds.minX, 1e-6)
  const bh = Math.max(bounds.maxY - bounds.minY, 1e-6)
  const needed = (height - 2 * pad) * (bw / bh) + 2 * pad
  return Math.min(available, Math.max(needed, 2 * pad + 1))
}

/** canvasScale = pixels par unité monde pour le même cadrage que worldToCanvas. */
export function canvasScale(
  bounds: ReplayBounds,
  width: number,
  height: number,
  pad: number,
): number {
  const bw = Math.max(bounds.maxX - bounds.minX, 1e-6)
  const bh = Math.max(bounds.maxY - bounds.minY, 1e-6)
  return Math.min((width - 2 * pad) / bw, (height - 2 * pad) / bh)
}

/**
 * sceneBounds cadre la scène.
 *
 * AVEC UN SOL RECONSTRUIT, LE CADRE EST LA ZONE JOUÉE. La structure d'une carte couvre
 * ±250 m (skybox, décor lointain) là où les joueurs en parcourent 50 : cadrer sur elle
 * réduirait le terrain à un timbre au centre de l'écran. C'est aussi le cadrage du POC.
 *
 * SANS SOL, les props Forge redeviennent le seul fond, et ils débordent de la zone parcourue :
 * on cadre alors sur l'union, sinon ils seraient rognés.
 */
export function sceneBounds(doc: ReplayDocumentReady): ReplayBounds {
  if (doc.structure?.length) return doc.bounds
  const g = doc.geometryBounds
  if (!g) return doc.bounds
  return {
    minX: Math.min(doc.bounds.minX, g.minX),
    minY: Math.min(doc.bounds.minY, g.minY),
    maxX: Math.max(doc.bounds.maxX, g.maxX),
    maxY: Math.max(doc.bounds.maxY, g.maxY),
    minZ: doc.bounds.minZ,
    maxZ: doc.bounds.maxZ,
  }
}

/**
 * framesPerSecond donne la cadence de lecture « 1× » : le rejeu suit alors le temps réel
 * du match (frameIntervalMs = durée réelle d'une frame). Sans échelle temporelle dans
 * l'artefact, on retombe sur la cadence historique (axe = index de record).
 */
export function framesPerSecond(doc: ReplayDocumentReady): number {
  const interval = doc.frameIntervalMs ?? 0
  return interval > 0 ? 1000 / interval : FALLBACK_FPS
}

/** frameToMs convertit une frame en millisecondes écoulées depuis le début du rejeu. */
export function frameToMs(frame: number, doc: ReplayDocumentReady): number {
  const interval = doc.frameIntervalMs ?? 0
  return interval > 0 ? frame * interval : (frame / FALLBACK_FPS) * 1000
}

/** msToFrames convertit une durée réelle (ms) en nombre de frames. */
export function msToFrames(ms: number, doc: ReplayDocumentReady): number {
  const interval = doc.frameIntervalMs ?? 0
  return interval > 0 ? ms / interval : (ms / 1000) * FALLBACK_FPS
}

/** formatClock formate une durée en `m:ss` (chronomètre du rejeu). */
export function formatClock(ms: number): string {
  const total = Math.max(Math.floor(ms / 1000), 0)
  const sec = total % 60
  return `${Math.floor(total / 60)}:${sec < 10 ? '0' : ''}${sec}`
}

/**
 * trackWindow borne la vie d'une track sur l'axe de temps. Les champs sont omitempty
 * côté artefact : startFrame absent = 0, endFrame absent = t du dernier point.
 */
export function trackWindow(track: ReplayTrackReady): { start: number; end: number } {
  const last = track.points[track.points.length - 1]
  return {
    start: track.startFrame ?? 0,
    end: track.endFrame ?? (last ? last.t : 0),
  }
}

/** isAliveAt indique si la vie de la track couvre la frame (sinon : ne rien dessiner). */
export function isAliveAt(track: ReplayTrackReady, frame: number): boolean {
  const w = trackWindow(track)
  return frame >= w.start && frame <= w.end
}

/**
 * HeldReading — une mesure du film et SON ÂGE, en frames.
 *
 * POURQUOI L'ÂGE VOYAGE AVEC LA VALEUR. Le flux est différentiel : le film ne retransmet une
 * grandeur que lorsqu'elle CHANGE. Ce qui est à l'écran entre deux transmissions est donc la
 * DERNIÈRE lecture connue, pas l'état de l'instant. Afficher l'une pour l'autre sans le dire
 * a produit un défaut réel, observé à 1:06 sur un joueur dont l'état affiché datait de 9,7 s.
 * La mesure qui a fondé la correction dit que ce n'est pas un cas rare : sur les 21 899 fiches
 * d'un match, l'âge médian de la lecture est de 8,4 s et 7,1 % seulement ont moins d'une
 * seconde. Une mesure ancienne doit donc PÂLIR, jamais disparaître ni se faire passer pour
 * fraîche.
 */
export interface HeldReading {
  value: number
  /** Âge de la lecture, en frames. 0 = mesurée à cette image même. */
  age: number
}

/**
 * heldReading rend la dernière valeur transmise au plus tard à `frame`, avec son âge, ou null
 * si aucune n'existe dans la fenêtre `maxAge`.
 *
 * `pick` doit rendre `undefined` pour « non transmis ». ATTENTION : `0` est une VALEUR — un
 * bouclier nul est l'information la plus utile du champ. Tester la véracité au lieu de
 * `undefined` effacerait précisément le cas qui compte.
 */
export function heldReading(
  points: ReplayPoint[],
  frame: number,
  pick: (p: ReplayPoint) => number | undefined,
  maxAge: number,
): HeldReading | null {
  let i = lastIndexAt(points, frame)
  while (i >= 0) {
    const p = points[i]
    const age = frame - p.t
    if (age > maxAge) return null
    const v = pick(p)
    if (v !== undefined) return { value: v, age }
    i--
  }
  return null
}

/** lastIndexAt rend l'index du dernier point d'horodatage <= t (-1 si aucun), par dichotomie. */
export function lastIndexAt(points: ReplayPoint[], t: number): number {
  if (points.length === 0 || t < points[0].t) return -1
  let lo = 0
  let hi = points.length - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (points[mid].t <= t) lo = mid
    else hi = mid - 1
  }
  return lo
}

/**
 * freshness convertit un âge en facteur d'opacité dans [1 - fade, 1] : franc quand la mesure
 * vient d'arriver, atténué quand elle date. C'est la même graduation partout dans le rejeu —
 * cône de visée, bouclier, inventaire — pour qu'« ancien » se lise toujours de la même façon.
 */
export function freshness(age: number, hold: number, fade: number): number {
  if (hold <= 0) return 1
  return 1 - fade * Math.min(1, age / hold)
}

/** altitudeRatio normalise un Z dans [0,1] sur l'amplitude verticale connue (0,5 si plate). */
export function altitudeRatio(z: number, minZ: number, maxZ: number): number {
  const span = maxZ - minZ
  if (!(span > 1e-6)) return 0.5
  const r = (z - minZ) / span
  return r < 0 ? 0 : r > 1 ? 1 : r
}

/** floorOf renvoie l'index de tranche d'altitude (0 = bas) pour un Z donné. */
export function floorOf(z: number, minZ: number, maxZ: number, bands = FLOOR_BANDS): number {
  const idx = Math.floor(altitudeRatio(z, minZ, maxZ) * bands)
  return idx >= bands ? bands - 1 : idx
}

/**
 * footprint renvoie les 4 coins MONDE de l'emprise d'un prop Forge (rectangle centré sur
 * (x,y), de taille (dx,dy), tourné de `yaw` degrés autour de la verticale). Renvoie une
 * liste vide si l'objet n'a pas d'emprise : le rendu tombe alors sur un simple point.
 */
export function footprint(o: ReplayMapObject): XY[] {
  const dx = o.dx ?? 0
  const dy = o.dy ?? 0
  if (dx <= 0 || dy <= 0) return []
  const rad = ((o.yaw ?? 0) * Math.PI) / 180
  const cos = Math.cos(rad)
  const sin = Math.sin(rad)
  const hx = dx / 2
  const hy = dy / 2
  return [
    [-hx, -hy],
    [hx, -hy],
    [hx, hy],
    [-hx, hy],
  ].map(([ux, uy]) => ({
    x: o.x + ux * cos - uy * sin,
    y: o.y + ux * sin + uy * cos,
  }))
}
