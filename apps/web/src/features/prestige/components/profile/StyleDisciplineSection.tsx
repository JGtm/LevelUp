/**
 * StyleDisciplineSection — Section A2 du profil (style FK/FD + engagement).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.2.
 */
import type { EngagementSnapshot, StyleSignature } from '@/lib/playerProfile'

const STYLE_LABELS_FR: Record<string, { title: string; subtitle: string }> = {
  opportunistic_finisher: {
    title: 'Finisseur opportuniste',
    subtitle: 'Tu cherches le dernier kill au bon moment.',
  },
  overextended: {
    title: 'Trop avancé',
    subtitle: 'Tu meurs souvent avant les autres — recule un peu.',
  },
  hyper_engaged: {
    title: 'Hyper engagé',
    subtitle: 'Premier au combat, premier dans la mêlée.',
  },
  passive: {
    title: 'Plus prudent',
    subtitle: 'Style mesuré — peu de premiers contacts.',
  },
}

const ENGAGEMENT_LABELS_FR: Record<string, string> = {
  low: 'Calme',
  regular: 'Régulier',
  high: 'Soutenu',
  intense: 'Intense',
}

interface StyleDisciplineSectionProps {
  style: StyleSignature
  engagement: EngagementSnapshot
}

export function StyleDisciplineSection({
  style,
  engagement,
}: StyleDisciplineSectionProps) {
  const styleMeta = style.style_key ? STYLE_LABELS_FR[style.style_key] : undefined

  return (
    <section className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <StyleCard style={style} meta={styleMeta} />
      <EngagementCard engagement={engagement} />
    </section>
  )
}

interface StyleCardProps {
  style: StyleSignature
  meta?: { title: string; subtitle: string }
}

function StyleCard({ style, meta }: StyleCardProps) {
  return (
    <article className="rounded-lg border border-border bg-card p-4">
      <h3 className="text-xs font-semibold uppercase text-muted-foreground">
        Style de jeu
      </h3>
      <p className="mt-1 text-lg font-semibold">
        {meta?.title ?? 'Encore peu marqué'}
      </p>
      {meta && <p className="text-sm text-muted-foreground">{meta.subtitle}</p>}
      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <Pair label="Premiers kills" value={style.first_kill_count} />
        <Pair label="Premiers morts" value={style.first_death_count} />
        <Pair
          label="Ratio FK/FD"
          value={style.fkfd_ratio > 0 ? style.fkfd_ratio.toFixed(2) : '—'}
          full
        />
      </dl>
    </article>
  )
}

interface EngagementCardProps {
  engagement: EngagementSnapshot
}

function EngagementCard({ engagement }: EngagementCardProps) {
  return (
    <article className="rounded-lg border border-border bg-card p-4">
      <h3 className="text-xs font-semibold uppercase text-muted-foreground">
        Engagement
      </h3>
      <p className="mt-1 text-lg font-semibold">
        {ENGAGEMENT_LABELS_FR[engagement.tier] ?? engagement.tier}
      </p>
      <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-2 rounded-full bg-primary"
          style={{ width: `${Math.min(100, engagement.score)}%` }}
          aria-label={`Score d'engagement ${engagement.score.toFixed(0)} sur 100`}
        />
      </div>
      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <Pair
          label="Matchs / jour"
          value={engagement.matches_per_day_avg.toFixed(1)}
        />
        <Pair label="Plus long écart" value={`${engagement.max_gap_days} j`} />
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
