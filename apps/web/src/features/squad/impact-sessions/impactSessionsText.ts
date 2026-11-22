/**
 * impactSessionsText.ts — les textes du tableau « Impact des coéquipiers » par session (Escouade ›
 * Contributions). Le titre de section, les noms et descriptions des rôles et les libellés des marques du
 * score (couronne, banane, passager clandestin) restent dans le dictionnaire de la feature (`t.impact`).
 *
 * Parité des langues garantie par le typage `Record<Locale, …>`.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ImpactSessionsText {
  /** Aide ⓘ du titre de section. */
  info: string
  colSession: string
  colBilan: string
  colPlayer: string
  colScore: string
  /** Libellé de la première ligne, qui somme les sessions affichées. */
  allSessions: string
  allSessionsSub: (matches: number) => string
  sessionSub: (matches: number) => string
  /** Infobulle d'un numéro de match : « Match 3 : victoire ». */
  matchHeader: (number: number, outcome: 'win' | 'loss' | 'other') => string
  /** Cellule d'un match où le joueur n'a aucun rôle. */
  noRole: string
  /** Infobulle du net par match d'un joueur. */
  netTooltip: (net: string, points: string, matches: number) => string
  /** Infobulle d'un joueur qui n'a joué aucun match de la ligne. */
  notPlayed: string
  tableLabel: string
  matrixLabel: (label: string) => string
  totalsLabel: string
}

const FR: ImpactSessionsText = {
  info:
    'Une ligne par session : le bilan, puis le net par match de chaque joueur (points d’impact de la session divisés ' +
    'par les matchs joués). Une session dépliée montre sa matrice : un match par colonne, les rôles de chaque joueur ' +
    'en icônes, le nombre de fois où il a eu chaque rôle, puis le score. Le meilleur est en vert, le pire en rouge. ' +
    'La première ligne donne le même décompte sur toutes les sessions affichées.',
  colSession: 'Session',
  colBilan: 'Bilan',
  colPlayer: 'Joueur',
  colScore: 'Score',
  allSessions: 'Toutes les sessions',
  allSessionsSub: (n) => `${n} match${n > 1 ? 's' : ''} · totaux par rôle`,
  sessionSub: (n) => `${n} match${n > 1 ? 's' : ''}`,
  matchHeader: (n, outcome) =>
    `Match ${n} : ${outcome === 'win' ? 'victoire' : outcome === 'loss' ? 'défaite' : 'sans victoire ni défaite'}`,
  noRole: 'Aucun rôle',
  netTooltip: (net, points, matches) => `Net par match ${net} (${points} points sur ${matches} match${matches > 1 ? 's' : ''})`,
  notPlayed: 'Aucun match joué sur cette session',
  tableLabel: 'Impact des coéquipiers par session',
  matrixLabel: (label) => `Rôles d’impact de la session ${label}`,
  totalsLabel: 'Rôles d’impact sur toutes les sessions affichées',
}

const EN: ImpactSessionsText = {
  info:
    'One row per session: the record, then each player’s net per match (the session impact points divided by the ' +
    'matches played). An opened session shows its matrix: one match per column, each player’s roles as icons, the ' +
    'number of times each role was earned, then the score. The best is green, the worst red. ' +
    'The first row gives the same tally over all the sessions shown.',
  colSession: 'Session',
  colBilan: 'Record',
  colPlayer: 'Player',
  colScore: 'Score',
  allSessions: 'All sessions',
  allSessionsSub: (n) => `${n} match${n > 1 ? 'es' : ''} · totals per role`,
  sessionSub: (n) => `${n} match${n > 1 ? 'es' : ''}`,
  matchHeader: (n, outcome) => `Match ${n}: ${outcome === 'win' ? 'win' : outcome === 'loss' ? 'loss' : 'no win or loss'}`,
  noRole: 'No role',
  netTooltip: (net, points, matches) => `Net per match ${net} (${points} points over ${matches} match${matches > 1 ? 'es' : ''})`,
  notPlayed: 'No match played in this session',
  tableLabel: 'Teammates impact per session',
  matrixLabel: (label) => `Impact roles of the session ${label}`,
  totalsLabel: 'Impact roles over all the sessions shown',
}

export const IMPACT_SESSIONS_TEXT: Record<Locale, ImpactSessionsText> = { fr: FR, en: EN }
