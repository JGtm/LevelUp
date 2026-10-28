/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (ADR 0040, D-8) — formats Intl hors de `lib/formatters/` et `lib/i18n/` :
 *  - étiquettes littérales (`'fr-FR'`, `'en-US'`, `'en-GB'`) ;
 *  - appels SANS étiquette (`x.toLocaleString()`, `toLocaleDateString(undefined, …)`, `new Intl.NumberFormat()`),
 *    rendus dans la langue du navigateur et non celle de l'interface (ajout de l'étape 4, item 4.6).
 *
 * L'étiquette de chaque langue vient de `config/locales.toml` (table générée `INTL_TAG`, lue par
 * `intlLocale()`) : un littéral fige un format de nombre ou de date dans une langue, quelle que
 * soit celle de l'interface. Le plafond est 0 : toute occurrence est une erreur.
 *
 * Regex : celle de l'état des lieux (`.ai/V8/ETAT_DES_LIEUX_I18N_MULTILANGUE_2026-10-23.md`,
 * section F). Périmètre : voir `localeScan.testutil.ts`.
 */
import { describe, expect, it } from 'vitest'
import { formatOverflow, scanLines, scanUntaggedIntl, TREE_SCAN_TIMEOUT_MS, untaggedIntlLines } from './localeScan.testutil'

/** Interdit : aucune étiquette Intl littérale hors des deux dossiers propriétaires de la langue. */
const INTL_TAG_LITERAL_CEILING = 0

const INTL_TAG_LITERAL = /'(fr-FR|en-US|en-GB)'/

/** Interdit : aucun appel de formatage sans étiquette hors des deux dossiers propriétaires de la langue. */
const UNTAGGED_INTL_CEILING = 0

describe('garde-rail langue — étiquettes Intl littérales', () => {
  it('aucune étiquette Intl littérale', () => {
    const hits = scanLines(INTL_TAG_LITERAL)
    expect(
      hits.length,
      formatOverflow(
        hits,
        INTL_TAG_LITERAL_CEILING,
        "Passer par intlLocale(locale) (table INTL_TAG de config/locales.toml) ou formatNumber / formatDate, qui prennent l'étiquette de la langue.",
      ),
    ).toBeLessThanOrEqual(INTL_TAG_LITERAL_CEILING)
  }, TREE_SCAN_TIMEOUT_MS)

  it("aucun appel de formatage sans étiquette (langue du navigateur au lieu de celle de l'interface)", () => {
    const hits = scanUntaggedIntl()
    expect(
      hits.length,
      formatOverflow(
        hits,
        UNTAGGED_INTL_CEILING,
        "Passer l'étiquette de la langue : x.toLocaleString(intlLocale(locale)), formatNumber(x, intlLocale(locale)) ou formatDate.",
      ),
    ).toBeLessThanOrEqual(UNTAGGED_INTL_CEILING)
  }, TREE_SCAN_TIMEOUT_MS)

  it('auto-preuve : la détection voit les appels sans étiquette et ignore les autres', () => {
    const seen = (code: string): number => untaggedIntlLines(code).length
    expect(seen('x.toLocaleString()')).toBe(1)
    expect(seen('d.toLocaleDateString(undefined, { dateStyle: "short" })')).toBe(1)
    expect(seen('d.toLocaleTimeString({ hour: "2-digit" })')).toBe(1)
    expect(seen('new Intl.NumberFormat()')).toBe(1)
    expect(seen('Intl.DateTimeFormat({ year: "numeric" })')).toBe(1)
    expect(seen('x\n  .toLocaleString(\n  )')).toBe(1)
    expect(seen('x.toLocaleString(intlLocale(locale))')).toBe(0)
    expect(seen('new Intl.NumberFormat(tag, { style: "percent" })')).toBe(0)
    expect(seen('x.toFixed(2)')).toBe(0)
  })

  it("auto-preuve : la regex voit les trois étiquettes et ignore le reste", () => {
    for (const s of ["toLocaleString('fr-FR')", "locale === 'en' ? 'en-GB' : 'fr-FR'", "'en-US'"]) {
      expect(INTL_TAG_LITERAL.test(s), s).toBe(true)
    }
    for (const s of ["intlLocale(locale)", "'fr'", "'de-DE'"]) {
      expect(INTL_TAG_LITERAL.test(s), s).toBe(false)
    }
  })
})
