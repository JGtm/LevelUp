/**
 * departureLogic.ts — UN DÉPART N'EST PAS UNE MORT.
 *
 * Un joueur qui quitte la partie laisse une fin de vie qu'aucun frag ne revendique : le jeu supprime
 * son corps (`endReason === 'deleted'`) sans lâcher de commande ni compte à rebours de réapparition,
 * puis son entrée du tableau (`roster[].presence`) se ferme et, souvent, un bot reprend sa place. Lue
 * comme une fin de manche ou comme une mort, cette fin de vie fait écrire « mort » pour quelqu'un qui
 * est parti.
 *
 * LA RÈGLE. Une fin de vie est un DÉPART quand les quatre conditions tiennent :
 *  1. c'est la DERNIÈRE vie du joueur dans le document (un corps supprimé suivi d'une autre vie est une
 *     fin de manche) ;
 *  2. son `endReason` vaut `deleted` (une dernière vie sans raison publiée ne se tranche pas) ;
 *  3. aucun compte à rebours de réapparition du joueur ne commence dans la tolérance de `lifeEndProof` :
 *     un compte à rebours est la signature d'une mort ;
 *  4. la présence du joueur au tableau se ferme avant la fin du document, hors de la marge de fin de
 *     partie (`SURVIVOR_GRACE_MS`) : un joueur présent jusqu'au bout est un survivant, pas un partant.
 *
 * Un document qui ne publie aucune présence n'a aucun départ : la fin de vie garde le comportement des
 * morts neutres. Une vraie mort suivie d'un départ (lâcher de commande, compte à rebours) n'est pas un
 * départ : sa ligne « mort » reste.
 *
 * « A QUITTÉ LA PARTIE » DURE JUSQU'À LA FIN DU DOCUMENT, pour tout partant :
 *  - le départ par corps supprimé (règle ci-dessus) : parti dès l'image qui suit la fin de sa dernière vie ;
 *  - tout autre joueur humain dont la présence se ferme pour de bon (aucune période ensuite, aucune vie qui
 *    commence après la fermeture, fermeture hors de la marge de fin de partie) : mort comme avant jusqu'à la
 *    fermeture de sa présence — sa ligne « mort » et sa croix ne changent pas —, parti dès l'image qui suit.
 * Un joueur qui revient n'est jamais parti entre-temps ; un bot garde le comportement actuel.
 *
 * Fonction pure, sans React ; la table se construit une fois par document.
 */
import { countdownStartsByXuid, lifeEndProof } from './lifeEndProof'
import { msToFrames, trackWindow } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'

/**
 * Marge sous l'horizon du document au-delà de laquelle une fin de vie est une VRAIE mort et non un
 * survivant de fin de partie (les vies des survivants se ferment dans les 3 dernières secondes), et
 * au-delà de laquelle une présence fermée est un départ et non la fin du match.
 */
export const SURVIVOR_GRACE_MS = 3_000

/** La valeur de `endReason` que publie le décodeur quand le corps est supprimé en cours de commande. */
export const END_REASON_DELETED = 'deleted'

/** Ce que le document dit d'UN joueur parti. */
export interface Departure {
  /** La dernière vie du joueur. */
  life: ReplayTrackReady
  /**
   * Vrai quand la fin de cette vie EST le départ (corps supprimé, sans mort) : le fil n'en fait pas une ligne
   * « mort ». Faux quand le joueur est mort avant de partir : sa mort reste une mort.
   */
  deletedBody: boolean
  /** Le joueur est parti à partir de l'image qui suit celle-ci, jusqu'à la fin du document. */
  fromFrame: number
}

type DepartureDoc = Pick<
  ReplayDocumentReady,
  'tracks' | 'roster' | 'respawnCountdowns' | 'frameCount' | 'frameIntervalMs'
>

/**
 * documentHorizonFrame — la dernière image du document : celle de la dernière image publiée, sinon la
 * fin de la dernière piste d'un joueur. C'est l'horizon sous lequel `SURVIVOR_GRACE_MS` se mesure.
 */
export function documentHorizonFrame(doc: Pick<ReplayDocumentReady, 'tracks' | 'frameCount'>): number {
  const frameCount = doc.frameCount ?? 0
  if (frameCount > 1) return frameCount - 1
  let maxEnd = 0
  for (const tr of doc.tracks) {
    if (!tr.xuid || tr.points.length === 0) continue
    maxEnd = Math.max(maxEnd, trackWindow(tr).end)
  }
  return maxEnd
}

/** La fermeture de présence de chaque joueur humain, la plus tardive de ses intervalles ; vide sans présence publiée. */
function presenceEnds(roster: ReplayDocumentReady['roster']): Map<string, number> {
  const ends = new Map<string, number>()
  for (const r of roster) {
    if (r.bot || !r.xuid || r.presence.length === 0) continue
    const end = Math.max(...r.presence.map((p) => p.to))
    ends.set(r.xuid, Math.max(ends.get(r.xuid) ?? end, end))
  }
  return ends
}

/** La dernière vie de chaque joueur : la piste de plus grande image de fin, la première à égalité. */
function lastLives(tracks: ReplayDocumentReady['tracks']): Map<string, ReplayTrackReady> {
  const last = new Map<string, ReplayTrackReady>()
  for (const tr of tracks) {
    if (!tr.xuid) continue
    const held = last.get(tr.xuid)
    if (!held || trackWindow(tr).end > trackWindow(held).end) last.set(tr.xuid, tr)
  }
  return last
}

/** Une vie du joueur commence-t-elle après `frame` ? Il est alors revenu : il n'a pas quitté la partie pour de bon. */
function livesAfter(tracks: ReplayDocumentReady['tracks'], xuid: string, frame: number): boolean {
  return tracks.some((tr) => tr.xuid === xuid && trackWindow(tr).start > frame)
}

function buildDepartures(doc: DepartureDoc): Map<string, Departure> {
  const departures = new Map<string, Departure>()
  const ends = presenceEnds(doc.roster)
  if (ends.size === 0) return departures
  const lives = lastLives(doc.tracks)
  const { starts } = countdownStartsByXuid(doc)
  const limit = documentHorizonFrame(doc) - msToFrames(SURVIVOR_GRACE_MS, doc)
  for (const [xuid, presenceEndFrame] of ends) {
    const life = lives.get(xuid)
    if (!life || presenceEndFrame >= limit || livesAfter(doc.tracks, xuid, presenceEndFrame)) continue
    const lifeEndFrame = trackWindow(life).end
    const deletedBody = life.endReason === END_REASON_DELETED && lifeEndProof(life, doc, starts) === null
    departures.set(xuid, {
      life,
      deletedBody,
      fromFrame: deletedBody ? lifeEndFrame : Math.max(presenceEndFrame, lifeEndFrame),
    })
  }
  return departures
}

const cache = new WeakMap<object, ReadonlyMap<string, Departure>>()

/**
 * departuresOf — les joueurs partis d'un document, par xuid. Construite une fois par document (les fiches la
 * relisent à chaque image).
 */
export function departuresOf(doc: DepartureDoc): ReadonlyMap<string, Departure> {
  let built = cache.get(doc)
  if (!built) {
    built = buildDepartures(doc)
    cache.set(doc, built)
  }
  return built
}

/**
 * hasLeftAt — vrai de l'image qui suit le départ du joueur jusqu'à la fin du document. Faux avant, et pour
 * tout joueur qui n'est pas parti pour de bon, bots compris.
 */
export function hasLeftAt(
  departures: ReadonlyMap<string, Departure>,
  xuid: string,
  frame: number,
): boolean {
  const d = departures.get(xuid)
  return d !== undefined && frame > d.fromFrame
}
