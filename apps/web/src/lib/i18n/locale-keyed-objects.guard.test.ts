/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (ADR 0040, étape 4 du plan i18n, item 4.5) — tout objet littéral qui porte les clés
 * `fr` et `en` dans `features/`, `components/` et `lib/` est typé par `Locale`.
 *
 * Pourquoi : un texte `{ fr, en }` typé à la main (`{ fr: string; en: string }`) ou sans type
 * compile encore quand on ajoute une 3e langue à `config/locales.toml` ; il la laisse sans texte
 * au lieu de casser le build. Typé `Record<Locale, T>` (en place, par `satisfies`, ou par un type
 * déclaré ailleurs qui l'étend), l'ajout d'une langue casse la compilation de cet objet : la
 * langue est complète ou refusée (ADR 0040, D-4).
 *
 * Aussi interdit : le type `'fr' | 'en'` écrit en toutes lettres (il s'écrit `Locale`).
 *
 * Périmètre : les `.ts` / `.tsx` des trois dossiers, hors tests (`*.test.*`), hors dossiers de
 * support de test (`test/`, `__tests__/`) et hors fichiers générés. Allowlist VIDE : une exception
 * se règle en typant l'objet, pas en l'inscrivant ici. Détection par le compilateur TypeScript :
 * voir `localeKeyedObjects.testutil.ts`.
 */
import { describe, expect, it } from 'vitest'
import {
  buildRealProgram,
  buildVirtualProgram,
  findViolations,
  relativeToSrc,
  type Violation,
} from './localeKeyedObjects.testutil'
import { TREE_SCAN_TIMEOUT_MS } from './localeScan.testutil'

/** Interdit : aucun objet à clés de langue non typé par `Locale`. */
const VIOLATION_CEILING = 0

function describeViolations(violations: Violation[]): string {
  return violations.map((v) => `  ${v.file}:${v.line} [${v.kind}] contexte ${v.context} : ${v.text}`).join('\n')
}

const VIRTUAL_PRELUDE = "declare const KNOWN: readonly ['fr', 'en']\ntype Locale = (typeof KNOWN)[number]\n"

function virtualViolations(body: string): Violation[] {
  const name = '/virtual/sample.ts'
  const program = buildVirtualProgram(name, VIRTUAL_PRELUDE + body)
  return findViolations(program, [name], (f) => f)
}

describe('garde-rail langue — objets à clés de langue typés par Locale', () => {
  it(
    'aucun objet { fr, en } hors Record<Locale, …> ni type \'fr\' | \'en\' écrit à la main',
    () => {
      const { program, files } = buildRealProgram()
      expect(files.length).toBeGreaterThan(30)
      const violations = findViolations(program, files, relativeToSrc)
      expect(
        violations.length,
        `${violations.length} violation(s) (plafond : ${VIOLATION_CEILING}, ADR 0040). Typer l'objet par Record<Locale, T> ` +
          "(ou `satisfies Record<Locale, T>`), jamais { fr: string; en: string } ni 'fr' | 'en'.\n" +
          describeViolations(violations),
      ).toBeLessThanOrEqual(VIOLATION_CEILING)
    },
    TREE_SCAN_TIMEOUT_MS,
  )

  it('auto-preuve : les objets non typés sont vus', () => {
    const untyped = virtualViolations("export const a = { fr: 'x', en: 'y' }\n")
    expect(untyped.map((v) => v.kind)).toEqual(['untyped-literal'])
    expect(untyped[0]?.context).toBe('-')

    const handWritten = virtualViolations(
      "interface Pair { fr: string; en: string }\nexport const b: Pair = { fr: 'x', en: 'y' }\n",
    )
    expect(handWritten.map((v) => v.kind)).toEqual(['untyped-literal'])
    expect(handWritten[0]?.context).toBe('Pair')

    const nested = virtualViolations(
      "interface Row { id: string; label: { fr: string; en: string } }\nexport const c: Row[] = [{ id: 'a', label: { fr: 'x', en: 'y' } }]\n",
    )
    expect(nested).toHaveLength(1)
  }, TREE_SCAN_TIMEOUT_MS)

  it('auto-preuve : les objets typés par Locale sont acceptés', () => {
    const typed = virtualViolations(
      [
        "export const a: Record<Locale, string> = { fr: 'x', en: 'y' }",
        "export const b = { fr: 'x', en: 'y' } as const satisfies Record<Locale, string>",
        "interface Tier extends Record<Locale, string> { min: number }",
        "export const c: Tier = { fr: 'x', en: 'y', min: 1 }",
        "type Pair = Record<Locale, string>",
        "export const d: Pair = { fr: 'x', en: 'y' }",
        "export const e: Partial<Record<Locale, string>> = { fr: 'x', en: 'y' }",
        "export const f: Record<string, Record<Locale, string>> = { k: { fr: 'x', en: 'y' } }",
        "export const g = () : Record<Locale, number> => ({ fr: 1, en: 2 })",
        "export const h = { fr: 'seule clé' }",
        '',
      ].join('\n'),
    )
    expect(typed).toEqual([])
  }, TREE_SCAN_TIMEOUT_MS)

  it("auto-preuve : le type 'fr' | 'en' écrit à la main est vu", () => {
    const inline = virtualViolations("export type X = 'fr' | 'en'\nexport type Y = 'a' | 'b'\nexport type Z = 'en' | 'fr' | 'de'\n")
    expect(inline.map((v) => `${v.kind}:${v.line}`)).toEqual(['inline-union:3', 'inline-union:5'])
  }, TREE_SCAN_TIMEOUT_MS)
})
