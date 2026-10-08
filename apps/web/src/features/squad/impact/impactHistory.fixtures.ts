/**
 * impactHistory.fixtures.ts — un `squad_impact_history` de test : trois joueurs, trois soirées
 * dont deux le même jour (chiffres des soirées du 1er et du 7 septembre de la maquette validée).
 */
import type { SquadImpactHistory } from '@/lib/api/types'

const SCALE = [
  { role: 'clutch_finisher', points: 2 },
  { role: 'first_blood', points: 2 },
  { role: 'silent_hero', points: 1.5 },
  { role: 'top_killer', points: 1 },
  { role: 'last_casualty', points: -2 },
  { role: 'false_brother', points: -1.5 },
  { role: 'first_group_death', points: -1 },
  { role: 'last_group_kill', points: -1 },
  { role: 'kamikaze', points: -1 },
  { role: 'thief', points: -1 },
]

const role = (r: string, count: number) => {
  const w = SCALE.find((s) => s.role === r)?.points ?? 0
  return { role: r, count, points: count * w }
}

const player = (name: string, roles: ReturnType<typeof role>[]) => ({
  player: name,
  roles,
  points: roles.reduce((a, r) => a + r.points, 0),
})

export const impactHistory3: SquadImpactHistory = {
  players: ['JGtm', 'Chocoboflor', 'Madina97294'],
  scale: SCALE,
  evenings: [
    {
      session_label: 'S200',
      start_time: '2026-09-01T12:23:00Z',
      matches: 11,
      wins: 5,
      players: [
        player('JGtm', [role('clutch_finisher', 1), role('top_killer', 1), role('last_casualty', 2), role('false_brother', 2),
          role('first_group_death', 1), role('last_group_kill', 2), role('kamikaze', 1), role('thief', 1)]),
        player('Chocoboflor', [role('clutch_finisher', 3), role('first_blood', 1), role('silent_hero', 3),
          role('first_group_death', 1), role('last_group_kill', 3), role('kamikaze', 4), role('thief', 1)]),
        player('Madina97294', [role('clutch_finisher', 1), role('first_blood', 1), role('silent_hero', 1), role('top_killer', 8),
          role('first_group_death', 3), role('last_group_kill', 3), role('kamikaze', 4), role('thief', 2)]),
      ],
    },
    {
      session_label: 'S201',
      start_time: '2026-09-07T12:26:00Z',
      matches: 7,
      wins: 1,
      players: [
        player('JGtm', [role('first_blood', 1), role('top_killer', 1), role('last_casualty', 1), role('last_group_kill', 1), role('kamikaze', 2)]),
        player('Chocoboflor', [role('first_blood', 3), role('false_brother', 2), role('first_group_death', 4),
          role('last_group_kill', 2), role('kamikaze', 1), role('thief', 1)]),
        player('Madina97294', [role('top_killer', 6), role('last_casualty', 3), role('first_group_death', 2), role('kamikaze', 2)]),
      ],
    },
    {
      session_label: 'S202',
      start_time: '2026-09-07T13:24:00Z',
      matches: 1,
      wins: 0,
      players: [
        player('JGtm', []),
        player('Chocoboflor', [role('last_casualty', 1), role('first_group_death', 1), role('thief', 1)]),
        player('Madina97294', [role('first_blood', 1), role('top_killer', 1)]),
      ],
    },
  ],
}
