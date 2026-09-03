/**
 * Tests composant — LeaderboardBlock (refonte Classement).
 *
 * Catégorie csr-world par défaut : tableau CSR mondial avec badges tier,
 * highlight du joueur local, états vide / erreur.
 */
import { describe, it, expect } from 'vitest'
import { screen, waitFor, within, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/setup'
import { LeaderboardBlock } from './LeaderboardBlock'

const p = (path: string) => `/api/v1${path}`

const ENTRIES = [
  { rank: 1, xuid: 'x1', gamertag: 'LocalAce', csr_value: 1850, tier: 'Onyx', sub_tier: 0, is_local: true },
  { rank: 2, xuid: 'x2', gamertag: 'RemoteRival', csr_value: 1720, tier: 'Diamond', sub_tier: 6, is_local: false },
  { rank: 3, xuid: 'x3', gamertag: 'LocalVet', csr_value: 1600, tier: 'Diamond', sub_tier: 3, is_local: true },
]

function mockLeaderboard(entries: typeof ENTRIES) {
  server.use(
    http.get(p('/players/:playerSlug/pages/leaderboard'), () =>
      HttpResponse.json({
        entries,
        category: 'csr-world',
        season_id: 'csrseason13-2',
        playlist_id: 'p',
        title_slug: 'halo_infinite',
        total: entries.length,
      }),
    ),
  )
}

// Playlists réelles (asset IDs) — utilisées par les tests de couplage saison/playlist.
const ARENA = 'edfef3ac-9cbe-4fa2-b949-8f29deafd483'
const SNIPERS = '6233381c-fc96-40b9-b1ff-f6a4de72dd7a'
const DOUBLES = 'fa5aa2a3-2428-4912-a023-e1eeea7b877c'

type CatalogSeason = { id: string; display_name: string; enriched: boolean; playlist_ids?: string[] }

function mockCatalog(
  seasons: CatalogSeason[],
  playlists: Array<{ id: string; display_name: string }> = [{ id: ARENA, display_name: 'Arène classée' }],
) {
  server.use(
    http.get(p('/players/:playerSlug/pages/leaderboard/catalog'), () =>
      HttpResponse.json({
        seasons,
        playlists: playlists.map((pl) => ({ ...pl, enriched: false })),
      }),
    ),
  )
}

/** Ligne enrichie (stats détaillées présentes) / ligne CSR seule, pour les seuils. */
function enrichedRow(rank: number) {
  return {
    rank, xuid: `e${rank}`, gamertag: `Enrichi${rank}`, csr_value: 1900 - rank, tier: 'Onyx', sub_tier: 0, is_local: false,
    match_count: 10, kills: 100, deaths: 50, assists: 20, win_rate: 0.6, kda: 15, accuracy: 500,
  }
}
function plainRow(rank: number) {
  return { rank, xuid: `p${rank}`, gamertag: `Brut${rank}`, csr_value: 1500 - rank, tier: 'Diamond', sub_tier: 1, is_local: false }
}

describe('LeaderboardBlock', () => {
  it('affiche le spinner pendant le chargement', () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    expect(document.querySelector('[class*="animate-spin"]')).toBeTruthy()
  })

  it('affiche le titre du classement CSR mondial', () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    expect(screen.getByText('Classement CSR mondial')).toBeInTheDocument()
  })

  it('marque les joueurs locaux avec le badge Local', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    expect(screen.getAllByText('Local')).toHaveLength(2) // LocalAce + LocalVet

    const remoteRow = screen.getByText('RemoteRival').closest('tr')!
    expect(within(remoteRow).queryByText('Local')).not.toBeInTheDocument()
  })

  it('signale une saison archivée (non enrichie) : suffixe dans le sélecteur + bandeau à la sélection', async () => {
    mockLeaderboard(ENTRIES)
    mockCatalog([
      { id: 'csrseason13-2', display_name: 'Infinite (13.2)', enriched: true },
      { id: 'csrseason4-1', display_name: 'Saison 4.1', enriched: false },
    ])
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    // L'option archivée porte le suffixe « (archivée) » dans le menu déroulant.
    await waitFor(() => {
      expect(screen.getByRole('option', { name: /archivée/i })).toBeInTheDocument()
    })
    // Saison active par défaut (13.2, enrichie) → pas de bandeau.
    expect(screen.queryByText(/Saison archivée/i)).not.toBeInTheDocument()

    // Sélection de la saison archivée → bandeau « classement seul » affiché.
    fireEvent.change(screen.getByLabelText('Saison'), { target: { value: 'csrseason4-1' } })
    expect(screen.getByText(/Saison archivée/i)).toBeInTheDocument()
  })

  it('affiche les lignes dans l’ordre du rang', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    const rows = screen.getAllByRole('row')
    expect(within(rows[1]).getByText('LocalAce')).toBeInTheDocument()
    expect(within(rows[2]).getByText('RemoteRival')).toBeInTheDocument()
    expect(within(rows[3]).getByText('LocalVet')).toBeInTheDocument()
  })

  it('affiche le CSR + l’image de rang dans la colonne Rang (image, pas de libellé tier)', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText(/1\s?850/)).toBeInTheDocument()
    })
    expect(screen.getByText(/1\s?720/)).toBeInTheDocument()
    expect(screen.getByText(/1\s?600/)).toBeInTheDocument()
    // Colonne Rang = image (via alt), plus de libellé texte « Diamond VI ».
    expect(screen.getByAltText('Onyx')).toBeInTheDocument()
    expect(screen.getAllByAltText('Diamond').length).toBeGreaterThan(0)
    expect(screen.queryByText('Diamond VI')).not.toBeInTheDocument()
  })

  it('mode enrichi : colonnes Frags/Morts/Assistances + FDA (pas KDA)', async () => {
    const enriched = [
      {
        rank: 1, xuid: 'e1', gamertag: 'Topfrag', csr_value: 1900, tier: 'Onyx', sub_tier: 0, is_local: false,
        match_count: 10, cumulative_match_count: 137, kills: 200, deaths: 80, assists: 50, win_rate: 0.7, kda: 25, accuracy: 55,
      },
      {
        rank: 2, xuid: 'e2', gamertag: 'Steady', csr_value: 1800, tier: 'Onyx', sub_tier: 0, is_local: false,
        match_count: 10, cumulative_match_count: 90, kills: 150, deaths: 60, assists: 70, win_rate: 0.5, kda: 18, accuracy: 50,
      },
    ]
    mockLeaderboard(enriched as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => expect(screen.getByText('Topfrag')).toBeInTheDocument())
    // En-têtes (boutons triables) des nouvelles colonnes en FR + FDA (pas KDA) + Matchs.
    expect(screen.getByRole('button', { name: /Frags/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Morts/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Assistances/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /FDA/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Matchs/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^KDA/ })).not.toBeInTheDocument()
    // Valeurs Frags + le total de matchs CUMULÉ (pas match_count de la saison).
    expect(screen.getByText('200')).toBeInTheDocument()
    expect(screen.getByText('150')).toBeInTheDocument()
    expect(screen.getByText('137')).toBeInTheDocument()
  })

  it('affiche un état vide si le classement est vide', async () => {
    mockLeaderboard([])
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Classement vide')).toBeInTheDocument()
    })
  })

  it('affiche une erreur en cas de réponse serveur 500', async () => {
    server.use(
      http.get(p('/players/:playerSlug/pages/leaderboard'), () =>
        HttpResponse.json({ error: 'internal' }, { status: 500 }),
      ),
    )
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Erreur de chargement')).toBeInTheDocument()
    })
  })

  it('masque les colonnes enrichies tant qu’aucun joueur n’est backfillé', async () => {
    mockLeaderboard(ENTRIES) // entrées sans champs d'enrichissement
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    // L'en-tête "Victoires" (col_win_rate) ne doit pas apparaître.
    expect(screen.queryByText('Victoires')).not.toBeInTheDocument()
    expect(screen.queryByText('Δ rang')).not.toBeInTheDocument()
  })

  it('affiche les colonnes enrichies (win rate, KDA, Δ rang) quand le joueur est backfillé', async () => {
    const enriched = [
      {
        // kda/accuracy sont des SOMMES brutes → affichées en moyenne (÷ match_count).
        rank: 1, xuid: 'x1', gamertag: 'Ace', csr_value: 1850, tier: 'Onyx', sub_tier: 0, is_local: false,
        match_count: 20, win_rate: 0.65, kda: 36, accuracy: 1050, win_rate_trend: 'up', kda_trend: 'down', rank_delta: 3,
        kills: 100, deaths: 80, assists: 30, damage_dealt: 50000, damage_taken: 48000,
      },
      {
        rank: 2, xuid: 'x2', gamertag: 'Rival', csr_value: 1700, tier: 'Diamond', sub_tier: 6, is_local: false,
        match_count: 15, win_rate: 0.4, kda: 16.5, accuracy: 600, win_rate_trend: 'stable', kda_trend: 'stable', rank_delta: -2,
        kills: 60, deaths: 70, assists: 12, damage_dealt: 30000, damage_taken: 35000,
      },
    ]
    mockLeaderboard(enriched as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Ace')).toBeInTheDocument()
    })
    // En-têtes enrichis présents.
    expect(screen.getByText('Victoires')).toBeInTheDocument()
    expect(screen.getByText('Δ rang')).toBeInTheDocument()

    // Valeurs de la ligne Ace : win rate FR, KDA moyen (36/20=1.80), précision
    // moyenne (1050/20=52,5%), delta signé, nb matchs.
    const aceRow = screen.getByText('Ace').closest('tr')!
    expect(within(aceRow).getByText('65,0%')).toBeInTheDocument()
    expect(within(aceRow).getByText('1.80')).toBeInTheDocument()
    expect(within(aceRow).getByText('52,5%')).toBeInTheDocument()
    expect(within(aceRow).getByText('+3')).toBeInTheDocument()
    expect(within(aceRow).getByText('20')).toBeInTheDocument()

    // Delta négatif sur la ligne Rival.
    const rivalRow = screen.getByText('Rival').closest('tr')!
    expect(within(rivalRow).getByText('-2')).toBeInTheDocument()
  })

  // ─── Seuil de couverture d'enrichissement (décision D2 : 25 %) ──────────────

  it('sous 25 % de lignes enrichies : colonnes détaillées masquées + bandeau « indisponibles »', async () => {
    // 1 enrichie sur 5 = 20 % → un seul joueur backfillé ne justifie pas 11 colonnes.
    mockLeaderboard([enrichedRow(1), plainRow(2), plainRow(3), plainRow(4), plainRow(5)] as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => expect(screen.getByText('Enrichi1')).toBeInTheDocument())
    expect(screen.queryByText('Victoires')).not.toBeInTheDocument()
    expect(screen.queryByText('Δ rang')).not.toBeInTheDocument()
    expect(screen.getByText(/Stats détaillées indisponibles pour ce relevé/i)).toBeInTheDocument()
  })

  it('entre 25 % et 80 % : colonnes détaillées affichées + bandeau « partielles »', async () => {
    // 2 enrichies sur 4 = 50 %.
    mockLeaderboard([enrichedRow(1), enrichedRow(2), plainRow(3), plainRow(4)] as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => expect(screen.getByText('Enrichi1')).toBeInTheDocument())
    expect(screen.getByText('Victoires')).toBeInTheDocument()
    expect(screen.getByText(/Stats détaillées partielles/i)).toBeInTheDocument()
    // Le bandeau chiffre la couverture (2 sur 4) plutôt que de rester vague.
    expect(screen.getByText(/2 joueurs enrichis sur 4/i)).toBeInTheDocument()
  })

  it('couverture complète : colonnes affichées et AUCUN bandeau de couverture', async () => {
    mockLeaderboard([enrichedRow(1), enrichedRow(2), enrichedRow(3)] as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => expect(screen.getByText('Enrichi1')).toBeInTheDocument())
    expect(screen.getByText('Victoires')).toBeInTheDocument()
    expect(screen.queryByText(/Stats détaillées partielles/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Stats détaillées indisponibles/i)).not.toBeInTheDocument()
  })

  // ─── Couplage saison ↔ playlist (playlist_ids du catalogue) ─────────────────

  it('couple les sélecteurs : la playlist se limite à la saison choisie et retombe sur une playlist valide', async () => {
    mockLeaderboard(ENTRIES)
    mockCatalog(
      [
        { id: 'csrseason13-3', display_name: 'Infinite (13.3)', enriched: true, playlist_ids: [ARENA, SNIPERS] },
        { id: 'csrseason4-1', display_name: 'Saison 4.1', enriched: true, playlist_ids: [DOUBLES] },
      ],
      [
        { id: ARENA, display_name: 'Arène classée' },
        { id: SNIPERS, display_name: 'Snipers classés' },
        { id: DOUBLES, display_name: 'Duo classé' },
      ],
    )
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    const playlistSelect = () => screen.getByLabelText('Sélection') as HTMLSelectElement
    const optionValues = () =>
      within(playlistSelect())
        .getAllByRole('option')
        .map((o) => (o as HTMLOptionElement).value)

    // Saison active (13.3) : seules SES 2 playlists sont proposées, pas les 3 du catalogue.
    await waitFor(() => expect(optionValues()).toEqual([ARENA, SNIPERS]))
    fireEvent.change(playlistSelect(), { target: { value: SNIPERS } })
    expect(playlistSelect().value).toBe(SNIPERS)

    // Bascule sur une saison où Snipers n'a jamais été relevé → repli sur SA playlist.
    fireEvent.change(screen.getByLabelText('Saison'), { target: { value: 'csrseason4-1' } })
    await waitFor(() => expect(optionValues()).toEqual([DOUBLES]))
    expect(playlistSelect().value).toBe(DOUBLES)
  })

  // ─── Tri actif sur une colonne qui disparaît (S4) ───────────────────────────

  it('tri sur une colonne enrichie : neutralisé quand elle disparaît, restauré quand elle revient', async () => {
    // Deux relevés servis par la même page : 13.3 couvert (colonnes enrichies),
    // 4.1 sous le seuil (aucune ligne enrichie → colonnes masquées).
    const covered = [
      { ...enrichedRow(1), kills: 10 },
      { ...enrichedRow(2), kills: 300 },
      { ...enrichedRow(3), kills: 200 },
    ]
    const bare = [plainRow(1), plainRow(2), plainRow(3), plainRow(4)]
    server.use(
      http.get(p('/players/:playerSlug/pages/leaderboard'), ({ request }) => {
        const season = new URL(request.url).searchParams.get('season')
        const entries = season === 'csrseason4-1' ? bare : covered
        return HttpResponse.json({
          entries,
          category: 'csr-world',
          season_id: season,
          playlist_id: ARENA,
          title_slug: 'halo_infinite',
          total: entries.length,
        })
      }),
    )
    mockCatalog([
      { id: 'csrseason13-3', display_name: 'Infinite (13.3)', enriched: true, playlist_ids: [ARENA] },
      { id: 'csrseason4-1', display_name: 'Saison 4.1', enriched: true, playlist_ids: [ARENA] },
    ])
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    const gamertagOrder = () =>
      screen
        .getAllByRole('row')
        .slice(1)
        .map((r) => r.querySelectorAll('td')[1]?.textContent)
    // En-têtes portant réellement le tri : aria-sort ascending/descending (les
    // colonnes non triables n'ont pas l'attribut, les triables inactives ont 'none').
    const sortedHeaders = () =>
      screen
        .getAllByRole('columnheader')
        .filter((th) => ['ascending', 'descending'].includes(th.getAttribute('aria-sort') ?? ''))
        .map((th) => [th.textContent, th.getAttribute('aria-sort')])

    // Tri par Frags décroissant : l'ordre s'écarte du rang.
    await waitFor(() => expect(screen.getByRole('button', { name: /Frags/ })).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: /Frags/ }))
    await waitFor(() => expect(gamertagOrder()).toEqual(['Enrichi2', 'Enrichi3', 'Enrichi1']))

    // Relevé sous le seuil : la colonne Frags disparaît → retour à l'ordre du rang,
    // et le SEUL en-tête marqué trié est « # » (aucun indicateur fantôme).
    fireEvent.change(screen.getByLabelText('Saison'), { target: { value: 'csrseason4-1' } })
    await waitFor(() => expect(gamertagOrder()).toEqual(['Brut1', 'Brut2', 'Brut3', 'Brut4']))
    expect(screen.queryByRole('button', { name: /Frags/ })).not.toBeInTheDocument()
    expect(sortedHeaders()).toEqual([['#▲', 'ascending']])

    // Retour au relevé couvert : le tri choisi n'a pas été perdu, seulement neutralisé.
    fireEvent.change(screen.getByLabelText('Saison'), { target: { value: 'csrseason13-3' } })
    await waitFor(() => expect(gamertagOrder()).toEqual(['Enrichi2', 'Enrichi3', 'Enrichi1']))
    expect(sortedHeaders()).toEqual([['Frags▼', 'descending']])
  })

  it('catalogue sans playlist_ids (backend antérieur) : toutes les playlists restent proposées', async () => {
    mockLeaderboard(ENTRIES)
    mockCatalog(
      [{ id: 'csrseason13-3', display_name: 'Infinite (13.3)', enriched: true }],
      [
        { id: ARENA, display_name: 'Arène classée' },
        { id: SNIPERS, display_name: 'Snipers classés' },
      ],
    )
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    const playlistSelect = () => screen.getByLabelText('Sélection') as HTMLSelectElement
    await waitFor(() =>
      expect(
        within(playlistSelect())
          .getAllByRole('option')
          .map((o) => (o as HTMLOptionElement).value),
      ).toEqual([ARENA, SNIPERS]),
    )
  })
})
