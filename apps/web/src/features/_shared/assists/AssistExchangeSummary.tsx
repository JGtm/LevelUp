/**
 * AssistExchangeSummary — la figure « échange d'assistances » avec un joueur :
 * têtes (il t'a assisté / tu l'as assisté), barre papillon, puis « total · part » de
 * chaque côté.
 *
 * Deux surfaces la rendent à l'identique — la carte Binôme du hub Relations et le bloc
 * « Part des assistances » de l'encart cible Explorer — d'où l'extraction ici plutôt
 * qu'une seconde copie du même balisage (règle des 2 copies).
 *
 * Longueur des barres = VOLUME sur une échelle log bornée par `volumeMax`, commun à la
 * surface appelante (assistExchange.ts) ; les parts s'affichent en chiffres, jamais dans
 * la longueur.
 */
import { tokenCssVar } from '@/lib/accessibility'
import { formatPercent } from '@/lib/formatters'
import type { RelationAssists } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { ASSIST_GIVEN_TOKEN, ASSIST_RECEIVED_TOKEN, AssistButterflyBar } from './AssistButterflyBar'
import { givenShare, receivedShare } from './assistExchange'
import { ASSISTS_TEXT } from './assistsI18n'

export function AssistExchangeSummary({
  assists,
  volumeMax,
  locale,
  className,
  testId,
}: {
  assists: RelationAssists
  /** Borne de l'échelle log — plus gros volume d'un sens sur la surface appelante. */
  volumeMax: number
  locale: Locale
  className?: string
  testId?: string
}) {
  const text = ASSISTS_TEXT[locale]
  const received = receivedShare(assists)
  const given = givenShare(assists)
  const fmt = (n: number) => n.toLocaleString(locale)
  return (
    <div className={className} data-testid={testId}>
      <div className="mb-1 flex items-baseline justify-between text-xs text-muted-foreground">
        <span>{text.receivedHead}</span>
        <span>{text.givenHead}</span>
      </div>
      <AssistButterflyBar assists={assists} volumeMax={volumeMax} text={text} locale={locale} variant="card" />
      <div className="mt-1 flex items-baseline justify-between font-mono text-xs tabular-nums">
        <span style={{ color: tokenCssVar(ASSIST_RECEIVED_TOKEN) }}>
          {fmt(assists.received.total)} · {formatPercent(received, 0)}
        </span>
        <span style={{ color: tokenCssVar(ASSIST_GIVEN_TOKEN) }}>
          {formatPercent(given, 0)} · {fmt(assists.given.total)}
        </span>
      </div>
    </div>
  )
}
