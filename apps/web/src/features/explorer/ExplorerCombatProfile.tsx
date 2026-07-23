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
import { useMemo, useState } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { DonutChart } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartPointDonut } from '@/components/charts/DonutChart'
import type { SemanticToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetRecentMatch, MedalDigestItem } from '@/lib/api/types'
import { CombatFdaChart } from './CombatFdaChart'
import { CombatScorePlacementChart } from './CombatScorePlacementChart'
import { ExplorerTargetMedals } from './ExplorerTargetMedals'
import type { Locale } from '@/lib/i18n/locale'

export interface ExplorerCombatProfileProps {
  /** ~20 derniers matchs fetchés en LIVE (source affichée par défaut). */
  liveMatches: ExplorerTargetRecentMatch[]
  /** Matchs de la cible en base locale (surtout matchs communs) — toggle "local". */
  localMatches: ExplorerTargetRecentMatch[]
  locale: string
  t: (key: ExplorerManifestKey) => string
  /** Top médailles lifetime de la cible — rendu à droite du donut modes. */
  topMedals?: MedalDigestItem[]
}

const CHART_HEIGHT = 300

// Libellés du toggle de source (inline FR/EN — pas de clé manifest dédiée).
const SOURCE_LABELS: Record<Locale, { live: string; local: string; empty: string }> = {
  fr: { live: 'En direct', local: 'Local', empty: 'Aucune donnée pour cette source.' },
  en: { live: 'Live', local: 'Local', empty: 'No data for this source.' },
}

export function ExplorerCombatProfile({ liveMatches, localMatches, locale, t, topMedals = [] }: ExplorerCombatProfileProps) {
  const hasLive = liveMatches.length > 0
  const hasLocal = localMatches.length > 0
  // Défaut : LIVE (les 20 derniers via l'API) ; bascule sur le local au besoin. Si
  // le live est vide (pas d'auth), on démarre sur le local pour montrer quelque chose.
  const [source, setSource] = useState<'live' | 'local'>(hasLive ? 'live' : 'local')
  const matches = source === 'live' ? liveMatches : localMatches

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

  if (!hasLive && !hasLocal) return null

  const lbl = SOURCE_LABELS[locale === 'en' ? 'en' : 'fr']
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
      <header className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-foreground">
            {t('explorer.combat.title')}
          </h3>
        </div>
        {/* Toggle source : live (défaut) / local. Désactive une source sans données. */}
        <div
          role="group"
          aria-label="Source"
          className="inline-flex shrink-0 rounded-md border border-border bg-muted p-0.5 text-xs"
        >
          <button
            type="button"
            onClick={() => setSource('live')}
            disabled={!hasLive}
            data-testid="combat-source-live"
            className={`rounded px-2 py-1 transition-colors disabled:opacity-40 ${
              source === 'live' ? 'bg-background font-medium text-foreground shadow-sm' : 'text-muted-foreground'
            }`}
          >
            {lbl.live}
          </button>
          <button
            type="button"
            onClick={() => setSource('local')}
            disabled={!hasLocal}
            data-testid="combat-source-local"
            className={`rounded px-2 py-1 transition-colors disabled:opacity-40 ${
              source === 'local' ? 'bg-background font-medium text-foreground shadow-sm' : 'text-muted-foreground'
            }`}
          >
            {lbl.local}
          </button>
        </div>
      </header>

      {matches.length === 0 ? (
        <p className="rounded-lg border border-dashed border-border bg-muted/30 px-4 py-6 text-center text-sm text-muted-foreground">
          {lbl.empty}
        </p>
      ) : (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* G1 — FDA + Frags/Morts/Assists (titre dans la barre ChartCard) */}
        <CombatFdaChart
          title={t('explorer.combat.fda_title')}
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

        {/* G2 — Dégâts infligés + subis (empilés) */}
        <BarStackedChart
          title={t('explorer.combat.damage_title')}
          series={damageSeries}
          orientation="horizontal"
          tooltipHideZero
          height={CHART_HEIGHT}
          componentColors={damageColors}
          componentOrder={[damageDealtLabel, damageTakenLabel]}
        />

        {/* G3 — Score + placement (axe inversé) */}
        <CombatScorePlacementChart
          title={t('explorer.combat.score_title')}
          matches={chrono}
          height={CHART_HEIGHT}
          labels={{
            score: scoreLabel,
            placement: t('explorer.combat.axis_placement'),
            yAxisLeft: t('explorer.combat.axis_score'),
            yAxisRight: t('explorer.combat.axis_placement'),
          }}
        />

        {/* G4 — Folie meurtrière max + frags parfaits */}
        <BarGroupedChart
          title={t('explorer.combat.spree_title')}
          series={spreeSeries}
          height={CHART_HEIGHT}
          componentColors={spreeColors}
          componentOrder={[spreeLabel, perfectLabel]}
        />

        {/* G5 — Donut répartition des modes (gauche, sans légende) + Top médailles (droite) */}
        <div className="grid grid-cols-1 gap-4 sm:col-span-2 sm:grid-cols-2">
          <DonutChart
            title={t('explorer.combat.modes_title')}
            series={modeSeries}
            showPercent
            showLegend={false}
            height={CHART_HEIGHT}
          />
          {topMedals.length > 0 && <ExplorerTargetMedals medals={topMedals} />}
        </div>
      </div>
      )}
    </section>
  )
}
