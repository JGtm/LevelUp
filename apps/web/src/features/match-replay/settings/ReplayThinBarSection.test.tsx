/**
 * Tests — l'option « Barre de lecture fine » du panneau de la vue : absente hors Plein cadre, un interrupteur nommé
 * dans la langue de l'interface au Plein cadre, qui porte l'état et le geste.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { ThinBarSection } from './ReplayThinBarSection'

describe('ThinBarSection', () => {
  it('ne rend rien là où la barre ne peut pas se poser sur le plan', () => {
    const { container } = render(<ThinBarSection locale="fr" thinBar={{ available: false, show: true, onToggle: vi.fn() }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it.each(['fr', 'en'] as ReplayLocale[])("%s : un interrupteur nommé, qui porte l'état et le geste", (locale) => {
    const onToggle = vi.fn()
    render(<ThinBarSection locale={locale} thinBar={{ available: true, show: true, onToggle }} />)
    const toggle = screen.getByRole('switch', { name: REPLAY_TEXT[locale].frames.thinBar })
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(toggle)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it("éteint : l'interrupteur le dit", () => {
    render(<ThinBarSection locale="fr" thinBar={{ available: true, show: false, onToggle: vi.fn() }} />)
    expect(screen.getByRole('switch', { name: REPLAY_TEXT.fr.frames.thinBar })).toHaveAttribute('aria-checked', 'false')
  })

  it("les deux langues ont un libellé et une phrase d'aide distincts", () => {
    expect(REPLAY_TEXT.fr.frames.thinBar).not.toBe(REPLAY_TEXT.en.frames.thinBar)
    expect(REPLAY_TEXT.fr.frames.thinBarHint).not.toBe(REPLAY_TEXT.en.frames.thinBarHint)
  })
})
