/**
 * Tests — ReplayGenerateAction : rendu de chaque état servi, dans les deux variantes ; rien sans la capacité ;
 * aucun bouton quand l'action est impossible ; la demande envoyée par le clic et l'état qui en découle.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { ReplayGenerateAction } from './ReplayGenerateAction'
import type { ReplayGenerationOutcome, ReplayGenerationStatus } from './queries'
import { REPLAY_GENERATION_TEXT } from './text'

// <Link> exige un RouterProvider : remplacé par un <a> qui interpole les params (patron de match-card.test.tsx).
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
      for (const [key, value] of Object.entries(params ?? {})) href = href.replace(`$${key}`, value)
      return (
        <a href={href} {...rest}>
          {children}
        </a>
      )
    },
  }
})

const API = '/api/v1'
const PLAYER = 'joueur'
const MATCH = 'match-1'
const GEN_URL = `${API}/players/${PLAYER}/matches/${MATCH}/replay/generation`
const POST_URL = `${API}/players/${PLAYER}/matches/${MATCH}/replay/generate`
const EMPTY = 'Aucun rejeu 2D disponible pour ce match.'
const FR = REPLAY_GENERATION_TEXT.fr
const EN = REPLAY_GENERATION_TEXT.en

/** Sert l'état courant ; le POST fait évoluer l'état comme le serveur. */
function serveur(initial: ReplayGenerationStatus, post?: () => Response) {
  let current = initial
  const calls = { get: 0, post: 0 }
  server.use(
    http.get(GEN_URL, () => {
      calls.get += 1
      return HttpResponse.json(current)
    }),
    http.post(POST_URL, () => {
      calls.post += 1
      if (post) return post()
      current = { state: 'queued', position: 1 }
      const body: ReplayGenerationOutcome = { created: true, status: current }
      return HttpResponse.json(body, { status: 202 })
    }),
  )
  return calls
}

function panel() {
  return renderWithProviders(
    <ReplayGenerateAction variant="panel" playerSlug={PLAYER} matchId={MATCH} emptyLabel={EMPTY} />,
  )
}

function inline() {
  return renderWithProviders(<ReplayGenerateAction variant="inline" playerSlug={PLAYER} matchId={MATCH} />)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_generate_replays: true }) })
})

afterEach(() => {
  cleanup()
  useAppShellStore.setState({ locale: 'fr', capabilities: null })
})

describe('ReplayGenerateAction — sans la capacité', () => {
  beforeEach(() => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_generate_replays: false }) })
  })

  it('panel : le bloc d’état vide seul, ni requête ni bouton', async () => {
    const calls = serveur({ state: 'generatable' })
    panel()
    expect(screen.getByText(EMPTY)).toBeInTheDocument()
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(calls.get).toBe(0)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('inline : rien du tout, aucune requête', async () => {
    const calls = serveur({ state: 'generatable' })
    const { container } = inline()
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(calls.get).toBe(0)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayGenerateAction — variante panel', () => {
  it('générable : la phrase et le bouton ; le clic envoie la demande puis montre la file, sans bouton', async () => {
    const calls = serveur({ state: 'generatable' })
    panel()
    expect(screen.getByText(EMPTY)).toBeInTheDocument()
    const button = await screen.findByRole('button', { name: FR.generate })
    expect(screen.getByText(FR.messages.generatable)).toBeInTheDocument()
    fireEvent.click(button)
    expect(await screen.findByTestId('analysis-pending-slot')).toHaveTextContent('En file d’attente : position 1.')
    expect(calls.post).toBe(1)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it.each([
    ['en file avec position', { state: 'queued', position: 3 }, 'En file d’attente : position 3.'],
    ['en file en attente de la partie', { state: 'queued', position: 1, waiting_for_game: true }, FR.messages.waitingForGame],
    ['en file en pause', { state: 'queued', position: 1, paused: true }, FR.messages.paused],
    ['téléchargement du film', { state: 'running', phase: 'downloading' }, FR.messages.downloading],
    ['génération du rejeu', { state: 'running', phase: 'building' }, FR.messages.building],
  ] as const)('%s : indicateur d’attente, aucun bouton', async (_nom, state, text) => {
    serveur({ ...state })
    panel()
    expect(await screen.findByTestId('analysis-pending-slot')).toHaveTextContent(text)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it.each([
    ['film indisponible', { state: 'film_unavailable' }, FR.messages.filmUnavailable],
    ['mode sans rejeu', { state: 'not_supported', reason: 'mode_unmanaged' }, FR.messages.modeUnmanaged],
    ['match inconnu', { state: 'not_supported', reason: 'match_unknown' }, FR.messages.matchUnknown],
    ['aucun compte relié', { state: 'not_supported', reason: 'no_token' }, FR.messages.noToken],
    ['échec définitif', { state: 'failed', reason: 'map_not_in_catalog', retryable: false }, FR.messages.failedFinal],
    ['jeton indisponible, même réessayable côté serveur', { state: 'failed', reason: 'token_unavailable', retryable: true }, FR.messages.tokenUnavailable],
  ] as const)('%s : la phrase, ni bouton ni lien', async (_nom, state, text) => {
    serveur({ ...state })
    panel()
    expect(await screen.findByText(text)).toBeInTheDocument()
    expect(screen.getByText(EMPTY)).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.queryByTestId('analysis-pending-slot')).not.toBeInTheDocument()
  })

  it('film pas encore prêt : la phrase dédiée et « Réessayer »', async () => {
    const calls = serveur({ state: 'failed', reason: 'film_not_ready', retryable: true })
    panel()
    expect(await screen.findByText(FR.messages.filmNotReady)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: FR.retry }))
    await waitFor(() => expect(calls.post).toBe(1))
  })

  it('échec réessayable : la phrase et « Réessayer »', async () => {
    const calls = serveur({ state: 'failed', reason: 'fetch_failed', retryable: true })
    panel()
    expect(await screen.findByText(FR.messages.failed)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: FR.retry }))
    await waitFor(() => expect(calls.post).toBe(1))
  })

  it('prêt : le lien vers le rejeu', async () => {
    serveur({ state: 'ready' })
    panel()
    const link = await screen.findByRole('link', { name: FR.open })
    expect(link).toHaveAttribute('href', expect.stringContaining('/players/joueur/matches/match-1/replay'))
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('file pleine : le message, et le bouton reste', async () => {
    serveur({ state: 'generatable' }, () =>
      HttpResponse.json(
        { code: 'replay_generation_queue_full', message: 'file pleine', retryable: true },
        { status: 429 },
      ),
    )
    panel()
    fireEvent.click(await screen.findByRole('button', { name: FR.generate }))
    expect(await screen.findByRole('alert')).toHaveTextContent(FR.queueFull)
    expect(screen.getByRole('button', { name: FR.generate })).toBeInTheDocument()
  })

  it('textes en anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveur({ state: 'generatable' })
    panel()
    expect(await screen.findByRole('button', { name: EN.generate })).toBeInTheDocument()
    expect(screen.getByText(EN.messages.generatable)).toBeInTheDocument()
  })
})

describe('ReplayGenerateAction — variante inline', () => {
  it('générable : bouton contour avec son libellé ; le clic envoie la demande et montre la pastille', async () => {
    const calls = serveur({ state: 'generatable' })
    inline()
    const button = await screen.findByRole('button', { name: FR.generate })
    expect(button.className).toContain('border-border')
    expect(button.className).not.toContain('bg-primary')
    fireEvent.click(button)
    const pill = await screen.findByRole('status')
    expect(pill).toHaveTextContent('En file d’attente : position 1.')
    expect(calls.post).toBe(1)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('en cours : pastille compacte, aucun bloc ni bouton', async () => {
    serveur({ state: 'running', phase: 'downloading' })
    inline()
    expect(await screen.findByRole('status')).toHaveTextContent(FR.messages.downloading)
    expect(screen.queryByTestId('analysis-pending-slot')).not.toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it.each([
    ['film indisponible', { state: 'film_unavailable' }],
    ['non pris en charge', { state: 'not_supported', reason: 'no_token' }],
    ['échec définitif', { state: 'failed', reason: 'unknown_film_key', retryable: false }],
  ] as const)('%s : rien dans l’en-tête', async (_nom, state) => {
    const calls = serveur({ ...state })
    const { container } = inline()
    await waitFor(() => expect(calls.get).toBe(1))
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(container).toBeEmptyDOMElement()
  })

  it('échec réessayable : bouton « Réessayer »', async () => {
    serveur({ state: 'failed', reason: 'build_failed', retryable: true })
    inline()
    expect(await screen.findByRole('button', { name: FR.retry })).toBeInTheDocument()
  })

  it('prêt : lien vers le rejeu', async () => {
    serveur({ state: 'ready' })
    inline()
    expect(await screen.findByRole('link', { name: FR.open })).toBeInTheDocument()
  })
})
