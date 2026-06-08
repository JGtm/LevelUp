/**
 * Tests composant — SetPasswordCard (PR-C).
 */
import { describe, it, expect, afterEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { SetPasswordCard } from './SetPasswordCard'

describe('SetPasswordCard', () => {
  afterEach(() => useAppShellStore.setState({ hasPassword: false }))

  it('refuse si les mots de passe ne correspondent pas', () => {
    const { container } = renderWithProviders(<SetPasswordCard />)
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Different9' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    expect(screen.getByText(/ne correspondent pas/i)).toBeInTheDocument()
  })

  it('enregistre et affiche la confirmation', async () => {
    server.use(http.post('/api/v1/auth/password', () => new HttpResponse(null, { status: 204 })))
    const { container } = renderWithProviders(<SetPasswordCard />)
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Abcd1234' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    await waitFor(() => {
      expect(screen.getByText(/enregistré/i)).toBeInTheDocument()
    })
  })

  it('titre « Changer » si un mot de passe existe déjà', () => {
    useAppShellStore.setState({ hasPassword: true })
    renderWithProviders(<SetPasswordCard />)
    expect(screen.getByText(/Changer ton mot de passe/i)).toBeInTheDocument()
  })
})
