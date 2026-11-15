/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail du lint des couleurs (`tools/lint-no-hardcoded-colors.mjs`) : le coeur d'analyse d'une ligne
 * (`tools/lint-no-hardcoded-colors.lib.mjs`) est testé ici, parce qu'un lint qui ne rougit plus se
 * désactive sans bruit. Deux défauts réels l'ont motivé (entrée du backlog du 2026-11-10) :
 *   - `ring-offset`, `outline`, `decoration`, `accent` et `caret` échappaient au balayage ;
 *   - un `color-allow` excusait TOUTE couleur de sa ligne, pas seulement celle qu'il justifiait.
 */
import { describe, expect, it } from 'vitest'
// @ts-expect-error — script Node hors du programme tsc d'apps/web (frontière ESM/TS).
import { allowedKinds, scanLine } from '../../../../../tools/lint-no-hardcoded-colors.lib.mjs'

interface Violation {
  kind: string
  match: string
}

const SRC = 'apps/web/src/features/x/Carte.tsx'
const scan = (line: string, path = SRC) => scanLine(path, line) as Violation[]

describe('lint des couleurs : motifs', () => {
  it.each([
    ['ring-offset', 'className="ring-offset-red-500"', 'ring-offset-red-500'],
    ['outline', 'className="outline-amber-300"', 'outline-amber-300'],
    ['decoration', 'className="underline decoration-blue-600"', 'decoration-blue-600'],
    ['accent', 'className="accent-green-500"', 'accent-green-500'],
    ['caret', 'className="caret-rose-400"', 'caret-rose-400'],
    ['ring-offset blanc', 'className="ring-offset-white"', 'ring-offset-white'],
    ['outline noir', 'className="outline-black/40"', 'outline-black/40'],
    ['caret blanc', 'className="caret-white"', 'caret-white'],
  ])('rougit %s', (_nom, line, match) => {
    expect(scan(line)).toEqual([{ kind: 'tailwind', match }])
  })

  it('rougit toujours les motifs historiques', () => {
    expect(scan("const c = '#ff00aa'")).toEqual([{ kind: 'hex', match: '#ff00aa' }])
    expect(scan('className="text-red-500"')).toEqual([{ kind: 'tailwind', match: 'text-red-500' }])
    expect(scan('className="bg-black/60"')).toEqual([{ kind: 'tailwind', match: 'bg-black/60' }])
    expect(scan('const c = rgba(0, 0, 0, 0.4)')).toEqual([{ kind: 'fonction-couleur', match: 'rgba(' }])
  })

  it('laisse passer les jetons et les utilitaires sans couleur nommée', () => {
    for (const line of [
      'className="ring-offset-background outline-none accent-primary caret-foreground"',
      'className="outline-2 outline-offset-2 decoration-2 underline-offset-4"',
      'className="text-foreground bg-card border-border"',
    ]) {
      expect(scan(line)).toEqual([])
    }
  })

  it('tolère les lignes de commentaire, les tests et les fichiers autorisés', () => {
    expect(scan('// ring-offset-red-500 #fff')).toEqual([])
    expect(scan(' * outline-white')).toEqual([])
    expect(scan('const c = "#fff"', 'apps/web/src/features/x/Carte.test.tsx')).toEqual([])
    expect(scan('const c = "#fff"', 'apps/web/src/lib/accessibility/palettes/default.ts')).toEqual([])
  })
})

describe('lint des couleurs : color-allow', () => {
  it('lit les genres nommés par le marqueur', () => {
    expect([...(allowedKinds('x // color-allow(hex, fonction-couleur): raison') as Set<string>)]).toEqual([
      'hex',
      'fonction-couleur',
    ])
    expect(allowedKinds('rien')).toBeNull()
  })

  it('excuse le genre nommé', () => {
    expect(scan("const c = '#f59e0b' // color-allow(hex): étoile favori")).toEqual([])
  })

  it('n’excuse QUE le genre nommé : une classe couleur ajoutée à la même ligne rougit', () => {
    const line = `<svg fill="#f59e0b" className="text-red-500 ring-offset-blue-300" /> {/* color-allow(hex): étoile */}`
    expect(scan(line).map((v) => v.match)).toEqual(['text-red-500', 'ring-offset-blue-300'])
  })

  it('n’excuse pas un second genre non nommé', () => {
    const line = "const c = { a: '#38bdf8', b: 'rgba(56,189,248,0.15)' } // color-allow(hex): bleu"
    expect(scan(line)).toEqual([{ kind: 'fonction-couleur', match: 'rgba(' }])
  })

  it('excuse plusieurs genres nommés', () => {
    const line = "const c = { a: '#38bdf8', b: 'rgba(56,189,248,0.15)' } // color-allow(hex, fonction-couleur): bleu"
    expect(scan(line)).toEqual([])
  })

  it('un color-allow sans genre n’excuse rien', () => {
    expect(scan("const c = '#f59e0b' // color-allow: étoile").map((v) => v.kind)).toEqual(['hex'])
    expect(scan("const c = '#f59e0b' // color-allow(): étoile").map((v) => v.kind)).toEqual(['hex'])
    expect(scan("const c = '#f59e0b' // color-allow(inconnu): étoile").map((v) => v.kind)).toEqual(['hex'])
  })
})
