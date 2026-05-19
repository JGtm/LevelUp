/**
 * StyleDisciplineSection — Section A2 du profil (style FK/FD + engagement).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.2.
 */
import type { EngagementSnapshot, StyleSignature } from '@/lib/playerProfile'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { useProfileI18n } from '../../hooks/useProfileI18n'

interface StyleDisciplineSectionProps {
  style: StyleSignature
  engagement: EngagementSnapshot
}

export function StyleDisciplineSection({
  style,
  engagement,
}: StyleDisciplineSectionProps) {
  return (
    <section className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <StyleCard style={style} />
      <EngagementCard engagement={engagement} />
    </section>
  )
}

function StyleCard({ style }: { style: StyleSignature }) {
  const { t } = useProfileI18n()
  const titleKey = style.style_key
    ? (`profile.style.${style.style_key}.title` as ProfileManifestKey)
    : null
  const subtitleKey = style.style_key
    ? (`profile.style.${style.style_key}.subtitle` as ProfileManifestKey)
    : null
  return (
    <article className="rounded-lg border border-border bg-card p-4">
      <h3 className="text-xs font-semibold uppercase text-muted-foreground">
        {t('profile.style.title')}
      </h3>
      <p className="mt-1 text-lg font-semibold">
        {titleKey ? t(titleKey) : t('profile.style.empty')}
      </p>
      {subtitleKey && <p className="text-sm text-muted-foreground">{t(subtitleKey)}</p>}
      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <Pair label={t('profile.style.first_kills')} value={style.first_kill_count} />
        <Pair label={t('profile.style.first_deaths')} value={style.first_death_count} />
        <Pair
          label={t('profile.style.fkfd_ratio')}
          value={style.fkfd_ratio > 0 ? style.fkfd_ratio.toFixed(2) : '—'}
          full
        />
      </dl>
    </article>
  )
}

function EngagementCard({ engagement }: { engagement: EngagementSnapshot }) {
  const { t } = useProfileI18n()
  const tierKey = `profile.engagement.tier.${engagement.tier}` as ProfileManifestKey
  return (
    <article className="rounded-lg border border-border bg-card p-4">
      <h3 className="text-xs font-semibold uppercase text-muted-foreground">
        {t('profile.engagement.title')}
      </h3>
      <p className="mt-1 text-lg font-semibold">{t(tierKey)}</p>
      <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-2 rounded-full bg-primary"
          style={{ width: `${Math.min(100, engagement.score)}%` }}
          aria-label={t('profile.engagement.score_aria', { score: engagement.score.toFixed(0) })}
        />
      </div>
      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <Pair
          label={t('profile.engagement.matches_per_day')}
          value={engagement.matches_per_day_avg.toFixed(1)}
        />
        <Pair
          label={t('profile.engagement.max_gap')}
          value={t('profile.engagement.gap_days', { days: engagement.max_gap_days })}
        />
      </dl>
    </article>
  )
}

interface PairProps {
  label: string
  value: number | string
  full?: boolean
}

function Pair({ label, value, full }: PairProps) {
  return (
    <div className={full ? 'col-span-2' : ''}>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-mono font-semibold">{value}</dd>
    </div>
  )
}
