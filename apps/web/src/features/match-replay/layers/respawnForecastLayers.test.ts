/**
 * respawnForecastLayers.test.ts — LA PROBABILITÉ DE RÉAPPARITION PEINTE, au contexte enregistreur,
 * EN LECTURE : le même jeu de peintres est appelé à des images successives — avant la mort, pendant le
 * compte à rebours, à la réapparition, après l'effacement — et ce que chacun émet est comparé à ce que la
 * maquette validée montre à ce moment-là. Un rendu qui ne se vérifie qu'à l'arrêt ne prouve pas qu'il
 * apparaît, se met à jour, puis disparaît.
 *
 * Ce qu'ils protègent : (1) rien n'est émis hors de la vie du compte à rebours et du point réel ; (2) la
 * teinte des zones suit la probabilité, les étiquettes sont les trois premières zones avec leur part, et
 * l'anneau se vide ; (3) au point réel, équerres, étiquette « X apparu en Zone, Ne choix » et zone réelle
 * cernée, puis tout s'efface ; (4) Seul / Masquer retirent la probabilité du joueur mort ; (5) le rognage
 * au plan est optionnel et passe APRÈS la teinte ; (6) aucune couleur écrite en dur.
 */
import { describe, expect, it, vi } from 'vitest'

import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'

import type { CanvasView } from '../model/replayView'
import {
  DISC_MIN_P,
  discRadiusPx,
  momentsAt,
  normalizeSpawnForecast,
  primaryMoment,
  REVEAL_HOLD_MS,
  tintAlpha,
  type ForecastDraw,
  type SpawnForecastReady,
} from '../model/respawnForecast.logic'
import { count, recordingContext, type CanvasOp } from '../test/recordingContext'
import { drawForecastLegend } from './respawnForecastLegend'
import { drawForecastMarks } from './respawnForecastMarks'
import { drawForecastPoints } from './respawnForecastPoints'
import type { ForecastPaint, ForecastWording, ForecastZone } from './respawnForecastPaint'
import { clipToMapPlan } from './mapBackgroundDraw'
import { drawForecastZones, type ZoneClip, type ZonesPaint } from './respawnForecastZones'

/** 4 px par unité monde, Y inversé. */
const VIEW: CanvasView = {
  bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 } as CanvasView['bounds'],
  width: 400,
  height: 400,
  pad: 0,
}

/** 100 ms par image. */
const FRAME_MS = 100

function square(volumeIndex: number, x0: number, y0: number, size: number): CalloutZoneReady {
  return {
    names: {}, x: x0, y: y0, z: 0, zBottom: -1, zTop: 1, volumeIndex, big: true,
    polygon: [{ x: x0, y: y0 }, { x: x0 + size, y: y0 }, { x: x0 + size, y: y0 + size }, { x: x0, y: y0 + size }],
    parts: [], holes: [], labelX: x0 + size / 2, labelY: y0 + size / 2,
  }
}

const ZONES = new Map<number, ForecastZone>([
  [3, { name: 'Fosse', shape: square(3, 0, 0, 30) }],
  [7, { name: 'Pont', shape: square(7, 40, 40, 30) }],
  [4, { name: 'Grenier', shape: square(4, 70, 0, 20) }],
  [9, { name: 'Hydro', shape: square(9, 0, 70, 20) }],
])

const WORDING: ForecastWording = {
  title: 'Probabilité de réapparition',
  chip: (name, s) => `${name} · ${s} s`,
  rank: (r) => (r === 1 ? '1er choix' : `${r}e choix`),
  share: (p) => `${Math.round(p * 100)} %`,
  legendCountdown: (name, s) => `${name} réapparaît dans ${s} s`,
  appearedIn: (name, zone) => `${name} apparu en ${zone}`,
  appearedOutside: (name) => `${name} apparu hors zone nommée`,
  appearedRank: (rank, share) => `${rank} · ${share}`,
  appearedUnlikely: 'hors des zones probables',
  legendPoints: 'Disque : probabilité du point',
  legendTintUnderPoints: 'Teinte légère : probabilité de la zone',
  legendScale: 'Teinte : probabilité de la zone',
  legendFooter: 'Mise à jour chaque seconde',
}

const NAMES: Record<string, string> = { 'xuid(a)': 'Brume', 'xuid(b)': 'Orage' }
const COLORS: Record<string, string> = { 'xuid(a)': 'ink-a', 'xuid(b)': 'ink-b' }

function paint(over: Partial<ForecastPaint> = {}): ForecastPaint {
  return {
    k: 1,
    draw: 'zones',
    ink: { panel: 'ink-panel', onPanel: 'ink-on-panel', underlay: 'ink-under', neutral: 'ink-neutral' },
    wording: WORDING,
    zones: ZONES,
    colorOf: (x) => COLORS[x] ?? 'ink-neutral',
    nameOf: (x) => NAMES[x] ?? '—',
    ...over,
  }
}

/** Brume meurt à l'image 100 et réapparaît à 130 dans « Pont » (2e puis 1re zone) ; Orage meurt à 110 et réapparaît à 150. */
function forecast(): SpawnForecastReady {
  const f = normalizeSpawnForecast({
    status: 'ready',
    zones: [{ volume_index: 3, name: 'Fosse' }, { volume_index: 7, name: 'Pont' }, { volume_index: 4, name: 'Grenier' }, { volume_index: 9, name: 'Hydro' }],
    // Un point par zone (zone_index 0 à 3), plus un point hors de toute zone servie (-1).
    points: [
      { x: 10, y: 10, zone_index: 0 }, { x: 50, y: 50, zone_index: 1 }, { x: 75, y: 10, zone_index: 2 },
      { x: 10, y: 75, zone_index: 3 }, { x: 90, y: 90, zone_index: -1 },
    ],
    deaths: [
      {
        xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130, death: { x: 10, y: 20 }, actual: { x: 50, y: 60 }, actual_zone: 7,
        samples: [
          { frame: 100, points_p: [0.5, 0.3, 0.1, 0.05, 0.05], zones: [{ zone: 3, p: 0.5 }, { zone: 7, p: 0.3 }, { zone: 4, p: 0.1 }, { zone: 9, p: 0.05 }] },
          { frame: 110, points_p: [0.2, 0.6, 0, 0, 0.2], zones: [{ zone: 7, p: 0.6 }, { zone: 3, p: 0.2 }] },
          { frame: 120, points_p: [0.1, 0.7, 0, 0, 0.2], zones: [{ zone: 7, p: 0.7 }, { zone: 3, p: 0.1 }] },
        ],
      },
      {
        xuid: 'xuid(b)', death_frame: 110, respawn_frame: 150, death: { x: 80, y: 10 }, actual: { x: 80, y: 10 }, actual_zone: 4,
        samples: [{ frame: 110, points_p: [0, 0, 0.8, 0, 0.2], zones: [{ zone: 4, p: 0.8 }] }],
      },
    ],
  })
  if (!f) throw new Error('prevision attendue prete')
  return f
}

/** Un contexte dont `measureText` rend une largeur : les pastilles se mesurent. */
function context() {
  const r = recordingContext()
  ;(r.ctx as unknown as { measureText: (t: string) => { width: number } }).measureText = (t) => ({ width: t.length * 6 })
  r.ops.length = 0 // la pose du double n'est pas un geste de dessin
  return r
}

interface Rendu {
  zones: CanvasOp[]
  points: CanvasOp[]
  marks: CanvasOp[]
  legend: CanvasOp[]
}

/** Les trois peintres à une image, chacun sur son contexte : ce que la scène aurait émis. */
function render(
  f: SpawnForecastReady,
  frame: number,
  over: { shown?: (x: string) => boolean; p?: ForecastPaint; clip?: ZoneClip; draw?: ForecastDraw } = {},
): Rendu {
  const moments = momentsAt(f, frame, FRAME_MS, over.shown)
  const p = over.p ?? paint({ draw: over.draw ?? 'zones' })
  const z = context()
  // Ce que le calque « fond » fait : la teinte sauf en « Points », les disques sauf en « Zones ».
  if (p.draw !== 'points') drawForecastZones(z.ctx, moments, VIEW, { ...p, clip: over.clip } satisfies ZonesPaint)
  const d = context()
  if (p.draw !== 'zones') drawForecastPoints(d.ctx, moments, VIEW, p)
  const m = context()
  drawForecastMarks(m.ctx, moments, VIEW, p)
  const l = context()
  drawForecastLegend(l.ctx, primaryMoment(moments), VIEW, p)
  return { zones: z.ops, points: d.ops, marks: m.ops, legend: l.ops }
}

const texts = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'fillText').map((o) => String(o.args[0]))
const alphas = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'set globalAlpha').map((o) => o.args[0] as number)
const arcs = (ops: CanvasOp[]) => ops.filter((o) => o.op === 'arc')

describe('la probabilité de réapparition, lue à des instants successifs', () => {
  const f = forecast()

  it('AVANT la mort : aucun peintre n’émet rien', () => {
    for (const draw of ['both', 'zones', 'points'] as const) {
      const avant = render(f, 90, { draw })
      expect([avant.zones, avant.points, avant.marks, avant.legend]).toEqual([[], [], [], []])
    }
    const r = render(f, 90)
    expect(r.zones).toEqual([])
    expect(r.marks).toEqual([])
    expect(r.legend).toEqual([])
  })

  it('PENDANT le compte à rebours : zones teintées, anneau, puce, trois étiquettes, légende', () => {
    const r = render(f, 105)
    // Seule la mort de Brume est ouverte à l'image 105 : quatre zones teintées de son encre, à l'opacité de leur part.
    expect(count(r.zones, 'fill')).toBe(4)
    expect(r.zones.filter((o) => o.op === 'fill').every((o) => o.args[0] === 'evenodd')).toBe(true)
    expect(alphas(r.zones).filter((a) => a > 0 && a < 1).map((a) => +a.toFixed(4))).toEqual(
      [0.5, 0.3, 0.1, 0.05].map((p) => +tintAlpha(p, 0.5).toFixed(4)),
    )
    expect(r.zones.filter((o) => o.op === 'set fillStyle').every((o) => o.args[0] === 'ink-a')).toBe(true)
    expect(count(r.zones, 'stroke')).toBe(0)
    // L'anneau : une piste entière et l'arc de ce qui reste (2,5 s sur 3 s), la puce du joueur.
    expect(arcs(r.marks)).toHaveLength(2)
    expect(texts(r.marks)).toContain('Brume · 3 s')
    // Les trois premières zones, avec leur part, de la plus à la moins probable.
    expect(texts(r.marks).filter((t) => /^\d /.test(t))).toEqual(['1  Fosse 50 %', '2  Pont 30 %', '3  Grenier 10 %'])
    // La légende raconte le compte à rebours.
    expect(texts(r.legend)).toContain('Probabilité de réapparition')
    expect(texts(r.legend)).toContain('Brume réapparaît dans 3 s')
    expect(texts(r.legend)).toContain('50 %')
  })

  it('l’anneau SE VIDE : l’arc du compte à rebours raccourcit d’une image à l’autre', () => {
    const arcEnd = (frame: number) => {
      const a = arcs(render(f, frame).marks)
      return (a[1].args as number[])[4]
    }
    expect(arcEnd(101)).toBeGreaterThan(arcEnd(115))
    expect(arcEnd(115)).toBeGreaterThan(arcEnd(129))
  })

  it('l’échantillon se met à jour à la seconde : les étiquettes suivent l’image', () => {
    expect(texts(render(f, 105).marks).filter((t) => /^\d /.test(t))).toHaveLength(3)
    // À 115, l'échantillon est celui de 110 : deux zones seulement, « Pont » passe en tête.
    const seul = (x: string) => x === 'xuid(a)'
    expect(texts(render(f, 112, { shown: seul }).marks).filter((t) => /^\d /.test(t))).toEqual(['1  Pont 60 %', '2  Fosse 20 %'])
  })

  it('deux joueurs morts ensemble : deux teintes, deux anneaux, la légende raconte le plus récent', () => {
    const r = render(f, 115)
    expect(r.zones.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])).toContain('ink-a')
    expect(r.zones.filter((o) => o.op === 'set fillStyle').map((o) => o.args[0])).toContain('ink-b')
    expect(arcs(r.marks).length).toBeGreaterThanOrEqual(4)
    expect(texts(r.legend).some((t) => t.startsWith('Orage'))).toBe(true)
  })

  it('À LA RÉAPPARITION : équerres, étiquette « X apparu en Zone, Ne choix · p % », zone réelle cernée, teinte retenue', () => {
    const r = render(f, 131)
    // Brume a réapparu dans « Pont » : 1re zone de la dernière probabilité, à 70 %. Orage est toujours en compte à rebours.
    expect(texts(r.marks)).toContain('Brume apparu en Pont')
    expect(texts(r.marks)).toContain('1er choix · 70 %')
    expect(texts(r.marks)).toContain('Orage · 2 s')
    // La zone réelle est cernée (trait de 3 px à l'encre du joueur), une seule fois.
    expect(r.zones.filter((o) => o.op === 'set lineWidth').map((o) => o.args[0])).toContain(3)
    expect(count(r.zones, 'stroke')).toBe(1)
    // Quatre équerres : quatre départs de polyligne d'angle, plus le rond fin du point.
    expect(count(r.marks, 'moveTo')).toBeGreaterThanOrEqual(4)
    // La légende : où il est apparu et à quel rang.
    expect(texts(r.legend)).toContain('Orage réapparaît dans 2 s')
    // La teinte de Brume se retient de moitié.
    const seul = (x: string) => x === 'xuid(a)'
    const plein = alphas(render(f, 129, { shown: seul }).zones).filter((a) => a > 0 && a < 1)
    const retenu = alphas(render(f, 131, { shown: seul }).zones).filter((a) => a > 0 && a < 1)
    expect(Math.max(...retenu)).toBeLessThan(Math.max(...plein))
  })

  it('la légende d’un point réel dit où et à quel rang (la seule mort active)', () => {
    const seul = (x: string) => x === 'xuid(a)'
    const r = render(f, 131, { shown: seul })
    expect(texts(r.legend)).toContain('Brume apparu en Pont, 1er choix · 70 %')
  })

  it('les étiquettes des zones autres que la zone réelle s’estompent au point réel', () => {
    const seul = (x: string) => x === 'xuid(a)'
    const labelAlpha = (frame: number, nom: string) => {
      const ops = render(f, frame, { shown: seul }).marks
      const i = ops.findIndex((o) => o.op === 'fillText' && String(o.args[0]).includes(nom))
      // L'opacité posée juste avant le texte de la pastille.
      return ops.slice(0, i).filter((o) => o.op === 'set globalAlpha').at(-1)?.args[0] as number
    }
    expect(labelAlpha(131, 'Fosse')).toBeLessThan(labelAlpha(131, 'Pont'))
    expect(labelAlpha(125, 'Fosse')).toBe(labelAlpha(125, 'Pont'))
  })

  it('le fondu : le dessin s’éteint avant la fin de la tenue, puis plus rien', () => {
    const seul = (x: string) => x === 'xuid(a)'
    const apres = (ms: number) => 130 + ms / FRAME_MS
    const opacite = (frame: number) => Math.max(...alphas(render(f, frame, { shown: seul }).legend))
    expect(opacite(apres(1000))).toBeGreaterThan(opacite(apres(REVEAL_HOLD_MS - 200)))
    expect(render(f, apres(REVEAL_HOLD_MS - 1), { shown: seul }).marks.length).toBeGreaterThan(0)
  })

  it('APRÈS l’effacement : plus rien, ni teinte, ni repère, ni légende', () => {
    const seul = (x: string) => x === 'xuid(a)'
    const r = render(f, 130 + REVEAL_HOLD_MS / FRAME_MS + 1, { shown: seul })
    expect(r.zones).toEqual([])
    expect(r.marks).toEqual([])
    expect(r.legend).toEqual([])
  })
})

describe('Seul et Masquer : la probabilité appartient au joueur mort', () => {
  const f = forecast()

  it('Seul sur Brume : sa probabilité reste, celle d’Orage est retirée', () => {
    const r = render(f, 115, { shown: (x) => x === 'xuid(a)' })
    expect(r.zones.filter((o) => o.op === 'set fillStyle').every((o) => o.args[0] === 'ink-a')).toBe(true)
    expect(texts(r.marks).some((t) => t.startsWith('Orage'))).toBe(false)
    expect(texts(r.marks)).toContain('Brume · 2 s')
  })

  it('Masquer Brume : sa probabilité est retirée avec ses autres traces, celle d’Orage reste', () => {
    const r = render(f, 115, { shown: (x) => x !== 'xuid(a)' })
    expect(r.zones.filter((o) => o.op === 'set fillStyle').every((o) => o.args[0] === 'ink-b')).toBe(true)
    expect(texts(r.marks).some((t) => t.startsWith('Brume'))).toBe(false)
    expect(texts(r.legend).some((t) => t.startsWith('Orage'))).toBe(true)
  })

  it('Seul sur un joueur qui n’est pas mort à cet instant : rien', () => {
    const r = render(f, 105, { shown: (x) => x === 'xuid(b)' })
    expect(r.zones).toEqual([])
    expect(r.marks).toEqual([])
    expect(r.legend).toEqual([])
  })
})

describe('le rognage au plan', () => {
  const f = forecast()

  it('sans rognage : la teinte se peint directement, rien n’est recopié', () => {
    const r = render(f, 105)
    expect(count(r.zones, 'drawImage')).toBe(0)
  })

  it('avec un rognage : teinte sur la toile de travail, masque APRÈS la teinte, puis recopie sur la scène', () => {
    const work = context()
    const clip: ZoneClip = {
      scratch: vi.fn(() => ({ canvas: {} as CanvasImageSource, ctx: work.ctx })),
      // Le masque se signe dans le journal de la toile de travail : son rang dit qu'il passe après la teinte.
      mask: vi.fn((ctx) => ctx.fillText('MASQUE', 0, 0)),
    }
    const r = render(f, 105, { clip })
    expect(clip.scratch).toHaveBeenCalledWith(VIEW.width, VIEW.height)
    const rangs = work.ops.map((o) => (o.op === 'fillText' ? 'masque' : o.op === 'fill' ? 'teinte' : ''))
    expect(rangs.filter((x) => x === 'teinte')).toHaveLength(4)
    expect(rangs.lastIndexOf('teinte')).toBeLessThan(rangs.indexOf('masque'))
    expect(clip.mask).toHaveBeenCalledTimes(1)
    // La scène ne reçoit que la recopie de la toile de travail.
    expect(count(r.zones, 'drawImage')).toBe(1)
    expect(count(r.zones, 'fill')).toBe(0)
  })

  it('avec le masque du plan (clipToMapPlan) : la teinte est rognée par l’image plan, sur la toile de travail', () => {
    const work = context()
    const plan = { calibration: { metersPerPixel: 1, originX: 0, originY: 100, widthPx: 100, heightPx: 100, convention: 'test' }, image: {} as CanvasImageSource }
    const clip: ZoneClip = {
      scratch: () => ({ canvas: {} as CanvasImageSource, ctx: work.ctx }),
      mask: (ctx, view) => clipToMapPlan(ctx, plan, view),
    }
    const r = render(f, 105, { clip })
    const rangs = work.ops.map((o) => (o.op === 'fill' ? 'teinte' : o.op === 'set globalCompositeOperation' ? String(o.args[0]) : o.op))
    expect(rangs.filter((x) => x === 'teinte')).toHaveLength(4)
    // Le rognage vient APRÈS toute la teinte : composition destination-in, puis l'image plan, puis retour d'état.
    const apres = rangs.slice(rangs.lastIndexOf('teinte') + 1)
    expect(apres).toContain('destination-in')
    expect(apres.indexOf('destination-in')).toBeLessThan(apres.indexOf('drawImage'))
    // La scène ne reçoit que la recopie rognée, jamais la teinte brute.
    expect(count(r.zones, 'fill')).toBe(0)
    expect(count(r.zones, 'drawImage')).toBe(1)
  })

  it('une toile de travail indisponible : la teinte se peint sans rognage', () => {
    const clip: ZoneClip = { scratch: () => null, mask: vi.fn() }
    const r = render(f, 105, { clip })
    expect(count(r.zones, 'fill')).toBe(4)
    expect(clip.mask).not.toHaveBeenCalled()
  })

  it('sans mort active, le rognage n’est même pas demandé', () => {
    const clip: ZoneClip = { scratch: vi.fn(() => null), mask: vi.fn() }
    render(f, 90, { clip })
    expect(clip.scratch).not.toHaveBeenCalled()
  })
})

describe('encres et zones', () => {
  const f = forecast()
  const INKS = new Set(['ink-a', 'ink-b', 'ink-panel', 'ink-on-panel', 'ink-under', 'ink-neutral'])

  it('AUCUNE couleur écrite en dur : seules les encres passées sont posées', () => {
    for (const frame of [105, 115, 131, 151]) {
      const r = render(f, frame)
      for (const o of [...r.zones, ...r.marks, ...r.legend]) {
        if (o.op === 'set fillStyle' || o.op === 'set strokeStyle') expect(INKS.has(o.args[0] as string)).toBe(true)
      }
    }
  })

  it('une zone sans forme ou inconnue du document n’est pas teintée, sans erreur', () => {
    const sansFormes = paint({ zones: new Map([[7, { name: 'Pont', shape: { ...square(7, 0, 0, 10), polygon: [] } }]]) })
    const r = render(f, 105, { p: sansFormes })
    expect(count(r.zones, 'fill')).toBe(0)
    expect(texts(r.marks).filter((t) => /^\d /.test(t))).toEqual([])
  })

  it('un point réel hors de toute zone nommée : étiquette sans zone', () => {
    const horsZone = normalizeSpawnForecast({
      status: 'ready',
      zones: [{ volume_index: 3, name: 'Fosse' }], points: null,
      deaths: [{ xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130, death: { x: 1, y: 1 }, actual: { x: 2, y: 2 }, samples: [{ frame: 100, points_p: null, zones: [{ zone: 3, p: 0.5 }] }] }],
    })!
    expect(texts(render(horsZone, 131).marks)).toContain('Brume apparu hors zone nommée')
  })

  it('un point réel hors des zones probables : l’étiquette le dit', () => {
    const horsProbables = normalizeSpawnForecast({
      status: 'ready',
      zones: [{ volume_index: 3, name: 'Fosse' }, { volume_index: 7, name: 'Pont' }], points: null,
      deaths: [{ xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130, death: { x: 1, y: 1 }, actual: { x: 50, y: 50 }, actual_zone: 7, samples: [{ frame: 100, points_p: null, zones: [{ zone: 3, p: 1 }] }] }],
    })!
    expect(texts(render(horsProbables, 131).marks)).toContain('hors des zones probables')
  })

  it('les tailles suivent la densité de la scène', () => {
    const k1 = render(f, 105).marks.filter((o) => o.op === 'arc').map((o) => (o.args as number[])[2])
    const k2 = render(f, 105, { p: paint({ k: 2 }) }).marks.filter((o) => o.op === 'arc').map((o) => (o.args as number[])[2])
    expect(k2[0]).toBeCloseTo(k1[0] * 2, 9)
  })
})

describe('le choix du dessin : zones et points, zones, points — lu à des instants successifs', () => {
  const f = forecast()
  const seul = (x: string) => x === 'xuid(a)'
  const rayons = (ops: CanvasOp[]) => arcs(ops).map((o) => (o.args as number[])[2])
  const attendus = (ps: number[]) => ps.map((p) => +discRadiusPx(p).toFixed(6))

  it('« Zones et points » (défaut) : teinte légère ET disques pesés par-dessus, légende des deux', () => {
    const r = render(f, 105, { draw: 'both' })
    // La teinte reste un fond léger : les opacités « sous les points », plus faibles que celles de « Zones ».
    const tint = alphas(r.zones).filter((a) => a > 0 && a < 1).map((a) => +a.toFixed(4))
    expect(tint).toEqual([0.5, 0.3, 0.1, 0.05].map((p) => +tintAlpha(p, 0.5, 'both').toFixed(4)))
    expect(Math.max(...tint)).toBeLessThan(Math.max(...alphas(render(f, 105, { draw: 'zones' }).zones).filter((a) => a < 1)))
    // Un disque par point, de rayon suivant la probabilité du point (les cinq points portent une probabilité).
    expect(rayons(r.points).map((x) => +x.toFixed(6))).toEqual(attendus([0.5, 0.3, 0.1, 0.05, 0.05]))
    expect(count(r.points, 'fill')).toBe(5)
    expect(count(r.points, 'stroke')).toBe(5)
    expect(texts(r.legend)).toContain('Disque : probabilité du point')
    expect(texts(r.legend)).toContain('Teinte légère : probabilité de la zone')
    expect(texts(r.legend)).not.toContain('Teinte : probabilité de la zone')
  })

  it('« Zones » : la teinte seule, aucun disque, légende de la teinte avec ses bornes', () => {
    const r = render(f, 105, { draw: 'zones' })
    expect(count(r.zones, 'fill')).toBe(4)
    expect(r.points).toEqual([])
    expect(texts(r.legend)).toContain('Teinte : probabilité de la zone')
    expect(texts(r.legend)).toContain('50 %')
    expect(texts(r.legend)).not.toContain('Disque : probabilité du point')
  })

  it('« Points » : les disques seuls, aucune teinte, légende des disques', () => {
    const r = render(f, 105, { draw: 'points' })
    expect(r.zones).toEqual([])
    expect(count(r.points, 'arc')).toBe(5)
    expect(texts(r.legend)).toContain('Disque : probabilité du point')
    expect(texts(r.legend)).not.toContain('Teinte : probabilité de la zone')
    expect(texts(r.legend)).not.toContain('Teinte légère : probabilité de la zone')
  })

  it('les disques SUIVENT l’échantillon : un point sans probabilité disparaît, les rayons changent à la seconde', () => {
    // À l'image 112, l'échantillon est celui de 110 : trois points seulement portent une probabilité.
    const r = render(f, 112, { draw: 'points', shown: seul })
    expect(rayons(r.points).map((x) => +x.toFixed(6))).toEqual(attendus([0.2, 0.6, 0.2]))
    // À 125 : celui de 120, « Pont » grossit.
    expect(rayons(render(f, 125, { draw: 'points', shown: seul }).points)[1]).toBeGreaterThan(rayons(r.points)[1])
    expect(DISC_MIN_P).toBeGreaterThan(0)
  })

  it('À LA RÉAPPARITION : disques estompés, équerres sur le point réel, quel que soit le choix', () => {
    for (const draw of ['both', 'points'] as const) {
      const avant = alphas(render(f, 129, { draw, shown: seul }).points)
      const apres = alphas(render(f, 131, { draw, shown: seul }).points)
      expect(Math.max(...apres.filter((a) => a < 1))).toBeLessThan(Math.max(...avant.filter((a) => a < 1)))
      expect(Math.max(...apres.filter((a) => a < 1))).toBeLessThan(0.2)
    }
    for (const draw of ['both', 'zones', 'points'] as const) {
      const r = render(f, 131, { draw, shown: seul })
      expect(texts(r.marks)).toContain('Brume apparu en Pont')
      // Quatre équerres : le départ de chaque polyligne d'angle (le rond fin du point s'y ajoute avec un arc).
      expect(count(r.marks, 'moveTo')).toBeGreaterThanOrEqual(4)
    }
  })

  it('le fondu puis l’effacement : plus aucun disque après la tenue', () => {
    const apres = 130 + REVEAL_HOLD_MS / FRAME_MS + 1
    for (const draw of ['both', 'zones', 'points'] as const) {
      const r = render(f, apres, { draw, shown: seul })
      expect([r.zones, r.points, r.marks, r.legend]).toEqual([[], [], [], []])
    }
    const plein = alphas(render(f, 131, { draw: 'points', shown: seul }).points).filter((a) => a < 1)
    const fondu = alphas(render(f, 130 + 2_600 / FRAME_MS, { draw: 'points', shown: seul }).points).filter((a) => a < 1)
    expect(Math.max(...fondu)).toBeLessThan(Math.max(...plein))
  })

  it('Seul / Masquer retirent aussi les disques du joueur mort', () => {
    const tous = render(f, 115, { draw: 'points' })
    expect(count(tous.points, 'arc')).toBe(3 + 2)
    expect(count(render(f, 115, { draw: 'points', shown: seul }).points, 'arc')).toBe(3)
    expect(count(render(f, 115, { draw: 'points', shown: (x) => x === 'xuid(b)' }).points, 'arc')).toBe(2)
    expect(count(render(f, 115, { draw: 'points', shown: (x) => x !== 'xuid(a)' && x !== 'xuid(b)' }).points, 'arc')).toBe(0)
  })

  it('une réponse sans probabilité de point alignée : aucun disque, la teinte reste', () => {
    const sansPoints = normalizeSpawnForecast({
      status: 'ready',
      zones: [{ volume_index: 3, name: 'Fosse' }], points: [{ x: 1, y: 1, zone_index: 0 }, { x: 2, y: 2, zone_index: -1 }],
      deaths: [{ xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130, death: { x: 1, y: 1 }, actual: { x: 2, y: 2 }, actual_zone: 3,
        samples: [{ frame: 100, points_p: [1], zones: [{ zone: 3, p: 1 }] }] }],
    })!
    const r = render(sansPoints, 105, { draw: 'both' })
    expect(r.points).toEqual([])
    expect(count(r.zones, 'fill')).toBe(1)
  })

  it('les rayons suivent la densité de la scène', () => {
    const k1 = rayons(render(f, 105, { draw: 'points' }).points)
    const k2 = rayons(render(f, 105, { draw: 'points', p: paint({ draw: 'points', k: 2 }) }).points)
    expect(k2[0]).toBeCloseTo(k1[0] * 2, 9)
  })
})
