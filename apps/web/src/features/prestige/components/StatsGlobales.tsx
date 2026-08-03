/**
 * StatsGlobales — bandeau de stats agrégées du parcours Prestige.
 *
 * Référence : Axe 8 du plan conceptuel (onglet Mon parcours, section "stats globales").
 *
 * Affiche :
 *  - Nombre de défis complétés par palier (Normal/Heroic/Legendary/Mythic)
 *  - Taux de complétion global (filtre auto/libre)
 *  - Top 5 métriques travaillées
 *
 * Phase 5 minimale : version statique calculée client-side depuis la liste de
 * défis complétés. Une vraie API d'agrégation viendra avec le calage post-alpha.
 */
import { useMemo, useState } from 'react'
import type { Challenge, Tier, ChallengeMode } from '@/lib/prestige'
import { TIER_COLORS, TIER_LABELS_FR } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { useMetricLabel } from '@/lib/i18n/metricLabel'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

interface StatsGlobalesProps {
  /** Tous les défis du joueur (actifs + terminés). */
  challenges: Challenge[]
}

type ModeFilter = 'all' | ChallengeMode

const TIERS: Tier[] = ['normal', 'heroic', 'legendary', 'mythic']

export function StatsGlobales({ challenges }: StatsGlobalesProps) {
  const [modeFilter, setModeFilter] = useState<ModeFilter>('all')
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const filtered = useMemo(
    () => (modeFilter === 'all' ? challenges : challenges.filter((c) => c.mode === modeFilter)),
    [challenges, modeFilter],
  )

  const stats = useMemo(() => computeStats(filtered), [filtered])

  return (
    <section className="space-y-3">
      <header className="flex items-center justify-between">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Stats globales
        </h3>
        <ModeFilterToggle value={modeFilter} onChange={setModeFilter} />
      </header>

      {/* Distribution par palier */}
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {TIERS.map((tier) => (
          <TierCount
            key={tier}
            tier={tier}
            completed={stats.completedByTier[tier] ?? 0}
            created={stats.createdByTier[tier] ?? 0}
          />
        ))}
      </div>

      {/* Taux de complétion global */}
      <div className="rounded-md border border-border bg-card p-3">
        <p className="text-xs text-muted-foreground">{t('common.prestige.completion_rate')}</p>
        <p className="mt-0.5 text-xl font-semibold">
          {stats.totalCreated === 0
            ? '—'
            : `${Math.round((stats.totalCompleted / stats.totalCreated) * 100)} %`}
        </p>
        <p className="text-xs text-muted-foreground">
          {stats.totalCompleted} complétés / {stats.totalCreated} créés
        </p>
      </div>

      {/* Top métriques */}
      <div className="rounded-md border border-border bg-card p-3">
        <p className="mb-2 text-xs text-muted-foreground">{t('common.prestige.top_metrics')}</p>
        {stats.topMetrics.length === 0 ? (
          <p className="text-xs text-muted-foreground">{t('common.prestige.no_data_for_filter')}</p>
        ) : (
          <ul className="space-y-1">
            {stats.topMetrics.map(({ metric, count }) => (
              <TopMetricRow key={metric} metric={metric} count={count} />
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}

// Une ligne = un composant : le libellé de métrique vient d'un hook
// (useMetricLabel), interdit dans le .map() de la liste.
function TopMetricRow({ metric, count }: { metric: string; count: number }) {
  const label = useMetricLabel(metric)
  return (
    <li className="flex items-center justify-between text-sm">
      <span className="font-medium">{label}</span>
      <span className="text-xs text-muted-foreground">{count}</span>
    </li>
  )
}

interface ModeFilterToggleProps {
  value: ModeFilter
  onChange: (v: ModeFilter) => void
}

function ModeFilterToggle({ value, onChange }: ModeFilterToggleProps) {
  // Libellé "Libre" servi par cadence.free des TOML (source unique — les deux
  // titres le déclarent ; plus aucun repli FR local depuis le 2026-08-02).
  // Le hook reste ICI, hors du .map() ci-dessous.
  const libreLabel = useAssetLabel('cadence', 'free')
  const labelOf = (opt: ModeFilter) => {
    if (opt === 'all') return 'Tous'
    if (opt === 'libre') return libreLabel
    return 'Pilote'
  }
  return (
    <div className="flex items-center rounded-md border border-border bg-card p-0.5 text-xs">
      {(['all', 'libre', 'pilote'] as const).map((opt) => (
        <button
          key={opt}
          type="button"
          onClick={() => onChange(opt)}
          className={[
            'rounded px-2 py-1 transition-colors',
            value === opt
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground',
          ].join(' ')}
        >
          {labelOf(opt)}
        </button>
      ))}
    </div>
  )
}

function TierCount({
  tier,
  completed,
  created,
}: {
  tier: Tier
  completed: number
  created: number
}) {
  const color = TIER_COLORS[tier]
  // Phase 4 plan finition multi-titres : libellé du tier via TOML, fallback dict.
  const tierLabelFromTOML = useAssetLabel('challenge_tier', tier)
  const tierLabel = tierLabelFromTOML !== tier ? tierLabelFromTOML : TIER_LABELS_FR[tier]
  return (
    <div
      className="rounded-md border bg-card p-3 text-center"
      style={{ borderLeftColor: color, borderLeftWidth: 3 }}
    >
      <p className="text-2xs uppercase tracking-wider text-muted-foreground">
        {tierLabel}
      </p>
      <p className="mt-1 text-xl font-bold" style={{ color }}>
        {completed}
      </p>
      <p className="text-2xs text-muted-foreground">/ {created} créés</p>
    </div>
  )
}

interface ComputedStats {
  totalCreated: number
  totalCompleted: number
  completedByTier: Record<Tier, number>
  createdByTier: Record<Tier, number>
  topMetrics: Array<{ metric: string; count: number }>
}

function computeStats(challenges: Challenge[]): ComputedStats {
  const completedByTier: Record<Tier, number> = {
    normal: 0, heroic: 0, legendary: 0, mythic: 0,
  }
  const createdByTier: Record<Tier, number> = {
    normal: 0, heroic: 0, legendary: 0, mythic: 0,
  }
  const metricCount: Record<string, number> = {}
  let totalCreated = 0
  let totalCompleted = 0

  for (const c of challenges) {
    totalCreated++
    if (c.tier) {
      createdByTier[c.tier] = (createdByTier[c.tier] ?? 0) + 1
    }
    if (c.status === 'completed') {
      totalCompleted++
      if (c.tier) {
        completedByTier[c.tier] = (completedByTier[c.tier] ?? 0) + 1
      }
    }
    metricCount[c.metric] = (metricCount[c.metric] ?? 0) + 1
  }

  const topMetrics = Object.entries(metricCount)
    .map(([metric, count]) => ({ metric, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 5)

  return {
    totalCreated,
    totalCompleted,
    completedByTier,
    createdByTier,
    topMetrics,
  }
}
