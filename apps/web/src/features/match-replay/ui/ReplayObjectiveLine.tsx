/**
 * ReplayObjectiveLine — LA LIGNE D'OBJECTIF DU FIL, en BANDEAU : un fond teinté à la couleur de
 * camp du joueur, un liseré à gauche, le nom en gras. Les objectifs sont les tournants du match :
 * l'œil les trouve sans lire (maquette « bandeaux »).
 *
 * LES RÈGLES DU FIL TIENNENT ICI COMME AILLEURS : une seule ligne, sans retour (la rangée
 * `FEED_ROW` est `flex-nowrap overflow-hidden`), pictogramme `shrink-0`, le libellé tronque
 * avant le nom — et son infobulle le rend en entier. L'encre est celle de l'allégeance du film ;
 * un joueur dont le film tait l'équipe prend l'encre neutre, jamais un camp deviné.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { OBJECTIVE_FAMILY_GLYPH } from '../model/feedGlyphs'
import { FEED_ROW } from '../model/feedRow'
import { objectiveFamilyOfFeedStat } from '../model/objectiveFeedLogic'
import type { ObjectiveEventLine } from '../model/feedEventLines'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { FeedName } from './ReplayFeedName'
import { EventGlyph } from './ReplayEventGlyph'
import { FeedClock } from './ReplayFeedClock'

/** Part de la couleur de camp dans le fond du bandeau : assez pour teinter, jamais pour crier. */
const BANNER_TINT_PCT = 15
/** Épaisseur du liseré de camp, à gauche. */
const BANNER_EDGE_PX = 3
/** Marge haute et basse du bandeau : plus haut qu'une ligne ordinaire, il se repère. */
const BANNER_PAD_PX = 9

interface Props {
  line: ObjectiveEventLine
  replayMs: number
  /** Encre de camp du joueur (allégeance du film), neutre quand elle est inconnue. */
  color: string
  mark: PlayerMarkKind | undefined
  name: string
  locale: ReplayLocale
}

export function ReplayObjectiveLine({ line, replayMs, color, mark, name, locale }: Props) {
  const action = REPLAY_TEXT[locale].feedObjectiveAction[line.stat]
  return (
    <li
      className={FEED_ROW}
      title={`${name} ${action}`}
      data-feed-family="objectives"
      style={{
        background: `color-mix(in srgb, ${color} ${BANNER_TINT_PCT}%, transparent)`,
        boxShadow: `inset ${BANNER_EDGE_PX}px 0 ${color}`,
        paddingTop: BANNER_PAD_PX,
        paddingBottom: BANNER_PAD_PX,
      }}
    >
      <FeedClock ms={replayMs} />
      <span className="shrink-0" style={{ color }}>
        <EventGlyph glyph={OBJECTIVE_FAMILY_GLYPH[objectiveFamilyOfFeedStat(line.stat)]} />
      </span>
      <FeedName kind={mark} color={color} locale={locale} className="font-extrabold" name={name} />
      <span className="min-w-0 truncate text-foreground">
        {action}
      </span>
    </li>
  )
}
