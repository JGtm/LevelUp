/**
 * WeaponRangeSection.options.test — CE QUE LA SECTION INJECTE VRAIMENT DANS ECHARTS.
 *
 * POURQUOI CE FICHIER EXISTE (revue adversariale du lot 5, 2026-09-06). Les tests purs
 * (`components/charts/weaponRangeChart.test.ts`, `_elevationCloudChart.test.ts`) prouvent que les deux
 * constructeurs honorent ce qu'on leur INJECTE ; le test de composant, lui, monte la section
 * avec ECharts mocké et ne regarde jamais l'option produite. Entre les deux, personne ne
 * vérifiait le CÂBLAGE : échanger les encres des frags et des morts, les libellés
 * d'infobulle, ou passer `tc.axisLabel` là où `tc.card` est attendu laissait tout vert.
 *
 * Ici on capture la prop `option` que `ChartCard` finit par passer à `echarts-for-react` —
 * donc l'option RÉELLE, après `useWeaponRangeOption` / `useElevationOption`, `useMemo` et résolution des tokens.
 * La palette d'accessibilité est appliquée pour de bon (`applyPalette`) : sans elle,
 * `resolveToken` rend la chaîne vide et deux couleurs échangées restent égales.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { applyPalette, _resetActivePalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import type { ElevationCloudBlock, SynthesisWeaponRange, WeaponRangeSide } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { WeaponRangeSection } from './WeaponRangeSection'
import { WEAPON_RANGE_ROW_PX } from '@/components/charts/weaponRangeChart'

/** Les options passées à ECharts, dans l'ordre de montage : portée, puis dénivelé. */
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

/**
 * Le nuage de la carte voisine (D25). Un point de chaque côté suffit : ce fichier teste le
 * CÂBLAGE (encres, libellés, infobulle), pas la géométrie — elle a ses tests purs.
 */
const ELEVATION: ElevationCloudBlock = {
  kills: [{ distance_m: 13.6, delta_z_m: 2.4, match_id: 'm1', time_ms: 1000, weapon: 'hinf_br75' }],
  deaths: [{ distance_m: 16.4, delta_z_m: -2.5, match_id: 'm1', time_ms: 2000, weapon: 'hinf_br75' }],
  kills_summary: {
    distance_p25: 7.1,
    distance_p50: 13.6,
    distance_p75: 24.9,
    delta_z_p25: 0.5,
    delta_z_p50: 2,
    delta_z_p75: 4,
    n: 281,
  },
  deaths_summary: {
    distance_p25: 8.9,
    distance_p50: 16.4,
    distance_p75: 29.7,
    delta_z_p25: -4,
    delta_z_p50: -1.9,
    delta_z_p75: 0.5,
    n: 402,
  },
  weapon_labels: { hinf_br75: { label: 'Fusil de combat BR75', label_en: 'BR75 Battle Rifle' } },
  measured_kills: 281,
  total_kills: 1602,
  measured_deaths: 402,
  total_deaths: 1455,
}

/** Monte la section et rend les deux options, dans l'ordre des graphes. */
async function mountAndCapture(range: SynthesisWeaponRange = RANGE) {
  renderWithProviders(<WeaponRangeSection range={range} elevation={ELEVATION} />)
  // Les deux graphes sont chargés en `lazy` : attendre les rend déterministes. S'ils
  // n'arrivent pas (série vide -> état « Aucune donnée »), ce `find` échoue, et c'est voulu.
  const mocks = await screen.findAllByTestId('echarts-mock')
  expect(mocks).toHaveLength(2)
  return { range: captured[0], elevation: captured[1] }
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
    // des libellés d'axe (qui, lui, sert la classe « à niveau » du dénivelé).
    expect(children[1].style.stroke).toBe(tc.card)
    expect(children[1].style.stroke).not.toBe(tc.axisLabel)
  })

  it('l’infobulle nomme le bon côté : 281 frags, 402 morts', async () => {
    const { range } = await mountAndCapture()
    const html = range.option.tooltip.formatter({ dataIndex: 1 })
    expect(html).toContain('Mes frags — 281')
    expect(html).toContain('Mes morts — 402')
  })

  it('la hauteur du graphe suit le nombre de lignes (48 + 34 × n)', async () => {
    const { range, elevation } = await mountAndCapture()
    expect(range.style.height).toBe(48 + WEAPON_RANGE_ROW_PX * 2)
    expect(elevation.style.height).toBe(48 + WEAPON_RANGE_ROW_PX * 2)
  })

  it('la série passée à ChartCard n’est pas vide — sinon les deux graphes cèdent la place', async () => {
    await mountAndCapture()
    // `findAllByTestId` ci-dessus a déjà tranché : avec une série vide, `ChartCard` rend son
    // état « Aucune donnée » et aucun canvas n'existe. On le dit explicitement.
    expect(screen.queryByTestId('chart-card-empty')).not.toBeInTheDocument()
  })
})

describe('les options injectées — nuage de dénivelé (D25)', () => {
  const serieNommee = (option: EChartsOption, name: string) =>
    option.series.find((s) => s.name === name)

  it('les deux côtés portent les encres des stats de combat, comme la portée', async () => {
    // MÊMES ENCRES QUE LA CARTE VOISINE (D25) : le nuage a deux côtés, pas trois classes.
    // L'ancienne rampe « d'en haut / à niveau / d'en bas » est partie avec les barres
    // empilées par arme — la position se lit maintenant sur l'axe des ordonnées.
    const { elevation } = await mountAndCapture()
    expect(serieNommee(elevation.option, 'Mes frags')?.itemStyle?.color).toBe(
      defaultPalette['stat-kills'],
    )
    expect(serieNommee(elevation.option, 'Mes morts')?.itemStyle?.color).toBe(
      defaultPalette['stat-deaths'],
    )
  })

  it('les médianes sont cernées du fond de CARTE, jamais du gris des libellés', async () => {
    const { elevation } = await mountAndCapture()
    const tc = getEChartsThemeColors()
    const mediane = serieNommee(elevation.option, 'médiane frags +2,0 m')
    expect(mediane?.itemStyle?.borderColor).toBe(tc.card)
    expect(mediane?.itemStyle?.borderColor).not.toBe(tc.axisLabel)
  })

  it('l’infobulle d’un point nomme son côté, sa distance, son dénivelé SIGNÉ et son arme', async () => {
    const { elevation } = await mountAndCapture()
    const html = elevation.option.tooltip.formatter({
      value: [13.6, -2.5, 'deaths', 'm1', 'hinf_br75'],
    })
    expect(html).toContain('Mes morts')
    expect(html).toContain('13,6 m')
    // Le signe vient du contrat, il n'est jamais recalculé côté web.
    expect(html).toContain('2,5 m')
    expect(html).toContain('Fusil de combat BR75')
  })
})
