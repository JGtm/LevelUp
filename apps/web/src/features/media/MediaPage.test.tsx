/**
 * Tests composant — MediaPage (Slice 8 — Médias).
 *
 * Smoke : monte, spinner, puis galerie vide depuis MSW.
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { screen, waitFor, fireEvent, act } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { server } from '@/test/setup'
import { MediaPage } from './MediaPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('MediaPage', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<MediaPage />)
    expect(container).toBeTruthy()
  })

  it("ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)", () => {
    const { container } = renderWithProviders(<MediaPage />)
    // Le spinner local de la page a été retiré — feedback global via TopProgressBar.
    // La toolbar reste visible mais la zone de contenu est vide pendant le fetch.
    expect(container.querySelector('.animate-spin')).not.toBeInTheDocument()
  })

  // Test "affiche le titre Médias" supprimé : titre h1 retiré du composant
  // (refacto post-84ae65ca, NavL1 expose la section). Rendu post-loading déjà
  // couvert par les tests de filtres (Tous types / Captures / Clips) ci-dessous.

  it('affiche les filtres de type de média', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Tous types')).toBeInTheDocument()
      expect(screen.getByText('Captures')).toBeInTheDocument()
    })
  })

  it('affiche le filtre Clips', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Clips')).toBeInTheDocument()
    })
  })

  it('affiche le sélecteur de tri', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByLabelText('Tri de la galerie')).toBeInTheDocument()
    })
  })

  it('distingue filtres et tri dans la toolbar', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Filtres :')).toBeInTheDocument()
      expect(screen.getByText('Tri :')).toBeInTheDocument()
    })
  })

  it('utilise des listes déroulantes pour les cartes et les modes', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByLabelText('Carte de la galerie')).toBeInTheDocument()
      expect(screen.getByLabelText('Mode de la galerie')).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Toutes cartes' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Tous modes' })).toBeInTheDocument()
      // Refacto post-84ae65ca : les modes sont maintenant groupés par catégorie
      // via <optgroup>. "Slayer" est un optgroup label (pas une option ARIA)
      // contenant "Toutes catégories" comme option canonique.
      const modeSelect = screen.getByLabelText('Mode de la galerie')
      expect(modeSelect.querySelector('optgroup[label="Slayer"]')).not.toBeNull()
    })
  })

  it('reconstruit les options cartes et modes depuis les items quand available_filters est vide', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/media', () => HttpResponse.json({
        items: {
          items: [
            {
              basename: 'clip1.mp4',
              file_path: '/clips/clip1.mp4',
              kind: 'clip',
              thumbnail_path: null,
              capture_end_utc: null,
              match_id: 'm1',
              match_start_time: '2025-01-01T10:00:00Z',
              section: 'mine',
              owner_gamertag: null,
              map_name: 'Recharge',
              mode_name: 'Oddball',
              liked: false,
              like_count: 0,
            },
          ],
          pagination: { total: 1, page: 1, page_size: 24, has_next: false, has_prev: false },
          freshness: null,
        },
        total_mine: 1,
        total_teammates: 0,
        total_unassigned: 0,
        available_filters: { maps: [], modes: [] },
      })),
    )

    renderWithProviders(<MediaPage />)

    await waitFor(() => {
      // "Recharge" est rendu comme <option> dans le select cartes (option ARIA OK).
      expect(screen.getByRole('option', { name: 'Recharge' })).toBeInTheDocument()
      // "Oddball" est maintenant un optgroup label (refacto post-84ae65ca,
      // les modes sont groupés par catégorie). On vérifie l'optgroup directement.
      const modeSelect = screen.getByLabelText('Mode de la galerie')
      expect(modeSelect.querySelector('optgroup[label="Oddball"]')).not.toBeNull()
    })
  })

  it('bascule les libellés de toolbar en anglais quand locale=en', async () => {
    act(() => {
      useAppShellStore.setState({ locale: 'en' })
    })
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Filters:')).toBeInTheDocument()
      expect(screen.getByText('Sort:')).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'All maps' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'All modes' })).toBeInTheDocument()
    })
  })

  it('affiche la zone upload', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      // La zone d'upload contient une mention "parcourir" ou "glisser"
      const matches = screen.getAllByText(/parcourir|glisser|déposer/i)
      expect(matches.length).toBeGreaterThan(0)
    })
  })

  it('affiche le toggle Aimés seulement', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByLabelText('Afficher seulement les médias aimés')).toBeInTheDocument()
    })
  })

  it('toggle Aimés est désactivé par défaut', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      const toggle = screen.getByLabelText('Afficher seulement les médias aimés')
      expect(toggle).toHaveAttribute('aria-pressed', 'false')
    })
  })

  it('cliquer sur un filtre de type ne lève pas d\'erreur', async () => {
    renderWithProviders(<MediaPage />)
    await waitFor(() => {
      expect(screen.getByText('Captures')).toBeInTheDocument()
    })
    // Cliquer ne doit pas lever d'exception
    expect(() => fireEvent.click(screen.getByText('Captures'))).not.toThrow()
  })

  // ─── Navigation contextuelle vers la page Match ────────────────────────────
  // Vérifie que cliquer sur l'icône "Voir le match" depuis la galerie persiste
  // un MatchNavContext (source=media) — verrou anti-régression du fix 2026-05-07
  // qui a rebranché MediaThumbnailCard / CoverFlowModal sur useNavigateToMatch.
  it('cliquer sur l\'icône "Voir le match" persiste un MatchNavContext (source=media) en sessionStorage', async () => {
    const ITEMS = [
      {
        basename: 'A.mp4',
        file_path: '/A.mp4',
        kind: 'clip',
        thumbnail_path: null,
        capture_end_utc: null,
        match_id: 'm-A',
        match_start_time: '2025-01-01T10:00:00Z',
        section: 'mine',
        owner_gamertag: null,
        map_name: 'Aquarius',
        mode_name: 'Slayer',
        liked: false,
        like_count: 0,
      },
      {
        basename: 'B.png',
        file_path: '/B.png',
        kind: 'screenshot',
        thumbnail_path: null,
        capture_end_utc: null,
        match_id: 'm-B',
        match_start_time: '2025-01-02T10:00:00Z',
        section: 'mine',
        owner_gamertag: null,
        map_name: 'Bazaar',
        mode_name: 'Slayer',
        liked: false,
        like_count: 0,
      },
      {
        basename: 'orphan.mp4',
        file_path: '/orphan.mp4',
        kind: 'clip',
        thumbnail_path: null,
        capture_end_utc: null,
        match_id: null,
        match_start_time: null,
        section: 'mine',
        owner_gamertag: null,
        map_name: null,
        mode_name: null,
        liked: false,
        like_count: 0,
      },
    ]
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/media', () => HttpResponse.json({
        items: {
          items: ITEMS,
          pagination: { total: 3, page: 1, page_size: 24, has_next: false, has_prev: false },
          freshness: null,
        },
        total_mine: 3,
        total_teammates: 0,
        total_unassigned: 1,
        available_filters: { maps: [], modes: [] },
      })),
    )

    sessionStorage.clear()
    renderWithProviders(<MediaPage />)

    // Attendre que les vignettes soient rendues — l'icône d'ouverture est
    // affichée pour les items avec match_id (donc 2 boutons attendus, m-A & m-B).
    const openBtns = await screen.findAllByRole('button', { name: /Ouvrir.*match/ })
    expect(openBtns.length).toBe(2)

    openBtns[0].click()

    // Le hook useNavigateToMatch persiste le ctx sous la clé `levelup:matchNav:<matchId>`.
    const persistedRaw = sessionStorage.getItem('levelup:matchNav:m-A')
    expect(persistedRaw).not.toBeNull()
    const persisted = JSON.parse(persistedRaw as string)
    expect(persisted.ctx.source).toBe('media')
    // Seuls les items AVEC match_id doivent être inclus (l'orphelin est filtré).
    expect(persisted.ctx.matchIds).toEqual(expect.arrayContaining(['m-A', 'm-B']))
    expect(persisted.ctx.matchIds).toHaveLength(2)
    // Phase 2c : filtersLabel (chaîne libre) remplacé par contextDescriptor typé.
    expect(persisted.ctx.contextDescriptor).toEqual({ kind: 'media' })
  })
})
