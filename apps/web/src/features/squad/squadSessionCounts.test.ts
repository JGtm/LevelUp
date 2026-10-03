/**
 * Tests squadSessionCounts — source unique du compte de matchs d'une session
 * escouade (ADR 0033, chantier A2).
 *
 * ÉCHEC ATTENDU (TDD rouge) : le module `./squadSessionCounts` n'existe pas
 * encore — échec de RÉSOLUTION DE MODULE, pas d'assertion.
 */
import { describe, expect, it } from 'vitest'
import { squadSessionCount, squadSessionShownCount } from './squadSessionCounts'

describe('squadSessionCount', () => {
  const fallback = [
    { label: 'S1 (11)', match_count_filtered: 8 },
    { label: 'S2 (4)', match_count_filtered: 4 },
  ]

  it('la réponse teammates est là (match_count_roster > match_count) : rend {shown, total} distincts', () => {
    const got = squadSessionCount(
      'S1 (11)',
      [{ label: 'S1 (11)', match_count: 4, match_count_roster: 7 }],
      fallback,
    )
    expect(got).toEqual({ shown: 4, total: 7, excluded: [] })
  })

  it('composition exacte OFF ou aucun écart : match_count_roster absent → total = shown', () => {
    const got = squadSessionCount('S1 (11)', [{ label: 'S1 (11)', match_count: 6 }], fallback)
    expect(got).toEqual({ shown: 6, total: 6, excluded: [] })
  })

  it('sessions non couvertes par composition_sessions : repli sur /filters/resolve', () => {
    const got = squadSessionCount(
      'S2 (4)',
      [{ label: 'S1 (11)', match_count: 6, match_count_roster: 6 }],
      fallback,
    )
    expect(got).toEqual({ shown: 4, total: 4, excluded: [] })
  })

  it('réponse teammates pas encore arrivée (composition_sessions vide) → uniquement le repli', () => {
    const got = squadSessionCount('S1 (11)', [], fallback)
    expect(got).toEqual({ shown: 8, total: 8, excluded: [] })
  })

  it('match_count absent ou nul côté composition → repli (pas de session affichée à 0)', () => {
    expect(squadSessionCount('S1 (11)', [{ label: 'S1 (11)' }], fallback)).toEqual({
      shown: 8,
      total: 8,
      excluded: [],
    })
    expect(squadSessionCount('S2 (4)', [{ label: 'S2 (4)', match_count: 0 }], fallback)).toEqual({
      shown: 4,
      total: 4,
      excluded: [],
    })
  })

  it('ni composition_sessions ni repli pour ce label → undefined', () => {
    expect(squadSessionCount('S9', [], [])).toBeUndefined()
  })

  it('SOURCE UNIQUE : la composition prime même si son nombre est plus BAS que le repli (composition exacte)', () => {
    // Scénario ADR 0033 : rail 7 (repli /filters/resolve, aveugle à la composition)
    // vs page 4 (composition_sessions, population réelle) — la page doit gagner.
    const got = squadSessionCount(
      'S1 (11)',
      [{ label: 'S1 (11)', match_count: 4, match_count_roster: 7 }],
      [{ label: 'S1 (11)', match_count_filtered: 7 }],
    )
    expect(got).toEqual({ shown: 4, total: 7, excluded: [] })
  })

  it('écart publié (D1) : `excluded_by_exact_composition` passe tel quel dans le compte', () => {
    const excluded = [
      { match_id: 'm1', start_time: '2026-08-27T20:00:00Z', map_ui: 'Aquarius', extra_gamertags: ['Nilton410'] },
    ]
    const got = squadSessionCount(
      'S1 (11)',
      [{ label: 'S1 (11)', match_count: 4, match_count_roster: 5, excluded_by_exact_composition: excluded }],
      fallback,
    )
    expect(got).toEqual({ shown: 4, total: 5, excluded })
  })
})

describe('squadSessionShownCount', () => {
  it('extrait uniquement "shown" — consommé par SessionMultiSelect (un seul nombre par ligne)', () => {
    const compositionSessions = [{ label: 'S1 (11)', match_count: 4, match_count_roster: 7 }]
    expect(squadSessionShownCount('S1 (11)', compositionSessions, [])).toBe(4)
  })

  it('undefined quand squadSessionCount ne résout rien', () => {
    expect(squadSessionShownCount('S9', [], [])).toBeUndefined()
  })
})
