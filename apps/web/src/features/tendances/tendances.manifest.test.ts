/**
 * Garde du manifeste des tendances : aucun texte français n'emploie l'anglicisme « matchmaking ».
 */
import { describe, expect, it } from 'vitest'

import { tendancesManifest } from '@/lib/i18n/generated/tendances'

describe('manifeste des tendances', () => {
  it('aucune valeur française ne contient « matchmaking »', () => {
    const fautifs = Object.entries(tendancesManifest)
      .filter(([, valeur]) => /matchmaking/i.test(valeur.fr))
      .map(([cle]) => cle)
    expect(fautifs).toEqual([])
  })
})
