/// <reference types="node" />
// @vitest-environment node
/**
 * LA CADENCE D'UN ENSEMBLE DE MATCHS, JEU DE CAS PARTAGÉ AVEC LE GO.
 *
 * `pooledPerMinute` (somme des comptes / somme des minutes) est le miroir de `analysis.PooledPerMinute`
 * (Go). Les deux côtés lisent le même fichier, c'est lui qui les empêche de diverger : un compte ou une
 * durée inconnue (`null`) sort du numérateur ET du dénominateur, une durée nulle ou négative aussi, et
 * sans aucun match retenu il n'y a pas de valeur (jamais zéro). Le test Go est
 * `pooled_per_minute_test.go`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { pooledPerMinute } from './perMinute'

const JEU_DE_CAS = resolve(__dirname, '../../../../go-api/internal/analysis/testdata/par_minute_cas.json')

/** Tolérance de comparaison écrite dans le fichier de cas. */
const TOLERANCE = 1e-9

interface MatchDuCas {
  count: number | null
  seconds: number | null
}

interface CasParMinute {
  nom: string
  matchs: MatchDuCas[]
  attendu: number | null
}

const jeu = JSON.parse(readFileSync(JEU_DE_CAS, 'utf8')) as { cas: CasParMinute[] }

describe('pooledPerMinute : le jeu de cas partagé avec le Go', () => {
  it('le jeu de cas n’est pas vide', () => {
    expect(jeu.cas.length).toBeGreaterThan(0)
  })

  for (const cas of jeu.cas) {
    it(cas.nom, () => {
      const lu = pooledPerMinute(
        cas.matchs,
        (m) => m.count,
        (m) => m.seconds,
      )
      if (cas.attendu === null) {
        expect(lu).toBeNull()
        return
      }
      expect(lu).not.toBeNull()
      expect(Math.abs((lu as number) - cas.attendu)).toBeLessThanOrEqual(TOLERANCE)
    })
  }
})
