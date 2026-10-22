import { describe, expect, it } from 'vitest'
import { REPLAY_TEXT } from '../i18n/i18n'
import { presenceWording } from './presenceWording'

describe('presenceWording', () => {
  it.each(['fr', 'en'] as const)(
    'le libellé est le même pour l API et pour le repli film (%s) ; seule l infobulle change',
    (locale) => {
      const t = REPLAY_TEXT[locale]
      for (const kind of ['joined', 'left'] as const) {
        const api = presenceWording({ kind, source: 'api' }, t)
        const film = presenceWording({ kind, source: 'film' }, t)
        expect(film.label).toBe(api.label)
        expect(film.hint).not.toBe(api.hint)
      }
    },
  )

  it('FR : « a rejoint la partie » / « a quitté la partie », réserve gardée dans l infobulle du repli', () => {
    const t = REPLAY_TEXT.fr
    expect(presenceWording({ kind: 'joined', source: 'film' }, t).label).toBe('a rejoint la partie')
    const left = presenceWording({ kind: 'left', source: 'film' }, t)
    expect(left.label).toBe('a quitté la partie')
    expect(left.hint).toContain('le film ne les distingue pas')
  })

  it('EN : « joined the game » / « left the game », réserve gardée dans l infobulle du repli', () => {
    const t = REPLAY_TEXT.en
    expect(presenceWording({ kind: 'joined', source: 'film' }, t).label).toBe('joined the game')
    const left = presenceWording({ kind: 'left', source: 'film' }, t)
    expect(left.label).toBe('left the game')
    expect(left.hint).toContain('does not tell them apart')
  })
})
