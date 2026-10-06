/**
 * Tests de la vue Escouade de l'onglet Tendances : bascule de vue (les blocs Solo disparaissent,
 * le sélecteur apparaît, type de partie, horizon et pas sont gardés), requête non lancée sans
 * coéquipier, corps de la requête, clé de requête, option « Composition stricte », grille,
 * sélection partagée avec la page Escouade (stockage par joueur) et enregistrement de l'escouade.
 *
 * ECharts est simulé (jsdom n'a pas de canvas) : le double expose l'option reçue.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'

import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { response } from './tendances.fixture'
import { compositionKey } from './queries'
import { squadResponse } from './tendancesSquad.fixture'
import { TendancesTab } from './TendancesTab'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useParams: () => ({ playerSlug: 'jgtm' }) }
})

const post = vi.fn()
const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      post: (...args: unknown[]) => post(...args),
      get: (...args: unknown[]) => get(...args),
    },
  }
})

const FIELD_MAPPINGS = {
  title_slug: 'halo_infinite',
  schema_version: 1,
  locale: 'fr',
  fields: { win_rate: { label: 'Taux de victoire' }, kda: { label: 'FDA' } },
}

beforeEach(() => {
  localStorage.clear()
  post.mockReset()
  get.mockReset()
  get.mockImplementation(async (path: string) => {
    if (path.includes('career/encounters')) {
      return {
        teammates: [
          { gamertag: 'Alice', xuid: 'x-alice', match_count: 20 },
          { gamertag: 'Bob', xuid: 'x-bob', match_count: 10 },
        ],
      }
    }
    if (path.includes('squads')) return { squads: [] }
    return FIELD_MAPPINGS
  })
  post.mockImplementation(async (_path: string, body: { view?: string }) =>
    body.view === 'squad' ? squadResponse() : response(),
  )
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    isBootstrapped: true,
  })
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const vue = () => screen.getByRole('group', { name: 'Vue' })

async function ouvrirEscouade() {
  renderWithProviders(<TendancesTab />)
  await screen.findByTestId('tendances-matrix')
  fireEvent.click(within(vue()).getByRole('button', { name: 'Escouade' }))
}

async function choisir(gamertag: string) {
  const champ = await screen.findByPlaceholderText(/coéquipiers/i)
  fireEvent.focus(champ)
  fireEvent.click(await screen.findByText(gamertag))
}

describe('clé de requête', () => {
  it('compositionKey : gamertags triés et joints, indépendants de l’ordre de sélection', () => {
    expect(compositionKey(['Bob', 'Alice'])).toBe('Alice,Bob')
    expect(compositionKey(['Alice', 'Bob'])).toBe(compositionKey(['Bob', 'Alice']))
    expect(compositionKey([])).toBe('')
    expect(compositionKey(undefined)).toBe('')
  })

  it('porte la composition et l’option stricte : une autre escouade ou règle = une autre clé', () => {
    const cle = (comp: string, stricte: boolean) =>
      queryKeys.trends('jgtm', 'halo_infinite', 'squad', '', 'fr', comp, stricte)
    expect(cle('Alice', true)).not.toEqual(cle('Alice,Bob', true))
    expect(cle('Alice', true)).not.toEqual(cle('Alice', false))
    expect(cle('Alice', true)).toEqual(cle('Alice', true))
    expect(cle('Alice', true)[3]).toBe('squad')
  })
})

describe('TendancesTab — bascule de vue', () => {
  it('Solo par défaut : bascule à gauche du type de partie, Solo pressé', async () => {
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    expect(within(vue()).getByRole('button', { name: 'Solo' })).toHaveAttribute('aria-pressed', 'true')
    const filtre = screen.getByLabelText('Type de partie')
    expect(vue().compareDocumentPosition(filtre) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(screen.queryByTestId('tendances-squad-picker')).toBeNull()
  })

  it('Escouade sans coéquipier : le sélecteur, l’invite, aucune requête Escouade', async () => {
    await ouvrirEscouade()
    expect(await screen.findByTestId('tendances-squad-picker')).toBeInTheDocument()
    expect(screen.getByText('Choisis une escouade pour voir ses tendances.')).toBeInTheDocument()
    expect(screen.queryByTestId('tendances-matrix')).toBeNull()
    expect(screen.queryByLabelText('Composition stricte')).toBeNull()
    expect(post.mock.calls.filter((c) => c[1]?.view === 'squad')).toHaveLength(0)
  })

  it('choisir un coéquipier lance la requête Escouade (composition stricte par défaut)', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    await screen.findByTestId('tendances-matrix')
    expect(post).toHaveBeenLastCalledWith(
      '/players/jgtm/pages/trends',
      {
        view: 'squad',
        game_type: '',
        selected_gamertags: ['Alice'],
        exact_composition: true,
        locale: 'fr',
      },
      undefined,
      expect.anything(),
    )
  })

  it('décocher « Composition stricte » refait la requête avec l’option désactivée', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    await screen.findByTestId('tendances-matrix')
    const case_ = screen.getByRole('checkbox', { name: 'Composition stricte' })
    expect(case_).toBeChecked()
    fireEvent.click(case_)
    await waitFor(() =>
      expect(post.mock.calls.at(-1)?.[1]).toMatchObject({
        view: 'squad',
        exact_composition: false,
      }),
    )
  })

  it('vue Escouade : les blocs Solo disparaissent, la grille de six graphiques apparaît', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    const evolution = await screen.findByTestId('tendances-evolution')
    expect(screen.queryByText('Victoires et défaites')).toBeNull()
    expect(screen.queryByTestId('tendances-mix')).toBeNull()
    const grille = within(evolution).getByTestId('tendances-grid-squad')
    expect(within(grille).getAllByTestId(/^tendances-chart-/)).toHaveLength(6)
  })

  it('matrice à deux groupes : « Escouade » puis « Membres »', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    const matrice = await screen.findByTestId('tendances-matrix')
    const groupes = within(matrice)
      .getAllByRole('heading', { level: 3 })
      .map((h) => h.textContent)
    expect(groupes).toEqual(['Escouade', 'Membres'])
    // Une ligne à variante s'écrit « libellé · gamertag », une ligne propre à la vue a son libellé.
    await waitFor(() => {
      expect(matrice.textContent).toContain('FDA · Alice')
      expect(matrice.textContent).toContain('Taux de victoire sans coéquipier suivi')
      expect(matrice.textContent).toContain('Part des frags de l')
    })
  })

  it('garde le type de partie, l’horizon et le pas en changeant de vue', async () => {
    renderWithProviders(<TendancesTab />)
    const barre = await screen.findByTestId('tendances-horizon-bar')
    fireEvent.click(within(barre).getByRole('button', { name: '30 j' }))
    const pas = screen.getByRole('group', { name: 'Pas de temps' })
    fireEvent.click(within(pas).getByRole('button', { name: 'Par semaine' }))
    fireEvent.change(await screen.findByLabelText('Type de partie'), {
      target: { value: 'ranked_slayer' },
    })
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))

    fireEvent.click(within(vue()).getByRole('button', { name: 'Escouade' }))
    await choisir('Alice')
    const barreEscouade = await screen.findByTestId('tendances-horizon-bar')
    expect(within(barreEscouade).getByRole('button', { name: '30 j' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(
      within(screen.getByRole('group', { name: 'Pas de temps' })).getByRole('button', {
        name: 'Par semaine',
      }),
    ).toHaveAttribute('aria-pressed', 'true')
    expect((screen.getByLabelText('Type de partie') as HTMLSelectElement).value).toBe('ranked_slayer')
    expect(post.mock.calls.at(-1)?.[1]).toMatchObject({
      view: 'squad',
      game_type: 'ranked_slayer',
    })
  })

  it('retour en Solo : le sélecteur disparaît, la matrice Solo revient', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    await screen.findByTestId('tendances-matrix')
    fireEvent.click(within(vue()).getByRole('button', { name: 'Solo' }))
    await waitFor(() => expect(screen.queryByTestId('tendances-squad-picker')).toBeNull())
    expect(await screen.findByTestId('tendances-matrix-results')).toBeInTheDocument()
  })

  it('escouade sans indicateur : l’état vide de la page, le sélecteur reste', async () => {
    post.mockImplementation(async (_path: string, body: { view?: string }) =>
      body.view === 'squad' ? squadResponse({ indicators: [] }) : response(),
    )
    await ouvrirEscouade()
    await choisir('Alice')
    expect(await screen.findByText('Aucune tendance à afficher')).toBeInTheDocument()
    expect(screen.getByTestId('tendances-squad-picker')).toBeInTheDocument()
  })

  it('anglais : bascule et invite', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    fireEvent.click(within(screen.getByRole('group', { name: 'View' })).getByRole('button', { name: 'Squad' }))
    expect(await screen.findByText('Choose a squad to see its trends.')).toBeInTheDocument()
  })
})

describe('TendancesTab — sélection partagée avec la page Escouade', () => {
  it('une sélection posée par la page Escouade est celle de la vue Escouade et part dans la requête', async () => {
    localStorage.setItem('squad-teammates-jgtm', JSON.stringify(['Alice', 'Bob']))
    await ouvrirEscouade()
    await screen.findByTestId('tendances-matrix')
    expect(post.mock.calls.at(-1)?.[1]).toMatchObject({
      view: 'squad',
      selected_gamertags: ['Alice', 'Bob'],
    })
  })

  it('choisir un coéquipier écrit la clé de la page Escouade', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    await screen.findByTestId('tendances-matrix')
    expect(JSON.parse(localStorage.getItem('squad-teammates-jgtm') ?? 'null')).toEqual(['Alice'])
  })

  it('décocher « Composition stricte » écrit false', async () => {
    await ouvrirEscouade()
    await choisir('Alice')
    await screen.findByTestId('tendances-matrix')
    fireEvent.click(screen.getByRole('checkbox', { name: 'Composition stricte' }))
    expect(localStorage.getItem('squad-exact-composition-jgtm')).toBe('false')
  })

  it('une composition stricte stockée à false est lue au montage', async () => {
    localStorage.setItem('squad-teammates-jgtm', JSON.stringify(['Alice']))
    localStorage.setItem('squad-exact-composition-jgtm', 'false')
    await ouvrirEscouade()
    await screen.findByTestId('tendances-matrix')
    expect(screen.getByRole('checkbox', { name: 'Composition stricte' })).not.toBeChecked()
    expect(post.mock.calls.at(-1)?.[1]).toMatchObject({ view: 'squad', exact_composition: false })
  })
})

describe('TendancesTab — enregistrer l’escouade', () => {
  it('le bouton est actif avec les membres de la réponse et crée l’escouade avec les coéquipiers', async () => {
    localStorage.setItem('squad-teammates-jgtm', JSON.stringify(['Alice', 'Bob']))
    await ouvrirEscouade()
    await screen.findByTestId('tendances-matrix')
    fireEvent.focus(await screen.findByRole('textbox'))
    const bouton = await screen.findByRole('button', { name: 'Enregistrer la compo' })
    await waitFor(() => expect(bouton).toBeEnabled())
    fireEvent.click(bouton)
    await waitFor(() => {
      const creation = post.mock.calls.find((c) => String(c[0]).includes('squads'))
      expect(creation?.[1]).toMatchObject({
        created_by: 'jgtm',
        members: [
          { xuid: 'x-alice', gamertag: 'Alice' },
          { xuid: 'x-bob', gamertag: 'Bob' },
        ],
      })
    })
  })
})
