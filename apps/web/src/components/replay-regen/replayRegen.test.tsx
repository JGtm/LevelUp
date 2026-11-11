/**
 * Tests — régénération des rejeux : phases pures, bandeau (capacité, `stale > 0`, « Plus tard », progression,
 * pause et reprise) et carte des Paramètres (même composant que le bandeau).
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { ReplayRegenCard } from './ReplayRegenCard'
import { bannerPhaseVisible, regenPercent, regenPhase } from './logic'
import { ReplayRegenBanner } from './ReplayRegenBanner'
import { useRegenDismissedStore } from './dismissed'
import type { ReplayRegenStatus } from './queries'

const API = '/api/v1'

const IDLE: ReplayRegenStatus = {
  checked: true, stale: 0, running: false, done: 0, total: 0, paused: false, waiting_for_game: false, failed: 0,
  compressing: false, compressed: 0, compress_failed: 0, raw: 0,
}

/** Sert l'état courant ; les commandes le font évoluer comme le serveur. */
function serveur(initial: Partial<ReplayRegenStatus>) {
  let state: ReplayRegenStatus = { ...IDLE, ...initial }
  const commands: string[] = []
  server.use(
    http.get(`${API}/desktop/replay-regen`, () => HttpResponse.json(state)),
    http.post(`${API}/desktop/replay-regen/:command`, ({ params }) => {
      const command = String(params.command)
      commands.push(command)
      if (command === 'start') state = { ...state, running: true, total: 4, done: 1, stale: 3 }
      if (command === 'pause') state = { ...state, paused: true }
      if (command === 'resume') state = { ...state, paused: false }
      if (command === 'rescan') state = { ...state, checked: false }
      return HttpResponse.json(state, { status: command === 'start' || command === 'rescan' ? 202 : 200 })
    }),
  )
  return { commands, set: (s: Partial<ReplayRegenStatus>) => (state = { ...state, ...s }) }
}

describe('regenPhase et visibilité', () => {
  it('lit la phase de l’état du serveur', () => {
    expect(regenPhase(undefined)).toBe('checking')
    expect(regenPhase({ ...IDLE, checked: false })).toBe('checking')
    expect(regenPhase(IDLE)).toBe('upToDate')
    expect(regenPhase({ ...IDLE, stale: 5 })).toBe('offer')
    // Aucun rejeu à refaire mais des rejeux bruts à compresser : l'offre est faite aussi.
    expect(regenPhase({ ...IDLE, raw: 4 })).toBe('offer')
    expect(regenPhase({ ...IDLE, running: true })).toBe('preparing')
    expect(regenPhase({ ...IDLE, running: true, total: 4, done: 1 })).toBe('running')
    expect(regenPhase({ ...IDLE, running: true, total: 4, paused: true })).toBe('paused')
    expect(regenPhase({ ...IDLE, running: true, total: 4, waiting_for_game: true })).toBe('waitingGame')
    // La compression des rejeux bruts, en fin de passe, même quand il n'y avait aucun rejeu à refaire (total 0).
    expect(regenPhase({ ...IDLE, running: true, total: 4, done: 4, compressing: true })).toBe('compressing')
    expect(regenPhase({ ...IDLE, running: true, compressing: true })).toBe('compressing')
    expect(regenPhase({ ...IDLE, running: true, compressing: true, paused: true })).toBe('paused')
    expect(regenPhase({ ...IDLE, running: true, compressing: true, waiting_for_game: true })).toBe('waitingGame')
  })

  it('« Plus tard » ne masque que l’offre, jamais une file lancée', () => {
    expect(bannerPhaseVisible('offer', false)).toBe(true)
    expect(bannerPhaseVisible('offer', true)).toBe(false)
    expect(bannerPhaseVisible('running', true)).toBe(true)
    expect(bannerPhaseVisible('compressing', true)).toBe(true)
    expect(bannerPhaseVisible('upToDate', false)).toBe(false)
    expect(bannerPhaseVisible('checking', false)).toBe(false)
  })

  it('calcule la part faite, bornée', () => {
    expect(regenPercent({ done: 0, total: 0 })).toBe(0)
    expect(regenPercent({ done: 1, total: 4 })).toBe(25)
    expect(regenPercent({ done: 9, total: 4 })).toBe(100)
  })
})

describe('ReplayRegenBanner', () => {
  beforeEach(() => {
    useRegenDismissedStore.setState({ dismissed: false })
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('montre le nombre de rejeux à régénérer, sous la capacité et quand stale > 0', async () => {
    serveur({ stale: 12 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('12 rejeux à régénérer après la mise à jour.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Régénérer' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Plus tard' })).toBeInTheDocument()
  })

  it('propose la compression seule quand il ne reste que des rejeux bruts, en français et en anglais', async () => {
    serveur({ stale: 0, raw: 4 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('4 rejeux à compresser pour libérer de la place sur le disque.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Compresser' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Régénérer' })).not.toBeInTheDocument()
    cleanup()
    useAppShellStore.setState({ locale: 'en' })
    serveur({ stale: 0, raw: 1 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('1 replay to compress to free up disk space.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Compress' })).toBeInTheDocument()
  })

  it('avec des rejeux à refaire, l offre reste celle de la régénération, bruts ou non', async () => {
    serveur({ stale: 3, raw: 9 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('3 rejeux à régénérer après la mise à jour.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Régénérer' })).toBeInTheDocument()
  })

  it('dit « 1 rejeu » au singulier, et parle anglais en anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveur({ stale: 1 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('1 replay to regenerate after the update.')).toBeInTheDocument()
  })

  it('ne dit rien sans la capacité, et ne demande même pas l’état', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_regenerate_replays: false }) })
    let asked = 0
    server.use(http.get(`${API}/desktop/replay-regen`, () => { asked++; return HttpResponse.json({ ...IDLE, stale: 9 }) }))
    renderWithProviders(<ReplayRegenBanner />)
    await new Promise((r) => setTimeout(r, 50))
    expect(screen.queryByTestId('replay-regen-banner')).not.toBeInTheDocument()
    expect(asked).toBe(0)
  })

  it('ne dit rien quand rien n’est en retard', async () => {
    serveur({ stale: 0 })
    renderWithProviders(<ReplayRegenBanner />)
    await new Promise((r) => setTimeout(r, 50))
    expect(screen.queryByTestId('replay-regen-banner')).not.toBeInTheDocument()
  })

  it('« Plus tard » masque le bandeau pour la session', async () => {
    serveur({ stale: 3 })
    renderWithProviders(<ReplayRegenBanner />)
    fireEvent.click(await screen.findByRole('button', { name: 'Plus tard' }))
    await waitFor(() => expect(screen.queryByTestId('replay-regen-banner')).not.toBeInTheDocument())
    expect(window.sessionStorage.getItem('levelup.replayRegen.later')).toBe('1')
  })

  it('« Régénérer » lance la file, montre la progression, puis Pause et Reprendre', async () => {
    const s = serveur({ stale: 4 })
    renderWithProviders(<ReplayRegenBanner />)
    fireEvent.click(await screen.findByRole('button', { name: 'Régénérer' }))
    expect(await screen.findByText('Régénération des rejeux : 1 sur 4.')).toBeInTheDocument()
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25')
    expect(screen.queryByRole('button', { name: 'Plus tard' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    expect(await screen.findByText('Régénération en pause : 1 sur 4.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reprendre' }))
    expect(await screen.findByText('Régénération des rejeux : 1 sur 4.')).toBeInTheDocument()
    expect(s.commands).toEqual(['start', 'pause', 'resume'])
  })

  it('une file lancée reste visible même après « Plus tard »', async () => {
    useRegenDismissedStore.setState({ dismissed: true })
    serveur({ running: true, total: 4, done: 2, stale: 2 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('Régénération des rejeux : 2 sur 4.')).toBeInTheDocument()
  })

  it('dit que la file compresse les rejeux bruts, avec Pause, en français et en anglais', async () => {
    serveur({ running: true, total: 4, done: 4, compressing: true, compressed: 7 })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('Compression des rejeux sur le disque : 7 faits.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Pause' })).toBeInTheDocument()
    cleanup()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('Compressing replays on disk: 7 done.')).toBeInTheDocument()
  })

  it('dit quand la file attend la fin de la partie', async () => {
    serveur({ running: true, total: 4, done: 2, stale: 2, waiting_for_game: true })
    renderWithProviders(<ReplayRegenBanner />)
    expect(await screen.findByText('En attente de la fin de la partie : 2 sur 4.')).toBeInTheDocument()
  })
})

describe('ReplayRegenCard (Paramètres)', () => {
  beforeEach(() => {
    useRegenDismissedStore.setState({ dismissed: true }) // la carte ignore « Plus tard »
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('utilise le même composant que le bandeau : mêmes boutons, sans « Plus tard »', async () => {
    const s = serveur({ stale: 7 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('7 rejeux à régénérer après la mise à jour.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Plus tard' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Régénérer' }))
    await waitFor(() => expect(s.commands).toEqual(['start']))
  })

  it('rend compte de la compression une fois la file finie : compressés et échecs, singulier et pluriel', async () => {
    serveur({ compressed: 12, compress_failed: 1 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('12 rejeux compressés sur le disque.')).toBeInTheDocument()
    expect(screen.getByText('1 rejeu n’a pas pu être compressé ; il reste lisible.')).toBeInTheDocument()
    cleanup()
    serveur({ compressed: 1, compress_failed: 3 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('1 rejeu compressé sur le disque.')).toBeInTheDocument()
    expect(screen.getByText('3 rejeux n’ont pas pu être compressés ; ils restent lisibles.')).toBeInTheDocument()
    cleanup()
    useAppShellStore.setState({ locale: 'en' })
    serveur({ compressed: 12, compress_failed: 1 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('12 replays compressed on disk.')).toBeInTheDocument()
    expect(screen.getByText('1 replay could not be compressed; it stays readable.')).toBeInTheDocument()
  })

  it('ne dit pas que tout est à jour tant qu il reste des bruts', async () => {
    serveur({ stale: 0, raw: 2 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('2 rejeux à compresser pour libérer de la place sur le disque.')).toBeInTheDocument()
    expect(screen.queryByText('Tous les rejeux sont à jour.')).not.toBeInTheDocument()
  })

  it('dit que tout est à jour', async () => {
    serveur({ stale: 0 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByText('Tous les rejeux sont à jour.')).toBeInTheDocument()
  })

  it('« Recompter » appelle la route de recomptage', async () => {
    const s = serveur({ stale: 2 })
    renderWithProviders(<ReplayRegenCard />)
    await screen.findByText('2 rejeux à régénérer après la mise à jour.')
    const button = screen.getByRole('button', { name: 'Recompter' })
    expect(button).toBeEnabled()
    fireEvent.click(button)
    await waitFor(() => expect(s.commands).toEqual(['rescan']))
    expect(await screen.findByText('Vérification des rejeux…')).toBeInTheDocument()
  })

  it('« Recompter » est désactivé pendant une file et tant que le décompte n’est pas fait', async () => {
    const s = serveur({ running: true, total: 4, done: 1, stale: 3 })
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByRole('button', { name: 'Recompter' })).toBeDisabled()
    s.set({ running: false, checked: false })
    cleanup()
    renderWithProviders(<ReplayRegenCard />)
    expect(await screen.findByRole('button', { name: 'Recompter' })).toBeDisabled()
    expect(s.commands).toEqual([])
  })

  it('le bandeau ne propose pas « Recompter »', async () => {
    useRegenDismissedStore.setState({ dismissed: false })
    serveur({ stale: 3 })
    renderWithProviders(<ReplayRegenBanner />)
    await screen.findByRole('button', { name: 'Régénérer' })
    expect(screen.queryByRole('button', { name: 'Recompter' })).not.toBeInTheDocument()
  })

  it('n’existe pas sans la capacité', () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_regenerate_replays: false }) })
    renderWithProviders(<ReplayRegenCard />)
    expect(screen.queryByText('Rejeux')).not.toBeInTheDocument()
  })
})
