/// <reference types="node" />
// @vitest-environment node
/**
 * Générateur de manifests pour N langues (ADR 0040, étape 4 du plan i18n) :
 *  - 4.1 une feuille porte TOUTES les langues de `config/locales.toml`, rien d'autre ; l'erreur
 *    nomme le fichier, la clé et la langue ;
 *  - 4.2 `--pseudo xx` fabrique `xx` depuis `en` (`⟦…⟧`) sans toucher aux fichiers versionnés, en
 *    gardant tout placeholder ICU : sur TOUTES les vraies clés, le message `xx` donne la même
 *    sortie que `en` entre les deux délimiteurs, pour les mêmes variables.
 *
 * Le module testé est `scripts/manifests.mjs` (celui qu'appelle `build_i18n_manifests.mjs`).
 */
import { execFileSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import TOML from '@iarna/toml'
import { IntlMessageFormat } from 'intl-messageformat'
import { describe, expect, it } from 'vitest'
import { substituteHpToken } from '@/lib/damage/effectiveHp'
import { formatMessage } from '@/lib/i18n/format'
import { adminManifest } from '@/lib/i18n/generated/admin'
import { commonManifest } from '@/lib/i18n/generated/common'
import { sessionManifest } from '@/lib/i18n/generated/session'
import { timeseriesManifest } from '@/lib/i18n/generated/timeseries'
// @ts-expect-error — script Node hors du programme tsc d'apps/web (frontière ESM/TS).
import { loadLocalesConfig } from '../../../scripts/locales.mjs'
import {
  flattenManifest,
  listManifestDomains,
  loadManifest,
  pseudoMessage,
  renderManifestModule,
  withPseudoLocale,
  // @ts-expect-error — idem.
} from '../../../scripts/manifests.mjs'

const WEB_ROOT = process.cwd()
const MANIFESTS_DIR = join(WEB_ROOT, 'src', 'lib', 'i18n', 'manifests')
const GENERATED_DIR = join(WEB_ROOT, 'src', 'lib', 'i18n', 'generated')
const GENERATOR = join(WEB_ROOT, 'scripts', 'build_i18n_manifests.mjs')

type Leaf = Record<string, string>
type Flat = Record<string, Leaf>

const CODES: string[] = (loadLocalesConfig() as { locales: { code: string }[] }).locales.map((l) => l.code)

function flattenOk(obj: unknown, codes: string[]): Flat {
  return flattenManifest(obj, codes, 'demo.toml') as Flat
}

function flattenError(obj: unknown, codes: string[]): string {
  try {
    flattenManifest(obj, codes, 'demo.toml')
  } catch (e) {
    return (e as Error).message
  }
  throw new Error('une erreur était attendue')
}

describe('générateur de manifests — langues lues dans config/locales.toml (4.1)', () => {
  it('toute feuille des vrais manifests porte toutes les langues listées', () => {
    let keys = 0
    for (const domain of listManifestDomains(MANIFESTS_DIR) as string[]) {
      const flat = loadManifest(MANIFESTS_DIR, domain, CODES) as Flat
      for (const [key, leaf] of Object.entries(flat)) {
        expect(Object.keys(leaf), `${domain}.toml : ${key}`).toEqual(CODES)
        keys++
      }
    }
    expect(keys).toBeGreaterThan(3000)
  })

  it('une langue listée absente d\'une feuille : l\'erreur nomme fichier, clé et langue', () => {
    const msg = flattenError({ common: { period: { all: { fr: 'Tout' } } } }, ['fr', 'en'])
    expect(msg).toBe('demo.toml : cle "common.period.all" : langue manquante "en"')
  })

  it('la 3e langue listée est exigée de chaque feuille (D-4 : complète ou refusée)', () => {
    const tree = { a: { b: { fr: 'x', en: 'y' } } }
    expect(flattenError(tree, ['fr', 'en', 'es'])).toBe('demo.toml : cle "a.b" : langue manquante "es"')
    expect(flattenOk({ a: { b: { fr: 'x', en: 'y', es: 'z' } } }, ['fr', 'en', 'es'])).toEqual({
      'a.b': { fr: 'x', en: 'y', es: 'z' },
    })
  })

  it('une langue en trop (non listée) reste une erreur', () => {
    expect(flattenError({ a: { fr: 'x', en: 'y', de: 'z' } }, ['fr', 'en'])).toBe(
      'demo.toml : cle "a" : langue non listee dans config/locales.toml : de',
    )
  })

  it('une langue vide est refusée, un scalaire hors table aussi', () => {
    expect(flattenError({ a: { fr: 'x', en: '  ' } }, ['fr', 'en'])).toBe('demo.toml : cle "a" : langue "en" vide')
    expect(flattenError({ a: 'x' }, ['fr', 'en'])).toContain('doit etre une table [section]')
  })

  it('le module généré garde sa forme : une cellule par langue, dans l\'ordre de la liste', () => {
    const flat = flattenOk({ k: { fr: 'a', en: 'b' } }, ['fr', 'en'])
    const text = renderManifestModule('demo_dom', flat, ['fr', 'en']) as string
    expect(text).toContain('export const demoDomManifest = {')
    expect(text).toContain('  "k": { fr: "a", en: "b" },')
    expect(text).toContain('export type DemoDomManifestKey = keyof typeof demoDomManifest')
  })

  it('les modules versionnés sont ceux que le générateur rend pour les langues listées', () => {
    for (const domain of listManifestDomains(MANIFESTS_DIR) as string[]) {
      const flat = loadManifest(MANIFESTS_DIR, domain, CODES)
      const onDisk = readFileSync(join(GENERATED_DIR, `${domain}.ts`), 'utf8')
      expect(onDisk, domain).toBe(renderManifestModule(domain, flat, CODES))
    }
  })
})

/** Feuilles `{ <langue>: texte }` d'un manifest, par chemin pointé, lues sans le contrôle du générateur. */
function rawLeaves(toml: string): Flat {
  const out: Flat = {}
  const walk = (node: Record<string, unknown>, prefix: string): void => {
    for (const [k, v] of Object.entries(node)) {
      const path = prefix ? `${prefix}.${k}` : k
      const table = v as Record<string, unknown>
      if (Object.values(table).some((x) => typeof x === 'string')) out[path] = table as Leaf
      else walk(table, path)
    }
  }
  walk(TOML.parse(toml) as Record<string, unknown>, '')
  return out
}

describe('générateur de manifests — chaque texte compile en message ICU (compléments 11.C)', () => {
  it("un texte qui ne compile pas est refusé : fichier, clé, langue et cause", () => {
    expect(flattenError({ a: { fr: 'ok', en: 'Dossier {slug}/ sans valeur {' } }, ['fr', 'en'])).toMatch(
      /^demo\.toml : cle "a" : langue "en" : texte non compilable en message ICU \(/,
    )
    // Une balise litterale non fermee (<ton-xuid>) est refusee comme une accolade nue.
    expect(flattenError({ a: { fr: 'Renomme en <ton-xuid>.db', en: 'x' } }, ['fr', 'en'])).toContain('langue "fr" : texte non compilable')
    // La forme echappee compile.
    expect(flattenOk({ a: { fr: "en '<ton-xuid>'.db", en: "data/'{slug}'/" } }, ['fr', 'en'])).toBeTruthy()
  })

  it("CHAQUE feuille de CHAQUE manifest compile avec intl-messageformat (le runtime avale l'échec)", () => {
    const failures: string[] = []
    for (const domain of listManifestDomains(MANIFESTS_DIR) as string[]) {
      // Lecture SANS le controle du generateur : la garde ne depend pas de l'outil qu'elle surveille.
      const raw = readFileSync(join(MANIFESTS_DIR, `${domain}.toml`), 'utf8')
      const flat = rawLeaves(raw)
      for (const [key, leaf] of Object.entries(flat)) {
        for (const code of CODES) {
          try {
            new IntlMessageFormat(leaf[code] ?? '', code)
          } catch (e) {
            failures.push(`${domain}.toml : ${key} [${code}] : ${(e as Error).message.split(/\r?\n/)[0]}`)
          }
        }
      }
    }
    expect(failures).toEqual([])
  })

  it("les cinq textes corriges s'affichent tels qu'ecrits (accolades et chevrons litteraux), dans chaque langue", () => {
    for (const code of CODES) {
      const body = formatMessage(adminManifest, 'admin.titles.register_help_body', code as never)
      expect(body, code).toContain('data/titles/{slug}/')
      expect(body, code).toContain('config/titles/{slug}/')
      expect(body, code).toContain('mappings/{fields,assets,outcomes,capabilities}.toml')
      expect(body, code).not.toContain("'{")

      const lowConf = formatMessage(commonManifest, 'common.onboarding.import_err_low_confidence', code as never)
      expect(lowConf, code).toMatch(/<(ton|your)-xuid>\.db/)
      expect(lowConf, code).not.toContain("'<")
    }
  })

  it('le jeton {{HP}} survit au formatage ICU et est ensuite remplace par le bareme du titre (225 / 115)', () => {
    const cases: [string, string][] = []
    for (const code of CODES) {
      cases.push(
        [code, formatMessage(commonManifest, 'common.charts.efficiency_tooltip', code as never)],
        [code, formatMessage(sessionManifest, 'session.detail.net_lives_tooltip', code as never)],
        [code, formatMessage(timeseriesManifest, 'timeseries.progression.net_lives_tooltip', code as never)],
      )
    }
    for (const [code, text] of cases) {
      expect(text, code).toContain('{{HP}}')
      expect(text, code).not.toContain("'{{")
      for (const hp of [225, 115]) {
        const out = substituteHpToken(text, hp)
        expect(out, `${code} ${hp}`).toContain(String(hp))
        expect(out, `${code} ${hp}`).not.toContain('{{')
        expect(out, `${code} ${hp}`).not.toContain(`'${hp}'`)
      }
    }
  })
})

// --- 4.2 pseudo-langue ---------------------------------------------------------------------

// Types d'éléments de l'AST ICU (`@formatjs/icu-messageformat-parser`, enum TYPE).
const T_LITERAL = 0
const T_ARGUMENT = 1
const T_NUMBER = 2
const T_DATE = 3
const T_TIME = 4
const T_SELECT = 5
const T_PLURAL = 6
const T_POUND = 7
const T_TAG = 8

interface IcuElement {
  type: number
  value: string
  options?: Record<string, { value: IcuElement[] }>
  children?: IcuElement[]
}

/** Variables à fournir à un message : une valeur par argument, selon sa nature. */
function collectArgs(elements: IcuElement[], into: Map<string, number>): void {
  for (const e of elements) {
    if (e.type !== T_LITERAL && e.type !== T_POUND) into.set(e.value, e.type)
    for (const opt of Object.values(e.options ?? {})) collectArgs(opt.value, into)
    if (e.children) collectArgs(e.children, into)
  }
}

function sampleVars(args: Map<string, number>, plural: number): Record<string, unknown> {
  const vars: Record<string, unknown> = {}
  for (const [name, type] of args) {
    if (type === T_PLURAL || type === T_NUMBER) vars[name] = plural
    else if (type === T_DATE || type === T_TIME) vars[name] = new Date(0)
    else if (type === T_SELECT) vars[name] = 'other'
    else if (type === T_TAG) vars[name] = (chunks: unknown[]) => chunks.join('')
    else if (type === T_ARGUMENT) vars[name] = 'V'
  }
  return vars
}

function render(message: string, vars: Record<string, unknown>): string {
  const out = new IntlMessageFormat(message, 'en').format(vars)
  return Array.isArray(out) ? out.join('') : String(out)
}

/** Code d'erreur ICU si le message ne se compile pas, sinon null. */
function parseError(message: string): string | null {
  try {
    new IntlMessageFormat(message, 'en')
    return null
  } catch (e) {
    return String((e as Error).message).split('\n')[0]
  }
}

describe('générateur de manifests — pseudo-langue (4.2)', () => {
  it('pseudoMessage encadre sans toucher au contenu', () => {
    expect(pseudoMessage('Last year')).toBe('⟦Last year⟧')
    expect(pseudoMessage('{n, plural, one {# match} other {# matches}}')).toBe(
      '⟦{n, plural, one {# match} other {# matches}}⟧',
    )
  })

  it('withPseudoLocale ajoute la langue sans modifier fr ni en', () => {
    const flat = { k: { fr: 'a', en: 'b' } }
    expect(withPseudoLocale(flat, 'xx', 'en')).toEqual({ k: { fr: 'a', en: 'b', xx: '⟦b⟧' } })
    expect(flat).toEqual({ k: { fr: 'a', en: 'b' } })
  })

  it('exemples réels : pluriel et texte simple', () => {
    const flat = withPseudoLocale(loadManifest(MANIFESTS_DIR, 'common', CODES), 'xx', 'en') as Flat
    expect(flat['common.period.last_1y']?.xx).toBe('⟦Last year⟧')
    expect(flat['common.kpi.matches_count']?.xx).toBe('⟦{n, plural, one {# match} other {# matches}}⟧')
    expect(render(flat['common.kpi.matches_count']!.xx!, { n: 1 })).toBe('⟦1 match⟧')
    expect(render(flat['common.kpi.matches_count']!.xx!, { n: 5 })).toBe('⟦5 matches⟧')
  })

  it('sur toutes les vraies clés : mêmes placeholders, même sortie entre les délimiteurs', () => {
    const natures = { plural: 0, select: 0, argument: 0, pound: 0, keys: 0 }
    for (const domain of listManifestDomains(MANIFESTS_DIR) as string[]) {
      const flat = withPseudoLocale(loadManifest(MANIFESTS_DIR, domain, CODES), 'xx', 'en') as Flat
      for (const [key, leaf] of Object.entries(flat)) {
        natures.keys++
        const where = `${domain} : ${key}`
        expect(leaf.xx, where).toBe(`⟦${leaf.en}⟧`)
        // Tout message source compile (garde 11.C) : la pseudo-langue le garde compilable.
        expect(parseError(leaf.en!), where).toBeNull()
        expect(parseError(leaf.xx!), where).toBeNull()
        const args = new Map<string, number>()
        const ast = new IntlMessageFormat(leaf.en!, 'en').getAst() as IcuElement[]
        collectArgs(ast, args)
        const xxArgs = new Map<string, number>()
        collectArgs(new IntlMessageFormat(leaf.xx!, 'en').getAst() as IcuElement[], xxArgs)
        expect([...xxArgs.entries()], where).toEqual([...args.entries()])
        for (const plural of [0, 1, 2, 5]) {
          const vars = sampleVars(args, plural)
          expect(render(leaf.xx!, vars), `${where} (n=${plural})`).toBe(`⟦${render(leaf.en!, vars)}⟧`)
        }
        for (const type of args.values()) {
          if (type === T_PLURAL) natures.plural++
          if (type === T_SELECT) natures.select++
          if (type === T_ARGUMENT) natures.argument++
        }
        if (/#/.test(leaf.en!)) natures.pound++
      }
    }
    // L'échantillon couvre bien les formes ICU qui existent dans les vrais manifests.
    expect(natures.keys).toBeGreaterThan(3000)
    expect(natures.plural).toBeGreaterThan(30)
    expect(natures.argument).toBeGreaterThan(100)
    expect(natures.select).toBeGreaterThan(0)
    expect(natures.pound).toBeGreaterThan(30)
  })

  it('CLI --pseudo xx : écrit dans le dossier demandé, fichiers versionnés et locales.toml intacts', () => {
    const out = mkdtempSync(join(tmpdir(), 'levelup-pseudo-test-'))
    const tomlPath = resolve(WEB_ROOT, '..', '..', 'config', 'locales.toml')
    const before = {
      common: readFileSync(join(GENERATED_DIR, 'common.ts'), 'utf8'),
      locales: readFileSync(join(GENERATED_DIR, 'locales.ts'), 'utf8'),
      toml: readFileSync(tomlPath, 'utf8'),
    }
    try {
      const log = execFileSync(process.execPath, [GENERATOR, '--pseudo', 'xx', '--out', out], { encoding: 'utf8' })
      expect(log).toContain('langues : fr, en, xx')
      const common = readFileSync(join(out, 'common.ts'), 'utf8')
      expect(common).toContain('xx: "⟦Last year⟧"')
      expect(readFileSync(join(out, 'locales.ts'), 'utf8')).toContain("KNOWN_LOCALES = ['fr', 'en', 'xx']")
      expect(readFileSync(join(out, 'locales_gen.go'), 'utf8')).toContain('{code: "xx"')
      expect(readFileSync(join(GENERATED_DIR, 'common.ts'), 'utf8')).toBe(before.common)
      expect(readFileSync(join(GENERATED_DIR, 'locales.ts'), 'utf8')).toBe(before.locales)
      expect(readFileSync(tomlPath, 'utf8')).toBe(before.toml)
    } finally {
      rmSync(out, { recursive: true, force: true })
    }
  })

  it('CLI : --out sans --pseudo et option inconnue sont refusés', () => {
    for (const args of [['--out', 'x'], ['--bogus']]) {
      expect(() => execFileSync(process.execPath, [GENERATOR, ...args], { stdio: 'pipe' })).toThrow()
    }
  })
})
