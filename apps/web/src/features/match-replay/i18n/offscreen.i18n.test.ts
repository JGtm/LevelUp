import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from './i18n'

/** La flèche hors cadre reçoit une distance en UNITÉS DU JEU (positions du rejeu) : elle l'écrit en mètres. */
describe('étiquette de la flèche hors cadre : unités du jeu → mètres', () => {
  it('10 unités du jeu = 30 m (30,48 arrondi), en français et en anglais', () => {
    expect(REPLAY_TEXT.fr.offscreenMarkerFmt('Alpha', 10)).toBe('Alpha · 30 m')
    expect(REPLAY_TEXT.en.offscreenMarkerFmt('Alpha', 10)).toBe('Alpha · 30 m')
    expect(REPLAY_TEXT.fr.offscreenGroupMarkerFmt(3, 10)).toBe('3 joueurs · 30 m')
    expect(REPLAY_TEXT.en.offscreenGroupMarkerFmt(3, 10)).toBe('3 players · 30 m')
  })
})
