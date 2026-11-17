/**
 * playerMovementStats.ts — LE SPRINT ET LES SAUTS D'UN JOUEUR DU DÉBUT DU MATCH JUSQU'À L'IMAGE LUE, sans React.
 *
 * DEUX TEMPS, comme `playerInstantStats`. `readMovementSeries` lit une fois par document les états de mouvement
 * (`doc.stances`), attribue chaque intervalle au joueur qui occupe son slot à son premier instant et range, par
 * joueur, des séries triées ; `playerMovementStatsAt` répond pour une image par dichotomie, sans rien relire.
 *
 * CE QUE LE DOCUMENT PUBLIE. `stances` est une liste d'intervalles `{slot, kind, t0, t1}` portés par une VIE
 * (le slot migre aux réapparitions : le joueur est celui qui occupe le slot à `t0`, `buildSlotOwnership`), sur
 * l'axe d'images des pistes, bornés à la fenêtre de la vie : un état ne court jamais pendant la mort.
 *  - `sprint` est LU dans un composant du film : un intervalle est un sprint mesuré.
 *  - `jumpDerived` est DÉRIVÉ de l'intégrale de la vitesse verticale et reconnu à sa hauteur : le document ne
 *    publie AUCUN compte de sauts lu, seulement ces intervalles dérivés. Tout compte de sauts est donc une
 *    estimation (`jumpsEstimated`) : l'écran doit le dire. Les intervalles dérivés se comptent un par un.
 *
 * LA RÈGLE DES FICHES VAUT ICI : une grandeur que le document ne permet pas d'établir est ABSENTE (`null`), jamais
 * un zéro. La couverture `coverage.stances` dit ce que la marche a établi :
 *  - sans couverture (artefact antérieur à ce calque) ou `scanned` faux (le balayage a refusé : « le reste est
 *    sans valeur »), ni sprint ni sauts ne sont établis ;
 *  - `absent` vrai (l'archétype du film ne déclare aucun des composants lus) : rien n'est établi non plus ;
 *  - `jumpEpisodes` à zéro ou omis (aucune montée examinée : le film ne réplique pas la vitesse, ou la dérivation
 *    n'existe pas à ce schéma) : les sauts ne sont pas établis, alors que le sprint peut l'être.
 * Quand la grandeur est établie, « aucun intervalle pour ce joueur » est un zéro MESURÉ.
 *
 * Les BOTS comptent comme tout joueur : leur clé de roster est `bot:<nom>` et le xuid de la feuille est
 * `board.xuid` ; l'index se lit par les deux (`identityKeys`).
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { frameToMs, lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildSlotOwnership, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { identityKeys } from './playerInstantStats'

/** Genre d'un intervalle de sprint (`stances[].kind`), LU dans un composant du film. */
export const STANCE_KIND_SPRINT = 'sprint'
/** Genre d'un intervalle de saut (`stances[].kind`), DÉRIVÉ de la vitesse verticale : une estimation. */
export const STANCE_KIND_JUMP_DERIVED = 'jumpDerived'

/** Ce que la lecture rend pour une image : tout est cumulé depuis le début du match. */
export interface PlayerMovementStats {
  /**
   * Temps cumulé de sprint jusqu'à l'image, en millisecondes ; un sprint EN COURS à l'image compte jusqu'à elle.
   * `0` = « aucun sprint » MESURÉ (la fiche n'écrit alors pas la ligne). `null` = le document ne permet pas de
   * l'établir (couverture des états de mouvement absente ou refusée, joueur sans aucune vie) : ce n'est PAS « à zéro ».
   */
  sprintMs: number | null
  /**
   * Nombre de sauts commencés jusqu'à l'image (un saut compte dès son premier instant). `0` = « aucun saut » mesuré.
   * `null` = le document ne permet pas de l'établir : mêmes cas que `sprintMs`, plus une dérivation qui n'a examiné
   * aucune montée (`coverage.stances.jumpEpisodes` nul ou omis).
   */
  jumps: number | null
  /**
   * Vrai dès que `jumps` est un nombre : le document ne publie aucun compte de sauts lu, tout compte vient de la
   * dérivation (`jumpDerived`) et se lit « estimation ». Faux quand `jumps` est `null` (rien à estimer).
   */
  jumpsEstimated: boolean
}

/** Un intervalle de sprint fusionné, daté à son début (`t`), avec les images de sprint des intervalles qui le précèdent. */
interface SprintRun {
  t: number
  end: number
  before: number
}

/** Un saut daté à son premier instant, consultable par `lastIndexAt`. */
interface Jump {
  t: number
}

/**
 * Ce que `readMovementSeries` range pour `playerMovementStatsAt`. Les champs ne se lisent pas ailleurs : ils sont
 * l'état interne d'une lecture, exposé pour le typage seulement. `null` = grandeur non établie pour ce document.
 */
export interface PlayerMovementSeries {
  readonly doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>
  /** Intervalles de sprint du joueur, fusionnés (un joueur ne sprinte pas deux fois à la fois) et triés. */
  readonly sprint: readonly SprintRun[] | null
  /** Sauts du joueur, triés. */
  readonly jumps: readonly Jump[] | null
}

/** Les séries de tous les joueurs, sous leur clé de roster et sous le xuid de leur ligne de feuille. */
export type MovementSeriesIndex = ReadonlyMap<string, PlayerMovementSeries>

/** Ce que la lecture a besoin du document. */
export type MovementSeriesSource = Pick<ReplayDocumentReady, 'stances' | 'coverage' | 'frameIntervalMs'>

/** Aucune des deux grandeurs n'est établie : la réponse d'un joueur sans série, partagée (elle est immuable). */
const NOT_ESTABLISHED: PlayerMovementStats = { sprintMs: null, jumps: null, jumpsEstimated: false }

/** Ce que la couverture des états de mouvement établit pour tout le document. */
function establishedOf(doc: MovementSeriesSource): { sprint: boolean; jumps: boolean } {
  const cov = doc.coverage?.stances
  const sprint = cov?.scanned === true && cov.absent !== true
  return { sprint, jumps: sprint && (cov?.jumpEpisodes ?? 0) > 0 }
}

/** Fusionne les intervalles de sprint d'un joueur (déjà triés par début) et cumule les images qui précèdent chacun. */
function sprintRuns(intervals: readonly { t0: number; t1: number }[]): SprintRun[] {
  const runs: SprintRun[] = []
  for (const iv of intervals) {
    if (iv.t1 <= iv.t0) continue
    const last = runs[runs.length - 1]
    if (last && iv.t0 <= last.end) last.end = Math.max(last.end, iv.t1)
    else runs.push({ t: iv.t0, end: iv.t1, before: 0 })
  }
  let before = 0
  for (const run of runs) {
    run.before = before
    before += run.end - run.t
  }
  return runs
}

/** Les intervalles d'un genre, rangés par joueur (clé = xuid de roster) et triés par début. */
function intervalsByOwner(
  doc: MovementSeriesSource,
  kind: string,
  ownerAt: (slot: number, frame: number) => ReplayPlayer | null,
): Map<string, { t0: number; t1: number }[]> {
  const byOwner = new Map<string, { t0: number; t1: number }[]>()
  for (const s of doc.stances) {
    if (s.kind !== kind) continue
    const owner = ownerAt(s.slot, s.t0)
    if (!owner) continue
    const own = byOwner.get(owner.xuid)
    if (own) own.push({ t0: s.t0, t1: s.t1 })
    else byOwner.set(owner.xuid, [{ t0: s.t0, t1: s.t1 }])
  }
  for (const own of byOwner.values()) own.sort((a, b) => a.t0 - b.t0)
  return byOwner
}

/**
 * readMovementSeries lit, une fois par document, les sprints et les sauts de chaque joueur qui a au moins une vie.
 * `players` est le roster du film (`buildPlayers`). Un intervalle dont le slot n'a pas de propriétaire à son
 * premier instant ne revient à personne : il n'est jamais deviné.
 */
export function readMovementSeries(doc: MovementSeriesSource, players: readonly ReplayPlayer[]): MovementSeriesIndex {
  const established = establishedOf(doc)
  const ownerAt = buildSlotOwnership(players).ownerAtFrame
  const sprints = established.sprint ? intervalsByOwner(doc, STANCE_KIND_SPRINT, ownerAt) : null
  const jumps = established.jumps ? intervalsByOwner(doc, STANCE_KIND_JUMP_DERIVED, ownerAt) : null
  const index = new Map<string, PlayerMovementSeries>()
  for (const player of players) {
    if (player.lives.length === 0) continue
    const series: PlayerMovementSeries = {
      doc,
      sprint: sprints ? sprintRuns(sprints.get(player.xuid) ?? []) : null,
      jumps: jumps ? (jumps.get(player.xuid) ?? []).map((j) => ({ t: j.t0 })) : null,
    }
    for (const key of identityKeys(player)) index.set(key, series)
  }
  return index
}

/**
 * movementSeriesOf — la série d'un joueur : sa clé de roster, puis le xuid de sa ligne de feuille. `null` = le
 * joueur n'a aucune vie dans le film : rien n'est établi pour lui.
 */
export function movementSeriesOf(index: MovementSeriesIndex, player: ReplayPlayer): PlayerMovementSeries | null {
  for (const key of identityKeys(player)) {
    const series = index.get(key)
    if (series) return series
  }
  return null
}

/** Les images de sprint du joueur jusqu'à `frame` (compris) : les intervalles terminés, plus celui en cours. */
function sprintFramesAt(runs: readonly SprintRun[], frame: number): number {
  const i = lastIndexAt(runs, frame)
  if (i < 0) return 0
  const run = runs[i]
  return run.before + Math.max(0, Math.min(run.end, frame) - run.t)
}

/**
 * playerMovementStatsAt rend le sprint et les sauts du joueur du début du match jusqu'à `frame` (compris). Une série
 * `null` (joueur sans vie) ne permet d'établir aucune des deux grandeurs. Les deux comptes ne reculent jamais d'une
 * image à l'autre ; un joueur mort ne sprinte plus (les intervalles s'arrêtent avec la vie), le cumul se fige.
 */
export function playerMovementStatsAt(series: PlayerMovementSeries | null, frame: number): PlayerMovementStats {
  if (!series) return NOT_ESTABLISHED
  const jumps = series.jumps ? lastIndexAt(series.jumps, frame) + 1 : null
  return {
    sprintMs: series.sprint ? frameToMs(sprintFramesAt(series.sprint, frame), series.doc) : null,
    jumps,
    jumpsEstimated: jumps !== null,
  }
}
