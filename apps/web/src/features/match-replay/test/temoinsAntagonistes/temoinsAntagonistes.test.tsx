/// <reference types="node" />
/**
 * temoinsAntagonistes.test.tsx : LES TÉMOINS DE LA TOILE, DE LA FRISE ET DU FILTRE SEUL / MASQUER.
 *
 * Ce fichier PRODUIT des sorties, il ne les compare à aucune fixation versionnée : chaque cas
 * vérifie qu'il sait produire la sienne, et que deux productions de suite sont identiques. Quand
 * la variable d'environnement `TEMOINS_ANT_DIR` désigne un dossier, les sorties y sont écrites à
 * la fin, avec `empreintes.txt` (une ligne `<sha256>  <fichier>` par témoin, au format de
 * `sha256sum -c`). Deux captures prises à deux états du code se comparent alors fichier par
 * fichier ; la commande exacte est dans le README de ce dossier.
 *
 * LES SORTIES :
 *  - `toile-sans-N.json`, `toile-avec-N.json` : la suite des appels de dessin de la toile à trois
 *    images sans antagoniste du point de vue, puis à trois images où ses deux titres existent ;
 *  - `toile-seul.json`, `toile-masquer.json` : la même toile avec Seul, puis Masquer, du premier
 *    adversaire à éliminer le point de vue ;
 *  - `frise.html`, `frise-piste-point-de-vue.html`, `frise-piste-coequipiers.html` : le DOM de la
 *    frise et de ses deux pistes de marques, une balise par ligne ;
 *  - `filtre-seul.json`, `filtre-masquer.json` : les réponses des règles de Seul et de Masquer ;
 *  - `tests-existants.txt` : les tests de la feature qui portent ces règles, par nom ;
 *  - `scene.json` : le film, le point de vue et les images retenues.
 */
import { createHash } from 'node:crypto'
import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

import { cleanup, render } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import { formatClock } from '@/lib/replay/replayLogic'

import type { CanvasOp } from '../recordingContext'

import { reduceFeed } from '../../hooks/useReplayTimeline'
import { buildEventTracks, trackScale, type EventTracks } from '../../model/replayTimelineTracksLogic'
import { ReplayMarkTrack } from '../../ui/ReplayMarkTrack'
import { renderTracks } from '../timelineTracksHarness'
import { listeDesTests, occupantDuSlot, sortiesDuFiltre } from './filtreTemoin'
import {
  BASCULES_ETEINTES_PAR_DEFAUT,
  FILMS,
  duelsDuPointDeVue,
  focusDe,
  imagesTemoins,
  installerDoubles,
  peindre,
  premierTueurDuPointDeVue,
  sceneDe,
  serialiserToile,
  type ImagesTemoins,
  type Scene,
} from './temoinsKit'

const DOSSIER = process.env.TEMOINS_ANT_DIR?.trim() || null
/** Le montage de la toile sur un film entier prend plusieurs secondes sur un poste chargé. */
const DELAI_MS = 180_000

const sorties = new Map<string, string>()
const ATTENDUES = [
  'toile-sans-1.json', 'toile-sans-2.json', 'toile-sans-3.json',
  'toile-avec-1.json', 'toile-avec-2.json', 'toile-avec-3.json',
  'toile-seul.json', 'toile-masquer.json',
  'frise.html', 'frise-piste-point-de-vue.html', 'frise-piste-coequipiers.html',
  'filtre-seul.json', 'filtre-masquer.json',
  'tests-existants.txt', 'scene.json',
]

let scene: Scene
let images: ImagesTemoins

beforeAll(() => {
  scene = sceneDe(FILMS.base)
  images = imagesTemoins(scene)
})

beforeEach(() => {
  localStorage.clear()
  for (const key of BASCULES_ETEINTES_PAR_DEFAUT) localStorage.setItem(key, 'true')
  installerDoubles()
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

afterAll(() => {
  if (!DOSSIER) return
  const manquantes = ATTENDUES.filter((nom) => !sorties.has(nom))
  if (manquantes.length > 0) throw new Error(`témoins : sorties non produites, rien n'est écrit : ${manquantes.join(', ')}`)
  mkdirSync(DOSSIER, { recursive: true })
  const empreintes: string[] = []
  for (const nom of [...sorties.keys()].sort()) {
    const contenu = sorties.get(nom) as string
    writeFileSync(join(DOSSIER, nom), contenu, 'utf8')
    empreintes.push(`${createHash('sha256').update(contenu, 'utf8').digest('hex')}  ${nom}`)
  }
  writeFileSync(join(DOSSIER, 'empreintes.txt'), `${empreintes.join('\n')}\n`, 'utf8')
})

/** Le nombre de duels du point de vue survenus à cette image ou avant. */
function duelsJusqua(frame: number): number {
  return duelsDuPointDeVue(scene).filter((e) => e.frame <= frame).length
}

describe('témoins des antagonistes : la scène', () => {
  it('le film de fixture nomme ses éliminations, et le fil les pose sur leur image', () => {
    expect(scene.eliminations.length).toBeGreaterThan(20)
    const kills = scene.feed.filter((e) => e.kill)
    expect(kills).toHaveLength(scene.eliminations.length)
    const frames = kills.map((e) => Math.round(e.replayMs / (scene.doc.frameIntervalMs ?? 1))).sort((a, b) => a - b)
    expect(frames).toEqual(scene.eliminations.map((e) => e.frame))
  })

  it('trois images sans antagoniste du point de vue, trois images avec ses deux titres', () => {
    expect(images.sans).toHaveLength(3)
    expect(images.avec).toHaveLength(3)
    for (const f of images.sans) expect(duelsJusqua(f)).toBe(0)
    for (const f of images.avec) expect(f).toBeGreaterThan(images.deuxTitres)
    const duels = duelsDuPointDeVue(scene)
    sorties.set(
      'scene.json',
      `${JSON.stringify(
        {
          film: scene.film,
          images: scene.doc.frameCount,
          joueurs: scene.players.length,
          pointDeVue: scene.players.findIndex((p) => p.xuid === scene.viewpoint),
          joueurFiltre: scene.players.indexOf(premierTueurDuPointDeVue(scene)),
          eliminations: scene.eliminations.length,
          duelsDuPointDeVue: duels.length,
          premierDuel: images.premierDuel,
          deuxTitres: images.deuxTitres,
          imagesSans: images.sans,
          imagesAvec: images.avec.map((f) => ({ image: f, duelsAvant: duelsJusqua(f) })),
        },
        null,
        1,
      )}\n`,
    )
  })
})

describe('témoins des antagonistes : la toile', () => {
  const cas = [
    ...[0, 1, 2].map((i) => ({ nom: `toile-sans-${i + 1}.json`, genre: 'sans' as const, i })),
    ...[0, 1, 2].map((i) => ({ nom: `toile-avec-${i + 1}.json`, genre: 'avec' as const, i })),
  ]
  it.each(cas)('$nom : la toile se peint, deux fois de la même façon', ({ nom, genre, i }) => {
    const frame = images[genre][i]
    const ops = peindre(scene, frame)
    expect(ops.length).toBeGreaterThan(100)
    expect(ops.some((o) => o.op === 'fillText')).toBe(true)
    cleanup()
    expect(peindre(scene, frame)).toEqual(ops)
    sorties.set(nom, serialiserToile({ film: scene.film, genre, image: frame, duelsAvant: duelsJusqua(frame) }, ops))
  }, DELAI_MS)

  it.each(cas)('$nom : les antagonistes n’ajoutent des icônes et des comptes qu’aux images où un titre existe', ({ genre, i }) => {
    const frame = images[genre][i]
    const avec = peindre(scene, frame)
    cleanup()
    const sans = peindre(scene, frame, null, false)
    const nombre = (ops: CanvasOp[], op: string) => ops.filter((o) => o.op === op).length
    if (genre === 'sans') {
      expect(avec).toEqual(sans)
      return
    }
    // Une icône teintée (une toile) par titre porté, et son compte écrit : un à deux titres, jamais plus de deux.
    const icones = nombre(avec, 'drawImage') - nombre(sans, 'drawImage')
    expect(icones).toBeGreaterThanOrEqual(1)
    expect(icones).toBeLessThanOrEqual(2)
    expect(nombre(avec, 'fillText') - nombre(sans, 'fillText')).toBe(icones)
  }, DELAI_MS)

  it.each(['seul', 'masquer'] as const)('toile-%s.json : la toile se peint sous le filtre d’un joueur', (mode) => {
    const frame = images.avec[1]
    const sansFiltre = peindre(scene, frame)
    cleanup()
    const ops = peindre(scene, frame, focusDe(scene, premierTueurDuPointDeVue(scene), mode))
    expect(ops.length).toBeGreaterThan(100)
    expect(ops).not.toEqual(sansFiltre)
    sorties.set(`toile-${mode}.json`, serialiserToile({ film: scene.film, genre: mode, image: frame }, ops))
  }, DELAI_MS)
})

/** Une balise par ligne : deux captures se comparent ligne à ligne. */
function domParLignes(html: string): string {
  return `${html.replace(/></g, '>\n<')}\n`
}

describe('témoins des antagonistes : la frise', () => {
  let tracks: EventTracks
  beforeAll(() => {
    const { doc, viewpoint, allegiance } = scene
    tracks = buildEventTracks(
      reduceFeed(scene.feed, viewpoint),
      { viewpoint, allegiance, marks: new Map([[viewpoint, 'me' as const]]) },
      doc.frameIntervalMs ?? 0,
      trackScale(null, doc.frameCount),
      formatClock,
    )
  })

  it('les deux pistes portent des marques : éliminations, morts et médailles du point de vue', () => {
    expect(tracks.own.some((m) => m.kind === 'kill')).toBe(true)
    expect(tracks.own.some((m) => m.kind === 'death')).toBe(true)
    expect(tracks.own.some((m) => m.medals.length > 0)).toBe(true)
    expect(tracks.teammates.length).toBeGreaterThan(0)
  })

  it('frise.html : la frise entière se rend, deux fois de la même façon', () => {
    const rendre = () => {
      const { container } = renderTracks({ own: tracks.own, teammates: tracks.teammates, maxFrame: scene.doc.frameCount - 1 })
      const html = container.innerHTML
      cleanup()
      return html
    }
    const html = rendre()
    expect(html).toContain('<input')
    expect(rendre()).toBe(html)
    sorties.set('frise.html', domParLignes(html))
  })

  it.each([
    { nom: 'frise-piste-point-de-vue.html', tall: true, height: 'h-[24px]' },
    { nom: 'frise-piste-coequipiers.html', tall: false, height: 'h-3.5' },
  ])('$nom : la piste de marques se rend seule', ({ nom, tall, height }) => {
    const marks = tall ? tracks.own : tracks.teammates
    const { container } = render(<ReplayMarkTrack marks={marks} height={height} tall={tall} />)
    const nues = marks.filter((m) => m.kind !== 'medal').length
    expect(container.querySelectorAll('span[title]').length).toBeGreaterThanOrEqual(nues)
    sorties.set(nom, domParLignes(container.innerHTML))
  })
})

describe('témoins des antagonistes : Seul et Masquer d’un joueur', () => {
  it.each(['seul', 'masquer'] as const)('filtre-%s.json : les règles répondent sur trois films', (mode) => {
    const vehicules = sceneDe(FILMS.vehicules)
    const trajet = vehicules.doc.vehicles.flatMap((v) => v.rides).find((r) => vehicules.players.some((p) => p.xuid === r.xuid))
    const passager = vehicules.players.find((p) => p.xuid === trajet?.xuid)
    const poussees = sceneDe(FILMS.poussees)
    const poussee = poussees.doc.knockbacks[0]
    const pousse = poussee && occupantDuSlot(poussees, poussee.slot, poussee.t)
    if (!passager || !pousse) throw new Error('témoins : les films de fixture n’ont plus de passager ou de joueur poussé')

    const base = sortiesDuFiltre(scene, premierTueurDuPointDeVue(scene), mode) as Record<string, { total: number; dessines: number }>
    // Le filtre retire ou garde quelque chose, sans tout retirer ni tout garder. Sous Masquer, un effet de mort a
    // toujours un second joueur affiché : tous les effets du film se dessinent, aucun n'est retiré.
    const retires = mode === 'seul' ? ['pions', 'effetsDeMort', 'poses'] : ['pions', 'poses']
    for (const genre of ['pions', 'effetsDeMort', 'poses'] as const) {
      expect(base[genre].total).toBeGreaterThan(0)
      expect(base[genre].dessines).toBeGreaterThan(0)
      if (retires.includes(genre)) expect(base[genre].dessines).toBeLessThan(base[genre].total)
      else expect(base[genre].dessines).toBeLessThanOrEqual(base[genre].total)
    }
    const sortie = [base, sortiesDuFiltre(vehicules, passager, mode), sortiesDuFiltre(poussees, pousse, mode)]
    sorties.set(`filtre-${mode}.json`, `${JSON.stringify(sortie, null, 1)}\n`)
  }, DELAI_MS)

  it('tests-existants.txt : la liste par nom des tests qui portent ces règles', () => {
    const liste = listeDesTests()
    expect(liste).toContain('model/playerPanelLogic.hidden.test.ts')
    expect(liste).toContain('model/focusFilter.test.ts')
    sorties.set('tests-existants.txt', liste)
  })
})

describe('témoins des antagonistes : la capture', () => {
  it('toutes les sorties attendues sont produites', () => {
    expect([...sorties.keys()].sort()).toEqual([...ATTENDUES].sort())
    for (const [nom, contenu] of sorties) expect(contenu.length, nom).toBeGreaterThan(0)
  })
})
