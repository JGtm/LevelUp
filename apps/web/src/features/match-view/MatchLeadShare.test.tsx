import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

import { LeadShareBlock, LeadShareRecall } from './MatchLeadShare'
import { showsLeadShare } from './matchLeadShare.logic'

describe('showsLeadShare — quand le bloc existe', () => {
  it('avec un film, ou avec une mesure déjà servie ; jamais sans film ni mesure', () => {
    expect(showsLeadShare({ replayAvailable: true })).toBe(true)
    expect(showsLeadShare({ leadShare: 0.4, replayAvailable: false })).toBe(true)
    expect(showsLeadShare({ replayAvailable: false })).toBe(false)
    // Zéro MESURÉ est une mesure : le bloc existe.
    expect(showsLeadShare({ leadShare: 0, replayAvailable: false })).toBe(true)
  })
})

describe('LeadShareBlock — le bloc de l’en-tête', () => {
  it('FR : pourcentage entier et retournements', () => {
    const view = render(<LeadShareBlock leadShare={0.624} leadChanges={3} replayAvailable locale="fr" />)
    expect(view.getByText(LEAD_SHARE_TEXT.fr.name)).toBeTruthy()
    expect(view.getByTestId('match-lead-share-value').textContent).toBe('62 %')
    expect(view.getByText('3 retournements')).toBeTruthy()
  })

  it('EN : nom et pluriel anglais, un retournement au singulier', () => {
    const view = render(<LeadShareBlock leadShare={0.5} leadChanges={1} replayAvailable locale="en" />)
    expect(view.getByText('Lead share')).toBeTruthy()
    expect(view.getByText('1 lead change')).toBeTruthy()
  })

  it('un match non mesuré : TIRET, jamais « 0 % », et l’infobulle dit pourquoi', () => {
    const view = render(<LeadShareBlock replayAvailable locale="fr" />)
    const value = view.getByTestId('match-lead-share-value')
    expect(value.textContent).toBe('—')
    expect(value.getAttribute('title')).toBe(LEAD_SHARE_TEXT.fr.notMeasured)
    expect(view.queryByText(/retournement/)).toBeNull()
  })

  it('une part mesurée à zéro s’affiche « 0 % » : c’est une mesure', () => {
    const view = render(<LeadShareBlock leadShare={0} leadChanges={0} replayAvailable locale="fr" />)
    expect(view.getByTestId('match-lead-share-value').textContent).toBe('0 %')
    expect(view.getByText('0 retournements')).toBeTruthy()
  })
})

describe('LeadShareRecall — le rappel sous « Écart de score »', () => {
  it('écrit la même mesure que l’en-tête', () => {
    const view = render(<LeadShareRecall leadShare={0.62} leadChanges={3} replayAvailable locale="fr" />)
    expect(view.getByTestId('match-lead-share-recall').textContent).toBe('Mainmise : 62 % · 3 retournements')
  })

  it('non mesuré : le tiret', () => {
    const view = render(<LeadShareRecall replayAvailable locale="fr" />)
    expect(view.getByTestId('match-lead-share-recall').textContent).toBe('Mainmise : —')
  })
})
