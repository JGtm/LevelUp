import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

import noHardcodedStrings from './eslint-rules/no-hardcoded-strings.js'
import noTitleSlugLiteral from './eslint-rules/no-title-slug-literal.js'

export default defineConfig([
  // Tests E2E Playwright : moins stricts (mocks any, ts-ignore acceptes
  // pour test-helpers, prefer-const variable selon contexte de test).
  globalIgnores(['dist', 'e2e/**', 'coverage/**']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      // Plugin custom pour la règle anti-hardcoded-strings (cf.
      // PLAN_META_FOUNDATIONS_GO § 3.4.3).
      // Activee en `warn` Phase 0 ; passera en `error` Phase 2 quand
      // tous les composants seront migres vers le manifest i18n.
      '@levelup': {
        rules: {
          'no-hardcoded-strings': noHardcodedStrings,
          'no-title-slug-literal': noTitleSlugLiteral,
        },
      },
    },
    rules: {
      '@levelup/no-hardcoded-strings': 'warn',
      '@levelup/no-title-slug-literal': 'warn',
      // react-refresh/only-export-components : downgrade en warn — cette
      // regle est cosmetique (HMR fast refresh) et beaucoup de fichiers
      // composants exposent legitimement des const/types/helpers a cote.
      // Refactor possible plus tard (split en *.types.ts / *.utils.ts).
      'react-refresh/only-export-components': 'warn',
      // Regles React Compiler (eslint-plugin-react-hooks v7) — downgrade en
      // warn pour cette phase. Necessitent du refactor cas-par-cas qui sort
      // du scope sprint stabilisation CI.
      //  - preserve-manual-memoization : useMemo defensifs avec deps elargies
      //  - refs : pattern legitime de ref-as-key dans certains hooks
      //  - set-state-in-effect : setState dans useEffect (parfois necessaire
      //    pour sync prop -> state quand source externe pilote la valeur)
      // A re-activer en error sprint cleanup React Compiler dedie.
      'react-hooks/preserve-manual-memoization': 'warn',
      'react-hooks/refs': 'warn',
      'react-hooks/set-state-in-effect': 'warn',
      // use-memo : JSON.stringify dans les deps est un pattern intentionnel
      // pour deep-compare les options ECharts (objets complexes non-stables).
      // A migrer vers useMemo(stabilize) + ref comparaison lors d'un sprint dédié.
      'react-hooks/use-memo': 'warn',
    },
  },
  {
    files: ['src/routes/**/*.{ts,tsx}', 'src/test/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
])
