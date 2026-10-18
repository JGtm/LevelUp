/**
 * sequenceLecture.logic — la lecture animée d'une séquence : bornes et avance de la tête de lecture, étape
 * en vigueur, position interpolée entre les échantillons (jamais à travers une lacune), scène de l'instant,
 * horloge écrite.
 */
import { describe, expect, it } from 'vitest'

import { SEQUENCE } from './mort.fixtures'
import {
  avancer,
  bornesDeLecture,
  etapeALInstant,
  etapeDeLaMort,
  horlogeAffichee,
  PAS_MAX_MS,
  positionALInstant,
  sceneALInstant,
  sceneDeSequence,
} from './sequenceLecture.logic'

const ETAPES = SEQUENCE.etapes ?? []
const acteur = (role: string) => SEQUENCE.acteurs?.find((a) => a.role === role)

describe('bornes et avance de la tête de lecture', () => {
  it('de la première à la dernière étape servie ; rien à lire sans deux étapes', () => {
    expect(bornesDeLecture(SEQUENCE)).toEqual({ debutMs: -4000, finMs: 3000 })
    expect(bornesDeLecture({ ...SEQUENCE, etapes: ETAPES.slice(0, 1) })).toBeNull()
    expect(bornesDeLecture({ ...SEQUENCE, etapes: [] })).toBeNull()
  })

  it('avance en temps réel, par pas bornés', () => {
    const bornes = { debutMs: -4000, finMs: 3000 }
    expect(avancer(-4000, 16, bornes)).toEqual({ offsetMs: -3984, fini: false })
    // Un onglet resté en arrière-plan ne fait pas sauter la lecture.
    expect(avancer(-4000, 5000, bornes)).toEqual({ offsetMs: -4000 + PAS_MAX_MS, fini: false })
    expect(avancer(-4000, -50, bornes).offsetMs).toBe(-4000)
  })

  it('s’arrête sur la dernière étape, sans la dépasser', () => {
    const bornes = { debutMs: -4000, finMs: 3000 }
    expect(avancer(2950, 100, bornes)).toEqual({ offsetMs: 3000, fini: true })
    expect(avancer(3000, 16, bornes)).toEqual({ offsetMs: 3000, fini: true })
  })

  it('une tête avant le début est ramenée au début', () => {
    expect(avancer(-9000, 16, { debutMs: -4000, finMs: 3000 }).offsetMs).toBe(-3984)
  })

  it('la lecture complète dure la durée réelle : 7 s à 60 images par seconde', () => {
    const bornes = { debutMs: -4000, finMs: 3000 }
    let t = bornes.debutMs
    let temps = 0
    for (let fini = false; !fini; ) {
      const pas = avancer(t, 1000 / 60, bornes)
      t = pas.offsetMs
      fini = pas.fini
      temps += 1000 / 60
    }
    expect(temps).toBeGreaterThan(6990)
    expect(temps).toBeLessThan(7020)
  })
})

describe('étape en vigueur', () => {
  it('la dernière étape dont le décalage est atteint', () => {
    expect(etapeALInstant(SEQUENCE, -4000)).toBe(0)
    expect(etapeALInstant(SEQUENCE, -1)).toBe(3)
    expect(etapeALInstant(SEQUENCE, 0)).toBe(4)
    expect(etapeALInstant(SEQUENCE, 2999)).toBe(6)
    expect(etapeALInstant(SEQUENCE, 3000)).toBe(7)
    expect(etapeALInstant(SEQUENCE, 9000)).toBe(7)
    expect(etapeALInstant(SEQUENCE, -9000)).toBe(0)
    expect(etapeALInstant({ ...SEQUENCE, etapes: [] }, 0)).toBeNull()
  })

  it('l’étape de la mort est celle à 0 ms', () => {
    expect(etapeDeLaMort(SEQUENCE)).toBe(4)
    expect(etapeDeLaMort({ ...SEQUENCE, etapes: ETAPES.filter((e) => e.offset_ms !== 0) })).toBeNull()
  })
})

describe('horloge écrite', () => {
  it('l’instant du rejeu de la mort plus le décalage, plancher 0', () => {
    expect(horlogeAffichee(SEQUENCE, 0)).toBe(440_000)
    expect(horlogeAffichee(SEQUENCE, -4000)).toBe(436_000)
    expect(horlogeAffichee(SEQUENCE, 1500)).toBe(441_500)
    const tot = { ...SEQUENCE, etapes: ETAPES.map((e) => (e.offset_ms === 0 ? { ...e, instant_affiche_ms: 2000 } : e)) }
    expect(horlogeAffichee(tot, -4000)).toBe(0)
  })

  it('coup d’envoi inconnu : pas d’instant, l’appelant écrit le décalage', () => {
    const sans = { ...SEQUENCE, etapes: ETAPES.map((e) => ({ ...e, instant_affiche_ms: undefined })) }
    expect(horlogeAffichee(sans, 0)).toBeNull()
  })
})

describe('position interpolée', () => {
  const victime = acteur('victime')?.points ?? []
  const tueur = acteur('tueur')?.points ?? []

  it('un échantillon exact est rendu tel quel', () => {
    expect(positionALInstant(victime, -4000)).toEqual({ x: 0, y: 10 })
  })

  it('entre deux échantillons consécutifs : l’interpolation linéaire', () => {
    // -4000 -> x = 0 ; -3800 -> x = 0,5 : à mi-chemin, 0,25.
    expect(positionALInstant(victime, -3900)).toEqual({ x: 0.25, y: 10 })
    expect(positionALInstant(tueur, 500)).toEqual({ x: 25, y: 10 })
  })

  it('jamais à travers une lacune : le second échantillon porte une coupure', () => {
    expect(positionALInstant(tueur, -1500)).toBeNull()
    expect(positionALInstant(tueur, -1000)).toEqual({ x: 25, y: 10 })
  })

  it('avant le premier et après le dernier échantillon : inconnue', () => {
    expect(positionALInstant(victime, -4100)).toBeNull()
    expect(positionALInstant(tueur, 1500)).toBeNull()
    expect(positionALInstant([], 0)).toBeNull()
  })
})

describe('scène de l’instant', () => {
  it('entre deux étapes : les acteurs interpolés, le cercle sur la victime, plein ou creux selon l’étape en vigueur', () => {
    const s = sceneALInstant(SEQUENCE, -3900)
    expect(s.marques.find((m) => m.role === 'mort')?.pt).toEqual({ x: 0.25, y: 10 })
    expect(s.cercle).toEqual({ centre: { x: 0.25, y: 10 }, rayonM: 18 })
    // Étape en vigueur : −4 s, personne dans le rayon.
    expect(s.marques.find((m) => m.role === 'coequipier')?.plein).toBe(false)
  })

  it('après l’instant 0 : le mort reste au lieu de sa mort et le coéquipier du rayon est plein', () => {
    const s = sceneALInstant(SEQUENCE, 500)
    expect(s.marques.find((m) => m.role === 'mort')?.pt).toEqual({ x: 10, y: 10 })
    expect(s.cercle?.centre).toEqual({ x: 10, y: 10 })
    expect(s.marques.find((m) => m.role === 'coequipier')?.plein).toBe(true)
  })

  it('un acteur dans une lacune n’est pas peint, puis réapparaît', () => {
    expect(sceneALInstant(SEQUENCE, -1500).marques.some((m) => m.role === 'tueur')).toBe(false)
    expect(sceneALInstant(SEQUENCE, -1000).marques.some((m) => m.role === 'tueur')).toBe(true)
  })

  it('la traîne se prolonge jusqu’à la tête interpolée et se coupe à la lacune', () => {
    // Tueur à 500 ms : échantillons −3000..0 (coupure à −1000), queue interpolée à −3500, tête à 500.
    const tueur = sceneALInstant(SEQUENCE, 500).traines.find((x) => x.role === 'tueur')
    expect(tueur?.segments.map((seg) => seg.length)).toEqual([3, 3])
    // Victime à −3900 : un échantillon (−4000) et la tête interpolée.
    const victime = sceneALInstant(SEQUENCE, -3900).traines.find((x) => x.role === 'mort')
    expect(victime?.segments[0]).toEqual([{ x: 0, y: 10 }, { x: 0.25, y: 10 }])
  })

  it('à un décalage d’étape, le cercle est exactement celui que le serveur publie', () => {
    ETAPES.forEach((e, i) => {
      expect(sceneALInstant(SEQUENCE, e.offset_ms).cercle?.centre).toEqual(e.centre)
      expect(sceneDeSequence(SEQUENCE, i).cercle?.centre).toEqual(e.centre)
    })
    expect(sceneDeSequence(SEQUENCE, 99).marques).toEqual([])
  })

  it('une séquence sans étape ne se peint pas', () => {
    expect(sceneALInstant({ ...SEQUENCE, etapes: [] }, 0).marques).toEqual([])
  })
})
