/**
 * Règle ESLint custom — interdit le littéral de slug de titre `halo_infinite`
 * codé en dur dans `src/features/**` et `src/components/**` (MT-12 / PMT-12).
 *
 * Le slug du titre courant doit venir du store (`currentTitleSlug`) ou du
 * contexte API title-agnostique — jamais d'un littéral, sous peine de coupler la
 * surface produit à Halo et de casser un 2e titre.
 *
 * Portée : features/ + components/ uniquement (lib/staticAssets, stores/default,
 * tests et i18n restent tolérés transitoirement). Activée en `warn` (migration
 * progressive), passera en `error` quand les surfaces seront nettoyées.
 */

const TITLE_SLUG_LITERAL = 'halo_infinite'

// Fichiers tolérés DANS features/components (tests, i18n, fixtures).
const FILE_WHITELIST = [
  /\.test\.tsx?$/,
  /\.spec\.tsx?$/,
  /\/test\//,
  /\/__tests__\//,
  /\/i18n\.ts$/,
  /\.i18n\.ts$/,
]

function isOutOfScopeOrWhitelisted(filename) {
  if (!filename) return true
  const n = filename.replace(/\\/g, '/')
  // Cible exclusive : features/ et components/.
  if (!/\/src\/(features|components)\//.test(n)) return true
  return FILE_WHITELIST.some((re) => re.test(n))
}

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'suggestion',
    docs: {
      description:
        "Interdit le littéral 'halo_infinite' hardcodé dans features/ et components/ (MT-12).",
    },
    schema: [],
    messages: {
      titleSlugLiteral:
        "Slug de titre 'halo_infinite' hardcodé. Utiliser currentTitleSlug (store) ou le contexte API title-agnostique.",
    },
  },
  create(context) {
    const filename =
      typeof context.filename === 'string'
        ? context.filename
        : typeof context.getFilename === 'function'
          ? context.getFilename()
          : ''
    if (isOutOfScopeOrWhitelisted(filename)) {
      return {}
    }

    return {
      Literal(node) {
        if (typeof node.value === 'string' && node.value === TITLE_SLUG_LITERAL) {
          context.report({ node, messageId: 'titleSlugLiteral' })
        }
      },
      TemplateElement(node) {
        if (node.value && node.value.cooked === TITLE_SLUG_LITERAL) {
          context.report({ node, messageId: 'titleSlugLiteral' })
        }
      },
    }
  },
}

export default rule
