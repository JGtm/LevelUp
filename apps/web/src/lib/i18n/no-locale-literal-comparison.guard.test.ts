/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (ADR 0040, D-8) — comparaisons de langue littérales hors de `lib/i18n/` et
 * `lib/formatters/` : `locale === 'en'`, `lang !== 'fr'`, `isEn`, `isEN`, `isFr`, `isFR`.
 *
 * Une comparaison binaire fr/en sert la 3e langue dans la mauvaise (`locale === 'en' ? 'en' :
 * 'fr'`). La langue passe par `normalizeLocale()` puis par une table `Record<Locale, T>`, un
 * manifest (`formatMessage`) ou `intlLocale()` — jamais par un littéral.
 *
 * Deux compteurs, tous deux À 0 (INTERDICTIONS) :
 *  - le PRINCIPAL : plus aucune comparaison de langue littérale ;
 *  - les lignes balisées `i18n-contrat` (sous-chaîne sur la ligne ou sur la ligne directement
 *    au-dessus) qui choisissaient un champ d'API suffixé `_fr` / `_en` : le contrat d'API sert UN
 *    libellé résolu par le serveur dans la langue de la requête (ADR 0040 D2 ; plan
 *    `.ai/PLAN_I18N_MULTILANGUE.md`, étape 6). La balise ne sert plus : la réintroduire échoue ici.
 *
 * Troisième compteur, à 0 (étape 4, item 4.7) : un littéral `'fr'` / `'en'` comme DÉFAUT d'une valeur de langue
 * (`locale = 'fr'`, `locale ?? 'fr'`) fige la langue par défaut hors de `config/locales.toml` ; le défaut est
 * `DEFAULT_LOCALE`.
 *
 * Regex : celle de l'état des lieux (`.ai/V8/ETAT_DES_LIEUX_I18N_MULTILANGUE_2026-10-23.md`,
 * section A), étendue aux variantes de casse `isEN` / `isFR` (étape 3, lot B). Périmètre : voir
 * `localeScan.testutil.ts`.
 */
import { describe, expect, it } from 'vitest'
import { formatOverflow, languageDefaultLiteralLines, scanLanguageDefaultLiterals, scanLines } from './localeScan.testutil'

/** Interdit : toute comparaison de langue littérale non balisée. */
const LOCALE_LITERAL_CEILING = 0

/** Interdit : aucun littéral de langue comme défaut d'un paramètre ou d'une valeur de langue. */
const LANGUAGE_DEFAULT_LITERAL_CEILING = 0

/** Interdit : choix d un champ d API suffixé (`label_fr` / `label_en`) — le serveur sert le libellé résolu. */
const CONTRACT_FIELD_CEILING = 0

const CONTRACT_TAG = 'i18n-contrat'

const LOCALE_LITERAL =
  /(locale|lang|Locale|appLocale|shellLocale)\s*(===|!==)\s*'(fr|en)'|\bis(En|EN|Fr|FR)\b/

const CONTRACT_HINT =
  "Le contrat d'API ne porte plus de paire `x_fr` / `x_en` : le serveur renvoie UN champ résolu dans la langue de la requête " +
  '(`label`, `name`...). Lire ce champ tel quel ; pour un libellé par langue du document de rejeu, passer par pickText(texte, locale) ' +
  '(lib/i18n/pickText).'

const HINT =
  "Passer par normalizeLocale() puis une table Record<Locale, T> (dictionnaire de la feature), un manifest " +
  '(formatMessage + lib/i18n/manifests/*.toml) ou intlLocale(locale) / formatNumber / formatDate pour un format.'

describe('garde-rail langue — comparaisons de langue littérales', () => {
  const hits = scanLines(LOCALE_LITERAL, { contractTag: CONTRACT_TAG })
  const plain = hits.filter((h) => !h.tagged)
  const contract = hits.filter((h) => h.tagged)

  it('aucune comparaison de langue littérale (hors champs d\'API balisés)', () => {
    expect(plain.length, formatOverflow(plain, LOCALE_LITERAL_CEILING, HINT)).toBeLessThanOrEqual(LOCALE_LITERAL_CEILING)
  })

  it('aucun champ d\'API suffixé (_fr / _en) choisi par la langue : le serveur sert le libellé résolu', () => {
    expect(contract.length, formatOverflow(contract, CONTRACT_FIELD_CEILING, CONTRACT_HINT)).toBeLessThanOrEqual(
      CONTRACT_FIELD_CEILING,
    )
  })

  it("aucun littéral de langue comme défaut d'un paramètre de langue (DEFAULT_LOCALE)", () => {
    const hits = scanLanguageDefaultLiterals()
    expect(
      hits.length,
      formatOverflow(hits, LANGUAGE_DEFAULT_LITERAL_CEILING, "Écrire `locale = DEFAULT_LOCALE` (lib/i18n/locale) : le défaut de l'application vient de config/locales.toml."),
    ).toBeLessThanOrEqual(LANGUAGE_DEFAULT_LITERAL_CEILING)
  })

  it('auto-preuve : la détection des défauts littéraux voit les formes visées et ignore les autres', () => {
    const seen = (code: string): number => languageDefaultLiteralLines(code).length
    expect(seen("function f(locale = 'fr') {}")).toBe(1)
    expect(seen("const f = (locale: Locale = 'en') => locale")).toBe(1)
    expect(seen("function g({ lang = 'fr' }: P) {}")).toBe(1)
    expect(seen("const l = o.locale ?? 'fr'")).toBe(1)
    expect(seen("const l = merged.lang || 'en'")).toBe(1)
    expect(seen('function f(locale = DEFAULT_LOCALE) {}')).toBe(0)
    expect(seen("const l = o.title ?? 'fr'")).toBe(0)
    expect(seen("function h(kind = 'fr-ca') {}")).toBe(0)
  })

  it('auto-preuve : la regex voit les formes comptées et ignore les autres', () => {
    for (const s of ["locale === 'en'", "lang !== 'fr'", "appLocale === 'fr'", 'const isEn = x', 'if (isFr) {', 'const isEN = x', 'if (isFR) {']) {
      expect(LOCALE_LITERAL.test(s), s).toBe(true)
    }
    for (const s of ["locale === DEFAULT_LOCALE", "const x = 'fr'", "kind === 'en-cours'", 'isEnabled', 'isEnglish', 'this.isFrame']) {
      expect(LOCALE_LITERAL.test(s), s).toBe(false)
    }
  })
})
