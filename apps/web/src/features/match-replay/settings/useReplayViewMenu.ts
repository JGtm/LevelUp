/**
 * useReplayViewMenu — LE BOUTON « VUE » et son panneau, assemblés en un objet.
 *
 * Le montage du panneau occupait une cinquantaine de lignes du canvas — trente réglages recopiés
 * un par un depuis `useReplaySettings`, plus les disponibilités lues des calques. Aucune de ces
 * lignes ne décide de quoi que ce soit : elles TRANSPORTENT. Le canvas garde le DESSIN ; ce hook
 * porte le formulaire, et il porte aussi l'état d'ouverture et la VUE (les préréglages).
 *
 * IL NE RÉINVENTE AUCUN RÉGLAGE : les états et leur persistance restent dans `useReplaySettings`
 * (calques, effets, chaleur, couleur des points) et `useReplaySound` (son et catégories). Ce hook
 * ne fait que grouper, et brancher la table des préréglages (`viewPresetLogic`) sur les bascules
 * réelles des calques.
 *
 * LA VUE ACTIVE N'EST PAS UN ÉTAT : elle se déduit des calques (`activeView`), si bien qu'une
 * bascule à la main la fait passer en « Personnalisé » sans rien écrire, et que les clés `replay-*`
 * d'avant ce lot restent lues telles quelles. Seul le DERNIER préréglage choisi est retenu
 * (`replay-view-preset`) : il départage deux préréglages égaux sur les calques d'un match.
 *
 * LES « available » NE SONT PAS DÉCORATIFS : c'est la règle du dépôt « pas de commande quand il
 * n'y a rien à commander » (un film sans socle publié ne montre pas la bascule des socles), et
 * c'est ce qui interdit à un préréglage d'allumer un calque que le match ne porte pas.
 *
 * LE PANNEAU EST FERMÉ PAR DÉFAUT et s'ouvre par le bouton « Vue » de la carte. Les réglages qu'il
 * porte SURVIVENT à la page (persistés) ; seule l'ouverture repart de zéro.
 */
import { useCallback, useRef, useState, type ComponentProps, type RefObject } from 'react'

import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayViewPanel } from './ReplayViewPanel'
import { persistViewPreset, readViewPreset } from './replayPreferences'
import type { ReplaySettings } from './useReplaySettings'
import {
  activeView,
  layerAvailability,
  layersToToggle,
  type FilmLayerAvailability,
  type LayerFlags,
  type LayerKey,
  type ViewId,
  type ViewPresetId,
} from './viewPresetLogic'
import type { ReplaySound } from '../sound/useReplaySound'

/** Ce que le canvas prête au panneau : les réglages, ce que le film porte, et les sorties. */
export interface ReplayViewMenuOptions {
  settings: ReplaySettings
  sound: ReplaySound
  /** Ce que le film porte réellement, calque par calque (cf. l'en-tête). */
  available: FilmLayerAvailability
  /** La lecture de la carte de chaleur RÉELLEMENT servie, et si les morts sont localisables. */
  heat: { mode: ReplayHeatmapMode; killsAvailable: boolean }
  locale: ReplayLocale
}

/** La lecture de la carte de chaleur, telle que `useReplayHeatmap` la publie. */
type ReplayHeatmapMode = ComponentProps<typeof ReplayViewPanel>['heatmap']['mode']

/** Les props du panneau, à répandre telles quelles quand il est ouvert. */
export type ReplayViewPanelProps = ComponentProps<typeof ReplayViewPanel>

/** Ce que le canvas reçoit : l'état du panneau, ce qui l'ouvre, et de quoi le peindre. */
export interface ReplayViewMenuState {
  open: boolean
  /** Bascule d'ouverture, posée sur le bouton « Vue ». */
  toggle: () => void
  /** Le bouton qui ouvre le panneau : exclu du « clic dehors », il reprend le focus. */
  buttonRef: RefObject<HTMLButtonElement | null>
  /** La vue active : un préréglage, ou « Personnalisé ». Le bouton la nomme. */
  active: ViewId
  /** Les props du panneau, SANS son déclencheur : `ReplayViewMenu` y pose `buttonRef`. */
  panel: ReplayViewPanelProps
}

/** L'état des treize calques, lu des réglages. */
function layerFlagsOf(s: ReplaySettings): LayerFlags {
  return {
    aim: s.showAim,
    trail: s.showTrail,
    zones: s.showZones,
    weaponPads: s.showWeaponPads,
    groundWeapons: s.showGroundWeapons,
    vehicles: s.showVehicles,
    placements: s.showPlacements,
    droppedPlacements: s.showDroppedPlacements,
    modeObjectives: s.showModeObjectives,
    flagCarries: s.showFlagCarries,
    vipCrown: s.showVipCrown,
    skullCarrier: s.showSkullCarrier,
    bombCarrier: s.showBombCarrier,
  }
}

/** La bascule de chaque calque : celle-là même que le panneau pose sur son interrupteur. */
function layerTogglesOf(s: ReplaySettings): Record<LayerKey, () => void> {
  return {
    aim: s.toggleAim,
    trail: s.toggleTrail,
    zones: s.toggleZones,
    weaponPads: s.toggleWeaponPads,
    groundWeapons: s.toggleGroundWeapons,
    vehicles: s.toggleVehicles,
    placements: s.togglePlacements,
    droppedPlacements: s.toggleDroppedPlacements,
    modeObjectives: s.toggleModeObjectives,
    flagCarries: s.toggleFlagCarries,
    vipCrown: s.toggleVipCrown,
    skullCarrier: s.toggleSkullCarrier,
    bombCarrier: s.toggleBombCarrier,
  }
}

/** Les entrées de l'assemblage du panneau (tout ce que le hook lui prête, hors ouverture). */
interface PanelInput extends ReplayViewMenuOptions {
  onClose: () => void
  view: ReplayViewPanelProps['view']
}

/**
 * Les props du panneau, calque par calque. Aucune de ces lignes ne décide de quoi que ce soit :
 * elles TRANSPORTENT les états et les bascules de `useReplaySettings` vers les commandes.
 */
function panelPropsOf(o: PanelInput): ReplayViewPanelProps {
  const { settings: s, sound, available, heat, locale, onClose, view } = o
  return {
    locale,
    onClose,
    view,
    showAim: s.showAim,
    onToggleAim: s.toggleAim,
    showZones: s.showZones,
    onToggleZones: s.toggleZones,
    showTrail: s.showTrail,
    onToggleTrail: s.toggleTrail,
    zonesAvailable: available.zones,
    placements: {
      available: available.placements.drawable,
      show: s.showPlacements,
      onToggle: s.togglePlacements,
      unnamedAvailable: available.placements.unnamed,
      showUnnamed: s.showUnnamedPlacements,
      onToggleUnnamed: s.toggleUnnamedPlacements,
      droppedAvailable: available.placements.dropped,
      showDropped: s.showDroppedPlacements,
      onToggleDropped: s.toggleDroppedPlacements,
    },
    weaponPads: {
      available: available.weaponPads,
      show: s.showWeaponPads,
      onToggle: s.toggleWeaponPads,
    },
    groundWeapons: {
      available: available.groundWeapons,
      show: s.showGroundWeapons,
      onToggle: s.toggleGroundWeapons,
      showSpecialOnly: s.showGroundWeaponsSpecialOnly,
      onToggleSpecialOnly: s.toggleGroundWeaponsSpecialOnly,
      showStarting: s.showGroundWeaponsStarting,
      onToggleStarting: s.toggleGroundWeaponsStarting,
    },
    modeObjectives: {
      available: available.modeObjectives,
      show: s.showModeObjectives,
      onToggle: s.toggleModeObjectives,
    },
    flagCarries: {
      available: available.flagCarries,
      show: s.showFlagCarries,
      onToggle: s.toggleFlagCarries,
    },
    vipCrown: {
      available: available.vipCrown,
      show: s.showVipCrown,
      onToggle: s.toggleVipCrown,
    },
    skullCarrier: {
      available: available.skullCarrier,
      show: s.showSkullCarrier,
      onToggle: s.toggleSkullCarrier,
    },
    bombCarrier: {
      available: available.bombCarrier,
      show: s.showBombCarrier,
      onToggle: s.toggleBombCarrier,
    },
    vehicles: {
      available: available.vehicles,
      show: s.showVehicles,
      onToggle: s.toggleVehicles,
    },
    wallPlan: {
      available: available.wallPlan,
      show: s.showWallPlan,
      onToggle: s.toggleWallPlan,
    },
    heatmap: {
      show: s.showHeatmap,
      onToggle: s.toggleHeatmap,
      // LA LECTURE SERVIE, pas celle demandée : sans mort localisable, la carte retombe sur
      // « présence » et le panneau doit montrer CE QU'ON VOIT (cf. useReplayHeatmap).
      mode: heat.mode,
      onSetMode: s.setHeatmapMode,
      span: s.heatmapSpan,
      onSetSpan: s.setHeatmapSpan,
      killsAvailable: heat.killsAvailable,
    },
    showShotFx: s.showShotFx,
    onToggleShotFx: s.toggleShotFx,
    showKillFx: s.showKillFx,
    onToggleKillFx: s.toggleKillFx,
    sound,
    markerColors: s.markerColors,
    onSetMarkerColors: s.setMarkerColors,
  }
}

export function useReplayViewMenu(o: ReplayViewMenuOptions): ReplayViewMenuState {
  const { settings: s, available } = o
  const [open, setOpen] = useState(false)
  const [remembered, setRemembered] = useState(readViewPreset)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const toggle = useCallback(() => setOpen((v) => !v), [])
  // LA FERMETURE REPREND LE FOCUS, et ce n'est pas un ornement : sans cela le focus retomberait
  // au document et la navigation au clavier repartirait du haut de la page.
  const onClose = useCallback(() => {
    setOpen(false)
    buttonRef.current?.focus({ preventScroll: true })
  }, [])
  const active = activeView(layerFlagsOf(s), layerAvailability(available), remembered)
  // CHOISIR UN PRÉRÉGLAGE = basculer exactement les calques disponibles qui diffèrent, puis
  // retenir le choix. Les effets, la chaleur, la couleur des points et le son ne sont pas touchés.
  const onSelect = (id: ViewPresetId) => {
    const toggles = layerTogglesOf(s)
    for (const key of layersToToggle(id, layerFlagsOf(s), layerAvailability(available))) {
      toggles[key]()
    }
    setRemembered(id)
    persistViewPreset(id)
  }
  // L'OBJET EST REFABRIQUÉ À CHAQUE RENDU : `useReplaySettings` rend un objet neuf à chaque fois,
  // et le panneau n'est de toute façon monté que lorsqu'il est OUVERT.
  const panel = panelPropsOf({ ...o, onClose, view: { active, onSelect } })
  return { open, toggle, buttonRef, active, panel }
}
