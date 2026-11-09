/**
 * Lecture, validation et rendu des manifests i18n (`apps/web/src/lib/i18n/manifests/*.toml`),
 * pour N langues (ADR 0040).
 *
 * Module sans effet de bord : il PRODUIT le texte des modules generes et valide les feuilles ;
 * `build_i18n_manifests.mjs` (ecriture) et les tests (`lib/i18n/manifestGenerator.test.ts`)
 * l'appellent tous deux, le texte attendu d'un module genere n'existe qu'ici.
 *
 * Contrat des feuilles (D-4 : une langue est complete ou refusee) : une feuille porte EXACTEMENT
 * les langues de `config/locales.toml`, chacune en chaine non vide. Une langue manquante, vide
 * ou non listee est une erreur qui nomme le fichier, le chemin de la cle et la langue.
 *
 * Chaque texte doit en outre COMPILER en message ICU (`intl-messageformat`, le moteur du runtime
 * `lib/i18n/format.ts`) : le runtime avale l'echec et affiche le texte brut, sans interpolation ;
 * une accolade ou une balise litterale doit donc etre echappee (`'{'`, `'<'`) pour que la regle
 * « le manifest est la source » tienne. Erreur : fichier, cle, langue, cause du parseur.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { basename, join } from 'node:path'

import TOML from '@iarna/toml'
import { IntlMessageFormat } from 'intl-messageformat'

/** Une table du manifest est une feuille des qu'elle porte une valeur texte ; sinon c'est une section. */
function isLeafTable(v) {
  return Object.values(v).some((x) => typeof x === 'string')
}

/**
 * Valide une feuille et la retourne sous forme `{ <code>: texte }` dans l'ordre de `codes`.
 * `where` = `<fichier> : cle "<chemin>"` : prefixe de tout message.
 */
function checkLeaf(v, codes, where) {
  const extras = Object.keys(v).filter((k) => !codes.includes(k))
  if (extras.length > 0) {
    throw new Error(`${where} : langue non listee dans config/locales.toml : ${extras.join(', ')}`)
  }
  const leaf = {}
  for (const code of codes) {
    if (typeof v[code] !== 'string') {
      throw new Error(`${where} : langue manquante "${code}"`)
    }
    if (v[code].trim() === '') {
      throw new Error(`${where} : langue "${code}" vide`)
    }
    try {
      new IntlMessageFormat(v[code], code)
    } catch (e) {
      const cause = String(e?.message ?? e).split(/\r?\n/)[0]
      throw new Error(`${where} : langue "${code}" : texte non compilable en message ICU (${cause})`)
    }
    leaf[code] = v[code]
  }
  return leaf
}

/**
 * Aplatit un manifest TOML imbrique en map<chemin pointe, { <code>: texte }>.
 * Ex : {common: {period: {all: {fr, en}}}} -> {"common.period.all": {fr, en}}.
 * `file` ne sert qu'aux messages d'erreur. `options.fabricated` : langues que l'appelant fabrique
 * lui-meme (mode pseudo-langue) ; leur eventuelle valeur dans le TOML est ignoree, pas validee.
 */
export function flattenManifest(obj, codes, file, options = {}) {
  const fabricated = options.fabricated ?? []
  const out = {}
  const walk = (node, prefix) => {
    for (const [k, v] of Object.entries(node)) {
      const path = prefix ? `${prefix}.${k}` : k
      const where = `${file} : cle "${path}"`
      if (!v || typeof v !== 'object' || Array.isArray(v)) {
        throw new Error(`${where} : doit etre une table [section], recu ${Array.isArray(v) ? 'tableau' : typeof v}`)
      }
      if (isLeafTable(v)) {
        const own = Object.fromEntries(Object.entries(v).filter(([code]) => !fabricated.includes(code)))
        out[path] = checkLeaf(own, codes, where)
      } else {
        walk(v, path)
      }
    }
  }
  walk(obj, '')
  return out
}

function toCamelCase(s) {
  return s.replace(/_([a-z0-9])/g, (_, c) => c.toUpperCase())
}

function toPascalCase(s) {
  const camel = toCamelCase(s)
  return camel.charAt(0).toUpperCase() + camel.slice(1)
}

/** Texte attendu de `lib/i18n/generated/<domaine>.ts`. Une cle = un objet `{ <code>: texte }` par langue. */
export function renderManifestModule(domainName, flat, codes) {
  const constName = `${toCamelCase(domainName)}Manifest`
  const typeName = `${toPascalCase(domainName)}ManifestKey`
  const lines = [
    '// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.',
    `// Source : apps/web/src/lib/i18n/manifests/${domainName}.toml`,
    '',
    `export const ${constName} = {`,
  ]
  for (const key of Object.keys(flat).sort()) {
    const cells = codes.map((code) => `${code}: ${JSON.stringify(flat[key][code])}`).join(', ')
    lines.push(`  ${JSON.stringify(key)}: { ${cells} },`)
  }
  lines.push('} as const', '', `export type ${typeName} = keyof typeof ${constName}`, '')
  return lines.join('\n')
}

/**
 * Encadre un message ICU de `⟦` et `⟧`. Les delimiteurs sont hors de la syntaxe ICU (ni accolade,
 * ni apostrophe, ni chevron, ni `#`) : arguments `{n}`, `plural` / `select`, balises et `#` du
 * message source restent intacts, et le message encadre reste un message ICU valide.
 */
export function pseudoMessage(message) {
  return `⟦${message}⟧`
}

/** Ajoute (ou remplace) la langue `code` de chaque feuille, fabriquee par `pseudoMessage` depuis `source`. */
export function withPseudoLocale(flat, code, source) {
  const out = {}
  for (const [key, leaf] of Object.entries(flat)) {
    out[key] = { ...leaf, [code]: pseudoMessage(leaf[source]) }
  }
  return out
}

/** Domaines (`<domaine>.toml`) d'un dossier de manifests, tries. */
export function listManifestDomains(dir) {
  if (!existsSync(dir)) throw new Error(`dossier de manifests introuvable : ${dir}`)
  return readdirSync(dir)
    .filter((f) => f.endsWith('.toml'))
    .sort()
    .map((f) => basename(f, '.toml'))
}

/** Lit et valide `<dir>/<domaine>.toml` : retourne la map aplatie `{ chemin: { <code>: texte } }`. */
export function loadManifest(dir, domain, codes, options = {}) {
  const file = `${domain}.toml`
  const parsed = TOML.parse(readFileSync(join(dir, file), 'utf8'))
  return flattenManifest(parsed, codes, file, options)
}
