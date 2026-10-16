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
  afterEach(() => useAppShellStore.setState({ hasPassword: false, locale: 'fr' }))

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

  it('refus 429 (identifiant bloqué ou limite de débit) : message clair', async () => {
    server.use(
      http.post('/api/v1/auth/password', () =>
        new HttpResponse('<html><body><h1>429 Too Many Requests</h1></body></html>', {
          status: 429,
          headers: { 'Content-Type': 'text/html' },
        }),
      ),
    )
    const { container } = renderWithProviders(<SetPasswordCard />)
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Abcd1234' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    expect(
      await screen.findByText('Trop de tentatives de connexion. Réessayez dans 15 minutes.'),
    ).toBeInTheDocument()
  })

  it('titre « Changer » si un mot de passe existe déjà', () => {
    useAppShellStore.setState({ hasPassword: true })
    renderWithProviders(<SetPasswordCard />)
    expect(screen.getByText(/Changer ton mot de passe/i)).toBeInTheDocument()
  })

  it('compte avec mot de passe : le mot de passe actuel est demandé et envoyé', async () => {
    let body: unknown = null
    server.use(
      http.post('/api/v1/auth/password', async ({ request }) => {
        body = await request.json()
        return new HttpResponse(null, { status: 204 })
      }),
    )
    useAppShellStore.setState({ hasPassword: true })
    const { container } = renderWithProviders(<SetPasswordCard />)
    expect(screen.getByLabelText('Mot de passe actuel')).toBeInTheDocument()
    expect(screen.getByLabelText('Nouveau mot de passe')).toBeInTheDocument()
    fireEvent.change(container.querySelector('#set-pwd-current')!, { target: { value: 'Ancien123' } })
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Abcd1234' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    await waitFor(() => {
      expect(screen.getByText('Mot de passe enregistré.')).toBeInTheDocument()
    })
    expect(body).toEqual({ password: 'Abcd1234', current_password: 'Ancien123' })
  })

  it('compte Xbox sans mot de passe : pas de champ « actuel », premier mot de passe seul', async () => {
    let body: unknown = null
    server.use(
      http.post('/api/v1/auth/password', async ({ request }) => {
        body = await request.json()
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { container } = renderWithProviders(<SetPasswordCard />)
    expect(container.querySelector('#set-pwd-current')).toBeNull()
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Abcd1234' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    await waitFor(() => {
      expect(screen.getByText('Mot de passe enregistré.')).toBeInTheDocument()
    })
    expect(body).toEqual({ password: 'Abcd1234' })
  })

  it('mot de passe actuel faux : message clair', async () => {
    server.use(
      http.post('/api/v1/auth/password', () =>
        HttpResponse.json(
          { code: 'invalid_current_password', message: 'current password is incorrect', retryable: false },
          { status: 403 },
        ),
      ),
    )
    useAppShellStore.setState({ hasPassword: true })
    const { container } = renderWithProviders(<SetPasswordCard />)
    fireEvent.change(container.querySelector('#set-pwd-current')!, { target: { value: 'Faux12345' } })
    fireEvent.change(container.querySelector('#set-pwd')!, { target: { value: 'Abcd1234' } })
    fireEvent.change(container.querySelector('#set-pwd-confirm')!, { target: { value: 'Abcd1234' } })
    fireEvent.click(screen.getByRole('button', { name: /Enregistrer/i }))
    expect(await screen.findByText('Mot de passe actuel incorrect.')).toBeInTheDocument()
  })

  it('libellés en anglais quand la langue est l\'anglais', () => {
    useAppShellStore.setState({ hasPassword: true, locale: 'en' })
    renderWithProviders(<SetPasswordCard />)
    expect(screen.getByText('Change your password')).toBeInTheDocument()
    expect(screen.getByLabelText('Current password')).toBeInTheDocument()
    expect(screen.getByLabelText('New password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})
