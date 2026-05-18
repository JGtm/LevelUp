/**
 * IdentitySection — Section A1 du profil (rôles + radar 6 axes).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.1.
 */
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'
import type { PlayerProfile, RadarAxisInsight } from '@/lib/playerProfile'

const AXIS_LABELS_FR: Record<string, string> = {
  combat: 'Combat',
  survival: 'Survie',
  support: 'Support',
  score: 'Score',
  objective: 'Objectif',
  impact: 'Impact',
}

const ROLE_LABELS_FR: Record<string, string> = {
  top_killer: 'Tueur en tête',
  survivor: 'Survivant',
  silent_hero: 'Héros silencieux',
  scorer: 'Marqueur',
  objective_runner: 'Coureur d’objectif',
  first_blood: 'Premier sang',
  clutch_finisher: 'Clutch finisher',
  last_casualty: 'Dernier tombé',
  last_group_kill: 'Dernier kill du groupe',
  first_group_death: 'Première mort du groupe',
  false_brother: 'Faux frère',
}

interface IdentitySectionProps {
  profile: PlayerProfile
}

export function IdentitySection({ profile }: IdentitySectionProps) {
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

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <header className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase text-muted-foreground">
          Identité de jeu
        </h2>
        <RoleBadge dominant={profile.dominant_role} secondary={profile.secondary_role} />
      </header>

      {series[0].axes.length > 0 ? (
        <RadarChart
          series={series}
          axisLabels={AXIS_LABELS_FR}
          height={300}
          emptyMessage="Pas de données radar pour cette fenêtre."
          seriesNameResolver={() => 'Toi'}
        />
      ) : (
        <p className="text-sm text-muted-foreground">
          Le radar 6 axes n&apos;est pas encore calculable sur cette fenêtre.
        </p>
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
  if (!dominant) return null
  return (
    <div className="text-right text-xs">
      <div>
        <span className="text-muted-foreground">Rôle dominant :</span>{' '}
        <span className="font-semibold">{ROLE_LABELS_FR[dominant] ?? dominant}</span>
      </div>
      {secondary && (
        <div className="text-muted-foreground">
          Secondaire : {ROLE_LABELS_FR[secondary] ?? secondary}
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
  if (!strengths?.length && !improvements?.length) return null
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <InsightColumn title="Forces" items={strengths} tone="positive" />
      <InsightColumn title="À renforcer" items={improvements} tone="negative" />
    </div>
  )
}

interface InsightColumnProps {
  title: string
  items?: RadarAxisInsight[]
  tone: 'positive' | 'negative'
}

function InsightColumn({ title, items, tone }: InsightColumnProps) {
  if (!items?.length) return null
  return (
    <div>
      <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
        {title}
      </h3>
      <ul className="space-y-1">
        {items.map((insight) => (
          <li
            key={insight.axis}
            className={`flex items-center justify-between text-sm ${
              tone === 'positive' ? 'text-foreground' : 'text-muted-foreground'
            }`}
          >
            <span>{AXIS_LABELS_FR[insight.axis] ?? insight.axis}</span>
            <span className="font-mono text-xs">{insight.value.toFixed(0)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
