/**
 * replayMarkers.antagonists.test.ts : les titres némésis et souffre-douleur sur le calque des
 * pions (piste B du plan des antagonistes).
 *
 * Le calque reçoit, par slot et par image, les titres que porte le joueur (déjà décidés par
 * `createSlotTitles`) : ici ils sont posés à la main, pour observer seulement ce qui se dessine.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SlotTitle } from '../model/antagonistTitles'
import { recordingContext, valuesOf, type CanvasOp } from '../test/recordingContext'
import { createTitleIcons, type AntagonistMarks } from './antagonistMarks'
import { drawTracksLayer, type MarkerStyle } from './replayMarkers'
import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 200, height: 100, pad: 4 }
const COULEUR = 'rgb(1 2 3)'
const CONTOUR = 'rgb(8 12 18)'
const CHAR = 6

const vie = (over: Partial<ReplayTrackReady> = {}): ReplayTrackReady =>
  ({
    slot: 512,
    team: -1,
    xuid: 'A',
    startFrame: 0,
    endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z: 0, h: 90 }],
    ...over,
  }) as ReplayTrackReady

const NEMESIS: SlotTitle = { role: 'nemesis', count: 3 }
const VICTIME: SlotTitle = { role: 'victim', count: 2 }

function marques(titles: readonly SlotTitle[] | ((slot: number, frame: number) => readonly SlotTitle[])): AntagonistMarks {
  const icons = createTitleIcons()
  icons.setSource('nemesis', { naturalWidth: 714, naturalHeight: 670 } as unknown as HTMLImageElement)
  icons.setSource('victim', { naturalWidth: 1099, naturalHeight: 1369 } as unknown as HTMLImageElement)
  const titlesOfSlot = typeof titles === 'function' ? titles : () => titles
  return { titlesOfSlot, titlesOfKey: () => [], icons }
}

function style(over: Partial<MarkerStyle> = {}): MarkerStyle {
  return {
    colorOfSlot: () => COULEUR,
    outline: { band: 'rgb(9 9 9)', rim: 'rgb(250 250 250)' },
    frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 },
    z: { min: 0, max: 10 },
    k: 1,
    showAim: true,
    markOfSlot: () => undefined,
    nameOfSlot: () => 'Bot',
    showTrail: true,
    selfInk: 'rgb(4 4 4)',
    deathInk: 'rgb(5 5 5)',
    labelStroke: CONTOUR,
    offscreenLabelOf: (name, meters) => `${name} ${Math.round(meters)}`,
    ...over,
  }
}

function peint(track: ReplayTrackReady, over: Partial<MarkerStyle>): CanvasOp[] {
  const { ops, ctx } = recordingContext({ charWidth: CHAR })
  drawTracksLayer(ctx, [track], VIEW, style(over))
  return ops
}

let toiles: Map<HTMLCanvasElement, CanvasOp[]>
beforeEach(() => {
  toiles = new Map()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (this: HTMLCanvasElement) {
    const { ops, ctx } = recordingContext()
    if (!toiles.has(this)) toiles.set(this, ops)
    return ctx
  } as never)
})
afterEach(() => vi.restoreAllMocks())

const de = (ops: CanvasOp[], op: string) => ops.filter((o) => o.op === op)
const textes = (ops: CanvasOp[]) => de(ops, 'fillText').map((o) => o.args[0])
const ordre = (ops: CanvasOp[]) =>
  ops.filter((o) => o.op === 'drawImage' || o.op === 'fillText').map((o) => (o.op === 'drawImage' ? 'icone' : o.args[0]))

describe('drawTracksLayer : les titres devant le nom', () => {
  it('l’icône et le compte précèdent le nom sous le pion', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS]) })
    expect(ordre(ops)).toEqual(['icone', '3', 'Bot'])
  })

  it('un joueur sans titre porte exactement ce qu’il portait avant', () => {
    const sans = peint(vie(), {})
    // Le cône de visée pose des dégradés (des objets neufs à chaque appel) : on compare les traces sérialisées.
    expect(JSON.stringify(peint(vie(), { antagonists: marques([]) }))).toBe(JSON.stringify(sans))
    expect(textes(sans)).toEqual(['Bot'])
  })

  it('demande les titres du slot à l’image lue', () => {
    const titlesOfSlot = vi.fn(() => [] as SlotTitle[])
    peint(vie({ slot: 7 }), { antagonists: marques(titlesOfSlot), frame: 50 })
    expect(titlesOfSlot).toHaveBeenCalledWith(7, 50)
  })

  it('un double tenant porte deux icônes devant son nom', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS, VICTIME]) })
    expect(ordre(ops)).toEqual(['icone', '3', 'icone', '2', 'Bot'])
  })

  it('l’icône est à la couleur du joueur, cernée de l’encre du contour des noms', () => {
    peint(vie(), { antagonists: marques([NEMESIS]) })
    const remplissages = [...toiles.values()].flatMap((o) => valuesOfFill(o))
    expect(remplissages).toContain(COULEUR)
    expect(remplissages).toContain(CONTOUR)
  })

  it('le compte et le nom portent la couleur du joueur', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS]) })
    const encres = ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])
    expect(encres.filter((e) => e === COULEUR).length).toBeGreaterThanOrEqual(2)
  })

  it('un joueur masqué ne porte rien : ni pion, ni icône, ni compte', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS]), shownAtSlot: () => false })
    expect(de(ops, 'drawImage')).toEqual([])
    expect(textes(ops)).toEqual([])
  })

  it('un slot dont le nom cède la place à la mise en valeur ne porte pas non plus ses icônes', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS]), nameHiddenAtSlot: () => true })
    expect(de(ops, 'drawImage')).toEqual([])
    expect(textes(ops)).toEqual([])
  })

  it('une vie sans nom n’a pas d’étiquette, donc pas d’icône', () => {
    const ops = peint(vie(), { antagonists: marques([NEMESIS]), nameOfSlot: () => null })
    expect(de(ops, 'drawImage')).toEqual([])
  })

  it('une croix de mort ne porte aucun titre', () => {
    const morte = vie({ startFrame: 0, endFrame: 40 })
    const ops = peint(morte, { antagonists: marques([NEMESIS]), frame: 45 })
    expect(de(ops, 'drawImage')).toEqual([])
  })

  it('un joueur hors cadre : l’étiquette de sa flèche porte l’icône et le compte', () => {
    const loin = vie({ points: [{ t: 50, x: 500, y: 5, z: 0, h: 90 }] })
    const ops = peint(loin, { antagonists: marques([NEMESIS]) })
    expect(ordre(ops).slice(0, 2)).toEqual(['icone', '3'])
    expect(String(textes(ops)[1])).toContain('Bot')
  })
})

describe('drawTracksLayer : la pulsation au changement de titre', () => {
  const arcs = (ops: CanvasOp[]) => de(ops, 'arc').length
  const largeurIcone = (ops: CanvasOp[]) => (de(ops, 'drawImage')[0].args as number[])[3]

  it('le nouveau tenant : deux anneaux autour du pion et l’icône qui grossit', () => {
    const repos = peint(vie(), { antagonists: marques([NEMESIS]) })
    const pulse = peint(vie(), { antagonists: marques([{ ...NEMESIS, motion: { kind: 'gain', progress: 0.5 } }]) })
    expect(arcs(pulse) - arcs(repos)).toBe(2)
    expect(largeurIcone(pulse)).toBeGreaterThan(largeurIcone(repos))
  })

  it('les anneaux sont à la couleur du joueur et centrés sur le pion', () => {
    const repos = peint(vie(), { antagonists: marques([NEMESIS]) })
    const pulse = peint(vie(), { antagonists: marques([{ ...NEMESIS, motion: { kind: 'gain', progress: 0.5 } }]) })
    const supplement = pulse.filter((o) => o.op === 'arc').filter((o) => !repos.some((r) => r.op === 'arc' && r.args.join() === o.args.join()))
    expect(supplement).toHaveLength(2)
    expect(new Set(supplement.map((o) => `${o.args[0]},${o.args[1]}`)).size).toBe(1)
    expect(pulse.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toContain(COULEUR)
  })

  it('au repos (fin des 1,4 s, ou mouvement réduit), ni anneau ni grossissement', () => {
    const repos = peint(vie(), { antagonists: marques([NEMESIS]) })
    expect(arcs(repos)).toBe(arcs(peint(vie(), {})))
  })

  it('l’ancien tenant s’efface : son icône et son compte à opacité décroissante, sans anneau', () => {
    const repos = peint(vie(), { antagonists: marques([NEMESIS]) })
    const efface = peint(vie(), { antagonists: marques([{ ...NEMESIS, motion: { kind: 'loss', progress: 0.5 } }]) })
    expect(valuesOf(efface, 'globalAlpha')).toContain(0.5)
    expect(arcs(efface)).toBe(arcs(repos))
    expect(largeurIcone(efface)).toBe(largeurIcone(repos))
  })

  it('l’étiquette de la flèche hors cadre pulse comme celle du pion, sans anneau', () => {
    const loin = vie({ points: [{ t: 50, x: 500, y: 5, z: 0, h: 90 }] })
    const repos = peint(loin, { antagonists: marques([NEMESIS]) })
    const pulse = peint(loin, { antagonists: marques([{ ...NEMESIS, motion: { kind: 'gain', progress: 0.5 } }]) })
    expect(largeurIcone(pulse)).toBeGreaterThan(largeurIcone(repos))
    expect(arcs(pulse)).toBe(arcs(repos))
  })
})

/** Les couleurs posées en remplissage sur une toile hors écran (la teinte `source-in`). */
function valuesOfFill(ops: CanvasOp[]): unknown[] {
  return ops.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])
}
