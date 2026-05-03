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

export function PeriodSessionRail() {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)
  const isAutoSnapping = useGlobalFilterStore((s) => s.isAutoSnappingToLatest)
  const setSessions = useGlobalFilterStore((s) => s.setSessions)
  const goToPrevSession = useGlobalFilterStore((s) => s.goToPrevSession)
  const goToNextSession = useGlobalFilterStore((s) => s.goToNextSession)
  const goToPrevPeriod = useGlobalFilterStore((s) => s.goToPrevPeriod)
  const goToNextPeriod = useGlobalFilterStore((s) => s.goToNextPeriod)

  const locale = (useAppShellStore((s) => s.locale) as Locale) ?? 'fr'
  const t = TEXTS[locale]

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const mode = getRailMode(filterContext, allSessions)

  if (mode.kind === 'hidden') return null

  const baseClass =
    'sticky top-0 z-30 flex h-12 shrink-0 items-center gap-3 border-b border-border bg-background px-4'

  // ---- Multi-session : info + boutons disabled ----
  if (mode.kind === 'multi-session') {
    return (
      <div className={baseClass} role="navigation" aria-label={t.ariaNav}>
        <span
          className="text-sm font-semibold text-foreground"
          title={t.multiSessionTooltip}
        >
          {t.multiSessionLabel(mode.count)}
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

  // ---- Mode session unique ----
  if (mode.kind === 'session') {
    const canGoPrev = mode.index < mode.total - 1
    const canGoNext = mode.index > 0
    const canGoLatest = mode.index !== 0

    function goLatest() {
      if (allSessions.length === 0) return
      setSessions({
        ...(filterContext.sessions ?? { picked_sessions: [], gap_minutes: 120 }),
        picked_sessions: [allSessions[0].session_id],
      })
    }

    return (
      <div className={baseClass} role="navigation" aria-label={t.ariaNav}>
        <div className="flex min-w-0 items-center gap-2">
          <span
            className="truncate text-sm font-semibold text-foreground"
            title={mode.session.label}
          >
            {mode.session.label}
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
          {t.positionLabel(mode.index, mode.total)}
          {mode.session.match_count > 0 && t.matchCountSuffix(mode.session.match_count)}
        </span>
      </div>
    )
  }

  // ---- Mode période (preset 7/30/90j ou custom range) ----
  // mode.kind === 'period'
  const startLabel = mode.period.start_date ? formatDateShort(mode.period.start_date, locale) : '?'
  const endLabel = mode.period.end_date ? formatDateShort(mode.period.end_date, locale) : '?'
  const canGoPrev = !!computePrevWindow(mode.period)
  const canGoNext = !!computeNextWindow(mode.period)

  return (
    <div className={baseClass} role="navigation" aria-label={t.ariaNav}>
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-sm font-semibold text-foreground">
          {t.periodLabel(startLabel, endLabel)}
        </span>
        <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
          {t.periodDuration(mode.durationDays)}
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
