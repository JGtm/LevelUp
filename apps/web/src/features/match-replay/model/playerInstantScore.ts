/**
 * playerInstantScore.ts — LE SCORE PERSONNEL D'UN JOUEUR À L'IMAGE LUE, la seule lecture qu'en font la fiche du
 * joueur et l'infobulle de sa tuile. Sans React.
 *
 * Le score à l'instant est la somme des points par action (`scoreActions`) jusqu'à l'image. Il n'est établi que si
 * TROIS gardes tiennent : le document déclare le score par action complet (`coverage.scoreActions.complete`),
 * l'horloge du film est recalée (`filmClockTrusted`, la garde du calque de score) et, à la fin du match, la somme du
 * joueur ÉGALE le score personnel de sa ligne de feuille de l'API (`ReplayPlayer.board.score`). Le verdict du
 * document seul ne suffit pas : il peut être vrai alors que la somme s'écarte de la feuille. La garde vaut pour
 * tout build et tout mode ; aucune liste n'en décide.
 *
 * UNE SEULE VALEUR À L'INSTANT : le calque de score du film date un point une image après l'action et peut
 * s'écarter durablement de la feuille ; il ne sert qu'aux compteurs (frags, morts, assistances). Tout ce qui écrit
 * un score à l'instant lit `scoreOfReading`, sur une lecture rangée UNE fois par document (`readInstantScoreSeries`
 * pour la rangée des tuiles, `readInstantEstimatesSeries` pour la fiche) ; chaque image n'est qu'une dichotomie.
 *
 * Un score que le document ne permet pas d'établir est `null` (jamais zéro), sa cause à part (`ScoreAbsence`).
 */
import type { ReplayScoreTick } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { filmClockTrusted, scoreAtFrame } from '../../../lib/replay/scoreTimeline'
import { identityKeys } from './playerInstantStats'

/**
 * Pourquoi le score est absent :
 *  - `score-incomplet` : le document ne déclare pas le score par action complet ;
 *  - `horloge-non-recalee` : l'horloge du film n'est pas recalée, les points seraient datés à un écart inconnu ;
 *  - `feuille-absente` : le joueur n'a pas de score sur la feuille de l'API, rien ne permet de vérifier la somme ;
 *  - `somme-differente` : à la fin du match, la somme des points par action n'égale pas le score de la feuille.
 */
export type ScoreAbsence = 'score-incomplet' | 'horloge-non-recalee' | 'feuille-absente' | 'somme-differente'

/** Ce que le document permet de dire du score d'un joueur : ses paliers cumulés, ou la cause de leur absence. */
export type ScoreReading = { ticks: ReplayScoreTick[]; absence: null } | { ticks: null; absence: ScoreAbsence }

/** Ce que la lecture du score demande du document. */
export type ScoreDoc = Pick<ReplayDocumentReady, 'originMs' | 'coverage' | 'scoreActions'>

/** Les lectures de score d'un lobby, par xuid de joueur ; un joueur étranger au lobby n'y figure pas. */
export type InstantScoreSeries = ReadonlyMap<string, ScoreReading>

/** La cause d'absence qui vaut pour tout le document, ou `null`. */
function documentScoreAbsence(doc: ScoreDoc): ScoreAbsence | null {
  if (doc.coverage?.scoreActions?.complete !== true) return 'score-incomplet'
  if (!filmClockTrusted(doc)) return 'horloge-non-recalee'
  return null
}

/** Les points par action du document, rangés par destinataire (xuid décimal), dans l'ordre du temps. */
function actionsByRecipient(doc: ScoreDoc): Map<string, ReplayDocumentReady['scoreActions']> {
  const out = new Map<string, ReplayDocumentReady['scoreActions']>()
  for (const a of doc.scoreActions) {
    const list = out.get(a.xuid)
    if (list) list.push(a)
    else out.set(a.xuid, [a])
  }
  return out
}

/** Le score d'un joueur : les paliers cumulés de ses points, si la somme finale égale sa feuille. */
function readScore(
  player: ReplayPlayer,
  byRecipient: ReadonlyMap<string, ReplayDocumentReady['scoreActions']>,
  docAbsence: ScoreAbsence | null,
): ScoreReading {
  if (docAbsence) return { ticks: null, absence: docAbsence }
  const sheet = player.board?.score
  if (sheet === undefined || sheet === null) return { ticks: null, absence: 'feuille-absente' }
  const actions = identityKeys(player)
    .flatMap((k) => byRecipient.get(k) ?? [])
    .sort((a, b) => a.t - b.t)
  const ticks: ReplayScoreTick[] = []
  let total = 0
  for (const a of actions) {
    total += a.points
    ticks.push({ t: a.t, v: total })
  }
  return total === sheet ? { ticks, absence: null } : { ticks: null, absence: 'somme-differente' }
}

/**
 * scoreReaderOf range, une fois, les points par action du document et rend la lecture du score de chaque joueur
 * qu'on lui donne. Le rangement ne se refait pas par joueur.
 */
export function scoreReaderOf(doc: ScoreDoc): (player: ReplayPlayer) => ScoreReading {
  const docAbsence = documentScoreAbsence(doc)
  const byRecipient = docAbsence ? new Map<string, ReplayDocumentReady['scoreActions']>() : actionsByRecipient(doc)
  return (player) => readScore(player, byRecipient, docAbsence)
}

/** readInstantScoreSeries lit, une fois par document, le score de chaque joueur du lobby. */
export function readInstantScoreSeries(doc: ScoreDoc, players: readonly ReplayPlayer[]): InstantScoreSeries {
  const scoreOf = scoreReaderOf(doc)
  return new Map(players.map((p) => [p.xuid, scoreOf(p)]))
}

/** scoreOfReading rend le score à `frame` (compris), ou `null` quand la lecture n'est pas établie. */
export function scoreOfReading(reading: ScoreReading | undefined, frame: number): number | null {
  return reading?.ticks ? scoreAtFrame(reading.ticks, frame) : null
}

/** playerScoreAt rend le score personnel de `player` à `frame`, ou `null` : joueur étranger au lobby, ou garde refusée. */
export function playerScoreAt(series: InstantScoreSeries, player: ReplayPlayer, frame: number): number | null {
  return scoreOfReading(series.get(player.xuid), frame)
}
