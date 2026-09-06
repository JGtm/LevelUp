/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — UN SEUL PRÉDICAT « CET INTERVALLE COUVRE CETTE IMAGE ».
 *
 * POURQUOI CE GARDE (registre 2026-09-05, K5). Le prédicat était écrit DIX fois, dans deux
 * orthographes selon l'auteur. Les dix étaient d'accord — mesuré en vérification adverse — et
 * c'est ce qui rendait la onzième copie facile à écrire sans y penser, donc facile à écrire
 * FAUX : `<` au lieu de `<=` sur `t1` fait disparaître un glyphe à sa dernière image, ce qui
 * ne se voit sur aucun écran.
 *
 * CE QU'IL DÉTECTE : les deux orthographes de la comparaison, sous leur forme littérale.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

/** `frame >= s.t0 && frame <= s.t1` et sa jumelle `s.t0 <= frame && frame <= s.t1`. */
const PREDICAT = [
  /(\w+) >= (\w+)\.t0 && \1 <= \2\.t1/,
  /(\w+)\.t0 <= (\w+) && \2 <= \1\.t1/,
]

const AUTORISES = new Set(['replaySpans.ts', 'replaySpans.guard.test.ts'])

function fautifs(): string[] {
  return readdirSync(__dirname)
    .filter((n) => /\.(ts|tsx)$/.test(n) && !AUTORISES.has(n))
    .filter((n) => {
      const src = readFileSync(join(__dirname, n), 'utf8')
      return PREDICAT.some((re) => re.test(src))
    })
}

describe('garde-rail : un seul prédicat d’intervalle', () => {
  it('personne ne réécrit la comparaison, dans aucune des deux orthographes', () => {
    expect(fautifs()).toEqual([])
  })

  it('et `replaySpans` la porte bien — sans quoi ce test ne garderait rien', () => {
    const src = readFileSync(join(__dirname, 'replaySpans.ts'), 'utf8')
    expect(PREDICAT.some((re) => re.test(src))).toBe(true)
  })
})
