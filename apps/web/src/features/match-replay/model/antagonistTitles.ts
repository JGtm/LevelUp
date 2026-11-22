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
 * LA PULSATION SUIT L'HORLOGE MURALE, pas l'axe du rejeu (1,4 s réelles, quelle que soit la vitesse de
 * lecture, et jusqu'à son terme si la lecture se met en pause : le même parti que la mise en valeur d'une
 * élimination). Elle ne démarre que sur un PAS DE LECTURE : `advance(de, vers)`, que la boucle de lecture
 * appelle à chaque pas, annonce l'intervalle qu'elle vient de parcourir ; un changement de titre (genre
 * `prise`) dont l'instant tombe dans cet intervalle, en avançant, lance la pulsation. Un saut du curseur
 * (frise, glissé, touche, pas d'image) n'appelle pas `advance` : il montre l'état de repos de l'instant atteint,
 * jamais une pulsation. Un retour en arrière non plus.
 *   - le nouveau tenant (`gain`) : le titre grossit puis revient, avec ses anneaux ;
 *   - l'ancien tenant (`loss`) : son titre, à son compte d'alors, s'efface pendant ce temps.
 * Plusieurs prises d'un même rôle franchies dans un seul pas ne font qu'une pulsation : le tenant final gagne, celui
 * d'avant le pas s'efface, ceux qui n'ont tenu que dans le pas ne portent rien. Une nouvelle prise pendant une
 * pulsation la relance. Une pulsation ne s'affiche que si le tenant de l'image lue est celui qu'elle désigne : après
 * un saut vers un instant où le titre est ailleurs, c'est le repos.
 * Une égalité ne change pas de main : elle ne pulse pas.
 *
 * `reducedMotion` : aucune pulsation, l'état final tout de suite.
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
  /** Durée de la pulsation, en millisecondes d'HORLOGE MURALE. */
  holdMs: number
  reducedMotion: boolean
  /** L'horloge murale, en millisecondes (`performance.now`). */
  now: () => number
}

export type SlotTitlesOf = (slot: number, frame: number) => readonly SlotTitle[]

export interface SlotTitlesReader {
  /** Les titres que porte un slot à une image, la pulsation en cours à l'horloge murale comprise. */
  titlesOfSlot: SlotTitlesOf
  /** Un pas de lecture a parcouru les images `de` à `vers` : un changement de titre franchi en avançant lance sa pulsation. */
  advance: (from: number, to: number) => void
  /** Une pulsation est en cours à l'horloge murale. */
  pulsing: () => boolean
  /** L'instant mural de la dernière lecture de titres, c'est-à-dire du dernier tracé : `-Infinity` avant la première. */
  lastReadAt: () => number
}

/** Une pulsation en cours pour un rôle : le tenant qu'elle désigne, celui qui s'efface, et son départ mural. */
interface Pulse {
  holder: string
  previousHolder: string | null
  previousCount: number
  startedAt: number
}

/** Aucun titre : partagé, jamais modifié. */
const NONE: readonly SlotTitle[] = []

/** Les prises d'un rôle, dans l'ordre du temps (l'index les rend ainsi). */
function prisesDe(antagonists: Antagonists, role: AntagonistRole): readonly AntagonistChange[] {
  return antagonists.changes.filter((c) => c.role === role && c.kind === 'prise')
}

/** Le nombre de prises d'instant <= `ms`, par recherche dichotomique. */
function prisesJusqua(prises: readonly AntagonistChange[], ms: number): number {
  let lo = 0
  let hi = prises.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (prises[mid].t <= ms) lo = mid + 1
    else hi = mid
  }
  return lo
}

/**
 * createSlotTitles : la lecture des titres d'un slot à une image, et la tenue des pulsations. L'état des titres
 * de la dernière image lue est gardé : appelée pour tous les slots d'une même image, la fonction ne lit l'index qu'une fois.
 */
export function createSlotTitles(input: SlotTitlesInput): SlotTitlesReader {
  const { antagonists, ownerKeyOfSlot, msOfFrame, holdMs, reducedMotion, now } = input
  const prises: Record<AntagonistRole, readonly AntagonistChange[]> = {
    nemesis: prisesDe(antagonists, 'nemesis'),
    victim: prisesDe(antagonists, 'victim'),
  }
  const pulses: Record<AntagonistRole, Pulse | null> = { nemesis: null, victim: null }
  let cached: { frame: number; at: AntagonistsAt } | null = null
  let readAt = Number.NEGATIVE_INFINITY

  const reading = (frame: number): AntagonistsAt => {
    if (!cached || cached.frame !== frame) cached = { frame, at: antagonists.at(msOfFrame(frame)) }
    return cached.at
  }

  const advance = (from: number, to: number): void => {
    if (reducedMotion || to <= from) return
    const fromMs = msOfFrame(from)
    const toMs = msOfFrame(to)
    for (const role of ROLES) {
      const first = prisesJusqua(prises[role], fromMs)
      const end = prisesJusqua(prises[role], toMs)
      if (end <= first) continue
      const opening = prises[role][first]
      const holder = prises[role][end - 1].holder
      // Le titre est revenu à celui qui le tenait avant le pas : rien n'a changé de main.
      if (opening.previousHolder === holder) continue
      pulses[role] = { holder, previousHolder: opening.previousHolder, previousCount: opening.previousCount, startedAt: now() }
    }
  }

  const progressOf = (pulse: Pulse, at: number): number => (at - pulse.startedAt) / holdMs

  const titlesOfSlot: SlotTitlesOf = (slot, frame) => {
    readAt = now()
    const key = ownerKeyOfSlot(slot, frame)
    if (key === null) return NONE
    const at = reading(frame)
    const out: SlotTitle[] = []
    for (const role of ROLES) {
      const holding = at[role]
      const pulse = pulses[role]
      // Une pulsation ne vaut que pour l'instant où le titre est bien à son nouveau tenant.
      const progress = pulse && holding?.holder === pulse.holder ? progressOf(pulse, readAt) : 1
      const moving = pulse !== null && progress < 1
      if (holding?.holder === key) {
        const title: SlotTitle = { role, count: holding.count }
        if (moving && pulse.holder === key) title.motion = { kind: 'gain', progress }
        out.push(title)
      } else if (moving && pulse.previousHolder === key) {
        out.push({ role, count: pulse.previousCount, motion: { kind: 'loss', progress } })
      }
    }
    return out.length > 0 ? out : NONE
  }

  const pulsing = (): boolean => {
    const at = now()
    return ROLES.some((role) => {
      const pulse = pulses[role]
      return pulse !== null && progressOf(pulse, at) < 1
    })
  }

  return { titlesOfSlot, advance, pulsing, lastReadAt: () => readAt }
}
