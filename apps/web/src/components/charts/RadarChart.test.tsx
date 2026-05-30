/**
 * Tests buildRadarOption — fonction pure (pas de montage ECharts/canvas).
 * Couvre le tooltip rawInTooltip (radar de frags par match) vs le tooltip
 * normalisé historique (radar participation Escouade).
 */
import { describe, expect, it } from 'vitest'

import { buildRadarOption, type RadarSeriesPayload } from './RadarChart'

const SERIES: RadarSeriesPayload[] = [
  {
    key: 'session',
    axes: [
      { axis: 'spree', value: 64, raw: 6.4 },
      { axis: 'headshots', value: 25, raw: 3 },
      { axis: 'perfect', value: 48, raw: 2.4 },
    ],
    meta: { raw_by_axis: { spree: 6.4, headshots: 3, perfect: 2.4 } },
  },
]

const AXIS_LABELS = { spree: 'Folie meurtrière (moy.)', headshots: 'Tirs à la tête / match', perfect: 'Œil de lynx / match' }

type Formatter = (p: { name: string; value: number[] }) => string

function formatterOf(opt: ReturnType<typeof buildRadarOption>): Formatter {
  return (opt as { tooltip: { formatter: Formatter } }).tooltip.formatter
}

describe('buildRadarOption — tooltip', () => {
  it('rawInTooltip=true → affiche les valeurs brutes par axe (1 décimale)', () => {
    const opt = buildRadarOption(SERIES, {
      axisLabels: AXIS_LABELS,
      seriesNameResolver: () => 'Session',
      rawInTooltip: true,
    })
    const html = formatterOf(opt)({ name: 'Session', value: [64, 25, 48] })
    // Valeurs brutes par match, pas le % normalisé.
    expect(html).toContain('Folie meurtrière (moy.): <b>6.4</b>')
    expect(html).toContain('Tirs à la tête / match: <b>3.0</b>')
    expect(html).toContain('Œil de lynx / match: <b>2.4</b>')
    // Ne doit PAS afficher les valeurs normalisées entières.
    expect(html).not.toContain('<b>64</b>')
  })

  it('rawInTooltip absent → comportement historique (% normalisé entier)', () => {
    const opt = buildRadarOption(SERIES, {
      axisLabels: AXIS_LABELS,
      seriesNameResolver: () => 'Session',
    })
    const html = formatterOf(opt)({ name: 'Session', value: [64, 25, 48] })
    expect(html).toContain('Folie meurtrière (moy.): <b>64</b>')
    expect(html).toContain('Œil de lynx / match: <b>48</b>')
    expect(html).not.toContain('6.4')
  })

  it('rawInTooltip=true, raw_by_axis présent mais clé d’axe manquante → tiret', () => {
    const partial: RadarSeriesPayload[] = [
      { key: 'session', axes: [{ axis: 'spree', value: 50, raw: 0 }], meta: { raw_by_axis: {} } },
    ]
    const opt = buildRadarOption(partial, {
      axisLabels: { spree: 'Folie' },
      seriesNameResolver: () => 'Session',
      rawInTooltip: true,
    })
    const html = formatterOf(opt)({ name: 'Session', value: [50] })
    expect(html).toContain('Folie: <b>—</b>')
  })

  it('rawInTooltip=true mais raw_by_axis totalement absent → fallback % normalisé', () => {
    const noMeta: RadarSeriesPayload[] = [
      { key: 'session', axes: [{ axis: 'spree', value: 50, raw: 0 }], meta: {} },
    ]
    const opt = buildRadarOption(noMeta, {
      axisLabels: { spree: 'Folie' },
      seriesNameResolver: () => 'Session',
      rawInTooltip: true,
    })
    const html = formatterOf(opt)({ name: 'Session', value: [50] })
    expect(html).toContain('Folie: <b>50</b>')
  })
})
