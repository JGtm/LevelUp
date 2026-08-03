import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { FeedbackDrawer } from './FeedbackDrawer'
import { useFeedbackDrawerStore } from './feedbackDrawer.store'
import { _resetSubmitsForTests } from './rateLimit'
import { resetCaptureBuffersForTests } from '@/lib/global-capture/buffers'
import { log } from './_logger'

let originalFetch: typeof window.fetch
let originalOpen: typeof window.open

beforeEach(() => {
  useFeedbackDrawerStore.setState({ isOpen: false })
  _resetSubmitsForTests()
  resetCaptureBuffersForTests()
  log._resetForTests()
  originalFetch = window.fetch
  originalOpen = window.open
  window.fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ items: [], total_count: 0 }), { status: 200 }),
  ) as typeof window.fetch
  window.open = vi.fn().mockReturnValue({ focus() {} } as unknown as Window) as typeof window.open
})

afterEach(() => {
  window.fetch = originalFetch
  window.open = originalOpen
  vi.restoreAllMocks()
})

describe('FeedbackDrawer — rendu et accessibilité', () => {
  it('rend le mini-tab caché initialement (drawer fermé)', () => {
    renderWithProviders(<FeedbackDrawer />)
    const btn = screen.getByRole('button', { name: /Envoyer un retour/i })
    expect(btn).toBeInTheDocument()
    expect(btn).toHaveAttribute('aria-expanded', 'false')
  })

  it("aria-hidden sur le panneau quand fermé", () => {
    renderWithProviders(<FeedbackDrawer />)
    const panel = screen.getByRole('complementary', { hidden: true })
    expect(panel).toHaveAttribute('aria-hidden', 'true')
  })

  it("ouvre le drawer au clic sur le mini-tab", () => {
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.click(screen.getByRole('button', { name: /Envoyer un retour/i }))
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })
})

describe('FeedbackDrawer — interactions', () => {
  it("Escape ferme le drawer", () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
  })

  it("submit désactivé si titre vide", () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    const btn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    expect(btn).toBeDisabled()
  })

  it("submit ouvre window.open avec URL GitHub valide", async () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    const titleInput = screen.getByPlaceholderText(/Résume ton retour/i)
    fireEvent.change(titleInput, { target: { value: 'mon retour' } })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeEnabled())
    fireEvent.click(submitBtn)
    expect(window.open).toHaveBeenCalled()
    const [url, target, features] = (window.open as unknown as ReturnType<typeof vi.fn>).mock
      .calls[0]
    expect(url).toContain('https://github.com/JGtm/LevelUp/issues/new?')
    expect(url).toContain('labels=feedback%2Cbug')
    expect(target).toBe('_blank')
    expect(features).toContain('noopener')
  })

  it("change de type via segmented control", () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    const ideaTab = screen.getByRole('tab', { name: /Idée/i })
    fireEvent.click(ideaTab)
    expect(ideaTab).toHaveAttribute('aria-selected', 'true')
  })
})

describe('FeedbackDrawer — popup blocker fallback', () => {
  it("utilise clipboard si window.open retourne null", async () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    window.open = vi.fn().mockReturnValue(null) as typeof window.open
    const writeTextSpy = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeTextSpy },
      configurable: true,
    })
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.change(screen.getByPlaceholderText(/Résume ton retour/i), {
      target: { value: 'titre' },
    })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeEnabled())
    fireEvent.click(submitBtn)
    await waitFor(() => expect(writeTextSpy).toHaveBeenCalled())
    expect((writeTextSpy.mock.calls[0]?.[0] as string)).toContain(
      'https://github.com/JGtm/LevelUp/issues/new',
    )
  })
})

describe('FeedbackDrawer — anti-spam', () => {
  it("bouton désactivé après 5 submits dans l'heure", async () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    // Pré-remplir le localStorage avec 5 timestamps récents
    window.localStorage.setItem(
      'levelup-feedback-submits',
      JSON.stringify(Array.from({ length: 5 }, () => Date.now())),
    )
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.change(screen.getByPlaceholderText(/Résume ton retour/i), {
      target: { value: 'titre' },
    })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeDisabled())
  })
})

describe('FeedbackDrawer — observabilité', () => {
  it("log.info structurée {labels, urlLength} au submit (pas de PII)", async () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    const infoSpy = vi.spyOn(console, 'info').mockImplementation(() => {})
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.change(screen.getByPlaceholderText(/Résume ton retour/i), {
      target: { value: 'titre top secret' },
    })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeEnabled())
    fireEvent.click(submitBtn)
    await waitFor(() => expect(infoSpy).toHaveBeenCalled())
    const call = infoSpy.mock.calls.find((c) => String(c[0]).includes('feedback submitted'))
    expect(call).toBeTruthy()
    const meta = call?.[1] as { labels: string; urlLength: number }
    expect(meta.labels).toContain('feedback,bug')
    expect(meta.urlLength).toBeGreaterThan(0)
    // Pas de PII : titre user n'apparait pas dans la metadata loggée
    const allArgs = JSON.stringify(call)
    expect(allArgs).not.toContain('top secret')
  })

  it("clipboard indisponible (HTTP context) → toast erreur sans crash", async () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    window.open = vi.fn().mockReturnValue(null) as typeof window.open
    Object.defineProperty(navigator, 'clipboard', {
      value: undefined,
      configurable: true,
    })
    renderWithProviders(<FeedbackDrawer />)
    fireEvent.change(screen.getByPlaceholderText(/Résume ton retour/i), {
      target: { value: 'titre' },
    })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeEnabled())
    expect(() => fireEvent.click(submitBtn)).not.toThrow()
  })
})
