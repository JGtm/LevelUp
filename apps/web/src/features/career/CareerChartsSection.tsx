/**
 * CareerChartsSection — assembler des 4 charts ECharts de la page Carrière.
 *
 * Sub-composants (1 fichier par chart, voir audit #6 god-file split) :
 *   career.01 — CareerRankGaugeChart      (CareerChartsSection.gauges.tsx)
 *   career.02 — CareerHeroGaugeChart      (CareerChartsSection.gauges.tsx)
 *   career.03 — CareerXpHistoryChart      (CareerChartsSection.xpHistory.tsx)
 *   career.04 — CareerLusrEvolutionChart  (CareerChartsSection.lusrEvolution.tsx)
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale as toIntlLocale } from '@/lib/formatters'
import type {
  CareerHistoryPoint,
  CareerLusrCheckpoint,
  HeroProgress,
  CareerSummary,
  CareerProjections,
  FriendXPHistory,
} from '@/lib/api/types'
import { CareerRankGaugeChart, CareerHeroGaugeChart } from './CareerChartsSection.gauges'
import { CareerXpHistoryChart } from './CareerChartsSection.xpHistory'
import { CareerLusrEvolutionChart } from './CareerChartsSection.lusrEvolution'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'

export interface CareerChartsSectionProps {
  xpHistory: CareerHistoryPoint[]
  lusrCheckpoints: CareerLusrCheckpoint[]
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  projections: CareerProjections | null
  friendsXpHistory?: FriendXPHistory[]
  /** Colonne droite optionnelle affichée à côté des charts XP (sidebar achievements). */
  rightSlot?: React.ReactNode
  /** Colonne gauche optionnelle affichée à côté du chart Évolution LUSR / CSR. */
  lusrLeftSlot?: React.ReactNode
}

export function CareerChartsSection({
  xpHistory,
  lusrCheckpoints,
  summary,
  heroProgress,
  projections,
  friendsXpHistory,
  rightSlot,
  lusrLeftSlot,
}: CareerChartsSectionProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const intlLocale = toIntlLocale(locale)
  return (
    <div className="space-y-4" data-testid="career-charts-section">
      {/* career.01 + career.02 + career.03 à gauche | sidebar Succès Xbox à droite */}
      <div className={rightSlot ? 'grid grid-cols-1 gap-4 xl:grid-cols-[1fr_288px]' : undefined}>
        <div className="min-w-0 space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <CareerRankGaugeChart summary={summary} locale={locale} intlLocale={intlLocale} />
            <CareerHeroGaugeChart heroProgress={heroProgress} locale={locale} intlLocale={intlLocale} />
          </div>
          <CareerXpHistoryChart
            xpHistory={xpHistory}
            projections={projections}
            friendsXpHistory={friendsXpHistory ?? []}
            locale={locale}
            heroProgress={heroProgress}
          />
        </div>
        {rightSlot && (
          <div className="relative">
            <div className="xl:absolute xl:inset-0">
              {rightSlot}
            </div>
          </div>
        )}
      </div>

      {/* career.04 — Évolution LUSR / CSR, optionnellement avec Classements à gauche.
          Graphe LUSR gaté sur `lusr` ; le slot gauche (CareerRankingBlock, gating
          CSR/LUSR interne) reste affiché pour un titre `ranked` sans `lusr`. */}
      {lusrLeftSlot ? (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[0.9fr_2fr]">
          <div className="min-w-0 h-full">{lusrLeftSlot}</div>
          <div className="min-w-0">
            <FeatureGate capability="lusr">
              <CareerLusrEvolutionChart lusrCheckpoints={lusrCheckpoints} locale={locale} />
            </FeatureGate>
          </div>
        </div>
      ) : (
        <FeatureGate capability="lusr">
          <CareerLusrEvolutionChart lusrCheckpoints={lusrCheckpoints} locale={locale} />
        </FeatureGate>
      )}
    </div>
  )
}
