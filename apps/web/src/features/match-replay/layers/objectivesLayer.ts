/**
 * objectivesLayer.ts — le CALQUE STATIQUE des objectifs du mode joué (lot 4) : normalisation,
 * géométrie et pulses d'action. Logique pure, pas de React.
 *
 * L'ÉTAT VIVANT DES ZONES VIT À CÔTÉ (`zoneStatesLayer.ts`, schéma 16). La frontière est celle du
 * TEMPS : ici la géométrie, qui ne change jamais et se cuit une fois hors écran ; là l'état, qui
 * change à chaque image et se peint dans la boucle. Les deux tracent la MÊME forme — `traceZonePath`
 * est exporté pour cela, plutôt que recopié.
 *
 * LES PULSES RESTENT, ET CE N'EST PAS UN DOUBLON. Ils marquent l'INSTANT d'une action (une
 * capture vient d'avoir lieu, un anneau s'ouvre et s'éteint) ; l'état vivant, lui, décrit une
 * DURÉE. La bascule de teinte d'une zone est d'ailleurs ce que le pulse annonce — les deux se
 * lisent ensemble, l'un ponctuel, l'autre continu.
 *
 * SAUF POUR LE DRAPEAU, ET C'EST LE LOT 3.1 (schéma 15). Le pulse de CTF n'était PAS un instant
 * annoncé : c'était un SUBSTITUT, faute d'objet — il posait l'action sur l'élément statique le
 * plus proche de son auteur, c'est-à-dire sur un socle voisin, jamais sur le drapeau. Depuis que
 * `flagCarries` publie l'objet lui-même (position, porteur, état, image par image), ce substitut
 * dirait la MÊME chose en moins juste, et les deux se contrediraient à l'écran. Il est donc
 * RETIRÉ dès que le document porte des drapeaux (cf. `buildObjectivePulses`). Les pulses de zone
 * et d'Oddball, eux, n'ont pas d'objet vivant et gardent leur rôle entier — la fonction n'est
 * donc pas supprimée, seule la FAMILLE drapeau sort.
 *
 * CE QUE LE SERVEUR A DÉJÀ DÉCIDÉ, et que ce calque ne rejoue pas : quels rôles servir
 * (table du titre jointe au pair_name), quelles équipes afficher (les modes à possession
 * dynamique arrivent neutres). Le front dessine CE QUI ARRIVE — un document sans
 * `mapObjectives` n'a simplement pas de calque.
 *
 * AUCUN LIBELLÉ, ET C'EST UNE RÈGLE : la lettre A/B/C affichée en jeu n'existe dans
 * aucune donnée décodée (le rang spatial du serveur n'est PAS un nom, cf.
 * objectives_catalog.go). Ce calque n'écrit donc JAMAIS de texte — le garde
 * `drawObjectivesLayer n'appelle ni fillText ni strokeText` est testé.
 *
 * GÉOMÉTRIE : mêmes transforms monde -> canvas que structure/tracks (worldToCanvas,
 * canvasScale). Une boîte arrive en demi-extents + vecteur Forward (projeté au plan,
 * normalisé ici) ; un cylindre en rayon monde. Les marqueurs sont des POINTS : un
 * losange (apparition/socle), doublé d'un anneau pour une livraison — jamais un disque
 * de zone inventé (règle shape.go).
 */
import type { ReplayMapObjectives } from '@/lib/api/types'

import { buildCarrierPosAt } from '../model/carrierPosition'
import { type XY } from '../../../lib/replay/replayLogic'
import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { type CanvasView, projectTo, scaleOf } from '../model/replayView'

/** Valeur « aucun camp » du team_index — celle du fichier de carte, servie telle quelle. */
export const OBJECTIVE_TEAM_NEUTRAL = -1

/** Un élément d'objectif prêt à dessiner : nullabilité résolue, Forward normalisé 2D. */
export interface ObjectiveElementReady {
  role: string
  /** Index d'équipe À AFFICHER : -1 = neutre (déjà arbitré côté serveur). */
  team: number
  x: number
  y: number
  z: number
  kind: 'zone' | 'marker'
  family?: 'box' | 'cylinder'
  /** Boîte : demi-cotés monde le long de fwd et de sa perpendiculaire. */
  halfX: number
  halfY: number
  /** Cylindre : rayon monde. */
  radius: number
  /** Axe de la boîte, unitaire 2D (les repères dégénérés sont refusés côté serveur). */
  fwd: XY
}

/**
 * normalizeMapObjectives résout la nullabilité du transport UNE fois, à l'entrée (même
 * règle que normalizeCallouts). L'ordre servi (tri spatial serveur) est conservé.
 */
export function normalizeMapObjectives(
  mo: ReplayMapObjectives | null | undefined,
): ObjectiveElementReady[] {
  if (!mo) return []
  const out: ObjectiveElementReady[] = []
  for (const z of mo.zones ?? []) {
    out.push({
      role: z.role,
      team: z.team,
      x: z.x,
      y: z.y,
      z: z.z,
      kind: 'zone',
      family: z.family === 'cylinder' ? 'cylinder' : 'box',
      halfX: z.halfX ?? 0,
      halfY: z.halfY ?? 0,
      radius: z.radius ?? 0,
      fwd: unit2D(z.fwdX, z.fwdY),
    })
  }
  for (const m of mo.markers ?? []) {
    out.push({
      role: m.role,
      team: m.team,
      x: m.x,
      y: m.y,
      z: m.z,
      kind: 'marker',
      halfX: 0,
      halfY: 0,
      radius: 0,
      fwd: { x: 1, y: 0 },
    })
  }
  return out
}

/**
 * unit2D normalise la projection au plan du Forward. Le serveur refuse les repères
 * dégénérés (mapvar) et l'axe Up des zones est vertical sur tout le catalogue mesuré :
 * la projection est donc un vecteur horizontal non nul — le repli (1,0) ne sert que de
 * ceinture si une donnée future violait l'invariant, et il rend une boîte alignée.
 */
function unit2D(x: number | undefined, y: number | undefined): XY {
  const vx = x ?? 0
  const vy = y ?? 0
  const n = Math.hypot(vx, vy)
  if (n < 1e-6) return { x: 1, y: 0 }
  return { x: vx / n, y: vy / n }
}

/** Style du calque : la couleur d'équipe est RÉSOLUE par l'appelant (règle color-tokens). */
export interface ObjectivesStyle {
  /** Couleur d'un index d'équipe ; -1 (neutre) rend l'encre neutre du thème. */
  colorOfTeam: (team: number) => string
}

// Réglages du calque : assez francs pour se lire sous les trajectoires, assez bas pour
// ne pas concurrencer les joueurs (mêmes ordres de grandeur que les callouts).
const ZONE_FILL_ALPHA = 0.09
const ZONE_STROKE_ALPHA = 0.6
const ZONE_STROKE_WIDTH = 1.5
const MARKER_SIZE = 5.5
const MARKER_RING = 8
const MARKER_ALPHA = 0.9

/**
 * drawObjectivesLayer peint zones puis marqueurs. Calque STATIQUE — l'appelant le cuit
 * hors écran et le recopie, comme le sol et les callouts. AUCUN texte (cf. en-tête).
 */
export function drawObjectivesLayer(
  ctx: CanvasRenderingContext2D,
  elements: ObjectiveElementReady[],
  view: CanvasView,
  style: ObjectivesStyle,
): void {
  const px = (p: XY) => projectTo(view, p)
  const scale = scaleOf(view)

  for (const e of elements) {
    const color = style.colorOfTeam(e.team)
    if (e.kind === 'zone') drawZone(ctx, e, px, scale, color)
  }
  // Les marqueurs par-dessus les zones : une livraison ponctuelle vit parfois DANS son
  // cylindre (mesuré sur Catalyst) et doit rester visible.
  for (const e of elements) {
    if (e.kind !== 'marker') continue
    drawMarker(ctx, e, px, style.colorOfTeam(e.team))
  }
  ctx.globalAlpha = 1
}

/** Zone : boîte ORIENTÉE (4 coins monde) ou cylindre (rayon monde -> pixels). */
function drawZone(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  px: (p: XY) => XY,
  scale: number,
  color: string,
): void {
  traceZonePath(ctx, e, px, scale)
  ctx.globalAlpha = ZONE_FILL_ALPHA
  ctx.fillStyle = color
  ctx.fill()
  ctx.globalAlpha = ZONE_STROKE_ALPHA
  ctx.strokeStyle = color
  ctx.lineWidth = ZONE_STROKE_WIDTH
  ctx.stroke()
}

/**
 * Marqueur : LOSANGE plein (apparition, socle) ; une LIVRAISON (`*_delivery`) gagne un
 * anneau — c'est un point d'arrivée, pas une apparition, et la différence se lit sans
 * texte.
 */
function drawMarker(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  px: (p: XY) => XY,
  color: string,
): void {
  const c = px(e)
  ctx.globalAlpha = MARKER_ALPHA
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.beginPath()
  ctx.moveTo(c.x, c.y - MARKER_SIZE)
  ctx.lineTo(c.x + MARKER_SIZE, c.y)
  ctx.lineTo(c.x, c.y + MARKER_SIZE)
  ctx.lineTo(c.x - MARKER_SIZE, c.y)
  ctx.closePath()
  ctx.fill()
  if (e.role.endsWith('_delivery')) {
    ctx.lineWidth = 1.5
    ctx.beginPath()
    ctx.arc(c.x, c.y, MARKER_RING, 0, Math.PI * 2)
    ctx.stroke()
  }
}

/**
 * Préfixe des statistiques d'objectif de CAPTURE DE DRAPEAU, tel que le serveur les nomme
 * (`objectiveevents/named.go` : `flag_grabs`, `flag_steals`, `flag_captures`, `flag_returns`,
 * `flag_carriers_killed`, `flag_capture_assists`). C'est ce préfixe qui distingue une action
 * dont l'OBJET est désormais publié d'une action de zone ou de crâne, qui n'en a pas.
 */
const FLAG_STAT_PREFIX = 'flag_'

/**
 * flagPulsesRetired dit si la famille DRAPEAU des pulses doit se taire : elle se tait dès que le
 * document publie la vie des drapeaux, parce qu'alors le substitut a un remplaçant EXACT.
 *
 * Ce n'est PAS conditionné à la bascule d'affichage du calque : un lecteur qui éteint les
 * drapeaux demande moins de choses à l'écran, pas le retour d'une approximation.
 */
export function flagPulsesRetired(doc: ReplayDocumentReady): boolean {
  return doc.flagCarries.length > 0
}

/** Un pulse : une ACTION d'objectif (doc.objectives) posée sur son élément le plus proche. */
export interface ObjectivePulse {
  frame: number
  x: number
  y: number
  team: number
}

/**
 * buildObjectivePulses apparie chaque action d'objectif du document à l'élément servi le
 * plus proche de son AUTEUR à l'instant de l'action (position relue par `buildCarrierPosAt` :
 * le VÉHICULE quand l'auteur y est embarqué, sinon ses vies de bipède et la même fenêtre
 * après-mort que le calque des morts).
 *
 * LA PROXIMITÉ EST 2D : la position interpolée d'une trace est XY (le z ne voyage pas
 * dans positionAt), et les objectifs d'un même mode ne se superposent pas en plan sur
 * les cartes mesurées. Une action sans position relue est ÉCARTÉE — un pulse posé au
 * hasard désignerait la mauvaise zone.
 *
 * `a.t` EST DÉJÀ UNE FRAME DU DOCUMENT — LE CLIENT NE RETRANCHE RIEN. C'est le contrat du
 * champ, écrit côté Go : `ObjectiveAction.T` est « l'index de frame, sur le même axe que
 * Point.T et Shot.T » (`analysis/replay/objectives.go`), et c'est `buildObjectiveActions`
 * qui pose l'instant sur la grille via `scoreClock.frameOf` — laquelle RETRANCHE l'origine
 * (`build_score.go`, `replayScoreClock` : `originMS = originMSOf(doc.OriginMs, …)`).
 *
 * CETTE FONCTION LA RETRANCHAIT UNE SECONDE FOIS (revue R1, 2026-08-18). La correction avait
 * été faite côté client le 2026-08-14 (lot containment), puis côté Go au lot A phase 1
 * (`63b90583c`, report `:123` du registre) sans que celle-ci soit retirée. Le double décalage
 * allumait les pulses `originMs` TROP TÔT — de 3,6 s à 50,8 s selon le match — et JETAIT
 * purement et simplement les actions dont la frame était inférieure à cette origine.
 *
 * Le fil des éliminations, lui, garde sa soustraction (`killFeedLogic`, `replayMs =
 * event_time_ms + t0Ms − originMs`) et ce n'est pas une incohérence : ses instants viennent
 * de la Match View, pas du film — personne ne les a recalés avant lui.
 */
export function buildObjectivePulses(
  doc: ReplayDocumentReady,
  elements: ObjectiveElementReady[],
): ObjectivePulse[] {
  if (elements.length === 0 || doc.objectives.length === 0 || doc.tracks.length === 0) return []
  // L'ORIGINE DOIT ÊTRE CONNUE POUR QUE `a.t` VEUILLE DIRE QUELQUE CHOSE (P2 de la revue du
  // lot A phase 1). Le recalage est fait côté Go, mais avec ZÉRO quand l'origine n'a pas pu
  // être établie (`replayScoreClock` → `originMSOf`) : les frames publiées sont alors décalées
  // de 3,6 s à 50,8 s selon le match, et l'appariement lirait la position de l'auteur ailleurs.
  // `coverage.originResolved` le dit, et un calque muet vaut mieux qu'un calque faux — c'est la
  // MÊME règle qui masque le score (cf. filmClockTrusted).
  if (!filmClockTrusted(doc)) return []
  // LE SUBSTITUT DU DRAPEAU SORT ICI, à la source, plutôt qu'au tracé : un pulse construit puis
  // non dessiné resterait dans la mémoire de la scène et dans les dépendances de `draw`.
  const dropFlags = flagPulsesRetired(doc)
  const posOf = buildCarrierPosAt(doc)
  const out: ObjectivePulse[] = []
  for (const a of doc.objectives) {
    if (dropFlags && a.stat.startsWith(FLAG_STAT_PREFIX)) continue
    // AUCUN RECALAGE ICI : `a.t` est déjà une frame du document (cf. en-tête). L'action que
    // la grille ne portait pas a été comptée hors fenêtre côté Go et n'est pas publiée.
    const frame = a.t
    const pos = posOf(a.xuid, frame)
    if (!pos) continue
    let best: ObjectiveElementReady | null = null
    let bd = Infinity
    for (const e of elements) {
      const dx = e.x - pos.x
      const dy = e.y - pos.y
      const d = dx * dx + dy * dy
      if (d < bd) {
        bd = d
        best = e
      }
    }
    if (!best) continue
    out.push({ frame, x: best.x, y: best.y, team: best.team })
  }
  return out
}

/** Fenêtre d'affichage d'un pulse (forme d'EventWindow de replayDraw). */
interface PulseWindow {
  frame: number
  hold: number
}

/**
 * drawObjectivePulses dessine les pulses de la fenêtre courante : un anneau qui S'OUVRE
 * depuis l'élément (l'action vient d'y avoir lieu) puis s'éteint. Sous « mouvement
 * réduit » : anneau statique, opacité constante — même règle que les autres effets.
 */
export function drawObjectivePulses(
  ctx: CanvasRenderingContext2D,
  pulses: ObjectivePulse[],
  view: CanvasView,
  win: PulseWindow,
  style: ObjectivesStyle,
  reducedMotion: boolean,
): void {
  for (const p of pulses) {
    const age = win.frame - p.frame
    if (age < 0 || age > win.hold) continue
    const k = age / Math.max(win.hold, 1)
    const c = projectTo(view, p)
    ctx.strokeStyle = style.colorOfTeam(p.team)
    ctx.lineWidth = 2
    ctx.globalAlpha = reducedMotion ? 0.6 : 0.9 * (1 - k)
    ctx.beginPath()
    ctx.arc(c.x, c.y, reducedMotion ? 11 : 7 + 14 * k, 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}

/**
 * zoneCornersWorld rend les QUATRE COINS MONDE d'une zone en boîte : centre ± fwd·halfX ±
 * perp·halfY. La perpendiculaire est le fwd tourné de +90° monde — l'inversion d'axe Y est
 * portée par `worldToCanvas`, jamais ici.
 *
 * IL EST EXPORTÉ, ET C'EST LA RAISON D'ÊTRE DE L'EXTRACTION (2026-08-25, item D-R). Deux
 * lecteurs ont besoin de cette formule : le TRACÉ du contour (`traceZonePath`) et l'EMPRISE
 * écran de la zone (`zoneStatesLayer`, pour clipper le remplissage progressif de capture). La
 * recopier ferait exactement ce que l'en-tête de `traceZonePath` interdisait déjà — deux
 * géométries qui divergent au premier correctif, avec un écart invisible parce que crédible.
 */
export function zoneCornersWorld(e: ObjectiveElementReady): XY[] {
  const perp = { x: -e.fwd.y, y: e.fwd.x }
  return [
    { x: e.x + e.fwd.x * e.halfX + perp.x * e.halfY, y: e.y + e.fwd.y * e.halfX + perp.y * e.halfY },
    { x: e.x - e.fwd.x * e.halfX + perp.x * e.halfY, y: e.y - e.fwd.y * e.halfX + perp.y * e.halfY },
    { x: e.x - e.fwd.x * e.halfX - perp.x * e.halfY, y: e.y - e.fwd.y * e.halfX - perp.y * e.halfY },
    { x: e.x + e.fwd.x * e.halfX - perp.x * e.halfY, y: e.y + e.fwd.y * e.halfX - perp.y * e.halfY },
  ]
}

/**
 * zoneCanvasRadius rend le rayon ÉCRAN d'une zone cylindrique, plancher compris. Exporté pour
 * la même raison que `zoneCornersWorld` : le tracé et l'emprise doivent lire le MÊME rayon, y
 * compris son plancher — sinon l'emprise et le contour se décalent sur les toutes petites zones.
 */
export function zoneCanvasRadius(e: ObjectiveElementReady, scale: number): number {
  return Math.max(e.radius * scale, 2)
}

/**
 * traceZonePath pose le contour d'une zone — boîte ORIENTÉE (4 coins monde) ou cylindre (rayon
 * monde -> pixels).
 *
 * IL EST EXPORTÉ PARCE QUE DEUX CALQUES LE TRACENT : celui-ci (géométrie, cuite une fois) et
 * l'état vivant des zones (`zoneStatesLayer.ts`, repeint à chaque image). Deux copies de la
 * même forme divergeraient au premier correctif de géométrie — et l'écart serait invisible :
 * un contour légèrement faux reste crédible.
 */
export function traceZonePath(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  px: (p: XY) => XY,
  scale: number,
): void {
  ctx.beginPath()
  if (e.family === 'cylinder') {
    const c = px(e)
    ctx.arc(c.x, c.y, zoneCanvasRadius(e, scale), 0, Math.PI * 2)
    return
  }
  zoneCornersWorld(e).forEach((w, i) => {
    const c = px(w)
    if (i === 0) ctx.moveTo(c.x, c.y)
    else ctx.lineTo(c.x, c.y)
  })
  ctx.closePath()
}
