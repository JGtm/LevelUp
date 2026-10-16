/**
 * Tests — UploadButton : refus d'un envoi quand le serveur est occupé (503 upload_busy).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { UploadButton } from './UploadButton'

const { toastError, toastSuccess } = vi.hoisted(() => ({ toastError: vi.fn(), toastSuccess: vi.fn() }))
vi.mock('sonner', () => ({ toast: { error: toastError, success: toastSuccess } }))

function pickFile(container: HTMLElement) {
  const input = container.querySelector('input[type="file"]') as HTMLInputElement
  const file = new File(['x'], 'clip.mp4', { type: 'video/mp4' })
  fireEvent.change(input, { target: { files: [file] } })
}

describe('UploadButton — serveur occupé', () => {
  beforeEach(() => {
    toastError.mockClear()
    useAppShellStore.setState({ demoMode: false, locale: 'fr' })
    server.use(
      http.post('/api/v1/players/:slug/media/upload', () =>
        HttpResponse.json(
          { code: 'upload_busy', message: 'serveur occupé par d\'autres envois', retryable: true },
          { status: 503, headers: { 'Retry-After': '30' } },
        ),
      ),
    )
  })
  afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

  it('explique en français que le serveur est occupé et invite à réessayer', async () => {
    const { container } = renderWithProviders(<UploadButton playerSlug="p1" />)
    pickFile(container)
    await waitFor(() => expect(toastError).toHaveBeenCalled())
    expect(toastError.mock.calls[0][1]).toEqual({
      description: "Le serveur traite déjà d'autres envois. Réessaie dans quelques instants.",
    })
  })

  it('explique en anglais quand la langue est EN', async () => {
    useAppShellStore.setState({ locale: 'en' })
    const { container } = renderWithProviders(<UploadButton playerSlug="p1" />)
    pickFile(container)
    await waitFor(() => expect(toastError).toHaveBeenCalled())
    expect(toastError.mock.calls[0][1]).toEqual({
      description: 'The server is already processing other uploads. Try again in a moment.',
    })
  })
})

describe('UploadButton — contenu non concordant avec l’extension (415)', () => {
  beforeEach(() => {
    toastError.mockClear()
    useAppShellStore.setState({ demoMode: false, locale: 'fr' })
    server.use(
      http.post('/api/v1/players/:slug/media/upload', () =>
        HttpResponse.json(
          { code: 'upload_content_mismatch', message: 'file content does not match its extension', retryable: false },
          { status: 415 },
        ),
      ),
    )
  })
  afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

  it('explique en français que le fichier n’est pas un média reconnu', async () => {
    const { container } = renderWithProviders(<UploadButton playerSlug="p1" />)
    pickFile(container)
    await waitFor(() => expect(toastError).toHaveBeenCalled())
    expect(toastError.mock.calls[0][1]).toEqual({
      description: "Un des fichiers n'est pas une vidéo ou une image valide : son contenu ne correspond pas à son extension.",
    })
  })

  it('explique en anglais quand la langue est EN', async () => {
    useAppShellStore.setState({ locale: 'en' })
    const { container } = renderWithProviders(<UploadButton playerSlug="p1" />)
    pickFile(container)
    await waitFor(() => expect(toastError).toHaveBeenCalled())
    expect(toastError.mock.calls[0][1]).toEqual({
      description: 'One of the files is not a valid video or image: its content does not match its extension.',
    })
  })
})
