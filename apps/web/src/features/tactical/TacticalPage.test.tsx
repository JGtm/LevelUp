/**
 * L'ÉCRAN UNIQUE de l'onglet Tactique (cockpit), rendu depuis une réponse de contrat.
 *
 * Ce que ces tests cadenassent :
 *   - la colonne rend une vignette par carte OUVRABLE, la plus jouée en tête ; une carte SOUS
 *     LE PLANCHER n'est pas une vignette, elle est dans le repli avec « n sur plancher » ;
 *   - sans `?carte=`, la plus jouée des ouvrables est lue d'office et l'URL n'est PAS réécrite ;
 *     le clic sur une vignette, lui, écrit la carte dans l'URL ;
 *   - la carte de l'URL est lue si elle est ouvrable ; sous le plancher ou hors du filtre, elle
 *     est nommée, sans requête de lecture ;
 *   - plus de bascule « Grille / Analyse », plus de pied de grille ;
 *   - l'échec du périmètre et la composition impossible se disent SUR le plan, cadre posé ; la
 *     colonne ne dit que ses propres états (liste en échec, attente, aucune carte) ;
 *   - aucune carte -> `EmptyState`, jamais une colonne vide muette ;
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
import type { FilterContextInput, TacticalMapsPage, TacticalRaster } from '@/lib/api/types'
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
    Link: (await import('@/test/linkDouble')).LinkDouble,
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

/** Réponse minimale du raster : suffit à faire sortir `TacticalAnalysisView` de son état
 *  d'attente. Le détail de cette lecture (plan, zone) est cadenassé ailleurs
 *  (`TacticalAnalysisView.test.tsx`) — ici on vérifie seulement QUI s'affiche quand
 *  `?carte=` est posé, pas ce que la vue d'analyse en fait. */
const RASTER_VIDE: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 0, min_y: 0, max_y: 0, valide: false },
  pas_m: 0,
  echelle: { p50: 0, p95: 0, borne: 0, n_cellules: 0, symetrique: false },
  cellules: [],
  matchs_filtres: 0,
  matchs_retenus: 0,
  matchs_victoire: 0,
  matchs_defaite: 0,
  points_ignores: 0,
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
    // La carte lue d'office (la plus jouée des ouvrables) : sa lecture répond.
    if (path.endsWith('/tactical/streets/raster')) return Promise.resolve(RASTER_VIDE)
    return Promise.reject(new Error(`appel inattendu : ${path}`))
  })
  getBlob.mockReset()
  // Défaut : la carte n'a pas de fond. La grille doit s'afficher quand même.
  getBlob.mockRejectedValue(new Error('pas de fond'))
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

/** Les lectures de raster postées, par carte. */
function lecturesRaster(): string[] {
  return post.mock.calls
    .map((c) => c[0] as string)
    .filter((p) => p.endsWith('/raster'))
}

describe('TacticalPage — l’écran unique', () => {
  it('la colonne : une vignette par carte ouvrable, les cartes sous le plancher dans le repli', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    // Le nom FR vient du contrat, pas du nom canonique ; le résumé tient sur une ligne.
    expect(streets.textContent).toContain('Ruelles')
    expect(streets.textContent).toContain('24 · 14 V / 9 D')
    const barre = streets.querySelector('[role="img"]')
    expect(barre?.getAttribute('aria-label')).toContain('14 victoires')
    expect(barre?.getAttribute('aria-label')).toContain('9 défaites')
    // Aquarius (9 matchs, sous le plancher) n'est pas une vignette : une ligne du repli.
    expect(screen.queryByTestId('tactical-map-aquarius')).toBeNull()
    expect(screen.getByTestId('tactical-map-plancher-aquarius').textContent).toContain('9 sur 10')
    expect(screen.getByText('1 carte sous le plancher')).toBeInTheDocument()
  })

  it('sans ?carte= : la plus jouée des ouvrables est lue d’office, l’URL n’est pas réécrite', async () => {
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByRole('region', { name: 'Ruelles' })).toBeInTheDocument()
    expect(screen.getByTestId('tactical-map-streets')).toHaveAttribute('aria-pressed', 'true')
    expect(lecturesRaster()).toEqual(['/players/JGtm/tactical/streets/raster'])
    expect(navigate).not.toHaveBeenCalled()
  })

  it('la vignette DIT ce qu’elle fait, et son clic écrit la carte dans l’URL — et rien d’autre', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets).toHaveAttribute('aria-label', 'Sélectionner Ruelles')
    fireEvent.click(streets)
    expect(navigate).toHaveBeenCalledTimes(1)
    const arg = navigate.mock.calls[0][0] as {
      search: (p: Record<string, unknown>) => Record<string, unknown>
    }
    // L'objet ENTIER : un encodage qui poserait `vue=all` ou `pl=` a chaque clic salirait
    // l'URL de parametres neutres sans qu'aucune autre assertion ne le voie.
    expect(arg.search({})).toEqual({ carte: 'streets' })
  })

  it('une carte sous le plancher ne navigue nulle part', async () => {
    renderWithProviders(<TacticalPage />)
    fireEvent.click(await screen.findByTestId('tactical-map-plancher-aquarius'))
    expect(navigate).not.toHaveBeenCalled()
  })

  it('la carte de l’URL, ouvrable : lue, et la colonne reste à côté', async () => {
    searchCourant = { carte: 'streets' }
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByRole('region', { name: 'Ruelles' })).toBeInTheDocument()
    expect(screen.getByTestId('tactical-map-streets')).toHaveAttribute('aria-pressed', 'true')
  })

  it('la carte de l’URL sous le plancher : nommée, « aucun match », aucune lecture', async () => {
    searchCourant = { carte: 'aquarius' }
    renderWithProviders(<TacticalPage />)
    const horsFiltre = await screen.findByTestId('tactical-carte-hors-filtre')
    expect(horsFiltre).toHaveTextContent('Aucun match sur cette carte dans le filtre')
    expect(screen.getByRole('region', { name: 'Aquarius' })).toBeInTheDocument()
    expect(lecturesRaster()).toEqual([])
    // Aucune vignette n'est présentée comme active : la carte affichée n'est pas dans la liste.
    expect(screen.getByTestId('tactical-map-streets')).toHaveAttribute('aria-pressed', 'false')
  })

  it('la carte de l’URL hors du filtre, jamais vue : un titre générique, jamais son identifiant', async () => {
    searchCourant = { carte: 'inconnue' }
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByTestId('tactical-carte-hors-filtre')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Carte hors du filtre' })).toBeInTheDocument()
    expect(screen.queryByText('inconnue')).toBeNull()
    expect(lecturesRaster()).toEqual([])
  })

  it('la carte de l’URL sortie du filtre : le dernier nom connu reste au titre', async () => {
    searchCourant = { carte: 'streets' }
    post.mockImplementation((path: string, body: unknown) => {
      if (path.endsWith('/filters/match-ids')) {
        const periode = (body as { period?: { start_date?: string | null } }).period
        return Promise.resolve({ match_ids: periode?.start_date ? ['m9'] : PERIMETRE })
      }
      if (path.endsWith('/tactical/maps')) {
        const ids = (body as { match_ids: string[] }).match_ids
        return Promise.resolve(ids[0] === 'm9' ? { ...page, cartes: (page.cartes ?? []).filter((c) => c.map_id !== 'streets') } : page)
      }
      if (path.endsWith('/tactical/streets/raster')) return Promise.resolve(RASTER_VIDE)
      return Promise.reject(new Error(`appel inattendu : ${path}`))
    })
    const rendu = renderWithProviders(<TacticalPage />)
    expect(await screen.findByRole('region', { name: 'Ruelles' })).toBeInTheDocument()
    searchCourant = { carte: 'streets', de: '2026-01-01' }
    rendu.rerender(<TacticalPage />)
    expect(await screen.findByTestId('tactical-carte-hors-filtre')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Ruelles' })).toBeInTheDocument()
    expect(screen.queryByText('streets')).toBeNull()
  })

  it('à trois colonnes, « Cartes jouées » et « Zone sélectionnée » prennent la hauteur de la carte du plan', async () => {
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    const cartes = screen.getByTestId('tactical-maps-column')
    expect(cartes.className).toContain('min-[1400px]:self-stretch')
    expect(cartes.className).toContain('min-[1400px]:[contain:size]')
    expect(cartes.className).not.toMatch(/\bh-\[/)
    const zone = screen.getByTestId('tactical-zone-card').parentElement as HTMLElement
    expect(zone.className).toContain('lg:self-stretch')
    expect(zone.className).toContain('lg:[contain:size]')
    expect(screen.getByTestId('tactical-cockpit').getAttribute('style')).not.toContain('--tac-cartes-h')
  })

  it('plus de bascule « Grille / Analyse », plus de pied de grille', async () => {
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    // Ni les libellés de l'ancienne bascule ni la phrase de l'ancien pied ne sont plus rendus.
    expect(screen.queryByText('Grille des cartes')).toBeNull()
    expect(screen.queryByText("Analyse d'une carte")).toBeNull()
    expect(screen.queryByText(/sur la période/)).toBeNull()
  })

  it('aucune carte : état vide explicite, jamais une grille muette', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: PERIMETRE })
        : Promise.resolve({ cartes: [], plancher_matchs: 10 }),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
    // La grille a REELLEMENT repondu : sans cette assertion, le test passerait aussi
    // sur un etat vide rendu AVANT la reponse (le defaut W1).
    expect(corpsGrille()).toBeDefined()
  })

  it('cartes nulles au contrat (slice Go vide) : même état vide, aucun plantage', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: PERIMETRE })
        : Promise.resolve({ cartes: null, plancher_matchs: 10 }),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
    expect(corpsGrille()).toBeDefined()
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
    // Le plan le dit SUR son fond, une seule fois.
    expect(await screen.findByTestId('tactical-plan-avis')).toHaveTextContent('Coéquipier introuvable')
    expect(corpsGrille()).toBeUndefined()
  })

  // ─── L'ANGLE « ESCOUADE » ET LE SELECTEUR DE COMPOSITION ──────────────────────────

  /** Le `qui` posté à la lecture de la carte affichée, au dernier appel. */
  const dernierQuiPoste = () =>
    (post.mock.calls.filter((c) => (c[0] as string).endsWith('/raster')).at(-1)?.[1] as { qui?: string })?.qui

  it('« Escouade » SANS composition : le sélecteur s’ouvre, mis en avant, et le texte le dit', async () => {
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    fireEvent.click(await screen.findByRole('button', { name: 'Escouade' }))
    expect(screen.getByTestId('tactical-escouade-attente')).toHaveTextContent('Aucune composition choisie')
    expect(screen.getByTestId('tactical-composition')).toHaveAttribute('data-mis-en-avant', 'true')
    expect(screen.getByPlaceholderText(/Rechercher parmi/)).toHaveFocus()
    expect(await screen.findByText('Ami')).toBeInTheDocument()
  })

  it('le sélecteur n’offre AUCUN bot, même parmi les plus croisés', async () => {
    get.mockResolvedValue({
      teammates: [
        { gamertag: '343 Bot', xuid: 'bid(3.0)', match_count: 90, as_teammate: 60, as_enemy: 30, avg_kda: null },
        { gamertag: 'Ami', xuid: 'xuid(42)', match_count: 30, as_teammate: 30, as_enemy: 0, avg_kda: null },
      ],
      enemies: [],
      total: 2,
    })
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher parmi 1 coéquipiers/))
    expect(await screen.findByText('Ami')).toBeInTheDocument()
    expect(screen.queryByText('343 Bot')).toBeNull()
  })

  it('« Escouade » AVEC une composition : l’angle s’applique à la lecture', async () => {
    searchCourant = { eq: 'Ami' }
    renderWithProviders(<TacticalPage />)
    await screen.findByTestId('tactical-map-streets')
    fireEvent.click(await screen.findByRole('button', { name: 'Escouade' }))
    await vi.waitFor(() => expect(dernierQuiPoste()).toBe('escouade'))
    expect(screen.queryByTestId('tactical-escouade-attente')).toBeNull()
    expect(screen.getByTestId('tactical-composition')).not.toHaveAttribute('data-mis-en-avant')
  })

  // ─── W1 — « AUCUNE CARTE » EST UNE REPONSE, PAS UNE ATTENTE ───────────────────────
  //
  // La grille est SUSPENDUE tant que le perimetre n'est pas resolu. En TanStack v5,
  // `isLoading` vaut `isPending && isFetching` — donc FAUX sur une requete desactivee :
  // s'y fier faisait rendre « Aucune carte jouee » au premier montage, a chaque clic sur
  // Analyser, et DEFINITIVEMENT quand la resolution echouait.

  it('resolution EN COURS : la page dit « chargement », jamais « aucune carte »', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? new Promise(() => {}) // jamais resolue
        : Promise.resolve(page),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Chargement des cartes…')).toBeInTheDocument()
    // Le plan, sans carte encore : son cadre sous l'indicateur.
    expect(screen.getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(screen.queryByText('Aucune carte jouée')).toBeNull()
    // Et la grille n'est PAS demandee tant que le perimetre n'existe pas.
    expect(corpsGrille()).toBeUndefined()
  })

  it('resolution EN ECHEC : la page le DIT, et ne rend pas un etat vide', async () => {
    const erreurs = vi.spyOn(console, 'error').mockImplementation(() => {})
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.reject(new Error('503'))
        : Promise.resolve(page),
    )
    try {
      renderWithProviders(<TacticalPage />)
      // Le plan le dit SUR son fond (cadre posé, sans carte encore) ; la colonne se tait.
      expect(await screen.findByTestId('tactical-plan-avis')).toHaveTextContent("L'analyse n'a pas pu être chargée.")
      expect(screen.getByTestId('tactical-plan-frame')).toBeInTheDocument()
      expect(screen.queryByText('Aucune carte jouée')).toBeNull()
      // L'echec est JOURNALISE, jamais avale.
      expect(erreurs).toHaveBeenCalled()
    } finally {
      erreurs.mockRestore()
    }
  })

  // ─── W5 — UNE RESOLUTION VIDE EST UNE REPONSE : LA REQUETE PART ───────────────────
  //
  // « Aucun match ne passe le filtre » et « le perimetre n'est pas encore la » sont deux
  // etats distincts. Le premier DOIT interroger la grille (avec une liste vide, que le
  // serveur traduit en « aucune carte ») ; le second doit attendre.

  it('resolution VIDE : la grille est bien demandee, avec une liste vide', async () => {
    post.mockImplementation((path: string) =>
      path.endsWith('/filters/match-ids')
        ? Promise.resolve({ match_ids: [] })
        : Promise.resolve({ cartes: [], plancher_matchs: 10 }),
    )
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
    expect(corpsGrille()?.match_ids).toEqual([])
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
    expect(streets.textContent).toContain('24 · 14 W / 9 L')
  })
})
