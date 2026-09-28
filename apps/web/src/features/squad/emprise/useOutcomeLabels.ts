/**
 * Libellés des issues de match pour l'onglet Emprise, lus dans les manifests du titre
 * (`outcomes.toml` via `useOutcomeLabel`) : aucun libellé d'issue en dur côté front.
 */
import { useMemo } from 'react'

import type { OutcomeValue } from '@/components/charts/outcomeSequence'
import { useOutcomeLabel } from '@/lib/i18n/fieldMappings'

export function useOutcomeLabels(): Record<OutcomeValue, string> {
  const win = useOutcomeLabel('win')
  const loss = useOutcomeLabel('loss')
  const tie = useOutcomeLabel('tie')
  const dnf = useOutcomeLabel('dnf')
  return useMemo(() => ({ win, loss, tie, dnf }), [win, loss, tie, dnf])
}

/** Forme en milieu de phrase : « Victoire » → « victoire » ; un sigle (« DNF ») reste tel quel. */
export function inSentence(label: string): string {
  const second = label.charAt(1)
  return second && second === second.toLowerCase() ? label.charAt(0).toLowerCase() + label.slice(1) : label
}
