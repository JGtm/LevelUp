/**
 * ResourceFilCard.test.tsx — la carte « au fil de la session » en vue compacte (tiroir de Sessions) :
 * graphe plus bas (170 contre 246) et axe match compact transmis à l'option ; la pleine page garde
 * sa hauteur. `ChartCard` est doublé pour lire ce que la carte lui passe.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import { buildResourceFil, empriseMatchIndex } from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { ResourceFilCard } from './ResourceFilCard'

const recu: { height?: number; option?: { xAxis: { axisLabel: { formatter: (v: string, i: number) => string } }[] } } = {}

vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: (p: { height: number; buildOption: () => unknown }) => {
    recu.height = p.height
    recu.option = p.buildOption() as typeof recu.option
    return <div data-testid="chart-card" />
  },
}))

const fil = buildResourceFil(EMPRISE_2209, empriseMatchIndex(HISTORY_2209))
const props = {
  fil,
  dominanceLabels: { 1: 'Domination', 2: 'Humiliation', 3: 'Remontada', 4: 'Débandade', 5: 'Contre-remontada', 6: 'Six', 7: 'Sept' },
  outcomeLabels: { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' },
  locale: 'fr' as const,
  t: EMPRISE_TEXT.fr,
}

describe('ResourceFilCard', () => {
  it('pleine page : 246 px, heure et carte sous l’axe', () => {
    render(<ResourceFilCard {...props} />)
    expect(recu.height).toBe(246)
    expect(recu.option!.xAxis[0].axisLabel.formatter('m1', 0)).not.toBe('')
  })

  it('compact : 170 px, rien sous l’axe', () => {
    render(<ResourceFilCard {...props} compact />)
    expect(recu.height).toBe(170)
    expect(recu.option!.xAxis[0].axisLabel.formatter('m1', 0)).toBe('')
  })
})
