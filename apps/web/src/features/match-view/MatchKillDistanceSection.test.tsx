/**
 * MatchKillDistanceSection.test.tsx — le graphe de distance par arme (réouverture DEC-8,
 * 2026-09-02). Couvre LES DEUX PORTES (règle du 2026-09-05, registre L3) : le TITRE
 * (`film.kill_positions` — sans elle, RIEN n'est rendu) puis le MATCH (l'ÉTAT VIDE QUI SE
 * DIT : retour user « je ne vois rien du tout » — la section explique au lieu de
 * disparaître). Plus le rendu nominal et le repli gamertag→xuid. Le graphe lui-même (bâton
 * min→max + losange de moyenne) est testé PUR dans `_killDistanceChart.test.ts` — ici
 * ECharts est mocké (jsdom).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { api } from '@/lib/api/client'

import { MatchKillDistanceSection } from './MatchKillDistanceSection'
import { MATCH_VIEW_TEXT } from './i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchKillDistancePlayer, MatchScoreboardRow } from '@/lib/api/types'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

// La porte de TITRE lit GET /titles/{slug}/capabilities : l'endpoint est servi par ce mock,
// et chaque test choisit le statut de `film.kill_positions`.
vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn() },
  getApiTitleSlug: () => 'halo_infinite',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const apiGet = vi.mocked(api.get)

function titreMesureLesPositions(statut: 'supported' | 'not_exposed') {
  apiGet.mockResolvedValue({
    title_slug: 'halo_infinite',
    schema_version: 1,
    capabilities: { 'film.kill_positions': statut },
  })
}

/** Le rendu passe par les providers : la porte de titre ouvre une requête. */
const render = renderWithProviders

const PLAYERS: MatchKillDistancePlayer[] = [
  {
    xuid: 'xuid(1)',
    weapons: [
      {
        weapon_key: 'hinf_br75',
        label: 'BR75',
        label_en: 'BR75',
        measured_kills: 2,
        avg_distance_m: 12.35,
        min_distance_m: 3.1,
        max_distance_m: 21.6,
      },
      {
        weapon_key: 'hinf_repulsor',
        label: '',
        label_en: '',
        measured_kills: 1,
        avg_distance_m: 5,
        min_distance_m: 5,
        max_distance_m: 5,
      },
    ],
  },
  {
    xuid: 'xuid(2)',
    weapons: [
      {
        weapon_key: 'hinf_sniper',
        label: 'Fusil de précision',
        label_en: 'Sniper Rifle',
        measured_kills: 1,
        avg_distance_m: 40,
        min_distance_m: 40,
        max_distance_m: 40,
      },
    ],
  },
]

const SCOREBOARD: MatchScoreboardRow[] = [
  { xuid: 'xuid(1)', gamertag: 'Alice', kills: 8, is_me: true } as MatchScoreboardRow,
  { xuid: 'xuid(2)', gamertag: 'Bob', kills: 5, is_me: false } as MatchScoreboardRow,
]

beforeEach(() => {
  apiGet.mockReset()
  titreMesureLesPositions('supported')
})

afterEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('MatchKillDistanceSection', () => {
  it('rend la section, un en-tête par joueur (X/Y frags mesurés) et un graphe par joueur', async () => {
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    await waitFor(() => expect(screen.getByText('Distance par arme')).toBeInTheDocument())
    expect(screen.getByText('Alice — 3/8 frags mesurés')).toBeInTheDocument()
    expect(screen.getByText('Bob — 1/5 frags mesurés')).toBeInTheDocument()
    // La réserve de couverture reste au pied : le bâton ne prétend pas à l'exhaustivité.
    expect(screen.getByText(/couverture partielle/)).toBeInTheDocument()
  })

  it('replie sur le xuid quand le joueur est absent du scoreboard', async () => {
    render(<MatchKillDistanceSection players={PLAYERS} scoreboard={[]} t={MATCH_VIEW_TEXT.fr} />)
    await waitFor(() => expect(screen.getByText('xuid(1) — 3/0 frags mesurés')).toBeInTheDocument())
  })

  // PORTE 2 (le MATCH) : le titre mesure les positions, mais pas sur CE match-là.
  it("SANS frag mesuré, la section DIT pourquoi au lieu de disparaître (retour user 02/09)", async () => {
    for (const players of [[] as MatchKillDistancePlayer[], null, undefined]) {
      const { unmount } = render(
        <MatchKillDistanceSection players={players} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
      )
      await waitFor(() => expect(screen.getByText('Distance par arme')).toBeInTheDocument())
      expect(screen.getByText(/Distances non mesurées sur ce match/)).toBeInTheDocument()
      unmount()
    }
  })

  // PAS DE FLASH (revue C-R1, constat C4) : sur un titre qui declare la cle `not_exposed`,
  // la carte ne doit pas etre peinte PUIS retiree. L assertion est SYNCHRONE, juste apres le
  // rendu, donc avant que la reponse des capabilities arrive : c est exactement la sonde qui
  // passait avant la correction (fail-open pendant le chargement) et qui echoue depuis.
  it('titre sans la cle : la carte n est JAMAIS peinte, pas meme le temps de la requete', async () => {
    titreMesureLesPositions('not_exposed')
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(screen.queryByText('Distance par arme')).not.toBeInTheDocument()
    await waitFor(() => expect(apiGet).toHaveBeenCalled())
    expect(screen.queryByText('Distance par arme')).not.toBeInTheDocument()
  })

  // MEME REGLE SUR UN TITRE SUPPORTE : rien pendant la requete, tout apres. Le cout du
  // fail-closed est symetrique et borne (une requete par titre et par session), et ce volet
  // interdit de « corriger » le flash en re-ouvrant la porte pendant le chargement.
  it('titre AVEC la cle : rien au premier rendu, la carte apres la reponse', async () => {
    titreMesureLesPositions('supported')
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(screen.queryByText('Distance par arme')).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('Distance par arme')).toBeInTheDocument())
  })

  // PORTE 1 (le TITRE) : rien a esperer, jamais — la section n existe pas.
  it("titre sans film.kill_positions : RIEN n'est rendu, pas même l'état vide", async () => {
    titreMesureLesPositions('not_exposed')
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    // La porte est FAIL-OPEN tant que la réponse n'est pas là (pas de clignotement au
    // montage) : on attend donc la DISPARITION, pas un état initial.
    await waitFor(() => expect(screen.queryByText('Distance par arme')).not.toBeInTheDocument())
    expect(screen.queryByText(/Distances non mesurées sur ce match/)).not.toBeInTheDocument()
  })

  it('titre EN en locale en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.en} />,
    )
    await waitFor(() => expect(screen.getByText('Distance by weapon')).toBeInTheDocument())
  })
})
