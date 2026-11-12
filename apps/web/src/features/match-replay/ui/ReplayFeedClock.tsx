/**
 * FeedClock — la GOUTTIÈRE D'HORLOGE en tête de ligne : largeur fixe, chiffres tabulaires, encre
 * discrète. Toutes les formes de ligne l'ouvrent — c'est la colonne qui rend le fil balayable
 * d'un regard.
 */
import { formatClock } from '../../../lib/replay/replayLogic'
import { FEED_CLOCK_W } from '../model/feedRow'

export function FeedClock({ ms }: { ms: number }) {
  return (
    <span
      className="shrink-0 font-mono text-[10.5px] tabular-nums text-muted-foreground"
      style={{ width: FEED_CLOCK_W }}
    >
      {formatClock(ms)}
    </span>
  )
}
