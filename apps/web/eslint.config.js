import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

import noHardcodedStrings from './eslint-rules/no-hardcoded-strings.js'

export default defineConfig([
  globalIgnores(['dist']),
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
        },
      },
    },
    rules: {
      '@levelup/no-hardcoded-strings': 'warn',
    },
  },
  {
    files: ['src/routes/**/*.{ts,tsx}', 'src/test/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
])
