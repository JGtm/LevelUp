/**
 * outcome.ts — mapping canonique du code outcome backend vers la valeur d'outcome
 * de la frise (OutcomeSequenceTape) / clés `outcomes.toml`.
 *
 * Contrat ADR 0006 : 2=win, 3=loss, 1=tie, 4=dnf. Clé 'tie' (frise + outcomes.toml).
 * NE PAS confondre avec la variante 'draw' des ÉCHELLES d'accessibilité/couleur
 * (`outcomeKey` de `lib/outcome-color.ts`) — jeux de clés distincts (tie vs draw).
 *
 * DÉFAUT EXPLICITE : un code HORS contrat (null / undefined / 0 / négatif / > 4)
 * mappe vers `null`. C'est le défaut le PLUS SÛR : ne jamais fabriquer un outcome
 * pour une entrée inconnue. Un consommateur qui COMPTE les issues (ex.
 * SessionOutcomeDonut : `if (!key) continue`) doit pouvoir EXCLURE l'inconnu — un
 * défaut 'tie' ou 'dnf' fausserait silencieusement ses statistiques.
 *
 * Source UNIQUE (garde-rail `outcome.guard.test.ts`, CLAUDE.md n°6) — ne jamais
 * redéfinir un mapping code→outcome localement.
 */
import type { OutcomeValue } from '@/components/charts/OutcomeSequenceTape'

export type { OutcomeValue }

/**
 * Mappe un code outcome backend vers sa valeur canonique, ou `null` si le code est
 * hors contrat ADR 0006 (voir docstring du module pour le choix du défaut).
 */
export function outcomeCodeToValue(code: number | null | undefined): OutcomeValue | null {
  switch (code) {
    case 2:
      return 'win'
    case 3:
      return 'loss'
    case 1:
      return 'tie'
    case 4:
      return 'dnf'
    default:
      return null
  }
}

/**
 * Variante pour les frises séquentielles (OutcomeSequenceTape) qui exigent une
 * valeur NON-null par match. Applique le défaut de frise 'dnf' — bucket neutre
 * « non classé » n'affirmant ni victoire, ni défaite, ni égalité. En pratique
 * l'API ne sert que les codes 1..4 → ce défaut ne se déclenche pas sur données
 * réelles ; il n'existe que pour satisfaire le typage non-null de la frise.
 */
export function outcomeCodeToTapeValue(code: number | null | undefined): OutcomeValue {
  return outcomeCodeToValue(code) ?? 'dnf'
}
