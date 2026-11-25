/**
 * ReplayQuickAccess.test.tsx — L'ACCÈS RAPIDE au bord de la carte : les interrupteurs épinglés, à gauche du bouton
 * « Vue », sur le même état que le panneau ; la raison d'un réglage grisé ; les épingles posées depuis le panneau ;
 * les trois cadres de la page.
 */
import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '../../../test/render-utils'
import { baseProps, installCanvasDoubles, smallDoc } from '../test/canvasTestKit'
import { SLAYER_FILM } from '../test/settingsContextFixtures'
import { canvasSlotsOf } from '../ui/frames/replayFrameSlots'
import { ReplayCanvas } from '../ui/ReplayCanvas'
import { renderOpenPanel, renderViewMenu } from './replayViewPanelHarness'

const toolbar = () => screen.getByRole('toolbar', { name: 'Accès rapide aux réglages' })
const quick = (name: string) => within(toolbar()).getByRole('button', { name })

beforeEach(() => window.localStorage.clear())

describe('l’accès rapide', () => {
  it('pose les cinq épingles d’usine à gauche du bouton Vue, dans le même conteneur', () => {
    renderViewMenu()
    expect(within(toolbar()).getAllByRole('button').map((b) => b.getAttribute('aria-label'))).toEqual([
      'Visée', 'Traînée', 'Carte de chaleur', 'Fil posé sur le plan', 'Répliques en bulles',
    ])
    const vue = screen.getByRole('button', { name: 'Vue Complet' })
    expect(toolbar().parentElement).toBe(vue.parentElement)
    expect(toolbar().compareDocumentPosition(vue) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('un bouton épinglé allume et éteint son réglage, sans ouvrir le panneau, et dit son état', () => {
    renderViewMenu()
    expect(quick('Visée')).toHaveAttribute('aria-pressed', 'true')
    expect(quick('Visée').title).toBe('Visée : activé')
    fireEvent.click(quick('Visée'))
    expect(quick('Visée')).toHaveAttribute('aria-pressed', 'false')
    expect(quick('Visée').title).toBe('Visée : désactivé')
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(quick('Carte de chaleur'))
    expect(window.localStorage.getItem('replay-show-heatmap')).toBe('true')
  })

  it('c’est le même état que le panneau : la ligne et le bouton bougent ensemble', () => {
    renderOpenPanel()
    fireEvent.click(quick('Traînée'))
    expect(screen.getByRole('switch', { name: 'Traînée' })).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(screen.getByRole('switch', { name: 'Traînée' }))
    expect(quick('Traînée')).toHaveAttribute('aria-pressed', 'true')
  })

  it('modifier un réglage épinglé rend la vue modifiée', () => {
    renderViewMenu()
    fireEvent.click(quick('Visée'))
    expect(screen.getByRole('button', { name: 'Vue Complet modifié' })).toBeInTheDocument()
  })

  it('un réglage épinglé sans objet sur ce match reste affiché, grisé, avec la raison', () => {
    window.localStorage.setItem('replay-pins', JSON.stringify(['aim', 'zones', 'planFeed']))
    renderViewMenu({ film: SLAYER_FILM, planFeedAvailable: false })
    expect(quick('Zones nommées')).toBeDisabled()
    expect(quick('Zones nommées').title).toBe('Zones nommées : Sans objet sur ce match')
    expect(quick('Fil posé sur le plan')).toBeDisabled()
    expect(quick('Fil posé sur le plan').title).toBe('Fil posé sur le plan : Disponible au Cinéma et au Plein cadre')
    expect(quick('Visée')).toBeEnabled()
  })

  it('une épingle posée depuis le panneau paraît au bord de la carte ; la retirer la retire', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Retirer « Visée » de l’accès rapide' }))
    expect(within(toolbar()).queryByRole('button', { name: 'Visée' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Épingler « Effets de tirs » à l’accès rapide' }))
    expect(quick('Effets de tirs')).toBeInTheDocument()
  })

  it('sans aucune épingle, rien n’est posé au bord de la carte', () => {
    window.localStorage.setItem('replay-pins', '[]')
    renderViewMenu()
    expect(screen.queryByRole('toolbar')).toBeNull()
    expect(screen.getByRole('button', { name: 'Vue Complet' })).toBeInTheDocument()
  })

  it('les épingles survivent à la page', () => {
    const premier = renderOpenPanel()
    fireEvent.click(screen.getByRole('tab', { name: /Carte/ }))
    fireEvent.click(screen.getByRole('tab', { name: /Joueurs/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Retirer « Visée » de l’accès rapide' })) // les cinq épingles d’usine sont posées
    fireEvent.click(screen.getByRole('tab', { name: /Carte/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Épingler « Zones nommées » à l’accès rapide' }))
    premier.unmount()
    renderViewMenu()
    expect(quick('Zones nommées')).toBeInTheDocument()
  })

  it('en anglais : le groupe, les boutons et les infobulles portent leur libellé anglais', () => {
    renderViewMenu({ locale: 'en' })
    const barre = screen.getByRole('toolbar', { name: 'Quick access to settings' })
    expect(within(barre).getByRole('button', { name: 'Aim' }).title).toBe('Aim: on')
    expect(within(barre).getByRole('button', { name: 'Feed on the map' })).toBeInTheDocument()
  })
})

describe('l’accès rapide dans les trois cadres', () => {
  beforeEach(installCanvasDoubles)
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  const montage = (frame: 'standard' | 'cinema' | 'plein-cadre') => {
    const doc = smallDoc()
    const slots = canvasSlotsOf(frame)
    return renderWithProviders(
      <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={frame !== 'standard'} />,
    )
  }

  it.each(['standard', 'cinema', 'plein-cadre'] as const)(
    'au cadre %s : la barre d’accès rapide est à côté du bouton Vue, et le fil posé sur le plan suit le cadre',
    (frame) => {
      montage(frame)
      expect(toolbar()).toBeInTheDocument()
      expect(quick('Visée')).toBeEnabled()
      if (frame === 'standard') expect(quick('Fil posé sur le plan')).toBeDisabled()
      else expect(quick('Fil posé sur le plan')).toBeEnabled()
      expect(toolbar().parentElement).toBe(screen.getByRole('button', { name: /^Vue / }).parentElement)
    },
  )

  it('un geste au bord de la carte écrit la préférence que le panneau lit', () => {
    montage('cinema')
    fireEvent.click(quick('Traînée'))
    expect(window.localStorage.getItem('replay-show-trail')).toBe('false')
    fireEvent.click(screen.getByRole('button', { name: /^Vue / }))
    expect(screen.getByRole('switch', { name: 'Traînée' })).toHaveAttribute('aria-checked', 'false')
  })
})
