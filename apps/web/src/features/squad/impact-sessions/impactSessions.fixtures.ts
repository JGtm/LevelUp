/**
 * impactSessions.fixtures.ts — un `squad_impact_sessions` de test : trois joueurs, quatre sessions
 * (la plus récente d'abord), des rôles variés, un joueur absent d'une session entière et un joueur
 * absent d'un match. Le serveur calcule les comptes, les points et les totaux : le constructeur
 * ci-dessous les déduit des rôles par match, avec le même barème et le même ordre de colonnes que
 * `squadimpact`, pour que la fixture reste cohérente avec ce que le serveur publie.
 */
import type { SquadImpactSessions } from '@/lib/api/types'

export const FIXTURE_PLAYERS = ['JGtm', 'joueur_117', 'joueur_165'] as const

/** L'ordre des colonnes de rôles (`squadimpact.RoleOrder`). */
export const FIXTURE_ROLE_ORDER = [
  'first_blood',
  'clutch_finisher',
  'last_casualty',
  'last_group_kill',
  'first_group_death',
  'silent_hero',
  'false_brother',
  'kamikaze',
  'thief',
  'glass_jaw',
  'top_killer',
]

/** Le barème servi, dans l'ordre d'empilement. */
export const FIXTURE_SCALE = [
  { role: 'clutch_finisher', points: 2 },
  { role: 'first_blood', points: 2 },
  { role: 'silent_hero', points: 1.5 },
  { role: 'top_killer', points: 1 },
  { role: 'last_casualty', points: -2 },
  { role: 'false_brother', points: -1.5 },
  { role: 'thief', points: -1 },
  { role: 'first_group_death', points: -0.5 },
  { role: 'last_group_kill', points: -0.5 },
  { role: 'kamikaze', points: -0.5 },
  { role: 'glass_jaw', points: -0.5 },
]

interface MatchSpec {
  id: string
  /** 2 victoire, 3 défaite, 1 égalité. */
  outcome: number
  /** Les rôles de chaque joueur sur ce match (absent : aucun rôle). */
  roles?: Partial<Record<(typeof FIXTURE_PLAYERS)[number], string[]>>
  /** Les joueurs qui ne jouent pas ce match. */
  absent?: (typeof FIXTURE_PLAYERS)[number][]
}

interface SessionSpec {
  label: string
  start: string
  matches: MatchSpec[]
}

const weightOf = (role: string) => FIXTURE_SCALE.find((s) => s.role === role)?.points ?? 0

const SESSION_SPECS: SessionSpec[] = [
  {
    label: '14/11/2026 20:05–23:10 (5)',
    start: '2026-11-14T19:05:00Z',
    matches: [
      { id: 'm-14-1', outcome: 2, roles: { JGtm: ['first_blood', 'top_killer'], joueur_117: ['last_casualty'], joueur_165: ['first_group_death'] } },
      { id: 'm-14-2', outcome: 2, roles: { JGtm: ['clutch_finisher'], joueur_165: ['silent_hero', 'glass_jaw'] } },
      { id: 'm-14-3', outcome: 3, roles: { joueur_117: ['first_blood'], JGtm: ['last_casualty', 'kamikaze'] }, absent: ['joueur_165'] },
      { id: 'm-14-4', outcome: 2, roles: { joueur_165: ['top_killer', 'clutch_finisher'], joueur_117: ['first_group_death'] } },
      { id: 'm-14-5', outcome: 1 },
    ],
  },
  {
    label: '10/11/2026 21:00–22:30 (3)',
    start: '2026-11-10T20:00:00Z',
    matches: [
      { id: 'm-10-1', outcome: 3, roles: { JGtm: ['top_killer'], joueur_117: ['false_brother'], joueur_165: ['last_casualty', 'thief'] } },
      { id: 'm-10-2', outcome: 3, roles: { joueur_117: ['first_blood'], joueur_165: ['kamikaze'] } },
      { id: 'm-10-3', outcome: 2, roles: { JGtm: ['clutch_finisher', 'silent_hero'], joueur_117: ['last_group_kill'] } },
    ],
  },
  {
    label: '07/11/2026 18:30–19:40 (2)',
    start: '2026-11-07T17:30:00Z',
    matches: [
      { id: 'm-07-1', outcome: 2, roles: { JGtm: ['first_blood'], joueur_117: ['top_killer'] }, absent: ['joueur_165'] },
      { id: 'm-07-2', outcome: 3, roles: { JGtm: ['glass_jaw'], joueur_117: ['clutch_finisher'] }, absent: ['joueur_165'] },
    ],
  },
  {
    label: '31/10/2026 22:00–22:45 (2)',
    start: '2026-10-31T21:00:00Z',
    matches: [
      { id: 'm-31-1', outcome: 2, roles: { joueur_165: ['first_blood', 'clutch_finisher'] }, absent: ['JGtm', 'joueur_117'] },
      { id: 'm-31-2', outcome: 2, roles: { joueur_165: ['top_killer'] }, absent: ['JGtm', 'joueur_117'] },
    ],
  },
]

const sum = (xs: number[]) => xs.reduce((a, b) => a + b, 0)

function buildSessions(specs: SessionSpec[]): SquadImpactSessions {
  const countsOf = (rolesList: string[]) =>
    FIXTURE_ROLE_ORDER.map((role) => ({ role, count: rolesList.filter((r) => r === role).length }))
  const sessions = specs.map((s) => ({
    session_label: s.label,
    start_time: s.start,
    matches: s.matches.map((m) => ({ match_id: m.id, outcome: m.outcome })),
    players: FIXTURE_PLAYERS.map((player) => {
      const played = s.matches.filter((m) => !m.absent?.includes(player))
      const all = played.flatMap((m) => m.roles?.[player] ?? [])
      return {
        player,
        matches_played: played.length,
        points: sum(all.map(weightOf)),
        counts: countsOf(all),
        per_match: s.matches
          .filter((m) => (m.roles?.[player] ?? []).length > 0)
          .map((m) => ({
            match_id: m.id,
            roles: FIXTURE_ROLE_ORDER.flatMap((role) => (m.roles?.[player] ?? []).filter((r) => r === role)),
          })),
      }
    }),
  }))
  const totals = FIXTURE_PLAYERS.map((player, i) => ({
    player,
    matches_played: sum(sessions.map((s) => s.players[i].matches_played)),
    points: sum(sessions.map((s) => s.players[i].points)),
    counts: FIXTURE_ROLE_ORDER.map((role) => ({
      role,
      count: sum(sessions.map((s) => s.players[i].counts.find((c) => c.role === role)?.count ?? 0)),
    })),
  }))
  return {
    players: [...FIXTURE_PLAYERS],
    scale: FIXTURE_SCALE,
    role_order: FIXTURE_ROLE_ORDER,
    sessions,
    totals,
  }
}

export const impactSessionsFixture: SquadImpactSessions = buildSessions(SESSION_SPECS)
