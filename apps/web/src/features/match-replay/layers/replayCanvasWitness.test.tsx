/// <reference types="node" />
/**
 * replayCanvasWitness.test.tsx : LE TEMOIN DU MOTEUR DE LA TOILE 2D.
 *
 * Il monte `ReplayCanvas` en jsdom, avec un contexte 2D enregistreur (`test/recordingContext.ts`),
 * et compare la SUITE COMPLETE des appels de dessin de la derniere image peinte a une fixation
 * versionnee (`__fixtures__/*.json`) : trois documents (un match a 8 joueurs, une grande equipe
 * avec vehicules et socles d'arme, un match a objectifs et etats de zone) a trois instants
 * chacun (debut, milieu, fin). Aucun pixel : c'est la geometrie emise, dans l'ordre, qui est
 * figee. Toute difference d'un seul appel fait echouer le test.
 *
 * LES FIXATIONS NE SE REGENERENT JAMAIS EN PASSANT : un changement de rendu voulu les reecrit
 * par `UPDATE_REPLAY_WITNESS=1 npx vitest run replayCanvasWitness`, et le diff git dit
 * exactement ce qui a bouge. Une fixation absente fait echouer le test.
 */
import { createHash } from 'node:crypto'
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'

import { renderWithProviders } from '../../../test/render-utils'
import { isAliveAt } from '../../../lib/replay/replayLogic'
import { buildPlayers, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { COTES_PION_CERCLE } from '../model/cotesFenetreFiche'
import { createPlaybackStore } from '../model/playbackStore'
import type { PlayerFocus } from '../model/playerPanelLogic'
import { makeFrameControl } from '../test/frameControlStub'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { recordingContext, type CanvasOp } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { ReplayCanvas } from '../ui/ReplayCanvas'

const FIXTURES_DIR = join(__dirname, '__fixtures__')
const UPDATE = process.env.UPDATE_REPLAY_WITNESS === '1'

type Variant = 'base' | 'vehicules' | 'objectifs'
type Instant = 'debut' | 'milieu' | 'fin'

/** Les films de test : par numero de film du manifeste des fixtures Go. */
const FILM_OF: Record<Variant, string> = {
  base: '000d5950',
  vehicules: 'a521164d',
  objectifs: 'fb1a1a72',
}

/** Les objectifs statiques et les etats de zone injectes dans le troisieme document. */
const OBJECTIFS = {
  zones: [
    { role: 'koth_hill', team: -1, x: -10, y: 0, z: 58, family: 'cylinder', radius: 4, fwdX: 1, fwdY: 0 },
    { role: 'strongholds_zone', team: -1, x: 12, y: -3, z: 58, family: 'box', halfX: 3, halfY: 2, fwdX: 0, fwdY: 1 },
  ],
  markers: [
    { role: 'flag_spawn', team: 0, x: -24, y: 0, z: 58 },
    { role: 'flag_delivery', team: 1, x: 22, y: 5, z: 58 },
  ],
}

function zoneStatesOf(frameCount: number) {
  const tiers = Math.floor(frameCount / 3)
  return [
    {
      zoneRef: 0,
      spans: [
        { t0: 0, t1: tiers, owner: null, active: true },
        { t0: tiers + 1, t1: 2 * tiers, owner: 0, active: true },
        { t0: 2 * tiers + 1, t1: frameCount - 1, owner: 1, active: true },
      ],
      gauge: [{ t: tiers, v: 0.5 }, { t: 2 * tiers, v: 0.9 }],
    },
    {
      zoneRef: 1,
      spans: [{ t0: 0, t1: frameCount - 1, owner: 1, active: true }],
      gauge: [],
    },
  ]
}

function documentOf(variant: Variant): ReplayDocumentReady {
  const entry = goFixtureEntries().find((e) => e.film === FILM_OF[variant])
  if (!entry) throw new Error(`temoin : film ${FILM_OF[variant]} absent du manifeste`)
  const raw: ReplayDocument = loadGoFixture(entry).doc
  if (variant !== 'objectifs') return testReplayDoc(raw)
  return testReplayDoc({
    ...raw,
    mapObjectives: OBJECTIFS,
    zoneStates: zoneStatesOf(raw.frameCount),
    coverage: { ...raw.coverage, zones: { ...raw.coverage?.zones, catalog: OBJECTIFS.zones.length } },
  } as ReplayDocument)
}

/** Un nom de classe pour tout ce qui n'est pas un scalaire JSON : l'enregistrement reste lisible. */
function plain(v: unknown, canvases: Map<HTMLCanvasElement, CanvasOp[]>): unknown {
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

function plainOp(o: CanvasOp, canvases: Map<HTMLCanvasElement, CanvasOp[]>): CanvasOp {
  return { op: o.op, args: o.args.map((a) => plain(a, canvases)) }
}

/** Les contextes rendus a chaque canvas : un enregistreur par toile, mesure de texte deterministe. */
function installRecorders(): Map<HTMLCanvasElement, CanvasOp[]> {
  const canvases = new Map<HTMLCanvasElement, CanvasOp[]>()
  const byCanvas = new WeakMap<HTMLCanvasElement, CanvasRenderingContext2D>()
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

/** La derniere image peinte : les appels depuis le dernier `setTransform` suivi de `clearRect`. */
function lastFrameOps(ops: CanvasOp[]): CanvasOp[] {
  for (let i = ops.length - 2; i >= 0; i--) {
    if (ops[i].op === 'setTransform' && ops[i + 1].op === 'clearRect') return ops.slice(i)
  }
  return []
}

/** `focus` : la fiche ouverte (le joueur est cherché dans le même roster que celui de la toile) ; absent = aucune. */
function render(doc: ReplayDocumentReady, frame: number, focus?: (players: ReplayPlayer[]) => PlayerFocus | null): CanvasOp[] {
  const canvases = installRecorders()
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.xuid.startsWith('bot:'))?.xuid ?? null
  const allegiance = buildFilmAllegiance(players, viewpoint)
  const fullscreen = { active: false, disabled: false, toggle: () => {} }
  const frameControl = makeFrameControl()
  const view = renderWithProviders(
    <ReplayCanvas
      doc={doc}
      locale="fr"
      playWindow={null}
      playbackStore={createPlaybackStore()}
      openAtFrame={frame}
      allegiance={allegiance}
      players={players}
      playerFocus={focus ? focus(players) : null}
      viewpoint={viewpoint}
      fullscreen={fullscreen}
      frameControl={frameControl}
      transport={() => null}
      marks={viewpoint ? new Map([[viewpoint, 'me' as const]]) : undefined}
    />,
  )
  const main = view.container.querySelector('canvas')
  if (!main) throw new Error('temoin : la toile ne s est pas montee')
  const ops = canvases.get(main) ?? []
  return lastFrameOps(ops).map((o) => plainOp(o, canvases))
}

/** Les calques eteints par defaut : le temoin les allume, les autres le sont deja. */
const BASCULES_ETEINTES_PAR_DEFAUT = [
  'replay-show-heatmap',
  'replay-show-unnamed-placements',
  'replay-show-ground-weapons-starting',
  'replay-show-spawns',
  'replay-show-deaths',
  'replay-show-killers',
]

function frameOf(doc: ReplayDocumentReady, instant: Instant): number {
  if (instant === 'debut') return Math.floor(doc.frameCount * 0.12)
  if (instant === 'milieu') return Math.floor(doc.frameCount / 2)
  return doc.frameCount - 1
}

/** Un appel par ligne : un diff git lisible et une comparaison qui designe le premier ecart. */
function serialize(variant: Variant, instant: Instant, frame: number, ops: CanvasOp[]): string {
  const head = JSON.stringify({ variant, instant, frame, opCount: ops.length })
  return `${head}\n${ops.map((o) => JSON.stringify(o)).join('\n')}\n`
}

function fixturePath(variant: Variant, instant: Instant): string {
  return join(FIXTURES_DIR, `rejeu-${variant}-${instant}.json`)
}

beforeAll(() => {
  // Aucune boucle d'animation : le temoin peint l'image demandee, une fois, et rien d'autre.
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterAll(() => {
  vi.unstubAllGlobals()
})

/**
 * Chaque variable CSS de jeton se resout en une couleur DISTINCTE et deterministe (empreinte du
 * nom) : jsdom ne charge aucune feuille de style, et sans cela toutes les encres vaudraient la
 * chaine vide, ce qui rendrait invisible une encre branchee sur le mauvais calque.
 */
function stubTokenColors(): void {
  const real = window.getComputedStyle.bind(window)
  vi.spyOn(window, 'getComputedStyle').mockImplementation(((el: Element, pseudo?: string | null) => {
    const style = real(el, pseudo)
    return new Proxy(style, {
      get(target, prop) {
        if (prop === 'getPropertyValue') {
          return (name: string) => {
            if (!name.startsWith('--')) return target.getPropertyValue(name)
            const h = createHash('sha256').update(name).digest()
            return `#${h.subarray(0, 3).toString('hex')}`
          }
        }
        const v = Reflect.get(target, prop)
        return typeof v === 'function' ? v.bind(target) : v
      },
    })
  }) as never)
}

beforeEach(() => {
  localStorage.clear()
  for (const key of BASCULES_ETEINTES_PAR_DEFAUT) localStorage.setItem(key, 'true')
  stubTokenColors()
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const VARIANTS: Variant[] = ['base', 'vehicules', 'objectifs']
const INSTANTS: Instant[] = ['debut', 'milieu', 'fin']

describe('temoin du moteur de la toile 2D', () => {
  for (const variant of VARIANTS) {
    describe(variant, () => {
      let doc: ReplayDocumentReady
      beforeAll(() => {
        doc = documentOf(variant)
      })
      it.each(INSTANTS)('%s : la suite des appels de dessin est celle de la fixation', (instant) => {
        const frame = frameOf(doc, instant)
        const actual = serialize(variant, instant, frame, render(doc, frame))
        const path = fixturePath(variant, instant)
        if (UPDATE) {
          mkdirSync(FIXTURES_DIR, { recursive: true })
          writeFileSync(path, actual, 'utf8')
          return
        }
        expect(existsSync(path), `fixation absente : ${path}`).toBe(true)
        const expected = readFileSync(path, 'utf8')
        if (actual === expected) return
        const a = actual.split('\n')
        const e = expected.split('\n')
        const i = a.findIndex((line, k) => line !== e[k])
        expect.fail(
          `${variant}/${instant} : premier ecart a la ligne ${i} (attendu ${e.length - 2} appels, obtenu ${a.length - 2})\n` +
            `attendu : ${e[i]}\nobtenu  : ${a[i]}`,
        )
      })
    })
  }
})

/** La couleur que `stubTokenColors` donne à une variable CSS : l'empreinte de son nom. */
function inkOf(name: string): string {
  return `#${createHash('sha256').update(name).digest().subarray(0, 3).toString('hex')}`
}

const FICHE_OUVERTE = { follow: false, solo: false, hide: false }

describe('temoin du moteur de la toile 2D : fiche ouverte (le pion cerclé)', () => {
  let doc: ReplayDocumentReady
  let frame: number
  let sans: CanvasOp[]
  beforeAll(() => {
    doc = documentOf('base')
    frame = frameOf(doc, 'milieu')
  })
  beforeEach(() => {
    sans = render(doc, frame)
  })

  const vivant = (p: ReplayPlayer) => p.lives.some((l) => isAliveAt(l, frame))
  const ouverte = (choix: (players: ReplayPlayer[]) => ReplayPlayer | undefined, over: Partial<typeof FICHE_OUVERTE> = {}) =>
    (players: ReplayPlayer[]): PlayerFocus | null => {
      const player = choix(players)
      if (!player) throw new Error('temoin : aucun joueur ne convient a ce cas')
      return { player, ...FICHE_OUVERTE, ...over }
    }

  it('joueur vivant : un anneau clair cerné d’un liseré à l’encre du fond, centré sur son pion, et rien d’autre ne change', () => {
    const avec = render(doc, frame, ouverte((ps) => ps.find(vivant)))
    expect(avec.length - sans.length).toBe(11)
    const debut = avec.findIndex((o, i) => JSON.stringify(o) !== JSON.stringify(sans[i]))
    const bloc = avec.slice(debut, debut + 11)
    expect([...avec.slice(0, debut), ...avec.slice(debut + 11)]).toEqual(sans)
    const arcs = bloc.filter((o) => o.op === 'arc')
    expect(arcs).toHaveLength(2)
    expect(arcs[0].args).toEqual(arcs[1].args)
    expect(arcs[0].args[2]).toBeGreaterThan(0)
    const encres = bloc.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toEqual([inkOf('--background'), inkOf('--foreground')])
    const [lisere, anneau] = bloc.filter((o) => o.op === 'set lineWidth').map((o) => o.args[0] as number)
    expect(lisere).toBeGreaterThan(anneau)
    expect(lisere / anneau).toBeCloseTo((COTES_PION_CERCLE.anneauPx + 2 * COTES_PION_CERCLE.lisereExtraPx) / COTES_PION_CERCLE.anneauPx)
  })

  it('joueur mort à l’image lue : aucun anneau', () => {
    // Une image où un joueur du roster n'a aucune vie qui la couvre : on la cherche, on ne la suppose pas.
    const players = buildPlayers(doc, [])
    let mort = -1
    for (let f = 0; f < doc.frameCount && mort < 0; f += 25) {
      if (players.some((p) => !p.lives.some((l) => isAliveAt(l, f)))) mort = f
    }
    expect(mort, 'temoin : aucune image ne laisse un joueur sans vie').toBeGreaterThanOrEqual(0)
    const sansFiche = render(doc, mort)
    const avec = render(doc, mort, ouverte((ps) => ps.find((p) => !p.lives.some((l) => isAliveAt(l, mort)))))
    expect(avec).toEqual(sansFiche)
  })

  it('fiche fermée : le dessin est celui d’avant', () => {
    expect(render(doc, frame, () => null)).toEqual(sans)
  })

  it('joueur masqué : pas d’anneau sans pion', () => {
    const avec = render(doc, frame, ouverte((ps) => ps.find(vivant), { hide: true }))
    const lignes = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'set strokeStyle' && o.args[0] === inkOf('--foreground')).length
    expect(lignes(avec)).toBeLessThanOrEqual(lignes(sans))
  })
})
