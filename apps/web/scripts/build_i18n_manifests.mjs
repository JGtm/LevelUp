/**
 * Build step manifests i18n : parse les fichiers TOML de
 * apps/web/src/lib/i18n/manifests/ et genere des modules TS typés dans
 * apps/web/src/lib/i18n/generated/.
 *
 * Utilisation :
 *   node apps/web/scripts/build_i18n_manifests.mjs
 *   node apps/web/scripts/build_i18n_manifests.mjs --pseudo xx [--out <dossier>]
 *
 * Conformement au PLAN_META_FOUNDATIONS_GO § 3.4 : chaque manifest produit un
 * objet `as const` qui sert de source de verite pour les types TypeScript des
 * cles. Le runtime `lib/i18n/format.ts` consomme ces modules pour resoudre les
 * strings via intl-messageformat.
 *
 * Les langues viennent de config/locales.toml (ADR 0040, scripts/locales.mjs) : le script emet
 * generated/locales.ts (web) et apps/go-api/internal/i18n/locales_gen.go (Go), et exige que
 * TOUTE feuille de manifest porte TOUTES les langues listees (D-4 : une langue est complete ou
 * refusee ; la regle vit dans scripts/manifests.mjs). Erreur : fichier, cle, langue.
 *
 * Mode --pseudo <code> : fabrique la langue <code> depuis `en` (texte entre `⟦` et `⟧`,
 * placeholders ICU intacts) pour rendre visible toute chaine qui n'en passe pas par un manifest.
 * C'est l'outil de la preuve de l'etape 10 du plan, jamais une langue livree : il n'ecrit NI
 * config/locales.toml, NI les fichiers versionnes ; les modules vont dans --out (par defaut un
 * dossier temporaire, affiche), avec locales.ts et locales_gen.go rendus pour les memes langues ;
 * <code> s'ajoute aux langues listees s'il n'y figure pas.
 *
 * Ce script ne valide pas les placeholders ICU : la validation runtime est
 * faite par intl-messageformat lors du t(). Le script verifie seulement :
 *   - Toute feuille a toutes les langues listees, et rien d'autre (echec sinon)
 *   - Pas de cle vide
 */
import { writeFile, mkdir, mkdtemp } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import {
  LOCALES_GO_PATH,
  LOCALES_TS_PATH,
  loadLocalesConfig,
  renderLocalesGo,
  renderLocalesTS,
} from './locales.mjs'
import { listManifestDomains, loadManifest, renderManifestModule, withPseudoLocale } from './manifests.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const MANIFESTS_DIR = join(__dirname, '..', 'src', 'lib', 'i18n', 'manifests')
const OUTPUT_DIR = join(__dirname, '..', 'src', 'lib', 'i18n', 'generated')

/** Langue source de la pseudo-langue. */
const PSEUDO_SOURCE = 'en'

/** Entree de langue d'une pseudo-langue absente de config/locales.toml : memes etiquettes que la source. */
function pseudoEntry(code) {
  const src = loadLocalesConfig().locales.find((l) => l.code === PSEUDO_SOURCE)
  return { code, intl: src.intl, halo: [...src.halo], required: false }
}

/** Lit `--pseudo <code>` et `--out <dossier>` ; toute autre option est une erreur. */
function parseArgs(argv) {
  const opts = { pseudo: null, out: null }
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    if (arg === '--pseudo' || arg === '--out') {
      const value = argv[++i]
      if (!value || value.startsWith('--')) throw new Error(`${arg} attend une valeur`)
      opts[arg.slice(2)] = value
    } else {
      throw new Error(`option inconnue : ${arg}`)
    }
  }
  if (opts.out && !opts.pseudo) throw new Error("--out n'a de sens qu'avec --pseudo (le mode normal ecrit generated/)")
  if (opts.pseudo && !/^[a-z]{2,3}$/.test(opts.pseudo)) throw new Error('--pseudo attend un code de langue en minuscules (ex. xx)')
  return opts
}

/** Ecrit un module par manifest dans `outDir` pour les langues `codes` ; retourne le nombre de cles. */
async function writeManifests(outDir, codes, pseudo) {
  if (!existsSync(outDir)) await mkdir(outDir, { recursive: true })
  const domains = listManifestDomains(MANIFESTS_DIR)
  if (domains.length === 0) throw new Error('aucun manifest .toml')
  const real = codes.filter((c) => c !== pseudo)
  let totalKeys = 0
  for (const domain of domains) {
    let flat = loadManifest(MANIFESTS_DIR, domain, real, { fabricated: pseudo ? [pseudo] : [] })
    if (pseudo) flat = withPseudoLocale(flat, pseudo, PSEUDO_SOURCE)
    const outPath = join(outDir, `${domain}.ts`)
    await writeFile(outPath, renderManifestModule(domain, flat, codes), 'utf8')
    console.log(`  ${domain}.toml -> ${Object.keys(flat).length} cles -> ${outPath}`)
    totalKeys += Object.keys(flat).length
  }
  console.log(`OK: ${domains.length} manifests, ${totalKeys} cles totales, langues : ${codes.join(', ')}.`)
}

async function main() {
  const opts = parseArgs(process.argv.slice(2))
  const localesCfg = loadLocalesConfig()
  const listed = localesCfg.locales.map((l) => l.code)

  if (opts.pseudo) {
    if (opts.pseudo === PSEUDO_SOURCE) throw new Error(`--pseudo ne peut pas remplacer la langue source "${PSEUDO_SOURCE}"`)
    const codes = listed.includes(opts.pseudo) ? listed : [...listed, opts.pseudo]
    const outDir = opts.out ? resolve(opts.out) : await mkdtemp(join(tmpdir(), 'levelup-i18n-pseudo-'))
    const withPseudo = { ...localesCfg, locales: codes.map((c) => localesCfg.locales.find((l) => l.code === c) ?? pseudoEntry(c)) }
    console.log(`pseudo-langue "${opts.pseudo}" fabriquee depuis "${PSEUDO_SOURCE}" -> ${outDir} (config/locales.toml et fichiers versionnes intacts)`)
    await mkdir(outDir, { recursive: true })
    await writeFile(join(outDir, 'locales.ts'), renderLocalesTS(withPseudo), 'utf8')
    await writeFile(join(outDir, 'locales_gen.go'), renderLocalesGo(withPseudo), 'utf8')
    await writeManifests(outDir, codes, opts.pseudo)
    return
  }

  await mkdir(OUTPUT_DIR, { recursive: true })
  await writeFile(LOCALES_TS_PATH, renderLocalesTS(localesCfg), 'utf8')
  await writeFile(LOCALES_GO_PATH, renderLocalesGo(localesCfg), 'utf8')
  console.log(`  config/locales.toml -> ${localesCfg.locales.length} langues -> ${LOCALES_TS_PATH}, ${LOCALES_GO_PATH}`)
  await writeManifests(OUTPUT_DIR, listed, null)
}

main().catch((err) => {
  console.error('build_i18n_manifests failed:', err.message)
  process.exit(1)
})
