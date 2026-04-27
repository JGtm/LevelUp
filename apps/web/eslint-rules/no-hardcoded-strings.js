/**
 * Règle ESLint custom — interdit les littéraux JSX texte qui ressemblent à
 * du contenu UI utilisateur en dehors des manifests i18n et des appels t() /
 * formatMessage().
 *
 * Conformément au PLAN_META_FOUNDATIONS_GO § 3.4.3 :
 *   - JSXText avec ≥ 3 mots OU ≥ 15 caractères contenant des lettres → erreur
 *   - JSX attribute strings (title, aria-label, placeholder, alt) ≥ 15 chars
 *     contenant des lettres → erreur
 *
 * Whitelist (heuristique simple, pas de scope analysis complète) :
 *   - Tests et fichiers de fixtures
 *   - Manifests i18n et runtime format.ts
 *   - Composants i18n locaux *.i18n.ts ou features/X/i18n.ts
 *   - Attributs techniques : className, data-*, key, id, role, name, type,
 *     for, htmlFor, form, list, target, rel, src, href, alt si chemin (URL)
 *   - Strings importées via templates literals avec variables (concat)
 *
 * Activée en `warn` dans la config flat eslint.config.js. Sera passée en
 * `error` après migration progressive des composants vers le manifest i18n
 * (cf. méta-plan Phase 2 done definition).
 */

const TECHNICAL_JSX_ATTRS = new Set([
  'className',
  'class',
  'key',
  'id',
  'role',
  'name',
  'type',
  'for',
  'htmlFor',
  'form',
  'list',
  'target',
  'rel',
  'src',
  'href',
  'data-testid',
  'data-test',
  'as',
  'tagName',
  'tag',
  'variant',
  'size',
  'width',
  'height',
  'fill',
  'stroke',
  'transform',
])

// Attributs candidats à porter du contenu utilisateur si non-techniques.
const TEXT_BEARING_JSX_ATTRS = new Set([
  'title',
  'aria-label',
  'aria-description',
  'aria-placeholder',
  'placeholder',
  'alt',
  'label',
])

const MIN_LETTER_COUNT = 4
const MIN_WORD_COUNT_TO_FLAG = 3
const MIN_LENGTH_TO_FLAG = 15

/**
 * Whitelist patterns sur le path complet du fichier (POSIX). Tout fichier
 * matchant un pattern est ignoré par la règle.
 */
const FILE_WHITELIST = [
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
  /\/test\//,
  /\/__tests__\//,
  /\/i18n\.ts$/,
  /\.i18n\.ts$/,
  /\/lib\/i18n\/manifests\//,
  /\/lib\/i18n\/generated\//,
  /\/lib\/i18n\/format\.tsx?$/,
  /\/lib\/i18n\/fieldMappings\.ts$/,
  /\/lib\/api\/types\.ts$/,
  /\/lib\/api\/generated\.ts$/,
  /\/test\/handlers\.ts$/,
]

function isFileWhitelisted(filename) {
  if (!filename) return false
  const normalized = filename.replace(/\\/g, '/')
  return FILE_WHITELIST.some((re) => re.test(normalized))
}

/**
 * Heuristique : la string ressemble-t-elle à du contenu utilisateur ?
 * Retourne `true` si la chaîne mérite d'être flaggée.
 */
function looksLikeUserContent(raw) {
  const text = raw.trim()
  if (text.length === 0) return false
  // Pas que des chiffres / symboles
  const letterMatches = text.match(/[\p{L}]/gu)
  if (!letterMatches || letterMatches.length < MIN_LETTER_COUNT) return false
  // 3+ mots → contenu probable
  const words = text.split(/\s+/).filter(Boolean)
  if (words.length >= MIN_WORD_COUNT_TO_FLAG) return true
  // Sinon ≥ 15 caractères → contenu probable
  return text.length >= MIN_LENGTH_TO_FLAG
}

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'suggestion',
    docs: {
      description:
        'Interdit les littéraux JSX texte qui ressemblent à du contenu UI utilisateur en dehors des manifests i18n.',
    },
    schema: [],
    messages: {
      hardcodedJSXText:
        'Littéral JSX texte "{{text}}" hardcodé. Utiliser formatMessage() ou ajouter au manifest i18n.',
      hardcodedAttr:
        'Attribut "{{attr}}" hardcodé : "{{text}}". Utiliser formatMessage() ou ajouter au manifest i18n.',
    },
  },
  create(context) {
    const filename =
      typeof context.filename === 'string'
        ? context.filename
        : typeof context.getFilename === 'function'
          ? context.getFilename()
          : ''
    if (isFileWhitelisted(filename)) {
      return {}
    }

    return {
      JSXText(node) {
        if (looksLikeUserContent(node.value)) {
          context.report({
            node,
            messageId: 'hardcodedJSXText',
            data: { text: node.value.trim().slice(0, 60) },
          })
        }
      },
      JSXAttribute(node) {
        const attrName =
          node.name && node.name.type === 'JSXIdentifier' ? node.name.name : null
        if (!attrName) return
        if (TECHNICAL_JSX_ATTRS.has(attrName)) return
        if (!TEXT_BEARING_JSX_ATTRS.has(attrName)) return

        const value = node.value
        if (!value || value.type !== 'Literal') return
        if (typeof value.value !== 'string') return
        if (looksLikeUserContent(value.value)) {
          context.report({
            node: value,
            messageId: 'hardcodedAttr',
            data: { attr: attrName, text: value.value.slice(0, 60) },
          })
        }
      },
    }
  },
}

export default rule
