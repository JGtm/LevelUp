/**
 * Tests composant — ReauthBanner (PR-B slice 2).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { ReauthBanner } from './ReauthBanner'

const navigateMock = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigateMock }
})

describe('ReauthBanner', () => {
  afterEach(() => {
    useAppShellStore.setState({ reauthRequired: false, oauthCodeFlowEnabled: false, locale: 'fr' })
    navigateMock.mockReset()
  })

  it('ne rend rien quand reauth_required est false', () => {
    useAppShellStore.setState({ reauthRequired: false })
    const { container } = renderWithProviders(<ReauthBanner />)
    expect(container.querySelector('[role="alert"]')).toBeNull()
  })

  it('affiche la bannière + bouton quand reauth_required est true', () => {
    useAppShellStore.setState({ reauthRequired: true })
    renderWithProviders(<ReauthBanner />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText(/connexion Xbox a expiré/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Rafraîchir/i })).toBeInTheDocument()
  })

  it('rend le message et le bouton en anglais quand locale=en', () => {
    useAppShellStore.setState({ reauthRequired: true, locale: 'en' })
    renderWithProviders(<ReauthBanner />)
    expect(screen.getByText(/The Xbox connection has expired/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Refresh/i })).toBeInTheDocument()
  })

  it('device mode (pas de redirect) : le bouton navigue vers /login', () => {
    useAppShellStore.setState({ reauthRequired: true, oauthCodeFlowEnabled: false })
    renderWithProviders(<ReauthBanner />)
    fireEvent.click(screen.getByRole('button', { name: /Rafraîchir/i }))
    expect(navigateMock).toHaveBeenCalledWith({ to: '/login' })
  })
})
