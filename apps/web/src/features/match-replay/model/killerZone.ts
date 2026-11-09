/**
 * killerZone.ts — LA ZONE NOMMÉE OÙ SE TROUVAIT LE TUEUR, pour chaque frag du fil.
 *
 * C'EST LE JUMEAU, ÉCRIT EN TYPESCRIPT, DE `NommerZone` (apps/go-api/internal/analysis/tactical/
 * zones.go et geometrie.go). Les deux ports tiennent le même contrat, prouvé par un jeu de cas
 * UNIQUE (apps/go-api/internal/testfixtures/testdata/cartes/nommage_zone_cas.json) que
 * `zones_test.go` et `killerZone.test.ts` lisent chacun : modifier une règle ici sans la modifier
 * dans le fichier Go (ou l'inverse) fait échouer l'un des deux tests.
 *
 * LA RÈGLE, DANS L'ORDRE (un point du plan, et le z du tueur à cet instant) :
 *
 *   (a) polygone  les zones dont la FORME contient le point ET dont la tranche
 *                 [zBottom − 0,25 ; zTop + 0,25] contient le z : une seule, c'est elle ;
 *   (b) empilée   plusieurs : la tranche la plus ÉTROITE ; à égalité parfaite, l'indice de volume ;
 *   (c) proche    aucune : la zone dont la forme est la plus proche du point, à MOINS de 2 m
 *                 (distance au bord, 0 si le point est dedans), parmi les tranches compatibles
 *                 d'abord, parmi toutes ensuite ;
 *   (d)           sinon, PAS DE NOM — jamais un nom de repli.
 *
 * LA FORME est celle que le calque dessine : contour, parties détachées et trous en règle
 * PAIR-IMPAIR (un point est dans la zone s'il est dans un nombre impair d'anneaux ; depuis un
 * trou, le bord le plus proche est celui du trou).
 *
 * Z INCONNU (les pistes du document n'en portent pas) : (a) sans test de tranche quand UNE SEULE
 * forme contient le point ; plusieurs formes empilées ne nomment RIEN — l'étage ne se devine pas ;
 * aucune : (c) sans test de tranche.
 *
 * UN FRAG, UN SEUL z. Le Go nomme une cellule du plan d'après PLUSIEURS événements (z médian,
 * majorité de tranches) ; ici la liste a zéro ou un z, et `nameZone` garde la signature et la
 * règle complètes du jumeau pour ne pas diverger de lui.
 *
 * L'INSTANT EST BRUT : `replayMs` du fil, AVANT `displayClockMs` (qui ne recale que l'affiché).
 * La position vient du résolveur de porteur (un tueur au volant est sur son véhicule) et le z de
 * la piste du bipède, sous la même règle de vie. Un tueur sans position à cet instant n'a pas de
 * zone : la carte n'a pas d'entrée, la ligne n'écrit rien.
 *
 * Pas de React, pas de canvas : logique pure, testée (killerZone.test.ts).
 */
import type { CalloutZoneReady } from '../../../lib/replay/calloutsPaint'
import { msToFrames, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildCarrierPosAt } from './carrierPosition'
import type { ReplayFeedEntry } from './killFeedLogic'
import { altitudeOfPlayerAt, buildLivesByXuid, deathWindowFrames } from './livesPosition'

/** Marge de tranche, en mètres : un joueur debout sur une dalle a les pieds sous son plancher déclaré. */
export const ZONE_SLICE_MARGIN_M = 0.25

/** Distance au bord, en mètres, en deçà de laquelle une zone voisine nomme un point hors de toute forme (borne stricte). */
export const ZONE_NEAR_RADIUS_M = 2

/** La règle qui a nommé un point (les mêmes mots que le Go). */
export type ZoneRule = 'polygone' | 'empilee' | 'proche'

/** La zone qui nomme un point, la règle qui l'a retenue et sa distance au bord (0 si le point est dedans). */
export interface NamedZone {
  zone: CalloutZoneReady
  rule: ZoneRule
  distanceM: number
}

type Ring = readonly XY[]

/** Les anneaux d'une zone : contour, parties, trous. Moins de trois sommets : pas une surface. */
function ringsOf(z: CalloutZoneReady): Ring[] {
  return [z.polygon, ...z.parts, ...z.holes].filter((r) => r.length >= 3)
}

/** La zone porte-t-elle au moins un contour ou une partie exploitable ? */
function hasShape(z: CalloutZoneReady): boolean {
  return z.polygon.length >= 3 || z.parts.some((p) => p.length >= 3)
}

/** Le point est-il dans l'anneau (test du rayon, pair-impair) ? */
function inRing(x: number, y: number, ring: Ring): boolean {
  let inside = false
  for (let i = 0, k = ring.length - 1; i < ring.length; k = i, i++) {
    const { x: xi, y: yi } = ring[i]
    const { x: xk, y: yk } = ring[k]
    if (yi > y !== yk > y && x < ((xk - xi) * (y - yi)) / (yk - yi) + xi) inside = !inside
  }
  return inside
}

/** Le point appartient-il à la forme (pair-impair sur ses anneaux) ? */
function inShape(x: number, y: number, z: CalloutZoneReady): boolean {
  let inside = false
  for (const r of ringsOf(z)) if (inRing(x, y, r)) inside = !inside
  return inside
}

/** Distance du point au segment [a, b]. */
function distanceToSegment(x: number, y: number, a: XY, b: XY): number {
  const dx = b.x - a.x
  const dy = b.y - a.y
  const l2 = dx * dx + dy * dy
  const t = l2 > 0 ? Math.max(0, Math.min(1, ((x - a.x) * dx + (y - a.y) * dy) / l2)) : 0
  return Math.hypot(x - (a.x + t * dx), y - (a.y + t * dy))
}

/** Plus courte distance du point à une arête de la forme (Infinity sans forme). */
function distanceToEdge(x: number, y: number, z: CalloutZoneReady): number {
  let best = Number.POSITIVE_INFINITY
  for (const r of ringsOf(z)) {
    for (let i = 0, k = r.length - 1; i < r.length; k = i, i++) {
      best = Math.min(best, distanceToSegment(x, y, r[k], r[i]))
    }
  }
  return best
}

/** 0 quand le point est dans la forme, sa distance au bord sinon. */
function distanceToZone(x: number, y: number, z: CalloutZoneReady): number {
  return inShape(x, y, z) ? 0 : distanceToEdge(x, y, z)
}

/** Le z est-il dans la tranche de la zone, marge comprise ? */
function inSlice(z: CalloutZoneReady, h: number): boolean {
  return h >= z.zBottom - ZONE_SLICE_MARGIN_M && h <= z.zTop + ZONE_SLICE_MARGIN_M
}

/** La hauteur de la tranche déclarée. */
function thickness(z: CalloutZoneReady): number {
  return z.zTop - z.zBottom
}

/** Le z médian (moyenne des deux du milieu pour un nombre pair). */
function median(v: readonly number[]): number {
  const s = [...v].sort((a, b) => a - b)
  const n = s.length
  return n % 2 === 1 ? s[(n - 1) / 2] : (s[n / 2 - 1] + s[n / 2]) / 2
}

/**
 * Règle (b) : plusieurs zones contiennent le point et le z médian. La majorité de tranches
 * d'abord (plus de la moitié des z), puis à défaut la plus peuplée, puis la plus étroite, puis
 * l'indice de volume. Le tri est stable, comme `sort.SliceStable` côté Go.
 */
function pickStacked(candidates: readonly CalloutZoneReady[], zs: readonly number[]): CalloutZoneReady {
  const scored = candidates.map((zone) => ({ zone, count: zs.filter((h) => inSlice(zone, h)).length }))
  const majority = (c: { count: number }) => 2 * c.count > zs.length
  scored.sort((a, b) => {
    const ma = majority(a)
    const mb = majority(b)
    if (ma !== mb) return ma ? -1 : 1
    if (!ma && a.count !== b.count) return b.count - a.count
    const ta = thickness(a.zone)
    const tb = thickness(b.zone)
    if (ta !== tb) return ta - tb
    return a.zone.volumeIndex - b.zone.volumeIndex
  })
  return scored[0].zone
}

/**
 * Règle (c) : à moins de 2 m, parmi les zones `compatible` d'abord (undefined = aucun premier
 * passage), parmi toutes ensuite.
 */
function nearest(
  x: number,
  y: number,
  shapes: readonly CalloutZoneReady[],
  compatible?: (z: CalloutZoneReady) => boolean,
): NamedZone | null {
  const pass = (keep?: (z: CalloutZoneReady) => boolean): NamedZone | null => {
    let found: CalloutZoneReady | null = null
    let best = Number.POSITIVE_INFINITY
    for (const z of shapes) {
      if (keep && !keep(z)) continue
      const d = distanceToZone(x, y, z)
      if (d >= ZONE_NEAR_RADIUS_M) continue
      if (found === null || d < best || (d === best && z.volumeIndex < found.volumeIndex)) {
        found = z
        best = d
      }
    }
    return found ? { zone: found, rule: 'proche', distanceM: best } : null
  }
  return (compatible ? pass(compatible) : null) ?? pass()
}

/** La règle quand aucun z n'est connu. */
function nameWithoutHeight(x: number, y: number, shapes: readonly CalloutZoneReady[]): NamedZone | null {
  const containing = shapes.filter((z) => inShape(x, y, z))
  if (containing.length === 0) return nearest(x, y, shapes)
  if (containing.length === 1) return { zone: containing[0], rule: 'polygone', distanceM: 0 }
  return null
}

/**
 * nameZone rend la zone qui nomme le point (x, y) compte tenu des z connus (`zs`, vide =
 * inconnu), ou null : aucune zone ne le nomme.
 */
export function nameZone(
  x: number,
  y: number,
  zs: readonly number[],
  zones: readonly CalloutZoneReady[],
): NamedZone | null {
  const shapes = zones.filter(hasShape)
  if (zs.length === 0) return nameWithoutHeight(x, y, shapes)
  const zm = median(zs)
  const candidates = shapes.filter((z) => inShape(x, y, z) && inSlice(z, zm))
  if (candidates.length === 0) return nearest(x, y, shapes, (z) => inSlice(z, zm))
  if (candidates.length === 1) return { zone: candidates[0], rule: 'polygone', distanceM: 0 }
  return { zone: pickStacked(candidates, zs), rule: 'empilee', distanceM: 0 }
}

/** Aucune zone : la valeur partagée quand la carte n'en a pas ou que le document est absent. */
const NO_ZONES: ReadonlyMap<string, CalloutZoneReady> = new Map()

/**
 * killerZones — la zone du tueur de chaque frag du fil, PAR CLÉ DE LIGNE (`ReplayFeedEntry.key`).
 * Un frag dont le tueur n'a pas de position à son instant, ou que le nommage laisse sans nom, n'a
 * PAS d'entrée : jamais un nom deviné. Les lignes qui ne sont pas des frags n'en ont pas non plus.
 */
export function killerZones(
  entries: readonly ReplayFeedEntry[],
  doc: ReplayDocumentReady,
  zones: readonly CalloutZoneReady[],
): ReadonlyMap<string, CalloutZoneReady> {
  if (zones.length === 0 || doc.tracks.length === 0) return NO_ZONES
  const posOf = buildCarrierPosAt(doc)
  const lives = buildLivesByXuid(doc.tracks)
  const deathFrames = deathWindowFrames(doc)
  const out = new Map<string, CalloutZoneReady>()
  for (const e of entries) {
    const k = e.kill
    if (!k) continue
    const frame = Math.round(msToFrames(k.replayMs, doc))
    const at = posOf(k.xuid, frame)
    if (!at) continue
    const z = altitudeOfPlayerAt(lives.get(k.xuid), frame, deathFrames)
    const named = nameZone(at.x, at.y, z === null ? [] : [z], zones)
    if (named) out.set(e.key, named.zone)
  }
  return out
}
