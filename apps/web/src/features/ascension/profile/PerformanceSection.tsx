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
import { useProfileI18n } from './useProfileI18n'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { useAppShellStore } from '@/stores/appShellStore'
import { localizeTierLabel } from '@/lib/skillTiers'

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
  const { t } = useProfileI18n()
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <header className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase text-muted-foreground">
          {t('profile.section.performance.title')}
        </h2>
        <TrendBadge trend={muTrend} />
      </header>

      <TierBlock rating={skillRating} />

      <ComponentsBreakdown components={components} />
    </section>
  )
}

function TierBlock({ rating }: { rating: SkillRatingSnapshot }) {
  const { t } = useProfileI18n()
  const locale = useAppShellStore((s) => s.locale)
  if (!rating.label) {
    return <p className="text-sm text-muted-foreground">{t('profile.performance.empty')}</p>
  }
  const progressPct = Math.round((rating.progress_ratio ?? 0) * 100)
  // tier_name (EN) / tier_name_fr (FR) portés par le DTO ; label/next_tier_label
  // sont composés en EN côté backend (package profile locale-agnostic) → localisés ici.
  return (
    <div>
      <div className="flex items-baseline justify-between">
        <span className="text-2xl font-bold">{(locale === 'en' ? rating.tier_name : rating.tier_name_fr) || rating.label}</span>
        <span className="font-mono text-xs text-muted-foreground">
          {t('profile.performance.mu_sigma', {
            mu: rating.mu.toFixed(0),
            sigma: rating.sigma.toFixed(0),
          })}
        </span>
      </div>
      <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="h-2 rounded-full bg-primary"
          style={{ width: `${progressPct}%` }}
          aria-label={t('profile.performance.progress_aria', {
            pct: progressPct,
            label: localizeTierLabel(rating.label, locale) ?? rating.label,
          })}
        />
      </div>
      {rating.next_tier_label && (
        <p className="mt-1 text-xs text-muted-foreground">
          {t('profile.performance.next_tier')}{' '}
          <span className="font-semibold">{localizeTierLabel(rating.next_tier_label, locale)}</span>
          {rating.gap_to_next !== undefined && (
            <>
              {' · '}
              {t('profile.performance.gap_to_next', { gap: rating.gap_to_next.toFixed(0) })}
            </>
          )}
        </p>
      )}
    </div>
  )
}

function TrendBadge({ trend }: { trend?: LOWESSTrend }) {
  const { t } = useProfileI18n()
  if (!trend || !trend.Slope || !trend.Window) return null
  const positive = (trend.Slope ?? 0) > 0
  // Soft-négatif neutralisé (Phase A) : le rouge d'alerte passe en token neutre
  // `info` — le « cap » prominent et son interprétation vivent dans CoachFocusCard.
  const colorToken = positive ? 'outcome-win' : 'info'
  return (
    <span
      className="rounded-full px-2 py-0.5 text-xs font-medium"
      style={{
        backgroundColor: `color-mix(in srgb, ${tokenCssVar(colorToken)} 20%, transparent)`,
        color: tokenCssVar(colorToken),
      }}
      title={t('profile.performance.trend_tooltip', {
        slope: trend.Slope?.toFixed(2),
        window: trend.Window ?? 0,
      })}
    >
      {t(
        positive
          ? 'profile.performance.trend_positive'
          : 'profile.performance.trend_negative',
      )}
    </span>
  )
}

function ComponentsBreakdown({ components }: { components?: LUSRComponentBreakdown[] }) {
  const { t } = useProfileI18n()
  if (!components?.length) return null
  const hasData = components.some(
    (c) => c.current_avg > 0 || c.personal_top_20 > 0 || c.target_for_tier > 0,
  )
  if (!hasData) {
    return (
      <p className="text-xs text-muted-foreground">
        {t('profile.performance.components_unavailable')}
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
  const { t } = useProfileI18n()
  const currentPct = Math.round(component.current_avg * 100)
  const targetPct = Math.round(component.target_for_tier * 100)
  const top20Pct = Math.round(component.personal_top_20 * 100)
  const labelKey = `profile.lusr.${component.name}` as ProfileManifestKey
  return (
    <li className="text-xs">
      <div className="flex justify-between">
        <span className="flex items-center gap-1 font-medium">
          {t(labelKey)}
          <TrendArrow trend={component.trend} t={t} />
        </span>
        <span className="font-mono text-muted-foreground">
          {t('profile.performance.component_current_target', {
            current: currentPct,
            target: targetPct,
          })}
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
          aria-label={t('profile.performance.aria_target')}
        />
        <div
          className="absolute inset-y-0 w-px bg-foreground/30"
          style={{ left: `${top20Pct}%` }}
          aria-label={t('profile.performance.aria_top20')}
        />
      </div>
    </li>
  )
}

const TREND_THRESHOLD = 0.02

interface TrendArrowProps {
  trend: number
  t: ReturnType<typeof useProfileI18n>['t']
}

function TrendArrow({ trend, t }: TrendArrowProps) {
  if (Math.abs(trend) < TREND_THRESHOLD) return null
  const positive = trend > 0
  const color = tokenCssVar(positive ? 'outcome-win' : 'outcome-loss')
  const title = t(positive ? 'profile.performance.trend_positive' : 'profile.performance.trend_negative')
  const path = positive ? 'M3 11 L8 4 L13 11' : 'M3 5 L8 12 L13 5'
  return (
    <svg
      width="10"
      height="10"
      viewBox="0 0 16 16"
      fill="none"
      stroke={color}
      strokeWidth="2.25"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <title>{title}</title>
      <path d={path} />
    </svg>
  )
}
