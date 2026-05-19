/**
 * IdentitySection — Section A1 du profil (rôles + radar 6 axes).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.1.
 */
import { useMemo } from 'react'
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'
import type { PlayerProfile, RadarAxisInsight } from '@/lib/playerProfile'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { useProfileI18n } from '../../hooks/useProfileI18n'

interface IdentitySectionProps {
  profile: PlayerProfile
}

export function IdentitySection({ profile }: IdentitySectionProps) {
  const { t } = useProfileI18n()

  const series: RadarSeriesPayload[] = [
    {
      key: profile.user_id,
      labelKey: 'profile.you',
      axes: (profile.radar_axes ?? []).map((a) => ({
        axis: a.axis,
        value: a.value,
        raw: a.raw ?? 0,
      })),
    },
  ]

  const axisLabels = useMemo<Record<string, string>>(
    () => ({
      combat: t('profile.axis.combat'),
      survival: t('profile.axis.survival'),
      support: t('profile.axis.support'),
      score: t('profile.axis.score'),
      objective: t('profile.axis.objective'),
      impact: t('profile.axis.impact'),
    }),
    [t],
  )
  const youLabel = t('profile.you')

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <header className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase text-muted-foreground">
          {t('profile.section.identity.title')}
        </h2>
        <RoleBadge dominant={profile.dominant_role} secondary={profile.secondary_role} />
      </header>

      {series[0].axes.length > 0 ? (
        <RadarChart
          series={series}
          axisLabels={axisLabels}
          height={300}
          emptyMessage={t('profile.section.identity.empty')}
          seriesNameResolver={() => youLabel}
        />
      ) : (
        <p className="text-sm text-muted-foreground">{t('profile.section.identity.empty')}</p>
      )}

      <InsightsRow strengths={profile.strengths} improvements={profile.improvement_areas} />
    </section>
  )
}

interface RoleBadgeProps {
  dominant?: string
  secondary?: string
}

function RoleBadge({ dominant, secondary }: RoleBadgeProps) {
  const { t } = useProfileI18n()
  if (!dominant) return null
  const dominantKey = `profile.role.${dominant}` as ProfileManifestKey
  const secondaryKey = secondary ? (`profile.role.${secondary}` as ProfileManifestKey) : null
  return (
    <div className="text-right text-xs">
      <div>
        <span className="text-muted-foreground">{t('profile.role.dominant_label')}</span>{' '}
        <span className="font-semibold">{t(dominantKey)}</span>
      </div>
      {secondaryKey && (
        <div className="text-muted-foreground">
          {t('profile.role.secondary_label')} {t(secondaryKey)}
        </div>
      )}
    </div>
  )
}

interface InsightsRowProps {
  strengths?: RadarAxisInsight[]
  improvements?: RadarAxisInsight[]
}

function InsightsRow({ strengths, improvements }: InsightsRowProps) {
  const { t } = useProfileI18n()
  if (!strengths?.length && !improvements?.length) return null
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <InsightColumn title={t('profile.insights.strengths')} items={strengths} tone="positive" />
      <InsightColumn
        title={t('profile.insights.improvements')}
        items={improvements}
        tone="negative"
      />
    </div>
  )
}

interface InsightColumnProps {
  title: string
  items?: RadarAxisInsight[]
  tone: 'positive' | 'negative'
}

function InsightColumn({ title, items, tone }: InsightColumnProps) {
  const { t } = useProfileI18n()
  if (!items?.length) return null
  return (
    <div>
      <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">{title}</h3>
      <ul className="space-y-1">
        {items.map((insight) => {
          const axisKey = `profile.axis.${insight.axis}` as ProfileManifestKey
          return (
            <li
              key={insight.axis}
              className={`flex items-center justify-between text-sm ${
                tone === 'positive' ? 'text-foreground' : 'text-muted-foreground'
              }`}
            >
              <span>{t(axisKey)}</span>
              <span className="font-mono text-xs">{insight.value.toFixed(0)}</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
