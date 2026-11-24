/**
 * ReplayViewMenu.test.tsx — LE BOUTON « VUE » de la carte : il nomme la vue active (et dit « modifié »), annonce par
 * `aria-expanded` si le panneau est ouvert, l'ouvre et le ferme à la souris comme au clavier, et Échap le ferme en
 * rendant le focus au bouton. L'emplacement du commutateur 2D / 3D vit dans le même conteneur, à sa gauche.
 */
import { act, fireEvent, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'

import { testCapabilities } from '../../../test/capabilities'
import { renderOpenPanel, renderViewMenu } from './replayViewPanelHarness'

describe('le bouton « Vue »', () => {
  beforeEach(() => window.localStorage.clear())

  it('est fermé au départ et nomme la vue active', () => {
    renderViewMenu()
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    expect(btn).toHaveAttribute('aria-haspopup', 'dialog')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('le clic ouvre le panneau et un second clic le referme', () => {
    renderViewMenu()
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    fireEvent.click(btn)
    expect(btn).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('dialog', { name: 'Vue' })).toBeInTheDocument()
    fireEvent.click(btn)
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('choisir un préréglage change le nom porté par le bouton ; un réglage à la main ajoute « · modifié »', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Combat' }))
    expect(screen.getByRole('button', { name: 'Vue Combat' })).toHaveTextContent('VueCombat')
    fireEvent.click(screen.getByRole('switch', { name: 'Visée' }))
    const btn = screen.getByRole('button', { name: 'Vue Combat modifié' })
    expect(btn).toHaveTextContent('VueCombat· modifié')
  })

  it('Échap ferme le panneau et le focus revient au bouton', () => {
    renderViewMenu()
    const btn = screen.getByRole('button', { name: 'Vue Complet' })
    fireEvent.click(btn)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(btn).toHaveFocus()
  })

  it('en anglais : View Full, puis « modified »', () => {
    renderViewMenu({ locale: 'en' })
    expect(screen.getByRole('button', { name: 'View Full' })).toBeInTheDocument()
  })
})

describe('le fil posé sur le plan, retenu', () => {
  const KEY = 'replay-show-plan-feed'
  const option = () => screen.queryByRole('switch', { name: 'Fil posé sur le plan' })
  const ouvrir = (planFeedAvailable: boolean) => {
    renderOpenPanel({ planFeedAvailable })
    fireEvent.click(screen.getByRole('tab', { name: /^Fil/ }))
  }

  beforeEach(() => window.localStorage.clear())

  it('affiché par défaut : l’interrupteur est allumé, sans rien avoir retenu', () => {
    ouvrir(true)
    expect(option()).toHaveAttribute('aria-checked', 'true')
    expect(window.localStorage.getItem(KEY)).toBeNull()
  })

  it('l’éteindre est retenu, l’allumer aussi, et il le reste après remontage', () => {
    ouvrir(true)
    fireEvent.click(option()!)
    expect(window.localStorage.getItem(KEY)).toBe('false')
    fireEvent.click(option()!)
    expect(window.localStorage.getItem(KEY)).toBe('true')
    fireEvent.click(option()!)
    expect(option()).toHaveAttribute('aria-checked', 'false')
  })

  it('en anglais, la ligne porte son libellé anglais', () => {
    renderOpenPanel({ locale: 'en' })
    fireEvent.click(screen.getByRole('tab', { name: /^Feed/ }))
    expect(screen.getByRole('switch', { name: 'Feed on the map' })).toBeInTheDocument()
  })
})

describe('l’emplacement du commutateur 2D / 3D', () => {
  const groupe = () => screen.queryByRole('radiogroup', { name: 'Dimension du rejeu' })
  const avec3d = (valeur: unknown = true) =>
    act(() => useAppShellStore.setState({ capabilities: Object.assign(testCapabilities(), { can_view_replay_3d: valeur }) }))

  beforeEach(() => {
    window.localStorage.clear()
    act(() => useAppShellStore.setState({ capabilities: testCapabilities() }))
  })

  afterEach(() => {
    act(() => useAppShellStore.setState({ capabilities: null }))
  })

  it('capacité absente (le serveur ne la publie pas) : rien n’est rendu, le bouton Vue reste seul', () => {
    renderViewMenu()
    expect(groupe()).toBeNull()
    expect(screen.queryByRole('radio', { name: '3D' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Vue Complet' })).toBeInTheDocument()
  })

  it('capacité vraie : le segment est rendu À GAUCHE du bouton Vue, dans le même conteneur', () => {
    avec3d()
    renderViewMenu()
    const segment = groupe()
    const vue = screen.getByRole('button', { name: 'Vue Complet' })
    expect(segment).toBeTruthy()
    expect(segment!.parentElement).toBe(vue.parentElement)
    expect(segment!.compareDocumentPosition(vue) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('la bascule change la dimension choisie, sans toucher au panneau Vue', () => {
    avec3d()
    renderViewMenu()
    expect(screen.getByRole('radio', { name: '2D' })).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(screen.getByRole('radio', { name: '3D' }))
    expect(screen.getByRole('radio', { name: '3D' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: '2D' })).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByRole('button', { name: 'Vue Complet' })).toHaveAttribute('aria-expanded', 'false')
  })

  it('une capacité qui n’est pas un booléen vrai ne montre rien', () => {
    avec3d('oui')
    renderViewMenu()
    expect(groupe()).toBeNull()
  })
})
