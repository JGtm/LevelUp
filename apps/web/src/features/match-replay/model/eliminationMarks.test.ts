/**
 * eliminationMarks.test.ts — LES MARQUES DE TIR À LA TÊTE ET DE FRAG PARFAIT POSÉES SUR LES LIGNES DU FIL.
 *
 * Ce qu'ils protègent : seules les marques VRAIES se posent (un faux ou un inconnu n'écrit rien) ; la jointure prend
 * le même tueur, la victime quand les deux la connaissent, la plus proche dans la fenêtre ; une élimination ne sert
 * qu'une ligne ; les lignes d'un même tueur s'apparient dans l'ordre du temps, et un groupe de nombres différents que
 * la victime ne tranche pas ne prend rien ; sans couverture des éliminations, rien n'est posé.
 */
import { describe, expect, it } from 'vitest'

import { attachEliminationMarks, ELIMINATION_JOIN_MS, type EliminationMarksSource, type MarkableKill } from './eliminationMarks'

const COVERAGE = {
  eliminations: 3, withVictim: 3, headYes: 1, headNo: 1, headUnknown: 1, headNoCounter: 1, headAmbiguous: 0, headGuarded: 0,
  headOrphanRises: 0, perfectYes: 1, perfectUnknown: 2, perfectAmbiguous: 0, perfectOrphans: 0,
}

type Row = { t: number; xuid: string; victim?: string; headshot?: boolean; perfect?: boolean }

/** Un document au pas de 100 ms ; `coverage` à `null` : aucune couverture des éliminations. */
const doc = (eliminations: Row[], coverage: object | null = COVERAGE): EliminationMarksSource =>
  ({ frameIntervalMs: 100, eliminations, coverage: coverage === null ? undefined : { eliminations: coverage } }) as never

const kill = (xuid: string, victimXuid: string, replayMs: number): MarkableKill => ({ xuid, victimXuid, replayMs })

describe('attachEliminationMarks — ce qui se pose', () => {
  it('pose le tir à la tête et le frag parfait vrais, jamais un faux ni un inconnu', () => {
    const kills = [kill('a', 'v1', 1000), kill('a', 'v2', 5000), kill('a', 'v3', 9000)]
    attachEliminationMarks(
      kills,
      doc([
        { t: 10, xuid: 'a', victim: 'v1', headshot: true, perfect: true },
        { t: 50, xuid: 'a', victim: 'v2', headshot: false },
        { t: 90, xuid: 'a', victim: 'v3' },
      ]),
    )
    expect(kills[0].fragMarks).toEqual({ headshot: true, perfect: true })
    expect(kills[1].fragMarks).toBeUndefined()
    expect(kills[2].fragMarks).toBeUndefined()
  })

  it('une ligne dont l’élimination n’a qu’une marque vraie ne porte que celle-là', () => {
    const kills = [kill('a', 'v1', 1000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', victim: 'v1', perfect: true }]))
    expect(kills[0].fragMarks).toEqual({ perfect: true })
  })

  it('un tueur ne prend pas la marque d’un autre', () => {
    const kills = [kill('b', 'v1', 1000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', victim: 'v1', headshot: true }]))
    expect(kills[0].fragMarks).toBeUndefined()
  })
})

describe('attachEliminationMarks — la jointure', () => {
  it('une victime différente écarte l’élimination, même à la même image', () => {
    const kills = [kill('a', 'v2', 1000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', victim: 'v1', headshot: true }]))
    expect(kills[0].fragMarks).toBeUndefined()
  })

  it('sans victime d’un côté, le tueur et l’instant suffisent', () => {
    const kills = [kill('a', '', 1000), kill('a', 'v9', 3000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', headshot: true }, { t: 30, xuid: 'a', headshot: true }]))
    expect(kills[0].fragMarks).toEqual({ headshot: true })
    expect(kills[1].fragMarks).toEqual({ headshot: true })
  })

  it('l’instant compte : au-delà de la fenêtre, aucune marque', () => {
    const ms = 1000
    const dans = [kill('a', 'v1', ms + ELIMINATION_JOIN_MS)]
    attachEliminationMarks(dans, doc([{ t: ms / 100, xuid: 'a', victim: 'v1', headshot: true }]))
    expect(dans[0].fragMarks).toEqual({ headshot: true })
    const hors = [kill('a', 'v1', ms + ELIMINATION_JOIN_MS + 1)]
    attachEliminationMarks(hors, doc([{ t: ms / 100, xuid: 'a', victim: 'v1', headshot: true }]))
    expect(hors[0].fragMarks).toBeUndefined()
  })

  it('une élimination ne sert qu’une ligne : celle dont la victime concorde, quand elle tranche', () => {
    const kills = [kill('a', 'v1', 1200), kill('a', 'v2', 1000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', victim: 'v2', headshot: true }]))
    expect(kills[0].fragMarks).toBeUndefined()
    expect(kills[1].fragMarks).toEqual({ headshot: true })
  })

  it('deux lignes pour une élimination et aucune victime pour trancher : aucune marque', () => {
    const kills = [kill('a', '', 1200), kill('a', '', 1000)]
    attachEliminationMarks(kills, doc([{ t: 10, xuid: 'a', headshot: true }]))
    expect(kills[0].fragMarks).toBeUndefined()
    expect(kills[1].fragMarks).toBeUndefined()
  })

  it('même tueur, deux frags à 200 ms, fil et tableau décalés de 150 ms : chaque ligne prend sa marque, sans croisement', () => {
    // Lignes à 1000 et 1200 ms ; tableau à 850 et 1050 ms : par la distance, la 2e ligne (50 ms) volerait la 1re élimination.
    const kills = [kill('a', '', 1000), kill('a', '', 1200)]
    attachEliminationMarks(kills, doc([{ t: 8.5, xuid: 'a', headshot: true }, { t: 10.5, xuid: 'a', perfect: true }]))
    expect(kills[0].fragMarks).toEqual({ headshot: true })
    expect(kills[1].fragMarks).toEqual({ perfect: true })
  })

  it('une ligne à deux candidats à égale distance ne prend rien', () => {
    const kills = [kill('a', '', 1000)]
    attachEliminationMarks(kills, doc([{ t: 9, xuid: 'a', headshot: true }, { t: 11, xuid: 'a', headshot: true }]))
    expect(kills[0].fragMarks).toBeUndefined()
  })

  it('deux frags du même tueur dans la même seconde, victimes nommées : chacun sa marque', () => {
    const kills = [kill('a', 'v1', 1000), kill('a', 'v2', 1100)]
    attachEliminationMarks(
      kills,
      doc([{ t: 10, xuid: 'a', victim: 'v1', headshot: true }, { t: 11, xuid: 'a', victim: 'v2', perfect: true }]),
    )
    expect(kills[0].fragMarks).toEqual({ headshot: true })
    expect(kills[1].fragMarks).toEqual({ perfect: true })
  })
})

describe('attachEliminationMarks — ce que le document ne permet pas d’établir', () => {
  const rows: Row[] = [{ t: 10, xuid: 'a', victim: 'v1', headshot: true }]

  it('sans couverture des éliminations, rien n’est posé', () => {
    const kills = [kill('a', 'v1', 1000)]
    attachEliminationMarks(kills, doc(rows, null))
    expect(kills[0].fragMarks).toBeUndefined()
  })

  it('sans document, ou sans ligne, rien n’est posé et rien ne casse', () => {
    const kills = [kill('a', 'v1', 1000)]
    attachEliminationMarks(kills, null)
    attachEliminationMarks(kills, doc([]))
    expect(kills[0].fragMarks).toBeUndefined()
  })
})
