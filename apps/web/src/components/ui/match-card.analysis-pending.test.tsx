/**
 * Tests — MatchCard, indicateur « analyse du film en cours » (`analysis_pending`).
 *
 * Trois états : EN COURS (un décodage porte sur ce match : `analysis_state == in_progress`),
 * EN ATTENTE (`queued`, ou état absent) et NORMAL (rien de tout cela). Pastille, infobulle et
 * emplacements ne disent « en cours » que dans le premier. Les emplacements ne s'affichent que
 * lorsque la donnée correspondante manque.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// <Link> exige un RouterProvider : remplacé par un <a> (patron de match-card.test.tsx).
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
import { commonManifest } from '@/lib/i18n/generated/common'
import type { MatchCitationSnippet, RecentMatchItem } from '@/lib/api/types'

const BASE: RecentMatchItem = {
  match_id: 'match-001',
  detail: '15K / 2D',
  started_at: '2026-04-10T20:00:00Z',
  outcome_tone: 'win',
  score_label: '50-42',
  map_ui: 'Aquarius',
  mode_ui: 'Assassin : Arène',
  playlist_ui: 'Arène classée',
  kills: 15,
  assists: 4,
  deaths: 2,
  map_image_url: null,
  is_favorite: false,
}

const ALL_PARTS = ['kill_source', 'citations', 'replay']
const PENDING: RecentMatchItem = {
  ...BASE,
  analysis_pending: true,
  analysis_state: 'in_progress',
  analysis_pending_parts: ALL_PARTS,
}
const QUEUED: RecentMatchItem = { ...PENDING, analysis_state: 'queued' }

function pendingWith(...parts: string[]): RecentMatchItem {
  return { ...BASE, analysis_pending: true, analysis_state: 'in_progress', analysis_pending_parts: parts }
}

const PARTS_BODY_FR = 'Frags assistés, arme de chaque frag · Citations · Rejeu 2D'

describe('MatchCard — analyse du film en cours', () => {
  describe('état EN COURS', () => {
    it('affiche la pastille « Analyse en cours », un bouton décrit par son infobulle', () => {
      render(<MatchCard match={PENDING} locale="fr" />)
      const badge = screen.getByRole('button', { name: 'Analyse en cours' })
      expect(badge.tagName).toBe('BUTTON')
      expect(badge.getAttribute('aria-describedby')).toBeTruthy()
      const tooltip = screen.getByRole('tooltip', { hidden: true })
      expect(tooltip.id).toBe(badge.getAttribute('aria-describedby'))
    })

    it('masque l\'infobulle au repos et l\'ouvre au survol (titre, texte, ligne de mise à jour)', async () => {
      const user = userEvent.setup()
      render(<MatchCard match={PENDING} locale="fr" />)
      expect(screen.queryByRole('tooltip')).toBeNull()

      await user.hover(screen.getByRole('button', { name: 'Analyse en cours' }))
      const tooltip = await screen.findByRole('tooltip')
      expect(within(tooltip).getByText('Analyse du film en cours')).toBeTruthy()
      expect(within(tooltip).getByTestId('match-card-analysis-pending-parts').textContent).toBe(PARTS_BODY_FR)
      expect(within(tooltip).getByText('La tuile se met à jour seule.')).toBeTruthy()

      await user.unhover(screen.getByRole('button', { name: 'Analyse en cours' }))
      expect(screen.queryByRole('tooltip')).toBeNull()
    })

    it('ouvre l\'infobulle au focus clavier et la ferme à Échap', async () => {
      const user = userEvent.setup()
      render(<MatchCard match={PENDING} locale="fr" />)
      await user.tab()
      // Le premier arrêt de tabulation de la tuile sans favori est la pastille.
      expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Analyse en cours' }))
      expect(await screen.findByRole('tooltip')).toBeTruthy()
      await user.keyboard('{Escape}')
      expect(screen.queryByRole('tooltip')).toBeNull()
    })

    it('infobulle : seules les parties en attente sont nommées', async () => {
      const user = userEvent.setup()
      render(<MatchCard match={pendingWith('citations', 'replay')} locale="fr" />)
      await user.hover(screen.getByRole('button', { name: 'Analyse en cours' }))
      const tooltip = await screen.findByRole('tooltip')
      expect(within(tooltip).getByTestId('match-card-analysis-pending-parts').textContent).toBe('Citations · Rejeu 2D')
    })

    it('infobulle sans cause servie : pas de ligne de parties', async () => {
      const user = userEvent.setup()
      render(<MatchCard match={{ ...BASE, analysis_pending: true, analysis_state: 'in_progress' }} locale="fr" />)
      await user.hover(screen.getByRole('button', { name: 'Analyse en cours' }))
      const tooltip = await screen.findByRole('tooltip')
      expect(within(tooltip).queryByTestId('match-card-analysis-pending-parts')).toBeNull()
    })

    it('aucun libellé ne promet un délai, ne dit « à venir » ni « en général » / « quelques minutes »', () => {
      for (const loc of ['fr', 'en'] as const) {
        const texts = Object.entries(commonManifest)
          .filter(([key]) => key.startsWith('common.match_card.analysis_pending.'))
          .map(([, entry]) => (entry as Record<'fr' | 'en', string>)[loc])
          .join(' | ')
        expect(texts).not.toMatch(/à venir|coming|upcoming|d'ici|within|moments|quelques|instants|en général/i)
      }
    })

    it('met les trois emplacements en attente à la place des trous', () => {
      render(<MatchCard match={PENDING} locale="fr" playerSlug="chief" />)
      expect(screen.getByTestId('match-card-assisted-frags-pending').textContent).toBe(
        'Frags assistés : analyse en cours',
      )
      const citations = screen.getByTestId('match-card-citations-pending')
      expect(citations.textContent).toBe('Citations : analyse en cours')
      expect(citations.querySelectorAll('.border-dashed')).toHaveLength(3)
      expect(screen.getByTestId('match-card-replay-pending').textContent).toBe('Rejeu 2D : analyse en cours')
      // L'emplacement vide d'origine cède la place à l'emplacement en attente.
      expect(screen.queryByTestId('match-card-assisted-frags-slot')).toBeNull()
    })

    it('traduit tout en anglais', () => {
      render(<MatchCard match={PENDING} locale="en" playerSlug="chief" />)
      expect(screen.getByRole('button', { name: 'Analysis in progress' })).toBeTruthy()
      expect(screen.getByTestId('match-card-assisted-frags-pending').textContent).toBe(
        'Assisted kills: analysis in progress',
      )
      expect(screen.getByTestId('match-card-citations-pending').textContent).toBe('Citations: analysis in progress')
      expect(screen.getByTestId('match-card-replay-pending').textContent).toBe('2D replay: analysis in progress')
    })

    it('n\'anime que sous motion-safe (prefers-reduced-motion respecté)', () => {
      render(<MatchCard match={PENDING} locale="fr" playerSlug="chief" />)
      const animated = document.querySelectorAll('[class*="animate-"]')
      expect(animated.length).toBeGreaterThanOrEqual(2) // point qui pulse + indicateur tournant
      for (const el of Array.from(animated)) {
        const animations = el.className.split(/\s+/).filter((c) => c.includes('animate-'))
        for (const cls of animations) expect(cls.startsWith('motion-safe:')).toBe(true)
      }
    })

    it('ne remplace que ce qui manque : frags assistés, citations et rejeu déjà là = aucun emplacement', () => {
      const citation = {
        key: 'bulltrue',
        name: 'Bull True',
        delta: 2,
        progress_pct: 40,
        is_newly_mastered: false,
      } as unknown as MatchCitationSnippet
      const done: RecentMatchItem = {
        ...PENDING,
        has_replay: true,
        top_citations: [citation],
        assisted_frags: {
          frags_film: 10,
          frags_official: 15,
          received: { total: 3, low: 1, mid: 1, high: 1 },
        } as RecentMatchItem['assisted_frags'],
      }
      render(<MatchCard match={done} locale="fr" playerSlug="chief" />)
      // La pastille reste (le signal serveur dit que l'analyse n'est pas finie).
      expect(screen.getByRole('button', { name: 'Analyse en cours' })).toBeTruthy()
      expect(screen.queryByTestId('match-card-assisted-frags-pending')).toBeNull()
      expect(screen.queryByTestId('match-card-citations-pending')).toBeNull()
      expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
      expect(screen.getByTestId('match-card-assisted-frags')).toBeTruthy()
    })

    describe('chaque emplacement regarde sa cause', () => {
      it('citations seules : cercles en pointillés, aucune puce de rejeu, frags assistés normaux', () => {
        render(<MatchCard match={pendingWith('citations')} locale="fr" playerSlug="chief" />)
        expect(screen.getByRole('button', { name: 'Analyse en cours' })).toBeTruthy()
        const citations = screen.getByTestId('match-card-citations-pending')
        expect(citations.querySelectorAll('.border-dashed')).toHaveLength(3)
        expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
        expect(screen.queryByTestId('match-card-assisted-frags-pending')).toBeNull()
        expect(screen.getByTestId('match-card-assisted-frags-slot')).toBeTruthy()
      })

      it('kill_source seule : frags assistés en attente, ni citations ni rejeu', () => {
        render(<MatchCard match={pendingWith('kill_source')} locale="fr" playerSlug="chief" />)
        expect(screen.getByTestId('match-card-assisted-frags-pending')).toBeTruthy()
        expect(screen.queryByTestId('match-card-citations-pending')).toBeNull()
        expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
      })

      it('replay seul : puce « Rejeu 2D » seule', () => {
        render(<MatchCard match={pendingWith('replay')} locale="fr" playerSlug="chief" />)
        expect(screen.getByTestId('match-card-replay-pending').textContent).toBe('Rejeu 2D : analyse en cours')
        expect(screen.queryByTestId('match-card-citations-pending')).toBeNull()
        expect(screen.queryByTestId('match-card-assisted-frags-pending')).toBeNull()
        expect(screen.getByTestId('match-card-assisted-frags-slot')).toBeTruthy()
      })

      it('booléen sans cause : la pastille reste, aucun emplacement', () => {
        render(<MatchCard match={{ ...BASE, analysis_pending: true }} locale="fr" playerSlug="chief" />)
        // Sans état servi, la pastille ne dit jamais « en cours ».
        expect(screen.getByRole('button', { name: 'Analyse en attente' })).toBeTruthy()
        expect(screen.queryByTestId('match-card-assisted-frags-pending')).toBeNull()
        expect(screen.queryByTestId('match-card-citations-pending')).toBeNull()
        expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
      })
    })

    it('sans playerSlug, pas de puce de rejeu (la route de rejeu est par joueur)', () => {
      render(<MatchCard match={PENDING} locale="fr" />)
      expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
    })
  })

  describe('état EN ATTENTE (aucun décodage ne porte sur ce match)', () => {
    it.each([
      ['queued', QUEUED],
      ['état absent', { ...QUEUED, analysis_state: undefined }],
      ['état inconnu', { ...QUEUED, analysis_state: 'autre' }],
    ])('%s : pastille et titre disent « en attente », jamais « en cours »', async (_label, match) => {
      const user = userEvent.setup()
      render(<MatchCard match={match} locale="fr" playerSlug="chief" />)
      const badge = screen.getByRole('button', { name: 'Analyse en attente' })
      expect(screen.queryByRole('button', { name: 'Analyse en cours' })).toBeNull()
      await user.hover(badge)
      const tooltip = await screen.findByRole('tooltip')
      expect(within(tooltip).getByText('Analyse du film en attente')).toBeTruthy()
      expect(within(tooltip).queryByText('Analyse du film en cours')).toBeNull()
      expect(within(tooltip).getByTestId('match-card-analysis-pending-parts').textContent).toBe(PARTS_BODY_FR)
      expect(within(tooltip).getByText('La tuile se met à jour seule.')).toBeTruthy()
    })

    it('les trois emplacements disent « analyse en attente »', () => {
      render(<MatchCard match={QUEUED} locale="fr" playerSlug="chief" />)
      expect(screen.getByTestId('match-card-assisted-frags-pending').textContent).toBe('Frags assistés : analyse en attente')
      expect(screen.getByTestId('match-card-citations-pending').textContent).toBe('Citations : analyse en attente')
      expect(screen.getByTestId('match-card-replay-pending').textContent).toBe('Rejeu 2D : analyse en attente')
    })

    it('traduit tout en anglais', async () => {
      const user = userEvent.setup()
      render(<MatchCard match={QUEUED} locale="en" playerSlug="chief" />)
      await user.hover(screen.getByRole('button', { name: 'Analysis pending' }))
      const tooltip = await screen.findByRole('tooltip')
      expect(within(tooltip).getByText('Film analysis pending')).toBeTruthy()
      expect(within(tooltip).getByTestId('match-card-analysis-pending-parts').textContent).toBe(
        'Assisted frags, weapon per frag · Citations · 2D replay',
      )
      expect(screen.getByTestId('match-card-assisted-frags-pending').textContent).toBe('Assisted kills: analysis pending')
      expect(screen.getByTestId('match-card-citations-pending').textContent).toBe('Citations: analysis pending')
      expect(screen.getByTestId('match-card-replay-pending').textContent).toBe('2D replay: analysis pending')
    })
  })

  describe('état NORMAL', () => {
    it.each([
      ['analysis_pending absent', BASE],
      ['analysis_pending faux', { ...BASE, analysis_pending: false }],
    ])('%s : ni pastille, ni infobulle, ni emplacement en attente', (_label, match) => {
      render(<MatchCard match={match} locale="fr" playerSlug="chief" />)
      expect(screen.queryByRole('button', { name: 'Analyse en cours' })).toBeNull()
      expect(screen.queryByRole('tooltip', { hidden: true })).toBeNull()
      expect(screen.queryByTestId('match-card-analysis-pending-badge')).toBeNull()
      expect(screen.queryByTestId('match-card-assisted-frags-pending')).toBeNull()
      expect(screen.queryByTestId('match-card-citations-pending')).toBeNull()
      expect(screen.queryByTestId('match-card-replay-pending')).toBeNull()
      // L'emplacement vide d'origine est conservé (alignement des tuiles voisines).
      expect(screen.getByTestId('match-card-assisted-frags-slot')).toBeTruthy()
    })
  })
})
