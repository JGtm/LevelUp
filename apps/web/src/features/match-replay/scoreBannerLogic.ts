/**
 * scoreBannerLogic — CE QUE LE BANDEAU DE SCORE A LE DROIT D'AFFICHER, au frame lu.
 *
 * Le bandeau tient au-dessus du terrain : `[ barre alliée ] — [ horloge ] — [ barre adverse ]`.
 * Ce module rend la LECTURE (deux camps, deux scores, deux fractions, la manche) ; le
 * composant n'en fait que du JSX. Il ne calcule RIEN du score lui-même : la lecture au frame
 * courant est celle de `lib/replay/scoreTimeline.ts`, déjà éprouvée par les trois témoins
 * re-cuits (Slayer 43/50, CTF 3-0, Oddball 200/121). Ce fichier n'ajoute que ce que le
 * bandeau, et lui seul, doit trancher : QUELS SONT LES DEUX CAMPS, LEQUEL EST À GAUCHE, et
 * JUSQU'OÙ VA LA BARRE.
 *
 * LES CAMPS SE LISENT AU SCOREBOARD, PAS DANS LE CALQUE DE SCORE — et c'est le piège que ce
 * module doit éviter. Une équipe qui n'a jamais marqué n'émet AUCUNE série : le témoin CTF
 * `530820e5` (3-0) ne publie qu'un seul camp sur les deux. Compter les camps dans
 * `timeline.teams` ferait donc disparaître le bandeau du match où l'écart est le plus net.
 * Les camps viennent de `team_side` (la même source que les colonnes de fiches,
 * `rosterLogic.groupByTeam`), et le score de chacun de `teamScoreAtFrame`, qui rend 0 pour
 * un camp sans série — la vérité du film, pas une lacune.
 *
 * MAIS UN FILM QUI NE PUBLIE AUCUNE SÉRIE NE DIT PAS « 0 — 0 » : il ne dit rien. Sans le
 * moindre camp dans le calque (artefact antérieur au schéma 12, mode sans compteur, horloge
 * non recalée), deux barres à zéro se liraient comme une mesure alors que personne n'a
 * compté. Le bandeau est alors absent — même doctrine qu'en tête de colonne
 * (`ReplayTeamHeader`).
 *
 * DEUX CAMPS, PAS UN DE PLUS, PAS UN DE MOINS. La forme demandée n'a que deux barres ; elle
 * ne peut donc pas dire un mode à trois camps sans nommer arbitrairement UN adversaire parmi
 * plusieurs, ni un FFA, où chacun joue pour soi. Dans ces cas la lecture est `null` et le
 * bandeau ne se rend pas : une barre absente ne ment pas, une barre qui désigne le mauvais
 * camp si.
 *
 * « ALLIÉ » EST UNE NOTION RELATIVE, ET SON ABSENCE EST DISQUALIFIANTE ICI. Le film numérote
 * ses équipes ; « allié » veut dire « du côté du joueur dont on regarde la page », ce que
 * seul le scoreboard sait dire (`allyOfTeamId`). Tant que ce pont n'est pas fait — vue match
 * pas encore chargée, aucun joueur reconnu — le bandeau n'a NI côté NI couleur : il ne se
 * rend pas. C'est la doctrine des deux autres panneaux (« un groupe sans camp connu
 * n'emprunte aucune des deux couleurs ») poussée à sa conséquence pour une surface dont le
 * camp EST toute la structure.
 *
 * AVEC DEUX CAMPS, UN SEUL SUFFIT À NOMMER L'AUTRE. `allyOfTeamId` rend `null` pour un camp
 * dont aucun joueur n'est reconnu. Quand l'AUTRE, lui, est identifié, on ne perd pas le
 * bandeau pour autant : entre exactement deux camps, « ce n'est pas le camp allié » vaut
 * « c'est le camp adverse ». Seules restent écartées les lectures réellement ambiguës — les
 * deux camps alliés, les deux adverses, ou aucun des deux résolu.
 *
 * LE DÉNOMINATEUR EST LA CIBLE DE VICTOIRE (demande utilisateur du 2026-08-24 : « pleine
 * quand le match est fini parce que le score de la victoire est atteint, c'est ça le
 * dénominateur »). Le film ne la porte pas ; c'est l'artefact qui la publie
 * (`scoreTimeline.targetScore`) depuis la table MESURÉE de la variante
 * (config regulation.toml [score_target], plateaux du score du vainqueur au registre). Une
 * victoire AU CHRONO s'affiche donc juste : 43/50, jamais une barre pleine à 43.
 *
 * À DÉFAUT (artefact antérieur au champ, variante hors table, table périmée que le
 * producteur a tue), le repli est celui du moteur de comeback
 * (`internal/analysis/comeback.go`) : `max(scoreFinalA, scoreFinalB)` — dans une partie
 * gagnée à la cible, le score final du vainqueur EST la cible. Dans les deux cas le
 * dénominateur est CONSTANT sur toute la lecture : une barre ne recule JAMAIS (la version
 * relative au camp d'en face au frame lu se vidait quand l'adversaire marquait — refusée
 * par l'utilisateur). Le nombre écrit dans la barre reste la mesure — c'est lui qui fait
 * foi.
 *
 * Module PUR : ni React, ni DOM, ni couleur.
 */
import {
  allyOfTeamId,
  roundAtFrame,
  teamIdOfSide,
  teamScoreAtFrame,
  teamSeriesFor,
  type ReplayScoreTimelineReady,
  type ReplayTeamScoreReady,
} from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

/**
 * Ce qu'il faut savoir d'un camp pour dessiner sa barre.
 *
 * `fill` est une fraction 0..1 RELATIVE à l'autre camp (cf. en-tête) ; `score` est la mesure.
 */
export interface ScoreBannerSide {
  /** Identifiant d'équipe du film — sert de clé de rendu, jamais de libellé. */
  teamId: number
  /** Score TOTAL du match au frame lu (pas celui de la manche). */
  score: number
  /** Part de barre remplie, dans [0,1]. */
  fill: number
}

/** La manche en cours, quand le mode en a plusieurs — sinon le bandeau n'en parle pas. */
export interface ScoreBannerRound {
  /** Rang d'affichage à partir de 1 (« Manche 2 »). */
  index: number
  /** Nombre de manches publiées. */
  count: number
}

/** La lecture complète du bandeau à un frame donné. */
export interface ScoreBannerReading {
  ally: ScoreBannerSide
  enemy: ScoreBannerSide
  round: ScoreBannerRound | null
}

/** Les lignes de scoreboard dont ce module a besoin : le camp et l'identité, rien d'autre. */
type ScoreboardRows = ReadonlyArray<Pick<MatchScoreboardRow, 'xuid' | 'team_side'>>

/** De quel côté est chaque joueur (`XuidMeta` s'y conforme sans que ce module le nomme). */
type AllyIndex = ReadonlyMap<string, { ally: boolean }>

/**
 * readScoreBanner rend la lecture du bandeau, ou `null` quand il ne doit pas se rendre :
 * calque sans aucun camp, mode qui n'oppose pas exactement deux camps, ou côté allié
 * indéterminable (cf. en-tête).
 */
export function readScoreBanner(
  timeline: ReplayScoreTimelineReady | undefined,
  scoreboard: ScoreboardRows,
  allies: AllyIndex | undefined,
  frame: number,
): ScoreBannerReading | null {
  if (!timeline || timeline.teams.length === 0) return null
  const camps = identifiedCamps(scoreboard)
  if (camps.length !== 2) return null
  const allyIdx = allySideIndex(scoreboard, allies, camps)
  if (allyIdx === null) return null
  const allyId = camps[allyIdx]
  const enemyId = camps[1 - allyIdx]
  const allyScore = teamScoreAtFrame(timeline, allyId, frame)
  const enemyScore = teamScoreAtFrame(timeline, enemyId, frame)
  const target = victoryTarget(timeline, allyId, enemyId)
  return {
    ally: { teamId: allyId, score: allyScore, fill: fillOf(allyScore, target) },
    enemy: { teamId: enemyId, score: enemyScore, fill: fillOf(enemyScore, target) },
    round: roundOf(timeline, allyId, enemyId, frame),
  }
}

/**
 * identifiedCamps rend les camps du match, par identifiant d'équipe du film, dans un ordre
 * déterministe (croissant — l'ordre à l'écran vient du côté allié, pas de celui-ci).
 *
 * Une ligne sans camp transmis n'en fabrique pas un : elle est simplement ignorée, comme le
 * groupe `side: null` des fiches. Un joueur non situé ne change pas le nombre de camps.
 */
function identifiedCamps(scoreboard: ScoreboardRows): number[] {
  const ids = new Set<number>()
  for (const row of scoreboard) {
    const id = teamIdOfSide(row.team_side)
    if (id != null) ids.add(id)
  }
  return [...ids].sort((a, b) => a - b)
}

/**
 * victoryTarget — le dénominateur des deux barres, CONSTANT sur toute la lecture (c'est ce
 * qui garantit qu'une barre ne recule jamais). Deux sources, dans l'ordre :
 *
 *  1. `timeline.targetScore` — la CIBLE DE VICTOIRE du mode, publiée par l'artefact depuis
 *     la table mesurée de la variante (regulation.toml [score_target]). C'est la seule
 *     lecture juste sur une victoire AU CHRONO : un Slayer arrêté à 43 affiche 43/50, pas
 *     une barre pleine (retour utilisateur du 2026-08-24). Le producteur garantit
 *     cible >= finals ; un artefact antérieur au champ n'en porte pas.
 *  2. le score FINAL du vainqueur, à défaut (cf. en-tête — dans une partie gagnée à la
 *     cible, c'est la cible ; au chrono, c'est une borne basse assumée, faute de donnée).
 *
 * Le `1` tient le cas dégénéré d'un match sans le moindre point : 0 sur 0 remplirait une
 * barre entière alors que personne n'a marqué.
 */
function victoryTarget(
  timeline: ReplayScoreTimelineReady,
  allyId: number,
  enemyId: number,
): number {
  if (timeline.targetScore != null && timeline.targetScore > 0) return timeline.targetScore
  return Math.max(finalScoreOf(timeline, allyId), finalScoreOf(timeline, enemyId), 1)
}

/** Le score de FIN DE MATCH d'un camp : son dernier palier, 0 sans série (camp muet). */
function finalScoreOf(timeline: ReplayScoreTimelineReady, teamId: number): number {
  const total = teamSeriesFor(timeline, teamId)?.total ?? []
  return total.length > 0 ? total[total.length - 1].v : 0
}

/** fillOf — la part de barre d'un camp : son score au frame lu, rapporté à la cible. */
function fillOf(score: number, target: number): number {
  return Math.min(score / target, 1)
}

/**
 * allySideIndex dit LEQUEL des deux camps est celui du joueur de la page (0 ou 1), ou `null`
 * quand la question n'a pas de réponse sûre.
 *
 * Un seul camp résolu suffit puisqu'ils sont exactement deux (cf. en-tête). Sont écartées les
 * seules lectures contradictoires ou muettes — deux camps du même côté (le scoreboard se
 * contredirait), ou aucun des deux reconnu.
 */
function allySideIndex(
  scoreboard: ScoreboardRows,
  allies: AllyIndex | undefined,
  camps: readonly number[],
): 0 | 1 | null {
  const first = allyOfTeamId(scoreboard, allies, camps[0])
  const second = allyOfTeamId(scoreboard, allies, camps[1])
  if (first === true && second !== true) return 0
  if (second === true && first !== true) return 1
  if (first === false && second !== false) return 1
  if (second === false && first !== false) return 0
  return null
}

/**
 * roundOf rend la manche en cours, et `null` sur un mode à manche unique — où l'indicateur
 * ne dirait rien que le total ne dise déjà.
 *
 * Le RANG est une propriété du match, pas d'un camp : peu importe de quelle série il se lit.
 * On retient celle qui publie le plus de manches, parce qu'un camp peut n'avoir marqué dans
 * aucune (témoin CTF : une seule série pour deux camps) et sous-déclarerait le compte.
 */
function roundOf(
  timeline: ReplayScoreTimelineReady,
  allyId: number,
  enemyId: number,
  frame: number,
): ScoreBannerRound | null {
  const best = longerSeries(
    teamSeriesFor(timeline, allyId),
    teamSeriesFor(timeline, enemyId),
  )
  const reading = roundAtFrame(best, frame)
  if (!reading || reading.count <= 1) return null
  return { index: reading.index, count: reading.count }
}

/** La série qui publie le plus de manches — l'autre peut n'en avoir aucune. */
function longerSeries(
  a: ReplayTeamScoreReady | null,
  b: ReplayTeamScoreReady | null,
): ReplayTeamScoreReady | null {
  return (a?.rounds.length ?? 0) >= (b?.rounds.length ?? 0) ? a : b
}
