import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

import noHardcodedStrings from './eslint-rules/no-hardcoded-strings.js'
import noTitleSlugLiteral from './eslint-rules/no-title-slug-literal.js'

export default defineConfig([
  globalIgnores(['dist', 'coverage/**']),
  {
    files: ['**/*.{ts,tsx}'],
    // Les specs E2E (e2e/**) ont leur propre bloc plus bas (pas de plugins
    // React/hooks/refresh — aucun composant ni hook n'y vit).
    ignores: ['e2e/**'],
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
      // Réglages effectifs dans le bloc `rules` ci-dessous : depuis I5
      // (2026-07-05) `no-hardcoded-strings` est en `error`.
      '@levelup': {
        rules: {
          'no-hardcoded-strings': noHardcodedStrings,
          'no-title-slug-literal': noTitleSlugLiteral,
        },
      },
    },
    rules: {
      // I5 (2026-07-05) : passé en `error` — bloque toute NOUVELLE string JSX
      // hardcodée (texte ≥3 mots/≥15 car + attributs title/aria/placeholder/alt).
      // Portée volontairement ciblée : n'attrape PAS les args de fonction ni les
      // libellés courts (chantier i18n manuel I1/I2/I4, hors gate — cf. plan LOT I).
      '@levelup/no-hardcoded-strings': 'error',
      // EXT-5 / MT-12 : périmètre features/+components/ nettoyé (slug courant lu
      // via useAppShellStore) → error pour bloquer toute régression de littéral.
      '@levelup/no-title-slug-literal': 'error',
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
  // ── Seuil de taille R5 (CLAUDE.md n°5) ──────────────────────────────────
  // Decision utilisateur 4 du 2026-09-05 (.ai/PLAN_V2_REJEU_FILM_2026-09-05.md) :
  // le seuil de 500 lignes se mesure en lignes de CODE, pas en lignes brutes.
  // `skipComments` + `skipBlankLines` sont donc les deux moities de cette
  // decision : un fichier bien documente n'est pas un god file, et 17 extractions
  // du canvas du rejeu ont ete payees sur un compte de lignes brutes que ses
  // 207 lignes de commentaires dominaient (annexe W1 de l'audit v7.5).
  // La regle est en `error` : le depassement se corrige par une extraction par
  // RESPONSABILITE, ou par un `/* eslint-disable max-lines -- <raison datee> */`
  // en tete du fichier (R5 admet l'exemption justifiee, jamais silencieuse).
  {
    files: ['src/**/*.{ts,tsx}'],
    rules: {
      'max-lines': ['error', { max: 500, skipComments: true, skipBlankLines: true }],
    },
  },
  // Les deux seules exemptions qui ne peuvent PAS vivre en tete de leur fichier.
  //  - Fichiers GENERES (`npm run generate-types`, `scripts/build_i18n_manifests.mjs`) :
  //    un `eslint-disable` ecrit dedans serait efface a la prochaine generation.
  //  - `ExplorerMatchesTable.tsx` : gele le 2026-09-06 pour le merge du lot C (v2 D.14),
  //    qui l'edite en parallele — l'exemption revient en tete du fichier apres integration,
  //    ou le fichier se decoupe, au choix du lot qui le touchera.
  {
    files: [
      'src/lib/api/generated.ts',
      'src/lib/i18n/generated/**/*.ts',
      'src/features/explorer/ExplorerMatchesTable.tsx',
    ],
    rules: {
      'max-lines': 'off',
    },
  },
  // ── Tests E2E Playwright ────────────────────────────────────────────────
  // Specs exécutées en Node + code navigateur passé à page.evaluate(). Même base
  // de rigueur que le repo (js.recommended + typescript-eslint recommended, donc
  // no-explicit-any / no-unused-vars en erreur) MAIS sans les plugins React /
  // react-hooks / react-refresh : aucun composant ni hook ne vit ici (les charts
  // sont vérifiés via le DOM rendu, jamais montés dans la spec).
  //
  // Les règles maison @levelup sont volontairement absentes :
  //   - no-hardcoded-strings ne cible que JSXText/JSXAttribute (aucun JSX dans les
  //     .ts specs) et whiteliste déjà *.spec.ts ; les libellés UI FR attendus, en
  //     dur dans les assertions de test, sont LÉGITIMES.
  //   - no-title-slug-literal ne s'applique qu'à src/features|components ; les
  //     slugs réels 'halo_infinite'/'halo_5' sont l'objet même des specs de routage
  //     (legacy-redirect) et du helper routes.ts.
  {
    files: ['e2e/**/*.ts'],
    extends: [js.configs.recommended, tseslint.configs.recommended],
    languageOptions: {
      ecmaVersion: 2023,
      // Node (process.env) + navigateur (globals disponibles dans page.evaluate).
      globals: { ...globals.node, ...globals.browser },
    },
  },
])
