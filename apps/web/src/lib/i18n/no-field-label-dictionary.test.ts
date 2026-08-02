/**
 * Garde-rail (règle n°6 CLAUDE.md) — aucun dictionnaire de libellés de FieldKey
 * hors des manifests TOML.
 *
 * Contexte : jusqu'au 2026-08-02, `lib/i18n/metricLabel.ts` portait deux Record
 * FR/EN indexés par FieldKey, en doublon des TOML de titre. Les deux sources ont
 * DIVERGÉ en silence (« Éliminations » côté TS contre « Frags » dans le
 * registre, « Matchs joués » contre « Matchs ») : l'écran affichait l'un ou
 * l'autre selon qu'un flag serveur exposait ou non /field-mappings.
 *
 * Pourquoi ce test EN PLUS de tools/lint-no-hardcoded-fields.mjs : ce dernier
 * compare les littéraux aux libellés EXACTS du TOML — il ne voit donc pas un
 * dictionnaire qui diverge, c'est-à-dire précisément le cas qui a causé le bug.
 * Ici on détecte la FORME (une clé canonique associée à un libellé humain),
 * quelle que soit la valeur.
 *
 * Détection : au moins DEUX associations `<field_key>: 'Libellé humain'` dans un
 * même fichier. Une association isolée peut être fortuite ; un dictionnaire en
 * aligne toujours plusieurs. Les valeurs techniques (unités, formats, tokens de
 * couleur, identifiants camelCase) sont exclues par l'heuristique de libellé.
 *
 * PÉRIMÈTRE — ce qui n'est PAS visé : les dictionnaires UI d'une feature
 * (`features/<x>/i18n.ts`, couche 1 du skill frontend-patterns). Ils indexent
 * parfois par nom de métrique des textes qui ne sont pas le libellé canonique
 * (« Morts/min » en en-tête de colonne, « Améliore ta précision » en conseil),
 * et leur migration éventuelle est un autre chantier. Le fichier doit s'appeler
 * exactement `i18n.ts` : un `*.i18n.ts` suffixé — la forme qu'avaient les
 * `fallback.i18n.ts` supprimés — reste, lui, sous surveillance.
 *
 * Source des clés : config/titles/halo_infinite/mappings/fields.toml — le
 * registre lui-même, pour que le garde-rail suive automatiquement le canonique.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, dirname, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const WEB_SRC = join(HERE, '..', '..')
const REPO_ROOT = join(WEB_SRC, '..', '..', '..')
const FIELDS_TOML = join(
  REPO_ROOT,
  'config',
  'titles',
  'halo_infinite',
  'mappings',
  'fields.toml',
)

/**
 * Exemptions — chemins autorisés à contenir de telles associations, avec
 * justification DATÉE (doctrine : aucune allowlist muette).
 */
const EXEMPTIONS: ReadonlyArray<{ pattern: RegExp; why: string }> = [
  {
    // Couche 1 (dictionnaire UI de feature) — voir PÉRIMÈTRE en tête de fichier.
    // Nom EXACT `i18n.ts` : les variantes suffixées restent scannées.
    pattern: /(^|\/)i18n\.ts$/,
    why: 'dictionnaire UI de feature (textes contextuels, pas le registre de libellés)',
  },
  {
    // Aligné sur tools/lint-no-hardcoded-fields.mjs (whitelist du 2026-07-06) :
    // libellés d'axes du chart calendrier, title-agnostic par construction.
    pattern: /(^|\/)lib\/formatters\/calendar\.ts$/,
    why: 'libellés d’axes du calendrier, title-agnostic (whitelist linter 2026-07-06)',
  },
  {
    // Aligné sur tools/lint-no-hardcoded-fields.mjs : sandbox de développement
    // /lab/charts, documentation visuelle hors surface utilisateur.
    pattern: /(^|\/)features\/lab\/ChartsShowcasePage\.tsx$/,
    why: 'sandbox /lab/charts (doc visuelle, hors surface user-facing)',
  },
]

function isExempt(rel: string): boolean {
  return EXEMPTIONS.some((e) => e.pattern.test(rel))
}

function collectSourceFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      // Modules i18n générés + manifests TOML : sorties d'outillage, pas du code.
      if (entry.name === 'generated' || entry.name === 'manifests') continue
      out.push(...collectSourceFiles(full))
      continue
    }
    if (!/\.(ts|tsx)$/.test(entry.name)) continue
    // Tests : fixtures et assertions citent légitimement des libellés.
    if (/\.test\.(ts|tsx)$/.test(entry.name)) continue
    out.push(full)
  }
  return out
}

/** Noms de sections [fields.X] du registre canonique. */
function canonicalFieldKeys(): Set<string> {
  const toml = readFileSync(FIELDS_TOML, 'utf8')
  const keys = new Set<string>()
  const re = /^\[fields\.([a-z0-9_]+)\]/gm
  let m: RegExpExecArray | null
  while ((m = re.exec(toml)) !== null) keys.add(m[1])
  return keys
}

/**
 * Un libellé humain commence par une majuscule ou contient une espace. Écarte
 * les unités (`count`), formats (`percent_1`), tokens (`var(--ac-x)`) et
 * identifiants camelCase (`matchId`), qui sont des valeurs techniques.
 */
function looksLikeHumanLabel(value: string): boolean {
  if (value.length < 2) return false
  if (/^(var\(|--|#|https?:)/.test(value)) return false
  return /^[A-ZÀ-Ý]/.test(value) || value.includes(' ')
}

/** Associations `cle: 'Libellé'` dont la clé est un FieldKey canonique. */
function findFieldLabelPairs(src: string, fieldKeys: Set<string>) {
  const hits: Array<{ line: number; key: string; value: string }> = []
  const lines = src.split('\n')
  const re = /^\s*['"]?([a-z][a-z0-9_]*)['"]?\s*:\s*(['"])((?:(?!\2).)*)\2\s*,?\s*$/
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()
    if (trimmed.startsWith('//') || trimmed.startsWith('*')) continue
    const m = re.exec(line)
    if (!m) continue
    const [, key, , value] = m
    if (!fieldKeys.has(key)) continue
    if (!looksLikeHumanLabel(value)) continue
    hits.push({ line: i + 1, key, value })
  }
  return hits
}

describe('garde-rail : aucun dictionnaire de libellés de FieldKey hors TOML', () => {
  const fieldKeys = canonicalFieldKeys()
  const files = collectSourceFiles(WEB_SRC)

  it('lit les FieldKey canoniques depuis le registre TOML', () => {
    expect(fieldKeys.size).toBeGreaterThan(30)
    expect(fieldKeys.has('kills')).toBe(true)
    expect(fieldKeys.has('win_rate')).toBe(true)
  })

  it('scanne des fichiers (le walker fonctionne)', () => {
    expect(files.length).toBeGreaterThan(100)
  })

  it("l'heuristique distingue un libellé d'une valeur technique", () => {
    expect(looksLikeHumanLabel('Frags')).toBe(true)
    expect(looksLikeHumanLabel('Taux de victoire')).toBe(true)
    expect(looksLikeHumanLabel('count')).toBe(false)
    expect(looksLikeHumanLabel('percent_1')).toBe(false)
    expect(looksLikeHumanLabel('var(--ac-kills)')).toBe(false)
    expect(looksLikeHumanLabel('matchId')).toBe(false)
  })

  it('aucun fichier ne reconstitue un dictionnaire FieldKey → libellé', () => {
    const offenders: string[] = []
    for (const file of files) {
      const rel = relative(WEB_SRC, file).replace(/\\/g, '/')
      if (isExempt(rel)) continue
      const hits = findFieldLabelPairs(readFileSync(file, 'utf8'), fieldKeys)
      if (hits.length < 2) continue
      const detail = hits.map((h) => `L${h.line} ${h.key}: "${h.value}"`).join(', ')
      offenders.push(`${rel} → ${hits.length} associations (${detail})`)
    }
    expect(
      offenders,
      'Les libellés de FieldKey viennent des manifests TOML ' +
        '(config/titles/{slug}/mappings/fields.toml), consommés par useFieldLabel / ' +
        'useMetricLabel. Un dictionnaire TS re-diverge du registre en silence.',
    ).toEqual([])
  })
})
