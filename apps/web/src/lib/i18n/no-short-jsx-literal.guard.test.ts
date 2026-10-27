/// <reference types="node" />
/**
 * Garde-rail à compteur gelé (ADR 0040) : texte d'interface COURT écrit en dur dans le JSX.
 *
 * La règle ESLint `no-hardcoded-strings` ne signale un littéral que s'il a 3 mots ou 15 caractères :
 * « Aide », « Analyser », « (moi) » lui échappent et s'affichent dans la langue du code sous toute
 * autre langue (constaté à l'étape 10, preuve par la pseudo-langue `xx`). Ce test compte, au compilateur,
 * tout texte JSX et tout attribut `title` / `aria-label` / `placeholder` / `alt` / `label` littéral ou en gabarit
 * d'au moins 3 lettres, hors tests et hors fichiers de langue.
 *
 * Le plafond ne fait que descendre : un littéral de plus échoue (passer par un manifest ou un dictionnaire
 * `Record<Locale, T>`) ; un littéral converti abaisse le plafond. Les 105 restants (2026-10-27, dont 5 attributs en gabarit « Retirer ${x} ») sont de la dette
 * hors pages témoins de l'étape 10 (administration, laboratoire, pages d'accueil de connexion et d'installation,
 * classement Prestige, médias), plus des noms propres et sigles (LevelUp, MVP, GitHub, CODE1234).
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import ts from 'typescript'
import { describe, expect, it } from 'vitest'

const SRC_ROOT = resolve(process.cwd(), 'src')
const SHORT_LITERAL_CEILING = 105

const TEXT_ATTRS = new Set(['title', 'aria-label', 'aria-description', 'aria-placeholder', 'placeholder', 'alt', 'label'])
const EXCLUDED = [
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
  /(^|\/)test\//,
  /(^|\/)__tests__\//,
  /(^|\/)i18n\//,
  /\.i18n\.ts$/,
  /(^|\/)i18n\.ts$/,
]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name !== 'node_modules' && entry.name !== 'generated') out.push(...walk(full))
    } else if (entry.name.endsWith('.tsx')) {
      out.push(full)
    }
  }
  return out
}

const hasThreeLetters = (s: string): boolean => (s.match(/\p{L}/gu) ?? []).length >= 3

function shortLiterals(): string[] {
  const hits: string[] = []
  for (const full of walk(SRC_ROOT)) {
    const rel = relative(SRC_ROOT, full).split(sep).join('/')
    if (EXCLUDED.some((re) => re.test(rel))) continue
    const sf = ts.createSourceFile(full, readFileSync(full, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
    const visit = (node: ts.Node): void => {
      if (ts.isJsxText(node)) {
        const text = node.text.trim()
        if (text && hasThreeLetters(text)) hits.push(`${rel} : texte « ${text.slice(0, 40)} »`)
      } else if (
        ts.isJsxAttribute(node) &&
        TEXT_ATTRS.has(node.name.getText()) &&
        node.initializer &&
        ts.isStringLiteral(node.initializer) &&
        hasThreeLetters(node.initializer.text)
      ) {
        hits.push(`${rel} : ${node.name.getText()}="${node.initializer.text.slice(0, 40)}"`)
      } else if (
        ts.isJsxAttribute(node) &&
        TEXT_ATTRS.has(node.name.getText()) &&
        node.initializer &&
        ts.isJsxExpression(node.initializer) &&
        node.initializer.expression &&
        (ts.isNoSubstitutionTemplateLiteral(node.initializer.expression) || ts.isTemplateExpression(node.initializer.expression))
      ) {
        const tpl = node.initializer.expression
        const fixed = ts.isTemplateExpression(tpl) ? [tpl.head.text, ...tpl.templateSpans.map((sp) => sp.literal.text)].join(' ') : tpl.text
        if (hasThreeLetters(fixed)) hits.push(`${rel} : ${node.name.getText()}={modèle « ${fixed.slice(0, 40)} »}`)
      }
      ts.forEachChild(node, visit)
    }
    visit(sf)
  }
  return hits
}

describe('garde-rail : texte JSX court écrit en dur (plafond gelé)', () => {
  it(`au plus ${SHORT_LITERAL_CEILING} littéraux ; chaque nouveau passe par un manifest ou un dictionnaire par langue`, () => {
    const hits = shortLiterals()
    expect(
      hits.length,
      `Littéraux JSX courts : ${hits.length} (plafond ${SHORT_LITERAL_CEILING}). Utiliser formatMessage() ou un dictionnaire Record<Locale, T>.\n${hits.join('\n')}`,
    ).toBeLessThanOrEqual(SHORT_LITERAL_CEILING)
  })
})
