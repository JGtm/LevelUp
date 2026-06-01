/**
 * ExplorerCombatProfile — section "Profil de combat" de l'Explorer (mode Joueur).
 *
 * 5 graphes sur les N derniers matchs PvP du joueur cible (Firefight exclu côté
 * backend) : G1 FDA + Frags/Morts/Assists, G2 dégâts infligés/subis (empilés),
 * G3 score + placement (axe inversé), G4 folie max + frags parfaits, G5 donut
 * répartition des modes. Données déjà servies par target_profile.combat_profile.
 *
 * Couleurs uniquement via tokens. Cf. PLAN_explorer_combat_profile_charts.md.
 */
import { useMemo } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { DonutChart } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartPointDonut } from '@/components/charts/DonutChart'
import type { SemanticToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetRecentMatch } from '@/lib/api/types'
import { CombatFdaChart } from './CombatFdaChart'
import { CombatScorePlacementChart } from './CombatScorePlacementChart'

export interface ExplorerCombatProfileProps {
  matches: ExplorerTargetRecentMatch[]
  locale: string
  t: (key: ExplorerManifestKey) => string
}

const CHART_HEIGHT = 300

export function ExplorerCombatProfile({ matches, locale, t }: ExplorerCombatProfileProps) {
  const { data: fieldMappings } = useFieldMappings()
  const fld = (key: string, fallback: string) =>
    fieldMappings?.fields[key]?.label ?? fallback

  // Copie triée chronologiquement croissante pour les graphes temporels.
  const chrono = useMemo(
    () =>
      [...matches].sort(
        (a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime(),
      ),
    [matches],
  )

  const matchLabel = (startTime: string): string => {
    const d = new Date(startTime)
    if (Number.isNaN(d.getTime())) return startTime
    return d.toLocaleDateString(locale, { day: '2-digit', month: '2-digit' })
  }

  const killsLabel = fld('kills', t('explorer.combat.label_kills'))
  const deathsLabel = fld('deaths', t('explorer.combat.label_deaths'))
  const assistsLabel = fld('assists', t('explorer.combat.label_assists'))
  const damageDealtLabel = fld('damage_dealt', t('explorer.combat.label_damage_dealt'))
  const damageTakenLabel = fld('damage_taken', t('explorer.combat.label_damage_taken'))
  const scoreLabel = fld('personal_score', t('explorer.combat.label_score'))
  const spreeLabel = fld('max_killing_spree', t('explorer.combat.label_max_spree'))
  const perfectLabel = t('explorer.combat.label_perfect_kills')

  // G2 — dégâts infligés + subis (empilés). ChartPointStacked = {category, components}.
  const damageSeries = useMemo<ChartSeries<ChartPointStacked>[]>(
    () => [
      {
        key: 'explorer.combat.damage',
        datapoints: chrono.map((m) => ({
          category: matchLabel(m.start_time),
          components: { [damageDealtLabel]: m.damage_dealt, [damageTakenLabel]: m.damage_taken },
        })),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [chrono, locale, damageDealtLabel, damageTakenLabel],
  )

  // G4 — folie max + frags parfaits (groupées).
  const spreeSeries = useMemo<ChartSeries<ChartPointStacked>[]>(
    () => [
      {
        key: 'explorer.combat.spree',
        datapoints: chrono.map((m) => ({
          category: matchLabel(m.start_time),
          components: { [spreeLabel]: m.max_killing_spree, [perfectLabel]: m.perfect_kills },
        })),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [chrono, locale, spreeLabel, perfectLabel],
  )

  // G5 — répartition des modes (group-by mode_ui sur l'ensemble).
  const modeSeries = useMemo<ChartSeries<ChartPointDonut>[]>(() => {
    const counts = new Map<string, number>()
    for (const m of matches) {
      const key = m.mode_ui?.trim() || t('explorer.combat.mode_unknown')
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    return [
      {
        key: 'explorer.combat.modes',
        datapoints: Array.from(counts, ([name, value]) => ({ name, value })),
      },
    ]
  }, [matches, t])

  if (!matches?.length) return null

  const damageColors: Record<string, SemanticToken> = {
    [damageDealtLabel]: 'divergent-pos',
    [damageTakenLabel]: 'divergent-neg',
  }
  const spreeColors: Record<string, SemanticToken> = {
    [spreeLabel]: 'perf-tier-1',
    [perfectLabel]: 'chart-series-4',
  }

  return (
    <section className="space-y-3" data-testid="explorer-combat-profile">
      <header>
        <h3 className="text-base font-semibold text-foreground">
          {t('explorer.combat.title')}
        </h3>
        <p className="text-sm text-muted-foreground">{t('explorer.combat.subtitle')}</p>
      </header>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* G1 — FDA + Frags/Morts/Assists */}
        <div>
          <h4 className="mb-1 text-sm font-medium text-foreground/80">
            {t('explorer.combat.fda_title')}
          </h4>
          <CombatFdaChart
            matches={chrono}
            height={CHART_HEIGHT}
            labels={{
              kills: killsLabel,
              deaths: deathsLabel,
              assists: assistsLabel,
              fda: t('explorer.combat.axis_fda'),
              yAxisLeft: t('explorer.combat.axis_count'),
              yAxisRight: t('explorer.combat.axis_fda'),
            }}
          />
        </div>

        {/* G2 — Dégâts infligés + subis (empilés) */}
        <div>
          <h4 className="mb-1 text-sm font-medium text-foreground/80">
            {t('explorer.combat.damage_title')}
          </h4>
          <BarStackedChart
            series={damageSeries}
            orientation="horizontal"
            tooltipHideZero
            height={CHART_HEIGHT}
            componentColors={damageColors}
            componentOrder={[damageDealtLabel, damageTakenLabel]}
          />
        </div>

        {/* G3 — Score + placement (axe inversé) */}
        <div>
          <h4 className="mb-1 text-sm font-medium text-foreground/80">
            {t('explorer.combat.score_title')}
          </h4>
          <CombatScorePlacementChart
            matches={chrono}
            height={CHART_HEIGHT}
            labels={{
              score: scoreLabel,
              placement: t('explorer.combat.axis_placement'),
              yAxisLeft: t('explorer.combat.axis_score'),
              yAxisRight: t('explorer.combat.axis_placement'),
            }}
          />
        </div>

        {/* G4 — Folie meurtrière max + frags parfaits */}
        <div>
          <h4 className="mb-1 text-sm font-medium text-foreground/80">
            {t('explorer.combat.spree_title')}
          </h4>
          <BarGroupedChart
            series={spreeSeries}
            height={CHART_HEIGHT}
            componentColors={spreeColors}
            componentOrder={[spreeLabel, perfectLabel]}
          />
        </div>

        {/* G5 — Donut répartition des modes (demi-largeur, centré) */}
        <div className="sm:col-span-2 sm:mx-auto sm:w-1/2">
          <h4 className="mb-1 text-sm font-medium text-foreground/80">
            {t('explorer.combat.modes_title')}
          </h4>
          <DonutChart series={modeSeries} showPercent height={CHART_HEIGHT} />
        </div>
      </div>
    </section>
  )
}
