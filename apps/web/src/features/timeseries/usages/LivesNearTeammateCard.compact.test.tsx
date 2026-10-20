/**
 * LivesNearTeammateCard.compact.test.tsx — « Mes vies : près d'un coéquipier ou seul » en vue
 * compacte (tiroir de comparaison de Sessions, maquette `makeLife` avec `cp`, décision D16) : les
 * parts entières seules dans la barre épaisse et sur la ligne des frags (comptes au survol).
 * Chiffres : 1 558 / 301 vies, 1 250 / 262 frags.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { lives } from './usages.fixtures'
import { buildLivesModel } from './usages.logic'
import { LivesNearTeammateCard } from './LivesNearTeammateCard'
import { USAGES_TEXT } from './usagesText'

const COMPACT = {
  killsLine: (pct: string, perLife: string) => `frags : ${pct} · ${perLife} par vie`,
  killsLineAlone: (perLife: string) => `${perLife} par vie`,
}

describe('LivesNearTeammateCard — compact', () => {
  it('parts entières dans la barre épaisse, sans les comptes', () => {
    render(<LivesNearTeammateCard model={buildLivesModel(lives())!} player="JGtm" ut={USAGES_TEXT.fr.cards} compact={COMPACT} />)
    const labels = [...document.querySelectorAll<HTMLElement>('[data-fit-label]')].map((l) => l.textContent)
    expect(labels).toEqual(['84 %', '16 %'])
  })

  it('ligne des frags en parts et frags par vie', () => {
    render(<LivesNearTeammateCard model={buildLivesModel(lives())!} player="JGtm" ut={USAGES_TEXT.fr.cards} compact={COMPACT} />)
    expect(screen.getByTestId('usages-lives-kills-line').textContent).toBe('frags : 83 % · 0,8 par vie0,9 par vie')
  })
})
