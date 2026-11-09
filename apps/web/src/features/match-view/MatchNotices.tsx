/**
 * MatchNotices — LA zone de messages de la vue match, juste sous l'en-tête, avant les onglets.
 *
 * Tous les messages de page y vivent (lecture en échec, données absentes, analyse du film,
 * confidentialité) : la sélection et l'ordre sont dans `matchNotices.logic.ts`. Les mentions
 * « se met à jour seule » ne sont posées que là où la vue se recharge réellement seule
 * (`matchViewPollInterval`).
 */
import { Notice, NoticeLiveStatus } from '@/components/ui/notice'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import type { Locale } from '@/lib/i18n/locale'
import type { MatchViewText } from './i18n'
import {
  analysisNoticeBody,
  analysisNoticeTitle,
  type MatchNoticesInput,
  selectMatchNotices,
  translatePartialReason,
  type MatchNotice,
} from './matchNotices.logic'

type Props = {
  data: MatchNoticesInput
  t: MatchViewText
  locale: Locale
}

function NoticeItem({ notice, t, locale }: { notice: MatchNotice; t: MatchViewText; locale: Locale }) {
  switch (notice.kind) {
    case 'unavailable':
      return (
        <Notice
          tone="error"
          testId="match-notice-unavailable"
          title={t.noticeUnavailable}
          aside={<NoticeLiveStatus tone="error">{t.noticeUnavailableRetry}</NoticeLiveStatus>}
        />
      )
    case 'partial':
      return (
        <Notice tone="warning" testId="match-notice-partial" title={t.noticePartialLead}>
          <ul className="space-y-1">
            {notice.reasons.map((reason) => (
              <li key={reason} className="flex gap-2">
                <span className="mt-px shrink-0 text-warning" aria-hidden="true">▸</span>
                <span>{translatePartialReason(reason, locale)}</span>
              </li>
            ))}
          </ul>
        </Notice>
      )
    case 'analysis':
      return (
        <Notice
          tone="info"
          testId="match-notice-analysis"
          title={analysisNoticeTitle(notice.inProgress, t)}
          aside={<NoticeLiveStatus tone="info">{t.analysisPendingAutoUpdate}</NoticeLiveStatus>}
        >
          {notice.parts.length > 0 ? analysisNoticeBody(notice.parts, t) : null}
        </Notice>
      )
    case 'privacy':
      return <PrivacyBanner warning={notice.warning} />
  }
}

export function MatchNotices({ data, t, locale }: Props) {
  const notices = selectMatchNotices(data)
  if (notices.length === 0) return null
  return (
    <div className="space-y-3 px-6 pt-4" data-testid="match-notices">
      {notices.map((notice) => (
        <NoticeItem key={notice.kind} notice={notice} t={t} locale={locale} />
      ))}
    </div>
  )
}
