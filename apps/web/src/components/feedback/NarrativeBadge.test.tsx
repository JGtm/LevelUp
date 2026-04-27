import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { NarrativeBadge } from './NarrativeBadge'

describe('NarrativeBadge', () => {
  it('rend le label fourni', () => {
    render(<NarrativeBadge label="Domination" />)
    expect(screen.getByTestId('narrative-badge-label').textContent).toBe('Domination')
  })

  it('rend le suffix detail si fourni', () => {
    render(<NarrativeBadge label="Rencontré" detailSuffix="(5×)" />)
    expect(screen.getByTestId('narrative-badge-suffix').textContent).toBe('(5×)')
  })

  it('omet le suffix si non fourni', () => {
    render(<NarrativeBadge label="X" />)
    expect(screen.queryByTestId('narrative-badge-suffix')).toBeNull()
  })

  it('applique la couleur via CSS variable en mode standard', () => {
    render(<NarrativeBadge label="X" colorVar="--narrative-dominance-win-strong" />)
    const badge = screen.getByTestId('narrative-badge')
    // color-mix() est appliqué ; on vérifie juste que la propriété est positionnée
    expect(badge.style.backgroundColor).toContain('color-mix')
    expect(badge.style.color).toContain('var(--narrative-dominance-win-strong)')
    expect(badge.getAttribute('data-inverted')).toBe('false')
  })

  it('mode inverted applique des proportions de mix différentes', () => {
    const { rerender } = render(
      <NarrativeBadge label="X" colorVar="--narrative-role-false-brother" inverted />,
    )
    const inverted = screen.getByTestId('narrative-badge')
    const invertedBg = inverted.style.backgroundColor
    expect(inverted.getAttribute('data-inverted')).toBe('true')

    rerender(
      <NarrativeBadge label="X" colorVar="--narrative-role-false-brother" inverted={false} />,
    )
    const standardBg = screen.getByTestId('narrative-badge').style.backgroundColor
    expect(invertedBg).not.toBe(standardBg)
  })

  it('sans colorVar -> pas de style couleur applique', () => {
    render(<NarrativeBadge label="X" />)
    const badge = screen.getByTestId('narrative-badge')
    expect(badge.style.backgroundColor).toBe('')
    expect(badge.style.color).toBe('')
  })

  it('passe le title (tooltip natif)', () => {
    render(<NarrativeBadge label="X" title="Détail tooltip" />)
    expect(screen.getByTestId('narrative-badge').getAttribute('title')).toBe('Détail tooltip')
  })

  it('size sm vs md applique des classes Tailwind différentes', () => {
    const { rerender } = render(<NarrativeBadge label="X" size="sm" />)
    const sm = screen.getByTestId('narrative-badge').className
    rerender(<NarrativeBadge label="X" size="md" />)
    const md = screen.getByTestId('narrative-badge').className
    expect(sm).not.toBe(md)
  })
})
