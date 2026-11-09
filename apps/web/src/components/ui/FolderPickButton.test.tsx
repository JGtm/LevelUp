/**
 * Tests composant — FolderPickButton : offert par la capacité `can_pick_folder`, jamais par autre chose ;
 * le chemin choisi arrive à `onPick` tel quel, une annulation ou un échec laissent le champ tranquille.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { FolderPickButton } from './FolderPickButton'

function setCapabilities(overrides: Parameters<typeof testCapabilities>[0]) {
  act(() => {
    useAppShellStore.setState({ capabilities: testCapabilities(overrides) })
  })
}

describe('FolderPickButton', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    act(() => {
      useAppShellStore.setState({ locale: 'fr', capabilities: null })
    })
  })

  it('n’existe pas sans la capacité (serveur, démo, avant hydratation)', () => {
    for (const overrides of [{}, { can_pick_folder: false }] as const) {
      setCapabilities(overrides)
      const { unmount } = renderWithProviders(<FolderPickButton value="" onPick={vi.fn()} />)
      expect(screen.queryByRole('button')).not.toBeInTheDocument()
      unmount()
    }
  })

  it('n’est pas offert d’après le profil : seule la capacité compte', () => {
    setCapabilities({ deployment: 'desktop' })
    renderWithProviders(<FolderPickButton value="" onPick={vi.fn()} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it.each([
    ['fr', 'Parcourir…'],
    ['en', 'Browse…'],
  ] as const)(
    'avec la capacité (%s) : un bouton qui envoie la valeur du champ et rend le chemin choisi',
    async (locale, label) => {
      useAppShellStore.setState({ locale })
      setCapabilities(desktopCapabilities())
      const invoke = vi.fn().mockResolvedValue('Z:/choisi')
      vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
      const onPick = vi.fn()
      renderWithProviders(<FolderPickButton value="Y:/courant" onPick={onPick} />)

      fireEvent.click(screen.getByRole('button', { name: label }))

      await waitFor(() => expect(onPick).toHaveBeenCalledWith('Z:/choisi'))
      expect(invoke).toHaveBeenCalledWith('pick_folder', { initial: 'Y:/courant' })
    },
  )

  it('une annulation n’appelle pas onPick', async () => {
    setCapabilities(desktopCapabilities())
    const invoke = vi.fn().mockResolvedValue(null)
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    const onPick = vi.fn()
    renderWithProviders(<FolderPickButton value="" onPick={onPick} />)

    fireEvent.click(screen.getByRole('button', { name: 'Parcourir…' }))

    await waitFor(() => expect(invoke).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Parcourir…' })).toBeEnabled())
    expect(onPick).not.toHaveBeenCalled()
  })

  it('un échec de la coquille est journalisé et n’appelle pas onPick', async () => {
    setCapabilities(desktopCapabilities())
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke: vi.fn().mockRejectedValue('refusé') })
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    const onPick = vi.fn()
    renderWithProviders(<FolderPickButton value="" onPick={onPick} />)

    fireEvent.click(screen.getByRole('button', { name: 'Parcourir…' }))

    await waitFor(() => expect(error).toHaveBeenCalledTimes(1))
    expect(onPick).not.toHaveBeenCalled()
  })

  it('reste désactivé pendant que la boîte est ouverte : pas de seconde boîte', async () => {
    setCapabilities(desktopCapabilities())
    let finish: (path: string | null) => void = () => {}
    const invoke = vi.fn().mockReturnValue(
      new Promise<string | null>((resolve) => {
        finish = resolve
      }),
    )
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    renderWithProviders(<FolderPickButton value="" onPick={vi.fn()} />)

    const button = screen.getByRole('button', { name: 'Parcourir…' })
    fireEvent.click(button)
    await waitFor(() => expect(button).toBeDisabled())
    fireEvent.click(button)
    expect(invoke).toHaveBeenCalledTimes(1)

    await act(async () => {
      finish(null)
    })
    await waitFor(() => expect(button).toBeEnabled())
  })

  it('respecte disabled', () => {
    setCapabilities(desktopCapabilities())
    renderWithProviders(<FolderPickButton value="" onPick={vi.fn()} disabled />)
    expect(screen.getByRole('button', { name: 'Parcourir…' })).toBeDisabled()
  })
})
