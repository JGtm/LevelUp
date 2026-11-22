/// <reference types="node" />
/**
 * temoinsKit : LA SCÈNE ET LES DOUBLES DES TÉMOINS DES ANTAGONISTES.
 *
 * Il FOURNIT, il ne juge pas : aucune assertion. Trois choses sortent d'ici.
 *
 *  - LA SCÈNE d'un film de fixture : document, joueurs, point de vue, allégeance et FIL. Les
 *    fixtures ne portent pas le fil de la base ; il est reconstruit depuis les éliminations que le
 *    film écrit lui-même (`scoreActions`, action `killPersonalScore` : tueur, victime, image), puis
 *    passé par la chaîne de production (`buildFeedEntries`). Chaque élimination tombe donc sur
 *    l'image que le film lui donne, avec ses médailles rattachées.
 *  - LES IMAGES TÉMOINS : trois images d'avant la première élimination entre le point de vue et un
 *    adversaire (aucun antagoniste n'existe encore), trois d'après le moment où le point de vue a
 *    éliminé un adversaire ET été éliminé par un adversaire (les deux titres existent). Dans chaque
 *    tiers de l'intervalle, l'image retenue est celle où le plus de joueurs sont en vie.
 *  - LA TOILE PEINTE : `ReplayCanvas` monté en jsdom sur un contexte enregistreur, la suite des
 *    appels de dessin de la dernière image.
 *
 * LE SEUL POINT QUI DÉPEND DE LA FORME DU PANNEAU DE JOUEUR est `focusDe` : Seul et Masquer d'UN
 * joueur, dans le type que la toile et le filtre des calques reçoivent. Un changement de ce type
 * se répercute ici, et nulle part ailleurs dans les témoins.
 */
import { createHash } from 'node:crypto'

import { vi } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import { buildFilmAllegiance, type FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { isAliveAt } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { renderWithProviders } from '../../../../test/render-utils'
import { buildFeedEntries, type ReplayFeedEntry } from '../../model/killFeedLogic'
import { readMatchMedals } from '../../model/matchMedals'
import { createPlaybackStore } from '../../model/playbackStore'
import type { PlayerFocus } from '../../model/playerPanelLogic'
import { replayClock } from '../../model/replayClock'
import { ReplayCanvas } from '../../ui/ReplayCanvas'
import { makeFrameControl } from '../frameControlStub'
import { goFixtureEntries, loadGoFixture } from '../goFixtures'
import { recordingContext, type CanvasOp } from '../recordingContext'
import { testReplayDoc } from '../testDoc'

/** Les films de fixture lus par les témoins : huit joueurs en deux camps, véhicules, poussées. */
export const FILMS = { base: '000d5950', vehicules: 'a521164d', poussees: 'fb1a1a72' } as const

/** L'action de score par laquelle le film nomme une élimination (tueur `xuid`, victime `target`). */
const ACTION_ELIMINATION = 'killPersonalScore'

/** Une élimination du film : image, tueur, victime (clés du film). */
export interface EliminationDuFilm {
  frame: number
  killer: string
  victim: string
}

/** Tout ce qu'un témoin lit d'un film de fixture. */
export interface Scene {
  film: string
  doc: ReplayDocumentReady
  players: ReplayPlayer[]
  /** Le point de vue : le premier humain du roster placé dans un camp. */
  viewpoint: string
  allegiance: FilmAllegiance
  eliminations: EliminationDuFilm[]
  feed: ReplayFeedEntry[]
}

export function documentDe(film: string): ReplayDocumentReady {
  const entry = goFixtureEntries().find((e) => e.film === film)
  if (!entry) throw new Error(`témoins : film ${film} absent du manifeste des fixtures`)
  return testReplayDoc(loadGoFixture(entry).doc)
}

function eliminationsDe(doc: ReplayDocumentReady): EliminationDuFilm[] {
  return doc.scoreActions
    .filter((a) => a.name === ACTION_ELIMINATION && !!a.xuid && !!a.target)
    .map((a) => ({ frame: a.t, killer: a.xuid as string, victim: a.target as string }))
    .sort((a, b) => a.frame - b.frame)
}

/**
 * L'élimination du film sous la forme que sert la vue match. `tMs` est posé sur l'axe du match par
 * l'horloge du rejeu (sans en-tête de match, son horloge de jeu est cet axe) : le recalage de
 * production la ramène sur son image.
 */
function versKillEvent(e: EliminationDuFilm, matchMsOfFrame: (frame: number) => number, teamOf: (xuid: string) => number | null): KillEvent {
  return {
    tMs: matchMsOfFrame(e.frame),
    xuid: e.killer,
    ally: false,
    teamID: teamOf(e.killer),
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: e.victim,
    victimGamertag: '',
  }
}

export function sceneDe(film: string): Scene {
  const doc = documentDe(film)
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.bot)?.xuid
  if (!viewpoint) throw new Error(`témoins : aucun humain placé dans un camp sur le film ${film}`)
  const allegiance = buildFilmAllegiance(players, viewpoint)
  const eliminations = eliminationsDe(doc)
  const clock = replayClock(doc)
  if (!clock) throw new Error(`témoins : le film ${film} n'établit pas son horloge`)
  const kills = eliminations.map((e) => versKillEvent(e, clock.gameplayMsOfFrame, allegiance.teamOfXuid))
  // Les fixtures ne portent pas les libellés de médaille, et une médaille sans libellé ne se pose pas
  // sur la frise : le nom que le film lui donne en tient lieu.
  const medals = readMatchMedals(doc, 'fr').map((m) => ({ ...m, label: m.label || m.name }))
  const feed = buildFeedEntries(kills, medals, 0, doc)
  return { film, doc, players, viewpoint, allegiance, eliminations, feed }
}

/** Les éliminations entre le point de vue et un adversaire, dans l'ordre du temps. */
export function duelsDuPointDeVue(scene: Scene): EliminationDuFilm[] {
  const { viewpoint, allegiance } = scene
  return scene.eliminations.filter(
    (e) =>
      (e.killer === viewpoint && allegiance.ofXuid(e.victim) === false) ||
      (e.victim === viewpoint && allegiance.ofXuid(e.killer) === false),
  )
}

/** Le premier adversaire à éliminer le point de vue : le joueur que les témoins masquent ou isolent. */
export function premierTueurDuPointDeVue(scene: Scene): ReplayPlayer {
  const duel = duelsDuPointDeVue(scene).find((e) => e.victim === scene.viewpoint)
  const player = duel && scene.players.find((p) => p.xuid === duel.killer)
  if (!player) throw new Error(`témoins : personne n'élimine le point de vue sur le film ${scene.film}`)
  return player
}

function vivants(doc: ReplayDocumentReady, frame: number): number {
  let n = 0
  for (const track of doc.tracks) if (track.xuid && isAliveAt(track, frame)) n++
  return n
}

/** Dans chaque tiers de [from, to[, la première image où le plus de joueurs sont en vie. */
function troisImages(doc: ReplayDocumentReady, from: number, to: number): number[] {
  const out: number[] = []
  const tiers = (to - from) / 3
  for (let i = 0; i < 3; i++) {
    const a = Math.floor(from + i * tiers)
    const b = Math.max(a + 1, Math.floor(from + (i + 1) * tiers))
    let best = a
    let bestCount = -1
    for (let f = a; f < b; f += 10) {
      const count = vivants(doc, f)
      if (count > bestCount) {
        best = f
        bestCount = count
      }
    }
    out.push(best)
  }
  return out
}

/** Les six images du témoin de la toile, et les deux bornes dont elles se déduisent. */
export interface ImagesTemoins {
  /** Première élimination entre le point de vue et un adversaire. */
  premierDuel: number
  /** Première image où le point de vue a éliminé un adversaire ET été éliminé par un adversaire. */
  deuxTitres: number
  sans: number[]
  avec: number[]
}

export function imagesTemoins(scene: Scene): ImagesTemoins {
  const duels = duelsDuPointDeVue(scene)
  const premierFrag = duels.find((e) => e.killer === scene.viewpoint)
  const premiereMort = duels.find((e) => e.victim === scene.viewpoint)
  if (!premierFrag || !premiereMort) {
    throw new Error(`témoins : le point de vue du film ${scene.film} n'a pas les deux titres`)
  }
  const premierDuel = duels[0].frame
  const deuxTitres = Math.max(premierFrag.frame, premiereMort.frame)
  return {
    premierDuel,
    deuxTitres,
    sans: troisImages(scene.doc, 0, premierDuel),
    avec: troisImages(scene.doc, deuxTitres + 1, scene.doc.frameCount),
  }
}

/** Seul ou Masquer d'UN joueur, dans le type que la toile et le filtre des calques reçoivent. */
export function focusDe(player: ReplayPlayer, mode: 'seul' | 'masquer'): PlayerFocus {
  return { player, follow: false, solo: mode === 'seul', hide: mode === 'masquer' }
}

/** Les calques éteints par défaut, que le témoin allume avant de monter la toile. */
export const BASCULES_ETEINTES_PAR_DEFAUT = [
  'replay-show-heatmap',
  'replay-show-unnamed-placements',
  'replay-show-ground-weapons-starting',
  'replay-show-spawns',
  'replay-show-deaths',
  'replay-show-killers',
]

type Enregistrements = Map<HTMLCanvasElement, CanvasOp[]>

/** Un argument de dessin ramené à du JSON lisible ; une toile hors écran devient son empreinte. */
function plain(v: unknown, canvases: Enregistrements): unknown {
  if (typeof v === 'number') return Number.isFinite(v) ? v : String(v)
  if (v === null || typeof v === 'string' || typeof v === 'boolean') return v
  if (v === undefined) return '[undefined]'
  if (v instanceof HTMLCanvasElement) {
    const ops = canvases.get(v) ?? []
    const sha = createHash('sha256').update(JSON.stringify(ops.map((o) => plainOp(o, canvases)))).digest('hex')
    return { canvas: { w: v.width, h: v.height, ops: ops.length, sha } }
  }
  if (Array.isArray(v)) return v.map((x) => plain(x, canvases))
  return `[${(v as object).constructor?.name ?? typeof v}]`
}

function plainOp(o: CanvasOp, canvases: Enregistrements): CanvasOp {
  return { op: o.op, args: o.args.map((a) => plain(a, canvases)) }
}

/** Un contexte enregistreur par toile, avec une mesure de texte déterministe (six unités par caractère). */
function installerEnregistreurs(): Enregistrements {
  const canvases: Enregistrements = new Map()
  const byCanvas = new WeakMap<HTMLCanvasElement, ReturnType<typeof recordingContext>['ctx']>()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (this: HTMLCanvasElement) {
    const known = byCanvas.get(this)
    if (known) return known
    const { ops, ctx } = recordingContext()
    const measuring = new Proxy(ctx, {
      get(target, prop) {
        if (prop === 'measureText') {
          return (text: string) => {
            ops.push({ op: 'measureText', args: [text] })
            return { width: String(text).length * 6 }
          }
        }
        return Reflect.get(target, prop)
      },
      set(target, prop, value) {
        return Reflect.set(target, prop, value)
      },
    })
    canvases.set(this, ops)
    byCanvas.set(this, measuring)
    return measuring
  } as never)
  return canvases
}

/**
 * Les doubles du navigateur, à poser avant chaque montage : aucune boucle d'animation, aucun réseau
 * (la sortie ne dépend pas d'un serveur qui tournerait sur le poste), une taille mesurée et une
 * couleur distincte par jeton (l'empreinte de son nom : jsdom ne charge aucune feuille de style, et
 * une encre vide masquerait un mauvais branchement).
 * `vi.restoreAllMocks()` et `vi.unstubAllGlobals()` les défont.
 */
export function installerDoubles(): void {
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.stubGlobal('fetch', () => Promise.reject(new TypeError('témoins : aucun réseau')))
  const real = window.getComputedStyle.bind(window)
  vi.spyOn(window, 'getComputedStyle').mockImplementation(((el: Element, pseudo?: string | null) => {
    const style = real(el, pseudo)
    return new Proxy(style, {
      get(target, prop) {
        if (prop === 'getPropertyValue') {
          return (name: string) => {
            if (!name.startsWith('--')) return target.getPropertyValue(name)
            return `#${createHash('sha256').update(name).digest().subarray(0, 3).toString('hex')}`
          }
        }
        const v = Reflect.get(target, prop)
        return typeof v === 'function' ? v.bind(target) : v
      },
    })
  }) as never)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
}

/** La dernière image peinte : les appels depuis le dernier `setTransform` suivi de `clearRect`. */
function derniereImage(ops: CanvasOp[]): CanvasOp[] {
  for (let i = ops.length - 2; i >= 0; i--) {
    if (ops[i].op === 'setTransform' && ops[i + 1].op === 'clearRect') return ops.slice(i)
  }
  return []
}

/**
 * peindre monte la toile sur la scène, ouverte à `frame`, et rend la suite des appels de dessin de
 * la dernière image. La toile reçoit le fil, le point de vue et l'allégeance : tout ce dont un
 * calque a besoin pour connaître les éliminations à l'instant.
 */
export function peindre(scene: Scene, frame: number, focus: PlayerFocus | null = null): CanvasOp[] {
  const canvases = installerEnregistreurs()
  const view = renderWithProviders(
    <ReplayCanvas
      doc={scene.doc}
      locale="fr"
      playWindow={null}
      playbackStore={createPlaybackStore()}
      openAtFrame={frame}
      allegiance={scene.allegiance}
      players={scene.players}
      playerFocus={focus}
      viewpoint={scene.viewpoint}
      feedEntries={scene.feed}
      fullscreen={{ active: false, disabled: false, toggle: () => {} }}
      frameControl={makeFrameControl()}
      transport={() => null}
      marks={new Map([[scene.viewpoint, 'me' as const]])}
    />,
  )
  const main = view.container.querySelector('canvas')
  if (!main) throw new Error('témoins : la toile ne s’est pas montée')
  return derniereImage(canvases.get(main) ?? []).map((o) => plainOp(o, canvases))
}

/** Un appel par ligne, précédé d'une ligne d'en-tête : la comparaison désigne le premier écart. */
export function serialiserToile(entete: Record<string, unknown>, ops: readonly CanvasOp[]): string {
  return `${JSON.stringify({ ...entete, appels: ops.length })}\n${ops.map((o) => JSON.stringify(o)).join('\n')}\n`
}
