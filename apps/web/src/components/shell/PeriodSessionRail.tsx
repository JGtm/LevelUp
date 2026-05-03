/**
 * PeriodSessionRail — barre sticky de navigation temporelle.
 *
 * Affiche le scope actif (session unique ou période) avec un label central et
 * 2 boutons prev/next pour explorer rapidement l'historique.
 *
 * Modes (via `getRailMode`) :
 *  - session       : 1 session pickée → ◀ Précédente · {label} · Suivante ▶
 *  - multi-session : ≥2 sessions     → label "N sessions" + boutons disabled
 *  - period        : preset ou custom range → ◀ Précédente · {plage dates} · Suivante ▶
 *  - hidden        : all-time / pas de scope → composant retourne null
 *
 * Remplace l'ancien `SessionNavBar` (Stats-only) par une nav universelle visible
 * sur toutes les pages joueur (Stats, Squad, Home, Synthèse).
 */
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { computeNextWindow, computePrevWindow, getRailMode } from '@/features/filters/periodSessionNav'

type Locale = 'fr' | 'en'

interface RailText {
  prev: string
  next: string
  latest: string
  prevTitle: string
  nextTitle: string
  latestTitle: string
  ariaNav: string
  ariaPrevSession: string
  ariaNextSession: string
  ariaLatestSession: string
  ariaPrevPeriod: string
  ariaNextPeriod: string
  positionLabel: (idx: number, total: number) => string
  matchCountSuffix: (n: number) => string
  multiSessionLabel: (n: number) => string
  multiSessionTooltip: string
  periodLabel: (start: string, end: string) => string
  periodDuration: (days: number) => string
  auto: string
  autoTitle: string
}

const TEXTS: Record<Locale, RailText> = {
  fr: {
    prev: '◀ Précédente',
    next: 'Suivante ▶',
    latest: 'Dernière',
    prevTitle: 'Plus ancienne',
    nextTitle: 'Plus récente',
    latestTitle: 'La plus récente',
    ariaNav: 'Navigation période / session',
    ariaPrevSession: 'Session précédente',
    ariaNextSession: 'Session suivante',
    ariaLatestSession: 'Aller à la dernière session',
    ariaPrevPeriod: 'Période précédente',
    ariaNextPeriod: 'Période suivante',
    positionLabel: (idx, total) => `${idx + 1} / ${total}`,
    matchCountSuffix: (n) => ` · ${n} match${n > 1 ? 's' : ''}`,
    multiSessionLabel: (n) => `${n} sessions sélectionnées`,
    multiSessionTooltip: 'Désélectionnez des sessions pour activer la navigation',
    periodLabel: (start, end) => `Période du ${start} au ${end}`,
    periodDuration: (days) => `${days} jour${days > 1 ? 's' : ''}`,
    auto: 'auto',
    autoTitle: 'Sélection automatique : nouvelle session détectée.',
  },
  en: {
    prev: '◀ Previous',
    next: 'Next ▶',
    latest: 'Latest',
    prevTitle: 'Older',
    nextTitle: 'Newer',
    latestTitle: 'Most recent',
    ariaNav: 'Period / session navigation',
    ariaPrevSession: 'Previous session',
    ariaNextSession: 'Next session',
    ariaLatestSession: 'Go to latest session',
    ariaPrevPeriod: 'Previous period',
    ariaNextPeriod: 'Next period',
    positionLabel: (idx, total) => `${idx + 1} / ${total}`,
    matchCountSuffix: (n) => ` · ${n} match${n > 1 ? 'es' : ''}`,
    multiSessionLabel: (n) => `${n} sessions selected`,
    multiSessionTooltip: 'Deselect sessions to enable navigation',
    periodLabel: (start, end) => `From ${start} to ${end}`,
    periodDuration: (days) => `${days} day${days > 1 ? 's' : ''}`,
    auto: 'auto',
    autoTitle: 'Auto-selection: new session detected.',
  },
}

/** Formate une date ISO (YYYY-MM-DD) en label localisé court. */
function formatDateShort(iso: string, locale: Locale): string {
  try {
    const d = new Date(iso + 'T00:00:00Z')
    return d.toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-US', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      timeZone: 'UTC',
    })
  } catch {
    return iso
  }
}

const RAIL_BASE_CLASS =
  'sticky top-0 z-30 flex h-12 shrink-0 items-center gap-3 border-b border-border bg-background px-4'

/** Composant principal — dispatcher selon le mode (session / multi-session / period). */
export function PeriodSessionRail() {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)

  const locale = (useAppShellStore((s) => s.locale) as Locale) ?? 'fr'
  const t = TEXTS[locale]

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const mode = getRailMode(filterContext, allSessions)

  // Sentinelle dev : émet le mode dans la console à chaque rerender pour
  // faciliter le diagnostic ("pourquoi le rail ne s'affiche pas ?"). Disparait
  // en build prod (import.meta.env.DEV est strippé par Vite).
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.debug(
      `[PeriodSessionRail] mode=${mode.kind}`,
      'filter_mode=', filterContext.filter_mode,
      'picked=', filterContext.sessions?.picked_sessions ?? [],
      'period=', filterContext.period,
      'allSessions.length=', allSessions.length,
    )
  }

  if (mode.kind === 'hidden') return null
  if (mode.kind === 'multi-session') return <MultiSessionRail count={mode.count} t={t} />
  if (mode.kind === 'session') {
    return (
      <SessionRail
        session={mode.session}
        index={mode.index}
        total={mode.total}
        t={t}
      />
    )
  }
  // mode.kind === 'period'
  return <PeriodRail period={mode.period} durationDays={mode.durationDays} locale={locale} t={t} />
}

// ---------------------------------------------------------------------------
// Sous-composants par mode
// ---------------------------------------------------------------------------

function MultiSessionRail({ count, t }: { count: number; t: RailText }) {
  return (
    <div
      className={RAIL_BASE_CLASS}
      role="navigation"
      aria-label={t.ariaNav}
      data-testid="period-session-rail"
      data-mode="multi-session"
    >
      <span className="text-sm font-semibold text-foreground" title={t.multiSessionTooltip}>
        {t.multiSessionLabel(count)}
      </span>
      <NavButtons
        prevLabel={t.prev}
        nextLabel={t.next}
        prevTitle={t.multiSessionTooltip}
        nextTitle={t.multiSessionTooltip}
        ariaPrev={t.ariaPrevSession}
        ariaNext={t.ariaNextSession}
        canPrev={false}
        canNext={false}
        onPrev={() => {}}
        onNext={() => {}}
      />
    </div>
  )
}

interface SessionRailProps {
  session: { session_id: string; label: string; match_count: number }
  index: number
  total: number
  t: RailText
}

function SessionRail({ session, index, total, t }: SessionRailProps) {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const isAutoSnapping = useGlobalFilterStore((s) => s.isAutoSnappingToLatest)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const goToPrevSession = useGlobalFilterStore((s) => s.goToPrevSession)
  const goToNextSession = useGlobalFilterStore((s) => s.goToNextSession)

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const canGoPrev = index < total - 1
  const canGoNext = index > 0
  const canGoLatest = index !== 0

  function goLatest() {
    if (allSessions.length === 0) return
    setSessions({
      ...(filterContext.sessions ?? { picked_sessions: [], gap_minutes: 120 }),
      picked_sessions: [allSessions[0].session_id],
    })
  }

  return (
    <div
      className={RAIL_BASE_CLASS}
      role="navigation"
      aria-label={t.ariaNav}
      data-testid="period-session-rail"
      data-mode="session"
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold text-foreground" title={session.label}>
          {session.label}
        </span>
        {isAutoSnapping && (
          <span
            className="shrink-0 rounded-full bg-primary/15 px-2 py-0.5 text-[10px] font-medium text-primary"
            title={t.autoTitle}
          >
            {t.auto}
          </span>
        )}
      </div>
      <NavButtons
        prevLabel={t.prev}
        nextLabel={t.next}
        prevTitle={t.prevTitle}
        nextTitle={t.nextTitle}
        ariaPrev={t.ariaPrevSession}
        ariaNext={t.ariaNextSession}
        canPrev={canGoPrev}
        canNext={canGoNext}
        onPrev={goToPrevSession}
        onNext={goToNextSession}
        extraButton={
          <button
            type="button"
            onClick={goLatest}
            disabled={!canGoLatest}
            className="rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
            title={t.latestTitle}
            aria-label={t.ariaLatestSession}
          >
            {t.latest}
          </button>
        }
      />
      <div className="flex-1" />
      <span className="shrink-0 text-xs text-muted-foreground" aria-live="polite">
        {t.positionLabel(index, total)}
        {session.match_count > 0 && t.matchCountSuffix(session.match_count)}
      </span>
    </div>
  )
}

interface PeriodRailProps {
  period: { start_date: string | null; end_date: string | null }
  durationDays: number
  locale: Locale
  t: RailText
}

function PeriodRail({ period, durationDays, locale, t }: PeriodRailProps) {
  const goToPrevPeriod = useGlobalFilterStore((s) => s.goToPrevPeriod)
  const goToNextPeriod = useGlobalFilterStore((s) => s.goToNextPeriod)

  const startLabel = period.start_date ? formatDateShort(period.start_date, locale) : '?'
  const endLabel = period.end_date ? formatDateShort(period.end_date, locale) : '?'
  const canGoPrev = !!computePrevWindow(period)
  const canGoNext = !!computeNextWindow(period)

  return (
    <div
      className={RAIL_BASE_CLASS}
      role="navigation"
      aria-label={t.ariaNav}
      data-testid="period-session-rail"
      data-mode="period"
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-sm font-semibold text-foreground">
          {t.periodLabel(startLabel, endLabel)}
        </span>
        <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
          {t.periodDuration(durationDays)}
        </span>
      </div>
      <NavButtons
        prevLabel={t.prev}
        nextLabel={t.next}
        prevTitle={t.prevTitle}
        nextTitle={t.nextTitle}
        ariaPrev={t.ariaPrevPeriod}
        ariaNext={t.ariaNextPeriod}
        canPrev={canGoPrev}
        canNext={canGoNext}
        onPrev={goToPrevPeriod}
        onNext={goToNextPeriod}
      />
      <div className="flex-1" />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sous-composants
// ---------------------------------------------------------------------------

interface NavButtonsProps {
  prevLabel: string
  nextLabel: string
  prevTitle: string
  nextTitle: string
  ariaPrev: string
  ariaNext: string
  canPrev: boolean
  canNext: boolean
  onPrev: () => void
  onNext: () => void
  extraButton?: React.ReactNode
}

function NavButtons({
  prevLabel,
  nextLabel,
  prevTitle,
  nextTitle,
  ariaPrev,
  ariaNext,
  canPrev,
  canNext,
  onPrev,
  onNext,
  extraButton,
}: NavButtonsProps) {
  const btnClass =
    'rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30'
  return (
    <div className="flex shrink-0 items-center gap-1.5">
      <button
        type="button"
        onClick={onPrev}
        disabled={!canPrev}
        className={btnClass}
        title={prevTitle}
        aria-label={ariaPrev}
      >
        {prevLabel}
      </button>
      <button
        type="button"
        onClick={onNext}
        disabled={!canNext}
        className={btnClass}
        title={nextTitle}
        aria-label={ariaNext}
      >
        {nextLabel}
      </button>
      {extraButton}
    </div>
  )
}
