import { afterEach, describe, expect, it, vi } from 'vitest'
import { openInNewTab } from './openInNewTab'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('openInNewTab', () => {
  it('clique un lien target=_blank noopener vers l’URL, puis le retire de la page', () => {
    const clicked: HTMLAnchorElement[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      clicked.push(this)
      // Au moment du clic le lien est dans le document : un clic sur un lien détaché n'est pas suivi.
      expect(document.body.contains(this)).toBe(true)
    })
    openInNewTab('https://github.com/JGtm/LevelUp/issues/new?title=a')
    expect(clicked).toHaveLength(1)
    expect(clicked[0].href).toBe('https://github.com/JGtm/LevelUp/issues/new?title=a')
    expect(clicked[0].target).toBe('_blank')
    expect(clicked[0].rel).toBe('noopener noreferrer')
    expect(document.body.contains(clicked[0])).toBe(false)
  })
})
