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
  allTimeLabel: (n: number) => string
  allTimeTooltip: string
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
    allTimeLabel: (n) => `Toutes les sessions (${n})`,
    allTimeTooltip: 'Choisissez une période ou une session via les filtres pour activer la navigation',
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
    allTimeLabel: (n) => `All sessions (${n})`,
    allTimeTooltip: 'Pick a period or session in the filters to enable navigation',
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

/**
 * Formate une session en label localisé long :
 *   FR : « Session du 6 avril 2026 de 21:43 à 23:40 »
 *   EN : « Session of Apr 6, 2026 from 9:43 PM to 11:40 PM »
 *
 * Fallback sur le `label` brut backend si les timestamps sont absents
 * (compat ancien backend qui ne renvoie pas started_at_utc/ended_at_utc).
 */
function formatSessionLabel(
  sessionLabel: string,
  startedAtUTC: string | undefined,
  endedAtUTC: string | undefined,
  locale: Locale,
): string {
  if (!startedAtUTC) return sessionLabel
  try {
    const start = new Date(startedAtUTC)
    if (isNaN(start.getTime())) return sessionLabel
    const dateFmt = new Intl.DateTimeFormat(locale === 'fr' ? 'fr-FR' : 'en-US', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    })
    const timeFmt = new Intl.DateTimeFormat(locale === 'fr' ? 'fr-FR' : 'en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: locale !== 'fr',
    })
    const dateLabel = dateFmt.format(start)
    const startTime = timeFmt.format(start)
    const end = endedAtUTC ? new Date(endedAtUTC) : null
    if (end && !isNaN(end.getTime()) && end.getTime() !== start.getTime()) {
      const endTime = timeFmt.format(end)
      return locale === 'fr'
        ? `Session du ${dateLabel} de ${startTime} à ${endTime}`
        : `Session of ${dateLabel} from ${startTime} to ${endTime}`
    }
    return locale === 'fr'
      ? `Session du ${dateLabel} à ${startTime}`
      : `Session of ${dateLabel} at ${startTime}`
  } catch {
    return sessionLabel
  }
}

// Layout 3-zones : [◀ Précédente | Label centré | Suivante ▶]
// Pas de sticky propre — le parent (NavL2 ou SquadLayout) gère sa propre
// barre sticky qui contient le rail. La border-t/-b délimite visuellement
// le rail des filtres au-dessus et du contenu en dessous.
const RAIL_BASE_CLASS =
  'flex h-12 shrink-0 items-center gap-3 border-t border-b border-border bg-background px-4'

const ZONE_LEFT_CLASS = 'flex shrink-0 items-center gap-1.5'
const ZONE_CENTER_CLASS = 'flex flex-1 items-center justify-center gap-2 min-w-0 text-center'
const ZONE_RIGHT_CLASS = 'flex shrink-0 items-center gap-1.5'

const NAV_BTN_CLASS =
  'rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-30'

/** Composant principal — dispatcher selon le mode (session / multi-session / period). */
export function PeriodSessionRail() {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const resolvedContext = useGlobalFilterStore((s) => s.resolvedContext)

  const locale = (useAppShellStore((s) => s.locale) as Locale) ?? 'fr'
  const t = TEXTS[locale]

  const allSessions = resolvedContext?.session_options?.all_sessions ?? []
  const mode = getRailMode(filterContext, allSessions)

  if (mode.kind === 'hidden') return null
  if (mode.kind === 'all-time') return <AllTimeRail total={mode.total} t={t} />
  if (mode.kind === 'multi-session') return <MultiSessionRail count={mode.count} t={t} />
  if (mode.kind === 'session') {
    return (
      <SessionRail
        session={mode.session}
        index={mode.index}
        total={mode.total}
        locale={locale}
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

// ---------------------------------------------------------------------------
// Frame 3-zones réutilisable : [zone gauche | zone centre | zone droite]
// ---------------------------------------------------------------------------

interface RailFrameProps {
  modeAttr: 'session' | 'multi-session' | 'period' | 'all-time'
  ariaLabel: string
  prev: React.ReactNode
  center: React.ReactNode
  next: React.ReactNode
}

function RailFrame({ modeAttr, ariaLabel, prev, center, next }: RailFrameProps) {
  return (
    <div
      className={RAIL_BASE_CLASS}
      role="navigation"
      aria-label={ariaLabel}
      data-testid="period-session-rail"
      data-mode={modeAttr}
    >
      <div className={ZONE_LEFT_CLASS}>{prev}</div>
      <div className={ZONE_CENTER_CLASS}>{center}</div>
      <div className={ZONE_RIGHT_CLASS}>{next}</div>
    </div>
  )
}

interface NavBtnProps {
  label: string
  title: string
  ariaLabel: string
  enabled: boolean
  onClick: () => void
}

function NavBtn({ label, title, ariaLabel, enabled, onClick }: NavBtnProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!enabled}
      className={NAV_BTN_CLASS}
      title={title}
      aria-label={ariaLabel}
    >
      {label}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Sous-composants par mode
// ---------------------------------------------------------------------------

function AllTimeRail({ total, t }: { total: number; t: RailText }) {
  const disabled = (
    <NavBtn label={t.prev} title={t.allTimeTooltip} ariaLabel={t.ariaPrevSession} enabled={false} onClick={() => {}} />
  )
  return (
    <RailFrame
      modeAttr="all-time"
      ariaLabel={t.ariaNav}
      prev={disabled}
      center={
        <span className="text-sm font-semibold text-muted-foreground" title={t.allTimeTooltip}>
          {t.allTimeLabel(total)}
        </span>
      }
      next={
        <NavBtn label={t.next} title={t.allTimeTooltip} ariaLabel={t.ariaNextSession} enabled={false} onClick={() => {}} />
      }
    />
  )
}

function MultiSessionRail({ count, t }: { count: number; t: RailText }) {
  return (
    <RailFrame
      modeAttr="multi-session"
      ariaLabel={t.ariaNav}
      prev={
        <NavBtn label={t.prev} title={t.multiSessionTooltip} ariaLabel={t.ariaPrevSession} enabled={false} onClick={() => {}} />
      }
      center={
        <span className="text-sm font-semibold text-foreground" title={t.multiSessionTooltip}>
          {t.multiSessionLabel(count)}
        </span>
      }
      next={
        <NavBtn label={t.next} title={t.multiSessionTooltip} ariaLabel={t.ariaNextSession} enabled={false} onClick={() => {}} />
      }
    />
  )
}

interface SessionRailProps {
  session: {
    session_id: string
    label: string
    match_count: number
    started_at_utc?: string
    ended_at_utc?: string
  }
  index: number
  total: number
  locale: Locale
  t: RailText
}

function SessionRail({ session, index, total, locale, t }: SessionRailProps) {
  const formattedLabel = formatSessionLabel(
    session.label,
    session.started_at_utc,
    session.ended_at_utc,
    locale,
  )
  const isAutoSnapping = useGlobalFilterStore((s) => s.isAutoSnappingToLatest)
  const goToPrevSession = useGlobalFilterStore((s) => s.goToPrevSession)
  const goToNextSession = useGlobalFilterStore((s) => s.goToNextSession)

  const canGoPrev = index < total - 1
  const canGoNext = index > 0

  return (
    <RailFrame
      modeAttr="session"
      ariaLabel={t.ariaNav}
      prev={
        <NavBtn
          label={t.prev}
          title={t.prevTitle}
          ariaLabel={t.ariaPrevSession}
          enabled={canGoPrev}
          onClick={goToPrevSession}
        />
      }
      center={
        <>
          <span className="truncate text-sm font-semibold text-foreground" title={session.label}>
            {formattedLabel}
          </span>
          {isAutoSnapping && (
            <span
              className="shrink-0 rounded-full bg-primary/15 px-2 py-0.5 text-[10px] font-medium text-primary"
              title={t.autoTitle}
            >
              {t.auto}
            </span>
          )}
          <span className="shrink-0 text-xs text-muted-foreground" aria-live="polite">
            ({t.positionLabel(index, total)}
            {session.match_count > 0 && t.matchCountSuffix(session.match_count)})
          </span>
        </>
      }
      next={
        <NavBtn
          label={t.next}
          title={t.nextTitle}
          ariaLabel={t.ariaNextSession}
          enabled={canGoNext}
          onClick={goToNextSession}
        />
      }
    />
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
    <RailFrame
      modeAttr="period"
      ariaLabel={t.ariaNav}
      prev={
        <NavBtn
          label={t.prev}
          title={t.prevTitle}
          ariaLabel={t.ariaPrevPeriod}
          enabled={canGoPrev}
          onClick={goToPrevPeriod}
        />
      }
      center={
        <>
          <span className="text-sm font-semibold text-foreground">
            {t.periodLabel(startLabel, endLabel)}
          </span>
          <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
            {t.periodDuration(durationDays)}
          </span>
        </>
      }
      next={
        <NavBtn
          label={t.next}
          title={t.nextTitle}
          ariaLabel={t.ariaNextPeriod}
          enabled={canGoNext}
          onClick={goToNextPeriod}
        />
      }
    />
  )
}
