/**
 * MatchKillDistanceSection.test.tsx — le graphe de distance par arme (réouverture DEC-8,
 * 2026-09-02). Couvre : rendu nominal (en-têtes de joueur, section présente), l'ÉTAT VIDE
 * QUI SE DIT (retour user « je ne vois rien du tout » — la section explique au lieu de
 * disparaître), et le repli gamertag→xuid. Le graphe lui-même (bâton min→max + losange de
 * moyenne) est testé PUR dans `_killDistanceChart.test.ts` — ici ECharts est mocké (jsdom).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchKillDistanceSection } from './MatchKillDistanceSection'
import { MATCH_VIEW_TEXT } from './i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchKillDistancePlayer, MatchScoreboardRow } from '@/lib/api/types'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

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

afterEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('MatchKillDistanceSection', () => {
  it('rend la section, un en-tête par joueur (X/Y frags mesurés) et un graphe par joueur', () => {
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(screen.getByText('Distance par arme')).toBeInTheDocument()
    expect(screen.getByText('Alice — 3/8 frags mesurés')).toBeInTheDocument()
    expect(screen.getByText('Bob — 1/5 frags mesurés')).toBeInTheDocument()
    // La réserve de couverture reste au pied : le bâton ne prétend pas à l'exhaustivité.
    expect(screen.getByText(/couverture partielle/)).toBeInTheDocument()
  })

  it('replie sur le xuid quand le joueur est absent du scoreboard', () => {
    render(<MatchKillDistanceSection players={PLAYERS} scoreboard={[]} t={MATCH_VIEW_TEXT.fr} />)
    expect(screen.getByText('xuid(1) — 3/0 frags mesurés')).toBeInTheDocument()
  })

  it("SANS frag mesuré, la section DIT pourquoi au lieu de disparaître (retour user 02/09)", () => {
    for (const players of [[] as MatchKillDistancePlayer[], null, undefined]) {
      const { unmount } = render(
        <MatchKillDistanceSection players={players} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
      )
      expect(screen.getByText('Distance par arme')).toBeInTheDocument()
      expect(screen.getByText(/Distances non mesurées sur ce match/)).toBeInTheDocument()
      unmount()
    }
  })

  it('titre EN en locale en', () => {
    useAppShellStore.setState({ locale: 'en' })
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.en} />,
    )
    expect(screen.getByText('Distance by weapon')).toBeInTheDocument()
  })
})
