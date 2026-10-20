/**
 * UsageForms.test.tsx — LES PROMESSES DE LA GRILLE DE JAUGES (carte « Appui reçu » de Sessions) :
 *
 *   - les colonnes sont celles que l'appelant nomme, une jauge par colonne, un axe gradué chacune ;
 *   - le compte brut n'est PAS dans une cellule, il est dans l'infobulle du rail (D2) : le
 *     pourcentage seul est ce qui se lit ;
 *   - une part sans dénominateur rend une jauge vide au tiret, jamais un 0 % ; le trait se pose
 *     seulement quand il existe.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { UsageGaugeGrid } from './UsageForms'
import type { UsageGaugeRowModel } from './usageGaugeModel'

const COLUMNS = [{ header: 'On me prépare' }, { header: 'Ma part des appuis' }]

function rows(prepared: number | null): UsageGaugeRowModel[] {
  return [
    {
      key: 'appui',
      label: 'Appui reçu',
      gauges: [
        {
          key: 'prepared',
          valuePct: prepared,
          parityPct: null,
          valueText: prepared == null ? '—' : `${prepared} %`,
          tooltip: prepared == null ? '—' : `${prepared} % · 9 sur 20`,
        },
        { key: 'share', valuePct: 34, parityPct: 25, valueText: '34 %', tooltip: '34 % · 17 sur 50' },
      ],
    },
  ]
}

describe('UsageGaugeGrid — colonnes nommées par l’appelant', () => {
  it('rend une colonne par en-tête, la part de chaque jauge, et un axe par colonne', () => {
    const { container } = render(<UsageGaugeGrid rows={rows(44)} columns={COLUMNS} />)
    expect(screen.getByText('On me prépare')).toBeInTheDocument()
    expect(screen.getByText('Ma part des appuis')).toBeInTheDocument()
    expect(screen.getByText('44 %')).toBeInTheDocument()
    expect(screen.getByText('34 %')).toBeInTheDocument()
    expect(screen.getAllByText('100 %')).toHaveLength(2)
    expect(container.querySelectorAll('[role="img"]')).toHaveLength(2)
  })

  it('sans ligne : rien ne se rend', () => {
    const { container } = render(<UsageGaugeGrid rows={[]} columns={COLUMNS} />)
    expect(container.firstChild).toBeNull()
  })
})

describe('UsageGaugeGrid — le compte brut vit dans l’infobulle (D2)', () => {
  it('la cellule porte le pourcentage seul ; le rail porte la fraction', () => {
    render(<UsageGaugeGrid rows={rows(44)} columns={COLUMNS} />)
    expect(screen.queryByText(/9 sur 20/)).not.toBeInTheDocument()
    const rail = screen.getByRole('img', { name: /9 sur 20/ })
    expect(rail).toHaveAttribute('aria-label', expect.stringContaining('44 %'))
  })
})

describe('UsageGauge — non mesuré n’est pas zéro', () => {
  it('une part absente : aucune tranche, un tiret ; le trait seulement là où il existe', () => {
    const { container } = render(<UsageGaugeGrid rows={rows(null)} columns={COLUMNS} />)
    const [prepared, share] = Array.from(container.querySelectorAll('[role="img"]'))
    expect(prepared.querySelector('[data-outcome-fill]')).toBeNull()
    expect(prepared.querySelector('[data-gauge-parity]')).toBeNull()
    expect(share.querySelector('[data-outcome-fill]')).not.toBeNull()
    expect(share.querySelector('[data-gauge-parity]')).not.toBeNull()
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})
