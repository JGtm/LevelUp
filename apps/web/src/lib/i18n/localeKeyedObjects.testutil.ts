/// <reference types="node" />
/**
 * Détection des objets « à clés de langue » non typés par `Locale` (ADR 0040, étape 4 du plan
 * `.ai/V8/PLAN_I18N_MULTILANGUE.md`), pour `locale-keyed-objects.guard.test.ts`.
 *
 * Pourquoi le compilateur et pas une regex : `{ fr: '…', en: '…' }` est typé par `Locale` soit
 * en place (`Record<Locale, T>`, `satisfies`), soit par un type déclaré ailleurs (une interface
 * `extends Record<Locale, string>`, un champ typé d'une structure plus large). Seul le type
 * contextuel de l'objet le dit. Une table `Record<Locale, T>` est un type MAPPÉ : c'est la
 * marque cherchée, par opposition à une interface `{ fr: string; en: string }`, qui compile
 * encore quand une 3e langue est ajoutée et la laisse sans texte.
 *
 * Deux violations :
 *  - `untyped-literal` : un objet littéral qui porte les clés `fr` ET `en` sans type contextuel
 *    mappé (aucun type, ou un type écrit à la main) ;
 *  - `inline-union` : le type `'fr' | 'en'` écrit en toutes lettres, qui fige la liste des
 *    langues hors de `config/locales.toml` (le type s'écrit `Locale`).
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import ts from 'typescript'

export interface Violation {
  kind: 'untyped-literal' | 'inline-union'
  /** Fichier relatif à `src/`, séparateur `/`. */
  file: string
  line: number
  /** Type contextuel de l'objet (`untyped-literal`), `-` quand il n'y en a pas. */
  context: string
  text: string
}

const SRC_ROOT = resolve(process.cwd(), 'src')
const SCANNED_ROOTS = ['features', 'components', 'lib']
const GENERATED_FILES = new Set(['generated.ts', 'routeTree.gen.ts'])

/** Dossiers de support de test (`test/`, `__tests__/`) : hors périmètre, comme les `*.test.*`. */
const TEST_SUPPORT_DIRS = new Set(['test', '__tests__'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated' || TEST_SUPPORT_DIRS.has(entry.name)) continue
      out.push(...walk(full))
    } else if (/\.tsx?$/.test(entry.name) && !/\.test\./.test(entry.name) && !GENERATED_FILES.has(entry.name)) {
      out.push(full)
    }
  }
  return out
}

/** Fichiers de `features/`, `components/`, `lib/` : hors tests, hors support de test, hors générés. */
export function listScannedFiles(): string[] {
  return SCANNED_ROOTS.flatMap((root) => walk(join(SRC_ROOT, root)))
}

/** Pré-filtre : seuls les fichiers qui nomment `fr` et `en` peuvent porter des clés de langue. */
function mayCarryLocaleKeys(text: string): boolean {
  return /\bfr\b/.test(text) && /\ben\b/.test(text)
}

function propertyName(p: ts.ObjectLiteralElementLike): string | null {
  const name = p.name
  if (name && (ts.isIdentifier(name) || ts.isStringLiteral(name))) return name.text
  return null
}

/** Vrai si `t` est une table par langue : type mappé (`Record`, `Partial<Record>`, `{ [L in Locale]: … }`), ou hérite d'un tel type. */
function isLocaleKeyed(checker: ts.TypeChecker, t: ts.Type | undefined, depth = 0): boolean {
  if (!t || depth > 4) return false
  if (t.isUnion() || t.isIntersection()) return t.types.some((x) => isLocaleKeyed(checker, x, depth + 1))
  if ((t as ts.ObjectType).objectFlags & ts.ObjectFlags.Mapped) return true
  if ((t as ts.ObjectType).objectFlags & (ts.ObjectFlags.Interface | ts.ObjectFlags.Class)) {
    return checker.getBaseTypes(t as ts.InterfaceType).some((b) => isLocaleKeyed(checker, b, depth + 1))
  }
  return false
}

/** `'fr' | 'en'` (dans cet ordre ou non, avec ou sans autres littéraux) : la liste des langues réécrite à la main. */
function isInlineLocaleUnion(node: ts.UnionTypeNode): boolean {
  const literals = node.types.flatMap((t) =>
    ts.isLiteralTypeNode(t) && ts.isStringLiteral(t.literal) ? [t.literal.text] : [],
  )
  return literals.includes('fr') && literals.includes('en')
}

function lineOf(sf: ts.SourceFile, node: ts.Node): { line: number; text: string } {
  const line = sf.getLineAndCharacterOfPosition(node.getStart(sf)).line
  return { line: line + 1, text: sf.text.split('\n')[line]?.trim().slice(0, 100) ?? '' }
}

/**
 * Violations des `files` dans `program`. `rel` rend le chemin affiché (relatif à `src/` pour les
 * vrais fichiers) ; les fichiers virtuels des auto-preuves gardent leur nom.
 */
export function findViolations(program: ts.Program, files: string[], rel: (f: string) => string): Violation[] {
  const checker = program.getTypeChecker()
  const out: Violation[] = []
  for (const file of files) {
    const sf = program.getSourceFile(file)
    if (!sf) throw new Error(`fichier absent du programme : ${file}`)
    const visit = (node: ts.Node): void => {
      if (ts.isObjectLiteralExpression(node)) {
        const keys = new Set(node.properties.map(propertyName))
        if (keys.has('fr') && keys.has('en')) {
          const context = checker.getContextualType(node)
          if (!isLocaleKeyed(checker, context)) {
            out.push({ kind: 'untyped-literal', file: rel(file), ...lineOf(sf, node), context: context ? checker.typeToString(context).slice(0, 60) : '-' })
          }
        }
      } else if (ts.isUnionTypeNode(node) && isInlineLocaleUnion(node)) {
        out.push({ kind: 'inline-union', file: rel(file), ...lineOf(sf, node), context: '-' })
      }
      ts.forEachChild(node, visit)
    }
    visit(sf)
  }
  return out
}

/** Programme des vrais fichiers candidats, avec les options de `tsconfig.app.json` (alias `@/`). */
export function buildRealProgram(
  keep: (text: string) => boolean = mayCarryLocaleKeys,
): { program: ts.Program; files: string[] } {
  const configPath = resolve(process.cwd(), 'tsconfig.app.json')
  const parsed = ts.getParsedCommandLineOfConfigFile(
    configPath,
    {},
    {
      ...ts.sys,
      onUnRecoverableConfigFileDiagnostic: (d) => {
        throw new Error(ts.flattenDiagnosticMessageText(d.messageText, '\n'))
      },
    },
  )
  if (!parsed) throw new Error('tsconfig.app.json : configuration TypeScript non chargée')
  const files = listScannedFiles().filter((f) => keep(readFileSync(f, 'utf8')))
  return { program: ts.createProgram({ rootNames: files, options: { ...parsed.options, noEmit: true } }), files }
}

/** Chemin relatif à `src/`, séparateur `/`. */
export function relativeToSrc(file: string): string {
  return relative(SRC_ROOT, file).split(sep).join('/')
}

/** Programme en mémoire d'un fichier virtuel (auto-preuves), sans dépendance au dépôt. */
export function buildVirtualProgram(name: string, source: string): ts.Program {
  const options: ts.CompilerOptions = { noEmit: true, strict: true, target: ts.ScriptTarget.ES2022, types: [], noLib: false }
  const host = ts.createCompilerHost(options)
  const original = host.getSourceFile.bind(host)
  host.getSourceFile = (fileName, languageVersion, ...rest) =>
    fileName === name ? ts.createSourceFile(fileName, source, languageVersion, true) : original(fileName, languageVersion, ...rest)
  const exists = host.fileExists.bind(host)
  host.fileExists = (f) => f === name || exists(f)
  return ts.createProgram({ rootNames: [name], options, host })
}
