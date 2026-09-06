/**
 * La GRILLE des cartes de l'onglet Tactique, rendue depuis une réponse de contrat.
 *
 * Ce que ces tests cadenassent :
 *   - la grille rend une vignette par carte, dans l'ordre décidé par la logique ;
 *   - une carte SOUS LE PLANCHER est désaturée, DÉSACTIVÉE (`aria-disabled`), et porte sa
 *     raison en clair — jamais une vignette qui promet une ouverture qui n'arrivera pas ;
 *   - une carte ouvrable est un bouton qui DIT ce qu'il fait, et son clic écrit la carte
 *     choisie dans l'URL (aucun lien mort vers la route de la phase 5, qui n'existe pas) ;
 *   - aucune carte -> `EmptyState`, jamais une grille vide muette ;
 *   - le pied de carte sert la couverture ET la phrase du plancher ;
 *   - LE PERIMETRE : la barre produit un contexte de filtre, `/filters/match-ids` le
 *     resout, et la grille poste les `match_id` obtenus (phase 4 bis). Sans ces
 *     assertions, une grille qui ignorerait le filtre resterait verte.
 *
 * La garde de capability (`replay`) est testée là où elle vit : `TacticalTab.test.tsx`
 * (la porte de route) et `features/ascension/AscensionLayout.test.tsx` (la porte d'onglet).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { FilterContextInput, TacticalMapsPage } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { TacticalPage } from './TacticalPage'

const navigate = vi.fn()
let searchCourant: Record<string, unknown> = {}

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
    useParams: () => ({ playerSlug: 'JGtm', titleSlug: 'halo_infinite' }),
    useSearch: () => searchCourant,
  }
})

const get = vi.fn()
const post = vi.fn()
const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: (path: string) => get(path),
      post: (path: string, body: unknown) => post(path, body),
      getBlob: (path: string) => getBlob(path),
    },
  }
})

/** Les match_id que `/filters/match-ids` rend par defaut dans ces tests. */
const PERIMETRE = ['m1', 'm2', 'm3']

/** Le corps poste a `/tactical/maps`, ou `undefined` si la grille n'a rien demande. */
function corpsGrille(): { match_ids: string[]; coequipiers: string[] } | undefined {
  const appel = post.mock.calls.find((c) => (c[0] as string).endsWith('/tactical/maps'))
  return appel?.[1] as { match_ids: string[]; coequipiers: string[] } | undefined
}

/** Le contexte envoye a `/filters/match-ids`. */
function contexteResolu(): FilterContextInput | undefined {
  const appel = post.mock.calls.find((c) => (c[0] as string).endsWith('/filters/match-ids'))
  return appel?.[1] as FilterContextInput | undefined
}

const page: TacticalMapsPage = {
  plancher_matchs: 10,
  cartes: [
    {
      map_id: 'streets',
      map_name: 'Streets',
      map_name_fr: 'Ruelles',
      matchs: 24,
      victoires: 14,
      defaites: 9,
      sous_plancher: false,
    },
    {
      map_id: 'aquarius',
      map_name: 'Aquarius',
      map_name_fr: 'Aquarius',
      matchs: 9,
      victoires: 4,
      defaites: 5,
      sous_plancher: true,
    },
  ],
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  searchCourant = {}
  localStorage.clear()
  navigate.mockReset()
  get.mockReset()
  // La liste des coequipiers proposes (avec leur xuid) : c'est elle qui traduit une
  // composition d'URL en identifiants.
  get.mockResolvedValue({
    teammates: [{ gamertag: 'Ami', xuid: 'xuid(42)', match_count: 30, as_teammate: 30, as_enemy: 0, avg_kda: null }],
    enemies: [],
    total: 1,
  })
  post.mockReset()
  post.mockImplementation((path: string) => {
    if (path.endsWith('/filters/match-ids')) return Promise.resolve({ match_ids: PERIMETRE })
    if (path.endsWith('/tactical/maps')) return Promise.resolve(page)
    return Promise.reject(new Error(`appel inattendu : ${path}`))
  })
  getBlob.mockReset()
  // Défaut : la carte n'a pas de fond. La grille doit s'afficher quand même.
  getBlob.mockRejectedValue(new Error('pas de fond'))
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('TacticalPage — la grille des cartes', () => {
  it('rend une vignette par carte, la plus jouée en tête', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    const aquarius = screen.getByTestId('tactical-map-aquarius')
    expect(streets).toBeInTheDocument()
    expect(aquarius).toBeInTheDocument()
    // L'ordre du DOM EST l'ordre affiché : Streets (24 matchs) avant Aquarius (9).
    expect(streets.compareDocumentPosition(aquarius) & Node.DOCUMENT_POSITION_FOLLOWING)
      .toBeTruthy()
    // Le nom FR vient du contrat, pas du nom canonique.
    expect(streets.textContent).toContain('Ruelles')
    expect(streets.textContent).toContain('24 matchs')
  })

  it('sert le compte de victoires ET de défaites — jamais un taux seul', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets.textContent).toContain('14 victoires')
    expect(streets.textContent).toContain('9 défaites')
  })

  it('carte SOUS LE PLANCHER : désaturée, désactivée, avec sa raison en clair', async () => {
    renderWithProviders(<TacticalPage />)
    const aquarius = await screen.findByTestId('tactical-map-aquarius')
    expect(aquarius).toHaveAttribute('aria-disabled', 'true')
    expect(aquarius).toBeDisabled()
    // Désaturation par des utilitaires SANS couleur : aucun token détourné.
    expect(aquarius.className).toContain('grayscale')
    expect(aquarius.className).toContain('opacity-60')
    expect(screen.getByTestId('tactical-map-plancher-aquarius').textContent).toContain(
      '9 matchs sur 10 requis',
    )
  })

  it('carte ouvrable : bouton actif qui DIT ce qu’il fait', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets).not.toBeDisabled()
    expect(streets).toHaveAttribute('aria-label', 'Sélectionner Ruelles')
    expect(streets).toHaveAttribute('aria-pressed', 'false')
  })

  it('le clic écrit la carte choisie dans l’URL — aucun lien mort', async () => {
    renderWithProviders(<TacticalPage />)
    fireEvent.click(await screen.findByTestId('tactical-map-streets'))
    expect(navigate).toHaveBeenCalledTimes(1)
    const arg = navigate.mock.calls[0][0] as {
      search: (p: Record<string, unknown>) => Record<string, unknown>
    }
    expect(arg.search({}).carte).toBe('streets')
  })

  it('une carte sous le plancher ne navigue nulle part', async () => {
    renderWithProviders(<TacticalPage />)
    fireEvent.click(await screen.findByTestId('tactical-map-aquarius'))
    expect(navigate).not.toHaveBeenCalled()
  })

  it('la carte sélectionnée dans l’URL est marquée comme telle', async () => {
    searchCourant = { carte: 'streets' }
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets).toHaveAttribute('aria-pressed', 'true')
    expect(streets.textContent).toContain('Carte sélectionnée')
  })

  it('pied de carte : la couverture ET la phrase du plancher', async () => {
    renderWithProviders(<TacticalPage />)
    const couverture = await screen.findByTestId('tactical-couverture')
    expect(couverture.textContent).toContain('2 cartes jouées')
    expect(couverture.textContent).toContain('33 matchs')
    expect(screen.getByText(/à partir de 10 matchs/)).toBeInTheDocument()
  })

  it('aucune carte : état vide explicite, jamais une grille muette', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: PERIMETRE })
        : Promise.resolve({ cartes: [], plancher_matchs: 10 }),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-couverture')).toBeNull()
  })

  it('cartes nulles au contrat (slice Go vide) : même état vide, aucun plantage', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: PERIMETRE })
        : Promise.resolve({ cartes: null, plancher_matchs: 10 }),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
  })

  it('lecture en échec : on le dit, on ne rend pas une grille vide', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: PERIMETRE })
        : Promise.reject(new Error('503')),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByTestId('tactical-erreur')).toBeInTheDocument()
    expect(screen.queryByText('Aucune carte jouée')).toBeNull()
  })

  // ─── LE PERIMETRE (phase 4 bis) ───────────────────────────────────────────────────
  //
  // La barre L2 produit un contexte de filtre, `/filters/match-ids` le resout sur la
  // base JOUEUR, et la grille poste les `match_id`. Sans ces assertions, une grille qui
  // enverrait une liste vide — ou qui ignorerait la session epinglee — resterait verte.

  it('poste a la grille les match_id obtenus de la resolution', async () => {
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')

    const appelGrille = post.mock.calls.find((c) => (c[0] as string).endsWith('/tactical/maps'))
    expect(appelGrille?.[0]).toBe('/players/JGtm/tactical/maps')
    expect(corpsGrille()?.match_ids).toEqual(PERIMETRE)
  })

  it('une session epinglee part en filter_mode « sessions », avec son label', async () => {
    searchCourant = { ses: 'Session du 3 mars' }
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')

    const ctx = contexteResolu()
    expect(ctx?.filter_mode).toBe('sessions')
    expect(ctx?.sessions?.picked_sessions).toEqual(['Session du 3 mars'])
  })

  it('sans session : filter_mode « period », et les bornes de la barre', async () => {
    searchCourant = { de: '2026-01-01', a: '2026-02-01', pl: 'Ranked Arena', md: 'Slayer' }
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')

    const ctx = contexteResolu()
    expect(ctx?.filter_mode).toBe('period')
    expect(ctx?.period).toEqual({ start_date: '2026-01-01', end_date: '2026-02-01' })
    expect(ctx?.cascade?.playlists).toEqual(['Ranked Arena'])
    expect(ctx?.cascade?.modes).toEqual(['Slayer'])
  })

  it('la vue solo/escouade descend en match_context', async () => {
    searchCourant = { vue: 'squad' }
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    expect(contexteResolu()?.match_context).toBe('squad')
  })

  it('la composition part en XUIDS, jamais en gamertags', async () => {
    searchCourant = { eq: 'Ami' }
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    expect(corpsGrille()?.coequipiers).toEqual(['xuid(42)'])
  })

  it('un coequipier introuvable ARRETE la grille — jamais un perimetre elargi', async () => {
    searchCourant = { eq: 'Inconnu' }
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Coéquipier introuvable')).toBeInTheDocument()
    expect(corpsGrille()).toBeUndefined()
  })

  // ─── W2 — LE CHEMIN NOMINAL DU FOND ────────────────────────────────────────────────
  //
  // Tous les autres tests doublent `getBlob` en REJET : le cas où une image existe
  // n'était joué nulle part, et `<img src={fond ?? ''}>` — une icône d'image cassée sur
  // chaque vignette sans fond — serait passé.

  it('carte AVEC un fond : la vignette porte l’image, demandée à la bonne URL', async () => {
    const urlObjet = 'blob:tactique/streets'
    const creerURL = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue(urlObjet)
    getBlob.mockResolvedValue(new Blob(['png']))
    try {
      renderWithProviders(<TacticalPage />)
      const img = await screen.findByTestId('tactical-map-fond-streets')
      expect(img).toHaveAttribute('src', urlObjet)
      expect(getBlob).toHaveBeenCalledWith(
        '/players/JGtm/tactical/streets/background.png',
      )
    } finally {
      creerURL.mockRestore()
    }
  })

  it('carte SANS fond : aucune image, jamais une icône cassée', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(screen.queryByTestId('tactical-map-fond-streets')).toBeNull()
    expect(streets.querySelector('img')).toBeNull()
  })

  it('en anglais, les libellés et le nom canonique', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets.textContent).toContain('Streets')
    expect(streets.textContent).toContain('24 matches')
  })
})
