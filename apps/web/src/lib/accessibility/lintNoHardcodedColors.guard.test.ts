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

  it.each([
    ['shadow', 'className="shadow-amber-500"', 'shadow-amber-500'],
    ['from', 'className="bg-gradient-to-r from-red-500"', 'from-red-500'],
    ['via', 'className="via-emerald-400"', 'via-emerald-400'],
    ['to', 'className="to-sky-600"', 'to-sky-600'],
    ['from avec opacité', 'className="from-slate-500/40"', 'from-slate-500'],
    ['shadow noir', 'className="shadow-black/20"', 'shadow-black/20'],
    ['to blanc', 'className="to-white"', 'to-white'],
  ])('rougit le préfixe %s', (_nom, line, match) => {
    expect(scan(line)).toEqual([{ kind: 'tailwind', match }])
  })

  it('laisse passer ces préfixes avec un jeton sémantique ou sans couleur nommée', () => {
    for (const line of [
      'className="shadow-md from-primary via-card to-background shadow-primary/20"',
      'className="bg-gradient-to-br from-success/40 via-warning/30 to-destructive/50"',
      'const dir = "to right"; const from = 1; const to = 2',
    ]) {
      expect(scan(line)).toEqual([])
    }
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

describe('lint des couleurs : marqueur inerte (plafond 0)', () => {
  const inerte = (match: string) => [{ kind: 'marqueur-inerte', match }]

  it('refuse un marqueur sur une ligne de code sans couleur', () => {
    expect(scan("const fg = 'var(--foreground)' // color-allow: encre neutre")).toEqual(inerte('color-allow'))
    expect(scan('...hoverRevealSymbol(tc.text), // color-allow(hex): point neutre')).toEqual(inerte('color-allow(hex)'))
    expect(scan('{/* color-allow: currentColor uniquement */}')).toEqual(inerte('color-allow'))
  })

  it('refuse un marqueur sur une ligne de commentaire, même quand elle cite une couleur', () => {
    expect(scan('// color-allow: hex en commentaires de documentation')).toEqual(inerte('color-allow'))
    expect(scan(" * color-allow: 'color' reçoit 'tc.text'")).toEqual(inerte('color-allow'))
    expect(scan('// couleur #f59e0b, color-allow(hex): étoile')).toEqual(inerte('color-allow(hex)'))
  })

  it('un marqueur ne peut pas excuser le genre marqueur-inerte', () => {
    expect(scan('const a = 1 // color-allow(marqueur-inerte): raison')).toEqual(inerte('color-allow(marqueur-inerte)'))
  })

  it('garde accepté un marqueur actif, qui couvre une vraie couleur', () => {
    expect(scan("const c = '#f59e0b' // color-allow(hex): étoile favori")).toEqual([])
    expect(scan('className="from-amber-500" // color-allow(tailwind): rareté')).toEqual([])
    expect(scan("const c = 'rgba(0,0,0,0.4)' // color-allow(fonction-couleur): ombre")).toEqual([])
  })

  it('une ligne de commentaire sans marqueur reste tolérée, même avec une couleur', () => {
    expect(scan('// ring-offset-red-500 #fff')).toEqual([])
    expect(scan(' * bg-black/60 rgba(0,0,0,1)')).toEqual([])
  })

  it('un fichier autorisé ne rougit pas pour un marqueur', () => {
    const palette = 'apps/web/src/lib/accessibility/palettes/default.ts'
    expect(scan('// color-allow: raison', palette)).toEqual([])
    expect(scan('// color-allow: raison', 'apps/web/src/features/x/Carte.test.tsx')).toEqual([])
  })
})
