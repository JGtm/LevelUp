/**
 * ReplayPresenceLine — la ligne d'ENTRÉE/SORTIE du fil (cf. presenceFeed.ts, 2026-09-02).
 *
 * Une flèche qui FRANCHIT UNE PORTE, dans le sens de l'événement : vers l'intérieur pour
 * une entrée en partie, vers l'extérieur pour un joueur qui ne reviendra plus. Le dessin
 * lui-même vit dans `PresenceGlyph.tsx` depuis le 2026-09-07 (lot L4) : la FRISE le pose
 * elle aussi, à la frontière de sa zone ombrée, et un second exemplaire du SVG aurait fini
 * par diverger de celui-ci. Cette ligne n'en garde que l'usage — encre de l'équipe quand
 * elle est connue, encre du repli sinon, jamais un camp deviné.
 *
 * LE LIBELLÉ RESTE AU FAIT : « ne reviendra plus » plutôt que « a quitté » — la dernière
 * vie d'un éliminé définitif (mode à manches) s'arrête exactement comme celle d'un
 * partant, et le film ne les distingue pas. L'infobulle porte cette réserve.
 */
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PresenceEvent } from '../model/presenceFeed'
import { presenceWording } from '../model/presenceWording'
import { FeedClock, FEED_ROW } from './ReplayKillFeed'
import { FeedName } from './ReplayFeedName'
import { PresenceGlyph } from './PresenceGlyph'

export function ReplayPresenceLine({
  presence,
  replayMs,
  color,
  mark,
  locale,
}: {
  presence: PresenceEvent
  replayMs: number
  color: string
  mark: PlayerMarkKind | undefined
  locale: ReplayLocale
}) {
  // Deux sources, deux vocabulaires : l'API affirme (« a rejoint / a quitté »), le repli
  // film reste au fait (« entre en partie / ne reviendra plus ») — cf. presenceFeed.ts. La
  // règle vit dans `model/presenceWording.ts`, PARTAGÉE avec la porte de la frise
  // (`ReplayPresenceShade`) : un seul foyer, sans quoi l'une affirmerait ce que l'autre
  // met au conditionnel.
  const { label, hint } = presenceWording(presence, REPLAY_TEXT[locale])
  return (
    <li className={FEED_ROW} title={hint}>
      <FeedClock ms={replayMs} />
      <PresenceGlyph kind={presence.kind} color={color} />
      <FeedName kind={mark} color={color} locale={locale} className="font-medium" name={presence.name} />
      <span className="min-w-0 truncate text-muted-foreground">{label}</span>
    </li>
  )
}
