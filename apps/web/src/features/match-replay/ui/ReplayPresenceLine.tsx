/**
 * ReplayPresenceLine — la ligne d'ENTRÉE/SORTIE du fil (cf. presenceFeed.ts, 2026-09-02).
 *
 * Une flèche qui FRANCHIT UNE PORTE, dans le sens de l'événement : vers l'intérieur pour
 * une entrée en partie, vers l'extérieur pour un joueur qui ne reviendra plus. Le glyphe
 * est VECTORIEL et à l'encre courante (même arbitrage que le crâne et la bombe : aucune
 * vignette d'atlas dont l'index bouge par saison), teinté par l'équipe de l'acteur quand
 * elle est connue — un bot sans camp joint garde l'encre du repli, jamais un camp deviné.
 *
 * LE LIBELLÉ RESTE AU FAIT : « ne reviendra plus » plutôt que « a quitté » — la dernière
 * vie d'un éliminé définitif (mode à manches) s'arrête exactement comme celle d'un
 * partant, et le film ne les distingue pas. L'infobulle porte cette réserve.
 */
import type { PlayerMarkKind } from '../model/playerMarks'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PresenceEvent } from '../model/presenceFeed'
import { FeedClock, FEED_ROW } from './ReplayKillFeed'
import { FeedName } from './ReplayFeedName'

/** La porte (montant vertical) et la flèche qui la franchit — sens donné par `kind`. */
function PresenceGlyph({ kind, color }: { kind: PresenceEvent['kind']; color: string }) {
  const entering = kind === 'joined'
  return (
    <svg
      viewBox="0 0 14 12"
      width={14}
      height={12}
      aria-hidden
      className="shrink-0"
      style={{ color }}
    >
      {/* Le montant de la porte : côté gauche pour entrer, côté droit pour sortir. */}
      <path
        d={entering ? 'M1.5 1v10' : 'M12.5 1v10'}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
      <path
        d={entering ? 'M4 6h7M8.4 3.4 11 6l-2.6 2.6' : 'M3 6h7M7.4 3.4 10 6l-2.6 2.6'}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  )
}

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
  const t = REPLAY_TEXT[locale]
  const joined = presence.kind === 'joined'
  // Deux sources, deux vocabulaires : l'API affirme (« a rejoint / a quitté »), le repli
  // film reste au fait (« entre en partie / ne reviendra plus ») — cf. presenceFeed.ts.
  const api = presence.source === 'api'
  const label = joined
    ? (api ? t.presenceJoined : t.presenceJoinedDerived)
    : (api ? t.presenceLeft : t.presenceLeftDerived)
  const hint = joined
    ? (api ? t.presenceJoinedHint : t.presenceJoinedDerivedHint)
    : (api ? t.presenceLeftHint : t.presenceLeftDerivedHint)
  return (
    <li className={FEED_ROW} title={hint}>
      <FeedClock ms={replayMs} />
      <PresenceGlyph kind={presence.kind} color={color} />
      <FeedName kind={mark} color={color} locale={locale} className="font-medium" name={presence.name} />
      <span className="min-w-0 truncate text-muted-foreground">{label}</span>
    </li>
  )
}
