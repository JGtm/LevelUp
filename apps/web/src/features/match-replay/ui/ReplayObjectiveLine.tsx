/**
 * ReplayObjectiveLine — LA LIGNE D'OBJECTIF DU FIL, en BANDEAU : un fond teinté à la couleur de
 * camp du joueur, le nom en gras. Les objectifs sont les tournants du match : l'œil les trouve sans
 * lire (maquette « bandeaux »). Sa hauteur est celle de toute ligne du fil, sans liseré.
 *
 * `ObjectiveBanner` est la COQUE commune à toutes les lignes d'objectif (action nommée, manche,
 * crâne, colline) ; `ReplayObjectiveLine` est la ligne d'une action nommée du joueur. Les autres
 * formes vivent dans `ReplayObjectiveStateLine`.
 *
 * LES RÈGLES DU FIL TIENNENT ICI COMME AILLEURS : une seule ligne, sans retour (la rangée
 * `FEED_ROW` est `flex-nowrap overflow-hidden`), pictogramme `shrink-0`, le libellé tronque
 * avant le nom — et son infobulle le rend en entier. L'encre est celle de l'allégeance du film ;
 * un joueur dont le film tait l'équipe prend l'encre neutre, jamais un camp deviné.
 *
 * UNE CAPTURE PORTE SES ASSISTANCES : les joueurs crédités ferment la rangée, derrière la marque
 * d'assistance du fil de frags (`AssistMark`, la même marque et la même encre).
 */
import type { ReactNode } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { OBJECTIVE_FAMILY_GLYPH, type FeedGlyphKey } from '../model/feedGlyphs'
import { FEED_ROW } from '../model/feedRow'
import { objectiveFamilyOfFeedStat } from '../model/objectiveFeedLogic'
import type { ObjectiveEventLine } from '../model/feedEventLines'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { AssistMark } from './ReplayAssistMark'
import { FeedName } from './ReplayFeedName'
import { EventGlyph } from './ReplayEventGlyph'
import { FeedClock } from './ReplayFeedClock'

/** Part de la couleur de camp dans le fond du bandeau : assez pour teinter, jamais pour crier. */
const BANNER_TINT_PCT = 15

/**
 * ObjectiveBanner — la coque d'une ligne d'objectif : horloge, pictogramme à l'encre `color`, puis
 * le contenu (`children`) ; `title` est l'infobulle, qui rend la ligne en entier.
 */
export function ObjectiveBanner({
  glyph,
  color,
  replayMs,
  title,
  children,
}: {
  glyph: FeedGlyphKey
  color: string
  replayMs: number
  title: string
  children: ReactNode
}) {
  return (
    <li
      className={FEED_ROW}
      title={title}
      data-feed-family="objectives"
      style={{
        background: `color-mix(in srgb, ${color} ${BANNER_TINT_PCT}%, transparent)`,
      }}
    >
      <FeedClock ms={replayMs} />
      <span className="shrink-0" style={{ color }}>
        <EventGlyph glyph={glyph} />
      </span>
      {children}
    </li>
  )
}

/** Un joueur crédité d'une assistance de capture : son nom et sa marque d'identité. */
export interface CaptureAssister {
  key: string
  name: string
  mark: PlayerMarkKind | undefined
}

interface Props {
  line: ObjectiveEventLine
  replayMs: number
  /** Encre de camp du joueur (allégeance du film), neutre quand elle est inconnue. */
  color: string
  mark: PlayerMarkKind | undefined
  name: string
  /** Les joueurs crédités d'une assistance sur cette capture ; absent ou vide : aucun. */
  assisters?: readonly CaptureAssister[]
  locale: ReplayLocale
}

export function ReplayObjectiveLine({ line, replayMs, color, mark, name, assisters = [], locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const action = t.feedObjectiveAction[line.stat]
  const credited = assisters.map((a) => a.name).join(', ')
  return (
    <ObjectiveBanner
      glyph={OBJECTIVE_FAMILY_GLYPH[objectiveFamilyOfFeedStat(line.stat)]}
      color={color}
      replayMs={replayMs}
      title={credited ? `${name} ${action} — ${t.feedLines.captureAssistedBy(credited)}` : `${name} ${action}`}
    >
      <FeedName kind={mark} color={color} locale={locale} className="font-extrabold" name={name} />
      <span className="min-w-0 truncate text-foreground">{action}</span>
      {assisters.length > 0 && (
        <span className="flex min-w-0 items-center gap-1 text-[10.5px] text-muted-foreground">
          <AssistMark label={t.killFeedAssistMark} />
          {assisters.map((a, i) => (
            <span key={a.key} className="flex min-w-0 items-center">
              <FeedName name={a.name} locale={locale} kind={a.mark} />
              {i < assisters.length - 1 && <span aria-hidden>,</span>}
            </span>
          ))}
        </span>
      )}
    </ObjectiveBanner>
  )
}
