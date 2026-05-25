/**
 * ProfileRadarSection — Section A1 du Profil de jeu.
 *
 * Affiche le radar 6 axes (Combat/Survival/Support/Score/Objective/Impact),
 * les 3 forces et les 3 axes de progression d'un joueur.
 *
 * Masqué si has_enough_data === false (< 30 matchs).
 */
import type { PlayerProfile, ParticipationAxisValue, RadarAxisInsight } from './types'
import { RadarChart } from '@/components/charts/RadarChart'
import type { AscensionText } from './i18n'

interface ProfileRadarSectionProps {
  profile: PlayerProfile
  t: AscensionText
}

export function ProfileRadarSection({ profile, t }: ProfileRadarSectionProps) {
  const axes = profile.radar_axes ?? []
  const strengths = profile.strengths ?? []
  const improvements = profile.improvement_areas ?? []

  const radarSeries = axes.length > 0
    ? [{
        key: 'self',
        axes: axes.map((a) => ({ axis: axisLabel(a.axis, t), value: a.value, raw: a.raw ?? 0 })),
      }]
    : []

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {profile.dominant_role && (
          <RoleBadge role={profile.dominant_role} primary />
        )}
        {profile.secondary_role && (
          <RoleBadge role={profile.secondary_role} />
        )}
      </div>

      {radarSeries.length > 0 && (
        <div className="h-64">
          <RadarChart series={radarSeries} loading={false} />
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {strengths.length > 0 && (
          <AxisInsightGroup
            title={t.profileStrengths}
            items={strengths}
            variant="strength"
            t={t}
          />
        )}
        {improvements.length > 0 && (
          <AxisInsightGroup
            title={t.profileImprovements}
            items={improvements}
            variant="improvement"
            t={t}
          />
        )}
      </div>
    </div>
  )
}

function RoleBadge({ role, primary = false }: { role: string; primary?: boolean }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
        primary
          ? 'bg-primary/10 text-primary ring-1 ring-primary/20'
          : 'bg-muted text-muted-foreground'
      }`}
    >
      {role}
    </span>
  )
}

function AxisInsightGroup({
  title,
  items,
  variant,
  t,
}: {
  title: string
  items: RadarAxisInsight[]
  variant: 'strength' | 'improvement'
  t: AscensionText
}) {
  return (
    <div className="rounded-md border border-border bg-card p-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </p>
      <ul className="space-y-1">
        {items.map((item) => (
          <li key={item.axis} className="flex items-center justify-between text-sm">
            <span className="font-medium">{axisLabel(item.axis, t)}</span>
            <span
              className={
                variant === 'strength'
                  ? 'font-semibold text-green-600 dark:text-green-400' // color-allow: strength/weakness indicator — CLAUDE.md §20
                  : 'font-semibold text-amber-600 dark:text-amber-400' // color-allow: strength/weakness indicator — CLAUDE.md §20
              }
            >
              {Math.round(item.value)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function axisLabel(axis: string, t: AscensionText): string {
  return t.radarAxis?.[axis] ?? axis
}

// axisLabel helper needs radarAxis map in i18n — added below
export type { ParticipationAxisValue }
