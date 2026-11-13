/// <reference types="node" />
/**
 * Balayage commun des garde-rails d'architecture du front (frontières de couches) : lit les
 * `.ts` / `.tsx` de `src/` avec le compilateur TypeScript, de sorte qu'un commentaire, une
 * chaîne ou un nom de variable qui ressemble à la cible ne compte pas.
 *
 * Périmètre par défaut : hors tests (`*.test.*`), hors fichiers générés (`generated/`,
 * `generated.ts`, `routeTree.gen.ts`) et hors `node_modules`.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import ts from 'typescript'

export const SRC_ROOT = resolve(process.cwd(), 'src')

const GENERATED_FILES = new Set(['generated.ts', 'routeTree.gen.ts'])

export interface ScannedSource {
  /** Chemin relatif à `src/`, séparateur `/`. */
  rel: string
  /** Texte source (pour les contrôles de forme qui ne passent pas par l'arbre). */
  text: string
  sf: ts.SourceFile
}

export interface ScanOptions {
  /** Sous-dossier de `src/` balayé, séparateur `/` (`''` = tout `src/`). */
  under?: string
  /** Inclut les fichiers de test (`*.test.*`). Faux par défaut. */
  includeTests?: boolean
  /** Restreint aux fichiers `.tsx`. Faux par défaut. */
  tsxOnly?: boolean
}

function walk(dir: string, includeTests: boolean): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full, includeTests))
    } else if (
      /\.tsx?$/.test(entry.name) &&
      !GENERATED_FILES.has(entry.name) &&
      (includeTests || !/\.test\./.test(entry.name))
    ) {
      out.push(full)
    }
  }
  return out
}

/** Parse `text` comme le ferait le balayage (utile aux auto-preuves des détecteurs). */
export function parseSource(rel: string, text: string): ScannedSource {
  const kind = rel.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  return { rel, text, sf: ts.createSourceFile(rel, text, ts.ScriptTarget.Latest, true, kind) }
}

export function scanSources(options: ScanOptions = {}): ScannedSource[] {
  const { under = '', includeTests = false, tsxOnly = false } = options
  const root = under === '' ? SRC_ROOT : join(SRC_ROOT, ...under.split('/'))
  return walk(root, includeTests)
    .filter((full) => !tsxOnly || full.endsWith('.tsx'))
    .map((full) => {
      const rel = relative(SRC_ROOT, full).split(sep).join('/')
      return parseSource(rel, readFileSync(full, 'utf8'))
    })
}

/** Numéro de ligne (1-based) du début d'un nœud. */
export function lineOf(source: ScannedSource, node: ts.Node): number {
  return source.sf.getLineAndCharacterOfPosition(node.getStart(source.sf)).line + 1
}

/** Visite tous les nœuds du fichier. */
export function visitAll(source: ScannedSource, fn: (node: ts.Node) => void): void {
  const walkNode = (node: ts.Node): void => {
    fn(node)
    ts.forEachChild(node, walkNode)
  }
  walkNode(source.sf)
}

export interface Hit {
  file: string
  line: number
  detail: string
}

/** Appels dont le nom appelé (identifiant seul, ou dernier segment d'un accès `a.b.c`) est dans `names`. */
export function callsNamed(source: ScannedSource, names: ReadonlySet<string>): Hit[] {
  const hits: Hit[] = []
  visitAll(source, (node) => {
    if (!ts.isCallExpression(node)) return
    const callee = node.expression
    const name = ts.isIdentifier(callee)
      ? callee.text
      : ts.isPropertyAccessExpression(callee)
        ? callee.name.text
        : null
    if (name !== null && names.has(name)) hits.push({ file: source.rel, line: lineOf(source, node), detail: `${name}(...)` })
  })
  return hits
}

/**
 * Spécificateurs de module que le fichier importe, réexporte ou charge dynamiquement :
 * `import x from 'm'`, `import type { T } from 'm'`, `export * from 'm'`, `import('m')`,
 * `import x = require('m')`, `typeof import('m')`.
 */
export function moduleSpecifiers(source: ScannedSource): Array<{ specifier: string; line: number }> {
  const out: Array<{ specifier: string; line: number }> = []
  visitAll(source, (node) => {
    let literal: ts.Expression | undefined
    if (ts.isImportDeclaration(node)) literal = node.moduleSpecifier
    else if (ts.isExportDeclaration(node)) literal = node.moduleSpecifier
    else if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) literal = node.arguments[0]
    else if (ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument)) literal = node.argument.literal
    else if (ts.isExternalModuleReference(node)) literal = node.expression
    if (literal !== undefined && ts.isStringLiteralLike(literal)) {
      out.push({ specifier: literal.text, line: lineOf(source, literal) })
    }
  })
  return out
}

/** Vrai si le fichier est sous l'un des dossiers `dirs` de `src/` (préfixes terminés par `/`). */
export function isUnder(rel: string, dirs: readonly string[]): boolean {
  return dirs.some((d) => rel.startsWith(d))
}

/** Message d'échec : une ligne par occurrence, plus la consigne. */
export function formatHits(hits: readonly Hit[], hint: string): string {
  if (hits.length === 0) return ''
  return [`${hits.length} occurrence(s), plafond 0 :`, ...hits.map((h) => `  ${h.file}:${h.line}  ${h.detail}`), hint].join('\n')
}

/**
 * Délai des tests qui lisent tout `src/` : une seconde en local, plus de cinq sous la charge
 * d'un lot vitest de la CI (le défaut de vitest, 5 s, les faisait échouer).
 */
export const SCAN_TIMEOUT_MS = 120_000
