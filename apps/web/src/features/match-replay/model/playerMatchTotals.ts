/**
 * playerMatchTotals.ts — LES TOTAUX DE FIN DE MATCH D'UN JOUEUR, lus sur sa ligne de feuille : précision,
 * dégâts infligés et subis, score, éliminations à la tête. Sans React.
 *
 * Ce sont les chiffres officiels du match : le film ne les publie pas en cours de partie, la fiche les écrit
 * à côté de ce qui est lu à l'instant, dans une section qui le dit. Une valeur que la feuille ne porte pas
 * n'a pas de ligne : un joueur sans ligne de feuille n'a pas de totaux (`null`), et un joueur dont la feuille
 * ne porte aucune de ces valeurs non plus.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Les lignes des totaux ; chacune est `null` quand la feuille ne porte pas ce qu'il faut pour l'écrire. */
export interface PlayerMatchTotals {
  /** La précision en pour-cent (0 à 100) et, quand la feuille les porte tous deux, les touches et les tirs. */
  accuracy: { pct: number; hit: number; fired: number } | { pct: number; hit: null; fired: null } | null
  /** Les dégâts infligés et subis : une ligne n'existe que si la feuille porte les deux. */
  damage: { dealt: number; taken: number } | null
  score: number | null
  headshotKills: number | null
}

/** Un nombre fini de la feuille, ou `null` (champ absent ou non numérique). */
function finite(value: number | null | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function accuracyOf(board: MatchScoreboardRow): PlayerMatchTotals['accuracy'] {
  const pct = finite(board.accuracy)
  if (pct === null) return null
  const hit = finite(board.shots_hit)
  const fired = finite(board.shots_fired)
  return hit !== null && fired !== null ? { pct, hit, fired } : { pct, hit: null, fired: null }
}

function damageOf(board: MatchScoreboardRow): PlayerMatchTotals['damage'] {
  const dealt = finite(board.damage_dealt)
  const taken = finite(board.damage_taken)
  return dealt !== null && taken !== null ? { dealt, taken } : null
}

/** playerMatchTotals lit les totaux sur la ligne de feuille ; `null` sans ligne ou sans aucune ligne à écrire. */
export function playerMatchTotals(board: MatchScoreboardRow | undefined): PlayerMatchTotals | null {
  if (!board) return null
  const totals: PlayerMatchTotals = {
    accuracy: accuracyOf(board),
    damage: damageOf(board),
    score: finite(board.score),
    headshotKills: finite(board.headshot_kills),
  }
  return Object.values(totals).some((v) => v !== null) ? totals : null
}
