/// <reference types="node" />
/**
 * Balayage commun des garde-rails à compteur gelé sur la langue (ADR 0040, D-8) :
 * `no-locale-literal-comparison.guard.test.ts` et `no-intl-tag-literal.guard.test.ts` (qui compte aussi les
 * appels Intl sans étiquette, `scanUntaggedIntl`).
 *
 * Périmètre : les `.ts` / `.tsx` de `src/`, hors tests (`*.test.*`), hors fichiers générés
 * (`generated/`, `generated.ts`, `routeTree.gen.ts`) et hors des deux dossiers qui possèdent la
 * langue (`lib/i18n/`, `lib/formatters/`). Le compte est en LIGNES (une ligne qui matche = 1),
 * comme `rg -c` dans l'état des lieux (`.ai/V8/ETAT_DES_LIEUX_I18N_MULTILANGUE_2026-10-23.md`).
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import ts from 'typescript'

const SRC_ROOT = resolve(process.cwd(), 'src')

/** Dossiers propriétaires de la langue, relatifs à `src/`, séparateur `/`. */
const OWNER_DIRS = ['lib/i18n/', 'lib/formatters/']

const GENERATED_FILES = new Set(['generated.ts', 'routeTree.gen.ts'])

export interface LineHit {
  file: string
  line: number
  /** Vrai si la ligne, ou celle directement au-dessus, porte `contractTag` (voir `scanLines`). */
  tagged: boolean
}

export interface ScanOptions {
  /** Sous-chaîne qui balise une ligne (sur elle ou sur la ligne du dessus) : `LineHit.tagged`. */
  contractTag?: string
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.tsx?$/.test(entry.name) && !/\.test\./.test(entry.name) && !GENERATED_FILES.has(entry.name)) {
      out.push(full)
    }
  }
  return out
}

/**
 * Lignes (fichier relatif à `src/`, n° de ligne) dont le texte satisfait `pattern`. Avec
 * `options.contractTag`, chaque ligne dit si elle est balisée (sous-chaîne sur la ligne ou sur
 * la ligne directement au-dessus : les lignes de texte JSX ne portent pas de commentaire de fin).
 */
export function scanLines(pattern: RegExp, options: ScanOptions = {}): LineHit[] {
  const hits: LineHit[] = []
  for (const full of walk(SRC_ROOT)) {
    const rel = relative(SRC_ROOT, full).split(sep).join('/')
    if (OWNER_DIRS.some((d) => rel.startsWith(d))) continue
    const lines = readFileSync(full, 'utf8').split('\n')
    lines.forEach((text, i) => {
      if (!pattern.test(text)) return
      const tag = options.contractTag
      const tagged = tag !== undefined && (text.includes(tag) || (lines[i - 1] ?? '').includes(tag))
      hits.push({ file: rel, line: i + 1, tagged })
    })
  }
  return hits
}

const NO_TAG_METHODS = /^toLocale(Date|Time)?String$/

/** Vrai si l'argument ne désigne aucune étiquette : absent, `undefined`, `null` ou un objet d'options. */
function carriesNoTag(arg: ts.Expression | undefined): boolean {
  return (
    arg === undefined ||
    ts.isObjectLiteralExpression(arg) ||
    arg.kind === ts.SyntaxKind.NullKeyword ||
    (ts.isIdentifier(arg) && arg.text === 'undefined')
  )
}

/**
 * Numéros de ligne des appels de formatage SANS étiquette de langue de `source`, donc rendus dans
 * la langue du NAVIGATEUR et non celle de l'interface : `x.toLocaleString()`,
 * `d.toLocaleDateString(undefined, {…})`, `d.toLocaleTimeString({…})`, `new Intl.NumberFormat()`,
 * `Intl.DateTimeFormat(undefined, …)`. Lu par le compilateur : les arguments peuvent tenir sur
 * plusieurs lignes.
 */
export function untaggedIntlLines(source: string): number[] {
  const sf = ts.createSourceFile('source.tsx', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
  const lines: number[] = []
  const visit = (node: ts.Node): void => {
    if ((ts.isCallExpression(node) || ts.isNewExpression(node)) && ts.isPropertyAccessExpression(node.expression)) {
      const callee = node.expression
      const isIntl = ts.isIdentifier(callee.expression) && callee.expression.text === 'Intl'
      const isMethod = ts.isCallExpression(node) && NO_TAG_METHODS.test(callee.name.text)
      if ((isIntl || isMethod) && carriesNoTag(node.arguments?.[0])) {
        lines.push(sf.getLineAndCharacterOfPosition(node.getStart(sf)).line + 1)
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(sf)
  return lines
}

/** `untaggedIntlLines` sur le périmètre de `scanLines` (hors `lib/i18n/` et `lib/formatters/`). */
export function scanUntaggedIntl(): LineHit[] {
  const hits: LineHit[] = []
  for (const full of walk(SRC_ROOT)) {
    const rel = relative(SRC_ROOT, full).split(sep).join('/')
    if (OWNER_DIRS.some((d) => rel.startsWith(d))) continue
    for (const line of untaggedIntlLines(readFileSync(full, 'utf8'))) hits.push({ file: rel, line, tagged: false })
  }
  return hits
}

/** Langue écrite en toutes lettres : `'fr'` ou `'en'` (chaîne ou gabarit sans substitution). */
function isLanguageLiteral(node: ts.Node | undefined): boolean {
  return (
    node !== undefined &&
    (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) &&
    (node.text === 'fr' || node.text === 'en')
  )
}

/**
 * Numéros de ligne de `source` où une langue littérale sert de DÉFAUT à une valeur de langue :
 *  - un paramètre ou un champ déstructuré à valeur par défaut (`locale = 'fr'`, `{ lang = 'en' }`),
 *    quel que soit son nom ;
 *  - un repli `x ?? 'fr'` / `x || 'fr'` dont l'opérande de gauche porte `locale` ou `lang` dans son texte.
 * Le défaut de l'application est `DEFAULT_LOCALE` (`config/locales.toml`), jamais un littéral.
 */
export function languageDefaultLiteralLines(source: string): number[] {
  const sf = ts.createSourceFile('source.tsx', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
  const lines: number[] = []
  const flag = (node: ts.Node): void => {
    lines.push(sf.getLineAndCharacterOfPosition(node.getStart(sf)).line + 1)
  }
  const visit = (node: ts.Node): void => {
    if ((ts.isParameter(node) || ts.isBindingElement(node)) && isLanguageLiteral(node.initializer)) flag(node)
    if (
      ts.isBinaryExpression(node) &&
      (node.operatorToken.kind === ts.SyntaxKind.QuestionQuestionToken || node.operatorToken.kind === ts.SyntaxKind.BarBarToken) &&
      isLanguageLiteral(node.right) &&
      /locale|lang/i.test(node.left.getText(sf))
    ) {
      flag(node)
    }
    ts.forEachChild(node, visit)
  }
  visit(sf)
  return lines
}

/** `languageDefaultLiteralLines` sur le périmètre de `scanLines` (hors `lib/i18n/` et `lib/formatters/`). */
export function scanLanguageDefaultLiterals(): LineHit[] {
  const hits: LineHit[] = []
  for (const full of walk(SRC_ROOT)) {
    const rel = relative(SRC_ROOT, full).split(sep).join('/')
    if (OWNER_DIRS.some((d) => rel.startsWith(d))) continue
    for (const line of languageDefaultLiteralLines(readFileSync(full, 'utf8'))) hits.push({ file: rel, line, tagged: false })
  }
  return hits
}

/** Message d'échec : le compte, le plafond et la liste fichier:lignes. */
export function formatOverflow(hits: LineHit[], ceiling: number, hint: string): string {
  const byFile = new Map<string, number[]>()
  for (const h of hits) byFile.set(h.file, [...(byFile.get(h.file) ?? []), h.line])
  const detail = [...byFile.entries()].map(([f, ls]) => `  ${f} : ${ls.join(', ')}`).join('\n')
  return `${hits.length} occurrences (plafond : ${ceiling}, ADR 0040). ${hint}\n${detail}`
}
