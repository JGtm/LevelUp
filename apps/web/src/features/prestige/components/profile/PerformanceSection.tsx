/**
 * PerformanceSection — Section B du profil (tier LUSR + 8 composantes).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.3.
 */
import { tokenCssVar } from '@/lib/accessibility'
import type {
  LOWESSTrend,
  LUSRComponentBreakdown,
  SkillRatingSnapshot,
} from '@/lib/playerProfile'

const COMPONENT_LABELS_FR: Record<string, string> = {
  kills_vs_expected: 'Kills vs attendus',
  deaths_vs_expected: 'Morts vs attendues',
  win_factor: 'Facteur de victoire',
  damage_efficiency: 'Efficacité dégâts',
  accuracy_delta: 'Précision (delta)',
  medal_exploit: 'Exploits / médailles',
  offensive_conversion: 'Conversion offensive',
  defensive_resistance: 'Résistance défensive',
}

interface PerformanceSectionProps {
  skillRating: SkillRatingSnapshot
  components?: LUSRComponentBreakdown[]
  muTrend?: LOWESSTrend
}

export function PerformanceSection({
  skillRating,
  components,
  muTrend,
}: PerformanceSectionProps) {
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <header className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase text-muted-foreground">
          Performance
        </h2>
        <TrendBadge trend={muTrend} />
      </header>

      <TierBlock rating={skillRating} />

      <ComponentsBreakdown components={components} />
    </section>
  )
}

function TierBlock({ rating }: { rating: SkillRatingSnapshot }) {
  if (!rating.label) {
    return (
      <p className="text-sm text-muted-foreground">
        Pas encore assez de matchs ratés pour estimer ton tier.
      </p>
    )
  }
  const progressPct = Math.round((rating.progress_ratio ?? 0) * 100)
  return (
    <div>
      <div className="flex items-baseline justify-between">
        <span className="text-2xl font-bold">{rating.tier_name_fr || rating.label}</span>
        <span className="font-mono text-xs text-muted-foreground">
          μ {rating.mu.toFixed(0)} · σ {rating.sigma.toFixed(0)}
        </span>
      </div>
      <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-2 rounded-full bg-primary"
          style={{ width: `${progressPct}%` }}
          aria-label={`${progressPct}% du sous-palier ${rating.label}`}
        />
      </div>
      {rating.next_tier_label && (
        <p className="mt-1 text-xs text-muted-foreground">
          Prochain palier : <span className="font-semibold">{rating.next_tier_label}</span>
          {rating.gap_to_next !== undefined && (
            <> · +{rating.gap_to_next.toFixed(0)} pts à gagner</>
          )}
        </p>
      )}
    </div>
  )
}

function TrendBadge({ trend }: { trend?: LOWESSTrend }) {
  if (!trend || !trend.Slope || !trend.Window) return null
  const positive = (trend.Slope ?? 0) > 0
  const label = positive ? 'En progression' : 'En recul'
  const colorToken = positive ? 'outcome-win' : 'outcome-loss'
  return (
    <span
      className="rounded-full px-2 py-0.5 text-xs font-medium"
      style={{
        backgroundColor: `color-mix(in srgb, ${tokenCssVar(colorToken)} 20%, transparent)`,
        color: tokenCssVar(colorToken),
      }}
      title={`Pente LOWESS ${trend.Slope?.toFixed(2)} sur ${trend.Window ?? 0} pts`}
    >
      {label}
    </span>
  )
}

function ComponentsBreakdown({ components }: { components?: LUSRComponentBreakdown[] }) {
  if (!components?.length) return null
  const hasData = components.some(
    (c) => c.current_avg > 0 || c.personal_top_20 > 0 || c.target_for_tier > 0,
  )
  if (!hasData) {
    return (
      <p className="text-xs text-muted-foreground">
        Détail des 8 composantes : données non disponibles pour cette fenêtre.
      </p>
    )
  }
  return (
    <ul className="space-y-2">
      {components.map((c) => (
        <ComponentRow key={c.name} component={c} />
      ))}
    </ul>
  )
}

function ComponentRow({ component }: { component: LUSRComponentBreakdown }) {
  const currentPct = Math.round(component.current_avg * 100)
  const targetPct = Math.round(component.target_for_tier * 100)
  const top20Pct = Math.round(component.personal_top_20 * 100)
  return (
    <li className="text-xs">
      <div className="flex justify-between">
        <span className="font-medium">
          {COMPONENT_LABELS_FR[component.name] ?? component.name}
        </span>
        <span className="font-mono text-muted-foreground">
          {currentPct}% / cible {targetPct}%
        </span>
      </div>
      <div className="relative mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className="absolute inset-y-0 left-0 bg-primary/70"
          style={{ width: `${currentPct}%` }}
        />
        <div
          className="absolute inset-y-0 w-px bg-foreground/50"
          style={{ left: `${targetPct}%` }}
          aria-label="Cible pour le prochain palier"
        />
        <div
          className="absolute inset-y-0 w-px bg-foreground/30"
          style={{ left: `${top20Pct}%` }}
          aria-label="Top 20% personnel"
        />
      </div>
    </li>
  )
}
