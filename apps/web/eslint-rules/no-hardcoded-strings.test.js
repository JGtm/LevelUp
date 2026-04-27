/**
 * Tests de la règle ESLint @levelup/no-hardcoded-strings.
 * Utilise RuleTester d'ESLint avec le parser typescript-eslint.
 *
 * RuleTester appelle describe()/it() en interne, donc tester.run() doit être
 * invoqué au top-level (Vitest interdit les suites imbriquées dans un it()).
 */
import { RuleTester } from 'eslint'
import tseslint from 'typescript-eslint'

import rule from './no-hardcoded-strings.js'

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

tester.run('@levelup/no-hardcoded-strings', rule, {
      valid: [
        // 1. JSX texte court (<3 mots et <15 chars) -> tolere
        {
          code: `function C() { return <div>OK</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        // 2. JSX texte uniquement chiffres / symboles -> tolere
        {
          code: `function C() { return <div>{42}</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        // 3. JSX texte issu d'un appel t() -> tolere (pas de littéral)
        {
          code: `function C() { return <div>{t('common.outcome.win')}</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        // 4. Fichier dans whitelist (test) -> tolere
        {
          code: `function C() { return <div>Une longue phrase avec plusieurs mots</div> }`,
          filename: '/web/src/features/foo/Bar.test.tsx',
        },
        // 5. Manifest i18n -> tolere
        {
          code: `function C() { return <div>Texte ignoré dans manifest</div> }`,
          filename: '/web/src/lib/i18n/manifests/common.toml.tsx',
        },
        // 6. Attribut technique long -> tolere (className, data-testid, etc.)
        {
          code: `function C() { return <div className="flex flex-col gap-4 items-center" /> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        {
          code: `function C() { return <div data-testid="mon-test-id-tres-explicite" /> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        // 7. Attribut href / src long -> tolere (URL technique)
        {
          code: `function C() { return <a href="https://example.com/long/path/here">x</a> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
        // 8. JSX text non-trivial mais 2 mots et <15 chars -> tolere (heuristique)
        {
          code: `function C() { return <div>Bonjour</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
        },
      ],
      invalid: [
        // I1. JSX texte ≥ 3 mots -> erreur
        {
          code: `function C() { return <div>Aucune donnée à afficher</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedJSXText' }],
        },
        // I2. JSX texte ≥ 15 chars (2 mots) -> erreur
        {
          code: `function C() { return <div>Configuration manuelle</div> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedJSXText' }],
        },
        // I3. Attribut title long -> erreur
        {
          code: `function C() { return <button title="Cliquer pour valider"></button> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedAttr' }],
        },
        // I4. Attribut aria-label long -> erreur
        {
          code: `function C() { return <button aria-label="Fermer la fenêtre principale"></button> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedAttr' }],
        },
        // I5. Attribut placeholder long -> erreur
        {
          code: `function C() { return <input placeholder="Entrer votre nom complet" /> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedAttr' }],
        },
        // I6. JSX texte avec mots français accentues -> doit etre detecte
        {
          code: `function C() { return <span>Vérification en cours…</span> }`,
          filename: '/web/src/features/foo/Bar.tsx',
          errors: [{ messageId: 'hardcodedJSXText' }],
        },
      ],
})
