/**
 * MatchKillDistanceSection.test.tsx — POC (LOT G.3, 2026-08-30). Couvre :
 * rendu nominal (2 joueurs, 2 armes, frags mesurés + distance), état vide
 * (retourne null), formats FR/EN de la distance moyenne, repli du libellé sur
 * weapon_key, et l'absence de plage min–max sur un kill mesuré unique.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchKillDistanceSection } from './MatchKillDistanceSection'
import { MATCH_VIEW_TEXT } from './i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchKillDistancePlayer, MatchScoreboardRow } from '@/lib/api/types'

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
  it('rend le décompte, la moyenne et la plage min-max par joueur/arme (locale FR)', () => {
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )

    expect(screen.getByText('Distance par arme')).toBeInTheDocument()
    expect(screen.getByText('POC')).toBeInTheDocument()
    // En-tête de groupe joueur : gamertag + frags mesurés / total du scoreboard.
    expect(screen.getByText('Alice — 3/8 frags mesurés')).toBeInTheDocument()
    expect(screen.getByText('Bob — 1/5 frags mesurés')).toBeInTheDocument()
    // Libellé résolu.
    expect(screen.getByText('BR75')).toBeInTheDocument()
    // Distance moyenne FR : virgule décimale + plage min–max (>1 kill mesuré).
    expect(screen.getByText('12,4 m')).toBeInTheDocument()
    expect(screen.getByText('(3,1–21,6 m)')).toBeInTheDocument()
  })

  it("replie le libellé sur weapon_key quand label et label_en sont vides", () => {
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(screen.getByText('hinf_repulsor')).toBeInTheDocument()
  })

  it("n'affiche aucune plage min-max sur un seul kill mesuré", () => {
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    // Le kill unique du répulseur (5 m) ne porte pas de "(5,0–5,0 m)".
    expect(screen.queryByText('(5,0–5,0 m)')).not.toBeInTheDocument()
  })

  it('utilise le libellé EN et le format décimal EN (point) en locale en', () => {
    useAppShellStore.setState({ locale: 'en' })
    render(
      <MatchKillDistanceSection players={PLAYERS} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.en} />,
    )
    expect(screen.getByText('Distance by weapon')).toBeInTheDocument()
    expect(screen.getByText('Sniper Rifle')).toBeInTheDocument()
    expect(screen.getByText('12.4 m')).toBeInTheDocument()
    expect(screen.getByText('(3.1–21.6 m)')).toBeInTheDocument()
  })

  it('replie sur le xuid quand le joueur est absent du scoreboard', () => {
    render(<MatchKillDistanceSection players={PLAYERS} scoreboard={[]} t={MATCH_VIEW_TEXT.fr} />)
    expect(screen.getByText('xuid(1) — 3/0 frags mesurés')).toBeInTheDocument()
  })

  it('ne rend rien quand players est vide', () => {
    const { container } = render(
      <MatchKillDistanceSection players={[]} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('ne rend rien quand players est null', () => {
    const { container } = render(
      <MatchKillDistanceSection players={null} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('ne rend rien quand players est undefined', () => {
    const { container } = render(
      <MatchKillDistanceSection players={undefined} scoreboard={SCOREBOARD} t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
