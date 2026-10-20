/**
 * Tests — placementRift : LA FAILLE ET LE PASSAGE du translocateur, au contexte enregistreur.
 *
 * Trois choses se vérifient ici : le LIEN de téléportation (sa fenêtre, ses encres, ses deux
 * bouts), la FAILLE dessinée par ses stations (jamais par une pose), et le fait qu'un point
 * n'est JAMAIS PEINT DEUX FOIS (revue du 2026-09-03, constat K3). L'ÉCLAT de téléportation — la
 * faille qui s'ouvre et son onde aux deux bouts — y est mesuré à ses arcs et à ses rayons.
 */
import { describe, expect, it } from 'vitest'

import { TRANSLOCATION_FLASH_S } from '../model/playerCardFx'
import { recordingContext } from '../test/recordingContext'
import { INK, projected, TIME, VIEW } from '../test/placementFixtures'
import { drawRiftLayer, RIFT_HALF_HEIGHT_PX, RIFT_LINK_MS, type RiftScene } from './placementRift'

function draw(scene: RiftScene, frame: number, over: Partial<typeof TIME> = {}) {
  const { ctx, ops } = recordingContext()
  drawRiftLayer(ctx, scene, VIEW, { ...TIME, ...over, frame }, INK.rift)
  return ops
}

/** Les ondes : des arcs posés SUR un point projeté (le halo d'une faille, lui, est translaté). */
function ondes(ops: ReturnType<typeof draw>, at: { x: number; y: number }) {
  return ops.filter(
    (o) =>
      o.op === 'arc' &&
      Math.abs((o.args[0] as number) - at.x) < 1e-6 &&
      Math.abs((o.args[1] as number) - at.y) < 1e-6,
  )
}

describe('la durée de l’éclat', () => {
  it('est celle de l’éclat de la fiche : la carte et la fiche datent le même saut', () => {
    expect(RIFT_LINK_MS).toBe(TRANSLOCATION_FLASH_S * 1000)
  })
})

describe('le LIEN de téléportation', () => {
  const passage = { slot: 3, frame: 48, from: { x: 1, y: 1 }, to: { x: 9, y: 9 } }
  const scene: RiftScene = { teleports: [passage], rifts: [] }

  /** Les opérations d un tracé de lien : le pointillé est sa signature. */
  function liens(frame: number) {
    return draw(scene, frame).filter(
      (o) => o.op === 'setLineDash' && Array.isArray(o.args[0]) && o.args[0].length > 0,
    )
  }

  it('se trace pendant les 1,2 s qui suivent le passage, et pas au-delà', () => {
    // frameMs vaut 100 dans la fixture : la fenêtre couvre les frames 48 à 59. La frame 60
    // tombe EXACTEMENT sur les 1200 ms — l effacement y est complet, donc plus rien n est tracé.
    expect(liens(48)).toHaveLength(1)
    expect(liens(59)).toHaveLength(1)
    expect(liens(60)).toHaveLength(0)
    expect(liens(61)).toHaveLength(0)
  })

  it('ne se trace pas AVANT le passage — un lien n annonce rien', () => {
    expect(liens(47)).toHaveLength(0)
  })

  it('porte les encres de la faille, jamais la couleur d équipe du joueur', () => {
    const encres = draw(scene, 49)
      .filter((o) => o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres).toContain('faille-coeur')
    expect(encres).not.toContain('equipe')
  })

  it('relie les DEUX positions mesurées : le départ et l arrivée', () => {
    const ops = draw(scene, 49)
    const depart = projected(1, 1)
    const arrivee = projected(9, 9)
    const courbe = ops.find((o) => o.op === 'quadraticCurveTo' && o.args[2] === arrivee.x)
    expect(courbe).toBeDefined()
    expect(ops.some((o) => o.op === 'moveTo' && o.args[0] === depart.x && o.args[1] === depart.y)).toBe(true)
  })

  it('sans passage, le calque n émet rien', () => {
    expect(draw({ teleports: [], rifts: [] }, 49)).toHaveLength(0)
  })
})

describe('l’ÉCLAT : une onde à CHAQUE bout, datée par l’événement', () => {
  const passage = { slot: 3, frame: 48, from: { x: 1, y: 1 }, to: { x: 9, y: 9 } }
  const scene: RiftScene = { teleports: [passage], rifts: [] }
  const depart = projected(1, 1)
  const arrivee = projected(9, 9)

  it('une onde au départ ET une à l’arrivée, pendant toute la fenêtre', () => {
    for (const frame of [48, 53, 59]) {
      const ops = draw(scene, frame)
      expect(ondes(ops, depart), `départ, image ${frame}`).toHaveLength(1)
      expect(ondes(ops, arrivee), `arrivée, image ${frame}`).toHaveLength(1)
    }
  })

  it('aucune onde avant le passage ni après la fenêtre', () => {
    expect(ondes(draw(scene, 47), arrivee)).toHaveLength(0)
    expect(ondes(draw(scene, 60), arrivee)).toHaveLength(0)
  })

  it('l’onde S’OUVRE : son rayon croît avec le temps', () => {
    const rayon = (frame: number) => ondes(draw(scene, frame), arrivee)[0].args[2] as number
    expect(rayon(48)).toBeLessThan(rayon(53))
    expect(rayon(53)).toBeLessThan(rayon(59))
  })

  it('mouvement réduit : le rayon est FIXE, l’onde ne fait que s’effacer', () => {
    const rayon = (frame: number) => ondes(draw(scene, frame, { reducedMotion: true }), arrivee)[0].args[2] as number
    expect(rayon(48)).toBe(rayon(59))
  })

  it('le rayon part de la faille : il se mesure en demi-hauteurs de faille', () => {
    const r = ondes(draw(scene, 48), arrivee)[0].args[2] as number
    expect(r).toBeGreaterThan(0.5 * RIFT_HALF_HEIGHT_PX)
    expect(r).toBeLessThan(3 * RIFT_HALF_HEIGHT_PX)
  })

  it('les failles s’ouvrent : plus grandes à l’instant du saut qu’à la fin de l’effacement', () => {
    const demiHauteur = (frame: number) => {
      const ops = draw(scene, frame)
      const m = ops.filter((o) => o.op === 'moveTo' && o.args[0] === arrivee.x)
      return arrivee.y - (m[0].args[1] as number)
    }
    expect(demiHauteur(48)).toBeGreaterThan(demiHauteur(58))
  })
})

describe('LA FAILLE, dessinée par ses STATIONS et non par une pose', () => {
  /** Une station : la faille est en (5, 5) des images 20 à 40, et nulle part ailleurs. */
  const station = { slot: 3, t0: 20, t1: 40, x: 5, y: 5 }
  const scene: RiftScene = { teleports: [], rifts: [station] }

  /** Le halo elliptique de la faille signe son tracé (cf. drawRift). */
  const halos = (frame: number) => draw(scene, frame).filter((o) => o.op === 'createRadialGradient')

  it('rien AVANT le premier échange : la pose du translocateur est illisible', () => {
    expect(halos(19)).toHaveLength(0)
  })

  it('elle se dessine pendant sa station, aux encres FIXES de la faille', () => {
    expect(halos(20)).toHaveLength(1)
    expect(halos(40)).toHaveLength(1)
    const encres = draw(scene, 30)
      .filter((o) => o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres).toContain('faille-bord')
    expect(encres).not.toContain('equipe')
  })

  it('rien APRÈS la fin mesurée : jamais à demeure', () => {
    expect(halos(41)).toHaveLength(0)
  })

  it('elle se dessine à sa position monde, projetée comme le reste du calque', () => {
    const c = projected(5, 5)
    const ops = draw(scene, 30)
    expect(ops.some((o) => o.op === 'translate' && o.args[0] === c.x && o.args[1] === c.y)).toBe(true)
  })
})

/**
 * K3 (revue ronde 1, 2026-09-03) — UN POINT N'EST JAMAIS PEINT DEUX FOIS.
 *
 * La station et le bout `from` du lien qui l'ouvre sont LE MÊME objet : même vie, même image
 * d'échange, même point. Les peindre tous deux composait le halo deux fois. Le halo est ici le
 * témoin : un `createRadialGradient` par faille peinte. L'onde, elle, n'a pas de halo : elle
 * se pose sur la station sans la redoubler.
 */
describe('LA FAILLE ET SON LIEN ensemble — le point quitté n est peint qu une fois', () => {
  /** Le saut qui ouvre la station : même vie, même image, même point de départ. */
  const lien = { slot: 3, frame: 20, from: { x: 5, y: 5 }, to: { x: 9, y: 9 } }
  const station = { slot: 3, t0: 20, t1: 400, x: 5, y: 5 }

  const halos = (frame: number, scene: RiftScene) =>
    draw(scene, frame).filter((o) => o.op === 'createRadialGradient').length

  it('pendant le lien : DEUX failles peintes — la station au départ, le bout du lien à l arrivée', () => {
    // Sans la correction : TROIS (station + les deux bouts du lien), le départ compté deux fois.
    expect(halos(21, { teleports: [lien], rifts: [station] })).toBe(2)
  })

  it('l’onde du départ se pose sur la station, sans second halo', () => {
    const ops = draw({ teleports: [lien], rifts: [station] }, 21)
    expect(ondes(ops, projected(5, 5))).toHaveLength(1)
  })

  it('après le lien : la station SEULE reste, à pleine taille', () => {
    // frameMs = 100 dans la fixture : le lien couvre 20..31, la frame 32 est au-delà.
    expect(halos(32, { teleports: [lien], rifts: [station] })).toBe(1)
  })

  it('un lien SANS station à son départ garde ses deux bouts (artefact sans positions ailleurs)', () => {
    expect(halos(21, { teleports: [lien], rifts: [] })).toBe(2)
  })

  it('l appariement se fait sur (vie, image), pas sur des coordonnées : une AUTRE vie garde son bout', () => {
    const autreVie = { ...station, slot: 9 }
    expect(halos(21, { teleports: [lien], rifts: [autreVie] })).toBe(3)
  })
})
