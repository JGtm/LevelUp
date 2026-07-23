/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail ratchet (D7 lot C, CLAUDE.md n°6) — `lib/i18n/locale.ts` est la
 * source UNIQUE du type de langue de l'application (`Locale`, dérivé de
 * `KNOWN_LOCALES`). Toute redéclaration feature-local de l'union locale a été
 * supprimée : les alias `*Locale` sont soit migrés vers `Locale`, soit
 * ré-exportés en alias de compat `= Locale` pour les consommateurs à >5
 * fichiers (`AdminLocale`, `MatchViewLocale`, `ManifestLocale`). Ce test
 * interdit la RE-divergence : une factorisation sans garde-rail re-diverge
 * (leçon CLAUDE.md n°6 — le prédicat bot repassé de 8 à 36 copies).
 *
 * DÉTECTION (position de TYPE, sans faux positif) : l'union locale n'est
 * signalée que précédée d'un marqueur de type — `:` (annotation / type de
 * retour), `=` (alias) ou `<` (générique, ex. `Record<...>`) — pour les 4
 * formes quotes/ordres. Les occurrences en PROSE (JSDoc/commentaires : forme
 * `('fr' | 'en')` entre parenthèses ou entre backticks) sont précédées de `(`
 * ou d'un backtick, JAMAIS de `:`/`=`/`<` → non matchées : aucune allowlist
 * n'est nécessaire. Les valeurs runtime (`locale === 'en' ? 'en' : 'fr'`)
 * n'emploient pas le pipe `|` et sont donc hors d'atteinte du pattern.
 *
 * HORS SCAN : `*.test.ts(x)` (fixtures/casts `x as ...` autorisés — cf. le
 * modèle `lib/title-routing/no-title-literals.ratchet.test.ts`), `generated.ts`
 * (types OpenAPI régénérés), `routeTree.gen.ts` (routeur généré) et
 * `lib/i18n/locale.ts` lui-même (la source : `KNOWN_LOCALES` y liste les
 * valeurs runtime, pas l'union de type).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Union locale en POSITION DE TYPE : un marqueur `:`/`=`/`<`, puis les deux
// littéraux quotés `fr`/`en` séparés par le pipe `|` (quotes appariées via
// backref). Couvre les 4 formes : simple/double quote et les 2 ordres.
const LOCALE_UNION_TYPE = /[:=<]\s*(['"])(?:fr|en)\1\s*\|\s*(['"])(?:fr|en)\2/

const srcRoot = resolve(process.cwd(), 'src')
// La source canonique elle-même est exclue (elle ne contient pas l'union de
// type, mais on la borne explicitement pour être robuste à ses évolutions).
const localeSource = join(srcRoot, 'lib', 'i18n', 'locale.ts')

// Fichiers générés (jamais édités) : types OpenAPI + arbre de routes.
const GENERATED_FILES = new Set(['generated.ts', 'routeTree.gen.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      if (GENERATED_FILES.has(entry.name)) continue
      if (full === localeSource) continue
      out.push(full)
    }
  }
  return out
}

const allFiles = walk(srcRoot)

describe('garde-rail locale (source unique du type Locale : lib/i18n/locale.ts)', () => {
  it("aucune redéclaration feature-local de l'union locale en position de type", () => {
    const offenders: string[] = []
    for (const file of allFiles) {
      const lines = readFileSync(file, 'utf8').split('\n')
      lines.forEach((line, index) => {
        if (LOCALE_UNION_TYPE.test(line)) {
          offenders.push(`${file.replace(srcRoot, 'src')}:${index + 1}`)
        }
      })
    }
    expect(
      offenders,
      `Union locale à remplacer par le type central Locale (import type { Locale } from '@/lib/i18n/locale') : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
