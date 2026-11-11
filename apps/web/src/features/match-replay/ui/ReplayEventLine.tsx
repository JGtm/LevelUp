/**
 * ReplayEventLine — la ligne d'une PRISE d'arme ou de bonus, ou d'une MONTÉE en véhicule : un
 * pictogramme, le joueur à l'encre de son camp, et une phrase. Même rangée que toutes les
 * lignes du fil (une seule ligne, sans retour, pictogramme `shrink-0`) ; contrairement à
 * l'objectif, elle n'est pas un bandeau : ce ne sont pas des tournants du match.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PickupEventLine, VehicleEventLine } from '../model/feedEventLines'
import { FEED_FILTER_GLYPH } from '../model/feedGlyphs'
import { FEED_ROW } from '../model/feedRow'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { FeedName } from './ReplayFeedName'
import { EventGlyph } from './ReplayEventGlyph'
import { FeedClock } from './ReplayFeedClock'

interface Props {
  line: PickupEventLine | VehicleEventLine
  replayMs: number
  /** Encre de camp du joueur (allégeance du film), neutre quand elle est inconnue. */
  color: string
  mark: PlayerMarkKind | undefined
  name: string
  locale: ReplayLocale
}

export function ReplayEventLine({ line, replayMs, color, mark, name, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const action = line.family === 'pickups' ? t.feedPickupAction(line.name) : t.feedVehicleAction(line.name)
  return (
    <li className={FEED_ROW} title={`${name} ${action}`} data-feed-family={line.family}>
      <FeedClock ms={replayMs} />
      <span className="shrink-0" style={{ color }}>
        <EventGlyph glyph={FEED_FILTER_GLYPH[line.family]} />
      </span>
      <FeedName kind={mark} color={color} locale={locale} className="font-medium" name={name} />
      <span className="min-w-0 truncate text-muted-foreground">{action}</span>
    </li>
  )
}
