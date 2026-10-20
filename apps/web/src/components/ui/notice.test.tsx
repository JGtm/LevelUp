/**
 * Notice — gabarit unique des messages de page : un ton, une icône, role="status".
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { Notice, NoticeLiveStatus, type NoticeTone } from './notice'

const TONES: NoticeTone[] = ['error', 'warning', 'info']

describe('Notice', () => {
  it.each(TONES)('ton %s : role=status, une icône décorative, titre et corps', (tone) => {
    render(
      <Notice tone={tone} testId="n" title="Titre">
        Corps
      </Notice>,
    )
    const notice = screen.getByTestId('n')
    expect(notice).toHaveAttribute('role', 'status')
    expect(notice).toHaveAttribute('data-tone', tone)
    expect(notice.querySelectorAll('svg')).toHaveLength(1)
    expect(notice.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
    expect(notice).toHaveTextContent('Titre')
    expect(notice).toHaveTextContent('Corps')
  })

  it('chaque ton a sa teinte sémantique et son icône propre', () => {
    const paths = new Set<string | null>()
    const teintes = { error: 'destructive', warning: 'warning', info: 'info' } as const
    for (const tone of TONES) {
      const { unmount } = render(<Notice tone={tone} testId="n" title="t" />)
      const notice = screen.getByTestId('n')
      expect(notice.className).toContain(`border-${teintes[tone]}`)
      expect(notice.className).toContain(`bg-${teintes[tone]}`)
      paths.add(notice.querySelector('path')!.getAttribute('d'))
      unmount()
    }
    expect(paths.size).toBe(3)
  })

  it('sans corps : pas de bloc de corps ; la mention latérale est rendue', () => {
    render(<Notice tone="info" testId="n" title="t" aside={<span>à droite</span>} />)
    const notice = screen.getByTestId('n')
    expect(notice.querySelector('.text-muted-foreground')).toBeNull()
    expect(notice).toHaveTextContent('à droite')
  })
})

describe('NoticeLiveStatus', () => {
  it('le point ne pulse que hors prefers-reduced-motion', () => {
    render(<NoticeLiveStatus tone="info">Mise à jour automatique</NoticeLiveStatus>)
    expect(screen.getByText('Mise à jour automatique')).toBeInTheDocument()
    const animated = document.querySelectorAll('[class*="animate-"]')
    expect(animated.length).toBeGreaterThan(0)
    for (const el of Array.from(animated)) {
      for (const cls of el.className.split(/\s+/).filter((c) => c.includes('animate-'))) {
        expect(cls.startsWith('motion-safe:')).toBe(true)
      }
    }
  })
})
