import { describe, expect, it } from 'vitest'
import { verificationLinkLabel } from './url'

describe('verificationLinkLabel', () => {
  it('affiche host/chemin sans protocole ni www', () => {
    expect(verificationLinkLabel('https://www.microsoft.com/link')).toBe('microsoft.com/link')
  })

  it('supprime les query params (jamais de code_challenge/client_id affichés)', () => {
    expect(
      verificationLinkLabel(
        'https://login.live.com/oauth20_authorize.srf?lw=1&code_challenge=abc&client_id=000000004C20A908&scope=x',
      ),
    ).toBe('login.live.com/oauth20_authorize.srf')
  })

  it('supprime le fragment', () => {
    expect(verificationLinkLabel('https://example.com/page#section')).toBe('example.com/page')
  })

  it('host seul quand le chemin est /', () => {
    expect(verificationLinkLabel('https://microsoft.com/')).toBe('microsoft.com')
  })

  it('fallback défensif sur une entrée non-URL (pas de throw)', () => {
    expect(verificationLinkLabel('microsoft.com/link?x=1')).toBe('microsoft.com/link')
  })
})
