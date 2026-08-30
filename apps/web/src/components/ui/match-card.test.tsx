/**
 * Tests unitaires — MatchCard (Sprint 56).
 *
 * Vérifie : image map, hiérarchie titre/sous-titre, score, badges narratifs et
 * lien rejeu 2D à droite de la playlist.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

// TanStack Router : <Link> (MatchReplayLink) exige un RouterProvider, absent en test
// unitaire. Remplacé par un <a> qui INTERPOLE les params dans le template de route —
// ce que le test vérifie (la route ciblée et ses params), pas le rendu du routeur.
// Patron : features/explorer/ExplorerMatchesTable.test.tsx.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  type LinkStubProps = {
    children?: React.ReactNode
    to: string
    params?: Record<string, string>
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>
  return {
    ...actual,
    Link: ({ children, to, params, ...rest }: LinkStubProps) => {
      let href = to
      for (const [key, value] of Object.entries(params ?? {})) {
        href = href.replace(`$${key}`, value)
      }
      return (
        <a href={href} {...rest}>
          {children}
        </a>
      )
    },
  }
})

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

  // Lien rejeu 2D à droite de la playlist : rendu UNIQUEMENT si l'artefact existe
  // (has_replay) ET que la tuile connaît le joueur (playerSlug — route par joueur).
  describe('lien rejeu 2D', () => {
    const REPLAY_LABEL = 'Ouvrir le rejeu 2D du match'

    it('affiche le lien vers le rejeu à côté de la playlist quand has_replay', () => {
      render(<MatchCard match={{ ...WIN_MATCH, has_replay: true }} playerSlug="chief" />)
      const link = screen.getByLabelText(REPLAY_LABEL)
      expect(link.getAttribute('href')).toContain('/matches/match-001/replay')
      expect(link.getAttribute('href')).toContain('/players/chief/')
      // À droite du label de playlist : même conteneur que le sous-titre.
      expect(link.parentElement?.textContent).toContain('Arène classée')
    })

    it('n\'affiche rien sans artefact de rejeu (pas de lien mort)', () => {
      render(<MatchCard match={WIN_MATCH} playerSlug="chief" />)
      expect(screen.queryByLabelText(REPLAY_LABEL)).toBeNull()
    })

    it('n\'affiche rien sans playerSlug (la route de rejeu est par joueur)', () => {
      render(<MatchCard match={{ ...WIN_MATCH, has_replay: true }} />)
      expect(screen.queryByLabelText(REPLAY_LABEL)).toBeNull()
    })

    it('affiche le lien même sans playlist (ligne dédiée)', () => {
      const noPlaylist: RecentMatchItem = { ...WIN_MATCH, playlist_ui: null, has_replay: true }
      render(<MatchCard match={noPlaylist} playerSlug="chief" />)
      expect(screen.getByLabelText(REPLAY_LABEL)).toBeTruthy()
    })
  })

  // V72-34 : la perf peut être structurellement absente (chaîne de performance en
  // calibration). La tuile doit le DIRE (« En placement (8/10) ») et surtout ne
  // JAMAIS afficher un 0 — un 0 se lirait comme la pire performance possible.
  describe('perf en placement (chaîne de performance en calibration)', () => {
    const IN_PLACEMENT: RecentMatchItem = {
      ...LOSS_MATCH,
      performance_score_relative: null,
      perf_placement_done: 8,
      perf_placement_total: 10,
    }

    it('affiche « En placement (8/10) » à la place du score', () => {
      render(<MatchCard match={IN_PLACEMENT} locale="fr" />)
      expect(screen.getByText('En placement (8/10)')).toBeTruthy()
    })

    it('n\'affiche JAMAIS un 0 fabriqué pour une perf absente', () => {
      render(<MatchCard match={IN_PLACEMENT} locale="fr" />)
      expect(screen.queryByText('0')).toBeNull()
    })

    it('EN : « In placement (8/10) »', () => {
      render(<MatchCard match={IN_PLACEMENT} locale="en" />)
      expect(screen.getByText('In placement (8/10)')).toBeTruthy()
    })

    it('perf absente SANS signal de placement → aucune mention (ni 0, ni badge)', () => {
      render(<MatchCard match={LOSS_MATCH} locale="fr" />)
      expect(screen.queryByText(/En placement/)).toBeNull()
      expect(screen.queryByText('0')).toBeNull()
    })

    it('perf présente → score affiché, pas de mention de placement', () => {
      render(<MatchCard match={WIN_MATCH} locale="fr" />)
      expect(screen.getByText('12')).toBeTruthy()
      expect(screen.queryByText(/En placement/)).toBeNull()
    })
  })
})
