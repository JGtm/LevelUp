/**
 * victoryLogic — COMMENT CE MATCH S'EST TERMINÉ POUR LE JOUEUR DE LA PAGE.
 *
 * L'ÉCRAN DE FIN EST LE SIEN, PAS CELUI DU VAINQUEUR (amendement utilisateur du 2026-08-26,
 * en cours de lot). Comme dans le jeu, l'écran de fin porte les couleurs et le logo de VOTRE
 * équipe, que vous ayez gagné ou perdu — on ne se fait pas afficher l'emblème de l'adversaire
 * parce qu'il a gagné. Ce module rend donc TROIS choses distinctes : l'ISSUE (victoire, défaite,
 * égalité, du point de vue du joueur de la page), SON équipe (celle qui habille l'écran), et
 * l'équipe qui GAGNE (que l'écran peut nommer). En victoire les deux dernières coïncident ; en
 * défaite elles diffèrent, et c'est tout l'intérêt de les séparer.
 *
 * L'ISSUE SE LIT DANS L'EN-TÊTE, JAMAIS DANS LE SCORE (décision D-B2). L'en-tête publie
 * `outcome_code`, qui est le verdict du joueur DE LA PAGE (2 = victoire, 3 = défaite,
 * 1 = égalité) : c'est une donnée servie, pas une déduction. Déduire le vainqueur du calque de
 * score serait faux au moins deux fois — un mode sans compteur ne publie aucune série, et un
 * match gagné au CHRONO peut finir sur un score à égalité (témoin 64e8adfa, 2-2). Le code de
 * l'en-tête sait dire « gagné 2-2 » ; deux nombres non.
 *
 * DE « MON RÉSULTAT » À « L'ÉQUIPE QUI GAGNE » IL FAUT UN PONT, et c'est le scoreboard qui le
 * fait : la ligne `is_me` donne mon camp, et entre exactement deux camps « j'ai perdu » suffit
 * à nommer l'autre. Sans ligne `is_me`, ou si son camp n'est pas transmis, la question n'a pas
 * de réponse sûre — `null`, et l'écran ne se rend pas. Ce pont est aussi ce qui donne à l'écran
 * son habillage : pas de camp connu, pas d'identité à porter.
 *
 * DEUX CAMPS, PAS UN DE PLUS, PAS UN DE MOINS (décision D-B1), y compris pour l'égalité. La
 * même doctrine que le bandeau de score (`scoreBannerLogic.ts`) : un FFA n'a pas d'« équipe
 * gagnante » à nommer, et un mode à trois camps désignerait arbitrairement un adversaire parmi
 * plusieurs. Le panneau d'égalité, lui, ne nomme personne — mais il annonce la fin d'un match
 * OPPOSANT DEUX CAMPS, et l'afficher sur un FFA laisserait croire que les huit joueurs ont fini
 * à égalité. Une lecture absente ne ment pas, une lecture qui invente un cadre si.
 *
 * L'ÉGALITÉ NE REND AUCUNE ÉQUIPE, et c'est délibéré : elle ne désigne personne (décision
 * D-B1), donc le panneau reste neutre. Rendre `null` pour les deux équipes rend cette
 * neutralité IMPOSSIBLE À CONTOURNER par mégarde à l'écran, plutôt que de la confier à une
 * discipline de rendu.
 *
 * LE CODE HORS CONTRAT NE FABRIQUE RIEN. Le mapping code→issue est celui du dépôt,
 * `lib/outcome.ts` (source unique, garde-rail `outcome.guard.test.ts`) : un code absent, nul ou
 * hors 1..4 y rend `null`. Le DNF (code 4) rend bien une issue, mais pas un écran de fin : un
 * match quitté ne se conclut pas (D-B1).
 *
 * Module PUR : ni React, ni DOM, ni couleur, ni libellé.
 */
import { outcomeCodeToValue } from '@/lib/outcome'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Ce que ce module lit d'une ligne de scoreboard : le camp, et si c'est le joueur de la page. */
type VictoryRows = ReadonlyArray<Pick<MatchScoreboardRow, 'team_side' | 'is_me'>>

/** L'issue du match DU POINT DE VUE du joueur de la page. */
export type VictoryOutcome = 'win' | 'loss' | 'tie'

/** Une équipe désignée par la lecture : de quoi la nommer, la teinter et la coiffer. */
export interface VictoryTeam {
  /** Identifiant d'équipe (`t{N}` décodé) — clé du logo et de la couleur d'identité. */
  teamID: number
  /** Camp au format du backend (`t{N}`) — ce que la cascade de libellé attend. */
  teamSide: string
  /** `true` si c'est l'équipe du joueur de la page. */
  ally: boolean
}

/** La lecture de fin de match : l'issue, l'équipe qui habille l'écran, celle qui gagne. */
export interface VictoryReading {
  outcome: VictoryOutcome
  /** L'équipe DU JOUEUR DE LA PAGE — l'habillage de l'écran. `null` sur une égalité. */
  mine: VictoryTeam | null
  /** L'équipe qui remporte le match. `null` sur une égalité, qui ne désigne personne. */
  winner: VictoryTeam | null
}

/** Un camp du match : son identifiant décodé et le `team_side` d'origine, qui nomme. */
interface Camp {
  id: number
  side: string
}

/**
 * readVictory rend la lecture de fin de match, ou `null` quand aucun écran ne doit s'afficher :
 * match qui n'oppose pas exactement deux camps, résultat non publié ou hors contrat, abandon,
 * ou joueur de la page introuvable au scoreboard (cf. l'en-tête du module).
 */
export function readVictory(
  scoreboard: VictoryRows,
  outcomeCode: number | null | undefined,
): VictoryReading | null {
  const camps = identifiedCamps(scoreboard)
  if (camps.length !== 2) return null
  const outcome = outcomeCodeToValue(outcomeCode)
  if (outcome === 'tie') return { outcome: 'tie', mine: null, winner: null }
  if (outcome !== 'win' && outcome !== 'loss') return null
  const mineIndex = myCampIndex(scoreboard, camps)
  if (mineIndex === null) return null
  const won = outcome === 'win'
  const mine = camps[mineIndex]
  const winner = won ? mine : camps[1 - mineIndex]
  return {
    outcome,
    mine: { teamID: mine.id, teamSide: mine.side, ally: true },
    winner: { teamID: winner.id, teamSide: winner.side, ally: won },
  }
}

/**
 * identifiedCamps rend les camps du match dans un ordre déterministe (identifiant croissant —
 * l'ordre à l'écran ne dépend pas de celui-ci).
 *
 * Une ligne sans camp transmis n'en fabrique pas un : elle est ignorée, comme au bandeau de
 * score. Un joueur non situé ne change pas le NOMBRE de camps, il ne fait que ne pas compter.
 */
function identifiedCamps(scoreboard: VictoryRows): Camp[] {
  const bySide = new Map<number, string>()
  for (const row of scoreboard) {
    const id = parseTeamSideID(row.team_side)
    if (id != null && row.team_side && !bySide.has(id)) bySide.set(id, row.team_side)
  }
  return [...bySide.entries()]
    .map(([id, side]) => ({ id, side }))
    .sort((a, b) => a.id - b.id)
}

/**
 * myCampIndex dit LEQUEL des deux camps est celui du joueur de la page (0 ou 1), ou `null`
 * quand le scoreboard ne le dit pas : aucune ligne `is_me`, camp non transmis sur cette ligne,
 * ou camp qui ne figure pas parmi les deux retenus.
 */
function myCampIndex(scoreboard: VictoryRows, camps: readonly Camp[]): 0 | 1 | null {
  const mine = scoreboard.find((r) => r.is_me)
  const id = parseTeamSideID(mine?.team_side)
  if (id == null) return null
  if (camps[0].id === id) return 0
  if (camps[1].id === id) return 1
  return null
}

/** Le score FINAL du match, du point de vue du joueur de la page. */
export interface FinalScoreReading {
  ally: number
  enemy: number
}

/** Ce que l'écran de fin lit de l'en-tête de la vue match pour connaître le score final. */
export interface FinalScoreHeader {
  score_kind?: string
  score_mine?: number
  score_theirs?: number
}

/**
 * finalScoreFromHeader rend le score final SERVI PAR L'API quand il ne se déduit pas du film,
 * et `null` sinon — auquel cas l'écran de fin garde sa lecture habituelle du calque.
 *
 * POURQUOI L'API ICI, ALORS QUE TOUT LE RESTE DU REJEU VIENT DU FILM. Sur un mode qui se
 * décide aux MANCHES, la lecture du calque à la borne de fin rend les points de la DERNIÈRE
 * MANCHE (Oddball : « 100 - 43 »), présentés comme le score du match. C'est faux : le match
 * s'est joué en deux manches à une. Le compte de manches qui fait foi est celui de l'API,
 * déjà affiché en tête de la vue match — le prendre ici, c'est garantir que les deux surfaces
 * disent la MÊME chose. Le pendant vivant (le compte en cours pendant la lecture) reste
 * dérivé du film, cf. `roundsTally`.
 *
 * NE S'APPLIQUE QU'AUX MANCHES : sur un mode en points, le calque du film est plus fin que
 * l'en-tête (il sait le score à n'importe quelle image) et reste la source.
 */
export function finalScoreFromHeader(header: FinalScoreHeader | undefined): FinalScoreReading | null {
  if (!header || header.score_kind !== 'rounds') return null
  if (header.score_mine == null || header.score_theirs == null) return null
  return { ally: header.score_mine, enemy: header.score_theirs }
}
