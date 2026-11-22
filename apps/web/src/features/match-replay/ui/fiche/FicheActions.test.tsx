/**
 * Tests — LE SEGMENT « Suivre / Seul / Masquer » de la fiche (`FicheActions`) : un seul contour de trois colonnes
 * égales, collées ; l'action active est pleine ; mêmes libellés et mêmes bascules qu'avant (`aria-pressed`) ;
 * accessible au clavier ; la souris ne vole pas le focus.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from '../../i18n/i18n'
import type { PlayerPanelAction } from '../../model/playerPanelLogic'
import { FicheActions } from './FicheActions'

const AUCUNE: Record<PlayerPanelAction, boolean> = { follow: false, solo: false, hide: false }

function monter(actions = AUCUNE, locale: 'fr' | 'en' = 'fr', onToggle = vi.fn()) {
  const view = render(<FicheActions name="JGtm" actions={actions} onToggle={onToggle} locale={locale} />)
  return { ...view, onToggle, groupe: screen.getByRole('group', { name: 'JGtm' }) }
}

describe('la forme : un segment de trois colonnes égales, collées', () => {
  it('un seul contour arrondi en grille de trois colonnes, sans écart entre elles', () => {
    const { groupe } = monter()
    expect(groupe.className).toContain('grid-cols-3')
    expect(groupe.className).toContain('rounded-lg')
    expect(groupe.className).toContain('border')
    expect(groupe.className).toContain('overflow-hidden')
    expect(groupe.className).not.toMatch(/\bgap-/)
    expect(groupe.children).toHaveLength(3)
  })

  it('chaque colonne est un bouton de même hauteur ; un filet sépare les colonnes, jamais devant la première', () => {
    const { groupe } = monter()
    const colonnes = [...groupe.children] as HTMLElement[]
    expect(new Set(colonnes.map((c) => c.className.match(/\bh-\d+\b/)?.[0])).size).toBe(1)
    for (const colonne of colonnes) {
      expect(colonne.tagName).toBe('BUTTON')
      expect(colonne.className).toContain('[&:not(:first-child)]:border-l')
      // Aucun contour propre : seul le segment est bordé (pas de rayon, pas de bordure sur la colonne elle-même).
      expect(colonne.className).not.toMatch(/(^|\s)(rounded|border)(\s|$)/)
    }
  })

  it('l\'anneau du focus clavier est rentrant : le contour arrondi ne le rogne pas', () => {
    const { groupe } = monter()
    for (const colonne of [...groupe.children] as HTMLElement[]) {
      expect(colonne.className).toContain('focus-visible:ring-inset')
    }
  })
})

describe('l\'action active est pleine', () => {
  it.each(['follow', 'solo', 'hide'] as const)('%s active : sa colonne est pleine, les deux autres sont au repos', (active) => {
    const { groupe } = monter({ ...AUCUNE, [active]: true })
    const ordre: PlayerPanelAction[] = ['follow', 'solo', 'hide']
    ordre.forEach((action, i) => {
      const colonne = groupe.children[i] as HTMLElement
      expect(colonne.getAttribute('aria-pressed')).toBe(String(action === active))
      expect(colonne.className.includes('bg-primary')).toBe(action === active)
      expect(colonne.className.includes('text-primary-foreground')).toBe(action === active)
    })
  })

  it('aucune action active : aucune colonne pleine', () => {
    const { groupe } = monter()
    for (const colonne of [...groupe.children] as HTMLElement[]) {
      expect(colonne.getAttribute('aria-pressed')).toBe('false')
      expect(colonne.className).not.toContain('bg-primary')
    }
  })
})

describe('les comportements et les libellés ne changent pas', () => {
  it.each(['fr', 'en'] as const)('%s : l\'ordre Suivre, Seul, Masquer, les libellés et les infobulles du dictionnaire', (locale) => {
    const { groupe } = monter(AUCUNE, locale)
    const t = REPLAY_TEXT[locale].playerPanel
    const colonnes = [...groupe.children] as HTMLElement[]
    expect(colonnes.map((c) => c.textContent)).toEqual([t.action.follow, t.action.solo, t.action.hide])
    expect(colonnes.map((c) => c.getAttribute('title'))).toEqual([t.actionHint.follow, t.actionHint.solo, t.actionHint.hide])
  })

  it('un clic demande la bascule de CETTE action, une seule fois', () => {
    const { onToggle } = monter()
    fireEvent.click(screen.getByRole('button', { name: REPLAY_TEXT.fr.playerPanel.action.solo }))
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggle).toHaveBeenCalledWith('solo')
  })

  it('au clavier : Tab atteint chaque colonne dans l\'ordre, Entrée et Espace activent', async () => {
    const user = userEvent.setup()
    const { onToggle } = monter()
    await user.tab()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: REPLAY_TEXT.fr.playerPanel.action.follow }))
    await user.tab()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: REPLAY_TEXT.fr.playerPanel.action.solo }))
    await user.keyboard('{Enter}')
    await user.keyboard(' ')
    expect(onToggle.mock.calls).toEqual([['solo'], ['solo']])
  })

  it('la souris ne prend pas le focus (Espace reste « Lecture »)', () => {
    monter()
    const bouton = screen.getByRole('button', { name: REPLAY_TEXT.fr.playerPanel.action.hide })
    const evenement = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    bouton.dispatchEvent(evenement)
    expect(evenement.defaultPrevented).toBe(true)
  })
})
