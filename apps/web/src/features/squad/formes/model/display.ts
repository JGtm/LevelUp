/**
 * display.ts — LE REPLI DES FORMES QUI ONT UNE LIGNE (ou une case) PAR MATCH.
 *
 * POURQUOI CE FICHIER EXISTE. L'artefact 2ec1b8eb l'annonçait dans ses propres
 * notes — « neuf lignes, c'est déjà long ; au-delà d'une vingtaine de matchs
 * elle demandera un repli » — et la mesure sur données réelles l'a confirmé :
 * sur une portée de 1 147 matchs, « Emprise de l'escouade » rendait 1 147
 * lignes, dont l'immense majorité en hachure « sans film décodé ». Quinze
 * écrans de hachure ne disent rien.
 *
 * DEUX RÈGLES, ET ELLES NE SE CONFONDENT PAS :
 *
 *   - une forme dont la matière VIENT DU FILM (gestes d'équipement, socles)
 *     ne montre QUE des matchs mesurés : un match sans film n'y est pas une
 *     ligne vide, il n'y est pas du tout, et le pied de la forme dit combien
 *     ont été écartés ;
 *   - une forme dont la matière vient de la FEUILLE DE MATCH (les valeurs
 *     brutes d'objectif) n'a pas besoin de film : elle garde tous ses matchs,
 *     et ne se borne qu'en nombre.
 *
 * ON GARDE LES PLUS RÉCENTS, ON LES AFFICHE DU PLUS ANCIEN AU PLUS RÉCENT : la
 * portée arrive du serveur du plus récent au plus ancien, et une frise se lit
 * dans le sens du temps (convention de l'artefact : « les matchs de la période,
 * dans l'ordre »).
 */
import type { SquadFormesBlock, SquadFormesMatch } from '@/lib/api/types'

import { allMatches } from './access'

/** Le nombre de matchs qu'une forme par match affiche au plus. */
export const MATCH_ROWS_LIMIT = 20

/** Ce qu'une forme affiche, et ce qu'elle a écarté. */
export interface MatchWindow {
  /** Les matchs retenus, du plus ancien au plus récent. */
  rows: SquadFormesMatch[]
  /** Matchs éligibles mais hors fenêtre (trop anciens). */
  hidden: number
  /** Matchs écartés faute de film décodé (0 pour une forme qui n'en exige pas). */
  unmeasured: number
}

/** Les `limit` derniers matchs d'une liste déjà ordonnée du plus récent au plus ancien. */
function lastOf(matches: SquadFormesMatch[], limit: number): { rows: SquadFormesMatch[]; hidden: number } {
  const kept = matches.slice(0, Math.max(0, limit))
  return { rows: [...kept].reverse(), hidden: Math.max(0, matches.length - kept.length) }
}

/**
 * La fenêtre des formes ALIMENTÉES PAR LE FILM : matchs mesurés seulement, les
 * plus récents, dans l'ordre du temps.
 */
export function measuredWindow(block: SquadFormesBlock, limit = MATCH_ROWS_LIMIT): MatchWindow {
  const all = allMatches(block)
  const measured = all.filter((m) => m.measured)
  const { rows, hidden } = lastOf(measured, limit)
  return { rows, hidden, unmeasured: all.length - measured.length }
}

/**
 * La fenêtre d'une liste déjà filtrée (les matchs d'une famille de mode) : on ne
 * borne que le nombre, aucun match n'est écarté pour absence de film.
 */
export function listWindow(matches: SquadFormesMatch[], limit = MATCH_ROWS_LIMIT): MatchWindow {
  const { rows, hidden } = lastOf(matches, limit)
  return { rows, hidden, unmeasured: 0 }
}
