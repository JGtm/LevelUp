/**
 * ExplorerTargetSampleStats — section "Sur N matchs joués ensemble" de l'encart.
 *
 * 3 colonnes (donut kill types / CombatYieldBar / 4 stats agrégées) +
 * OutcomeBar dessous.
 *
 * Calcul local depuis common_matches (DuckDB), indépendant des tokens Halo.
 * Affichée seulement quand `sampleStats != null && sample_size > 0`.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetSampleStats } from '@/lib/api/types'

interface ExplorerTargetSampleStatsProps {
  sampleStats: ExplorerTargetSampleStats
}

interface DonutSlice {
  label: string
  count: number
  token: SemanticToken
}

function fmtPctRatio(value: number, locale: string): string {
  return `${(value * 100).toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

function fmtNumber(value: number, locale: string, fractionDigits = 2): string {
  return value.toLocaleString(locale, { maximumFractionDigits: fractionDigits })
}

export function ExplorerTargetSampleStats({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const numberLocale = appLocale === 'en' ? 'en-US' : 'fr-FR'
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, appLocale, values)
  const dashLabel = t('explorer.target_profile.value_unavailable')

  // Slices du donut (5 catégories). "Other" = kills - somme des catégories
  // détaillées, avec garde-fou si la somme dépasse les kills (overlap).
  const tracked = sampleStats.headshot_kills + sampleStats.melee_kills +
    sampleStats.power_weapon_kills + sampleStats.grenade_kills
  const other = Math.max(0, sampleStats.kills - tracked)
  const donutSlices: DonutSlice[] = [
    { label: t('explorer.target_profile.kill_type_headshot'), count: sampleStats.headshot_kills, token: 'chart-series-1' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_melee'), count: sampleStats.melee_kills, token: 'chart-series-2' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_power_weapon'), count: sampleStats.power_weapon_kills, token: 'chart-series-3' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_grenade'), count: sampleStats.grenade_kills, token: 'chart-series-4' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_other'), count: other, token: 'chart-series-5' as SemanticToken },
  ].filter((s) => s.count > 0)

  const damagePerKill = sampleStats.kills > 0 ? sampleStats.damage_dealt / sampleStats.kills : null
  const damagePerDeath = sampleStats.deaths > 0 ? sampleStats.damage_taken / sampleStats.deaths : null

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">
          {t('explorer.target_profile.section_sample_title', { count: sampleStats.sample_size })}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid gap-4 lg:grid-cols-3">
          {/* Colonne 1 : Donut kill types */}
          <div className="flex flex-col items-center gap-2">
            <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
              {t('explorer.target_profile.label_kill_types')}
            </span>
            {donutSlices.length > 0 ? (
              <KillTypesDonut slices={donutSlices} />
            ) : (
              <span className="text-xs text-muted-foreground">{dashLabel}</span>
            )}
          </div>

          {/* Colonne 2 : CombatYieldBar */}
          <div className="flex flex-col items-center gap-2">
            <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
              {t('explorer.target_profile.label_combat_yield')}
            </span>
            <CombatYieldBar
              offensiveConversion={sampleStats.offensive_conversion ?? null}
              defensiveResistance={sampleStats.defensive_resistance ?? null}
              damagePerKill={damagePerKill}
              damagePerDeath={damagePerDeath}
            />
          </div>

          {/* Colonne 3 : Stats agrégées */}
          <div className="grid grid-cols-2 gap-2">
            <SmallTile label={t('explorer.target_profile.label_kda')} value={sampleStats.kda != null ? fmtNumber(sampleStats.kda, numberLocale) : dashLabel} />
            <SmallTile label={t('explorer.target_profile.label_accuracy')} value={sampleStats.accuracy != null ? fmtPctRatio(sampleStats.accuracy, numberLocale) : dashLabel} />
            <SmallTile label={t('explorer.target_profile.label_headshot_rate')} value={sampleStats.headshot_rate != null ? fmtPctRatio(sampleStats.headshot_rate, numberLocale) : dashLabel} />
            <SmallTile
              label={t('explorer.target_profile.label_medals')}
              value={t('explorer.target_profile.label_medals_value', { total: sampleStats.total_medals, unique: sampleStats.unique_medals })}
            />
          </div>
        </div>

        {(sampleStats.wins + sampleStats.losses + sampleStats.draws) > 0 && (
          <OutcomeBar wins={sampleStats.wins} draws={sampleStats.draws} losses={sampleStats.losses} />
        )}
      </CardContent>
    </Card>
  )
}

interface SmallTileProps {
  label: string
  value: string
}

function SmallTile({ label, value }: SmallTileProps) {
  return (
    <div className="flex flex-col gap-1 rounded-md border border-border bg-card px-2 py-1.5">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
      <span className="text-sm font-semibold text-foreground">{value}</span>
    </div>
  )
}

interface KillTypesDonutProps {
  slices: DonutSlice[]
}

function KillTypesDonut({ slices }: KillTypesDonutProps) {
  const total = slices.reduce((acc, s) => acc + s.count, 0)
  if (total === 0) return null

  const radius = 50
  const strokeWidth = 18
  const innerRadius = radius - strokeWidth / 2
  const circumference = 2 * Math.PI * innerRadius

  // Pré-calcule les dashOffset (cumulative) pour éviter une mutation pendant
  // le render (interdit par react-hooks/incompatible-library).
  const sliceMetrics = slices.reduce<Array<{ slice: DonutSlice; dashLength: number; dashOffset: number }>>(
    (acc, slice) => {
      const previous = acc[acc.length - 1]
      const accumulated = previous ? previous.dashOffset / -circumference + previous.dashLength / circumference : 0
      const fraction = slice.count / total
      return [
        ...acc,
        {
          slice,
          dashLength: fraction * circumference,
          dashOffset: -accumulated * circumference,
        },
      ]
    },
    [],
  )
  return (
    <div className="flex flex-col items-center gap-2">
      <svg width="120" height="120" viewBox="0 0 120 120">
        <circle cx="60" cy="60" r={innerRadius} fill="none" stroke={tokenCssVar('perf-tier-5')} strokeWidth={strokeWidth} opacity="0.15" />
        {sliceMetrics.map((m, i) => (
          <circle
            key={i}
            cx="60"
            cy="60"
            r={innerRadius}
            fill="none"
            stroke={tokenCssVar(m.slice.token)}
            strokeWidth={strokeWidth}
            strokeDasharray={`${m.dashLength} ${circumference - m.dashLength}`}
            strokeDashoffset={m.dashOffset}
            transform="rotate(-90 60 60)"
          />
        ))}
        <text x="60" y="60" textAnchor="middle" dominantBaseline="central" className="fill-foreground text-base font-semibold">
          {total}
        </text>
      </svg>
      <ul className="flex flex-wrap justify-center gap-x-3 gap-y-1 text-2xs text-muted-foreground">
        {slices.map((slice, i) => (
          <li key={i} className="flex items-center gap-1">
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: tokenCssVar(slice.token) }} aria-hidden="true" />
            <span>{slice.label}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
