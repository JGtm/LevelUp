/**
 * filmAllegiance.ts — ALLIÉ OU ADVERSE, LU DANS LE FILM : le seul calcul d'allégeance du rejeu.
 *
 * # LA RÈGLE (décision du 2026-10-06, accordée par l'utilisateur)
 *
 * allié(p) = (équipe du film de p) === (équipe du film du joueur de RÉFÉRENCE). L'équipe d'un
 * joueur est le désignateur que le film écrit à son entrée de roster (`ReplayPlayer.team`, ADR 0034
 * D-9) ; la feuille de match NOMME les camps (`replayCamps.ts`) et ne décide pas de l'encre.
 *
 * Jusque-là, l'encre venait de la feuille, par la table d'identité `xuidMeta` — dont les clés sont
 * les xuid de la BASE. Un bot, que le film désigne par sa clé `bot:<nom>`, n'y était jamais
 * trouvé : il prenait l'encre adverse jusque dans le camp du joueur regardé. Toutes les surfaces du
 * rejeu (pions, colonnes, fil, frise, bandeau, écran de fin, objectifs, sons, tables de l'onglet
 * Arsenal) lisent désormais ce module, et `xuidMeta` ne sert plus que l'identité (les noms).
 *
 * # LA RÉFÉRENCE EST DONNÉE, JAMAIS DÉCOUVERTE ICI
 *
 * `buildFilmAllegiance` reçoit le xuid de BASE du joueur de référence : sur la page Rejeu, le point
 * de vue (`model.viewpoint`, le joueur de la page à défaut) ; pour la fin de partie sonore et les
 * tables de l'onglet Arsenal, le joueur de la page. Il le retrouve dans le film par sa ligne de
 * feuille (`board.xuid`, la jointure de `buildPlayers`) ou par sa clé.
 *
 * # TROIS RÉPONSES : allié (`true`), adverse (`false`), SANS ENCRE DE CAMP (`null`)
 *
 *  - UN JOUEUR DONT LE FILM TAIT L'ÉQUIPE n'a pas d'encre de camp : `null`, chaque surface sert son
 *    encre neutre. C'est un défaut de SOURCE, compté par la cuisson (`coverage.seats.sansEquipe`)
 *    et corrigé côté Go ; la feuille ne le comble pas.
 *  - UNE RÉFÉRENCE SANS ÉQUIPE DU FILM (ou absente du film) ne situe personne : `null` pour TOUS,
 *    elle comprise — c'est un joueur sans équipe comme un autre. Les surfaces dont le camp est la
 *    structure (bandeau de score, écran de fin, sons à deux variantes) se taisent, comme elles se
 *    taisaient sans ligne « moi ». Jamais une encre devinée.
 *  - MODE SANS CAMPS (désignateur `-1` de la référence, « aucune équipe ») : la référence est
 *    alliée d'elle-même et tout autre joueur est un adversaire — la mêlée générale, telle que la
 *    page la peignait déjà. Un CAMP `-1` n'en est pas un : `ofTeam(-1)` rend `null`, de même qu'un
 *    joueur `-1` dans un mode à camps.
 *
 * Tout est PUR : ni React, ni DOM, ni couleur.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ReplayCamp } from './replayCamps'
import type { ReplayDocumentReady } from './replayNormalize'
import { buildPlayers, groupByTeam, type ReplayPlayer } from './rosterLogic'

/** Allié (`true`), adverse (`false`), ou sans encre de camp (`null`) — cf. l'en-tête. */
export type Allegiance = boolean | null

/** L'allégeance de chacun, vue du joueur de référence. */
export interface FilmAllegiance {
  /** L'équipe du film de la référence ; `null` quand elle est absente du film ou que le film la tait. */
  readonly referenceTeam: number | null
  /**
   * Les camps du match (désignateurs >= 0, croissants), nommés par la feuille de leurs membres
   * (`ReplayCamp.side`). Le bandeau de score et l'écran de fin en exigent exactement deux.
   */
  readonly camps: readonly ReplayCamp[]
  /** L'allégeance d'un CAMP du film (une colonne, un camp de table, le tenant d'une zone, un drapeau). */
  ofTeam: (team: number | null | undefined) => Allegiance
  /** L'allégeance d'un JOUEUR du film (le propriétaire d'une vie). */
  ofPlayer: (player: ReplayPlayer | null | undefined) => Allegiance
  /**
   * L'allégeance d'un joueur désigné par un identifiant : sa clé du film (`ReplayPlayer.xuid`,
   * `bot:<nom>` pour un bot) OU un xuid de BASE (fil, feuille) — un bot s'y relie par sa ligne de
   * feuille (`board.xuid`, `bid(N.0)`), la jointure de `buildPlayers`. Inconnu → `null`.
   */
  ofXuid: (xuid: string | null | undefined) => Allegiance
  /** L'équipe du film d'un joueur désigné comme pour `ofXuid` (`-1` compris) ; `null` si le film la tait. */
  teamOfXuid: (xuid: string | null | undefined) => number | null
  /**
   * Les clés du FILM (`ReplayPlayer.xuid`) des joueurs d'un camp — celles des relectures de
   * position. Vide pour un désignateur négatif : « aucune équipe » n'a pas de membres.
   */
  membersOf: (team: number) => readonly string[]
}

const AUCUN_MEMBRE: readonly string[] = Object.freeze([])

/**
 * buildFilmAllegiance — l'allégeance de chaque joueur du film vue de `reference` (xuid de BASE,
 * ou clé du film). `players` : la jointure de `buildPlayers`, dont seules l'équipe, la clé et la
 * ligne de feuille sont lues.
 */
export function buildFilmAllegiance(
  players: readonly ReplayPlayer[],
  reference: string | null | undefined,
): FilmAllegiance {
  const parCle = new Map<string, ReplayPlayer>()
  for (const p of players) {
    parCle.set(p.xuid, p)
    if (p.board?.xuid && !parCle.has(p.board.xuid)) parCle.set(p.board.xuid, p)
  }
  const ref = reference ? parCle.get(reference) : undefined
  const refKey = ref?.xuid ?? null
  const refTeam = ref?.team ?? null
  const ofTeam = (team: number | null | undefined): Allegiance => {
    if (refTeam === null || refTeam < 0 || team == null || team < 0) return null
    return team === refTeam
  }
  const ofPlayer = (p: ReplayPlayer | null | undefined): Allegiance => {
    if (!p || p.team === undefined || refTeam === null) return null
    // MODE SANS CAMPS : personne ne partage le camp de la référence, qui reste alliée d'elle-même.
    if (refTeam < 0) return p.xuid === refKey
    return ofTeam(p.team)
  }
  const membres = new Map<number, string[]>()
  for (const p of players) {
    if (p.team === undefined || p.team < 0) continue
    const liste = membres.get(p.team)
    if (liste) liste.push(p.xuid)
    else membres.set(p.team, [p.xuid])
  }
  return {
    referenceTeam: refTeam,
    camps: groupByTeam(players)
      .filter((camp) => camp.team >= 0)
      .map(({ team, side }) => ({ team, side })),
    ofTeam,
    ofPlayer,
    ofXuid: (xuid) => (xuid ? ofPlayer(parCle.get(xuid)) : null),
    teamOfXuid: (xuid) => (xuid ? parCle.get(xuid)?.team ?? null : null),
    membersOf: (team) => membres.get(team) ?? AUCUN_MEMBRE,
  }
}

/** L'allégeance d'une page sans film (ou sans référence) : personne n'a d'encre de camp. */
export const NO_ALLEGIANCE: FilmAllegiance = buildFilmAllegiance([], null)

/**
 * filmAllegianceOf — la même allégeance, construite depuis le document et la feuille : pour les
 * surfaces qui ne reçoivent pas la jointure toute faite (les tables de l'onglet Arsenal, montées
 * sur la page Match).
 */
export function filmAllegianceOf(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[],
  reference: string | null | undefined,
): FilmAllegiance {
  return buildFilmAllegiance(buildPlayers(doc, scoreboard), reference)
}
