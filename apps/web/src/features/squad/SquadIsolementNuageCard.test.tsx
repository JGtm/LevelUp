/**
 * Le NUAGE « isolement x couverture » (item 7.7) — ce qu'il montre et ce qu'il tait.
 *
 * Ce que ces tests cadenassent : un état vide n'est jamais un nuage à zéro point ; l'aide ⓘ
 * du titre NOMME les deux planchers (session, échantillon) plutôt que de les recopier en dur
 * côté client ; et les deux langues rendent deux textes.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadEchangeJoueur, SquadIsolementPoint, SquadNuageIsolement } from '@/lib/api/types'

import { SquadIsolementNuageCard } from './SquadIsolementNuageCard'

// jsdom n'a pas de canvas : on mocke echarts-for-react (comme FirstBloodLanes) pour
// capturer l'option ECharts construite, notamment le formatter de tooltip.
const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="isolement-nuage-stub" />
  },
}))

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  captured.length = 0
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const joueurs: SquadEchangeJoueur[] = [
  { xuid: 'x1', gamertag: 'Alice' },
  { xuid: 'x2', gamertag: 'Bob' },
]

function isoCouverture(brut: number, n: number, echantillonFaible: boolean) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: n > 0 ? brut / 3 : 0,
    n,
    echantillon_faible: echantillonFaible,
  }
}

function point(over: Partial<SquadIsolementPoint> = {}): SquadIsolementPoint {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    session_label: '02/09 soir',
    morts_examinees: 10,
    morts_isolees: 4,
    part_isolee: isoCouverture(4, 10, true),
    couverture: isoCouverture(6, 10, true),
    ...over,
  } as SquadIsolementPoint
}

function nuageDe(over: Partial<SquadNuageIsolement> = {}): SquadNuageIsolement {
  return {
    points: [point(), point({ xuid: 'x2', gamertag: 'Bob', session_label: '05/09 soir' })],
    plancher_morts_session: 5,
    plancher_echantillon_faible: 30,
    ...over,
  } as SquadNuageIsolement
}

describe('SquadIsolementNuageCard', () => {
  it('ÉTAT VIDE, jamais un nuage à zéro point, quand aucune session ne franchit le plancher', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe({ points: [] })} joueurs={joueurs} />)
    expect(screen.getByText(/Aucun point mesuré/i)).toBeTruthy()
    expect(screen.queryByTestId('chart-card')).toBeNull()
  })

  it('rend le nuage quand au moins un point franchit le plancher', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    expect(screen.getByTestId('squad-isolement-nuage')).toBeTruthy()
    expect(screen.getByTestId('chart-card')).toBeTruthy()
  })

  it('l’aide ⓘ NOMME les deux planchers (5 et 30), pas de valeur en dur', () => {
    // Les deux paragraphes sont passés en infobulle le 2026-09-09 : ils ne sont dans le DOM
    // qu'une fois l'aide ouverte.
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ plancher_morts_session: 5, plancher_echantillon_faible: 30 })}
        joueurs={joueurs}
      />,
    )
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toContain('5')
    expect(aide).toContain('30')
  })

  it('PARITÉ FR/EN : les deux langues rendent un titre, et deux titres différents', () => {
    const { unmount } = renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />,
    )
    const fr = screen.getByText('Isolement et couverture').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    const en = screen.getByText('Isolation and coverage').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
  })

  it('la réserve « échantillon faible » se lit dans le tooltip d’un point atténué, pas seulement dans son opacité', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          points: [point({ part_isolee: isoCouverture(4, 10, true) })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<{ raw: SquadIsolementPoint }> }>
    }
    const datum = option.series[0].data[0]
    const html = option.tooltip.formatter({ data: datum })
    expect(html).toContain('échantillon faible')
  })

  it('un point non atténué ne porte aucune mention « échantillon faible » dans son tooltip', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          points: [point({ part_isolee: isoCouverture(4, 10, false) })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<{ raw: SquadIsolementPoint }> }>
    }
    const datum = option.series[0].data[0]
    const html = option.tooltip.formatter({ data: datum })
    expect(html).not.toContain('échantillon faible')
  })

  // Lot C3 (D4) : le "gros point" médian par joueur.
  it('sur deux sessions et deux joueurs, deux gros points médians sont émis, à la médiane', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          points: [
            point({
              morts_examinees: 10,
              part_isolee: isoCouverture(2, 10, false),
              couverture: isoCouverture(2, 10, false),
            }),
            point({
              morts_examinees: 20,
              part_isolee: isoCouverture(6, 10, false),
              couverture: isoCouverture(6, 10, false),
            }),
            point({
              xuid: 'x2',
              gamertag: 'Bob',
              morts_examinees: 15,
              part_isolee: isoCouverture(3, 10, false),
              couverture: isoCouverture(3, 10, false),
            }),
          ],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      series: Array<{ name: string; data: Array<{ value: [number, number]; medianRaw?: unknown }> }>
    }
    const medianSeries = option.series.filter((s) => 'medianRaw' in s.data[0])
    expect(medianSeries).toHaveLength(2)
    const aliceMedian = medianSeries.find((s) => s.name === 'Alice')
    // Médiane de [20%, 60%] (2 valeurs) = leur moyenne = 40%.
    expect(aliceMedian?.data[0].value[0]).toBeCloseTo(40)
    expect(aliceMedian?.data[0].value[1]).toBeCloseTo(40)
  })

  it('le tooltip d’un point NOMME son quadrant (quadrantDuPoint rebranché)', async () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<{ raw: SquadIsolementPoint }> }>
    }
    const datum = option.series[0].data[0]
    const html = option.tooltip.formatter({ data: datum })
    expect(html).toContain('Quadrant :')
  })

  it('échantillon faible : cercle POINTILLÉ (bordure en tirets), pas d’opacité réduite', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ points: [point({ part_isolee: isoCouverture(4, 10, true) })] })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      series: Array<{ data: Array<{ itemStyle: Record<string, unknown> }> }>
    }
    const datum = option.series[0].data[0]
    expect(datum.itemStyle.borderType).toBe('dashed')
    expect(datum.itemStyle.opacity).toBeUndefined()
  })

  it('échantillon suffisant : cercle plein, aucune bordure pointillée', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ points: [point({ part_isolee: isoCouverture(12, 40, false) })] })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = captured[captured.length - 1].option as {
      series: Array<{ data: Array<{ itemStyle: Record<string, unknown> }> }>
    }
    const datum = option.series[0].data[0]
    expect(datum.itemStyle.borderType).toBeUndefined()
  })
})
