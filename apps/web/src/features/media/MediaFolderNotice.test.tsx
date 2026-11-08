/**
 * Tests — MediaFolderNotice : trois états (aucun bandeau, dossier inaccessible, retrait massif
 * en attente), clic de confirmation, FR et EN.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { MediaFolderNotice } from './MediaFolderNotice'
import * as queries from './missingQueries'

vi.mock('./missingQueries', () => ({
  useConfirmBulkMissing: vi.fn(),
}))

const confirm = vi.fn()

function setMutation(state: { isPending?: boolean; isSuccess?: boolean } = {}) {
  vi.mocked(queries.useConfirmBulkMissing).mockReturnValue({
    mutate: confirm,
    isPending: state.isPending ?? false,
    isSuccess: state.isSuccess ?? false,
  } as unknown as ReturnType<typeof queries.useConfirmBulkMissing>)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  confirm.mockReset()
  setMutation()
})

describe('MediaFolderNotice', () => {
  it('ne rend rien quand le dossier va bien', () => {
    const { container } = renderWithProviders(
      <MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={0} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('dossier inaccessible : message seul, sans bouton', () => {
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable bulkPending={0} />)
    expect(screen.getByRole('status')).toHaveTextContent(/dossier des captures est inaccessible/)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('dossier inaccessible : passe avant un retrait massif en attente', () => {
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable bulkPending={30} />)
    expect(screen.getByRole('status')).toHaveTextContent(/inaccessible/)
    expect(screen.queryByRole('button', { name: 'Confirmer le retrait' })).not.toBeInTheDocument()
  })

  it('retrait massif : le nombre et le bouton de confirmation', () => {
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={30} />)
    expect(screen.getByRole('status')).toHaveTextContent(/30 médias n'ont plus de fichier/)
    expect(screen.getByRole('status')).toHaveTextContent(/Rien n'est masqué tant que ce retrait n'est pas confirmé/)
    fireEvent.click(screen.getByRole('button', { name: 'Confirmer le retrait' }))
    expect(confirm).toHaveBeenCalledTimes(1)
  })

  it('singulier pour un seul média', () => {
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={1} />)
    expect(screen.getByRole('status')).toHaveTextContent(/1 média n'a plus de fichier/)
  })

  it('le bouton est figé pendant et après la confirmation', () => {
    setMutation({ isPending: true })
    const first = renderWithProviders(
      <MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={30} />,
    )
    expect(screen.getByRole('button', { name: 'Confirmer le retrait' })).toBeDisabled()
    first.unmount()
    setMutation({ isSuccess: true })
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={30} />)
    expect(screen.getByRole('button', { name: 'Confirmer le retrait' })).toBeDisabled()
  })

  it('anglais : message et bouton sans résidu français', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable={false} bulkPending={30} />)
    expect(screen.getByRole('status')).toHaveTextContent(/30 media have no file left/)
    expect(screen.getByRole('button', { name: 'Confirm the removal' })).toBeInTheDocument()
  })

  it('anglais : dossier inaccessible', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<MediaFolderNotice playerSlug="p" rootUnavailable bulkPending={0} />)
    expect(screen.getByRole('status')).toHaveTextContent(/captures folder is unreachable/)
  })
})
