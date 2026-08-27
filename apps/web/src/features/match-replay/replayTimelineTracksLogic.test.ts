/**
 * Tests — replayTimelineTracks (ce que la frise montre en dehors du temps).
 *
 * Ce qu'ils protègent, dans l'ordre des règles produit de la planche 2a :
 *  1. QUI EST SUR UNE PISTE. Un acteur sans marque d'identité n'y est pas — la frise parle du
 *     joueur de la page et de ses amis, pas de la salle entière.
 *  2. LES MORTS NE VONT QUE SUR `own`. Une piste alliée mêlant kills et morts serait illisible.
 *  3. LES BORNES DE LA FENÊTRE DE GAMEPLAY. Une marque hors match est ÉCARTÉE, pas rabattue sur
 *     un bord : collée à l'origine, elle se lirait comme un premier frag qui n'a pas eu lieu.
 *  4. AUCUN CHANGEMENT DE MENEUR = AUCUNE BANDE. La donnée dit qu'il n'y a pas eu de
 *     retournement, elle ne dit pas qui menait.
 *  5. UN CLIP OCCUPE SA DURÉE. C'est ce qui le distingue d'une capture à l'œil, sans légende.
 */
import { describe, expect, it } from 'vitest'

import type { PlayerMarkKind } from './playerMarks'
import {
  buildDominance,
  buildEventTracks,
  clipFrameCount,
  placeMedia,
  ratioOfMs,
  THUMB_PX,
  trackLeft,
  trackScale,
  trackWidth,
  type ReplayMediaItem,
  type TrackDeath,
  type TrackKill,
} from './replayTimelineTracksLogic'
import type { ReplayWindowBounds } from './replayWindow'

/** Un document 10 Hz : une image toutes les 100 ms. */
const FRAME_MS = 100

/** Le match court de l'image 100 (10 s) à l'image 400 (40 s) sur l'axe du film. */
const FENETRE: ReplayWindowBounds = {
  startFrame: 100,
  endFrame: 400,
  startMs: 10_000,
  endMs: 40_000,
}

const SCALE = trackScale(FENETRE, 1_000)

/** Horloge de test : on vérifie qu'elle est APPELÉE, pas sa mise en forme (foyer du dépôt). */
const clockOf = (ms: number) => `@${ms}`

const MARKS: ReadonlyMap<string, PlayerMarkKind> = new Map([
  ['me', 'me'],
  ['pote', 'friend'],
])

function kill(over: Partial<TrackKill> = {}): TrackKill {
  return { key: 'k1', replayMs: 20_000, xuid: 'me', ...over }
}

function death(over: Partial<TrackDeath> = {}): TrackDeath {
  return { key: 'd1', replayMs: 25_000, xuid: 'me', ...over }
}

describe('trackScale — l’échelle est celle de la frise', () => {
  it('prend les bornes de la FENÊTRE DE GAMEPLAY quand elle est établie', () => {
    expect(SCALE).toEqual({ from: 100, span: 300 })
  })

  it('retombe sur le FILM ENTIER sans fenêtre — comme la frise elle-même', () => {
    expect(trackScale(null, 501)).toEqual({ from: 0, span: 500 })
  })

  it('rend une portée NULLE sur un document d’une seule image : rien ne se place', () => {
    const degenere = trackScale(null, 1)
    expect(degenere.span).toBe(0)
    expect(buildEventTracks([kill()], [], MARKS, FRAME_MS, degenere, clockOf)).toEqual({
      own: [],
      allies: [],
    })
    expect(buildDominance([{ frame: 10, teamId: 0 }], degenere)).toEqual([])
    expect(placeMedia([media()], FRAME_MS, degenere)).toEqual([])
  })
})

describe('ratioOfMs — un instant sur l’échelle des pistes', () => {
  it('place le coup d’envoi à 0, la fin à 1, le milieu à la moitié', () => {
    expect(ratioOfMs(10_000, FRAME_MS, SCALE)).toBe(0)
    expect(ratioOfMs(40_000, FRAME_MS, SCALE)).toBe(1)
    expect(ratioOfMs(25_000, FRAME_MS, SCALE)).toBe(0.5)
  })

  it('sans échelle temporelle, rend zéro plutôt qu’une position inventée', () => {
    expect(ratioOfMs(25_000, 0, SCALE)).toBe(0)
  })
})

describe('buildEventTracks — qui est sur quelle piste', () => {
  it('un acteur SANS marque n’est sur AUCUNE piste', () => {
    const tracks = buildEventTracks([kill({ xuid: 'inconnu' })], [], MARKS, FRAME_MS, SCALE, clockOf)
    expect(tracks.own).toEqual([])
    expect(tracks.allies).toEqual([])
  })

  it('« moi » sur ma piste, un ami sur celle des alliés', () => {
    const tracks = buildEventTracks(
      [kill({ key: 'k-me' }), kill({ key: 'k-pote', xuid: 'pote' })],
      [],
      MARKS,
      FRAME_MS,
      SCALE,
      clockOf,
    )
    expect(tracks.own.map((m) => m.key)).toEqual(['k-me'])
    expect(tracks.allies.map((m) => m.key)).toEqual(['k-pote'])
  })

  it('LES MORTS NE VONT QUE SUR `own` — celle d’un ami est ignorée', () => {
    const tracks = buildEventTracks(
      [],
      [death({ key: 'd-me' }), death({ key: 'd-pote', xuid: 'pote' })],
      MARKS,
      FRAME_MS,
      SCALE,
      clockOf,
    )
    expect(tracks.own.map((m) => m.key)).toEqual(['d-me'])
    expect(tracks.allies).toEqual([])
  })

  it('la NATURE de la marque voyage avec elle — c’est elle qui décidera de sa couleur', () => {
    const tracks = buildEventTracks([kill()], [death()], MARKS, FRAME_MS, SCALE, clockOf)
    expect(tracks.own.map((m) => m.kind)).toEqual(['kill', 'death'])
  })

  it('une entrée HORS FENÊTRE est écartée, jamais rabattue sur un bord', () => {
    const tracks = buildEventTracks(
      [
        kill({ key: 'avant', replayMs: 5_000 }), // le countdown d'avant-match
        kill({ key: 'apres', replayMs: 45_000 }), // la queue du film
        kill({ key: 'dedans', replayMs: 20_000 }),
      ],
      [],
      MARKS,
      FRAME_MS,
      SCALE,
      clockOf,
    )
    expect(tracks.own.map((m) => m.key)).toEqual(['dedans'])
  })

  it('les BORNES EXACTES du match, elles, sont dedans', () => {
    const tracks = buildEventTracks(
      [kill({ key: 'debut', replayMs: 10_000 }), kill({ key: 'fin', replayMs: 40_000 })],
      [],
      MARKS,
      FRAME_MS,
      SCALE,
      clockOf,
    )
    expect(tracks.own.map((m) => m.ratio)).toEqual([0, 1])
  })

  it('les clés et l’ordre viennent de la source — le rendu peut s’y fier', () => {
    const tracks = buildEventTracks(
      [kill({ key: 'k-a', replayMs: 12_000 }), kill({ key: 'k-b', replayMs: 30_000 })],
      [death({ key: 'd-a', replayMs: 20_000 })],
      MARKS,
      FRAME_MS,
      SCALE,
      clockOf,
    )
    // Les kills d'abord, les morts ensuite : deux passes, deux clés stables et distinctes.
    expect(tracks.own.map((m) => m.key)).toEqual(['k-a', 'k-b', 'd-a'])
    expect(tracks.own.map((m) => m.clock)).toEqual(['@12000', '@30000', '@20000'])
  })
})

describe('buildDominance — les retournements deviennent des durées', () => {
  it('AUCUN changement = AUCUNE bande (la donnée ne dit pas qui menait)', () => {
    expect(buildDominance([], SCALE)).toEqual([])
  })

  it('chaque bande court jusqu’au changement suivant, la dernière jusqu’au bout', () => {
    const spans = buildDominance(
      [
        { frame: 100, teamId: 0 },
        { frame: 250, teamId: 1 },
      ],
      SCALE,
    )
    expect(spans).toEqual([
      { key: '100-0', from: 0, to: 0.5, teamId: 0 },
      { key: '250-1', from: 0.5, to: 1, teamId: 1 },
    ])
  })

  it('un changement hors fenêtre est BORNÉ à la frise, pas rejeté', () => {
    // Le calque de score est daté par l'horloge du film : une valeur en amont du coup d'envoi
    // reste un fait du match, contrairement à une marque d'événement qu'on désignerait à côté.
    const spans = buildDominance([{ frame: 20, teamId: 1 }], SCALE)
    expect(spans).toEqual([{ key: '20-1', from: 0, to: 1, teamId: 1 }])
  })
})

function media(over: Partial<ReplayMediaItem> = {}): ReplayMediaItem {
  return {
    id: 'm1',
    kind: 'image',
    replayMs: 25_000,
    thumbUrl: '/thumb.png',
    url: '/full.png',
    ...over,
  }
}

describe('placeMedia — un clip occupe sa durée', () => {
  it('UNE CAPTURE est un instant : sa largeur est nulle', () => {
    expect(placeMedia([media()], FRAME_MS, SCALE)).toEqual([
      { item: media(), from: 0.5, to: 0.5 },
    ])
  })

  it('UN CLIP occupe exactement sa durée sur la frise', () => {
    const clip = media({ kind: 'clip', replayMs: 10_000, durationMs: 15_000 })
    const [placed] = placeMedia([clip], FRAME_MS, SCALE)
    // 15 s de clip sur 30 s de match : la moitié de la piste.
    expect(placed.from).toBe(0)
    expect(placed.to).toBe(0.5)
  })

  it('un clip qui DÉBORDE la fin est tronqué au bord, pas rejeté', () => {
    const clip = media({ kind: 'clip', replayMs: 35_000, durationMs: 60_000 })
    const [placed] = placeMedia([clip], FRAME_MS, SCALE)
    expect(placed.to).toBe(1)
  })

  it('un média HORS FENÊTRE est écarté', () => {
    expect(placeMedia([media({ replayMs: 5_000 })], FRAME_MS, SCALE)).toEqual([])
  })

  it('la piste est rendue dans l’ordre du match, quelle que soit la source', () => {
    const tard = media({ id: 'tard', replayMs: 35_000 })
    const tot = media({ id: 'tot', replayMs: 15_000 })
    expect(placeMedia([tard, tot], FRAME_MS, SCALE).map((p) => p.item.id)).toEqual(['tot', 'tard'])
  })
})

describe('la géométrie de la piste suit celle du curseur natif', () => {
  it('trackLeft réserve la demi-largeur du curseur aux deux bouts', () => {
    expect(trackLeft(0)).toBe(`calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * 0)`)
    expect(trackLeft(1)).toBe(`calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * 1)`)
  })

  it('trackLeft borne un ratio aberrant au lieu de sortir de la piste', () => {
    expect(trackLeft(-3)).toBe(trackLeft(0))
    expect(trackLeft(9)).toBe(trackLeft(1))
  })

  it('trackWidth rend une largeur POSITIVE, même sur un intervalle inversé', () => {
    expect(trackWidth(0.25, 0.75)).toBe(`calc((100% - ${THUMB_PX}px) * 0.5)`)
    expect(trackWidth(0.8, 0.2)).toBe(`calc((100% - ${THUMB_PX}px) * 0)`)
  })
})

describe('clipFrameCount — la bande d’images dit une durée', () => {
  it('une image toutes les trois secondes', () => {
    expect(clipFrameCount(18_000)).toBe(6)
  })

  it('bornée en bas (une seule vignette ne dirait pas « durée »)', () => {
    expect(clipFrameCount(0)).toBe(4)
    expect(clipFrameCount(3_000)).toBe(4)
  })

  it('bornée en haut (un long clip rendrait la bande illisible)', () => {
    expect(clipFrameCount(600_000)).toBe(12)
  })
})
