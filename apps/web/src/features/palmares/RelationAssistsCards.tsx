/**
 * RelationAssistsCards — les assistances échangées dans les cartes du haut de la page
 * Relations (maquette validée le 2026-09-16) :
 *
 *  - BinomeAssistsBlock : carte Binôme, option B — barre papillon (il t'a assisté à
 *    gauche, tu l'as assisté à droite) sous deux têtes, puis « total · part » de chaque
 *    côté. Aucun texte de verdict ni de couverture. La figure elle-même vit dans
 *    `_shared/assists/AssistExchangeSummary` (l'encart cible Explorer la rend aussi) ;
 *    ici on ne pose que l'habillage propre à la carte (filet de séparation).
 *  - CoreRankingList : carte Noyau dur, option A8 — une rangée par fidèle : rang, nom,
 *    papillon, matchs, taux de victoire ; légende « ◀ te sert · tu le sers ▶ » alignée
 *    sous la colonne des barres. Classement inchangé (taux de victoire), 3 premiers puis
 *    « voir les autres ».
 *
 * Une grille unique porte toutes les rangées : les colonnes restent alignées d'une ligne
 * à l'autre (le nom est tronqué au-delà de sa largeur).
 */
import { Fragment } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import {
  ASSIST_GIVEN_TOKEN,
  ASSIST_RECEIVED_TOKEN,
  AssistButterflyBar,
} from '@/features/_shared/assists/AssistButterflyBar'
import { AssistExchangeSummary } from '@/features/_shared/assists/AssistExchangeSummary'
import { ASSISTS_TEXT } from '@/features/_shared/assists/assistsI18n'
import { tokenCssVar } from '@/lib/accessibility'
import { winRateColor } from '@/lib/colors/outcomePalette'
import { formatPercent } from '@/lib/formatters'
import type { RelationAssists, RelationInsight } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { PalmaresText } from './i18n'

type RelationsText = PalmaresText['relations']

export function BinomeAssistsBlock({
  assists,
  volumeMax,
  locale,
}: {
  assists: RelationAssists
  volumeMax: number
  locale: Locale
}) {
  return (
    <AssistExchangeSummary
      assists={assists}
      volumeMax={volumeMax}
      locale={locale}
      className="mt-3 border-t border-border pt-3"
      testId="binome-assists"
    />
  )
}

export function CoreRankingList({
  rows,
  volumeMax,
  labels,
  locale,
  onPlayerClick,
}: {
  rows: RelationInsight[]
  /** Borne de l'échelle log, commune à toute la page (binôme et noyau comparables). */
  volumeMax: number
  labels: RelationsText
  locale: Locale
  onPlayerClick: (gamertag: string) => void
}) {
  const text = ASSISTS_TEXT[locale]
  const anyMeasured = rows.some((r) => r.assists)
  return (
    <div
      className="grid items-center gap-x-2 gap-y-1 text-sm"
      style={{ gridTemplateColumns: 'auto minmax(0, 6rem) minmax(0, 1fr) auto auto' }}
      data-testid="core-ranking"
    >
      {rows.map((r, i) => (
        <Fragment key={r.xuid}>
          <span className="text-right font-mono text-xs text-muted-foreground tabular-nums">{i + 1}</span>
          <button
            type="button"
            className="block max-w-full truncate text-left font-semibold text-foreground hover:underline"
            onClick={() => onPlayerClick(r.gamertag)}
          >
            {r.gamertag}
          </button>
          {r.assists ? (
            <AssistButterflyBar assists={r.assists} volumeMax={volumeMax} text={text} locale={locale} variant="row" />
          ) : (
            <Tooltip className="justify-center" content={text.notMeasured}>
              <span className="w-full cursor-help text-center font-mono text-xs text-muted-foreground">—</span>
            </Tooltip>
          )}
          <Tooltip content={labels.hero.matchesPlayed(r.total_matches.toLocaleString(locale))}>
            <span className="cursor-help text-right font-mono text-xs text-muted-foreground tabular-nums">
              {r.total_matches.toLocaleString(locale)}
            </span>
          </Tooltip>
          <span
            className="text-right font-mono text-xs font-bold tabular-nums"
            style={{ color: winRateColor(r.teammate_win_rate) }}
          >
            {formatPercent(r.teammate_win_rate, 0)}
          </span>
        </Fragment>
      ))}
      {anyMeasured && (
        <>
          <span />
          <span />
          <span className="flex items-baseline justify-between text-2xs leading-tight">
            <span style={{ color: tokenCssVar(ASSIST_RECEIVED_TOKEN) }}>{text.legendReceived}</span>
            <span style={{ color: tokenCssVar(ASSIST_GIVEN_TOKEN) }}>{text.legendGiven}</span>
          </span>
          <span />
          <span />
        </>
      )}
    </div>
  )
}
