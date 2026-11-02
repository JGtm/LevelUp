import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { act } from '@testing-library/react'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { FeedbackDrawer } from './FeedbackDrawer'
import { useFeedbackDrawerStore } from './feedbackDrawer.store'
import { _resetSubmitsForTests } from './rateLimit'
import { resetCaptureBuffersForTests } from '@/lib/global-capture/buffers'
import { log } from './_logger'

let originalFetch: typeof window.fetch
let clickedLinks: HTMLAnchorElement[] = []

beforeEach(() => {
  useFeedbackDrawerStore.setState({ isOpen: false })
  _resetSubmitsForTests()
  resetCaptureBuffersForTests()
  log._resetForTests()
  originalFetch = window.fetch
  window.fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ items: [], total_count: 0 }), { status: 200 }),
  ) as typeof window.fetch
  clickedLinks = []
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    clickedLinks.push(this)
  })
})

afterEach(() => {
  window.fetch = originalFetch
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

describe('FeedbackDrawer — application de bureau', () => {
  afterEach(() => {
    act(() => useAppShellStore.setState({ capabilities: null }))
  })

  it('le mini-onglet du bord est masqué : l’entrée est au pied de la barre latérale', () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) }))
    renderWithProviders(<FeedbackDrawer />)
    expect(screen.getByRole('button', { name: /Envoyer un retour/i }).className).toMatch(/(^|\s)hidden(\s|$)/)
    expect(screen.getByRole('button', { name: /Envoyer un retour/i }).className).not.toMatch(/sm:flex/)
  })

  it('sur le site, le mini-onglet reste affiché dès sm', () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities() }))
    renderWithProviders(<FeedbackDrawer />)
    expect(screen.getByRole('button', { name: /Envoyer un retour/i }).className).toMatch(/sm:flex/)
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

  it("submit ouvre l'URL GitHub dans un nouvel onglet (lien target=_blank, sans window.open)", async () => {
    const openSpy = vi.spyOn(window, 'open')
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    const titleInput = screen.getByPlaceholderText(/Résume ton retour/i)
    fireEvent.change(titleInput, { target: { value: 'mon retour' } })
    const submitBtn = screen.getByRole('button', { name: /Ouvrir sur GitHub/i })
    await waitFor(() => expect(submitBtn).toBeEnabled())
    fireEvent.click(submitBtn)
    expect(clickedLinks).toHaveLength(1)
    const link = clickedLinks[0]
    expect(link.href).toContain('https://github.com/JGtm/LevelUp/issues/new?')
    expect(link.href).toContain('labels=feedback%2Cbug')
    expect(link.target).toBe('_blank')
    expect(link.rel).toContain('noopener')
    expect(openSpy).not.toHaveBeenCalled()
    // Le formulaire se referme et le lien temporaire ne reste pas dans la page.
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
    expect(document.body.querySelector('a[target="_blank"][hidden]')).toBeNull()
  })

  it("change de type via segmented control", () => {
    useFeedbackDrawerStore.setState({ isOpen: true })
    renderWithProviders(<FeedbackDrawer />)
    const ideaTab = screen.getByRole('tab', { name: /Idée/i })
    fireEvent.click(ideaTab)
    expect(ideaTab).toHaveAttribute('aria-selected', 'true')
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
})
