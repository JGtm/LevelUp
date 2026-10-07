/**
 * SquadFragBreakdownCard.compact.test.tsx — « Répartition des frags » en vue compacte (tiroir de
 * comparaison de Sessions, maquette `makeFragbar` avec `cp`) : la part entière de chaque classe dans
 * son segment (et au repli), le total en sous-libellé du nom, rien au bout de la barre.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { SquadFragBreakdownCard } from './SquadFragBreakdownCard'
import { getSquadText } from './i18n'

const DATA = { JGtm: [{ class: 'shoulder', kills: 30, authoritative: false }, { class: 'melee', kills: 10, authoritative: false }] }

describe('SquadFragBreakdownCard — compact', () => {
  it('parts entières (75 % / 25 %), total en sous-libellé, pas de total au bout', () => {
    render(
      <SquadFragBreakdownCard
        fragClassesByPlayer={DATA}
        playerOrder={['JGtm']}
        playerColors={{}}
        classLabel={(c) => c}
        emptyTitle="Aucune donnée"
        t={getSquadText('fr')}
        compact={{ totalSub: (n) => `${n} frags`, pctFmt: (v) => `${Math.round(v)} %` }}
      />,
    )
    // jsdom : aucune largeur, tout part au repli — dans l'ordre des segments.
    expect(screen.getByTestId('frag-breakdown-repli-JGtm').textContent).toBe('75 %25 %')
    expect(screen.getByTestId('frag-breakdown-row-JGtm').textContent).toContain('40 frags')
    expect(screen.queryByTestId('frag-breakdown-total-JGtm')).toBeNull()
  })
})
