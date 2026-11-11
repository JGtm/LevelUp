/**
 * ReplayViewPanel.test.tsx — LE PANNEAU DE LA VUE : les préréglages en tête, puis chaque section
 * n'apparaît que si elle a quelque chose à commander, chaque bascule appelle EXACTEMENT son
 * callback (jamais une voisine), et le panneau se ferme par son bouton comme par Échap (même
 * contrat que les autres panneaux du dépôt — AlertDialog, AssetDrawer).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplayHeatmapLegend } from '../ui/ReplayHeatmapLegend'
import {
  ReplayViewPanel,
  type ReplayGroundWeaponControls,
  type ReplayHeatmapControls,
  type ReplayModeObjectivesControls,
  type ReplayPlacementControls,
  type ReplayFlagControls,
  type ReplayWeaponPadControls,
} from './ReplayViewPanel'
import type { ReplaySound } from '../sound/useReplaySound'

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

function makeGroundWeapons(
  over: Partial<ReplayGroundWeaponControls> = {},
): ReplayGroundWeaponControls {
  return {
    available: true,
    show: true,
    onToggle: vi.fn(),
    showSpecialOnly: false,
    onToggleSpecialOnly: vi.fn(),
    showStarting: false,
    onToggleStarting: vi.fn(),
    ...over,
  }
}

function makeFlagCarries(over: Partial<ReplayFlagControls> = {}): ReplayFlagControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

function makeModeObjectives(
  over: Partial<ReplayModeObjectivesControls> = {},
): ReplayModeObjectivesControls {
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
    seek: vi.fn(),
    setTransportPlaying: vi.fn(),
    endMatch: vi.fn(),
    intro: vi.fn(),
    recordingTrack: () => null,
  exportTrack: () => ({ timeline: [], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
    ...over,
  }
}

function renderPanel(over: Partial<Parameters<typeof ReplayViewPanel>[0]> = {}) {
  const onClose = vi.fn()
  const onSelect = vi.fn()
  const onToggleAim = vi.fn()
  const onToggleZones = vi.fn()
  const onToggleTrail = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleShotFx = vi.fn()
  const onToggleKillFx = vi.fn()
  const onSetMarkerColors = vi.fn()
  const utils = render(
    <ReplayViewPanel
      locale="fr"
      onClose={onClose}
      view={{ active: 'complet', onSelect }}
      showAim
      onToggleAim={onToggleAim}
      showZones
      onToggleZones={onToggleZones}
      showTrail
      onToggleTrail={onToggleTrail}
      zonesAvailable
      placements={makePlacements()}
      weaponPads={makeWeaponPads()}
      groundWeapons={makeGroundWeapons()}
      modeObjectives={makeModeObjectives()}
      flagCarries={makeFlagCarries()}
      vipCrown={makeFlagCarries()}
      skullCarrier={makeFlagCarries()}
      bombCarrier={makeFlagCarries()}
      vehicles={makeWeaponPads()}
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
    ...utils, onClose, onSelect, onToggleAim, onToggleZones, onToggleTrail, onSetSpeed,
    onToggleShotFx, onToggleKillFx, onSetMarkerColors,
  }
}

describe('ReplayViewPanel — les deux formes de commande (2026-08-29)', () => {
  // UN OUI/NON EST UN INTERRUPTEUR, UN CHOIX EXCLUSIF RESTE UN BOUTON PRESSÉ. La demande
  // utilisateur (« je préfère un toggle plutôt que des boutons ») porte sur les réglages
  // oui/non ; un interrupteur sur « Par équipe » promettrait un « tout éteint » que le
  // réglage n'accepte pas. Ce test épingle la frontière, qui se perdrait à la relecture.
  it('les calques et les catégories de son sont des interrupteurs', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Visée', 'Traînée', 'Zones', 'Carte de chaleur', 'Armes']) {
      expect(screen.getByRole('switch', { name: nom })).toBeTruthy()
    }
  })

  it('les choix exclusifs ne sont JAMAIS des interrupteurs', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Présence', 'Éliminations', 'Par équipe', 'Par joueur']) {
      expect(screen.queryByRole('switch', { name: nom })).toBeNull()
    }
  })

  // LES DEUX AXES DE LA CHALEUR SONT PASSÉS EN SEGMENTÉ le 2026-09-02 : un choix parmi N
  // s'annonce en radiogroup/radio, pas comme une grappe de boutons pressés. C'est la forme
  // qui porte la contrainte — des options empilées laissaient croire qu'on pouvait en allumer
  // deux. Les couleurs de marqueur, elles, restent des boutons : leur liste n'a pas bougé.
  it('les axes de la chaleur sont un choix radio, pas une grappe de boutons', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true }) })
    for (const nom of ['Présence', 'Éliminations']) {
      expect(screen.getByRole('radio', { name: nom })).toBeTruthy()
    }
    expect(screen.getAllByRole('radiogroup').length).toBeGreaterThanOrEqual(2)
  })
})

describe('ReplayViewPanel — calques', () => {
  it('bascule Visée : reflète showAim et appelle onToggleAim au clic, jamais onToggleZones', () => {
    const { onToggleAim, onToggleZones } = renderPanel({ showAim: true })
    const btn = screen.getByRole('switch', { name: 'Visée' })
    expect(btn).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(btn)
    expect(onToggleAim).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })

  // LE CALQUE DES NOMS N'A PLUS DE BASCULE (2026-09-02) : il est toujours allumé. Le test
  // garde l'ABSENCE — sans lui, une revue future la réintroduirait par symétrie avec les
  // autres calques, et le panneau reprendrait la ligne qu'on vient de lui faire rendre.
  it('aucune bascule « Noms » : le calque est toujours allumé', () => {
    renderPanel({ zonesAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Noms' })).toBeNull()
  })

  // V1 (2026-08-18) : la TRAÎNÉE devient un calque comme les autres — toujours proposée (elle
  // n'a aucune condition de disponibilité : une vie a toujours un passé), allumée par défaut.
  it('bascule Traînée : toujours proposée, reflète showTrail, appelle onToggleTrail au clic', () => {
    const { onToggleTrail, onToggleAim } = renderPanel({ showTrail: false, zonesAvailable: false })
    const btn = screen.getByRole('switch', { name: 'Traînée' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleTrail).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('bouton Zones absent quand la carte n a pas de zones nommées', () => {
    renderPanel({ zonesAvailable: false })
    expect(screen.queryByRole('switch', { name: 'Zones' })).toBeNull()
  })

  it('bouton Zones présent, reflète showZones, appelle onToggleZones au clic', () => {
    const { onToggleZones } = renderPanel({ zonesAvailable: true, showZones: false })
    const btn = screen.getByRole('switch', { name: 'Zones' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleZones).toHaveBeenCalledTimes(1)
  })
})

describe('ReplayViewPanel — poses d équipement', () => {
  it('film sans pose dessinable : ni le calque ni les objets non identifiés', () => {
    renderPanel({ placements: makePlacements({ available: false }) })
    expect(screen.queryByRole('switch', { name: 'Équipements posés' })).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Équipements posés : reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleAim } = renderPanel({ placements: makePlacements({ show: false, onToggle }) })
    const btn = screen.getByRole('switch', { name: 'Équipements posés' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleAim).not.toHaveBeenCalled()
  })

  it('calque éteint : la bascule des objets non identifiés ne commanderait rien, elle est absente', () => {
    renderPanel({ placements: makePlacements({ show: false }) })
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('toutes les poses nommées : rien à révéler, la bascule n est pas proposée', () => {
    renderPanel({ placements: makePlacements({ unnamedAvailable: false }) })
    expect(screen.getByRole('switch', { name: 'Équipements posés' })).toBeTruthy()
    expect(screen.queryByRole('switch', { name: 'Objets non identifiés' })).toBeNull()
  })

  it('bascule Objets non identifiés : éteinte par défaut, appelle SON callback', () => {
    const onToggleUnnamed = vi.fn()
    const onToggle = vi.fn()
    renderPanel({ placements: makePlacements({ onToggle, onToggleUnnamed }) })
    const btn = screen.getByRole('switch', { name: 'Objets non identifiés' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggleUnnamed).toHaveBeenCalledTimes(1)
    expect(onToggle).not.toHaveBeenCalled()
  })
})

describe('ReplayViewPanel — emplacements d arme', () => {
  it('film sans socle publié : pas de bascule (elle ne commanderait rien)', () => {
    renderPanel({ weaponPads: makeWeaponPads({ available: false }) })
    expect(screen.queryByRole('switch', { name: "Emplacements d'arme" })).toBeNull()
  })

  it('bascule Emplacements d arme : reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleAim } = renderPanel({
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

describe('ReplayViewPanel — objectifs du mode (2026-09-18)', () => {
  /** Les objets PORTÉS tous indisponibles : ce qu'un film sans drapeau, VIP, crâne ni bombe publie. */
  const sansObjetsPortes = () => ({
    flagCarries: makeFlagCarries({ available: false }),
    vipCrown: makeFlagCarries({ available: false }),
    skullCarrier: makeFlagCarries({ available: false }),
    bombCarrier: makeFlagCarries({ available: false }),
  })

  it('bascule Objectifs du mode : reflète son état et appelle SON callback, jamais les drapeaux', () => {
    const onToggle = vi.fn()
    const onToggleFlags = vi.fn()
    renderPanel({
      modeObjectives: makeModeObjectives({ show: false, onToggle }),
      flagCarries: makeFlagCarries({ onToggle: onToggleFlags }),
    })
    const btn = screen.getByRole('switch', { name: 'Objectifs du mode' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleFlags).not.toHaveBeenCalled()
  })

  it('elle ouvre le groupe Objectifs, EN PREMIER, avant les drapeaux', () => {
    renderPanel()
    const titre = screen.getByText('Objectifs')
    const groupe = titre.parentElement
    expect(groupe).not.toBeNull()
    const bascules = Array.from(groupe!.querySelectorAll('[role="switch"]')).map((b) =>
      b.getAttribute('aria-label') ?? b.textContent,
    )
    expect(bascules[0]).toContain('Objectifs du mode')
    expect(bascules.findIndex((n) => n?.includes('Drapeaux'))).toBeGreaterThan(0)
  })

  it('le groupe Objectifs existe même sans drapeau, VIP, crâne ni bombe : le terrain de l enjeu suffit', () => {
    renderPanel(sansObjetsPortes())
    expect(screen.getByText('Objectifs')).toBeTruthy()
    expect(screen.getByRole('switch', { name: 'Objectifs du mode' })).toBeTruthy()
    expect(screen.queryByRole('switch', { name: 'Drapeaux' })).toBeNull()
  })

  it('film sans objectif de mode ni objet porté : ni la bascule, ni le titre du groupe', () => {
    renderPanel({ ...sansObjetsPortes(), modeObjectives: makeModeObjectives({ available: false }) })
    expect(screen.queryByRole('switch', { name: 'Objectifs du mode' })).toBeNull()
    expect(screen.queryByText('Objectifs')).toBeNull()
  })
})

describe('ReplayViewPanel — carte de chaleur', () => {
  it('calque éteint : la bascule est là, le choix de lecture ne l est pas', () => {
    renderPanel({ heatmap: makeHeatmap({ show: false }) })
    expect(screen.getByRole('switch', { name: 'Carte de chaleur' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
    expect(screen.queryByRole('button', { name: 'Présence' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Éliminations' })).toBeNull()
  })

  it('calque allumé : les deux lectures, la courante pressée (choix exclusif)', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true, mode: 'kills' }) })
    expect(screen.getByRole('radio', { name: 'Présence' })).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByRole('radio', { name: 'Éliminations' })).toHaveAttribute('aria-checked', 'true')
  })

  it('cliquer une lecture appelle onSetMode avec SA clé, jamais la bascule du calque', () => {
    const onSetMode = vi.fn()
    const onToggle = vi.fn()
    renderPanel({ heatmap: makeHeatmap({ show: true, onSetMode, onToggle }) })
    fireEvent.click(screen.getByRole('radio', { name: 'Éliminations' }))
    expect(onSetMode).toHaveBeenCalledTimes(1)
    expect(onSetMode).toHaveBeenCalledWith('kills')
    expect(onToggle).not.toHaveBeenCalled()
  })

  it('aucune mort localisée : la lecture éliminations n est pas proposée', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true, killsAvailable: false }) })
    expect(screen.queryByRole('button', { name: 'Éliminations' })).toBeNull()
    // Et le choix disparaît entièrement : une seule lecture ne se choisit pas.
    expect(screen.queryByRole('button', { name: 'Présence' })).toBeNull()
  })

  it('la bascule appelle onToggle, jamais un calque voisin', () => {
    const onToggle = vi.fn()
    const { onToggleZones } = renderPanel({ heatmap: makeHeatmap({ onToggle }) })
    fireEvent.click(screen.getByRole('switch', { name: 'Carte de chaleur' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })
})

describe('ReplayViewPanel — couleur des points', () => {
  it('propose les deux lectures, la courante pressée (choix exclusif)', () => {
    renderPanel({ markerColors: 'player' })
    expect(screen.getByRole('button', { name: 'Par équipe' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Par joueur' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('cliquer une lecture appelle onSetMarkerColors avec SON mode', () => {
    const { onSetMarkerColors } = renderPanel({ markerColors: 'team' })
    fireEvent.click(screen.getByRole('button', { name: 'Par joueur' }))
    expect(onSetMarkerColors).toHaveBeenCalledWith('player')
  })
})

describe('ReplayViewPanel — son (le filtre par catégorie seul : l’interrupteur vit à la barre de lecture)', () => {
  it('aucune commande de son ni de catégorie quand le match n a aucun son', () => {
    renderPanel({ sound: makeSound({ available: false }) })
    expect(screen.queryByText('Sons par catégorie')).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Armes' })).toBeNull()
    expect(screen.queryByRole('switch', { name: 'Son' })).toBeNull()
  })

  it('l’interrupteur du son n’est PLUS dans le panneau, même quand le son est disponible', () => {
    renderPanel()
    expect(screen.queryByRole('switch', { name: 'Son' })).toBeNull()
    expect(screen.getByText('Sons par catégorie')).toBeTruthy()
  })

  it('les quatre catégories sont affichées, chacune avec son état', () => {
    renderPanel({
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
    renderPanel({ sound: makeSound({ toggleCategory }) })
    fireEvent.click(screen.getByRole('switch', { name: 'Grenades' }))
    expect(toggleCategory).toHaveBeenCalledTimes(1)
    expect(toggleCategory).toHaveBeenCalledWith('grenade')
  })
})

describe('ReplayViewPanel — effets d événement', () => {
  it('les deux effets ont leur bascule, chacune avec son état', () => {
    renderPanel({ showShotFx: true, showKillFx: false })
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
    const { onToggleShotFx, onToggleKillFx } = renderPanel()
    fireEvent.click(screen.getByRole('switch', { name: 'Effets de mort' }))
    expect(onToggleKillFx).toHaveBeenCalledTimes(1)
    expect(onToggleShotFx).not.toHaveBeenCalled()
  })

  it('le (i) des effets de tir dit ce qui est dessiné, jamais une couverture', () => {
    // Le (i) demandé le 16/08 dit la règle du dessin (un tir qui applique un dégât) ; depuis le
    // 2026-10-09, il ne parle plus de couverture (aucun inconnu à l'écran).
    renderPanel()
    const mark = screen.getByRole('img', { name: /tir s'affiche lorsqu'il applique un dégât/i })
    expect(mark.getAttribute('title') ?? '').not.toMatch(/couverture/i)
  })
})

describe('ReplayViewPanel — cohabitation avec la légende de la carte de chaleur', () => {
  // LE PANNEAU A QUITTÉ LE CADRE le 2026-09-02 : il n'est plus `absolute right-0` dans la carte
  // mais un portail `fixed`, placé par `useAnchoredPanel`. Ce que le test garde n'est donc plus
  // sa position — elle est calculée, pas déclarée — mais les DEUX invariants qui survivent au
  // changement : il flotte hors du flux, et il passe AU-DESSUS de la légende si un écran étroit
  // les rapproche (un panneau ouvert prime une légende ; ce n'est pas une collision).
  it('le panneau flotte hors du flux et prime la légende, qui garde son coin bas-GAUCHE', () => {
    const panel = renderPanel().getByRole('region', { name: 'Vue' })
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

describe('ReplayViewPanel — préréglages de vue', () => {
  it('les quatre préréglages viennent EN PREMIER, avant les calques', () => {
    renderPanel()
    const titres = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)
    expect(titres[0]).toBe('Préréglages')
    expect(titres.indexOf('Calques')).toBeGreaterThan(0)
    for (const nom of ['Complet', 'Combat', 'Déplacements', 'Objectif']) {
      expect(screen.getByRole('button', { name: nom })).toBeTruthy()
    }
  })

  it('le préréglage actif est pressé, les autres non', () => {
    renderPanel({ view: { active: 'combat', onSelect: vi.fn() } })
    expect(screen.getByRole('button', { name: 'Combat' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Complet' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('cliquer un préréglage appelle onSelect avec SON identifiant', () => {
    const { onSelect } = renderPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Déplacements' }))
    expect(onSelect).toHaveBeenCalledTimes(1)
    expect(onSelect).toHaveBeenCalledWith('deplacements')
  })

  it('Personnalisé : aucun préréglage pressé, et une phrase dit pourquoi', () => {
    renderPanel({ view: { active: 'personnalise', onSelect: vi.fn() } })
    for (const nom of ['Complet', 'Combat', 'Déplacements', 'Objectif']) {
      expect(screen.getByRole('button', { name: nom })).toHaveAttribute('aria-pressed', 'false')
    }
    expect(screen.getByText(/ne correspondent à aucun préréglage/)).toBeTruthy()
  })

  it('un préréglage actif ne montre pas la phrase de Personnalisé', () => {
    renderPanel()
    expect(screen.queryByText(/ne correspondent à aucun préréglage/)).toBeNull()
  })

  it('en anglais : Full, Combat, Movement, Objective, et Custom', () => {
    renderPanel({ locale: 'en', view: { active: 'personnalise', onSelect: vi.fn() } })
    for (const nom of ['Full', 'Combat', 'Movement', 'Objective']) {
      expect(screen.getByRole('button', { name: nom })).toBeTruthy()
    }
    expect(screen.getByText('Custom')).toBeTruthy()
  })
})

describe('ReplayViewPanel — rien ne disparaît du tiroir d’avant', () => {
  it('les effets, la chaleur, la couleur des points et le son sont toujours là', () => {
    renderPanel({ heatmap: makeHeatmap({ show: true }) })
    expect(screen.getByRole('switch', { name: 'Effets de tirs' })).toBeTruthy()
    expect(screen.getByRole('switch', { name: 'Carte de chaleur' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Par équipe' })).toBeTruthy()
    expect(screen.getByRole('switch', { name: 'Armes' })).toBeTruthy()
  })
})

describe('ReplayViewPanel — fermeture et focus', () => {
  it('le bouton de fermeture appelle onClose', () => {
    const { onClose } = renderPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer la vue' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Échap appelle onClose', () => {
    const { onClose } = renderPanel()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic DEHORS ferme le panneau (il recouvre la carte : on doit pouvoir en sortir)', () => {
    const { onClose } = renderPanel()
    fireEvent.pointerDown(document.body)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic DEDANS ne ferme rien — sinon régler quoi que ce soit refermerait le panneau', () => {
    const { onClose } = renderPanel()
    fireEvent.pointerDown(screen.getByRole('switch', { name: 'Visée' }))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('le DÉCLENCHEUR est exclu du clic dehors : sinon il fermerait puis rouvrirait', () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    const { onClose } = renderPanel({ triggerRef: { current: trigger } })
    fireEvent.pointerDown(trigger)
    expect(onClose).not.toHaveBeenCalled()
    trigger.remove()
  })

  it('le focus entre au panneau à l ouverture (il recouvre les commandes de la carte)', () => {
    renderPanel()
    expect(document.activeElement).toBe(screen.getByRole('region', { name: 'Vue' }))
  })
})
