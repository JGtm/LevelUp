/**
 * Tests PerformanceSection — focus sur la sparkline de tendance LUSR 90 j
 * (DEC-5/D2). La sparkline monte ECharts via ChartCard (lazy import
 * echarts-for-react) ; on mocke echarts-for-react car jsdom n'a pas de canvas
 * (cf. mémoire reference_echarts_jsdom_test_mock). Le stub rend l'option en JSON
 * pour vérifier les valeurs lissées + le nom de série localisé.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SkillRatingSnapshot, SkillTrendPoint } from '@/lib/playerProfile'

// Stub echarts : rend l'option en JSON (permet d'inspecter les datapoints).
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

// i18n : retourne la clé (suffit pour vérifier le rendu structurel + aria).
vi.mock('./useProfileI18n', () => ({
  useProfileI18n: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}::${Object.entries(vars).map(([k, v]) => `${k}=${v}`).join(',')}` : key,
    locale: 'fr' as const,
  }),
}))

import { PerformanceSection } from './PerformanceSection'

const RATING: SkillRatingSnapshot = {
  tier_name: 'Onyx',
  tier_name_fr: 'Onyx',
  sub_tier: 3,
  label: 'Onyx III',
  mu: 1850,
  sigma: 95,
  progress_ratio: 0.72,
}

const TREND: SkillTrendPoint[] = [
  { date: '2026-05-01', value: 1800.4 },
  { date: '2026-05-05', value: 1812.6 },
  { date: '2026-05-09', value: 1825.2 },
  { date: '2026-05-14', value: 1840.9 },
]

describe('PerformanceSection — sparkline tendance LUSR (D2)', () => {
  it('rend la sparkline quand ≥ 2 points, avec valeurs arrondies et série localisée', async () => {
    renderWithProviders(<PerformanceSection skillRating={RATING} skillTrend={TREND} />)
    // Lazy import echarts-for-react → Suspense : attendre la résolution du stub.
    const stub = await screen.findByTestId('echarts-stub')
    const json = stub.textContent ?? ''
    // Valeurs LISSÉES arrondies (DEC-6 : jamais le μ brut ; ici arrondi points).
    expect(json).toContain('1800')
    expect(json).toContain('1841')
    // Nom de série localisé via le manifest (clé retournée par le stub i18n).
    expect(json).toContain('profile.performance.trend_chart_series')
  })

  it('expose un libellé accessible (role img + aria manifesté)', () => {
    renderWithProviders(<PerformanceSection skillRating={RATING} skillTrend={TREND} />)
    expect(
      screen.getByRole('img', { name: 'profile.performance.trend_chart_aria' }),
    ).toBeInTheDocument()
  })

  it("n'affiche RIEN quand skillTrend est absent (état vide propre)", () => {
    renderWithProviders(<PerformanceSection skillRating={RATING} />)
    expect(screen.queryByTestId('echarts-stub')).not.toBeInTheDocument()
  })

  it("n'affiche RIEN avec un seul point (< 2 → pas de graphe cassé)", () => {
    renderWithProviders(
      <PerformanceSection skillRating={RATING} skillTrend={[{ date: '2026-05-01', value: 1800 }]} />,
    )
    expect(screen.queryByTestId('echarts-stub')).not.toBeInTheDocument()
  })
})
