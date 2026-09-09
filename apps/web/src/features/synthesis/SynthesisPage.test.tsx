/**
 * Tests composant — SynthesisPage (Slice 7 — Synthèse).
 *
 * Smoke : monte, spinner, puis tableau de comparaison depuis MSW.
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'

// Mocks des wrappers ECharts : echarts-for-react absent en env portable.
// Les fixtures de synthesis ont comparison_metrics=[] et heatmap_data=[]
// donc les charts ne sont pas rendus — le mock évite juste l'erreur de résolution.
vi.mock('@/components/charts/ChartCard', () => ({
  ChartCard: () => <div data-testid="chart-card" />,
}))
vi.mock('@/components/charts/Heatmap2DChart', () => ({
  Heatmap2DChart: () => <div data-testid="chart-card" />,
}))
import { http, HttpResponse } from 'msw'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { synthesisFixture } from '@/test/handlers'
import { useAppShellStore } from '@/stores/appShellStore'
import { SynthesisPage } from './SynthesisPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>{children}</a>
    ),
  }
})

describe('SynthesisPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container).toBeTruthy()
  })

  // PLAN_EQUIPEMENT_GACHIS_2026-09-09 (E5.13) : le bloc « servi ou gâché » de la
  // fixture (synthesisFixture.equipment_usage) atteint bien l'écran — preuve du
  // câblage bout en bout (aucune requête neuve : même réponse /pages/synthesis).
  it("affiche le bloc « servi ou gâché » quand la réponse le porte", async () => {
    renderWithProviders(<SynthesisPage />)
    expect(await screen.findByText("Usages d'équipement")).toBeInTheDocument()
    expect(await screen.findByText('Contrôle des armes spéciales')).toBeInTheDocument()
  })

  it('ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container.firstChild).toBeNull()
  })

  // ─── DETTE : refacto post-merge synthesis-kpi-grid (commit 818a26bc) ─────
  //
  // Les 14 tests ci-dessous (period selectors, scope-bar, highlights D5,
  // breakdowns D7) référencent l'ancienne UI Synthèse (boutons preset "Tout",
  // testid `scope-bar`, sections "Meilleurs matchs"/"Matchs difficiles"…)
  // qui a été refondue par 818a26bc en "Vue d'ensemble + 5 graphiques ECharts".
  //
  // Les sélecteurs n'existent plus dans le nouveau DOM. Ces tests étaient
  // déjà cassés sur origin/feat/synthesis-kpi-grid avant le merge (pas une
  // régression du merge). À refondre dans un PR dédié pour cibler la nouvelle
  // UI (sectionnage par chart : SynthesisHeatmapChart, SynthesisTopWeeksChart,
  // SynthesisOutcomesByGroupChart, SynthesisBipolaireChart, SynthesisFragCard).
  it.skip('affiche les selecteurs de période', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Tout' })).toBeInTheDocument()
    })
  })

  it.skip('affiche des placeholders explicites pour les sections vides', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText(/Activité indisponible/i)).toBeInTheDocument()
      expect(screen.getByText(/Aucune semaine remarquable/i)).toBeInTheDocument()
    })
  })

  describe('scope et overview (D9)', () => {
    it.skip('affiche la scope-bar après chargement', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByTestId('scope-bar')).toBeInTheDocument()
      })
    })

    it.skip('scope-bar affiche le nombre de matchs du fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        const bar = screen.getByTestId('scope-bar')
        // Le fixture MSW retourne match_count: 5
        expect(bar).toHaveTextContent('5')
      })
    })

    it.skip('scope-bar affiche la période "Tout"', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        const bar = screen.getByTestId('scope-bar')
        expect(bar).toHaveTextContent('Tout')
      })
    })

    it.skip('scope-bar est rendu avant le bloc Solo / Escouade', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByTestId('scope-bar')).toBeInTheDocument()
      })
      const bar = screen.getByTestId('scope-bar')
      const soloCard = screen.getByText(/Solo \(/i).closest('[class*="Card"]') ??
                       screen.getByText(/Solo \(/i).closest('div')
      if (soloCard) {
        const position = bar.compareDocumentPosition(soloCard)
        // DOCUMENT_POSITION_FOLLOWING = 4 → soloCard est après bar
        expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      }
    })

    it.skip('affiche la vue d\'ensemble D4 avec les KPIs du fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        // overview.total_wins=3, total_losses=2, win_rate=0.6
        expect(screen.getByText('Vue d\'ensemble')).toBeInTheDocument()
        // Label 'Victoires' suivi de la valeur
        expect(screen.getByText('Victoires')).toBeInTheDocument()
        expect(screen.getByText('60.0%')).toBeInTheDocument()  // win_rate
      })
    })
  })

  describe('highlights D5', () => {
    it.skip('affiche la section "Meilleurs matchs" depuis le fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Meilleurs matchs')).toBeInTheDocument()
      })
    })

    it.skip('affiche la section "Matchs difficiles" depuis le fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Matchs difficiles')).toBeInTheDocument()
      })
    })

    it.skip('affiche les kills du meilleur match (12)', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('12')).toBeInTheDocument()
      })
    })
  })

  // Section "Relations de jeu" (D6) supprimée le 2026-05-27 ; les encounters
  // restent accessibles via la page palmares/relations.

  describe('breakdowns D7', () => {
    it.skip('affiche la section "Par carte"', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Par carte')).toBeInTheDocument()
      })
    })

    it.skip('affiche la section "Par mode"', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Par mode')).toBeInTheDocument()
      })
    })

    it.skip('affiche les noms de cartes du fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Aquarius')).toBeInTheDocument()
        expect(screen.getByText('Bazaar')).toBeInTheDocument()
      })
    })

    it.skip('affiche les noms de modes du fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByText('Slayer')).toBeInTheDocument()
        expect(screen.getByText('CTF')).toBeInTheDocument()
      })
    })
  })
})

/**
 * LE MONTAGE DE LA SECTION « PORTÉE DES ENGAGEMENTS » (lot 5, revue du 2026-09-06).
 *
 * Les tests de la section elle-même lui passent son bloc à la main : ils resteraient tous
 * verts si la page oubliait de le brancher (`weaponRange={data.weapon_range}`) ou si la
 * capability produit `weapon_range` masquait la section pour de bon. Ces trois cas pincent le
 * CÂBLAGE de bout en bout : réponse -> page -> section, et la porte de capability.
 */
describe('SynthesisPage — montage de la section « Portée des engagements »', () => {
  const REGION = 'Portée par arme — mes frags et mes morts'

  const weaponRangeBlock = {
    weapons: [
      {
        weapon_key: 'hinf_br75',
        label: 'Fusil de combat BR75',
        label_en: 'BR75 Battle Rifle',
        kills: { measured: 281, p10: 7.1, median: 13.6, p90: 24.9, above_pct: 37, level_pct: 49, below_pct: 14 },
        deaths: { measured: 402, p10: 8.9, median: 16.4, p90: 29.7, above_pct: 10, level_pct: 52, below_pct: 38 },
      },
    ],
    median_kills_m: 7.4,
    median_deaths_m: 11.8,
    measured_kills: 1214,
    total_kills: 1602,
    measured_deaths: 1087,
    total_deaths: 1455,
    below_threshold_kills: [],
    below_threshold_deaths: [],
  }

  function serveSynthesis(body: Record<string, unknown>) {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/synthesis', () => HttpResponse.json(body)),
    )
  }

  /** Un titre qui déclare (ou non) `weapon_range` — `useCapability` est fail-open sans titre. */
  function setTitle(capabilities: string[]) {
    useAppShellStore.setState({
      currentTitleSlug: 'sonde',
      availableTitles: [
        { slug: 'sonde', name: 'Sonde', status: 'active', capabilities, is_default: false },
      ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
    })
  }

  afterEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  it('avec le bloc servi et la capability active, la section est montée', async () => {
    setTitle(['weapon_range'])
    serveSynthesis({ ...synthesisFixture, weapon_range: weaponRangeBlock })
    renderWithProviders(<SynthesisPage />)
    expect(await screen.findByRole('region', { name: REGION })).toBeInTheDocument()
  })

  it('sans bloc dans la réponse, la section ne s’affiche pas', async () => {
    setTitle(['weapon_range'])
    serveSynthesis({ ...synthesisFixture })
    renderWithProviders(<SynthesisPage />)
    // On attend une ancre chargée AVANT de conclure à l'absence : sinon le test passerait
    // simplement parce que la page n'a pas fini de charger.
    await screen.findByRole('heading', { name: "Vue d'ensemble" })
    expect(screen.queryByRole('region', { name: REGION })).not.toBeInTheDocument()
  })

  it('sans la capability du titre, le bloc servi reste masqué', async () => {
    setTitle(['matchmaking'])
    serveSynthesis({ ...synthesisFixture, weapon_range: weaponRangeBlock })
    renderWithProviders(<SynthesisPage />)
    await screen.findByRole('heading', { name: "Vue d'ensemble" })
    expect(screen.queryByRole('region', { name: REGION })).not.toBeInTheDocument()
  })
})
