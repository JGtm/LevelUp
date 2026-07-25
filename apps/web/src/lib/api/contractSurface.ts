/**
 * Analyseur STATIQUE de `generated.ts` — source unique des garde-rails de contrat.
 *
 * Pourquoi statique : `generated.ts` n'est qu'un fichier de TYPES (effacé à l'exécution) ;
 * on ne peut donc pas « lire » sa surface au runtime. Les garde-rails du contrat
 * (`contract-surface.guard.test.ts`, `response-types.guard.test.ts`) l'analysent par
 * parcours de texte : commentaires retirés, littéraux de chaîne neutralisés (les gabarits
 * de chemin `{player_slug}` et les exemples JSON en commentaire fausseraient le comptage
 * d'accolades), profondeur d'accolades suivie ligne à ligne.
 *
 * Module CENTRAL (CLAUDE.md n°6) : ne pas ré-implémenter l'extraction dans un test.
 * NB : `tools/lint-contract-ratchet.mjs` garde sa propre extraction minimale — c'est un
 * script Node hors `apps/web` (frontière ESM/TS), avec son propre contrôle anti-vacuité.
 */

/** Surface publique du contrat, telle que figée par le snapshot du garde-rail. */
export interface ContractSurface {
  /** Chemins HTTP documentés (clés de `export interface paths`). */
  paths: string[]
  /** Noms de schéma (clés de `components.schemas`). */
  schemas: string[]
  /** Réponses partagées (clés de `components.responses`). */
  responses: string[]
  /** Paramètres partagés (clés de `components.parameters`). */
  parameters: string[]
  /** operationIds (clés de `export interface operations`). */
  operations: string[]
  /** Unions de littéraux, indexées `<porteur>.<champ>` → membres triés. */
  enums: Record<string, string[]>
}

interface Block {
  start: number
  end: number
}

function count(line: string, ch: string): number {
  let n = 0
  for (const c of line) if (c === ch) n++
  return n
}

/** Version « code seul » de chaque ligne (commentaires retirés, chaînes neutralisées). */
function codeOnly(lines: string[]): string[] {
  const out: string[] = []
  let inComment = false
  for (const raw of lines) {
    let line = raw
    if (inComment) {
      const end = line.indexOf('*/')
      if (end === -1) {
        out.push('')
        continue
      }
      line = line.slice(end + 2)
      inComment = false
    }
    line = line.replace(/\/\*.*?\*\//g, '')
    const open = line.indexOf('/*')
    if (open !== -1) {
      line = line.slice(0, open)
      inComment = true
    }
    line = line.replace(/"(?:[^"\\]|\\.)*"/g, '""')
    line = line.replace(/\/\/.*$/, '')
    out.push(line)
  }
  return out
}

/** Localise le bloc `{ … }` ouvert par la première ligne matchant `headerRe`. */
function findBlock(code: string[], headerRe: RegExp): Block | null {
  for (let i = 0; i < code.length; i++) {
    if (!headerRe.test(code[i])) continue
    let depth = 0
    for (let j = i; j < code.length; j++) {
      depth += count(code[j], '{') - count(code[j], '}')
      if (j === i && depth <= 0) break
      if (j > i && depth <= 0) return { start: i, end: j }
    }
  }
  return null
}

const KEY_RE = /^\s*(?:"([^"]+)"|([A-Za-z0-9_$]+))\??:/

/** Clés de premier niveau du bloc (profondeur 1 relative à sa ligne d'en-tête). */
function topLevelKeys(raw: string[], code: string[], block: Block): string[] {
  const keys: string[] = []
  let depth = 0
  for (let j = block.start; j <= block.end; j++) {
    const before = depth
    depth += count(code[j], '{') - count(code[j], '}')
    if (j === block.start) continue
    if (before !== 1) continue
    const m = raw[j].match(KEY_RE)
    if (m) keys.push(m[1] ?? m[2])
  }
  return keys
}

/**
 * Membres d'une union de littéraux de chaîne, ou null si le membre droit n'est PAS une
 * union purement littérale (`string`, `$ref`, objet inline…). `null`/`undefined` sont
 * tolérés comme marqueurs de nullabilité et ne sont pas figés comme membres.
 */
function literalUnionMembers(rhs: string): string[] | null {
  const body = rhs.trim().replace(/;+$/, '').trim()
  if (!body) return null
  const members: string[] = []
  for (const part of body.split('|')) {
    const p = part.trim()
    if (p === 'null' || p === 'undefined') continue
    const m = p.match(/^"((?:[^"\\]|\\.)*)"$/)
    if (!m) return null
    members.push(m[1])
  }
  return members.length > 0 ? members : null
}

/**
 * Enums du bloc, indexés `<cléDePremierNiveau>.<champ>`. Les champs homonymes imbriqués
 * dans un même porteur fusionnent leurs membres (union conservatrice : le garde-rail
 * exige la présence de CHAQUE membre déjà observé).
 */
function collectEnums(raw: string[], code: string[], block: Block, into: Map<string, Set<string>>): void {
  let depth = 0
  let currentTop: string | null = null
  for (let j = block.start; j <= block.end; j++) {
    const before = depth
    depth += count(code[j], '{') - count(code[j], '}')
    if (j === block.start) continue
    const m = raw[j].match(KEY_RE)
    if (before === 1) {
      currentTop = m ? (m[1] ?? m[2]) : currentTop
      continue
    }
    if (!m || currentTop === null) continue
    const field = m[1] ?? m[2]
    const colon = raw[j].indexOf(':', raw[j].indexOf(field))
    if (colon === -1) continue
    const members = literalUnionMembers(raw[j].slice(colon + 1))
    if (!members) continue
    const key = `${currentTop}.${field}`
    const set = into.get(key) ?? new Set<string>()
    for (const v of members) set.add(v)
    into.set(key, set)
  }
}

const sorted = (xs: string[]): string[] => [...new Set(xs)].sort()

/** Extrait la surface complète du contrat depuis le SOURCE de `generated.ts`. */
export function extractContractSurface(src: string): ContractSurface {
  const raw = src.split('\n')
  const code = codeOnly(raw)

  const pathsBlock = findBlock(code, /^export interface paths \{/)
  const componentsBlock = findBlock(code, /^export interface components \{/)
  const operationsBlock = findBlock(code, /^export interface operations \{/)
  if (!pathsBlock || !componentsBlock || !operationsBlock) {
    throw new Error(
      'contractSurface: blocs paths/components/operations introuvables dans generated.ts — parser à revoir.',
    )
  }
  const shift = (b: Block | null): Block | null =>
    b ? { start: b.start + componentsBlock.start, end: b.end + componentsBlock.start } : null
  const inComponents = code.slice(componentsBlock.start)
  const schemasBlock = shift(findBlock(inComponents, /^ {4}schemas: \{/))
  const responsesBlock = shift(findBlock(inComponents, /^ {4}responses: \{/))
  const parametersBlock = shift(findBlock(inComponents, /^ {4}parameters: \{/))
  if (!schemasBlock) {
    throw new Error('contractSurface: bloc components.schemas introuvable — parser à revoir.')
  }

  const enums = new Map<string, Set<string>>()
  collectEnums(raw, code, schemasBlock, enums)
  collectEnums(raw, code, operationsBlock, enums)

  return {
    paths: sorted(topLevelKeys(raw, code, pathsBlock)),
    schemas: sorted(topLevelKeys(raw, code, schemasBlock)),
    responses: responsesBlock ? sorted(topLevelKeys(raw, code, responsesBlock)) : [],
    parameters: parametersBlock ? sorted(topLevelKeys(raw, code, parametersBlock)) : [],
    operations: sorted(topLevelKeys(raw, code, operationsBlock)),
    enums: Object.fromEntries(
      [...enums.entries()]
        .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
        .map(([k, v]) => [k, [...v].sort()]),
    ),
  }
}
