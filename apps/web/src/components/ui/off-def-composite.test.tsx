/**
 * Tests unitaires — OffDefComposite (barre composite Rendement / Résistance).
 *
 * Vérifie : fallback "—", valeurs affichées (transfo off*100 / (def-1)*100),
 * proportions des segments (valeurs brutes), alignement.
 */
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'

import { OffDefComposite } from './off-def-composite'

describe('OffDefComposite', () => {
  it('affiche "—" quand OC et DR sont null', () => {
    render(<OffDefComposite offensiveConversion={null} defensiveResistance={null} />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('affiche off*100% et (def-1)*100% (transfo d\'affichage)', () => {
    render(<OffDefComposite offensiveConversion={0.42} defensiveResistance={1.18} />)
    expect(screen.getByText('42%')).toBeInTheDocument()
    expect(screen.getByText('18%')).toBeInTheDocument()
  })

  it('proportion du segment OC = off/(off+def) sur les valeurs brutes', () => {
    const { container } = render(<OffDefComposite offensiveConversion={1} defensiveResistance={1} />)
    const segs = container.querySelectorAll('.h-2 > div')
    expect(segs).toHaveLength(2)
    // off=def=1 → 1/2 = 50%
    expect((segs[0] as HTMLElement).style.width).toBe('50%')
    expect((segs[1] as HTMLElement).style.width).toBe('50%')
  })

  it('align="start" → conteneur des valeurs en justify-start', () => {
    const { container } = render(
      <OffDefComposite offensiveConversion={0.5} defensiveResistance={1.1} align="start" />,
    )
    expect(container.querySelector('.justify-start')).toBeTruthy()
    expect(container.querySelector('.justify-center')).toBeNull()
  })

  it('align par défaut (center) → conteneur des valeurs en justify-center', () => {
    const { container } = render(
      <OffDefComposite offensiveConversion={0.5} defensiveResistance={1.1} />,
    )
    expect(container.querySelector('.justify-center')).toBeTruthy()
  })
})
