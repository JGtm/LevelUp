/**
 * ReplayChatterLine — LA LIGNE D'UNE RÉPLIQUE DE SPARTAN dans le fil : le pictogramme de la réplique, le joueur qui parle à
 * l'encre de son camp, un haut-parleur quand la réplique a une voix dans la langue de l'interface, et ce qu'il dit entre
 * guillemets. Même rangée que toutes les lignes du fil (une seule ligne, sans retour, pictogrammes `shrink-0`) ; l'infobulle
 * donne la phrase entière.
 *
 * LE HAUT-PARLEUR NE PROMET PAS UN SON QUI N'EXISTE PAS : sans voix dans la langue, la ligne n'en porte pas.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ChatterEventLine } from '../model/feedEventLines'
import { FEED_FILTER_GLYPH } from '../model/feedGlyphs'
import { FEED_ROW, feedRowClass, followedAttr } from '../model/feedRow'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { FeedName } from './ReplayFeedName'
import { EventGlyph } from './ReplayEventGlyph'
import { FeedClock } from './ReplayFeedClock'

interface Props {
  line: ChatterEventLine
  replayMs: number
  /** Encre de camp du joueur (allégeance du film), neutre quand elle est inconnue. */
  color: string
  mark: PlayerMarkKind | undefined
  /** Le nom du joueur qui parle. */
  name: string
  locale: ReplayLocale
  /** La ligne concerne le joueur suivi par la caméra : fond léger (`FEED_FOLLOWED`). */
  followed?: boolean
}

/** Le haut-parleur : le même dessin que l'icône de voix de la carte (grille de 24). */
function SpeakerIcon({ label }: { label: string }) {
  return (
    <svg width={12} height={12} viewBox="0 0 24 24" className="shrink-0 text-muted-foreground" role="img" aria-label={label}>
      <title>{label}</title>
      <path d="M4 9v6h4l5 4V5L8 9H4z" fill="currentColor" />
      <path d="M16 9a4.5 4.5 0 0 1 0 6M18.5 6.5a8 8 0 0 1 0 11" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" />
    </svg>
  )
}

export function ReplayChatterLine({ line, replayMs, color, mark, name, locale, followed }: Props) {
  const t = REPLAY_TEXT[locale].chatter
  const quoted = t.quote(line.text)
  return (
    <li
      className={feedRowClass(FEED_ROW, followed)}
      title={`${name} ${quoted}`}
      aria-label={t.feedLineAriaFmt(name, t.family[line.chatterFamily], line.text)}
      data-feed-family={line.family}
      data-followed={followedAttr(followed)}
    >
      <FeedClock ms={replayMs} />
      <span className="shrink-0" style={{ color }}>
        <EventGlyph glyph={FEED_FILTER_GLYPH.replies} />
      </span>
      <FeedName kind={mark} color={color} locale={locale} className="font-medium" name={name} />
      {line.voice && <SpeakerIcon label={t.hasVoice} />}
      <span className="min-w-0 truncate text-muted-foreground">{quoted}</span>
    </li>
  )
}
