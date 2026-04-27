/**
 * Build step manifests i18n : parse les fichiers TOML de
 * apps/web/src/lib/i18n/manifests/ et genere des modules TS typés dans
 * apps/web/src/lib/i18n/generated/.
 *
 * Utilisation :
 *   node apps/web/scripts/build_i18n_manifests.mjs
 *
 * Conformement au PLAN_META_FOUNDATIONS_GO § 3.4 : chaque manifest produit un
 * objet `as const` qui sert de source de verite pour les types TypeScript des
 * cles. Le runtime `lib/i18n/format.ts` consomme ces modules pour resoudre les
 * strings via intl-messageformat.
 *
 * Ce script ne valide pas les placeholders ICU : la validation runtime est
 * faite par intl-messageformat lors du t(). Le script verifie seulement :
 *   - Toute cle a `fr` ET `en` (echec sinon)
 *   - Pas de cle vide
 */
import { readdir, readFile, writeFile, mkdir } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { dirname, join, basename } from 'node:path'
import { fileURLToPath } from 'node:url'

import TOML from '@iarna/toml'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MANIFESTS_DIR = join(__dirname, '..', 'src', 'lib', 'i18n', 'manifests')
const OUTPUT_DIR = join(__dirname, '..', 'src', 'lib', 'i18n', 'generated')

const REQUIRED_LOCALES = ['fr', 'en']

/**
 * Aplatit un objet TOML imbrique en map<dotted_key, {fr, en}>.
 * Ex: {common: {period: {all: {fr, en}}}} -> {"common.period.all": {fr, en}}.
 */
function flattenManifest(obj, prefix = '') {
  const out = {}
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      const isLeaf = REQUIRED_LOCALES.every((l) => typeof v[l] === 'string')
      if (isLeaf) {
        for (const required of REQUIRED_LOCALES) {
          if (typeof v[required] !== 'string' || v[required].trim() === '') {
            throw new Error(`manifest leaf "${path}" missing locale "${required}"`)
          }
        }
        const extras = Object.keys(v).filter((k2) => !REQUIRED_LOCALES.includes(k2))
        if (extras.length > 0) {
          throw new Error(`manifest leaf "${path}" has unsupported locales: ${extras.join(', ')}`)
        }
        out[path] = { fr: v.fr, en: v.en }
      } else {
        Object.assign(out, flattenManifest(v, path))
      }
    } else {
      throw new Error(`manifest key "${path}" must be a [section] table, got ${typeof v}`)
    }
  }
  return out
}

function generateTSModule(domainName, flat) {
  const sortedKeys = Object.keys(flat).sort()
  const lines = []
  lines.push('// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.')
  lines.push(`// Source : apps/web/src/lib/i18n/manifests/${domainName}.toml`)
  lines.push('')
  lines.push(`export const ${domainName}Manifest = {`)
  for (const key of sortedKeys) {
    const v = flat[key]
    const fr = JSON.stringify(v.fr)
    const en = JSON.stringify(v.en)
    lines.push(`  ${JSON.stringify(key)}: { fr: ${fr}, en: ${en} },`)
  }
  lines.push('} as const')
  lines.push('')
  lines.push(`export type ${capitalize(domainName)}ManifestKey = keyof typeof ${domainName}Manifest`)
  lines.push('')
  return lines.join('\n')
}

function capitalize(s) {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

async function main() {
  if (!existsSync(MANIFESTS_DIR)) {
    console.error(`manifests directory not found: ${MANIFESTS_DIR}`)
    process.exit(1)
  }
  if (!existsSync(OUTPUT_DIR)) {
    await mkdir(OUTPUT_DIR, { recursive: true })
  }

  const files = (await readdir(MANIFESTS_DIR)).filter((f) => f.endsWith('.toml'))
  if (files.length === 0) {
    console.error('no .toml manifests found')
    process.exit(1)
  }

  let totalKeys = 0
  for (const f of files) {
    const domain = basename(f, '.toml')
    const raw = await readFile(join(MANIFESTS_DIR, f), 'utf8')
    const parsed = TOML.parse(raw)
    const flat = flattenManifest(parsed)
    const ts = generateTSModule(domain, flat)
    const outPath = join(OUTPUT_DIR, `${domain}.ts`)
    await writeFile(outPath, ts, 'utf8')
    console.log(`  ${f} -> ${Object.keys(flat).length} cles -> ${outPath}`)
    totalKeys += Object.keys(flat).length
  }
  console.log(`OK: ${files.length} manifests, ${totalKeys} cles totales.`)
}

main().catch((err) => {
  console.error('build_i18n_manifests failed:', err.message)
  process.exit(1)
})
