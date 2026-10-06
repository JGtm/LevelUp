/**
 * Garde-rail D7 (plan PLAN_SESSIONS_EMPRISE_2026-10-06, S4.15) — LA PAGE SESSIONS NE LIT QUE LA
 * PARTIE OBJECTIF DE `formes_retenues`.
 *
 * Sessions reçoit le bloc des formes réduit par L7 (`available`, `unavailable_reason`,
 * `matches_total`, `matches_measured`, `main_xuid`, `squad`, `matches[].{match_id, start_time,
 * mode_label, map_label, player_team, objective}`) pour ses deux cartes d'objectif. Les champs que
 * L7 a retirés (lobby, armes, socles, durées, effectifs, mesure par joueur) ne doivent pas revenir
 * par la page : une lecture de l'un d'eux, sur un objet de formes, fait rougir ce test.
 */
import { describe, expect, it } from 'vitest'

const sources = import.meta.glob(['/src/features/session-detail/*.ts', '/src/features/session-detail/*.tsx'], {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/** Un accès à un champ retiré, au bout d'une chaîne partie d'un objet de formes ou d'objectif. */
const FORBIDDEN_ACCESS =
  /\b(?:formes|formes_retenues|compare_formes_retenues|objective)\b[\w?.![\]]*?\.(?:lobby|weapons|weapon_pads|pad_named|pad_unnamed|duration_seconds|team_size|lobby_size|measured)\b/

function offenders(files: Record<string, string>): string[] {
  return Object.entries(files)
    .filter(([path]) => !/\.test\.tsx?$/.test(path) && !/\.fixtures\.ts$/.test(path))
    .filter(([, code]) => FORBIDDEN_ACCESS.test(code))
    .map(([path]) => path)
}

describe('garde-rail D7 — Sessions ne lit que la partie objectif des formes', () => {
  it('le glob scanne bien la page', () => {
    expect(Object.keys(sources).length).toBeGreaterThan(30)
  })

  it('auto-test : le motif reconnaît une lecture de champ retiré, pas un champ gardé', () => {
    expect(offenders({ 'a.ts': 'const x = col.formes?.lobby' })).toEqual(['a.ts'])
    expect(offenders({ 'b.ts': 'm.objective.players[0].measured' })).toEqual(['b.ts'])
    expect(offenders({ 'c.ts': 'data.formes_retenues?.matches?.[0]?.weapons' })).toEqual(['c.ts'])
    expect(offenders({ 'd.ts': 'formes.matches_measured + formes.matches_total' })).toEqual([])
    expect(offenders({ 'e.ts': 'm.objective?.columns' })).toEqual([])
  })

  it('aucun fichier de la page ne lit un champ retiré des formes', () => {
    expect(
      offenders(sources),
      'Champ de formes hors de la liste de L7 lu par la page Sessions : seule la partie objectif est servie (D7).',
    ).toEqual([])
  })
})
