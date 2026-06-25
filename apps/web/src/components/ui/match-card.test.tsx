/**
 * Tests unitaires — MatchCard (Sprint 56).
 *
 * Vérifie : image map, hiérarchie titre/sous-titre, score et badges narratifs.
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
  score_label: '50-42',
  narrative_badges: ['dominant', 'remontada'],
  map_ui: 'Aquarius',
  mode_ui: 'Assassin : Arène',
  playlist_ui: 'Arène classée',
  kills: 15,
  assists: 4,
  deaths: 2,
  performance_score_relative: 12,
  offensive_conversion: 0.95,
  defensive_resistance: 1.80,
  damage_dealt: 3200,
  damage_taken: 1800,
  map_image_url: null,
  is_favorite: false,
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
  is_favorite: false,
}

describe('MatchCard', () => {
  it('n\'affiche plus la pastille de résultat', () => {
    render(<MatchCard match={WIN_MATCH} />)
    expect(screen.queryByText('Victoire')).toBeNull()
  })

  it('affiche le titre centré mode sur carte et la playlist', () => {
    render(<MatchCard match={WIN_MATCH} locale="fr" />)
    expect(screen.getByText('Assassin sur Aquarius')).toBeTruthy()
    expect(screen.queryByText('Assassin : Arène sur Aquarius')).toBeNull()
    expect(screen.getByText('Arène classée')).toBeTruthy()
  })

  it('utilise le connecteur anglais quand la locale UI est en', () => {
    render(<MatchCard match={WIN_MATCH} locale="en" />)
    expect(screen.getByText('Assassin on Aquarius')).toBeTruthy()
  })

  it('strip le nom de carte EN collé au mode même si map_ui est FR (régression "Slayer on Forest sur Forêt")', () => {
    const crossLang: RecentMatchItem = { ...WIN_MATCH, mode_ui: 'Slayer on Forest', map_ui: 'Forêt' }
    render(<MatchCard match={crossLang} locale="fr" />)
    expect(screen.getByText('Slayer sur Forêt')).toBeTruthy()
    expect(screen.queryByText('Slayer on Forest sur Forêt')).toBeNull()
  })

  it('rend sans crasher quand les champs S56 sont absents', () => {
    render(<MatchCard match={LOSS_MATCH} />)
    expect(screen.getByText('Empyrean · CTF')).toBeTruthy()
    expect(screen.getByTestId('match-card-score').textContent).toBe('')
  })

  it('affiche le score et les badges narratifs dans une section fixe', () => {
    render(<MatchCard match={WIN_MATCH} />)
    const panel = screen.getByTestId('match-card-stats-panel')
    expect(panel).toBeTruthy()
    expect(screen.getByTestId('match-card-score').textContent).toBe('50-42')
    expect(screen.getByText('DOMINATION')).toBeTruthy()
    expect(screen.getByText('REMONTADA')).toBeTruthy()
    expect(screen.getByTestId('match-card-badges-row')).toBeTruthy()
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

  it('affiche la barre KDA avec les labels frags/assist./morts', () => {
    render(<MatchCard match={WIN_MATCH} />)
    const bar = screen.getByTestId('match-card-kda-bar')
    expect(bar).toBeTruthy()
    expect(screen.getByText('frags')).toBeTruthy()
    expect(screen.getByText('assist.')).toBeTruthy()
    expect(screen.getByText('morts')).toBeTruthy()
    expect(bar.textContent).toContain('15')
    expect(bar.textContent).toContain('4')
    expect(bar.textContent).toContain('2')
  })

  it('affiche la barre KDA même sans bloc perf/skill', () => {
    render(<MatchCard match={LOSS_MATCH} />)
    const bar = screen.getByTestId('match-card-kda-bar')
    expect(bar).toBeTruthy()
    expect(screen.getByText('frags')).toBeTruthy()
    expect(screen.getByText('assist.')).toBeTruthy()
    expect(screen.getByText('morts')).toBeTruthy()
    expect(bar.textContent).toContain('5')
    expect(bar.textContent).toContain('2')
    expect(bar.textContent).toContain('10')
  })

  it('n\'affiche pas la barre KDA si tous les champs sont null', () => {
    const matchNoKDA: RecentMatchItem = { ...LOSS_MATCH, kills: null, assists: null, deaths: null }
    render(<MatchCard match={matchNoKDA} />)
    expect(screen.queryByTestId('match-card-kda-bar')).toBeNull()
  })
})
