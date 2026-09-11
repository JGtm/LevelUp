/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail anti-régression — littéraux FR hors i18n (bilan fork ChaseWoodhams
 * 2026-09-11, point 6). La règle eslint `no-hardcoded-strings`
 * (eslint-rules/no-hardcoded-strings.js) a DEUX angles morts qu'elle documente
 * elle-même comme heuristique simple, sans analyse de portée :
 *
 *   1. Un littéral FR assigné à une VARIABLE avant d'être posé sur un attribut
 *      JSX texte (aria-label/title/placeholder) n'est jamais vu — le visiteur
 *      `JSXAttribute` n'inspecte que les `Literal` directement en valeur
 *      d'attribut (ex. ThemeToggle.tsx : `const label = isDark ? '...' : '...'`
 *      puis `aria-label={label}`).
 *   2. Un littéral COURT (< 3 mots ET < 15 caractères, ex. "Précédent",
 *      "Suivant") est sous le seuil `looksLikeUserContent` et passe toujours,
 *      qu'il soit direct ou via variable.
 *
 * Ce garde-rail grep le SOURCE des fichiers effectivement corrigés par le lot
 * C.3 et interdit la RÉ-INTRODUCTION des littéraux FR précis qui y ont été
 * remplacés par un appel i18n. Il ne remplace pas la règle eslint (qui reste
 * la première ligne de défense pour du code NEUF) — il ferme sa lacune connue
 * sur ces fichiers précis, remplis d'exemples réels des deux angles morts.
 *
 * Ne pas étendre mécaniquement cette liste à tout littéral FR trouvé ailleurs :
 * chaque entrée correspond à un remplacement réellement fait dans ce lot.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const SRC = join(process.cwd(), 'src')

interface FileRule {
  file: string
  forbidden: string[]
}

const RULES: FileRule[] = [
  {
    file: 'components/shell/ThemeToggle.tsx',
    forbidden: [`'Passer au thème clair'`, `'Passer au thème sombre'`, `>Thème<`],
  },
  {
    file: 'components/ui/carousel.tsx',
    forbidden: [`aria-label="Précédent"`, `aria-label="Suivant"`],
  },
  {
    file: 'features/palmares/BattlePassRewardCarousel.tsx',
    forbidden: [
      `freeLabel = 'gratuit'`,
      `prevAriaLabel = 'Paliers précédents'`,
      `nextAriaLabel = 'Paliers suivants'`,
      '`Voir le détail de ${card.title}`',
    ],
  },
  {
    file: 'features/setup/StepPlayer.tsx',
    forbidden: [
      `'Erreur lors de la création du profil.'`,
      `placeholder="MonGamertag"`,
      `'Création…'`,
      `'Confirmer et créer mon profil'`,
      `: 'Ajouter'`,
    ],
  },
  {
    file: 'features/feedback-drawer/FeedbackDrawer.tsx',
    forbidden: [`'_(sans titre)_'`],
  },
  {
    file: 'features/feedback-drawer/buildIssueUrl.ts',
    forbidden: [
      `'[Idée] '`,
      `'## Description'`,
      `'## Contexte'`,
      `'## Environnement client'`,
      `'## Filtres actifs'`,
      `'_(aucune description fournie)_'`,
      `'_(aucun filtre actif)_'`,
      `'_(aucune erreur console capturée)_'`,
      `'_(aucune requête échouée capturée)_'`,
      `'*Auto-généré par le drawer feedback — LevelUp web*'`,
    ],
  },
  {
    file: 'features/auth/XboxLoginPage.tsx',
    forbidden: [
      `'Identifiants incorrects.'`,
      `'En mode SSO Xbox, le login par mot de passe est réservé aux administrateurs. Utilisez la connexion Xbox.'`,
      `?? 'Erreur de connexion.'`,
    ],
  },
  {
    file: 'features/synthesis/SynthesisPage.tsx',
    forbidden: [
      `?? 'Tirs à la tête'`,
      `?? 'Tirs effectués'`,
      `?? 'Tirs au but'`,
      `?? 'Dégâts infligés'`,
      `?? 'Dégâts reçus'`,
    ],
  },
  {
    file: 'features/timeseries/TimeseriesPage.tsx',
    forbidden: [`?? 'Égalité'`, `?? 'Abandon'`],
  },
  {
    file: 'features/timeseries/TimeseriesPage.summary.tsx',
    forbidden: [`?? 'Égalité'`, `?? 'Durée de vie moyenne'`, `?? 'MMR équipe'`],
  },
  {
    file: 'features/timeseries/TimeseriesPage.distributions.tsx',
    forbidden: [
      `?? 'Précision'`,
      `?? 'Score personnel'`,
      `?? 'Score de performance'`,
      `?? 'MMR équipe'`,
      `?? 'MMR adverse'`,
    ],
  },
  {
    file: 'features/timeseries/TimeseriesPage.progression.tsx',
    forbidden: [`?? 'Tirs à la tête'`, `?? 'Score personnel'`],
  },
]

describe('garde-rail anti-régression — littéraux FR hors i18n (bilan fork 2026-09-11 pt 6)', () => {
  for (const { file, forbidden } of RULES) {
    it(`${file} : les littéraux corrigés ne sont pas réintroduits`, () => {
      const content = readFileSync(join(SRC, file), 'utf8')
      const offenders = forbidden.filter((literal) => content.includes(literal))
      expect(
        offenders,
        `Littéral(aux) FR en dur réintroduit(s) dans ${file} :\n${offenders.join('\n')}\n` +
          `Repasser par le manifest i18n de la feature (formatMessage + lib/i18n/generated/*).`,
      ).toEqual([])
    })
  }
})
