/**
 * hillSequence.ts — LA SUITE DES COLLINES JOUÉES du Roi de la colline, à l'image lue, et la prochaine quand elle est
 * connue. Sans React.
 *
 * LA SOURCE. Une colline est une zone dont `zoneStates[].spans` portent `active` (`isHillState`) ; chaque intervalle
 * actif dit où la colline en est de son activation. Les intervalles qui se suivent d'une même zone (le suivant commence
 * au plus tard à l'image qui suit la fin du précédent) sont UNE activation ; un trou en fait deux. Les activations de
 * toutes les zones, rangées par début, sont la suite des collines jouées.
 *
 * LA LETTRE EST CELLE QUE LA CARTE ÉCRIT (`zoneLetterOf(letterRank)`) : la même suite, deux écrans. Une zone sans
 * lettre se nomme par son rang parmi les collines du match (1, 2, ...), jamais par une lettre inventée.
 *
 * À L'IMAGE LUE. Seules les collines COMMENCÉES à l'image comptent : on ne montre pas, en lisant le début du match, les
 * collines qui suivent. `currentIndex` désigne la colline active à l'image, `-1` entre deux collines.
 *
 * LA PROCHAINE N'EST PAS UNE DONNÉE DU FILM. L'ordre de passage est une donnée de la variante de mode : le document ne
 * le porte pas, il ne montre que ce qui a été joué. Elle ne se dit donc que quand la suite DÉJÀ JOUÉE montre un cycle
 * complet répété en entier (la période minimale `p`, au moins `2p` collines jouées) ; sinon rien, jamais un ordre
 * deviné. Elle ne se dit pas non plus quand aucune autre activation ne suit au document : la dernière colline du match
 * n'a pas de suivante, même si le cycle la prédirait.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { isHillState, zoneLetterOf } from '../layers/zoneStatesLayer'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Ce que la lecture demande du document. */
export type HillSequenceDocument = Pick<ReplayDocumentReady, 'zoneStates'>

/** Une activation de colline : la zone, son nom court, et ses bornes (incluses) sur l'axe d'images du document. */
export interface HillActivation {
  zoneRef: number
  label: string
  t0: number
  t1: number
}

/** La suite lue à une image. */
export interface HillSequenceReading {
  /** Les collines commencées à l'image, dans l'ordre. */
  played: readonly HillActivation[]
  /** Indice, dans `played`, de la colline active à l'image ; `-1` entre deux collines. */
  currentIndex: number
  /** Le nom de la prochaine colline, quand le cycle est établi et qu'une autre activation suit ; sinon `null`. */
  next: string | null
}

/** Les activations de toutes les collines du document, dans l'ordre du temps. */
export function hillActivationsOf(doc: HillSequenceDocument): HillActivation[] {
  const hills = doc.zoneStates.filter(isHillState)
  const rankOf = new Map([...hills].sort((a, b) => a.zoneRef - b.zoneRef).map((st, i) => [st.zoneRef, String(i + 1)]))
  const out: HillActivation[] = []
  for (const st of hills) {
    const label = zoneLetterOf(st.letterRank) ?? rankOf.get(st.zoneRef) ?? String(st.zoneRef)
    let run: HillActivation | null = null
    for (const span of st.spans) {
      if (!span.active) continue
      if (run && span.t0 <= run.t1 + 1) {
        run.t1 = Math.max(run.t1, span.t1)
        continue
      }
      run = { zoneRef: st.zoneRef, label, t0: span.t0, t1: span.t1 }
      out.push(run)
    }
  }
  return out.sort((a, b) => a.t0 - b.t0 || a.zoneRef - b.zoneRef)
}

/** La période minimale `p` telle que la suite répète en entier un cycle de `p` noms (au moins `2p` noms), ou `null`. */
function establishedPeriod(labels: readonly string[]): number | null {
  for (let p = 1; 2 * p <= labels.length; p++) {
    if (labels.every((label, i) => i < p || label === labels[i - p])) return p
  }
  return null
}

/** hillSequenceAt lit la suite des collines à `frame` (comprise) ; `null` quand aucune colline n'a encore commencé. */
export function hillSequenceAt(activations: readonly HillActivation[], frame: number): HillSequenceReading | null {
  const playedCount = activations.filter((a) => a.t0 <= frame).length
  if (playedCount === 0) return null
  const played = activations.slice(0, playedCount)
  const last = played[played.length - 1]
  const currentIndex = frame <= last.t1 ? played.length - 1 : -1
  const labels = played.map((a) => a.label)
  const period = establishedPeriod(labels)
  const anotherFollows = playedCount < activations.length
  const next = period !== null && anotherFollows ? labels[labels.length - period] : null
  return { played, currentIndex, next }
}
