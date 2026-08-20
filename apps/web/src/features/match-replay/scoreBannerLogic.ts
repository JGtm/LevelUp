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
 * LE REMPLISSAGE EST RELATIF, FAUTE D'OBJECTIF PUBLIÉ — et c'est le repli prévu par la
 * demande. Un objectif de score (50 frags, 3 drapeaux) donnerait la seule fraction
 * absolument juste, mais AUCUN champ ne le porte, ni au contrat ni dans le film (recherche
 * du 2026-08-20) : `ScoreTimeline` n'a que `players` et `teams`, `TeamScore` que
 * `teamId`/`rounds`/`total`, l'entrée du constructeur (`replay.ScoreInput`) ne reçoit que
 * les scores FINAUX, et le seul champ de score de l'en-tête de match
 * (`MatchViewHeader.score_label`) est ce même score final mis en forme. Les vraies limites
 * (Slayer 50, Oddball 200) n'existent qu'en PROSE, dans le commentaire d'une borne
 * anti-aberration du backend (`objectiveevents/statborg.go`) — jamais en donnée.
 *
 * Le dépôt a déjà tranché ce manque au même endroit : le moteur de comeback, qui a besoin
 * d'un normalisateur, prend `max(scoreFinalA, scoreFinalB)` faute de mieux
 * (`internal/analysis/comeback.go`). On fait donc pareil, au frame lu :
 * `score / max(scoreAllié, scoreAdverse, 1)`. Le camp en tête remplit sa barre, l'autre
 * montre son RETARD RELATIF. Ce que la barre dessine est « où en est ce camp par rapport à
 * l'autre », jamais « où en est ce camp par rapport à la victoire ». Le nombre écrit dans la
 * barre, lui, reste la mesure — c'est lui qui fait foi. Le jour où le producteur publiera une
 * cible, seule `fillOf` change.
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
  return {
    ally: { teamId: allyId, score: allyScore, fill: fillOf(allyScore, enemyScore) },
    enemy: { teamId: enemyId, score: enemyScore, fill: fillOf(enemyScore, allyScore) },
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
 * fillOf — la part de barre d'un camp, relative au meilleur des deux (cf. en-tête).
 *
 * Le `1` du dénominateur tient le début de match : 0 sur 0 remplirait une barre entière
 * alors que personne n'a marqué. Deux barres vides, c'est la vérité du score 0-0.
 */
function fillOf(score: number, other: number): number {
  return score / Math.max(score, other, 1)
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
