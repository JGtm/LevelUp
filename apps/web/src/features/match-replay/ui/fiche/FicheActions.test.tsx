/**
 * Tests — L'ACTION « SEUL » de la fiche (`FicheActions`) : un seul bouton à bascule (`aria-pressed`), libellé et
 * infobulle de chaque état dans les deux langues, accessible au clavier, la souris ne vole pas le focus. Suivre et
 * Masquer n'y sont plus (ils sont sur les tuiles et les lignes).
 */
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from '../../i18n/i18n'
import { FicheActions } from './FicheActions'
import type { FicheViewpoint } from './ficheViewpoint'

function monter(solo = false, locale: 'fr' | 'en' = 'fr', onSolo = vi.fn()) {
  const view = render(<FicheActions name="JGtm" solo={solo} onSolo={onSolo} locale={locale} />)
  return { ...view, onSolo, groupe: screen.getByRole('group', { name: 'JGtm' }) }
}

describe('la forme : « Seul » est la seule action de la fiche', () => {
  it('un seul bouton dans le groupe', () => {
    const { groupe } = monter()
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(groupe.querySelectorAll('button')).toHaveLength(1)
  })

  it('ni Suivre ni Masquer ne sont dans la fiche', () => {
    monter()
    expect(screen.queryByRole('button', { name: 'Caméra : suivre' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Masquer' })).toBeNull()
  })

  it('l\'anneau du focus clavier est visible', () => {
    monter()
    expect(screen.getByRole('button').className).toContain('focus-visible:ring-2')
  })
})

describe('l\'état actif', () => {
  it('au repos : « Seul », non enfoncé, au repos (fond neutre)', () => {
    monter(false)
    const bouton = screen.getByRole('button', { name: 'Seul' })
    expect(bouton.getAttribute('aria-pressed')).toBe('false')
    expect(bouton.className).not.toContain('bg-primary')
  })

  it('actif : « Seul · tout réafficher », enfoncé, plein', () => {
    monter(true)
    const bouton = screen.getByRole('button', { name: 'Seul · tout réafficher' })
    expect(bouton.getAttribute('aria-pressed')).toBe('true')
    expect(bouton.className).toContain('bg-primary')
    expect(bouton.className).toContain('text-primary-foreground')
  })
})

describe('les libellés et les infobulles', () => {
  it.each([
    ['fr', false, 'Seul', "Masque tous les autres joueurs. L'œil d'une tuile en réaffiche un."],
    ['fr', true, 'Seul · tout réafficher', 'Réaffiche tous les joueurs.'],
    ['en', false, 'Solo', "Hides every other player. A tile's eye shows one again."],
    ['en', true, 'Solo · show all', 'Shows every player again.'],
  ] as const)('%s, actif=%s : « %s » et son infobulle', (locale, solo, libelle, infobulle) => {
    monter(solo, locale)
    const bouton = screen.getByRole('button')
    expect(bouton.textContent).toBe(libelle)
    expect(bouton.getAttribute('title')).toBe(infobulle)
  })

  it('les libellés viennent du dictionnaire du rejeu', () => {
    monter(false, 'fr')
    const t = REPLAY_TEXT.fr.playerPanel.solo
    expect(screen.getByRole('button').textContent).toBe(t.label)
  })
})

describe('« Prendre le point de vue »', () => {
  const montrer = (viewpoint: FicheViewpoint, locale: 'fr' | 'en' = 'fr') =>
    render(<FicheActions name="JGtm" solo={false} onSolo={vi.fn()} locale={locale} viewpoint={viewpoint} />)

  it('sans point de vue donné, la fiche ne porte que « Seul » (le bouton n’existe pas)', () => {
    monter()
    expect(screen.queryByRole('button', { name: 'Prendre le point de vue' })).toBeNull()
  })

  it('un clic prend le point de vue de ce joueur, une seule fois', () => {
    const onTake = vi.fn()
    montrer({ current: false, inert: false, onTake })
    fireEvent.click(screen.getByRole('button', { name: 'Prendre le point de vue' }))
    expect(onTake).toHaveBeenCalledTimes(1)
  })

  it('le joueur déjà regardé n’a pas de bouton : rien d’inutile à cliquer', () => {
    montrer({ current: true, inert: false, onTake: vi.fn() })
    expect(screen.queryByRole('button', { name: 'Prendre le point de vue' })).toBeNull()
  })

  it('un joueur sans ligne de tableau de score reste inerte, avec la raison en infobulle', () => {
    const onTake = vi.fn()
    montrer({ current: false, inert: true, onTake })
    const bouton = screen.getByRole('button', { name: 'Prendre le point de vue' })
    expect(bouton).toBeDisabled()
    expect(bouton.getAttribute('title')).toBe(REPLAY_TEXT.fr.viewpointNoData)
    fireEvent.click(bouton)
    expect(onTake).not.toHaveBeenCalled()
  })

  it('en anglais', () => {
    montrer({ current: false, inert: false, onTake: vi.fn() }, 'en')
    expect(screen.getByRole('button', { name: 'Take the viewpoint' })).toBeInTheDocument()
  })

  it('la souris ne prend pas le focus (Espace reste « Lecture »)', () => {
    montrer({ current: false, inert: false, onTake: vi.fn() })
    const evenement = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    screen.getByRole('button', { name: 'Prendre le point de vue' }).dispatchEvent(evenement)
    expect(evenement.defaultPrevented).toBe(true)
  })
})

describe('les comportements', () => {
  it('un clic demande Seul, une seule fois', () => {
    const { onSolo } = monter()
    fireEvent.click(screen.getByRole('button', { name: 'Seul' }))
    expect(onSolo).toHaveBeenCalledTimes(1)
  })

  it('un clic sur « Seul · tout réafficher » demande la même bascule (la logique décide)', () => {
    const { onSolo } = monter(true)
    fireEvent.click(screen.getByRole('button', { name: 'Seul · tout réafficher' }))
    expect(onSolo).toHaveBeenCalledTimes(1)
  })

  it('au clavier : Tab atteint le bouton, Entrée et Espace activent', async () => {
    const user = userEvent.setup()
    const { onSolo } = monter()
    await user.tab()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Seul' }))
    await user.keyboard('{Enter}')
    await user.keyboard(' ')
    expect(onSolo).toHaveBeenCalledTimes(2)
  })

  it('la souris ne prend pas le focus (Espace reste « Lecture »)', () => {
    monter()
    const evenement = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    screen.getByRole('button', { name: 'Seul' }).dispatchEvent(evenement)
    expect(evenement.defaultPrevented).toBe(true)
  })
})
