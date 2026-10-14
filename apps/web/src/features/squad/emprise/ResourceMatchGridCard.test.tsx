/**
 * ResourceMatchGridCard.test.tsx — la grille « match par match » en vue compacte (tiroir de
 * Sessions, maquette `renderGridCompact`) : en-tête réduit à l'heure, la carte et l'initiale du
 * résultat ; table compacte. La pleine page est couverte par `SquadEmprisePage.test.tsx`.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { buildMatchGrid, empriseMatchIndex } from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { ResourceMatchGridCard } from './ResourceMatchGridCard'

const props = {
  grid: buildMatchGrid(EMPRISE_2209, empriseMatchIndex(HISTORY_2209)),
  itemName: () => 'Objet',
  playerName: () => 'Moi',
  dominanceLabels: { 1: 'Domination', 2: 'Humiliation', 3: 'Remontada', 4: 'Débandade', 5: 'Contre-remontada', 6: 'Six', 7: 'Sept' },
  outcomeLabels: { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' },
  locale: 'fr' as const,
  t: EMPRISE_TEXT.fr,
}

describe('ResourceMatchGridCard — compact', () => {
  it('en-tête : la carte et l’initiale du résultat, sans mode, score ni dominance', () => {
    render(<ResourceMatchGridCard {...props} compact />)
    const head = screen.getByTestId('emprise-grid-head-m1')
    expect(head.textContent).toContain('Starboard')
    expect(screen.getByTestId('emprise-grid-result-m1').textContent).toBe('V')
    expect(screen.queryByTestId('emprise-grid-dominance-m1')).toBeNull()
  })

  it('pleine page inchangée : résultat et score en clair', () => {
    render(<ResourceMatchGridCard {...props} />)
    expect(screen.getByTestId('emprise-grid-result-m1').textContent).toBe('Victoire 3–0')
  })
})

describe('ResourceMatchGridCard — heure du match', () => {
  const heure = /\d{1,2}[:h]\d{2}/

  it('par défaut (Sessions) : l’heure de début figure dans l’en-tête', () => {
    render(<ResourceMatchGridCard {...props} />)
    expect(screen.getByTestId('emprise-grid-head-m1').textContent).toMatch(heure)
  })

  it('showTime faux (Escouade) : ni heure dans l’en-tête, ni dans l’infobulle d’en-tête', () => {
    render(<ResourceMatchGridCard {...props} showTime={false} />)
    const head = screen.getByTestId('emprise-grid-head-m1')
    expect(head.textContent).not.toMatch(heure)
    expect(head.textContent).toContain('Starboard')
  })
})
