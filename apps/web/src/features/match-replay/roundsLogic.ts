/**
 * roundsLogic — LES MANCHES D'UN MATCH MULTI-MANCHE, lues dans le calque de score.
 *
 * Oddball est le premier mode à passer au rejeu qui se joue en PLUSIEURS MANCHES (premier à
 * deux gagnées, trois au plus). Ce module dit trois choses que le calque de score porte sans
 * les nommer : COMBIEN de manches ont été jouées, QUI a gagné chacune, et OÙ elles se
 * touchent (la bascule d'une manche à la suivante). Le bandeau en tire ses pastilles, l'écran
 * inter-manche sa bascule. Module PUR : ni React, ni DOM, ni couleur, ni libellé.
 *
 * CE QUE LE FILM PUBLIE, ET CE QU'IL NE PUBLIE PAS. Chaque équipe porte ses `rounds[]`
 * (`ReplayTeamScoreReady`), et une manche est une suite de paliers `{t, v}` qui REPART DE
 * ZÉRO à la manche suivante (cf. `scoreTimeline.ts`). Le film ne publie AUCUN seuil de
 * « manches gagnantes » : `targetScore` est le plafond de score DANS une manche (100 pour
 * Oddball), pas le « premier à deux ». On ne code donc pas « trois manches » en dur — le
 * nombre de pastilles est le nombre de manches RÉELLEMENT JOUÉES (celles qui portent au moins
 * un palier), et une pastille pleine est une manche déjà TRANCHÉE à l'image lue.
 *
 * LE VAINQUEUR D'UNE MANCHE SE DÉDUIT, IL NE SE SUPPOSE PAS : c'est le camp au score de manche
 * le plus haut À LA FIN de la manche (le dernier palier de la manche). Un mode où l'on gagne
 * en atteignant le plafond finit sur ce plafond ; un mode arrêté au chrono finit sur l'écart —
 * dans les deux cas, le plus haut l'emporte. Une manche où les deux finissent à égalité (cas
 * dégénéré, qu'Oddball ne produit pas puisqu'une manche se gagne à 100) ne désigne personne :
 * pastille vide, jamais un vainqueur inventé.
 *
 * « ALLIÉ / ADVERSE » EST RELATIF AU JOUEUR DE LA PAGE, comme partout sur le rejeu : le pont
 * entre les `teamId` du film et ces deux notions est fait par l'appelant (`scoreBannerLogic`),
 * qui seul lit le scoreboard. Ce module reçoit les deux identifiants d'équipe déjà tranchés.
 *
 * UNE MANCHE EST TRANCHÉE À L'IMAGE LUE quand la lecture a atteint son DERNIER PALIER — l'instant
 * où le score de la manche cesse de bouger, c'est-à-dire où elle est gagnée (le plafond atteint
 * en Oddball, l'écart figé au chrono). Le film étant complet, ce dernier palier EST l'instant
 * décisif : avant lui la manche est en cours, après lui elle est gagnée. La pastille se remplit
 * donc AU MOMENT DE LA VICTOIRE, pas à l'ouverture de la manche suivante (il y a un entracte
 * entre les deux) — dérivée de la position de lecture, elle se vide quand on remonte la frise.
 *
 * LA BASCULE, elle, est datée AU DÉBUT DE LA MANCHE SUIVANTE (`roundTransitions`) : c'est là que
 * l'entracte se termine et que le message « manche terminée » a sa place. Pastille et message ne
 * partagent donc pas la même borne, et c'est voulu — l'une dit « gagnée » (au point décisif),
 * l'autre « on passe à la suite » (à la reprise).
 */
import {
  scoreAtFrame,
  teamSeriesFor,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'

/** Le camp gagnant d'une manche, du point de vue du joueur de la page. */
export type RoundWinner = 'ally' | 'enemy'

/** Une pastille de manche, à l'image lue. */
export interface RoundDot {
  /** Numéro de manche du film (0-based) — clé de rendu, jamais un libellé. */
  round: number
  /**
   * Le vainqueur, quand la manche est TRANCHÉE à l'image lue (pastille pleine) ; `null` quand
   * elle est en cours, à jouer, ou finie sur une égalité (pastille vide).
   */
  winner: RoundWinner | null
}

/** Une bascule de manche : l'instant où une manche se termine et où la suivante commence. */
export interface RoundTransition {
  /** Rang d'affichage (1-based) de la manche qui vient de se terminer. */
  endedIndex: number
  /** Frame du film où la manche suivante commence. */
  frame: number
}

/** Une manche ordonnée du match : son numéro, son début et sa borne de clôture, en frames. */
interface OrderedRound {
  round: number
  /** Frame du premier palier de la manche (toutes équipes confondues). */
  start: number
  /** Frame du DERNIER palier de la manche : à partir d'elle, la manche est tranchée (gagnée). */
  end: number
}

/**
 * orderedRounds rend les manches JOUÉES du match, dans l'ordre, avec leur borne de clôture.
 *
 * Une manche « jouée » porte au moins un palier chez l'un des deux camps. Une manche que le
 * film ne renseigne nulle part n'existe pas ici — pas de pastille pour une manche fantôme.
 * L'ordre est celui des NUMÉROS de manche (0-based, croissants avec le temps).
 */
function orderedRounds(timeline: ReplayScoreTimelineReady): OrderedRound[] {
  const byRound = new Map<number, { start: number; last: number }>()
  for (const team of timeline.teams) {
    for (const r of team.rounds) {
      if (r.points.length === 0) continue
      const first = r.points[0].t
      const last = r.points[r.points.length - 1].t
      const cur = byRound.get(r.round)
      if (!cur) byRound.set(r.round, { start: first, last })
      else {
        cur.start = Math.min(cur.start, first)
        cur.last = Math.max(cur.last, last)
      }
    }
  }
  const nums = [...byRound.keys()].sort((a, b) => a - b)
  return nums.map((round) => {
    const cur = byRound.get(round)!
    return { round, start: cur.start, end: cur.last }
  })
}

/** roundCount rend le nombre de manches JOUÉES — 0 ou 1 quand le mode n'est pas multi-manche. */
export function roundCount(timeline: ReplayScoreTimelineReady | undefined): number {
  return timeline ? orderedRounds(timeline).length : 0
}

/**
 * roundDots rend une pastille par manche jouée, à l'image lue — pleine (au camp gagnant) quand
 * la manche est tranchée, vide sinon. Tableau vide sur un mode à manche unique (une seule
 * pastille ne dirait rien que le total ne dise déjà).
 */
export function roundDots(
  timeline: ReplayScoreTimelineReady | undefined,
  allyId: number,
  enemyId: number,
  frame: number,
): RoundDot[] {
  if (!timeline) return []
  const rounds = orderedRounds(timeline)
  if (rounds.length <= 1) return []
  return rounds.map((r) => ({
    round: r.round,
    winner: frame >= r.end ? winnerOfRound(timeline, allyId, enemyId, r) : null,
  }))
}

/**
 * roundTransitions rend les bascules de manche : une par passage d'une manche à la suivante.
 * Aucune bascule sur un mode à manche unique. La bascule est datée au DÉBUT de la manche
 * suivante — l'instant où le film ouvre un nouveau compteur, donc où le précédent est clos.
 */
export function roundTransitions(
  timeline: ReplayScoreTimelineReady | undefined,
): RoundTransition[] {
  if (!timeline) return []
  const rounds = orderedRounds(timeline)
  const out: RoundTransition[] = []
  for (let i = 1; i < rounds.length; i++) {
    out.push({ endedIndex: i, frame: rounds[i].start })
  }
  return out
}

/**
 * activeRoundTransition rend la bascule dont la FENÊTRE D'AFFICHAGE contient l'image lue, ou
 * `null`. La fenêtre court de la bascule à `windowFrames` frames plus loin ; on retient la
 * PLUS RÉCENTE qui la contient (deux fenêtres ne se chevauchent pas sur un vrai match, mais
 * la plus récente est la bonne réponse si elles le faisaient). Dérivée de la position de
 * lecture, exactement comme l'écran de fin : elle se rejoue si l'on repasse dessus, et
 * disparaît dès qu'on la quitte.
 */
export function activeRoundTransition(
  transitions: readonly RoundTransition[],
  frame: number,
  windowFrames: number,
): RoundTransition | null {
  let best: RoundTransition | null = null
  for (const tr of transitions) {
    if (frame >= tr.frame && frame < tr.frame + windowFrames) {
      if (!best || tr.frame > best.frame) best = tr
    }
  }
  return best
}

/** winnerOfRound compare les scores de FIN de manche des deux camps ; `null` sur une égalité. */
function winnerOfRound(
  timeline: ReplayScoreTimelineReady,
  allyId: number,
  enemyId: number,
  r: OrderedRound,
): RoundWinner | null {
  const ally = roundFinal(timeline, allyId, r.round, r.end)
  const enemy = roundFinal(timeline, enemyId, r.round, r.end)
  if (ally > enemy) return 'ally'
  if (enemy > ally) return 'enemy'
  return null
}

/**
 * roundFinal rend le score d'un camp DANS une manche à la borne de clôture — son dernier
 * palier de la manche. 0 quand le camp n'a pas de série pour cette manche (il n'a pas marqué),
 * ce qui est une mesure et non une lacune (cf. la doctrine « une équipe sans série vaut zéro »).
 */
function roundFinal(
  timeline: ReplayScoreTimelineReady,
  teamId: number,
  roundNumber: number,
  endFrame: number,
): number {
  const team = teamSeriesFor(timeline, teamId)
  const r = team?.rounds.find((x) => x.round === roundNumber)
  if (!r) return 0
  return scoreAtFrame(r.points, endFrame)
}
