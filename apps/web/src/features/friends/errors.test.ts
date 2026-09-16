/**
 * Tests — traduction des codes d'erreur de la liste d'amis.
 *
 * L'API n'envoie qu'un code : si cette table ne le connaît pas, l'utilisateur
 * doit tout de même lire une phrase, jamais une chaîne vide.
 */
import { describe, it, expect } from 'vitest'
import { friendsErrorMessage } from './errors'

describe('friendsErrorMessage', () => {
  it('traduit un code connu en FR et en EN', () => {
    const err = { code: 'friends_forbidden', message: '' }
    expect(friendsErrorMessage(err, 'fr')).toMatch(/propriétaire/i)
    expect(friendsErrorMessage(err, 'en')).toMatch(/owner/i)
  })

  it('distingue les deux motifs de refus de la liste', () => {
    expect(friendsErrorMessage({ code: 'invalid_friends_too_many' }, 'fr')).toMatch(/50 au maximum/i)
    expect(friendsErrorMessage({ code: 'invalid_friends_gamertag_too_long' }, 'fr')).toMatch(/50 caractères/i)
  })

  it('code inconnu, code absent ou erreur nulle : message générique, jamais vide', () => {
    for (const err of [{ code: 'code_jamais_vu' }, {}, null, undefined, new Error('boom')]) {
      const fr = friendsErrorMessage(err, 'fr')
      expect(fr.length).toBeGreaterThan(0)
      expect(fr).toMatch(/liste d'amis/i)
    }
  })

  it('locale inconnue retombe sur le français', () => {
    expect(friendsErrorMessage({ code: 'player_not_found' }, 'de')).toMatch(/introuvable/i)
  })
})
