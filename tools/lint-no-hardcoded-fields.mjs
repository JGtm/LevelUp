#!/usr/bin/env node
/**
 * Lint custom — anti-hardcode des libellés FieldKey côté React.
 *
 * Phase D du plan multi-titres : détecte les littéraux strings qui
 * correspondent **exactement** aux labels FR ou EN d'un FieldKey canonique
 * (extraits de config/titles/halo_infinite/mappings/fields.toml) en dehors
 * des fichiers autorisés.
 *
 * Usage :
 *   node tools/lint-no-hardcoded-fields.mjs
 *
 * Exit code :
 *   0 = clean
 *   1 = violations détectées
 *
 * Whitelist (fichiers autorisés à hardcoder pendant la transition Phase D) :
 *   - apps/web/src/features/* /i18n.ts                 (dicts FR/EN locaux)
 *   - apps/web/src/features/* /*.i18n.ts                (dicts FR/EN locaux)
 *   - apps/web/src/lib/i18n/fieldMappings.test.ts       (fixtures de test)
 *   - apps/web/src/components/ui/timeseries-scatter.tsx (fallback FR explicite)
 *   - apps/web/src/components/ui/timeseries-kda-bars.tsx (fallback FR explicite)
 *   - apps/web/src/features/squad/SquadContributionsPage.tsx (fallback FR explicite)
 *   - apps/web/src/features/match-view/MatchViewPage.tsx (fallback FR explicite)
 *   - apps/web/src/features/home/HomePage.tsx           (fallback FR explicite)
 *   - tous les *.test.ts / *.test.tsx                    (fixtures et assertions)
 *
 * Cette whitelist sera retirée endpoint par endpoint au fil de la migration.
 */

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const WEB_SRC = join(REPO_ROOT, 'apps/web/src')
const TOML_PATH = join(
  REPO_ROOT,
  'config/titles/halo_infinite/mappings/fields.toml',
)
const ASSETS_TOML_PATH = join(
  REPO_ROOT,
  'config/titles/halo_infinite/mappings/assets.toml',
)
const OUTCOMES_TOML_PATH = join(
  REPO_ROOT,
  'config/titles/halo_infinite/mappings/outcomes.toml',
)

// ─── 1. Extraction des labels FR + EN depuis le TOML HI ──────────────────────

function extractLabelsFromTOML(content) {
  // Parse simple : on récupère toutes les lignes `labels = { en = "X", fr = "Y" }`
  // Suffisant pour ce lint, pas besoin de parser TOML complet.
  const labels = new Set()
  const re = /labels\s*=\s*\{\s*en\s*=\s*"([^"]+)"\s*,\s*fr\s*=\s*"([^"]+)"\s*\}/g
  let m
  while ((m = re.exec(content)) !== null) {
    labels.add(m[1]) // EN
    labels.add(m[2]) // FR
  }
  return labels
}

// ─── 2. Whitelist ─────────────────────────────────────────────────────────────

/**
 * Whitelist : fichiers où les littéraux FR/EN sont la source de vérité
 * (dicts locaux, tests, le hook lui-même). Aucun composant React de
 * production ne doit être dans cette whitelist : les fallbacks explicites
 * (`labelOf('kda', 'KDA')` ou `??` après un useFieldLabel) sont détectés
 * par le scanner sans whitelist.
 */
const WHITELIST_PATTERNS = [
  /\.test\.tsx?$/,                         // tous les tests
  /\/i18n\.ts$/,                           // dicts FR/EN locaux par feature
  /\.i18n\.ts$/,                           // dicts FR/EN locaux suffixés (ex: kpi.i18n.ts)
  /\/lib\/i18n\/fieldMappings\.ts$/,       // le hook lui-même
  /\/lib\/i18n\/fieldMappings\.test\.ts$/, // le test du hook
  /\/test\/handlers\.ts$/,                  // fixtures MSW (mocks API, pas un libellé UI)
  /\/lib\/api\/types\.ts$/,                 // types TS purs (commentaires explicatifs)
  /\/lib\/api\/generated\.ts$/,             // types generes par openapi-typescript (enums du contrat OpenAPI)
  /\/features\/compare\/i18n\.ts$/,         // dict FR/EN local de compare
  /\/lib\/prestige\.ts$/,                   // dict TIER_LABELS_FR — fallback canonique
  /\/features\/palmares\/rarity\.ts$/,      // dict rarity Halo (asset Halo natif)
  /\/lib\/medalDifficulty\.ts$/,            // dict difficulty médailles Halo (Normal/Heroic/Legendary/Mythic — valeurs API)
  /\/lib\/skillTiers\.ts$/,                 // constantes paliers skill CSR/LUSR (Bronze/Silver/Gold… — valeurs API)
  /\/lib\/formatters\/rating\.ts$/,         // normalisation du libellé de méthode de note (LUSR/CSR — valeurs API, identiques FR/EN, non traduisibles)
  /\/features\/_shared\/combatProfileLabels\.ts$/, // dict descripteurs profil de combat (5 bandes canoniques, pas des FieldKey)
  /\/lib\/formatters\/calendar\.ts$/,       // dict FR/EN libellés chart calendrier (axes + winRate/wins/matches title-agnostic, jamais renommés par titre) — I2 dedup, whitelist 2026-07-06
  /\/lib\/i18n\/manifests\//,                // sources TOML i18n (whitelistees a la racine du linter mais le walker filtre .ts uniquement)
  /\/lib\/i18n\/generated\//,                // modules generes par scripts/build_i18n_manifests.mjs
  /\/lib\/i18n\/format\.tsx?$/,              // wrapper runtime ICU MessageFormat (fallback strings techniques uniquement)
  /\/features\/lab\/ChartsShowcasePage\.tsx$/, // dev sandbox /lab/charts (doc visuelle, hors scope user-facing)
]

function isWhitelisted(filePath) {
  return WHITELIST_PATTERNS.some((re) => re.test(filePath.replace(/\\/g, '/')))
}

// ─── 3. Walk de l'arborescence apps/web/src ──────────────────────────────────

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    const st = statSync(full)
    if (st.isDirectory()) {
      if (name === 'node_modules' || name === 'dist' || name.startsWith('.')) continue
      yield* walk(full)
    } else if (/\.(ts|tsx)$/.test(name)) {
      yield full
    }
  }
}

// ─── 4. Détection des violations ─────────────────────────────────────────────

function findViolations(filePath, labels) {
  const content = readFileSync(filePath, 'utf8')
  const violations = []
  const lines = content.split('\n')
  for (const label of labels) {
    // On cherche le littéral entre quotes simples ou doubles ou backticks.
    const patterns = [
      new RegExp(`'${escapeRegex(label)}'`),
      new RegExp(`"${escapeRegex(label)}"`),
      new RegExp(`\`${escapeRegex(label)}\``),
    ]
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      const trimmed = line.trim()
      // Skip lignes commentées (basique).
      if (trimmed.startsWith('//') || trimmed.startsWith('*')) continue
      // Skip lignes contenant un commentaire inline avec ce label (ex: `// "Kills" | "Deaths"`).
      const commentIdx = line.indexOf('//')
      const codeOnly = commentIdx >= 0 ? line.slice(0, commentIdx) : line
      for (const pattern of patterns) {
        if (pattern.test(codeOnly)) {
          // Skip si le littéral est le 2e argument d'un appel labelOf(key, fallback)
          // (pattern de fallback explicite vers la valeur FR locale).
          const fallbackPattern = new RegExp(
            `labelOf\\([^)]+,\\s*['"\`]${escapeRegex(label)}['"\`]\\s*\\)`,
          )
          if (fallbackPattern.test(codeOnly)) continue
          // Skip si le littéral est utilisé comme valeur de fallback `?? 'X'`
          // (pattern de fallback explicite côté composant).
          const nullishCoalescePattern = new RegExp(
            `\\?\\?\\s*['"\`]${escapeRegex(label)}['"\`]`,
          )
          if (nullishCoalescePattern.test(codeOnly)) continue
          // Skip si le littéral est utilisé dans une comparaison d'enum API
          // (ex: `rating_type === 'LUSR'`). C'est une valeur de donnée, pas
          // un libellé d'affichage.
          const enumComparePattern = new RegExp(
            `[=!]==?\\s*['"\`]${escapeRegex(label)}['"\`]`,
          )
          if (enumComparePattern.test(codeOnly)) continue
          // Skip si le littéral est l'argument d'une opération de collection
          // sur une valeur d'enum (ex: `types.has('CSR')`, `set.includes('LUSR')`,
          // `map.get('CSR')`). C'est une valeur de donnée, pas un libellé d'affichage.
          const collectionMembershipPattern = new RegExp(
            `\\.(has|includes|get)\\(\\s*['"\`]${escapeRegex(label)}['"\`]`,
          )
          if (collectionMembershipPattern.test(codeOnly)) continue
          violations.push({ line: i + 1, label, snippet: line.trim().slice(0, 120) })
        }
      }
    }
  }
  return violations
}

function escapeRegex(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// ─── 5. Main ─────────────────────────────────────────────────────────────────

function readTOMLOrEmpty(path) {
  try {
    return readFileSync(path, 'utf8')
  } catch {
    // Le fichier est optionnel (assets.toml, outcomes.toml peuvent ne pas exister
    // pour certains titres). Pas d'erreur fatale.
    return ''
  }
}

function main() {
  let tomlContent
  try {
    tomlContent = readFileSync(TOML_PATH, 'utf8')
  } catch (err) {
    console.error(`erreur de lecture ${TOML_PATH}: ${err.message}`)
    process.exit(1)
  }

  const labels = extractLabelsFromTOML(tomlContent)
  if (labels.size === 0) {
    console.error(`aucun label extrait de ${TOML_PATH} — parser cassé ?`)
    process.exit(1)
  }

  // Phase 4.3 plan finition multi-titres : étendre le scan aux assets et outcomes.
  // Un libellé d'asset (ex : "Héroïque") ou d'outcome (ex : "Victoire") hardcodé
  // dans un composant React est tout aussi problématique qu'un libellé de FieldKey.
  const assetsTOML = readTOMLOrEmpty(ASSETS_TOML_PATH)
  if (assetsTOML) {
    for (const lbl of extractLabelsFromTOML(assetsTOML)) {
      labels.add(lbl)
    }
  }
  const outcomesTOML = readTOMLOrEmpty(OUTCOMES_TOML_PATH)
  if (outcomesTOML) {
    for (const lbl of extractLabelsFromTOML(outcomesTOML)) {
      labels.add(lbl)
    }
  }

  console.log(`lint-no-hardcoded-fields: ${labels.size} labels FR+EN à vérifier.`)

  let totalViolations = 0
  let filesScanned = 0
  for (const filePath of walk(WEB_SRC)) {
    filesScanned += 1
    if (isWhitelisted(filePath)) continue
    const violations = findViolations(filePath, labels)
    if (violations.length === 0) continue
    const rel = relative(REPO_ROOT, filePath).replace(/\\/g, '/')
    console.log(`\n${rel}:`)
    for (const v of violations) {
      console.log(`  ${rel}:${v.line} — littéral "${v.label}" hardcodé`)
      console.log(`    ${v.snippet}`)
    }
    totalViolations += violations.length
  }

  console.log(`\nfichiers scannés : ${filesScanned}`)
  if (totalViolations > 0) {
    console.error(`\n${totalViolations} violation(s) détectée(s).`)
    console.error(`utilisez useFieldLabel('<key>') au lieu d'un littéral hardcodé.`)
    console.error(`voir AUDIT_I18N_REACT_2026-04-25.md §5 pour la stratégie de migration.`)
    process.exit(1)
  }
  console.log(`\naucune violation : ✓`)
}

main()
