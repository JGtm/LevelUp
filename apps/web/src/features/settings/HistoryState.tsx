/**
 * HistoryState — l'état de l'historique d'un joueur, par titre suivi, en texte : état, matchs enregistrés et
 * plus ancien, dernière passe, prochaine passe. Présentationnel : l'état arrive tout lu (voir `SyncNowCard`).
 *
 * Le libellé de l'état et celui d'une passe arrivent résolus du serveur ; les dates et les nombres se
 * formatent ici dans la langue de l'interface. Une passe en échec se signale par le jeton `destructive`.
 */
import type { ReactNode } from 'react'
import { formatDate, formatNumber, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import type { HistorySyncStatus, TitleHistoryStatus } from './syncNowQueries'
import { historyStateLabel, lastPassAddedLine, lastPassLine, nextPassText } from './syncNowLogic'
import type { SyncNowText } from './syncNowText'

interface HistoryStateProps {
  status: HistorySyncStatus
  cycleRunning: boolean
  locale: Locale
  t: SyncNowText
}

function Row({ label, testId, children }: { label: string; testId: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-foreground" data-testid={testId}>
        {children}
      </dd>
    </>
  )
}

function TitleHistory({
  title,
  status,
  cycleRunning,
  locale,
  t,
}: { title: TitleHistoryStatus } & HistoryStateProps) {
  const pass = title.last_pass
  const added = pass ? lastPassAddedLine(t, pass) : null
  return (
    <section aria-label={title.title_name} data-testid={`history-title-${title.title_slug}`}>
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{title.title_name}</h3>
      <dl className="grid grid-cols-[minmax(0,11rem)_minmax(0,1fr)] gap-x-4 gap-y-1.5 text-sm">
        <Row label={t.rowHistory} testId="history-state">
          {historyStateLabel(t, title, locale, cycleRunning, status.auto_sync_enabled)}
        </Row>
        <Row label={t.rowMatches} testId="history-matches">
          {title.match_count > 0 ? formatNumber(title.match_count, intlLocale(locale)) : t.noMatches}
        </Row>
        <Row label={t.rowOldest} testId="history-oldest">
          {title.oldest_match_at ? formatDate(title.oldest_match_at, intlLocale(locale)) : '—'}
        </Row>
        <Row label={t.rowLastPass} testId="history-last-pass">
          {pass ? (
            <>
              <span className={pass.status === 'failed' ? 'text-destructive' : undefined}>{lastPassLine(t, locale, pass)}</span>
              {added && <span className="block text-xs text-muted-foreground">{added}</span>}
            </>
          ) : (
            <span className="text-muted-foreground">{t.lastPassNone}</span>
          )}
        </Row>
        <Row label={t.rowNextPass} testId="history-next-pass">
          {nextPassText(t, locale, status, cycleRunning)}
        </Row>
      </dl>
    </section>
  )
}

export function HistoryState({ status, cycleRunning, locale, t }: HistoryStateProps) {
  return (
    <div className="space-y-5">
      {(status.titles ?? []).map((title) => (
        <TitleHistory key={title.title_slug} title={title} status={status} cycleRunning={cycleRunning} locale={locale} t={t} />
      ))}
    </div>
  )
}
