/**
 * Tests de la règle ESLint @levelup/no-title-slug-literal (MT-12 / PMT-12).
 */
import { RuleTester } from 'eslint'
import tseslint from 'typescript-eslint'

import rule from './no-title-slug-literal.js'

const tester = new RuleTester({
  languageOptions: {
    parser: tseslint.parser,
    parserOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      ecmaFeatures: { jsx: true },
    },
  },
})

tester.run('@levelup/no-title-slug-literal', rule, {
  valid: [
    // 1. Slug venu du store -> toléré (pas de littéral).
    {
      code: `function C(){ const s = useStore(x => x.currentTitleSlug); return s }`,
      filename: '/web/src/features/foo/Bar.tsx',
    },
    // 2. Littéral dans un fichier hors features/components -> hors portée.
    {
      code: `export const DEFAULT = 'halo_infinite'`,
      filename: '/web/src/lib/staticAssets.ts',
    },
    // 3. Littéral dans un store -> hors portée.
    {
      code: `const s = { currentTitleSlug: 'halo_infinite' }`,
      filename: '/web/src/stores/appShellStore.ts',
    },
    // 4. Littéral dans un test de features -> toléré (whitelist).
    {
      code: `const slug = 'halo_infinite'`,
      filename: '/web/src/features/foo/Bar.test.tsx',
    },
    // 5. Autre littéral string dans features -> toléré.
    {
      code: `const x = 'halo_mcc'`,
      filename: '/web/src/features/foo/Bar.tsx',
    },
  ],
  invalid: [
    // I1. Littéral halo_infinite dans une feature -> erreur.
    {
      code: `const slug = 'halo_infinite'`,
      filename: '/web/src/features/foo/Bar.tsx',
      errors: [{ messageId: 'titleSlugLiteral' }],
    },
    // I2. Littéral halo_infinite (exact) dans un component -> erreur.
    {
      code: `const slug = 'halo_infinite'`,
      filename: '/web/src/components/Foo.tsx',
      errors: [{ messageId: 'titleSlugLiteral' }],
    },
    // I3. Template string contenant halo_infinite -> erreur.
    {
      code: 'const url = `halo_infinite`',
      filename: '/web/src/features/foo/Bar.tsx',
      errors: [{ messageId: 'titleSlugLiteral' }],
    },
  ],
})
