/**
 * _kdCumul.ts — logique pure des FRAGS CUMULÉS par équipe (`MatchKDCumulChart`).
 *
 * POURQUOI CE FICHIER EXISTE (registre 2026-09-05, N1). Le calcul vivait dans un
 * `useCallback` de 260 lignes à l'intérieur du `.tsx` : au-dessus du seuil de 80 lignes par
 * fonction (CLAUDE.md n° 5), et surtout exécuté sans qu'AUCUNE assertion ne porte sur ce
 * qu'il produit — le seul test qui le montait vérifiait qu'il ne plantait pas. Il est
 * extrait sur le patron de `_scoreCurve.ts` : la géométrie ici, l'habillage ECharts là-bas.
 *
 * CE QU'IL CALCULE, ET CE QU'IL LAISSE AU COMPOSANT. Ici : les deux cumuls, l'étendue des
 * axes, le PLACEMENT anti-collision des pastilles de fait marquant, et les repères de
 * capture. Là-bas : les couleurs (résolues à la palette d'accessibilité au moment du rendu),
 * les libellés d'axe et les infobulles.
 *
 * L'AXE DES TEMPS EST CELUI DU GAMEPLAY, SANS RIEN FAIRE : `event_time_ms` est déjà recalé
 * sur le coup d'envoi par le serveur (`correctMatchViewEventsT0`). C'est la RÉFÉRENCE de
 * l'onglet Chronologie — le bloc « Score dans le temps », qui vient du film, se ramène sur
 * cet axe par `lib/replay/matchClock` (P0-7). Rien n'est corrigé ici : toute correction
 * ajoutée à ces abscisses les décalerait de la référence.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_kdCumul.test.ts`).
 */
import type {
  MatchHighlightEvent,
  MatchImpactBadge,
  MatchObjectiveEvent,
  MatchScoreboardRow,
} from '@/lib/api/types'

import { extractCtfCaptures, type CtfCapture } from './_objectiveCaptures'
import type { XuidMeta } from './xuidMeta'

/** Un point du cumul : l'instant du frag, et le total de l'équipe juste après. */
export interface KdCumulPoint {
  tMs: number
  y: number
}

/** Le camp d'une courbe, du point de vue du joueur de la page. */
export type KdCumulTeam = 'ally' | 'enemy'

/** Ce qu'un fait marquant vaut pour la lecture : un bon coup, ou un mauvais. */
export type KdCumulTone = 'good' | 'bad'

/**
 * Une pastille de fait marquant, PLACÉE : ancrée sur la courbe de son équipe à `yAt`, et
 * décalée à `yChip` pour ne pas en recouvrir une autre (cf. `placeBadges`).
 */
export interface KdCumulBadge {
  key: string
  tMs: number
  team: KdCumulTeam
  tone: KdCumulTone
  /** Ancrage sur la courbe : la valeur du cumul de son équipe à cet instant. */
  yAt: number
  /** Ordonnée de la pastille elle-même, au-dessus (kill) ou en dessous (mort). */
  yChip: number
  /** Ce que la pastille écrit : le pictogramme du fait, puis le gamertag. */
  label: string
}

/** Le modèle complet du graphe, prêt à habiller. */
export interface KdCumul {
  ally: KdCumulPoint[]
  enemy: KdCumulPoint[]
  /** Fin de l'axe des temps, en ms depuis le coup d'envoi (plancher : une minute). */
  totalMs: number
  /** Bornes de l'axe vertical, déjà élargies pour laisser respirer les pastilles. */
  yMin: number
  yMax: number
  badges: KdCumulBadge[]
  /** Repères verticaux des captures de drapeau ; vide hors CTF. */
  captures: CtfCapture[]
}

export interface KdCumulInput {
  events: MatchHighlightEvent[] | null | undefined
  badges: MatchImpactBadge[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /** La cascade « allié = même camp que moi », résolue une fois par l'appelant. */
  xuidMeta: XuidMeta
  objectiveEvents?: MatchObjectiveEvent[] | null
}

/**
 * LES FAITS MARQUANTS QUI SE POSENT SUR LA COURBE, et de quel côté.
 *
 * `kill` : le fait appartient au tueur, la pastille se pose AU-DESSUS de sa courbe.
 * `death` : `player_xuid` est la VICTIME, et le frag revient donc à l'équipe ADVERSE — la
 * pastille se pose sur SA courbe, et EN DESSOUS pour ne jamais heurter celles du dessus.
 * `tone` `auto` : bon si le fait profite au camp du joueur de la page, mauvais sinon.
 *
 * `kamikaze` en est absent à dessein : c'est un badge de match entier, sans instant précis.
 */
const BADGE_MAP: Record<string, { emoji: string; eventKind: 'kill' | 'death'; tone: 'good' | 'bad' | 'auto' }> = {
  first_blood: { emoji: '⚡', eventKind: 'kill', tone: 'auto' },
  top_gun: { emoji: '🔫', eventKind: 'kill', tone: 'good' },
  clutch_finisher: { emoji: '🎯', eventKind: 'kill', tone: 'good' },
  last_group_kill: { emoji: '🐌', eventKind: 'kill', tone: 'bad' },
  first_group_death: { emoji: '🪦', eventKind: 'death', tone: 'bad' },
  last_casualty: { emoji: '💀', eventKind: 'death', tone: 'good' },
}

/** Plancher de l'axe des temps : un match d'une poignée de frags garde une minute d'échelle. */
export const KD_CUMUL_MIN_SPAN_MS = 60_000

/**
 * buildKdCumul projette les faits marquants du match en deux cumuls et leurs pastilles.
 *
 * Rend `null` — donc RIEN À L'ÉCRAN — quand aucun frag n'est attribuable : sans event de
 * type `kill`, ou avec des acteurs qu'aucune ligne de scoreboard ne rattache à un camp, il
 * n'y a pas de courbe à tracer et un cadre titré vide serait une promesse non tenue.
 */
export function buildKdCumul(input: KdCumulInput): KdCumul | null {
  const { events, badges, scoreboard, meXUID, xuidMeta, objectiveEvents } = input
  if (!events || events.length === 0) return null

  const { ally, enemy } = cumulativeCurves(events, xuidMeta)
  if (ally.length === 0 && enemy.length === 0) return null

  const allyCum = ally.length > 0 ? ally[ally.length - 1].y : 0
  const enemyCum = enemy.length > 0 ? enemy[enemy.length - 1].y : 0
  const totalMs = Math.max(
    ally.length > 0 ? ally[ally.length - 1].tMs : 0,
    enemy.length > 0 ? enemy[enemy.length - 1].tMs : 0,
    KD_CUMUL_MIN_SPAN_MS,
  )
  const yMaxData = Math.max(allyCum, enemyCum, 1)
  const placed = placeBadges(badges, xuidMeta, { ally, enemy }, yMaxData, totalMs)
  return {
    ally,
    enemy,
    totalMs,
    ...axisBounds(placed, yMaxData),
    badges: placed.badges,
    captures: extractCtfCaptures(objectiveEvents, scoreboard, meXUID),
  }
}

/**
 * cumulativeCurves compte les frags de chaque camp, dans l'ordre du temps.
 *
 * SEULS LES EVENTS `kill` PEUPLENT LA COURBE, et seulement ceux dont l'acteur est rattaché à
 * un camp : un tueur hors scoreboard ne compte pour personne — l'attribuer par défaut
 * fausserait le meneur.
 */
function cumulativeCurves(
  events: readonly MatchHighlightEvent[],
  xuidMeta: XuidMeta,
): { ally: KdCumulPoint[]; enemy: KdCumulPoint[] } {
  const sorted = events
    .filter(
      (e) =>
        (e.event_type ?? '').toLowerCase() === 'kill' &&
        e.actor_xuid &&
        e.event_time_ms != null,
    )
    .sort((a, b) => (a.event_time_ms as number) - (b.event_time_ms as number))

  const ally: KdCumulPoint[] = []
  const enemy: KdCumulPoint[] = []
  for (const ev of sorted) {
    const meta = xuidMeta.get(ev.actor_xuid as string)
    if (!meta) continue
    const courbe = meta.ally ? ally : enemy
    courbe.push({ tMs: ev.event_time_ms as number, y: courbe.length + 1 })
  }
  return { ally, enemy }
}

/** valueAtMs — la valeur du cumul à un instant : le dernier point déjà passé, 0 avant. */
export function valueAtMs(curve: readonly KdCumulPoint[], tMs: number): number {
  let last = 0
  for (const p of curve) {
    if (p.tMs > tMs) break
    last = p.y
  }
  return last
}

/** Les pastilles placées, et les deux couloirs qui ont servi à les écarter. */
interface PlacedBadges {
  badges: KdCumulBadge[]
  above: number[]
  below: number[]
  chipHeightY: number
}

/**
 * placeBadges pose chaque pastille sur la courbe de son camp, puis l'ÉCARTE de ses voisines.
 *
 * LE DÉCALAGE EST EN UNITÉS DE L'AXE, pas en pixels : la hauteur d'une pastille est une
 * fraction de l'amplitude du graphe, et sa fenêtre de voisinage une fraction de sa durée.
 * Deux couloirs indépendants — au-dessus (les frags) et en dessous (les morts) — parce
 * qu'ils ne peuvent pas se recouvrir l'un l'autre.
 *
 * LA BOUCLE EST BORNÉE (12 essais) : sur un match où douze faits tomberaient au même instant,
 * mieux vaut deux pastilles superposées qu'une boucle qui ne rend pas la main.
 */
function placeBadges(
  badges: MatchImpactBadge[] | null | undefined,
  xuidMeta: XuidMeta,
  curves: { ally: KdCumulPoint[]; enemy: KdCumulPoint[] },
  yMaxData: number,
  totalMs: number,
): PlacedBadges {
  const baseOffset = Math.max(2.5, yMaxData * 0.18)
  const chipHeightY = baseOffset * 1.05
  const chipTimeWindow = totalMs * 0.06
  const above: Array<{ tMs: number; yChip: number }> = []
  const below: Array<{ tMs: number; yChip: number }> = []
  const out: KdCumulBadge[] = []

  const dates = (badges ?? [])
    .filter((b) => b.time_ms != null && BADGE_MAP[b.key])
    .map((b) => ({ b, spec: BADGE_MAP[b.key], tMs: b.time_ms as number }))
    .sort((a, b) => a.tMs - b.tMs)

  for (const item of dates) {
    const meta = item.b.player_xuid ? xuidMeta.get(item.b.player_xuid) : undefined
    if (!meta) continue
    const team: KdCumulTeam =
      item.spec.eventKind === 'kill'
        ? meta.ally
          ? 'ally'
          : 'enemy'
        : meta.ally
          ? 'enemy'
          : 'ally'
    const yAt = valueAtMs(curves[team], item.tMs)
    const placeAbove = item.spec.eventKind === 'kill'
    const couloir = placeAbove ? above : below
    let yChip = placeAbove ? yAt + baseOffset : yAt - baseOffset
    let safety = 12
    while (
      safety-- > 0 &&
      couloir.some(
        (p) =>
          Math.abs(p.tMs - item.tMs) < chipTimeWindow && Math.abs(p.yChip - yChip) < chipHeightY,
      )
    ) {
      yChip += placeAbove ? chipHeightY : -chipHeightY
    }
    couloir.push({ tMs: item.tMs, yChip })
    out.push({
      key: item.b.key,
      tMs: item.tMs,
      team,
      tone: item.spec.tone === 'auto' ? (team === 'ally' ? 'good' : 'bad') : item.spec.tone,
      yAt,
      yChip,
      label: `${item.spec.emoji} ${meta.gamertag}`,
    })
  }
  return {
    badges: out,
    above: above.map((p) => p.yChip),
    below: below.map((p) => p.yChip),
    chipHeightY,
  }
}

/**
 * axisBounds élargit l'axe vertical de quoi laisser respirer les pastilles : le haut monte
 * au-dessus de la plus haute, le bas ne descend sous zéro que si une pastille l'exige.
 */
function axisBounds(placed: PlacedBadges, yMaxData: number): { yMin: number; yMax: number } {
  const yMax = Math.max(
    yMaxData * 1.3,
    ...placed.above.map((y) => y + placed.chipHeightY),
    yMaxData + 1,
  )
  const yMin =
    placed.below.length > 0 ? Math.min(0, ...placed.below.map((y) => y - placed.chipHeightY)) : 0
  return { yMin: Math.floor(yMin), yMax: Math.ceil(yMax) }
}
