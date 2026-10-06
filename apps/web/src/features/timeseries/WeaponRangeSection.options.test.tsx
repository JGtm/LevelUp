/**
 * WeaponRangeSection.options.test — CE QUE LA SECTION INJECTE VRAIMENT DANS ECHARTS.
 *
 * POURQUOI CE FICHIER EXISTE (revue adversariale du lot 5, 2026-09-06). Les tests purs
 * (`components/charts/weaponRangeChart.test.ts`) prouvent que le constructeur honore ce qu'on leur INJECTE ; le test de composant, lui, monte la section
 * avec ECharts mocké et ne regarde jamais l'option produite. Entre les deux, personne ne
 * vérifiait le CÂBLAGE : échanger les encres des frags et des morts, les libellés
 * d'infobulle, ou passer `tc.axisLabel` là où `tc.card` est attendu laissait tout vert.
 *
 * Ici on capture la prop `option` que `ChartCard` finit par passer à `echarts-for-react` —
 * donc l'option RÉELLE, après `useWeaponRangeOption`, `useMemo` et résolution des tokens.
 * La palette d'accessibilité est appliquée pour de bon (`applyPalette`) : sans elle,
 * `resolveToken` rend la chaîne vide et deux couleurs échangées restent égales.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { applyPalette, _resetActivePalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import type { SynthesisWeaponRange, WeaponRangeSide } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { WeaponRangeSection } from './WeaponRangeSection'
import { WEAPON_RANGE_ROW_PX } from '@/components/charts/weaponRangeChart'

/** Les options passées à ECharts (un seul graphe : la portée). */
const captured: { option: EChartsOption; style: { height?: number } }[] = []

interface EChartsOption {
  tooltip: { formatter: (p: unknown) => string }
  series: {
    name?: string
    itemStyle?: { color?: string; borderColor?: string }
    data?: unknown[]
    renderItem?: (p: { dataIndex: number }, api: FakeApi) => RenderedGroup
  }[]
}
interface FakeApi {
  coord(p: [number, number]): number[]
}
interface RenderedGroup {
  children: { type: string; style: Record<string, string> }[]
}
/** API `renderItem` factice, identique à celle du test pur : x doublé, bandes de 34 px. */
const API: FakeApi = { coord: ([v, i]) => [v * 2, 100 + i * WEAPON_RANGE_ROW_PX] }

vi.mock('echarts-for-react', () => ({
  default: (props: { option: EChartsOption; style: { height?: number } }) => {
    captured.push({ option: props.option, style: props.style })
    return <div data-testid="echarts-mock" />
  },
}))


const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  // min_m / max_m : ajoutés au contrat le 2026-09-17 (profil d'armes du Face-à-face).
  // Publiés pour l'infobulle, JAMAIS tracés — la Synthèse les ignore. Ils encadrent
  // simplement le bâton p10 -> p90 de cette fixture.
  min_m: 2,
  max_m: 14,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

/** Deux armes : la première mesurée des DEUX côtés (c'est elle qu'on inspecte). */
const RANGE: SynthesisWeaponRange = {
  weapons: [
    {
      weapon_key: 'hinf_br75',
      label: 'Fusil de combat BR75',
      label_en: 'BR75 Battle Rifle',
      kills: side({ measured: 281, p10: 7.1, median: 13.6, p90: 24.9 }),
      deaths: side({ measured: 402, p10: 8.9, median: 16.4, p90: 29.7 }),
    },
    {
      weapon_key: 'hinf_commando',
      label: 'Commando VK78',
      label_en: 'VK78 Commando',
      kills: side({ measured: 64 }),
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

/** Monte la section et rend l'option du graphe de portée. */
async function mountAndCapture(range: SynthesisWeaponRange = RANGE) {
  renderWithProviders(<WeaponRangeSection range={range} />)
  // Le graphe est chargé en `lazy` : l'attendre le rend déterministe. S'il
  // n'arrive pas (série vide -> état « Aucune donnée »), ce `find` échoue, et c'est voulu.
  const mocks = await screen.findAllByTestId('echarts-mock')
  expect(mocks).toHaveLength(1)
  return { range: captured[0] }
}

beforeEach(() => {
  captured.length = 0
  useAppShellStore.setState({ locale: 'fr' })
  _resetActivePalette()
  applyPalette(defaultPalette, 'default')
})

describe('les options injectées — graphe de portée', () => {
  it('le bâton du haut porte l’encre des frags, celui du bas celle des morts', async () => {
    const { range } = await mountAndCapture()
    // `ordered` est la liste RENVERSÉE : dataIndex 1 = la première arme du backend, la seule
    // mesurée des deux côtés. Les enfants sortent dans l'ordre : bâton frags, losange, bâton
    // morts, losange.
    const children = range.option.series[0].renderItem!({ dataIndex: 1 }, API).children
    expect(children.map((c) => c.type)).toEqual(['rect', 'polygon', 'rect', 'polygon'])
    // Encres des stats de combat (famille dédiée, 2026-09-17) : `stat-kills` / `stat-deaths`.
    expect(children[0].style.fill).toBe(defaultPalette['stat-kills'])
    expect(children[2].style.fill).toBe(defaultPalette['stat-deaths'])
    expect(children[0].style.fill).not.toBe(children[2].style.fill)
  })

  it('le losange de médiane porte l’encre du palier, cerné du fond de CARTE', async () => {
    const { range } = await mountAndCapture()
    const children = range.option.series[0].renderItem!({ dataIndex: 1 }, API).children
    const tc = getEChartsThemeColors()
    expect(children[1].style.fill).toBe(defaultPalette['perf-tier-2'])
    // Le contour détache le losange de son bâton : c'est le fond de carte, JAMAIS le gris
    // des libellés d'axe.
    expect(children[1].style.stroke).toBe(tc.card)
    expect(children[1].style.stroke).not.toBe(tc.axisLabel)
  })

  it('l’infobulle nomme le bon côté : 281 frags, 402 morts', async () => {
    const { range } = await mountAndCapture()
    const html = range.option.tooltip.formatter({ dataIndex: 1 })
    expect(html).toContain('Frags du joueur — 281')
    expect(html).toContain('Morts du joueur — 402')
  })

  it('la hauteur du graphe suit le nombre de lignes (48 + 34 × n)', async () => {
    const { range } = await mountAndCapture()
    expect(range.style.height).toBe(48 + WEAPON_RANGE_ROW_PX * 2)
  })

  it('la série passée à ChartCard n’est pas vide — sinon le graphe cède la place', async () => {
    await mountAndCapture()
    // `findAllByTestId` ci-dessus a déjà tranché : avec une série vide, `ChartCard` rend son
    // état « Aucune donnée » et aucun canvas n'existe. On le dit explicitement.
    expect(screen.queryByTestId('chart-card-empty')).not.toBeInTheDocument()
  })
})
