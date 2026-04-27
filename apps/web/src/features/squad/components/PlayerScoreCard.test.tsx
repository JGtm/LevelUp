import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { PlayerScoreCard } from './PlayerScoreCard'

describe('PlayerScoreCard', () => {
  const baseProps = {
    gamertag: 'TestPlayer',
    score: 75,
    label: 'good',
    comparison: 'above' as const,
  }

  it('rend gamertag + score + label', () => {
    render(<PlayerScoreCard {...baseProps} />)
    expect(screen.getByTestId('player-score-card-gamertag').textContent).toBe('TestPlayer')
    expect(screen.getByTestId('player-score-card-score').textContent).toBe('75')
    expect(screen.getByTestId('player-score-card-label').textContent).toBe('good')
  })

  it('arrondit le score à l\'entier le plus proche', () => {
    render(<PlayerScoreCard {...baseProps} score={74.6} />)
    expect(screen.getByTestId('player-score-card-score').textContent).toBe('75')
    render(<PlayerScoreCard {...baseProps} score={74.4} />)
    // Multiple instances rendered, take last
    const scores = screen.getAllByTestId('player-score-card-score')
    expect(scores[scores.length - 1].textContent).toBe('74')
  })

  it('comparison above ▲', () => {
    render(<PlayerScoreCard {...baseProps} comparison="above" />)
    const cmp = screen.getByTestId('player-score-card-comparison')
    expect(cmp.textContent).toBe('▲')
    expect(cmp.getAttribute('data-comparison')).toBe('above')
  })

  it('comparison below ▼', () => {
    render(<PlayerScoreCard {...baseProps} comparison="below" />)
    expect(screen.getByTestId('player-score-card-comparison').textContent).toBe('▼')
  })

  it('comparison near =', () => {
    render(<PlayerScoreCard {...baseProps} comparison="near" />)
    expect(screen.getByTestId('player-score-card-comparison').textContent).toBe('=')
  })

  it('isMainPlayer applique un border highlight', () => {
    const { rerender } = render(<PlayerScoreCard {...baseProps} isMainPlayer />)
    const main = screen.getByTestId('player-score-card')
    expect(main.getAttribute('data-main')).toBe('true')
    expect(main.className).toContain('border-primary/40')

    rerender(<PlayerScoreCard {...baseProps} isMainPlayer={false} />)
    const second = screen.getByTestId('player-score-card')
    expect(second.getAttribute('data-main')).toBe('false')
    expect(second.className).not.toContain('border-primary/40')
  })

  it('applique la couleur via CSS variable selon le label', () => {
    render(<PlayerScoreCard {...baseProps} label="excellent" />)
    const score = screen.getByTestId('player-score-card-score') as HTMLElement
    expect(score.style.color).toContain('--score-excellent')
  })

  it('label inconnu fallback sur --score-average', () => {
    render(<PlayerScoreCard {...baseProps} label="unknown_label" />)
    const score = screen.getByTestId('player-score-card-score') as HTMLElement
    expect(score.style.color).toContain('--score-average')
  })

  it('comparaison applique la couleur via CSS variable', () => {
    render(<PlayerScoreCard {...baseProps} comparison="above" />)
    const cmp = screen.getByTestId('player-score-card-comparison') as HTMLElement
    expect(cmp.style.color).toContain('--narrative-trend-positive')
  })
})
