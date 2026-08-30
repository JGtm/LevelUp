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
 * LA BASCULE PARTAGE CETTE BORNE (`roundTransitions`), ET C'EST UN CORRECTIF DU 2026-08-29.
 * Elle était datée AU DÉBUT DE LA MANCHE SUIVANTE — « on passe à la suite », à la reprise — et
 * c'est de là que partaient le message « Manche N terminée » et la voix de l'annonceur. Or ce
 * début n'est pas un instant du déroulé : c'est le PREMIER PALIER DE SCORE de la manche
 * suivante, donc le moment où quelqu'un marque à nouveau, après l'entracte ET après le temps
 * qu'il faut pour reprendre le crâne. Mesure sur les quatre témoins multi-manches du dossier
 * d'artefacts (2026-08-29, `data/cache/replays/halo_infinite`) : 24dbb67d +26,9 s ; 43716616
 * +19,0 s ; 51ebbc0f +28,9 s ; d9781168 +34,1 s puis +21,5 s. L'annonce arrivait donc une
 * demi-minute après la manche qu'elle annonçait, par-dessus la manche suivante déjà commencée.
 *
 * RÈGLE DU DÉPÔT (retour utilisateur du 2026-08-29) : un message, un son, une voix se déclenche
 * SUR SON ÉVÉNEMENT quand cet événement est daté, et sur T0 sinon — jamais sur un instant
 * voisin. L'événement, ici, c'est la fin de la manche : le dernier palier, celui-là même qui
 * remplit la pastille. Les deux bornes n'en font donc plus qu'une, et il n'y a plus qu'un
 * instant « fin de manche » dans tout le rejeu.
 *
 * # CE DERNIER PALIER N'EST PAS UN PIS-ALLER : C'EST LA MARQUE DE MANCHE DU FILM
 *
 * La question mérite d'être écrite parce qu'elle se reposera : le film ne publie AUCUN
 * événement « manche terminée », alors sur quoi s'appuie-t-on vraiment ?
 *
 * Sur le fait que le film ÉTIQUETTE CHACUN DE SES ENREGISTREMENTS PAR UNE MANCHE
 * (`StatRecord.Round`, statborg — cf. `analysis/objectiveevents/statborg.go`). Toutes les séries
 * que l'artefact en tire sont donc bornées par la manche : le score de MODE de chaque camp, et
 * les quatre compteurs de CHAQUE joueur (score personnel, frags, morts, assistances). Mesure du
 * 2026-08-29 sur les 4 films multi-manches du dossier d'artefacts, 5 bascules :
 * **toutes ces séries se figent sur la MÊME image, à zéro milliseconde près** (écart mesuré
 * entre « dernier palier d'équipe » et « dernier palier toutes séries confondues » : 0,0 s, 5
 * fois sur 5). Ce n'est donc pas « le dernier point marqué » qu'on lit, c'est **l'instant où le
 * film cesse de décrire cette manche**.
 *
 * DEUX CONSÉQUENCES PRATIQUES :
 *  - inutile de publier des bornes de manche depuis le décodeur Go : elles tomberaient sur
 *    exactement cette image, au prix d'un champ de schéma et d'une re-cuisson de tous les
 *    artefacts ;
 *  - inutile d'aller chercher un signal « plus physique ». Le terrain se VIDE bien (le film
 *    cesse de publier tout le monde pendant 0,5 à 1,7 s, le temps de la téléportation) mais
 *    **7,6 à 9,1 s APRÈS** : c'est l'écran de fin de manche du jeu, pas sa fin. S'y caler
 *    replacerait un décalage de huit secondes là où on vient d'en retirer trente.
 *
 * LA LIMITE QUI RESTE, NOMMÉE : une manche close AU CHRONO dont plus personne ne marque NI ne
 * meurt dans les dernières secondes figerait ses séries un peu avant sa fin réelle. Le corpus
 * n'en contient aucune (les 5 bascules finissent au plafond de manche — 80 ou 100 selon la
 * variante), et l'erreur serait EN AVANCE de quelques secondes, pas de trente. On ne corrige pas
 * ce qu'on n'a pas mesuré : le jour où un film le montre, c'est ici qu'il faudra revenir.
 */
import {
  teamRoundScoreAtFrame,
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

/** Une bascule de manche : l'instant où une manche se termine. */
export interface RoundTransition {
  /** Rang d'affichage (1-based) de la manche qui vient de se terminer. */
  endedIndex: number
  /**
   * Frame du film où la manche s'est terminée — son DERNIER PALIER de score, la même borne
   * que la pastille pleine du bandeau (cf. l'en-tête). Ce n'est plus le début de la manche
   * suivante : celui-ci tombe 19 à 34 s plus tard sur les témoins mesurés.
   */
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

/** La manche courante d'un MATCH (pas d'un camp), et ce qu'il faut pour la nommer. */
export interface CurrentRound {
  /** Numéro de manche du film (0-based) — sert au numérateur par-manche, jamais de libellé. */
  round: number
  /** Rang d'affichage à partir de 1 (« Manche 2 »). */
  index: number
  /** Nombre de manches JOUÉES. */
  count: number
}

/**
 * currentRoundAtFrame rend la manche EN COURS à l'image lue, ou `null` si aucune manche n'est
 * ventilée (mode sans manche non déclaré, calque absent).
 *
 * LA BORNE EST PARTAGÉE : une manche commence au premier palier de l'un OU l'autre camp
 * (`orderedRounds.start` prend déjà le min des deux). Les deux barres du bandeau basculent
 * donc ENSEMBLE, jamais l'une avant l'autre. La manche courante est la DERNIÈRE dont le début
 * est déjà passé ; avant le premier début, c'est la première (le match a commencé, pas le
 * compteur). Dans la fenêtre inter-manche — entre la fin d'une manche et le début de la
 * suivante — c'est encore la manche PRÉCÉDENTE : son compteur tient jusqu'à la reprise.
 * Un mode à manche unique rend `{index: 1, count: 1}`.
 */
export function currentRoundAtFrame(
  timeline: ReplayScoreTimelineReady | undefined,
  frame: number,
): CurrentRound | null {
  if (!timeline) return null
  const rounds = orderedRounds(timeline)
  if (rounds.length === 0) return null
  let idx = 0
  for (let i = 0; i < rounds.length; i++) {
    if (rounds[i].start <= frame) idx = i
  }
  return { round: rounds[idx].round, index: idx + 1, count: rounds.length }
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
 * Aucune bascule sur un mode à manche unique. La bascule est datée à la FIN de la manche qui
 * se termine (son dernier palier), pas au début de la suivante — cf. l'en-tête et la mesure.
 *
 * LA DERNIÈRE MANCHE N'EN PRODUIT AUCUNE, et c'est inchangé : sa fin est la fin du MATCH, que
 * l'écran de fin et la voix de conclusion annoncent déjà (`ReplayVictoryOverlay`,
 * `endMatchSound.ts`). Deux annonces sur le même instant se marcheraient dessus.
 */
export function roundTransitions(
  timeline: ReplayScoreTimelineReady | undefined,
): RoundTransition[] {
  if (!timeline) return []
  const rounds = orderedRounds(timeline)
  const out: RoundTransition[] = []
  for (let i = 1; i < rounds.length; i++) {
    out.push({ endedIndex: i, frame: rounds[i - 1].end })
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
 *
 * Délègue à `teamRoundScoreAtFrame` (le point unique du motif `find(round) + scoreAtFrame`) :
 * lu à la borne de clôture, le score de manche EST son score final.
 */
function roundFinal(
  timeline: ReplayScoreTimelineReady,
  teamId: number,
  roundNumber: number,
  endFrame: number,
): number {
  return teamRoundScoreAtFrame(timeline, teamId, roundNumber, endFrame)
}

/** Le compte de manches gagnées par camp, à l'image lue. */
export interface RoundsTally {
  ally: number
  enemy: number
}

/**
 * roundsTally compte les manches DÉJÀ TRANCHÉES par camp, à l'image lue.
 *
 * C'est la lecture chiffrée des pastilles, et rien d'autre : mêmes données, même borne, même
 * relativité au joueur de la page. Une manche en cours, à jouer, ou finie sur une égalité
 * n'est comptée nulle part — elle n'a pas de vainqueur, et en inventer un ferait mentir le
 * total.
 *
 * CE N'EST PAS LE VERDICT DU MATCH. Ce compte suit la lecture : il vaut 0-0 au coup d'envoi
 * et se remplit à mesure. Le résultat final, lui, vient de l'API (`score_mine` /
 * `score_theirs` de l'en-tête de la vue match) — la seule source qui fasse foi, et la même
 * que celle affichée partout ailleurs dans l'app.
 */
export function roundsTally(dots: readonly RoundDot[]): RoundsTally {
  let ally = 0
  let enemy = 0
  for (const dot of dots) {
    if (dot.winner === 'ally') ally++
    else if (dot.winner === 'enemy') enemy++
  }
  return { ally, enemy }
}
