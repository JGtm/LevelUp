/**
 * AssistExchangeCell — cellule « Assistances » des tableaux (page Relations, historique
 * des rencontres de la vue match). Même langage que les colonnes Rencontres et Frags /
 * morts : compte gauche, barre proportionnelle, compte droit.
 *
 * Gauche = assistances DONNÉES (tu l'as assisté), droite = REÇUES (il t'a assisté).
 * Absent (aucun match ensemble dont le film porte l'assistance) → « — », jamais 0, sans
 * infobulle d'inconnu.
 */
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { RelationAssists } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { ASSIST_GIVEN_TOKEN, ASSIST_RECEIVED_TOKEN } from './AssistTierBar'
import { ASSISTS_TEXT } from './assistsI18n'

export function AssistExchangeCell({
  assists,
  locale,
}: {
  assists: RelationAssists | null | undefined
  locale: Locale
}) {
  if (!assists) return <span className="font-mono">—</span>
  const text = ASSISTS_TEXT[locale]
  const given = assists.given.total
  const received = assists.received.total
  const total = given + received
  const givenColor = tokenCssVar(ASSIST_GIVEN_TOKEN)
  const receivedColor = tokenCssVar(ASSIST_RECEIVED_TOKEN)
  const givenPct = total > 0 ? Math.round((given / total) * 100) : 50
  const fmt = (n: number) => n.toLocaleString(locale)
  return (
    <Tooltip
      content={
        <span className="flex flex-col">
          <span>{text.cellGiven(fmt(given))}</span>
          <span>{text.cellReceived(fmt(received))}</span>
        </span>
      }
    >
      <span className="inline-flex cursor-help items-center gap-1 font-mono tabular-nums" data-testid="assist-cell">
        <span style={{ color: givenColor }}>{fmt(given)}</span>
        <span className="inline-flex h-2 w-12 overflow-hidden border border-border">
          {total > 0 && (
            <>
              <span style={{ width: `${givenPct}%`, backgroundColor: givenColor }} />
              <span style={{ flex: 1, backgroundColor: receivedColor }} />
            </>
          )}
        </span>
        <span style={{ color: receivedColor }}>{fmt(received)}</span>
      </span>
    </Tooltip>
  )
}
