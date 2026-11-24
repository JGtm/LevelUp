/**
 * ReplayViewPanel.views.test.tsx — LES VUES DU PANNEAU, de bout en bout sur les vrais réglages : les préréglages
 * souples (« Combat · modifié », revenir, enregistrer comme ma vue), « Mes vues » (trois au plus, renommer, supprimer,
 * vue d'ouverture, rattachement à un mode), l'ouverture d'un rejeu, et ce que la mémoire retient d'une visite à l'autre.
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { FULL_FILM, SLAYER_FILM } from '../test/settingsContextFixtures'
import { renderOpenPanel, renderViewMenu } from './replayViewPanelHarness'

const sw = (name: string | RegExp) => screen.getByRole('switch', { name })
const chip = (name: string | RegExp) => screen.getByRole('button', { name, pressed: undefined })
const group = () => screen.getByRole('group', { name: 'Préréglages' })
const pressed = () => within(group()).getAllByRole('button').filter((b) => b.getAttribute('aria-pressed') === 'true').map((b) => b.textContent)

beforeEach(() => window.localStorage.clear())

describe('les préréglages souples', () => {
  it('quatre puces ; Complet est la vue d’ouverture, pressée, sans « modifié » ; le bouton la nomme', () => {
    renderOpenPanel()
    expect(within(group()).getAllByRole('button').map((b) => b.textContent)).toEqual(['Complet', 'Combat', 'Déplacements', 'Objectif'])
    expect(pressed()).toEqual(['Complet'])
    expect(screen.getByRole('button', { name: 'Vue Complet' })).toBeInTheDocument()
    expect(screen.getByText(/Un préréglage est un point de départ/)).toBeInTheDocument()
  })

  it('choisir Combat écrit ses réglages (morts et tueurs allumés, zones éteintes, fil sur les éliminations) sans « modifié »', () => {
    renderOpenPanel()
    fireEvent.click(chip('Combat'))
    expect(pressed()).toEqual(['Combat'])
    expect(window.localStorage.getItem('replay-show-deaths')).toBe('true')
    expect(window.localStorage.getItem('replay-show-killers')).toBe('true')
    expect(window.localStorage.getItem('replay-show-zones')).toBe('false')
    expect(window.localStorage.getItem('replay-feed-filter')).toBe('kills')
    expect(window.localStorage.getItem('replay-view-preset')).toBe('combat')
    expect(screen.queryByText(/· modifié/)).toBeNull()
  })

  it('un réglage changé à la main dit « Combat · modifié », combien ont changé, et le bouton Vue le dit aussi', () => {
    renderOpenPanel()
    fireEvent.click(chip('Combat'))
    fireEvent.click(sw('Visée'))
    expect(within(group()).getByRole('button', { name: /Combat/ })).toHaveTextContent('Combat· modifié')
    expect(screen.getByText('Combat · modifié')).toBeInTheDocument()
    expect(screen.getByText('1 réglage changé')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Vue Combat modifié' })).toBeInTheDocument()
    fireEvent.click(sw('Traînée'))
    expect(screen.getByText('2 réglages changés')).toBeInTheDocument()
  })

  it('« Revenir à Combat » rétablit la vue et efface « modifié »', () => {
    renderOpenPanel()
    fireEvent.click(chip('Combat'))
    fireEvent.click(sw('Visée'))
    fireEvent.click(screen.getByRole('button', { name: 'Revenir à Combat' }))
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'true')
    expect(screen.queryByText(/· modifié/)).toBeNull()
    expect(screen.getByRole('status')).toHaveTextContent('Vue rétablie : Combat')
  })

  it('un réglage sans objet sur ce match ne rend pas la vue modifiée', () => {
    renderOpenPanel({ film: SLAYER_FILM })
    fireEvent.click(chip('Combat'))
    expect(screen.queryByText(/· modifié/)).toBeNull()
    expect(pressed()).toEqual(['Combat'])
  })

  it('l’état modifié se déduit : rebasculer le réglage rend la vue telle quelle, rien n’est écrit pour lui', () => {
    renderOpenPanel()
    fireEvent.click(chip('Déplacements'))
    fireEvent.click(sw('Visée'))
    expect(screen.getByText('Déplacements · modifié')).toBeInTheDocument()
    fireEvent.click(sw('Visée'))
    expect(screen.queryByText(/· modifié/)).toBeNull()
  })

  it('le fond, l’orientation, l’étage et la couleur des points ne font pas partie des préréglages', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('radio', { name: 'Par joueur' }))
    expect(screen.queryByText(/· modifié/)).toBeNull()
    fireEvent.click(chip('Objectif'))
    expect(screen.getByRole('radio', { name: 'Par joueur' })).toHaveAttribute('aria-checked', 'true')
  })

  it('en anglais : Full, Combat, Movement, Objective, « modified » et « Back to »', () => {
    renderOpenPanel({ locale: 'en' })
    expect(within(screen.getByRole('group', { name: 'Presets' })).getAllByRole('button').map((b) => b.textContent)).toEqual(['Full', 'Combat', 'Movement', 'Objective'])
    fireEvent.click(screen.getByRole('button', { name: 'Combat' }))
    fireEvent.click(sw('Aim'))
    expect(screen.getByText('Combat · modified')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back to Combat' })).toBeInTheDocument()
  })
})

describe('enregistrer comme ma vue', () => {
  const saveAs = (name: string) => {
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer comme ma vue' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Nom de la vue' }), { target: { value: name } })
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer' }))
  }

  it('l’état courant devient une vue nommée, pressée à côté des préréglages, retenue d’une visite à l’autre', () => {
    const view = renderOpenPanel()
    fireEvent.click(sw('Visée'))
    saveAs('Sans regard')
    expect(within(group()).getByRole('button', { name: 'Sans regard' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('status')).toHaveTextContent('Vue « Sans regard » enregistrée.')
    expect(screen.getByText(/Ma vue enregistrée/)).toBeInTheDocument()
    view.unmount()
    renderOpenPanel()
    expect(pressed()).toEqual(['Sans regard'])
    expect(JSON.parse(window.localStorage.getItem('replay-views') ?? '[]')[0].name).toBe('Sans regard')
  })

  it('un nom vide laisse le formulaire ouvert ; Échap ferme le formulaire d’abord, le panneau à la frappe suivante', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer comme ma vue' }))
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer' }))
    expect(screen.getByRole('textbox', { name: 'Nom de la vue' })).toBeInTheDocument()
    fireEvent.keyDown(screen.getByRole('textbox', { name: 'Nom de la vue' }), { key: 'Escape' })
    expect(screen.queryByRole('textbox', { name: 'Nom de la vue' })).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Vue' })).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('appliquer une autre vue puis revenir à la sienne rétablit ses réglages', () => {
    renderOpenPanel()
    fireEvent.click(sw('Visée'))
    saveAs('Sans regard')
    fireEvent.click(chip('Combat'))
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(chip('Sans regard'))
    expect(sw('Visée')).toHaveAttribute('aria-checked', 'false')
  })

  it('« Mettre à jour cette vue » n’existe que pour une vue enregistrée modifiée, et en retient le nouvel état', () => {
    renderOpenPanel()
    fireEvent.click(chip('Combat'))
    fireEvent.click(sw('Visée'))
    expect(screen.queryByRole('button', { name: 'Mettre à jour cette vue' })).toBeNull()
    saveAs('Ma vue')
    fireEvent.click(sw('Traînée'))
    expect(screen.getByText('Ma vue · modifié')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Mettre à jour cette vue' }))
    expect(screen.queryByText(/· modifié/)).toBeNull()
    expect(screen.getByRole('status')).toHaveTextContent('Vue « Ma vue » mise à jour.')
  })

  it('trois vues au plus : à la quatrième, une phrase dit quoi faire', () => {
    renderOpenPanel()
    for (const n of ['A', 'B', 'C']) saveAs(n)
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer comme ma vue' }))
    expect(screen.getByText(/3 vues au plus/)).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Nom de la vue' })).toBeNull()
  })
})

describe('Mes vues : le gestionnaire', () => {
  const withOneView = () => {
    renderOpenPanel()
    fireEvent.click(sw('Visée'))
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer comme ma vue' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Nom de la vue' }), { target: { value: 'Ma vue' } })
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer' }))
    fireEvent.click(screen.getByRole('button', { name: /^Mes vues/ }))
  }

  it('dit combien de vues sur trois, et aucune vue quand il n’y en a pas', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Mes vues (0 sur 3)' }))
    expect(screen.getByText(/Aucune vue enregistrée/)).toBeInTheDocument()
  })

  it('renomme une vue en place', () => {
    withOneView()
    const nom = screen.getByRole('textbox', { name: 'Nom de la vue (renommer)' })
    fireEvent.change(nom, { target: { value: 'Autre nom' } })
    fireEvent.blur(nom)
    expect(within(group()).getByRole('button', { name: 'Autre nom' })).toBeInTheDocument()
  })

  it('l’étoile pose la vue d’ouverture et la retire ; la liste « À l’ouverture d’un rejeu » le dit aussi', () => {
    withOneView()
    fireEvent.click(screen.getByRole('button', { name: 'Vue d’ouverture par défaut : Ma vue' }))
    expect(window.localStorage.getItem('replay-view-default')).not.toBe('')
    expect(screen.getByRole('combobox', { name: 'Vue appliquée à l’ouverture d’un rejeu' })).toHaveDisplayValue('Ma vue')
    fireEvent.click(screen.getByRole('button', { name: 'Vue d’ouverture par défaut : Ma vue' }))
    expect(screen.getByRole('combobox', { name: 'Vue appliquée à l’ouverture d’un rejeu' })).toHaveDisplayValue('Dernière vue utilisée')
  })

  it('la liste d’ouverture propose Dernière vue utilisée, les préréglages et les vues', () => {
    withOneView()
    const liste = screen.getByRole('combobox', { name: 'Vue appliquée à l’ouverture d’un rejeu' })
    expect(within(liste).getAllByRole('option').map((o) => o.textContent)).toEqual(['Dernière vue utilisée', 'Complet', 'Combat', 'Déplacements', 'Objectif', 'Ma vue'])
    fireEvent.change(liste, { target: { value: 'combat' } })
    expect(window.localStorage.getItem('replay-view-default')).toBe('combat')
  })

  it('le rattachement à un mode est facultatif et propose les enjeux du film', () => {
    withOneView()
    const mode = screen.getByRole('combobox', { name: 'Rattacher « Ma vue » à un mode de jeu' })
    expect(within(mode).getAllByRole('option').map((o) => o.textContent)).toEqual(['Aucun', 'à drapeau', 'à crâne', 'à VIP', 'à bombe', 'à zones', 'sans objectif'])
    fireEvent.change(mode, { target: { value: 'flag' } })
    expect(JSON.parse(window.localStorage.getItem('replay-views') ?? '[]')[0].mode).toBe('flag')
  })

  it('supprimer la vue active ramène à Complet ; la vue disparaît des puces', () => {
    withOneView()
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer la vue Ma vue' }))
    expect(within(group()).queryByRole('button', { name: 'Ma vue' })).toBeNull()
    // Les réglages restent comme ils sont : la vue Complet, dont la visée éteinte s'écarte, se dit modifiée.
    expect(pressed()).toEqual(['Complet· modifié'])
    expect(screen.getByRole('status')).toHaveTextContent('Vue « Ma vue » supprimée.')
  })
})

describe('l’ouverture d’un rejeu', () => {
  const seed = (views: unknown, extra: Record<string, string> = {}) => {
    window.localStorage.setItem('replay-views', JSON.stringify(views))
    for (const [k, v] of Object.entries(extra)) window.localStorage.setItem(k, v)
  }
  const MA_VUE = (mode: string | null = null) => ({
    id: 'v1', name: 'Ma vue', mode,
    values: { aim: false, trail: true, forecast: true, feedFilter: 'tout' },
  })

  it('sans choix, les réglages restent tels qu’ils ont été laissés', () => {
    window.localStorage.setItem('replay-show-aim', 'false')
    renderViewMenu()
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
    expect(screen.getByRole('button', { name: /^Vue Complet/ })).toBeInTheDocument()
  })

  it('la vue d’ouverture s’applique à l’ouverture du rejeu', () => {
    seed([MA_VUE()], { 'replay-view-default': 'v1' })
    renderViewMenu()
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
    expect(screen.getByRole('button', { name: 'Vue Ma vue' })).toBeInTheDocument()
  })

  it('une vue rattachée au mode du match passe avant la vue d’ouverture', () => {
    seed([MA_VUE('flag')], { 'replay-view-default': 'combat' })
    renderViewMenu({ film: FULL_FILM })
    expect(screen.getByRole('button', { name: 'Vue Ma vue' })).toBeInTheDocument()
  })

  it('une vue rattachée à un autre mode ne s’applique pas : la vue d’ouverture reste', () => {
    seed([MA_VUE('bomb')], { 'replay-view-default': 'combat' })
    renderViewMenu({ film: FULL_FILM })
    expect(screen.getByRole('button', { name: 'Vue Combat' })).toBeInTheDocument()
  })
})
