/**
 * MatchCardAssistedFrags — ligne « N / M frags assistés · part » de la tuile de match,
 * sous la barre frags / assistances / décès, avec la barre à trois tons du sens
 * « reçues » (un coéquipier t'a assisté), part de dégâts par segment en infobulle.
 *
 * Rendu UNIQUEMENT quand `assisted_frags` est présent (match mesuré : film analysé).
 * Absent → rien du tout, pas de « — » : la tuile n'a pas de place pour l'incertitude,
 * et un « 0 » fabriqué se lirait comme une mesure. Même part que la page Relations
 * (`assistShare`), même formatage (`formatPercent(…, 0)`).
 */
import { AssistTierBar, ASSIST_RECEIVED_TOKEN } from '@/features/_shared/assists/AssistTierBar'
import { assistShare, assistShareSegments } from '@/features/_shared/assists/assistExchange'
import { ASSISTS_TEXT } from '@/features/_shared/assists/assistsI18n'
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchAssistedFrags } from '@/lib/api/types'
import { formatPercent } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

export function MatchCardAssistedFrags({ assisted, locale }: { assisted: MatchAssistedFrags; locale: Locale }) {
  const frags = assisted.frags_measured
  const received = assisted.received.total
  const share = assistShare(received, frags)
  const fmt = (n: number) => n.toLocaleString(locale)
  return (
    <div data-testid="match-card-assisted-frags" className="space-y-1">
      <div
        className="text-center font-mono text-2xs tabular-nums leading-none"
        style={{ color: tokenCssVar(ASSIST_RECEIVED_TOKEN) }}
      >
        {formatMessage(commonManifest, 'common.match_card.assisted_frags', locale, {
          assisted: fmt(received),
          frags: fmt(frags),
          share: formatPercent(share, 0),
        })}
      </div>
      <AssistTierBar
        segments={assistShareSegments(assisted.received, frags)}
        color={tokenCssVar(ASSIST_RECEIVED_TOKEN)}
        text={ASSISTS_TEXT[locale]}
        locale={locale}
        variant="tile"
        testId="match-card-assist-segment"
      />
    </div>
  )
}
