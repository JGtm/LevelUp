/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : la taille des fonctions du front ne grossit pas (CLAUDE.md règle 5 : fonction
 * <= 80 lignes de code, <= 5 paramètres). ESLint n'a ni `max-params` ni `max-lines-per-function` :
 * rien d'autre ne bloquait ces ajouts côté web.
 *
 * Deux compteurs sous `apps/web/src`, hors tests (fichiers de test, dossiers `test/`, fixtures) et
 * fichiers générés, mesurés avec l'API du compilateur TypeScript :
 *  - les fonctions à plus de 5 paramètres ;
 *  - les fonctions de plus de 80 lignes de code (lignes vides et commentaires exclus).
 *
 * Chaque compteur est gelé à sa valeur mesurée : le test échoue quand il MONTE et aussi quand il
 * DESCEND sans que le plafond le suive (cliquet : l'abaisser est le geste qui fixe le gain).
 * Une fonction se compte à part de celles qu'elle contient.
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'
import { parseSource, scanSources, SCAN_TIMEOUT_MS, visitAll, type ScannedSource } from '@/lib/archScan.testutil'

const MAX_PARAMS = 5
const MAX_CODE_LINES = 80

/** Valeurs mesurées à la clôture de E.11 : ne font que descendre. */
const PLAFOND_PARAMETRES = 46
const PLAFOND_LONGUEUR = 249

interface FunctionHit {
  file: string
  line: number
  name: string
  params: number
  codeLines: number
}

function isFunctionLike(node: ts.Node): node is ts.FunctionLikeDeclaration {
  return (
    ts.isFunctionDeclaration(node) ||
    ts.isFunctionExpression(node) ||
    ts.isArrowFunction(node) ||
    ts.isMethodDeclaration(node) ||
    ts.isGetAccessorDeclaration(node) ||
    ts.isSetAccessorDeclaration(node) ||
    ts.isConstructorDeclaration(node)
  )
}

/** Les lignes (0-based) qui portent au moins un jeton de code : ni vides, ni de commentaire seul. */
function codeLinesOf(source: ScannedSource): Set<number> {
  const { sf, text } = source
  const lines = new Set<number>()
  const mark = (from: number, to: number): void => {
    const first = sf.getLineAndCharacterOfPosition(from).line
    const last = sf.getLineAndCharacterOfPosition(Math.max(from, to - 1)).line
    for (let l = first; l <= last; l++) lines.add(l)
  }
  const walk = (node: ts.Node): void => {
    // `{/* ... */}` : un commentaire, pas du code.
    if (ts.isJsxExpression(node) && !node.expression) return
    if (node.kind === ts.SyntaxKind.JsxText) {
      const raw = text.slice(node.pos, node.end)
      const lead = raw.length - raw.trimStart().length
      if (raw.trim() !== '') mark(node.pos + lead, node.pos + lead + raw.trim().length)
      return
    }
    const children = node.getChildren(sf)
    if (children.length === 0) {
      if (node.kind !== ts.SyntaxKind.EndOfFileToken) mark(node.getStart(sf), node.end)
      return
    }
    children.forEach(walk)
  }
  walk(sf)
  return lines
}

function functionName(node: ts.FunctionLikeDeclaration): string {
  if (node.name && ts.isIdentifier(node.name)) return node.name.text
  const parent = node.parent
  if (ts.isVariableDeclaration(parent) && ts.isIdentifier(parent.name)) return parent.name.text
  if (ts.isPropertyAssignment(parent) && ts.isIdentifier(parent.name)) return parent.name.text
  return '(anonyme)'
}

/** Toutes les fonctions du fichier avec leur nombre de paramètres et de lignes de code. */
export function measureFunctions(source: ScannedSource): FunctionHit[] {
  const code = codeLinesOf(source)
  const hits: FunctionHit[] = []
  visitAll(source, (node) => {
    if (!isFunctionLike(node)) return
    const first = source.sf.getLineAndCharacterOfPosition(node.getStart(source.sf)).line
    const last = source.sf.getLineAndCharacterOfPosition(node.end).line
    let codeLines = 0
    for (let l = first; l <= last; l++) if (code.has(l)) codeLines++
    const params = node.parameters.filter((p) => !(ts.isIdentifier(p.name) && p.name.text === 'this')).length
    hits.push({ file: source.rel, line: first + 1, name: functionName(node), params, codeLines })
  })
  return hits
}

function describeHit(h: FunctionHit): string {
  return `  ${h.file}:${h.line}  ${h.name}  (${h.params} paramètres, ${h.codeLines} lignes de code)`
}

/** Le soutien des tests (dossiers `test/`, fixtures, utilitaires de test) n'est pas du code du produit. */
const TEST_SUPPORT = /(^|\/)test\/|\.fixtures?\.tsx?$|\.testutil\.tsx?$/

let measured: FunctionHit[] | null = null

/** Toutes les fonctions du produit, mesurées une seule fois pour les deux compteurs. */
function allFunctions(): FunctionHit[] {
  measured ??= scanSources()
    .filter((s) => !TEST_SUPPORT.test(s.rel))
    .flatMap(measureFunctions)
  return measured
}

function ratchet(kind: string, hits: FunctionHit[], plafond: number): void {
  const message =
    `${hits.length} fonction(s) ${kind}, plafond ${plafond}. ` +
    (hits.length > plafond
      ? 'Extraire des sous-fonctions ou un objet d’options ; ne pas relever le plafond.'
      : 'Le compte a baissé : abaisser le plafond à cette valeur.') +
    `\n${hits.map(describeHit).join('\n')}`
  expect(hits.length, message).toBe(plafond)
}

describe('garde-rail : taille des fonctions du front', () => {
  it(`fonctions à plus de ${MAX_PARAMS} paramètres : plafond gelé`, () => {
    ratchet(`à plus de ${MAX_PARAMS} paramètres`, allFunctions().filter((h) => h.params > MAX_PARAMS), PLAFOND_PARAMETRES)
  }, SCAN_TIMEOUT_MS)

  it(`fonctions de plus de ${MAX_CODE_LINES} lignes de code : plafond gelé`, () => {
    ratchet(
      `de plus de ${MAX_CODE_LINES} lignes de code`,
      allFunctions().filter((h) => h.codeLines > MAX_CODE_LINES),
      PLAFOND_LONGUEUR,
    )
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : paramètres, lignes vides et commentaires, JSX, gabarit multiligne, fonctions imbriquées', () => {
    const code = [
      'export function six(a: number, b: number, c: number, d: number, e: number, f: number) {',
      '  // un commentaire seul',
      '',
      '  /* un bloc',
      '     sur deux lignes */',
      '  return a + b + c + d + e + f // fin de ligne',
      '}',
      'export const cinq = (a: number, b: number, c: number, d: number, e: number) => {',
      '  const t = `un',
      '',
      '  gabarit`',
      '  return () => (',
      '    <div>',
      "      l'apostrophe",
      '      {/* commentaire JSX */}',
      '    </div>',
      '  )',
      '}',
    ].join('\n')
    const hits = measureFunctions(parseSource('lib/x.tsx', code))
    const six = hits.find((h) => h.name === 'six')
    const cinq = hits.find((h) => h.name === 'cinq')
    const interne = hits.find((h) => h.name === '(anonyme)')
    expect(six).toMatchObject({ params: 6, codeLines: 3 })
    expect(cinq).toMatchObject({ params: 5, codeLines: 10 })
    expect(interne).toMatchObject({ params: 0, codeLines: 5 })
  })
})
