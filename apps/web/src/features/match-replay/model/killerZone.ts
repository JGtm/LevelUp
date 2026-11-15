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
 *                 d'abord, parmi toutes ensuite ; à égalité de distance, l'indice de volume ;
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
 * LE PORT TRAITE UN z UNIQUE. Le Go nomme une cellule du plan d'après PLUSIEURS événements (z
 * médian, majorité de tranches) ; un frag est UN événement, donc UN z (ou aucun). Avec un seul z,
 * la médiane est ce z et toute tranche qui le contient détient la majorité : ces deux étapes du
 * jumeau ne sont pas portées, et le jeu de cas partagé n'en exerce aucune. Le côté Go garde ses
 * tests multi-z propres.
 *
 * ZONES SANS NOM : une zone dont aucune langue ne porte de libellé est écartée avant le nommage,
 * comme `zonesNommees` côté Go — elle ne peut nommer personne, et la garder ferait d'elle la
 * « plus proche » d'un point qu'elle laisserait sans nom. Limite : le Go retombe sur le nom de
 * conception de la zone quand un libellé manque ; la normalisation web ne le garde pas, une zone
 * muette dans toutes les langues est donc écartée ici même si le Go la nommerait par son nom de
 * conception.
 *
 * L'INSTANT EST BRUT : `replayMs` du fil, AVANT `displayClockMs` (qui ne recale que l'affiché).
 * La position vient du résolveur de porteur (un tueur au volant est sur son véhicule). Le z suit
 * la position : celui du véhicule quand elle en vient (null si le véhicule n'a pas d'altitude ou
 * si sa position vient de sa naissance), jamais celui du bipède embarqué ; sinon celui de la
 * piste du bipède, sous la même règle de vie. Un tueur sans position à cet instant n'a pas de
 * zone : la carte n'a pas d'entrée, la ligne n'écrit rien.
 *
 * LES BOTS comptent : un bot du fil porte un xuid de BASE (`bid(N.0)`), que la jointure de
 * `buildPlayers` (gamertag de la feuille, nom nu) relie à ses vies. Il est lu sans véhicule : ses
 * épisodes d'occupation n'ont pas de xuid, comme pour tous les calques du rejeu. Sans feuille
 * pour le joindre, il n'a pas d'entrée.
 *
 * Pas de React, pas de canvas : logique pure, testée (killerZone.test.ts).
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { CalloutZoneReady } from '../../../lib/replay/calloutsPaint'
import { msToFrames, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { buildCarrierAltitudeAt, buildCarrierPosAt } from './carrierPosition'
import type { ReplayFeedEntry } from './killFeedLogic'
import { altitudeOfPlayerAt, deathWindowFrames, posOfPlayerAt } from './livesPosition'

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

/**
 * Règle (b) : plusieurs zones contiennent le point et le z. Le z unique est dans chaque tranche :
 * la plus étroite l'emporte, puis l'indice de volume à égalité parfaite.
 */
function pickStacked(candidates: readonly CalloutZoneReady[]): CalloutZoneReady {
  return [...candidates].sort((a, b) => thickness(a) - thickness(b) || a.volumeIndex - b.volumeIndex)[0]
}

/**
 * Règle (c) : à moins de 2 m, parmi les zones `compatible` d'abord (undefined = aucun premier
 * passage), parmi toutes ensuite ; à égalité de distance, l'indice de volume.
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
 * nameZone rend la zone qui nomme le point (x, y) compte tenu du z connu (`null` = inconnu), ou
 * null : aucune zone ne le nomme.
 */
export function nameZone(
  x: number,
  y: number,
  z: number | null,
  zones: readonly CalloutZoneReady[],
): NamedZone | null {
  const shapes = zones.filter(hasShape)
  if (z === null) return nameWithoutHeight(x, y, shapes)
  const candidates = shapes.filter((c) => inShape(x, y, c) && inSlice(c, z))
  if (candidates.length === 0) return nearest(x, y, shapes, (c) => inSlice(c, z))
  if (candidates.length === 1) return { zone: candidates[0], rule: 'polygone', distanceM: 0 }
  return { zone: pickStacked(candidates), rule: 'empilee', distanceM: 0 }
}

/** Aucune zone : la valeur partagée quand la carte n'en a pas ou que le document est absent. */
const NO_ZONES: ReadonlyMap<string, CalloutZoneReady> = new Map()

/** Une zone nomme-t-elle quelqu'un ? Au moins une langue porte un libellé. */
function hasName(z: CalloutZoneReady): boolean {
  return Object.values(z.names).some((v) => !!v)
}

/**
 * Les vies des bots, par xuid de BASE (`bid(N.0)`) : la jointure de `buildPlayers` relie le bot du
 * film (clé `bot:<nom>`, vies sans xuid) à sa ligne de feuille, dont le xuid est celui du fil.
 */
function botLivesByBaseXuid(
  doc: ReplayDocumentReady,
  scoreboard: readonly MatchScoreboardRow[],
): Map<string, ReplayTrackReady[]> {
  const out = new Map<string, ReplayTrackReady[]>()
  if (scoreboard.length === 0) return out
  for (const p of buildPlayers(doc, [...scoreboard])) {
    if (p.bot && p.board?.xuid && p.lives.length > 0) out.set(p.board.xuid, p.lives)
  }
  return out
}

/**
 * killerZones — la zone du tueur de chaque frag du fil, PAR CLÉ DE LIGNE (`ReplayFeedEntry.key`).
 * Un frag dont le tueur n'a pas de position à son instant, ou que le nommage laisse sans nom, n'a
 * PAS d'entrée : jamais un nom deviné. Les lignes qui ne sont pas des frags n'en ont pas non plus.
 * `scoreboard` joint les bots du fil à leurs vies (cf. l'en-tête).
 */
export function killerZones(
  entries: readonly ReplayFeedEntry[],
  doc: ReplayDocumentReady,
  zones: readonly CalloutZoneReady[],
  scoreboard: readonly MatchScoreboardRow[] = [],
): ReadonlyMap<string, CalloutZoneReady> {
  const named = zones.filter(hasName)
  if (named.length === 0 || doc.tracks.length === 0) return NO_ZONES
  const posOf = buildCarrierPosAt(doc)
  const altOf = buildCarrierAltitudeAt(doc)
  const bots = botLivesByBaseXuid(doc, scoreboard)
  const deathFrames = deathWindowFrames(doc)
  const out = new Map<string, CalloutZoneReady>()
  for (const e of entries) {
    const k = e.kill
    if (!k) continue
    const frame = Math.round(msToFrames(k.replayMs, doc))
    const botLives = bots.get(k.xuid)
    const at = botLives ? posOfPlayerAt(botLives, frame, deathFrames) : posOf(k.xuid, frame)
    if (!at) continue
    const z = botLives ? altitudeOfPlayerAt(botLives, frame, deathFrames) : altOf(k.xuid, frame)
    const zone = nameZone(at.x, at.y, z, named)
    if (zone) out.set(e.key, zone.zone)
  }
  return out
}
