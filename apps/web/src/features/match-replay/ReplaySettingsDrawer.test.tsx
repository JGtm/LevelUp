/**
 * ReplaySettingsDrawer.test.tsx — LE TIROIR DE RÉGLAGES : chaque section n'apparaît que si
 * elle a quelque chose à commander, chaque bascule appelle EXACTEMENT son callback (jamais
 * une voisine), et le panneau se ferme par son bouton comme par Échap (même contrat que les
 * autres panneaux du dépôt — AlertDialog, AssetDrawer).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplayHeatmapLegend } from './ReplayHeatmapLegend'
import {
  ReplaySettingsDrawer,
  type ReplayHeatmapControls,
  type ReplayPlacementControls,
} from './ReplaySettingsDrawer'
import type { ReplaySound } from './useReplaySound'

function makePlacements(over: Partial<ReplayPlacementControls> = {}): ReplayPlacementControls {
  return {
    available: true,
    show: true,
    onToggle: vi.fn(),
    unnamedAvailable: true,
    showUnnamed: false,
    onToggleUnnamed: vi.fn(),
    ...over,
  }
}

function makeHeatmap(over: Partial<ReplayHeatmapControls> = {}): ReplayHeatmapControls {
  return {
    show: false,
    onToggle: vi.fn(),
    mode: 'presence',
    onSetMode: vi.fn(),
    killsAvailable: true,
    ...over,
  }
}

function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true,
    on: false,
    toggle: vi.fn(),
    volume: 0.7,
    setVolume: vi.fn(),
    mutedBySpeed: false,
    categories: { weapon: true, grenade: true, melee: true, equipment: true },
    toggleCategory: vi.fn(),
    tick: vi.fn(),
    ...over,
  }
}

function renderDrawer(over: Partial<Parameters<typeof ReplaySettingsDrawer>[0]> = {}) {
  const onClose = vi.fn()
  const onToggleAim = vi.fn()
  const onToggleZones = vi.fn()
  const onToggleNames = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleShotFx = vi.fn()
  const onToggleKillFx = vi.fn()
  const utils = render(
    <ReplaySettingsDrawer
      locale="fr"
      onClose={onClose}
      showAim
      onToggleAim={onToggleAim}
      showZones
      onToggleZones={onToggleZones}
      showNames
      onToggleNames={onToggleNames}
      zonesAvailable
      placements={makePlacements()}
      heatmap={makeHeatmap()}
      showShotFx
      onToggleShotFx={onToggleShotFx}
      showKillFx={false}
      onToggleKillFx={onToggleKillFx}
      sound={makeSound()}
      speed={1}
      onSetSpeed={onSetSpeed}
      {...over}
    />,
  )
  return {
    ...utils, onClose, onToggleAim, onToggleZones, onToggleNames, onSetSpeed,
    onToggleShotFx, onToggleKillFx,
  }
}

describe('ReplaySettingsDrawer — calques', () => {
  it('bascule Visée : reflète showAim et appelle onToggleAim au clic, jamais onToggleZones', () => {
    const { onToggleAim, onToggleZones } = renderDrawer({ showAim: true })
    const btn = screen.getByRole('button', { name: 'Visée' })
    expect(btn).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(btn)
    expect(onToggleAim).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })

  // Le calque des NOMS n'a PAS de condition de disponibilité, contrairement aux zones : un
  // rejeu a toujours des joueurs, donc toujours des noms à écrire ou à taire.
  it('bascule Noms : toujours proposée, reflète showNames, appelle onToggleNames au clic', () => {
    const { onToggleNames, onToggleAim } = renderDrawer({ showNames: false, zonesAvailable: false })
    const btn = screen.getByRole('button', { name: 'Noms' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(onToggleNames).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('bouton Zones absent quand la carte n a pas de zones nommées', () => {
    renderDrawer({ zonesAvailable: false })
    expect(screen.queryByRole('button', { name: 'Zones' })).toBeNull()
  })

  it('bouton Zones présent, reflète showZones, appelle onToggleZones au clic', () => {
    const { onToggleZones } = renderDrawer({ zonesAvailable: true, showZones: false })
    const btn = screen.getByRole('button', { name: 'Zones' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(onToggleZones).toHaveBeenCalledTimes(1)
  })
})

describe('ReplaySettingsDrawer — poses d équipement', () => {
  it('film sans pose dessinable : ni le calque ni les objets non identifiés', () => {
    renderDrawer({ placements: makePlacements({ available: false }) })
    expect(screen.queryByRole('button', { name: 'Équipements posés' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Équipements posés : reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleAim } = renderDrawer({ placements: makePlacements({ show: false, onToggle }) })
    const btn = screen.getByRole('button', { name: 'Équipements posés' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('calque éteint : la bascule des objets non identifiés ne commanderait rien, elle est absente', () => {
    renderDrawer({ placements: makePlacements({ show: false }) })
    expect(screen.queryByRole('button', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('toutes les poses nommées : rien à révéler, la bascule n est pas proposée', () => {
    renderDrawer({ placements: makePlacements({ unnamedAvailable: false }) })
    expect(screen.getByRole('button', { name: 'Équipements posés' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Objets non identifiés : éteinte par défaut, appelle SON callback', () => {
    const onToggleUnnamed = vi.fn()
    const onToggle = vi.fn()
    renderDrawer({ placements: makePlacements({ onToggle, onToggleUnnamed }) })
    const btn = screen.getByRole('button', { name: 'Objets non identifiés' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(onToggleUnnamed).toHaveBeenCalledTimes(1)
    expect(onToggle).not.toHaveBeenCalled()
  })
})

describe('ReplaySettingsDrawer — carte de chaleur', () => {
  it('calque éteint : la bascule est là, le choix de lecture ne l est pas', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: false }) })
    expect(screen.getByRole('button', { name: 'Carte de chaleur' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
    expect(screen.queryByRole('button', { name: 'Présence' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Éliminations' })).toBeNull()
  })

  it('calque allumé : les deux lectures, aria-pressed sur celle en cours', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true, mode: 'kills' }) })
    expect(screen.getByRole('button', { name: 'Présence' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
    expect(screen.getByRole('button', { name: 'Éliminations' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('cliquer une lecture appelle onSetMode avec SA clé, jamais la bascule du calque', () => {
    const onSetMode = vi.fn()
    const onToggle = vi.fn()
    renderDrawer({ heatmap: makeHeatmap({ show: true, onSetMode, onToggle }) })
    fireEvent.click(screen.getByRole('button', { name: 'Éliminations' }))
    expect(onSetMode).toHaveBeenCalledTimes(1)
    expect(onSetMode).toHaveBeenCalledWith('kills')
    expect(onToggle).not.toHaveBeenCalled()
  })

  it('aucune mort localisée : la lecture éliminations n est pas proposée', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true, killsAvailable: false }) })
    expect(screen.queryByRole('button', { name: 'Éliminations' })).toBeNull()
    // Et le choix disparaît entièrement : une seule lecture ne se choisit pas.
    expect(screen.queryByRole('button', { name: 'Présence' })).toBeNull()
  })

  it('la bascule appelle onToggle, jamais un calque voisin', () => {
    const onToggle = vi.fn()
    const { onToggleZones } = renderDrawer({ heatmap: makeHeatmap({ onToggle }) })
    fireEvent.click(screen.getByRole('button', { name: 'Carte de chaleur' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })
})

describe('ReplaySettingsDrawer — vitesse', () => {
  // Le titre de ce test annonce `aria-pressed` : il l'ASSERTE. Sans la ligne du bas,
  // inverser la condition de selection (`speed !== m`) ou la supprimer ne casserait rien.
  it('propose les quatre multiplicateurs, aria-pressed sur celui en cours', () => {
    renderDrawer({ speed: 2 })
    for (const label of ['0.5×', '1×', '2×', '4×']) {
      expect(screen.getByRole('button', { name: label })).toBeTruthy()
    }
    const pressed = ['0.5×', '1×', '2×', '4×'].filter(
      (label) => screen.getByRole('button', { name: label }).getAttribute('aria-pressed') === 'true',
    )
    expect(pressed).toEqual(['2×'])
  })

  it('cliquer un multiplicateur appelle onSetSpeed avec cette valeur', () => {
    const { onSetSpeed } = renderDrawer({ speed: 1 })
    fireEvent.click(screen.getByRole('button', { name: '4×' }))
    expect(onSetSpeed).toHaveBeenCalledWith(4)
  })
})

describe('ReplaySettingsDrawer — son', () => {
  it('aucune commande de son ni de catégorie quand le match n a aucun son', () => {
    renderDrawer({ sound: makeSound({ available: false }) })
    expect(screen.queryByText('Sons par catégorie')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Armes' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Son' })).toBeNull()
  })

  it('les quatre catégories sont affichées, chacune avec son état', () => {
    renderDrawer({
      sound: makeSound({
        categories: { weapon: true, grenade: false, melee: true, equipment: false },
      }),
    })
    expect(screen.getByRole('button', { name: 'Armes' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Grenades' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Mêlée' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Équipements' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('cliquer une catégorie appelle toggleCategory avec SA clé, jamais une voisine', () => {
    const toggleCategory = vi.fn()
    renderDrawer({ sound: makeSound({ toggleCategory }) })
    fireEvent.click(screen.getByRole('button', { name: 'Grenades' }))
    expect(toggleCategory).toHaveBeenCalledTimes(1)
    expect(toggleCategory).toHaveBeenCalledWith('grenade')
  })
})

describe('ReplaySettingsDrawer — effets d événement', () => {
  it('les deux effets ont leur bascule, chacune avec son état', () => {
    renderDrawer({ showShotFx: true, showKillFx: false })
    expect(screen.getByRole('button', { name: 'Effets de tirs' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: 'Effets de mort' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })

  it('chaque bascule appelle SON callback, jamais celui de l autre effet', () => {
    const { onToggleShotFx, onToggleKillFx } = renderDrawer()
    fireEvent.click(screen.getByRole('button', { name: 'Effets de mort' }))
    expect(onToggleKillFx).toHaveBeenCalledTimes(1)
    expect(onToggleShotFx).not.toHaveBeenCalled()
  })

  it('la RÉSERVE de couverture des tirs est à l écran, pas dans un commentaire', () => {
    // Elle est la raison d'être du (i) demandé le 16/08 : le film n'enregistre un tir que
    // lorsqu'un dégât est appliqué, donc l'absence d'éclair ne veut pas dire l'absence de tir.
    renderDrawer()
    const mark = screen.getByRole('img', {
      name: /couverture des tirs peut ne pas être totale/i,
    })
    expect(mark).toHaveAttribute('title', expect.stringContaining("dégât est appliqué"))
  })
})

describe('ReplaySettingsDrawer — cohabitation avec la légende de la carte de chaleur', () => {
  it('le panneau tient le bord DROIT, la légende le coin bas-GAUCHE : deux coins opposés', () => {
    // Depuis que le panneau se pose SUR la carte (16/08), il pourrait masquer ce qui vit
    // dans le cadre du canvas. La légende est le seul élément dans ce cas — elle est ancrée
    // à l'opposé, et le panneau reste AU-DESSUS si un écran étroit les rapproche : c'est
    // l'ordre correct (un panneau ouvert prime une légende), pas une collision.
    const panel = renderDrawer().getByRole('region', { name: 'Réglages' })
    expect(panel.className).toContain('right-0')
    expect(panel.className).toContain('z-20')
    const legend = render(<ReplayHeatmapLegend locale="fr" mode="presence" />)
    const box = legend.container.firstElementChild as HTMLElement
    expect(box.className).toContain('left-2')
    expect(box.className).toContain('bottom-2')
  })
})

describe('ReplaySettingsDrawer — fermeture et focus', () => {
  it('le bouton de fermeture appelle onClose', () => {
    const { onClose } = renderDrawer()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer les réglages' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Échap appelle onClose', () => {
    const { onClose } = renderDrawer()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic DEHORS ferme le panneau (il recouvre la carte : on doit pouvoir en sortir)', () => {
    const { onClose } = renderDrawer()
    fireEvent.pointerDown(document.body)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic DEDANS ne ferme rien — sinon régler quoi que ce soit refermerait le panneau', () => {
    const { onClose } = renderDrawer()
    fireEvent.pointerDown(screen.getByRole('button', { name: 'Visée' }))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('le DÉCLENCHEUR est exclu du clic dehors : sinon il fermerait puis rouvrirait', () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    const { onClose } = renderDrawer({ triggerRef: { current: trigger } })
    fireEvent.pointerDown(trigger)
    expect(onClose).not.toHaveBeenCalled()
    trigger.remove()
  })

  it('le focus entre au panneau à l ouverture (il recouvre les commandes de la carte)', () => {
    renderDrawer()
    expect(document.activeElement).toBe(screen.getByRole('region', { name: 'Réglages' }))
  })
})
