/**
 * impactSessions.logic.ts — la mise en forme de `squad_impact_sessions` pour le tableau « Impact des
 * coéquipiers » (Escouade › Contributions) : lignes, net par match, état ouvert initial, surlignage.
 *
 * Le serveur calcule tout ce qui est compté (rôles, points, matchs joués, totaux) ; l'écran ne fait
 * que de la mise en page : le net par match (points / matchs joués), le bilan victoires-défaites, le
 * meilleur et le pire de chaque colonne. Le sens d'un rôle (un compte plus élevé est pire) se lit sur
 * le barème servi (points < 0), jamais sur une table locale.
 */
import { formatNumber, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { outcomeCodeToValue } from '@/lib/outcome'
import { stripSessionCountSuffix } from '@/lib/sessions/sessionLabels'
import type { SquadImpactSessions } from '@/lib/api/types'

/** Clé de la ligne « Toutes les sessions ». */
export const ALL_SESSIONS_KEY = 'all'

type MatchOutcomeKind = 'win' | 'loss' | 'other'

export interface ImpactMatchView {
  matchId: string
  /** Rang du match dans la session, à partir de 1 (ordre chronologique). */
  number: number
  outcome: MatchOutcomeKind
}

export interface ImpactPlayerView {
  player: string
  matchesPlayed: number
  /** Net de la session : somme du barème sur ses rôles. */
  points: number
  /** Points / matchs joués ; `null` quand le joueur n'a joué aucun match de la ligne. */
  netPerMatch: number | null
  /** Compte de chaque rôle (tous les rôles de l'ordre servi, zéros compris). */
  counts: Record<string, number>
  /** Rôles du joueur par match ; un match absent n'a aucun rôle pour ce joueur. */
  rolesByMatch: Record<string, string[]>
}

export interface ImpactRowView {
  key: string
  kind: 'all' | 'session'
  /** Libellé de session sans son suffixe volatil « (N) » ; vide pour la ligne « Toutes les sessions ». */
  label: string
  matchCount: number
  wins: number
  losses: number
  /** Les matchs de la session ; vide pour la ligne « Toutes les sessions ». */
  matches: ImpactMatchView[]
  /** Une entrée par joueur, dans l'ordre de `players`. */
  players: ImpactPlayerView[]
}

export interface ImpactSessionsView {
  players: string[]
  roleOrder: string[]
  /** Les rôles qui retirent des points : un compte plus élevé y est pire. */
  costlyRoles: ReadonlySet<string>
  /** « Toutes les sessions » d'abord, puis les sessions de la plus récente à la plus ancienne. */
  rows: ImpactRowView[]
}

type ServedSession = NonNullable<SquadImpactSessions['sessions']>[number]
type ServedCounts = { role: string; count: number }[] | null | undefined

function outcomeKind(code: number): MatchOutcomeKind {
  const value = outcomeCodeToValue(code)
  return value === 'win' || value === 'loss' ? value : 'other'
}

/** Le net par match d'un joueur : `null` quand il n'a joué aucun match. */
function netPerMatch(points: number, matchesPlayed: number): number | null {
  return matchesPlayed > 0 ? points / matchesPlayed : null
}

function countsOf(list: ServedCounts, roleOrder: string[]): Record<string, number> {
  const out: Record<string, number> = Object.fromEntries(roleOrder.map((r) => [r, 0]))
  for (const c of list ?? []) out[c.role] = c.count
  return out
}

function playerView(
  player: string,
  served: { matches_played: number; points: number; counts: ServedCounts } | undefined,
  roleOrder: string[],
  rolesByMatch: Record<string, string[]>,
): ImpactPlayerView {
  const matchesPlayed = served?.matches_played ?? 0
  const points = served?.points ?? 0
  return {
    player,
    matchesPlayed,
    points,
    netPerMatch: netPerMatch(points, matchesPlayed),
    counts: countsOf(served?.counts, roleOrder),
    rolesByMatch,
  }
}

function sessionRow(session: ServedSession, players: string[], roleOrder: string[]): ImpactRowView {
  const matches = (session.matches ?? []).map<ImpactMatchView>((m, i) => ({
    matchId: m.match_id,
    number: i + 1,
    outcome: outcomeKind(m.outcome),
  }))
  const byPlayer = new Map((session.players ?? []).map((p) => [p.player, p]))
  const label = stripSessionCountSuffix(session.session_label)
  return {
    key: `s:${label}`,
    kind: 'session',
    label,
    matchCount: matches.length,
    wins: matches.filter((m) => m.outcome === 'win').length,
    losses: matches.filter((m) => m.outcome === 'loss').length,
    matches,
    players: players.map((name) => {
      const p = byPlayer.get(name)
      const rolesByMatch = Object.fromEntries((p?.per_match ?? []).map((m) => [m.match_id, m.roles ?? []]))
      return playerView(name, p, roleOrder, rolesByMatch)
    }),
  }
}

/** La ligne « Toutes les sessions » : somme des sessions affichées, lue sur les totaux servis. */
function allRow(data: SquadImpactSessions, players: string[], roleOrder: string[], sessionRows: ImpactRowView[]): ImpactRowView {
  const byPlayer = new Map((data.totals ?? []).map((t) => [t.player, t]))
  return {
    key: ALL_SESSIONS_KEY,
    kind: 'all',
    label: '',
    matchCount: sessionRows.reduce((n, r) => n + r.matchCount, 0),
    wins: sessionRows.reduce((n, r) => n + r.wins, 0),
    losses: sessionRows.reduce((n, r) => n + r.losses, 0),
    matches: [],
    players: players.map((name) => playerView(name, byPlayer.get(name), roleOrder, {})),
  }
}

/** La vue du tableau ; `null` quand la réponse n'a aucune session à montrer. */
export function buildImpactSessionsView(data: SquadImpactSessions | null | undefined): ImpactSessionsView | null {
  const players = data?.players ?? []
  if (!data || players.length === 0 || (data.sessions ?? []).length === 0) return null
  const roleOrder = data.role_order ?? []
  const sessionRows = (data.sessions ?? []).map((s) => sessionRow(s, players, roleOrder))
  return {
    players,
    roleOrder,
    costlyRoles: new Set((data.scale ?? []).filter((w) => w.points < 0).map((w) => w.role)),
    rows: [allRow(data, players, roleOrder, sessionRows), ...sessionRows],
  }
}

/**
 * Les lignes ouvertes d'office et les lignes teintées. Les sessions sélectionnées dans le filtre de la
 * page s'ouvrent et se teintent ; l'identité d'une session est son libellé sans le suffixe de compte
 * « (N) », qui change à chaque synchronisation (`stripSessionCountSuffix`, comme la réconciliation des
 * sessions pickées). Sans session sélectionnée visible (période, ou sélection hors des sessions
 * affichées), la plus récente s'ouvre, sans teinte.
 */
export function initialOpenState(
  rows: ImpactRowView[],
  pickedLabels: readonly string[],
): { open: string[]; selected: string[] } {
  const picked = new Set(pickedLabels.map(stripSessionCountSuffix))
  const sessions = rows.filter((r) => r.kind === 'session')
  const selected = sessions.filter((r) => picked.has(r.label)).map((r) => r.key)
  if (selected.length > 0) return { open: selected, selected }
  return { open: sessions.length > 0 ? [sessions[0].key] : [], selected: [] }
}

export type CellState = 'best' | 'worst' | 'neutral'

/**
 * Meilleur et pire d'une colonne de rôle : comptes non nuls, au moins deux valeurs distinctes. Pour un
 * rôle qui retire des points, le plus grand compte est le pire.
 */
export function roleCellStates(counts: readonly number[], costly: boolean): CellState[] {
  const positive = counts.filter((c) => c > 0)
  const min = Math.min(...positive)
  const max = Math.max(...positive)
  if (positive.length === 0 || min === max) return counts.map(() => 'neutral')
  return counts.map((c) => {
    if (c <= 0) return 'neutral'
    if (c === max) return costly ? 'worst' : 'best'
    if (c === min) return costly ? 'best' : 'worst'
    return 'neutral'
  })
}

/** Meilleur et pire score d'une ligne, parmi les joueurs qui ont joué (au moins deux valeurs distinctes). */
export function scoreCellStates(players: readonly ImpactPlayerView[]): CellState[] {
  const played = players.filter((p) => p.matchesPlayed > 0).map((p) => p.points)
  const min = Math.min(...played)
  const max = Math.max(...played)
  if (played.length < 2 || min === max) return players.map(() => 'neutral')
  return players.map((p) => {
    if (p.matchesPlayed === 0) return 'neutral'
    if (p.points === max) return 'best'
    if (p.points === min) return 'worst'
    return 'neutral'
  })
}

export type ScoreMark = 'champion' | 'maillon_faible' | 'passager_clandestin' | null

/** La marque du score : couronne au premier, banane au dernier s'il est négatif, passager clandestin sinon. */
export function scoreMark(state: CellState, points: number): ScoreMark {
  if (state === 'best') return 'champion'
  if (state === 'worst') return points < 0 ? 'maillon_faible' : 'passager_clandestin'
  return null
}

/** Un nombre signé à N décimales, à la langue de l'écran : « +1,5 », « −2,0 », « ±0,0 ». */
export function formatSignedNet(value: number, locale: Locale, decimals = 1): string {
  const factor = 10 ** decimals
  const rounded = Math.round(value * factor) / factor
  const abs = formatNumber(Math.abs(rounded), intlLocale(locale), decimals)
  if (rounded === 0) return `±${abs}`
  return rounded > 0 ? `+${abs}` : `−${abs}`
}

/** Le sens d'un net à une décimale, pour la couleur : gain, perte ou nul. */
export function netSign(value: number): 'gain' | 'loss' | 'zero' {
  const rounded = Math.round(value * 10) / 10
  if (rounded > 0) return 'gain'
  return rounded < 0 ? 'loss' : 'zero'
}

/** Le net par match est borné à ±`NET_BAR_RANGE` points pour la mini-barre. */
const NET_BAR_RANGE = 2

/** La longueur de la mini-barre, de -1 à 1. */
export function netBarRatio(value: number): number {
  return Math.max(-NET_BAR_RANGE, Math.min(NET_BAR_RANGE, value)) / NET_BAR_RANGE
}
