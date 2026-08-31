/**
 * Tests — objectiveMark (qui porte l'objectif, à une image donnée).
 *
 * CE QU'ILS PROTÈGENT, et pourquoi chacun compte :
 *  - la frontière des ÉTATS de drapeau : `dropped` et `home` portent le xuid du DERNIER
 *    porteur, et les lire comme un portage collerait le drapeau à un joueur qui ne l'a plus ;
 *  - la nature d'ÉVÉNEMENT de la prise de base : elle se tient quelques secondes puis s'éteint,
 *    parce que la donnée n'attribue que l'instant — jamais une durée de capture ;
 *  - la GARDE D'HORLOGE sur cet événement : sans origine résolue, la marque s'allumerait sur la
 *    mauvaise image, et muet vaut mieux que faux ;
 *  - le genre `hill`, prêt avant sa source : il passe par le MÊME résolveur de périodes que le
 *    crâne et le VIP, et ce test est ce qui l'empêche de diverger d'ici le décodage de KOTH.
 */
import { describe, expect, it } from 'vitest'

import { objectiveMarkAt, objectiveMarkFromPeriods } from './objectiveMark'
import { testReplayDoc } from './test/testDoc'

const ME = 'x1'

describe('objectiveMark — les portages, un état qui dure', () => {
  it('le drapeau porté marque la fiche sur tout son intervalle, bornes comprises', () => {
    const doc = testReplayDoc({
      flagCarries: [{ team: 0, spans: [{ state: 'carried', xuid: ME, t0: 10, t1: 20, x: 0, y: 0 }] }],
    })
    expect(objectiveMarkAt(doc, ME, 9)).toBeNull()
    expect(objectiveMarkAt(doc, ME, 10)).toBe('flag')
    expect(objectiveMarkAt(doc, ME, 20)).toBe('flag')
    expect(objectiveMarkAt(doc, ME, 21)).toBeNull()
  })

  it('`carried_open` est un portage — le porteur est visible, il tient quand même le drapeau', () => {
    const doc = testReplayDoc({
      flagCarries: [{ team: 0, spans: [{ state: 'carried_open', xuid: ME, t0: 5, t1: 8, x: 0, y: 0 }] }],
    })
    expect(objectiveMarkAt(doc, ME, 6)).toBe('flag')
  })

  it('`dropped` et `home` ne sont PAS des portages, malgré leur xuid de dernier porteur', () => {
    const doc = testReplayDoc({
      flagCarries: [{
        team: 0,
        spans: [
          { state: 'dropped', xuid: ME, t0: 10, t1: 20, x: 0, y: 0 },
          { state: 'home', xuid: ME, t0: 21, t1: 30, x: 0, y: 0 },
        ],
      }],
    })
    expect(objectiveMarkAt(doc, ME, 15)).toBeNull()
    expect(objectiveMarkAt(doc, ME, 25)).toBeNull()
  })

  it('le crâne et la couronne VIP marquent leur porteur, et lui seul', () => {
    const skull = testReplayDoc({ skullCarries: [{ xuid: ME, t0: 4, t1: 9, closed: true }] })
    expect(objectiveMarkAt(skull, ME, 5)).toBe('skull')
    expect(objectiveMarkAt(skull, 'autre', 5)).toBeNull()

    const vip = testReplayDoc({ vipCrown: [{ xuid: ME, t0: 4, t1: 9, closed: true }] })
    expect(objectiveMarkAt(vip, ME, 5)).toBe('vip')
  })

  it('un porteur qui vient de prendre une base garde son objet : l’état prime sur l’instant', () => {
    const doc = testReplayDoc({
      frameCount: 200,
      flagCarries: [{ team: 0, spans: [{ state: 'carried', xuid: ME, t0: 10, t1: 40, x: 0, y: 0 }] }],
      objectives: [{ stat: 'zone_captures', t: 20, timeMs: 2_000, xuid: ME }],
    })
    expect(objectiveMarkAt(doc, ME, 20)).toBe('flag')
  })
})

describe('objectiveMark — la prise de base, un instant tenu', () => {
  const doc = (over = {}) => testReplayDoc({
    frameCount: 200,
    objectives: [{ stat: 'zone_captures', t: 30, timeMs: 3_000, xuid: ME }],
    ...over,
  })

  it('s’allume à l’instant de la prise, tient quelques secondes, puis s’éteint', () => {
    const d = doc()
    expect(objectiveMarkAt(d, ME, 29)).toBeNull()
    expect(objectiveMarkAt(d, ME, 30)).toBe('zone')
    expect(objectiveMarkAt(d, ME, 31)).toBe('zone')
    expect(objectiveMarkAt(d, ME, 300)).toBeNull()
  })

  it('la sécurisation d’une base marque aussi son auteur', () => {
    const d = doc({ objectives: [{ stat: 'zone_secures', t: 30, timeMs: 3_000, xuid: ME }] })
    expect(objectiveMarkAt(d, ME, 30)).toBe('zone')
  })

  it('une action qui n’est pas une action de zone ne marque rien', () => {
    const d = doc({ objectives: [{ stat: 'kills', t: 30, timeMs: 3_000, xuid: ME }] })
    expect(objectiveMarkAt(d, ME, 30)).toBeNull()
  })

  it('SANS ORIGINE RÉSOLUE, aucune marque : l’écart d’horloge est inconnu', () => {
    // Même garde que l'onde de capture du drapeau : les `objectives[]` sont datés par
    // l'horloge du FILM, et sans recalage la marque s'allumerait sur la mauvaise image.
    const d = testReplayDoc({
      // `originResolved: false` ET aucun `originMs` : le recalage n'a pas eu lieu (même
      // fixture que flagCaptureFx.test — la garde est la même, le `as never` aussi : le type
      // de transport exige une couverture complète, dont rien ici ne dépend).
      coverage: { originResolved: false } as never,
      objectives: [{ stat: 'zone_captures', t: 30, timeMs: 3_000, xuid: ME }],
    })
    expect(objectiveMarkAt(d, ME, 30)).toBeNull()
  })
})

describe("objectiveMark — l'explosion de la bombe, un instant tenu", () => {
  const doc = (over = {}) => testReplayDoc({
    frameCount: 200,
    objectives: [{ stat: 'bomb_detonations', t: 30, timeMs: 3_000, xuid: ME }],
    ...over,
  })

  it("s'allume à l'instant de l'explosion, tient quelques secondes, puis s'éteint", () => {
    const d = doc()
    expect(objectiveMarkAt(d, ME, 29)).toBeNull()
    expect(objectiveMarkAt(d, ME, 30)).toBe('bomb')
    expect(objectiveMarkAt(d, ME, 31)).toBe('bomb')
    expect(objectiveMarkAt(d, ME, 300)).toBeNull()
  })

  it("ne marque que son auteur — le point de mode d'Assaut est attribué", () => {
    expect(objectiveMarkAt(doc(), 'autre', 30)).toBeNull()
  })

  it('SANS ORIGINE RÉSOLUE, aucune marque : la même garde que la prise de base', () => {
    const d = testReplayDoc({
      coverage: { originResolved: false } as never,
      objectives: [{ stat: 'bomb_detonations', t: 30, timeMs: 3_000, xuid: ME }],
    })
    expect(objectiveMarkAt(d, ME, 30)).toBeNull()
  })
})

describe('objectiveMark — KOTH, prêt avant sa source', () => {
  it('le résolveur de périodes vaut pour la colline comme pour le crâne et le VIP', () => {
    // Le document ne publie AUCUNE occupation de colline attribuée à ce jour (les
    // emplacements de stats `hill` ne sont pas nommés) : c'est le résolveur partagé qui est
    // vérifié ici, celui-là même que `carrySourcesOf` branchera le jour venu.
    const periods = [{ xuid: ME, t0: 12, t1: 18 }]
    expect(objectiveMarkFromPeriods(periods, ME, 11)).toBe(false)
    expect(objectiveMarkFromPeriods(periods, ME, 12)).toBe(true)
    expect(objectiveMarkFromPeriods(periods, ME, 18)).toBe(true)
    expect(objectiveMarkFromPeriods(periods, ME, 19)).toBe(false)
    expect(objectiveMarkFromPeriods(periods, 'autre', 15)).toBe(false)
  })

  it('et un document sans aucune donnée d’objectif ne marque personne', () => {
    expect(objectiveMarkAt(testReplayDoc(), ME, 10)).toBeNull()
  })
})
