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
  type ReplayFlagControls,
  type ReplayWeaponPadControls,
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
    droppedAvailable: true,
    showDropped: true,
    onToggleDropped: vi.fn(),
    ...over,
  }
}

function makeWeaponPads(over: Partial<ReplayWeaponPadControls> = {}): ReplayWeaponPadControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

function makeFlagCarries(over: Partial<ReplayFlagControls> = {}): ReplayFlagControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

function makeHeatmap(over: Partial<ReplayHeatmapControls> = {}): ReplayHeatmapControls {
  return {
    show: false,
    onToggle: vi.fn(),
    mode: 'presence',
    onSetMode: vi.fn(),
    span: 'match',
    onSetSpan: vi.fn(),
    killsAvailable: true,
    ...over,
  }
}

function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true,
    on: false,
    toggle: vi.fn(),
    wake: vi.fn(),
    volume: 0.7,
    setVolume: vi.fn(),
    mutedBySpeed: false,
    categories: { weapon: true, grenade: true, melee: true, equipment: true, objective: true },
    toggleCategory: vi.fn(),
    tick: vi.fn(),
    endMatch: vi.fn(),
    recordingTrack: () => null,
  exportTrack: () => ({ timeline: [], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] } }),
    ...over,
  }
}

function renderDrawer(over: Partial<Parameters<typeof ReplaySettingsDrawer>[0]> = {}) {
  const onClose = vi.fn()
  const onToggleAim = vi.fn()
  const onToggleZones = vi.fn()
  const onToggleTrail = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleShotFx = vi.fn()
  const onToggleKillFx = vi.fn()
  const onSetMarkerColors = vi.fn()
  const utils = render(
    <ReplaySettingsDrawer
      locale="fr"
      onClose={onClose}

      showAim
      onToggleAim={onToggleAim}
      showZones
      onToggleZones={onToggleZones}
      showTrail
      onToggleTrail={onToggleTrail}
      zonesAvailable
      placements={makePlacements()}
      weaponPads={makeWeaponPads()}
      groundWeapons={makeWeaponPads()}
      flagCarries={makeFlagCarries()}
      vipCrown={makeFlagCarries()}
      skullCarrier={makeFlagCarries()}
      bombCarrier={makeFlagCarries()}
      heatmap={makeHeatmap()}
      showShotFx
      onToggleShotFx={onToggleShotFx}
      showKillFx={false}
      onToggleKillFx={onToggleKillFx}
      sound={makeSound()}
      markerColors="team"
      onSetMarkerColors={onSetMarkerColors}
      {...over}
    />,
  )
  return {
    ...utils, onClose, onToggleAim, onToggleZones, onToggleTrail, onSetSpeed,
    onToggleShotFx, onToggleKillFx, onSetMarkerColors,
  }
}

describe('ReplaySettingsDrawer — les deux formes de commande (2026-08-29)', () => {
  // UN OUI/NON EST UN INTERRUPTEUR, UN CHOIX EXCLUSIF RESTE UN BOUTON PRESSÉ. La demande
  // utilisateur (« je préfère un toggle plutôt que des boutons ») porte sur les réglages
  // oui/non ; un interrupteur sur « Par équipe » promettrait un « tout éteint » que le
  // réglage n'accepte pas. Ce test épingle la frontière, qui se perdrait à la relecture.
  it('les calques et les catégories de son sont des interrupteurs', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Visée', 'Traînée', 'Zones', 'Carte de chaleur', 'Armes']) {
      expect(screen.getByRole('switch', { name: nom })).toBeTruthy()
    }
  })

  it('les choix exclusifs ne sont JAMAIS des interrupteurs', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Présence', 'Éliminations', 'Par équipe', 'Par joueur']) {
      expect(screen.queryByRole('switch', { name: nom })).toBeNull()
    }
  })

  // LES DEUX AXES DE LA CHALEUR SONT PASSÉS EN SEGMENTÉ le 2026-09-02 : un choix parmi N
  // s'annonce en radiogroup/radio, pas comme une grappe de boutons pressés. C'est la forme
  // qui porte la contrainte — des options empilées laissaient croire qu'on pouvait en allumer
  // deux. Les couleurs de marqueur, elles, restent des boutons : leur liste n'a pas bougé.
  it('les axes de la chaleur sont un choix radio, pas une grappe de boutons', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Présence', 'Éliminations']) {
      expect(screen.getByRole('radio', { name: nom })).toBeTruthy()
    }
    expect(screen.getAllByRole('radiogroup').length).toBeGreaterThanOrEqual(2)
  })
})

describe('ReplaySettingsDrawer — calques', () => {
  it('bascule Visée : reflète showAim et appelle onToggleAim au clic, jamais onToggleZones', () => {
    const { onToggleAim, onToggleZones } = renderDrawer({ showAim: true })
    const btn = screen.getByRole('switch', { name: 'Visée' })
    expect(btn).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(btn)
    expect(onToggleAim).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })

  // LE CALQUE DES NOMS N'A PLUS DE BASCULE (2026-09-02) : il est toujours allumé. Le test
  // garde l'ABSENCE — sans lui, une revue future la réintroduirait par symétrie avec les
  // autres calques, et le tiroir reprendrait la ligne qu'on vient de lui faire rendre.
  it('aucune bascule « Noms » : le calque est toujours allumé', () => {
    renderDrawer({ zonesAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Noms' })).toBeNull()
  })

  // V1 (2026-08-18) : la TRAÎNÉE devient un calque comme les autres — toujours proposée (elle
  // n'a aucune condition de disponibilité : une vie a toujours un passé), allumée par défaut.
  it('bascule Traînée : toujours proposée, reflète showTrail, appelle onToggleTrail au clic', () => {
    const { onToggleTrail, onToggleAim } = renderDrawer({ showTrail: false, zonesAvailable: false })
    const btn = screen.getByRole('switch', { name: 'Traînée' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleTrail).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('bouton Zones absent quand la carte n a pas de zones nommées', () => {
    renderDrawer({ zonesAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Zones' })).toBeNull()
  })

  it('bouton Zones présent, reflète showZones, appelle onToggleZones au clic', () => {
    const { onToggleZones } = renderDrawer({ zonesAvailable: true, showZones: false })
    const btn = screen.getByRole('switch', { name: 'Zones' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleZones).toHaveBeenCalledTimes(1)
  })
})

describe('ReplaySettingsDrawer — poses d équipement', () => {
  it('film sans pose dessinable : ni le calque ni les objets non identifiés', () => {
    renderDrawer({ placements: makePlacements({ available: false }) })
    expect(screen.queryByRole('switch', { name: 'Équipements posés' })).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Équipements posés : reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleAim } = renderDrawer({ placements: makePlacements({ show: false, onToggle }) })
    const btn = screen.getByRole('switch', { name: 'Équipements posés' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('calque éteint : la bascule des objets non identifiés ne commanderait rien, elle est absente', () => {
    renderDrawer({ placements: makePlacements({ show: false }) })
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('toutes les poses nommées : rien à révéler, la bascule n est pas proposée', () => {
    renderDrawer({ placements: makePlacements({ unnamedAvailable: false }) })
    expect(screen.getByRole('switch', { name: 'Équipements posés' })).toBeTruthy()
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Objets non identifiés : éteinte par défaut, appelle SON callback', () => {
    const onToggleUnnamed = vi.fn()
    const onToggle = vi.fn()
    renderDrawer({ placements: makePlacements({ onToggle, onToggleUnnamed }) })
    const btn = screen.getByRole('switch', { name: 'Objets non identifiés' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleUnnamed).toHaveBeenCalledTimes(1)
    expect(onToggle).not.toHaveBeenCalled()
  })
})

describe('ReplaySettingsDrawer — emplacements d arme', () => {
  it('film sans socle publié : pas de bascule (elle ne commanderait rien)', () => {
    renderDrawer({ weaponPads: makeWeaponPads({ available: false }) })
    expect(screen.queryByRole('switch', { name: "Emplacements d'arme" })).toBeNull()
  })

  it('bascule Emplacements d arme : reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleAim } = renderDrawer({
      weaponPads: makeWeaponPads({ show: false, onToggle }),
      placements: makePlacements({ onToggle: vi.fn() }),
    })
    const btn = screen.getByRole('switch', { name: "Emplacements d'arme" })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })
})

describe('ReplaySettingsDrawer — carte de chaleur', () => {
  it('calque éteint : la bascule est là, le choix de lecture ne l est pas', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: false }) })
    expect(screen.getByRole('switch', { name: 'Carte de chaleur' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
    expect(screen.queryByRole('button', { name: 'Présence' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Éliminations' })).toBeNull()
  })

  it('calque allumé : les deux lectures, la courante pressée (choix exclusif)', () => {
    renderDrawer({ heatmap: makeHeatmap({ show: true, mode: 'kills' }) })
    expect(screen.getByRole('radio', { name: 'Présence' })).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByRole('radio', { name: 'Éliminations' })).toHaveAttribute('aria-checked', 'true')
  })

  it('cliquer une lecture appelle onSetMode avec SA clé, jamais la bascule du calque', () => {
    const onSetMode = vi.fn()
    const onToggle = vi.fn()
    renderDrawer({ heatmap: makeHeatmap({ show: true, onSetMode, onToggle }) })
    fireEvent.click(screen.getByRole('radio', { name: 'Éliminations' }))
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
    fireEvent.click(screen.getByRole('switch', { name: 'Carte de chaleur' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })
})

describe('ReplaySettingsDrawer — couleur des points', () => {
  it('propose les deux lectures, la courante pressée (choix exclusif)', () => {
    renderDrawer({ markerColors: 'player' })
    expect(screen.getByRole('button', { name: 'Par équipe' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Par joueur' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('cliquer une lecture appelle onSetMarkerColors avec SON mode', () => {
    const { onSetMarkerColors } = renderDrawer({ markerColors: 'team' })
    fireEvent.click(screen.getByRole('button', { name: 'Par joueur' }))
    expect(onSetMarkerColors).toHaveBeenCalledWith('player')
  })
})

describe('ReplaySettingsDrawer — son (le filtre par catégorie seul : l’interrupteur vit à la barre de lecture)', () => {
  it('aucune commande de son ni de catégorie quand le match n a aucun son', () => {
    renderDrawer({ sound: makeSound({ available: false }) })
    expect(screen.queryByText('Sons par catégorie')).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Armes' })).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Son' })).toBeNull()
  })

  it('l’interrupteur du son n’est PLUS au tiroir, même quand le son est disponible', () => {
    renderDrawer()
    expect(screen.queryByRole('switch', { name: 'Son' })).toBeNull()
    expect(screen.getByText('Sons par catégorie')).toBeTruthy()
  })

  it('les quatre catégories sont affichées, chacune avec son état', () => {
    renderDrawer({
      sound: makeSound({
        categories: { weapon: true, grenade: false, melee: true, equipment: false, objective: false },
      }),
    })
    expect(screen.getByRole('switch', { name: 'Armes' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('switch', { name: 'Grenades' })).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByRole('switch', { name: 'Mêlée' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('switch', { name: 'Équipements' })).toHaveAttribute('aria-checked', 'false')
  })

  it('cliquer une catégorie appelle toggleCategory avec SA clé, jamais une voisine', () => {
    const toggleCategory = vi.fn()
    renderDrawer({ sound: makeSound({ toggleCategory }) })
    fireEvent.click(screen.getByRole('switch', { name: 'Grenades' }))
    expect(toggleCategory).toHaveBeenCalledTimes(1)
    expect(toggleCategory).toHaveBeenCalledWith('grenade')
  })
})

describe('ReplaySettingsDrawer — effets d événement', () => {
  it('les deux effets ont leur bascule, chacune avec son état', () => {
    renderDrawer({ showShotFx: true, showKillFx: false })
    expect(screen.getByRole('switch', { name: 'Effets de tirs' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    expect(screen.getByRole('switch', { name: 'Effets de mort' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
  })

  it('chaque bascule appelle SON callback, jamais celui de l autre effet', () => {
    const { onToggleShotFx, onToggleKillFx } = renderDrawer()
    fireEvent.click(screen.getByRole('switch', { name: 'Effets de mort' }))
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
  // LE PANNEAU A QUITTÉ LE CADRE le 2026-09-02 : il n'est plus `absolute right-0` dans la carte
  // mais un portail `fixed`, placé par `useAnchoredPanel`. Ce que le test garde n'est donc plus
  // sa position — elle est calculée, pas déclarée — mais les DEUX invariants qui survivent au
  // changement : il flotte hors du flux, et il passe AU-DESSUS de la légende si un écran étroit
  // les rapproche (un panneau ouvert prime une légende ; ce n'est pas une collision).
  it('le panneau flotte hors du flux et prime la légende, qui garde son coin bas-GAUCHE', () => {
    const panel = renderDrawer().getByRole('region', { name: 'Réglages' })
    expect(panel.className).toContain('fixed')
    expect(panel.className).toContain('z-50')
    // Hors du cadre = hors du sous-arbre de la carte : il se rend sur `body`.
    expect(panel.closest('body')).toBeTruthy()
    const legend = render(<ReplayHeatmapLegend locale="fr" mode="presence" />)
    const box = legend.container.firstElementChild as HTMLElement
    expect(box.className).toContain('left-3')
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
    fireEvent.pointerDown(screen.getByRole('switch', { name: 'Visée' }))
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
