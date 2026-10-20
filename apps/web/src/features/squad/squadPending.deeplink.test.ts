/**
 * Tests decideCompositionReanchor — premier ancrage d'un LIEN PROFOND de l'accueil (lot
 * perf L9-web, 2026-09-23, revue adversariale C). Les autres règles de la décision sont
 * testées dans `squadPending.test.ts` ; le câblage (le lien ne vaut que pour le premier
 * ancrage de SA composition) dans `useSquadPageRequests.test.tsx` et
 * `SquadLayout.deeplink.test.tsx`.
 *
 * Le carrousel de l'accueil ouvre l'Escouade sur une session choisie
 * (`?session=…&teammates=…`), souvent pas la dernière de la composition. La dernière
 * n'ayant jamais été ancrée (`lastKnownLatestSessionId` nul ou celui d'une autre
 * composition), la règle « nouvelle session jamais ancrée → snap » écrasait le choix.
 */
import { describe, expect, it } from 'vitest'
import { decideCompositionReanchor } from './squadPending'

describe('decideCompositionReanchor — premier ancrage d un lien profond', () => {
  const lien = {
    hasTeammates: true,
    followLatest: false,
    latestCompositionSession: 'S2 (3)',
    compositionSessionLabels: ['S2 (3)', 'S1 (2)'],
    pinnedByDeepLink: true,
  }

  it.each([
    ['aucun ancrage connu', ''],
    ['ancrage d une AUTRE composition', 'X9 (5)'],
  ])('session du lien dans la composition (%s) : none, pas de snap', (_nom, dernierAncrage) => {
    expect(
      decideCompositionReanchor({ ...lien, pickedSessions: ['S1 (2)'], lastAnchoredLatestSession: dernierAncrage }),
    ).toEqual({ kind: 'none' })
  })

  it('suffixe « (N) » du lien périmé : même session, none', () => {
    expect(
      decideCompositionReanchor({ ...lien, pickedSessions: ['S1 (1)'], lastAnchoredLatestSession: '' }),
    ).toEqual({ kind: 'none' })
  })

  it('session du lien INCONNUE de la composition : les règles ordinaires reprennent la main (snap)', () => {
    expect(
      decideCompositionReanchor({ ...lien, pickedSessions: ['S0 (1)'], lastAnchoredLatestSession: '' }),
    ).toEqual({ kind: 'snap', label: 'S2 (3)' })
  })

  it('composition jamais jouée ensemble : clear (état vide), comme sans lien', () => {
    expect(
      decideCompositionReanchor({
        ...lien,
        latestCompositionSession: '',
        compositionSessionLabels: [],
        pickedSessions: ['S1 (2)'],
        lastAnchoredLatestSession: '',
      }),
    ).toEqual({ kind: 'clear' })
  })

  it('témoin : la même entrée SANS le lien snappe sur la dernière (règle inchangée)', () => {
    expect(
      decideCompositionReanchor({
        ...lien,
        pinnedByDeepLink: false,
        pickedSessions: ['S1 (2)'],
        lastAnchoredLatestSession: '',
      }),
    ).toEqual({ kind: 'snap', label: 'S2 (3)' })
  })
})
