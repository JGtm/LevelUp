/**
 * ObjectiveBalanceCard.compact.test.tsx — « Rapport de force par famille de mode » en vue compacte
 * (tiroir de comparaison de Sessions, maquette `makeBalance` avec `cp`) : une barre par RÔLE et par
 * famille, part entière seule dans les segments, prises nettes de drapeau en ligne à part.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FORMES_TEXT } from '../formes/i18n'
import { block0709 } from './objectif.fixtures'
import { buildObjectiveBalance } from './objectif.logic'
import { ObjectiveBalanceCard } from './ObjectiveBalanceCard'
import { OBJECTIF_TEXT } from './objectifStrings'

function renderCompact() {
  render(
    <ObjectiveBalanceCard
      families={buildObjectiveBalance(block0709())}
      familyLabel={(f) => f}
      columns={FORMES_TEXT.fr.columns}
      t={OBJECTIF_TEXT.fr}
      compact
    />,
  )
}

describe('ObjectiveBalanceCard — compact', () => {
  it('une barre par rôle, nommée par le rôle, sans les actions', () => {
    renderCompact()
    expect(screen.getByTestId('objective-balance-row-zones_strongholds-take').textContent).toContain('Prendre')
    expect(screen.queryByTestId('objective-balance-row-zones_strongholds-zone_captures')).toBeNull()
  })

  it('part entière seule dans les segments (prendre, Bases : 46 % / 54 %)', () => {
    renderCompact()
    const row = screen.getByTestId('objective-balance-row-zones_strongholds-take')
    expect([...row.querySelectorAll('[data-fit-label]')].map((l) => l.textContent)).toEqual(['46 %', '54 %'])
  })

  it('prises nettes de drapeau en ligne à part', () => {
    renderCompact()
    expect(screen.getByTestId('objective-balance-row-ctf-flag_grabs_net')).toBeTruthy()
  })
})
