/**
 * Tests composant — OAuthRefusalBanner (lot L9-9) : le motif du refus d'une reconnexion Xbox,
 * gardé par la racine quand un utilisateur déjà connecté revient de /login?oauth_error=<code>.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { OAuthRefusalBanner } from './OAuthRefusalBanner'

describe('OAuthRefusalBanner', () => {
  afterEach(() => {
    useAppShellStore.setState({ oauthRefusal: null, locale: 'fr' })
  })

  it('ne rend rien sans refus', () => {
    const { container } = renderWithProviders(<OAuthRefusalBanner />)
    expect(container.querySelector('[role="alert"]')).toBeNull()
  })

  it('affiche le message du refus en français', () => {
    useAppShellStore.setState({ oauthRefusal: 'instance_locked' })
    renderWithProviders(<OAuthRefusalBanner />)
    expect(screen.getByRole('alert')).toHaveTextContent('Cette instance est fermée aux nouvelles inscriptions.')
  })

  it('affiche le message du refus en anglais', () => {
    useAppShellStore.setState({ oauthRefusal: 'instance_locked', locale: 'en' })
    renderWithProviders(<OAuthRefusalBanner />)
    expect(screen.getByRole('alert')).toHaveTextContent('This instance is closed to new registrations.')
  })

  it('code inconnu : message générique, jamais le code', () => {
    useAppShellStore.setState({ oauthRefusal: 'code_inconnu' })
    renderWithProviders(<OAuthRefusalBanner />)
    expect(screen.getByRole('alert')).not.toHaveTextContent('code_inconnu')
    expect(screen.getByRole('alert').textContent?.length).toBeGreaterThan(10)
  })

  it('le bouton Fermer efface le motif', () => {
    useAppShellStore.setState({ oauthRefusal: 'state_mismatch' })
    renderWithProviders(<OAuthRefusalBanner />)
    fireEvent.click(screen.getByRole('button', { name: /Fermer/i }))
    expect(useAppShellStore.getState().oauthRefusal).toBeNull()
    expect(screen.queryByRole('alert')).toBeNull()
  })
})
