/**
 * FeedClock — la GOUTTIÈRE D'INSTANT en tête de ligne : largeur fixe (`COTES_LIGNE_FIL`), chiffres
 * tabulaires, encre discrète. Toutes les formes de ligne l'ouvrent — c'est la colonne qui rend le fil
 * balayable d'un regard.
 */
import { formatClock } from '../../../lib/replay/replayLogic'
import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'

export function FeedClock({ ms }: { ms: number }) {
  return (
    <span
      data-feed-col="instant"
      className="shrink-0 tabular-nums text-muted-foreground"
      style={{ width: COTES_LIGNE_FIL.colonnes.instantPx, fontSize: COTES_LIGNE_FIL.tailleSecondairePx }}
    >
      {formatClock(ms)}
    </span>
  )
}
