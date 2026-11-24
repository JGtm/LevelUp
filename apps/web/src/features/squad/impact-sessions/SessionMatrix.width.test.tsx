import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { SquadImpactSessions } from '@/lib/api/types'

import { getSquadText } from '../i18n'
import { impactSessionsFixture } from './impactSessions.fixtures'
import {
  buildImpactSessionsView,
  MATCH_COL_MIN_PX,
  matrixMinWidthPx,
} from './impactSessions.logic'
import { SessionMatrix } from './SessionMatrix'
import { IMPACT_SESSIONS_TEXT } from './impactSessionsText'

/** La session la plus fournie de la fixture, étirée à `n` matchs (les joueurs n'y ont aucun rôle de plus). */
function sessionDeMatchs(n: number): SquadImpactSessions {
  const base = impactSessionsFixture.sessions?.[0]
  if (!base) throw new Error('fixture sans session')
  const matches = Array.from({ length: n }, (_, i) => ({ match_id: `m-${i + 1}`, outcome: i % 2 === 0 ? 2 : 3 }))
  return {
    ...impactSessionsFixture,
    sessions: [{ ...base, matches, players: (base.players ?? []).map((p) => ({ ...p, per_match: [] })) }],
  }
}

function rendre(n: number) {
  const view = buildImpactSessionsView(sessionDeMatchs(n))!
  const row = view.rows.find((r) => r.matches.length === n)!
  render(
    <SessionMatrix
      row={row}
      view={view}
      impact={getSquadText('fr').impact}
      text={IMPACT_SESSIONS_TEXT.fr}
      locale="fr"
      inkOf={() => 'var(--ac-muted)'}
      ariaLabel="matrice"
    />,
  )
  return { table: screen.getByRole('table', { name: 'matrice' }), roles: view.roleOrder.length }
}

describe('matrixMinWidthPx', () => {
  it('garde le plancher d’une session courte et grandit avec les matchs d’une longue', () => {
    expect(matrixMinWidthPx(2, 11)).toBe(900)
    expect(matrixMinWidthPx(20, 11)).toBeGreaterThan(matrixMinWidthPx(10, 11))
    expect(matrixMinWidthPx(20, 11)).toBeGreaterThanOrEqual(20 * MATCH_COL_MIN_PX + 11 * 40)
  })
})

describe('SessionMatrix — largeur des colonnes de match', () => {
  it('une session de 20 matchs laisse à chaque colonne de match sa largeur minimale (défilement du tableau)', () => {
    const { table, roles } = rendre(20)
    expect(parseInt(table.style.minWidth, 10)).toBe(matrixMinWidthPx(20, roles))
    expect(parseInt(table.style.minWidth, 10)).toBeGreaterThanOrEqual(20 * MATCH_COL_MIN_PX)
  })
})
