/**
 * moyenneGlissante (Séries temporelles, coordination) avant et après son passage sur `movingMean`
 * (étape 7.8) : l'ancien calcul est gardé ici comme référence, mêmes nombres au bit près.
 */
import { describe, expect, it } from 'vitest'

import { FENETRE_TENDANCE, moyenneGlissante, type SerieDeSoirees } from './timeseriesCoordination.logic'

function ancienne(serie: SerieDeSoirees): (number | null)[] {
  return serie.valuesPct.map((_, i) => {
    const fenetre: number[] = []
    for (let j = Math.max(0, i - FENETRE_TENDANCE + 1); j <= i; j += 1) {
      const v = serie.valuesPct[j]
      if (v != null && !serie.echantillonFaible[j]) fenetre.push(v)
    }
    if (fenetre.length < FENETRE_TENDANCE) return null
    return fenetre.reduce((a, b) => a + b, 0) / fenetre.length
  })
}

function serie(values: (number | null)[], faible: boolean[]): SerieDeSoirees {
  return { valuesPct: values, echantillonFaible: faible, volumes: values.map(() => 0), decomptes: values.map(() => null) }
}

const CAS: Array<[string, SerieDeSoirees]> = [
  ['série vide', serie([], [])],
  ['moins de trois soirées', serie([10, 20], [false, false])],
  ['toutes valides', serie([10.1, 20.2, 30.3, 40.4, 50.5], [false, false, false, false, false])],
  ['une soirée sans mesure dans la fenêtre', serie([10, null, 30, 40, 50], [false, false, false, false, false])],
  ['échantillon faible exclu', serie([10, 20, 30, 40, 50, 60], [false, true, false, false, false, true])],
  ['décimales non représentables', serie([0.1, 0.7, 0.2, 0.3, 33.333333333], [false, false, false, false, false])],
]

describe('moyenneGlissante (coordination) : mêmes nombres qu’avant 7.8', () => {
  it.each(CAS)('%s', (_label, s) => {
    const attendu = ancienne(s)
    const obtenu = moyenneGlissante(s)
    expect(obtenu).toHaveLength(attendu.length)
    obtenu.forEach((v, i) => expect(Object.is(v, attendu[i])).toBe(true))
  })
})
