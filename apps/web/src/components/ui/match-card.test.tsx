/**
 * Tests unitaires — MatchCard (Sprint 56).
 *
 * Vérifie : badge résultat, K/A/D, image map, CombatYieldBar, performance score.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MatchCard } from './match-card'
import type { RecentMatchItem } from '@/lib/api/types'

const WIN_MATCH: RecentMatchItem = {
  match_id: 'match-001',
  title: 'Aquarius · Slayer',
  detail: '15K / 2D',
  started_at: '2026-04-10T20:00:00Z',
  outcome_label: 'Victoire',
  outcome_tone: 'win',
  map_ui: 'Aquarius',
  mode_ui: 'Slayer',
  kills: 15,
  assists: 4,
  deaths: 2,
  performance_score_relative: 12,
  offensive_conversion: 0.95,
  defensive_resistance: 1.80,
  damage_dealt: 3200,
  damage_taken: 1800,
  map_image_url: null,
}

const LOSS_MATCH: RecentMatchItem = {
  match_id: 'match-002',
  title: 'Empyrean · CTF',
  detail: '5K / 10D',
  started_at: '2026-04-11T18:00:00Z',
  outcome_label: 'Défaite',
  outcome_tone: 'loss',
  kills: 5,
  assists: 2,
  deaths: 10,
}

describe('MatchCard', () => {
  it('affiche le badge victoire en vert', () => {
    render(<MatchCard match={WIN_MATCH} />)
    const badge = screen.getByText('Victoire')
    expect(badge).toBeTruthy()
    expect(badge.className).toContain('green')
  })

  it('affiche le badge défaite en rouge', () => {
    render(<MatchCard match={LOSS_MATCH} />)
    const badge = screen.getByText('Défaite')
    expect(badge).toBeTruthy()
    expect(badge.className).toContain('red')
  })

  it('affiche les stats K/A/D', () => {
    render(<MatchCard match={WIN_MATCH} />)
    expect(screen.getByText('15')).toBeTruthy() // kills
    expect(screen.getByText('4')).toBeTruthy()  // assists
    expect(screen.getByText('2')).toBeTruthy()  // deaths
  })

  it("affiche le nom de la map et du mode", () => {
    render(<MatchCard match={WIN_MATCH} />)
    // "Aquarius" apparaît dans l'image fallback + le titre → getAllByText
    expect(screen.getAllByText('Aquarius').length).toBeGreaterThan(0)
    expect(screen.getByText('Slayer')).toBeTruthy()
  })

  it('affiche le performance score positif', () => {
    render(<MatchCard match={WIN_MATCH} />)
    expect(screen.getByText('+12')).toBeTruthy()
  })

  it('rend sans crasher quand les champs S56 sont absents', () => {
    render(<MatchCard match={LOSS_MATCH} />)
    expect(screen.getByText('Défaite')).toBeTruthy()
  })

  it('affiche un placeholder quand map_image_url est null', () => {
    render(<MatchCard match={WIN_MATCH} />)
    // Pas d'image → texte fallback dans le div
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('affiche l\'image map quand map_image_url est fournie', () => {
    const matchWithImg: RecentMatchItem = {
      ...WIN_MATCH,
      map_image_url: 'https://example.com/aquarius.webp',
    }
    render(<MatchCard match={matchWithImg} />)
    const img = screen.getByRole('img')
    expect(img).toBeTruthy()
    expect(img.getAttribute('src')).toBe('https://example.com/aquarius.webp')
  })

  it('appelle onClick quand fourni', () => {
    const handler = vi.fn()
    render(<MatchCard match={WIN_MATCH} onClick={handler} />)
    const card = screen.getByRole('button')
    card.click()
    expect(handler).toHaveBeenCalledOnce()
  })
})
