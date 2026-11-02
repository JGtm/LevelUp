import { describe, expect, it } from 'vitest'
import { thirdPartyNoticesUrl } from './appLinks'

describe('thirdPartyNoticesUrl', () => {
  it('suit le tag de la version installée', () => {
    expect(thirdPartyNoticesUrl('8.1.0')).toBe(
      'https://github.com/JGtm/LevelUp/blob/v8.1.0/apps/desktop/THIRD_PARTY_NOTICES.md',
    )
  })

  it.each(['dev', '', null, undefined, '8.1', 'v8.1.0', '8.1.0-rc1'])(
    'retombe sur la branche principale pour %s',
    (version) => {
      expect(thirdPartyNoticesUrl(version)).toBe(
        'https://github.com/JGtm/LevelUp/blob/main/apps/desktop/THIRD_PARTY_NOTICES.md',
      )
    },
  )
})
