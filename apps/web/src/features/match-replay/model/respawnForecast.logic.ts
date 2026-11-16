/**
 * respawnForecast.logic.ts — LA PROBABILITÉ DE RÉAPPARITION d'un joueur mort, lue à un instant du
 * rejeu. Logique PURE : la normalisation de la réponse du serveur, l'échantillon courant d'une mort,
 * les trois premières zones, le rang du point réel et la fin du dessin. Le dessin vit dans
 * `layers/respawnForecast*.ts`, le câblage dans `layers/useReplayRespawnForecast.ts`.
 *
 * LA PROBABILITÉ APPARTIENT AU JOUEUR MORT : chaque mort porte le xuid de son joueur, et le
 * prédicat `shownAuthor` (Seul / Masquer, `focusFilter.author`) la montre ou la retire avec lui.
 *
 * TROIS TEMPS PAR MORT, lus sur l'image affichée :
 *  - avant `deathFrame` : rien ;
 *  - de `deathFrame` à `respawnFrame` : le COMPTE À REBOURS, avec l'échantillon d'une seconde le
 *    plus récent (la réponse en porte un par seconde de compte à rebours, les zones de la plus à la
 *    moins probable) ;
 *  - de `respawnFrame` à `respawnFrame + REVEAL_HOLD_MS` : le POINT RÉEL, avec le dernier
 *    échantillon, puis tout s'efface (`REVEAL_FADE_MS` de fondu à la fin).
 *
 * UNE ZONE ABSENTE D'UN ÉCHANTILLON N'A PAS DE PROBABILITÉ À MONTRER : le serveur ne publie pas une
 * zone dont la probabilité arrondie est nulle ; un point réel hors de ces zones n'a donc pas de rang.
 */
import type { ReplaySpawnForecast, ReplaySpawnForecastSample } from '@/lib/api/types'

import type { XY } from '../../../lib/replay/replayLogic'

/**
 * CE QUE LE CALQUE DESSINE (réglage « Dessin » du bouton « Vue ») : les zones teintées ET les points pesés par-dessus
 * (défaut, maquette validée le 2026-11-16), les zones seules (la teinte porte alors la lecture) ou les points seuls.
 */
export const FORECAST_DRAWS = ['both', 'zones', 'points'] as const
export type ForecastDraw = (typeof FORECAST_DRAWS)[number]
export const DEFAULT_FORECAST_DRAW: ForecastDraw = 'both'

/** Combien de zones portent une étiquette pendant le compte à rebours. */
export const FORECAST_TOP_ZONES = 3
/** Durée d'affichage du point réel après la réapparition, puis tout s'efface. */
export const REVEAL_HOLD_MS = 3000
/** Part de cette durée où le dessin s'éteint en fondu. */
export const REVEAL_FADE_MS = 600

export interface ForecastZoneChance {
  /** `volume_index` d'une zone nommée du document (`/replay/callouts`). */
  zone: number
  /** Probabilité entre 0 et 1. */
  p: number
}

export interface ForecastSample {
  frame: number
  /** De la plus à la moins probable, probabilités strictement positives. */
  zones: readonly ForecastZoneChance[]
  /** Probabilité de chaque point candidat, alignée sur `SpawnForecastReady.points` ; vide si la réponse n'en porte pas d'alignée. */
  pointsP: readonly number[]
}

/** Un point candidat de la carte (mètres monde) et la zone qui le nomme (`volume_index`), `null` si aucune zone servie. */
export interface ForecastPoint extends XY {
  zone: number | null
}

export interface ForecastDeath {
  xuid: string
  deathFrame: number
  respawnFrame: number
  death: XY
  actual: XY
  /** `volume_index` de la zone qui nomme le point réel ; `null` quand aucune zone ne le nomme. */
  actualZone: number | null
  /** Par image croissante, au moins un. */
  samples: readonly ForecastSample[]
}

export interface ForecastZoneName {
  volumeIndex: number
  name: string
}

/** La réponse du serveur, nullabilité résolue : il y a au moins une mort à montrer. */
export interface SpawnForecastReady {
  zones: readonly ForecastZoneName[]
  /** Les points candidats, servis une fois pour tous les échantillons. */
  points: readonly ForecastPoint[]
  /** Par image de mort croissante. */
  deaths: readonly ForecastDeath[]
}

export type ForecastPhase = 'countdown' | 'reveal'

/** Une mort, vue à l'image affichée. */
export interface ForecastMoment {
  death: ForecastDeath
  phase: ForecastPhase
  sample: ForecastSample
  /** Les points candidats (ceux de la réponse) : `sample.pointsP` en donne la probabilité. */
  points: readonly ForecastPoint[]
  /** Les trois premières zones de l'échantillon. */
  top: readonly ForecastZoneChance[]
  /** Probabilité de la première zone (0 sans zone). */
  peak: number
  /** Compte à rebours : temps restant avant la réapparition, en ms (0 au point réel). */
  remainingMs: number
  /** Compte à rebours : part restante du compte à rebours, de 1 à 0 (0 au point réel). */
  fraction: number
  /** Point réel : rang de sa zone dans l'échantillon (1 = première) ; `null` hors des zones probables. */
  rank: number | null
  /** Point réel : probabilité de sa zone (0 hors des zones probables). */
  actualP: number
  /** Opacité du dessin : 1 pendant le compte à rebours, fondu en fin de point réel. */
  opacity: number
}

/**
 * normalizeSpawnForecast résout la nullabilité du transport UNE fois. `null` = aucun calque : réponse
 * absente, statut `unavailable`, ou aucune mort exploitable. Une mort sans échantillon, sans durée ou
 * sans coordonnées finies est écartée (rien n'est deviné).
 */
export function normalizeSpawnForecast(res: ReplaySpawnForecast | null | undefined): SpawnForecastReady | null {
  if (!res || res.status !== 'ready') return null
  const zones = (res.zones ?? []).map((z) => ({ volumeIndex: z.volume_index, name: z.name }))
  const points = normalizePoints(res.points, zones)
  const deaths: ForecastDeath[] = []
  for (const d of res.deaths ?? []) {
    const samples = normalizeSamples(d.samples, points.length)
    if (samples.length === 0 || !(d.respawn_frame > d.death_frame)) continue
    if (![d.death.x, d.death.y, d.actual.x, d.actual.y].every(Number.isFinite)) continue
    deaths.push({
      xuid: d.xuid,
      deathFrame: d.death_frame,
      respawnFrame: d.respawn_frame,
      death: { x: d.death.x, y: d.death.y },
      actual: { x: d.actual.x, y: d.actual.y },
      actualZone: d.actual_zone ?? null,
      samples,
    })
  }
  if (deaths.length === 0) return null
  deaths.sort((a, b) => a.deathFrame - b.deathFrame)
  return { zones, points, deaths }
}

/** Les points candidats : `zone_index` (indice dans `zones[]`, -1 si aucun) devient le `volume_index` de la zone. */
function normalizePoints(raw: ReplaySpawnForecast['points'], zones: readonly ForecastZoneName[]): ForecastPoint[] {
  // Aucun point n'est écarté : `points_p` est aligné sur eux, un décalage fausserait chaque disque.
  return (raw ?? []).map((p) => ({ x: p.x, y: p.y, zone: zones[p.zone_index]?.volumeIndex ?? null }))
}

/**
 * Les échantillons d'une mort. Une liste de probabilités de points qui n'est pas alignée sur les points
 * (longueur différente) ne se lit pas : elle est écartée, jamais devinée.
 */
function normalizeSamples(raw: readonly ReplaySpawnForecastSample[] | null, pointCount: number): ForecastSample[] {
  const out: ForecastSample[] = []
  for (const s of raw ?? []) {
    const zones = (s.zones ?? [])
      .filter((z) => Number.isFinite(z.p) && z.p > 0)
      .map((z) => ({ zone: z.zone, p: z.p }))
      .sort((a, b) => b.p - a.p || a.zone - b.zone)
    const pointsP = (s.points_p ?? []).length === pointCount ? [...(s.points_p ?? [])] : []
    out.push({ frame: s.frame, zones, pointsP })
  }
  return out.sort((a, b) => a.frame - b.frame)
}

/** L'échantillon d'une mort à une image : le plus récent qui ne la dépasse pas, le premier avant eux. */
export function sampleAt(death: ForecastDeath, frame: number): ForecastSample {
  let found = death.samples[0]
  for (const s of death.samples) {
    if (s.frame > frame) break
    found = s
  }
  return found
}

/** Le rang (1 = première) d'une zone dans un échantillon ; `null` si l'échantillon ne la porte pas. */
export function rankOfZone(sample: ForecastSample, zone: number | null): number | null {
  if (zone === null) return null
  const i = sample.zones.findIndex((z) => z.zone === zone)
  return i < 0 ? null : i + 1
}

/** Le dernier échantillon du compte à rebours : celui qui précède la réapparition. */
function lastSample(death: ForecastDeath): ForecastSample {
  return death.samples[death.samples.length - 1]
}

/** L'opacité d'un point réel `ageMs` après la réapparition : pleine, puis fondu ; 0 une fois effacé. */
export function revealOpacity(ageMs: number): number {
  if (ageMs < 0 || ageMs >= REVEAL_HOLD_MS) return 0
  const fadeStart = REVEAL_HOLD_MS - REVEAL_FADE_MS
  return ageMs <= fadeStart ? 1 : (REVEAL_HOLD_MS - ageMs) / REVEAL_FADE_MS
}

/**
 * momentOf lit UNE mort à une image : `null` avant la mort ou après l'effacement. `frameMs` est la
 * durée d'une image du document.
 */
export function momentOf(
  death: ForecastDeath,
  frame: number,
  frameMs: number,
  points: readonly ForecastPoint[] = NO_POINTS,
): ForecastMoment | null {
  if (frame < death.deathFrame) return null
  if (frame < death.respawnFrame) {
    const sample = sampleAt(death, frame)
    const span = death.respawnFrame - death.deathFrame
    return {
      death, phase: 'countdown', sample, points,
      top: sample.zones.slice(0, FORECAST_TOP_ZONES),
      peak: sample.zones[0]?.p ?? 0,
      remainingMs: (death.respawnFrame - frame) * frameMs,
      fraction: (death.respawnFrame - frame) / span,
      rank: null, actualP: 0, opacity: 1,
    }
  }
  const opacity = revealOpacity((frame - death.respawnFrame) * frameMs)
  if (opacity <= 0) return null
  const sample = lastSample(death)
  const rank = rankOfZone(sample, death.actualZone)
  return {
    death, phase: 'reveal', sample, points,
    top: sample.zones.slice(0, FORECAST_TOP_ZONES),
    peak: sample.zones[0]?.p ?? 0,
    remainingMs: 0, fraction: 0,
    rank,
    actualP: rank === null ? 0 : sample.zones[rank - 1].p,
    opacity,
  }
}

/**
 * momentsAt rend les morts actives à une image — plusieurs joueurs peuvent être morts ensemble —,
 * celles que Seul / Masquer retire écartées (`shownAuthor` : la probabilité est au joueur mort).
 */
export function momentsAt(
  forecast: SpawnForecastReady,
  frame: number,
  frameMs: number,
  shownAuthor?: (xuid: string) => boolean,
): ForecastMoment[] {
  const out: ForecastMoment[] = []
  for (const death of forecast.deaths) {
    if (shownAuthor && !shownAuthor(death.xuid)) continue
    const m = momentOf(death, frame, frameMs, forecast.points)
    if (m) out.push(m)
  }
  return out
}

/**
 * primaryMoment choisit la mort que la légende et les étiquettes racontent : la plus récente (le
 * compte à rebours le plus frais), à égalité celle du xuid le plus petit — un choix stable d'une
 * image à l'autre.
 */
export function primaryMoment(moments: readonly ForecastMoment[]): ForecastMoment | null {
  let best: ForecastMoment | null = null
  for (const m of moments) {
    if (!best || m.death.deathFrame > best.death.deathFrame || (m.death.deathFrame === best.death.deathFrame && m.death.xuid < best.death.xuid)) {
      best = m
    }
  }
  return best
}

const NO_POINTS: readonly ForecastPoint[] = []

/**
 * L'opacité de remplissage d'une zone teintée : un fond léger qui monte avec sa part de la première zone. Sous
 * les points pesés (« Zones et points »), la teinte reste plus légère : ce sont les disques qui portent la lecture.
 */
export const TINT_BASE_ALPHA = 0.05
export const TINT_AMPLITUDE_ALPHA = 0.55
export const TINT_BASE_ALPHA_UNDER_POINTS = 0.03
export const TINT_AMPLITUDE_ALPHA_UNDER_POINTS = 0.3

export function tintAlpha(p: number, peak: number, draw: ForecastDraw = 'zones'): number {
  const under = draw === 'both'
  const base = under ? TINT_BASE_ALPHA_UNDER_POINTS : TINT_BASE_ALPHA
  const amplitude = under ? TINT_AMPLITUDE_ALPHA_UNDER_POINTS : TINT_AMPLITUDE_ALPHA
  return base + (peak > 0 ? amplitude * Math.min(1, p / peak) : 0)
}

/** Le disque d'un point pesé : son rayon (pixels d'écran) suit la racine de sa probabilité, comme la maquette. */
export const DISC_BASE_RADIUS_PX = 2.5
export const DISC_SPAN_RADIUS_PX = 21
/** En dessous, un point est trop improbable pour un disque : il ne se dessine pas. */
export const DISC_MIN_P = 0.001

export function discRadiusPx(p: number): number {
  return DISC_BASE_RADIUS_PX + DISC_SPAN_RADIUS_PX * Math.sqrt(Math.max(0, p))
}

/** Les trois repères de l'échelle des disques de la légende, en probabilités. */
export const DISC_SCALE_P = [0.05, 0.2, 0.45] as const
