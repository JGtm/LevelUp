/**
 * StreakCard — carte d'une streak (série). Partagée entre la page Ascension
 * (mode complet, `StreakDashboard`) et le widget home (`HomeAscensionWidget`,
 * mode `compact`). Composant unique pour éviter un fork dégradé (cf. incidents
 * de divergence home/page).
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §4 + §8.3.
 */
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { formatAscensionDate, formatMultiplier, interpolate, nextPPTier, streakTierProgressPct } from './format'
import { getAscensionText, type AscensionLocale } from './i18n'
import type { Streak } from './types'

/** Classe du badge de statut (couleurs d'état UI génériques, cf. CLAUDE.md §20). */
function streakStatusBadgeClass(status: Streak['status']): string {
  return status === 'active'
    ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300' // color-allow: emerald status badge active (CLAUDE.md §20)
    : status === 'paused'
      ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300' // color-allow: amber status badge paused (CLAUDE.md §20)
      : 'bg-muted text-muted-foreground'
}

/** Unité de période de la série selon le type : jour(s) pour daily_*, semaine(s)
 *  pour weekly_*. Corrige l'imprécision « jours » affichée pour les séries hebdo
 *  (la longueur compte des semaines, distinct du « 5 matchs » de la condition). */
function streakUnit(t: ReturnType<typeof getAscensionText>, type: Streak['type'], n: number): string {
  return interpolate(type.startsWith('weekly') ? t.streakUnitWeek : t.streakUnitDay, { n })
}

interface StreakCardProps {
  streak: Streak
  locale: AscensionLocale
  t: ReturnType<typeof getAscensionText>
  /** Mode compact (widget home) : padding/typo réduits, dates masquées. */
  compact?: boolean
}

export function StreakCard({ streak: s, locale, t, compact = false }: StreakCardProps) {
  const statusLabel = s.status === 'active' ? t.streakActive : s.status === 'paused' ? t.streakPaused : t.streakBroken
  const nextTier = s.status !== 'broken' ? nextPPTier(s.current_length) : null
  const shieldsLeft = s.shields_available - s.shields_used

  if (compact) {
    return (
      <article className="rounded-lg border border-border bg-card p-3">
        {/* Grand chiffre courant (gauche) | colonne [nom + pill multiplicateur
            AU-DESSUS de la barre] | grand seuil cible (droite). Chiffres aux
            extrémités. Plus de badge statut : la barre suffit (verte à 100%, cf.
            demande user). Au max → barre pleine, pas de seuil cible. */}
        <div className="flex items-center gap-2.5">
          <span className="shrink-0 text-2xl font-bold leading-none tabular-nums text-foreground">{s.current_length}</span>
          <div className="min-w-0 flex-1 space-y-1.5">
            <div className="flex items-center justify-between gap-2">
              <h3 className="min-w-0 truncate text-xs font-medium text-foreground">{t.streakTypeName[s.type]}</h3>
              <span
                className="shrink-0 rounded-full bg-info/10 px-2 py-0.5 text-2xs font-semibold tabular-nums text-info"
                title={interpolate(t.streakPPMultiplier, { value: formatMultiplier(nextTier ? nextTier.multiplier : s.pp_multiplier).replace('×', '') })}
              >
                PP {formatMultiplier(nextTier ? nextTier.multiplier : s.pp_multiplier)}
              </span>
            </div>
            <CompositeProgressBar
              value={nextTier ? streakTierProgressPct(s.current_length) : 100}
              fillTestId="home-streak-progress-fill"
            />
          </div>
          {nextTier && (
            <span className="shrink-0 text-2xl font-bold leading-none tabular-nums text-foreground">
              {nextTier.length}
              <span className="ml-1 text-3xs font-medium text-muted-foreground">{streakUnit(t, s.type, nextTier.length)}</span>
            </span>
          )}
        </div>
      </article>
    )
  }

  // ── Mode complet (page Ascension) ──
  return (
    <article className="flex flex-col gap-2 rounded-md border border-border bg-card p-4">
      <header className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium">{t.streakTypeName[s.type]}</h3>
        <span className={`rounded-full px-2 py-0.5 text-2xs font-medium ${streakStatusBadgeClass(s.status)}`}>
          {statusLabel}
        </span>
      </header>

      <div className="flex items-baseline gap-2">
        <span className="text-3xl font-bold">{s.current_length}</span>
        <span className="text-xs text-muted-foreground">{streakUnit(t, s.type, s.current_length)}</span>
      </div>

      <p className="text-xs text-muted-foreground">{interpolate(t.streakBestLength, { n: s.best_length })}</p>

      <dl className="mt-1 space-y-1 text-xs">
        <div>
          <dt className="inline text-muted-foreground">
            {interpolate(t.streakPPMultiplier, { value: formatMultiplier(s.pp_multiplier).replace('×', '') })}
          </dt>
        </div>
        {nextTier && s.status === 'active' && (
          <div className="text-muted-foreground">
            {interpolate(t.streakNextMilestone, {
              n: nextTier.length,
              mul: formatMultiplier(nextTier.multiplier).replace('×', ''),
              unit: streakUnit(t, s.type, nextTier.length),
            })}
          </div>
        )}
        {!nextTier && s.status === 'active' && (
          <div className="text-emerald-700 dark:text-emerald-300"> {/* color-allow: emerald streak active at max multiplier (CLAUDE.md §20) */}
            {t.streakAtMaxMultiplier}
          </div>
        )}
        <div className="text-muted-foreground">{interpolate(t.streakShieldsAvailable, { n: shieldsLeft })}</div>
        {s.status === 'broken' && s.broken_at && (
          <div className="text-muted-foreground">{interpolate(t.streakBrokenAt, { date: formatAscensionDate(s.broken_at, locale) })}</div>
        )}
        <div className="text-muted-foreground">{interpolate(t.streakStarted, { date: formatAscensionDate(s.started_at, locale) })}</div>
      </dl>
    </article>
  )
}
