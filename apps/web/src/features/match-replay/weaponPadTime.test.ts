/**
 * Tests — weaponPadTime : LES TROIS ÉTATS D'UN SOCLE ET SON COMPTE À REBOURS.
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - PLEIN de `t0` à `tLow`, INCERTAIN jusqu'à `tHigh`, VIDE ensuite ; un socle « jamais vidé »
 *    (`tHigh <= tLow`) reste PLEIN jusqu'au bout — aucune absence n'y a été prouvée ;
 *  - LE COMPTE À REBOURS VISE D'ABORD LA PROCHAINE APPARITION MESURÉE (D3, 2026-08-27) : le
 *    rejeu connaît la suite du film, donc un trou refermé par une apparition se compte EXACTEMENT
 *    et se dit « mesuré ». Le cycle ne sert plus que de repli pour le DERNIER trou ;
 *  - SANS SOURCE, RIEN : ni chiffre, ni tiret. C'était le défaut signalé — le compte n'existait
 *    qu'avec un cycle établi, donc jamais sur la moitié des socles.
 *
 * Extrait de `weaponPadsLayer.test.ts` le 2026-08-27, en même temps que le module mesuré.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayWeaponPadReady } from './replayNormalize'
import { padOccupancyAt, padRespawnAt, padStateAt } from './weaponPadTime'

/** Une image de 100 ms : 10 images = 1 s, ce qui rend les comptes lisibles à l'œil nu. */
const FRAME_MS = 100

/** Le cycle des témoins : 40 s pile. */
const CYCLE = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }

function pad(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: 5,
    y: 5,
    weapon: '0x0A1992BC',
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 120 }],
    ...over,
  }
}

/** Deux occupations : le socle se vide à 120 et une arme revient à 300 (trou MÉDIAN). */
function deuxOccupations(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return pad({
    spawns: [0, 300],
    presence: [
      { t0: 0, tLow: 100, tHigh: 120 },
      { t0: 300, tLow: 400, tHigh: 420 },
    ],
    ...over,
  })
}

describe('padStateAt — les trois états, et leurs frontières', () => {
  const p = pad()

  it('PLEIN de l’apparition au dernier instant prouvé', () => {
    expect(padStateAt(p, 0)).toBe('full')
    expect(padStateAt(p, 99)).toBe('full')
  })

  it('INCERTAIN entre la dernière preuve de présence et la première preuve d’absence', () => {
    expect(padStateAt(p, 100)).toBe('uncertain')
    expect(padStateAt(p, 119)).toBe('uncertain')
  })

  it('VIDE dès que l’absence est prouvée', () => {
    expect(padStateAt(p, 120)).toBe('empty')
    expect(padStateAt(p, 500)).toBe('empty')
  })

  it('VIDE avant la première apparition : le socle n’a rien porté du tout', () => {
    const tardif = pad({ spawns: [200], presence: [{ t0: 200, tLow: 300, tHigh: 320 }] })
    expect(padStateAt(tardif, 0)).toBe('empty')
    expect(padOccupancyAt(tardif, 0)).toBeNull()
    expect(padStateAt(tardif, 200)).toBe('full')
  })

  it('une RÉAPPARITION reprend à plein : c’est la dernière occupation qui gouverne', () => {
    const deux = deuxOccupations()
    expect(padStateAt(deux, 250)).toBe('empty')
    expect(padStateAt(deux, 300)).toBe('full')
    expect(padOccupancyAt(deux, 350)?.t0).toBe(300)
  })

  it('un socle JAMAIS VIDÉ reste PLEIN jusqu’au bout : aucune absence n’a été prouvée', () => {
    // Cas mesuré sur `bcb6d393` : 8 occupations sur 28 s'achèvent ainsi (tHigh = tLow = fin).
    const jamais = pad({ presence: [{ t0: 0, tLow: 3464, tHigh: 3464 }] })
    expect(padStateAt(jamais, 3463)).toBe('full')
    expect(padStateAt(jamais, 3464)).toBe('full')
    expect(padRespawnAt(jamais, 3464, FRAME_MS)).toBeNull()
  })
})

describe('padRespawnAt — la MESURE d’abord, le cycle en repli (D3)', () => {
  it('TROU MÉDIAN : le compte vise la prochaine apparition VUE, et il est EXACT', () => {
    const p = deuxOccupations()
    // Le socle se vide à 120, l'arme revient à 300 : 180 images = 18,0 s, au dixième près.
    expect(padRespawnAt(p, 120, FRAME_MS)).toEqual({ seconds: 18, measured: true })
    expect(padRespawnAt(p, 200, FRAME_MS)).toEqual({ seconds: 10, measured: true })
    expect(padRespawnAt(p, 299, FRAME_MS)?.seconds).toBeCloseTo(0.1, 6)
  })

  it('TROU MÉDIAN : la mesure l’emporte sur le cycle, même quand les deux existent', () => {
    // Le cycle dirait 40 s au moment du vidage ; le film en montre 18. C'est le film qui gagne.
    const p = deuxOccupations({ cycle: CYCLE })
    expect(padRespawnAt(p, 120, FRAME_MS)).toEqual({ seconds: 18, measured: true })
  })

  it('DERNIER TROU avec cycle : le compte est ATTENDU, et il part de tHigh', () => {
    const p = pad({ cycle: CYCLE })
    expect(padRespawnAt(p, 120, FRAME_MS)).toEqual({ seconds: 40, measured: false })
    expect(padRespawnAt(p, 320, FRAME_MS)?.seconds).toBeCloseTo(20, 6)
    expect(padRespawnAt(p, 320, FRAME_MS)?.measured).toBe(false)
  })

  it('DERNIER TROU sans cycle : RIEN — c’est là que le compte s’arrête d’exister', () => {
    const p = pad()
    expect(p.cycle).toBeUndefined()
    for (const f of [120, 200, 5000]) expect(padRespawnAt(p, f, FRAME_MS)).toBeNull()
  })

  it('SOCLE NON VIDE : rien, l’attente n’a pas commencé', () => {
    const p = deuxOccupations({ cycle: CYCLE })
    expect(padRespawnAt(p, 50, FRAME_MS)).toBeNull()
    expect(padRespawnAt(p, 110, FRAME_MS)).toBeNull()
    expect(padRespawnAt(p, 350, FRAME_MS)).toBeNull()
  })

  it('AVANT la première apparition : rien à attendre, le socle n’a jamais rien porté', () => {
    const tardif = pad({ spawns: [200], presence: [{ t0: 200, tLow: 300, tHigh: 320 }] })
    expect(padStateAt(tardif, 0)).toBe('empty')
    expect(padRespawnAt(tardif, 0, FRAME_MS)).toBeNull()
  })

  it('le compte ATTENDU épuisé s’efface : jamais un nombre négatif ni un zéro qui traîne', () => {
    const p = pad({ cycle: CYCLE })
    expect(padRespawnAt(p, 520, FRAME_MS)).toBeNull()
    expect(padRespawnAt(p, 900, FRAME_MS)).toBeNull()
  })

  it('une durée d’image inconnue n’invente aucun compte, mesuré ou attendu', () => {
    expect(padRespawnAt(pad({ cycle: CYCLE }), 200, 0)).toBeNull()
    expect(padRespawnAt(deuxOccupations(), 200, 0)).toBeNull()
  })
})
