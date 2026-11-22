/**
 * antagonistTitles.ts : CE QUE CHAQUE PION PORTE, À CHAQUE IMAGE, des deux titres du point de vue
 * (némésis et souffre-douleur).
 *
 * Les titres sont ceux de l'index des antagonistes du point de vue (`lib/antagonists`, tenu par
 * `createReplayAntagonists`) : ce module ne compte rien et ne rapproche aucune clé. Il répond à une
 * seule question, « qui, parmi les propriétaires de slot à cette image, tient un titre, avec quel
 * compte, et en train de bouger ou non ». La clé d'un joueur est sa clé du film : celle de
 * `holder`, celle de `SlotOwnership`.
 *
 * LA PULSATION EST UNE FONCTION DE L'AXE DU REJEU, pas d'une horloge : elle dure `holdMs` (la
 * rémanence des événements ponctuels, 1,4 s) à partir de l'instant de la prise, comme la croix de mort
 * ou le lancer d'une grenade. Rien à retenir d'une image à l'autre, rien qui dépende de la vitesse de
 * lecture ; un saut dans le rejeu tombe sur l'état juste, pulsation en cours comprise.
 *   - le nouveau tenant (`gain`) : le titre grossit puis revient, avec ses anneaux ;
 *   - l'ancien tenant (`loss`) : son titre, à son compte d'alors, s'efface pendant ce temps.
 * Seuls les changements de genre `prise` pulsent ; une égalité ne change pas de main.
 *
 * `reducedMotion` : l'état final tout de suite, sans mouvement et sans titre qui s'efface.
 *
 * L'index est lu UNE fois par image (`at`), quel que soit le nombre de slots dessinés.
 */
import type { AntagonistChange, AntagonistRole, Antagonists, AntagonistsAt } from '@/lib/antagonists/antagonists'

/** L'ordre des titres sur un pion qui les tient tous les deux. */
const ROLES: readonly AntagonistRole[] = ['nemesis', 'victim']

/** Où en est le mouvement d'un titre : `progress` va de 0 (début) à 1 (fin, exclue : au repos). */
export interface TitleMotion {
  kind: 'gain' | 'loss'
  progress: number
}

/** Un titre porté par un slot à une image. Sans `motion` : au repos. */
export interface SlotTitle {
  role: AntagonistRole
  count: number
  motion?: TitleMotion
}

export interface SlotTitlesInput {
  /** L'index du point de vue. */
  antagonists: Antagonists
  /** La clé du film du propriétaire du slot à l'image ; `null` = slot libre. */
  ownerKeyOfSlot: (slot: number, frame: number) => string | null
  /** L'instant de l'axe du rejeu d'une image, en millisecondes (`frameToMs`). */
  msOfFrame: (frame: number) => number
  /** Durée de la pulsation, en millisecondes du rejeu. */
  holdMs: number
  reducedMotion: boolean
}

export type SlotTitlesOf = (slot: number, frame: number) => readonly SlotTitle[]

/** Aucun titre : partagé, jamais modifié. */
const NONE: readonly SlotTitle[] = []

/** Une prise de titre encore en cours de pulsation à l'image lue. */
interface Pulse {
  change: AntagonistChange
  progress: number
}

/** Ce qui est vrai de toute l'image : l'état des titres et les prises qui pulsent. */
interface FrameState {
  frame: number
  at: AntagonistsAt
  pulse: Record<AntagonistRole, Pulse | null>
}

/** Les prises d'un rôle, dans l'ordre du temps (l'index les rend ainsi). */
function prisesDe(antagonists: Antagonists, role: AntagonistRole): readonly AntagonistChange[] {
  return antagonists.changes.filter((c) => c.role === role && c.kind === 'prise')
}

/** La dernière prise d'instant <= `ms`, par recherche dichotomique ; `null` avant la première. */
function priseAvant(prises: readonly AntagonistChange[], ms: number): AntagonistChange | null {
  let lo = 0
  let hi = prises.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (prises[mid].t <= ms) lo = mid + 1
    else hi = mid
  }
  return lo === 0 ? null : prises[lo - 1]
}

/**
 * createSlotTitles : la lecture des titres d'un slot à une image. La fonction rendue garde l'état de
 * la dernière image lue : appelée pour tous les slots d'une même image, elle ne lit l'index qu'une fois.
 */
export function createSlotTitles(input: SlotTitlesInput): SlotTitlesOf {
  const { antagonists, ownerKeyOfSlot, msOfFrame, holdMs, reducedMotion } = input
  const prises: Record<AntagonistRole, readonly AntagonistChange[]> = {
    nemesis: prisesDe(antagonists, 'nemesis'),
    victim: prisesDe(antagonists, 'victim'),
  }
  let state: FrameState | null = null

  const read = (frame: number): FrameState => {
    if (state && state.frame === frame) return state
    const ms = msOfFrame(frame)
    const pulseOf = (role: AntagonistRole): Pulse | null => {
      if (reducedMotion) return null
      const change = priseAvant(prises[role], ms)
      if (!change) return null
      const progress = (ms - change.t) / holdMs
      return progress < 1 ? { change, progress } : null
    }
    state = { frame, at: antagonists.at(ms), pulse: { nemesis: pulseOf('nemesis'), victim: pulseOf('victim') } }
    return state
  }

  return (slot, frame) => {
    const key = ownerKeyOfSlot(slot, frame)
    if (key === null) return NONE
    const { at, pulse } = read(frame)
    const out: SlotTitle[] = []
    for (const role of ROLES) {
      const reading = at[role]
      const moving = pulse[role]
      if (reading?.holder === key) {
        const title: SlotTitle = { role, count: reading.count }
        if (moving && moving.change.holder === key) title.motion = { kind: 'gain', progress: moving.progress }
        out.push(title)
      } else if (moving && moving.change.previousHolder === key) {
        out.push({ role, count: moving.change.previousCount, motion: { kind: 'loss', progress: moving.progress } })
      }
    }
    return out.length > 0 ? out : NONE
  }
}
