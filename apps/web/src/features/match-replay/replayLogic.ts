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

/**
 * usefulHeight — LA HAUTEUR AU-DELÀ DE LAQUELLE ON N'AJOUTE PLUS DE CARTE, MAIS DU VIDE.
 *
 * C'est l'inverse exact de `fitWidth`, et il répond à une question que la hauteur fixe n'avait
 * jamais eu à poser : jusqu'où le terrain gagne-t-il à grandir ? `canvasScale` prend le PLUS
 * PETIT des deux rapports (largeur/largeur de scène, hauteur/hauteur de scène). Passé le point
 * où la largeur devient le facteur limitant, chaque pixel de hauteur en plus n'agrandit plus
 * rien : il ajoute une bande vide au-dessus et au-dessous de la carte.
 *
 * Ce plafond est donc PAR CARTE. Une carte quasi carrée peut occuper beaucoup de hauteur dans
 * une colonne large ; une carte très allongée sature bien avant. C'est ce que le plafond
 * constant ne pouvait pas exprimer.
 */
export function usefulHeight(bounds: ReplayBounds, available: number, pad: number): number {
  const bw = Math.max(bounds.maxX - bounds.minX, 1e-6)
  const bh = Math.max(bounds.maxY - bounds.minY, 1e-6)
  return Math.max((available - 2 * pad) * (bh / bw) + 2 * pad, 2 * pad + 1)
}

/**
 * LES PALIERS DE ZOOM, et pourquoi ce ne sont pas des crans continus.
 *
 * Changer le cadrage fait RECUIRE les quatre calques statiques (le sol et ses ~45 000 cellules
 * en tête). Un zoom continu — la molette — en déclencherait un par cran, donc des dizaines par
 * geste. Des paliers rendent chaque changement rare, prévisible, et NOMMABLE à l'écran : « 2x »
 * se lit, « 1,37x » ne veut rien dire pour qui regarde un match.
 */
export const ZOOM_LEVELS = [1, 1.5, 2, 3] as const
export type ZoomLevel = (typeof ZOOM_LEVELS)[number]

function clamp(v: number, lo: number, hi: number): number {
  return v < lo ? lo : v > hi ? hi : v
}

/**
 * clampCenter — le centre de la fenêtre, ramené là où la fenêtre reste DANS la scène.
 *
 * Exporté séparément de `visibleBounds` parce que l'état du zoom en a besoin sans vouloir les
 * bornes : quand on baisse le zoom, le centre courant peut devenir illégal (une fenêtre plus
 * large ne tient plus aussi près du bord) et il faut le rapprocher AVANT de le stocker — sinon
 * l'état garde une valeur que l'affichage corrige en silence, et les deux divergent.
 *
 * À zoom 1 la fenêtre vaut la scène : les deux bornes du `clamp` se rejoignent, il ne reste
 * qu'une position possible. La croix directionnelle se désactive donc d'elle-même, sans que
 * personne ait à écrire la règle « pas de déplacement à zoom 1 ».
 */
export function clampCenter(
  scene: ReplayBounds,
  zoom: number,
  cx: number,
  cy: number,
): { x: number; y: number } {
  const halfW = Math.max(scene.maxX - scene.minX, 0) / (2 * Math.max(zoom, 1))
  const halfH = Math.max(scene.maxY - scene.minY, 0) / (2 * Math.max(zoom, 1))
  return {
    x: clamp(cx, scene.minX + halfW, scene.maxX - halfW),
    y: clamp(cy, scene.minY + halfH, scene.maxY - halfH),
  }
}

/**
 * visibleBounds — LA FENÊTRE VISIBLE, et c'est TOUT ce que le zoom change.
 *
 * # POURQUOI PAR LES BORNES, ET PAS PAR UNE ÉCHELLE
 *
 * La tentation était d'ajouter `{échelle, panX, panY}` au cadrage et de les appliquer partout.
 * Cela aurait touché `worldToCanvas`, `canvasScale`, le survol, les quatre calques et les
 * infobulles — et surtout introduit DEUX façons de dire où tombe un point du monde, qui
 * auraient divergé au premier oubli.
 *
 * Or la projection est entièrement définie par les BORNES : `worldToCanvas` mappe `bounds` vers
 * la toile. Zoomer, c'est donc rétrécir les bornes ; se déplacer, c'est les translater. Rien
 * d'autre ne bouge — le survol lit la même projection que le dessin, il suit sans une ligne.
 *
 * # CE QUE ÇA RÈGLE GRATUITEMENT
 *
 * Les calques statiques cuisent depuis le cadrage : avec des bornes rétrécies, ils cuisent la
 * FENÊTRE à la résolution de l'écran. Leur surface ne dépend donc PAS du niveau de zoom — la
 * crainte d'une mémoire qui enfle avec le grossissement est évitée par construction, et non par
 * une précaution qu'il faudrait maintenir.
 *
 * # L'ASPECT EST PRÉSERVÉ
 *
 * Les deux dimensions sont divisées par le MÊME facteur. Sans cela, `usefulHeight` (le plafond
 * de hauteur calculé par carte) et le cadrage se contrediraient à chaque palier : la fenêtre
 * réclamerait une forme que la toile ne peut pas lui donner, et la carte flotterait dans des
 * bandes vides qui changeraient de taille à chaque cran.
 *
 * L'amplitude verticale (`minZ`/`maxZ`) traverse inchangée : le zoom est plan, il ne dit rien
 * des étages.
 */
export function visibleBounds(
  scene: ReplayBounds,
  zoom: number,
  cx: number,
  cy: number,
): ReplayBounds {
  const z = Math.max(zoom, 1)
  const halfW = Math.max(scene.maxX - scene.minX, 0) / (2 * z)
  const halfH = Math.max(scene.maxY - scene.minY, 0) / (2 * z)
  const c = clampCenter(scene, z, cx, cy)
  return {
    minX: c.x - halfW,
    maxX: c.x + halfW,
    minY: c.y - halfH,
    maxY: c.y + halfH,
    minZ: scene.minZ,
    maxZ: scene.maxZ,
  }
}

/** Le centre de la scène — la position de départ du cadrage, et celle du retour à 1x. */
export function sceneCenter(scene: ReplayBounds): { x: number; y: number } {
  return { x: (scene.minX + scene.maxX) / 2, y: (scene.minY + scene.maxY) / 2 }
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
 * `hasMapImage` ÉTEND CETTE MÊME RÈGLE AU FOND DE CARTE, et le défaut qu'elle corrige était
 * visible à l'écran (2026-08-26) : quand une image est posée, `ReplayCanvas` ne dessine PAS
 * les props Forge (ils sont le `else if` du fond), mais l'union les gardait au dénominateur
 * du cadre. Le cadre était donc dimensionné sur une matière INVISIBLE, et la carte se
 * réduisait à un timbre dans un canvas vide. Une image de carte est un fond au même titre
 * qu'un sol reconstruit : elle rend les props inutiles au cadrage.
 *
 * SANS SOL NI IMAGE, les props Forge redeviennent le seul fond, et ils débordent de la zone
 * parcourue : on cadre alors sur l'union, sinon ils seraient rognés.
 */
export function sceneBounds(doc: ReplayDocumentReady, hasMapImage = false): ReplayBounds {
  if (hasMapImage || doc.structure?.length) return doc.bounds
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

/** formatSeconds rend un délai court en secondes avec une décimale (virgule décimale FR/EN). */
export function formatSeconds(ms: number): string {
  return `${(Math.max(0, ms) / 1000).toFixed(1)} s`
}

/**
 * READING_FADE — une lecture au plancher d'âge garde la moitié de son opacité : elle
 * s'estompe, elle ne disparaît pas. C'est le `fade` passé à `freshness` par toutes les
 * surfaces des fiches (vitals, armes, inventaire), pour qu'« ancien » se lise pareil partout.
 */
export const READING_FADE = 0.5

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
 *
 * L'âge peut être NÉGATIF (lecture d'une image-clé à venir, cf. loadoutAt) : une lecture
 * éloignée dans le temps est une lecture éloignée, quel que soit le sens — l'estompage porte
 * sur la valeur absolue.
 */
export function freshness(age: number, hold: number, fade: number): number {
  if (hold <= 0) return 1
  return 1 - fade * Math.min(1, Math.abs(age) / hold)
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
