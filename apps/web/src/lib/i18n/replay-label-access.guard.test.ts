/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (ADR 0040, plan i18n étape 6 lot A) — aucun accès `.fr` / `.en` (ni `['fr']` /
 * `['en']`) sur un libellé du document de rejeu hors de `lib/i18n/`.
 *
 * Pourquoi : le document de rejeu garde un libellé PAR LANGUE (`{ en, fr, … }`, une propriété par
 * langue de `config/locales.toml`). Lire `label.fr` ou `locale === 'fr' ? l.fr : l.en` fige deux
 * langues dans le code : une 3e langue ajoutée à la liste serait servie dans la langue de repli
 * sans qu'aucun test ne rougisse. Le seul lecteur est `pickText` (`lib/i18n/pickText.ts`), qui
 * replie sur `FALLBACK_LOCALE`.
 *
 * Ce que le test interdit : une lecture d'une propriété nommée comme une langue de la liste, sur une
 * valeur dont le type est celui d'un libellé généré (`components['schemas']['Label' | 'WeaponLabel'
 * | 'VehicleLabel' | 'CalloutZone' | 'ObjectiveObjectLife' | 'VehicleWeapon']`), alias et
 * intersections compris. Détection par le compilateur : la déclaration de la propriété doit se
 * trouver sous `schemas.<un de ces six noms>`. Périmètre : `features/`, `components/`, `lib/`
 * hors `lib/i18n/`, hors tests, hors générés. Allowlist VIDE.
 */
import ts from 'typescript'
import { describe, expect, it } from 'vitest'

import { buildRealProgram, buildVirtualProgram, relativeToSrc } from './localeKeyedObjects.testutil'
import { KNOWN_LOCALES } from './locale'

/** Les schémas du contrat qui portent un libellé par langue à plat (`i18n.MarshalFlat` côté Go). */
const LABEL_SCHEMAS = new Set([
  'Label',
  'WeaponLabel',
  'VehicleLabel',
  'CalloutZone',
  'ObjectiveObjectLife',
  'VehicleWeapon',
])

/** Plafond : interdit. */
const VIOLATION_CEILING = 0

interface Access {
  file: string
  line: number
  schema: string
  text: string
}

/** Le schéma (`schemas.<Nom>`) sous lequel une propriété est déclarée, ou null. */
function declaringSchema(decl: ts.Declaration): string | null {
  let node: ts.Node | undefined = decl.parent
  while (node) {
    if (ts.isPropertySignature(node) && ts.isIdentifier(node.name)) {
      const owner = node.parent?.parent
      if (owner && ts.isPropertySignature(owner) && ts.isIdentifier(owner.name) && owner.name.text === 'schemas') {
        return node.name.text
      }
    }
    node = node.parent
  }
  return null
}

function memberName(node: ts.Node): { receiver: ts.Expression; name: string } | null {
  if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.name)) {
    return { receiver: node.expression, name: node.name.text }
  }
  if (ts.isElementAccessExpression(node) && ts.isStringLiteralLike(node.argumentExpression)) {
    return { receiver: node.expression, name: node.argumentExpression.text }
  }
  return null
}

function findAccesses(program: ts.Program, files: string[], rel: (f: string) => string): Access[] {
  const checker = program.getTypeChecker()
  const langs = new Set<string>(KNOWN_LOCALES)
  const out: Access[] = []
  for (const file of files) {
    const sf = program.getSourceFile(file)
    if (!sf) throw new Error(`fichier absent du programme : ${file}`)
    const visit = (node: ts.Node): void => {
      const member = memberName(node)
      if (member && langs.has(member.name)) {
        const type = checker.getNonNullableType(checker.getTypeAtLocation(member.receiver))
        const candidates = type.isUnion() ? type.types : [type]
        for (const t of candidates) {
          const prop = t.getProperty(member.name)
          const schema = (prop?.declarations ?? []).map(declaringSchema).find((s) => s && LABEL_SCHEMAS.has(s))
          if (schema) {
            const line = sf.getLineAndCharacterOfPosition(node.getStart(sf)).line
            out.push({ file: rel(file), line: line + 1, schema, text: sf.text.split('\n')[line]?.trim().slice(0, 100) ?? '' })
            break
          }
        }
      }
      ts.forEachChild(node, visit)
    }
    visit(sf)
  }
  return out
}

const NAMED = /\.(fr|en)\b|\[['"](fr|en)['"]\]/

const VIRTUAL_PRELUDE = [
  'interface components { schemas: {',
  '  Label: { en: string; fr: string; img?: string }',
  '  WeaponLabel: { en: string; fr: string }',
  '  Autre: { en: string; fr: string }',
  '} }',
  "type ReplayLabel = components['schemas']['Label'] & { extra?: number }",
  '',
].join('\n')

function virtual(body: string): Access[] {
  const name = '/virtual/sample.ts'
  return findAccesses(buildVirtualProgram(name, VIRTUAL_PRELUDE + body), [name], (f) => f)
}

describe('garde-rail langue — aucun accès .fr/.en sur un libellé du document de rejeu', () => {
  it(
    'les libellés générés se lisent par pickText, hors de lib/i18n/',
    () => {
      const { program, files } = buildRealProgram((text) => NAMED.test(text))
      const scanned = files.filter((f) => !relativeToSrc(f).startsWith('lib/i18n/'))
      expect(scanned.length).toBeGreaterThan(5)
      const found = findAccesses(program, scanned, relativeToSrc)
      expect(
        found.length,
        `${found.length} accès (plafond : ${VIOLATION_CEILING}, ADR 0040). Lire le libellé par ` +
          'pickText(label, locale) (lib/i18n/pickText.ts), jamais label.fr / label.en.\n' +
          found.map((a) => `  ${a.file}:${a.line} [${a.schema}] ${a.text}`).join('\n'),
      ).toBeLessThanOrEqual(VIOLATION_CEILING)
    },
    120_000,
  )

  it('auto-preuve : les accès sur un libellé généré sont vus, sous toutes leurs formes', () => {
    const seen = virtual(
      [
        "declare const l: components['schemas']['Label']",
        'export const a = l.fr',
        "export const b = l['en']",
        "declare const w: components['schemas']['WeaponLabel'] | undefined",
        'export const c = w?.fr',
        'declare const r: ReplayLabel',
        'export const d = r.en',
        "export const e = (locale: 'fr' | 'en') => (locale === 'fr' ? l.fr : l.en)",
        '',
      ].join('\n'),
    )
    const first = VIRTUAL_PRELUDE.split('\n').length
    expect(seen.map((a) => `${a.line - first + 1}:${a.schema}`)).toEqual([
      '2:Label',
      '3:Label',
      '5:WeaponLabel',
      '7:Label',
      '8:Label',
      '8:Label',
    ])
  })

  it('auto-preuve : un autre objet à clés fr/en, et les champs qui ne sont pas des langues, ne sont pas vus', () => {
    const seen = virtual(
      [
        "declare const o: components['schemas']['Autre']",
        'export const a = o.fr',
        "declare const l: components['schemas']['Label']",
        'export const b = l.img',
        'declare const loose: { fr: string; en: string }',
        'export const c = loose.en',
        '',
      ].join('\n'),
    )
    expect(seen).toEqual([])
  })
})
