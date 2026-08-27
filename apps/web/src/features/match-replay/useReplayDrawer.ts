/**
 * useReplayDrawer — LE TIROIR DE RÉGLAGES, assemblé en un objet.
 *
 * QUATORZIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE TAILLE (`placementFamily.guard.test.ts`).
 * Le montage du tiroir occupait une cinquantaine de lignes du canvas — trente réglages
 * recopiés un par un depuis `useReplaySettings`, plus les quatre disponibilités lues des
 * calques. Aucune de ces lignes ne décide de quoi que ce soit : elles TRANSPORTENT. Le canvas
 * garde le DESSIN ; ce hook porte le formulaire.
 *
 * IL NE RÉINVENTE AUCUN RÉGLAGE, et c'est la condition pour que l'extraction soit sûre : les
 * états et leur persistance restent dans `useReplaySettings` (calques, effets, vitesse) et
 * `useReplaySound` (son et catégories). Ce hook ne fait que grouper — même patron que
 * `ReplaySound`, `ReplayCapture` et `timeline` : un objet que le canvas repasse tel quel.
 *
 * LES QUATRE « available » NE SONT PAS DÉCORATIFS : c'est la règle du dépôt « pas de commande
 * quand il n'y a rien à commander » (un film sans socle publié ne montre pas la bascule des
 * socles). Ils viennent des calques eux-mêmes, seuls à savoir ce que le film porte.
 */
import { useMemo, type ComponentProps, type RefObject } from 'react'

import type { ReplayLocale } from './i18n'
import type { ReplaySettingsDrawer } from './ReplaySettingsDrawer'
import type { ReplaySettings } from './useReplaySettings'
import type { ReplaySound } from './useReplaySound'

/** Ce que le canvas prête au tiroir : les réglages, ce que le film porte, et les sorties. */
export interface ReplayDrawerOptions {
  settings: ReplaySettings
  sound: ReplaySound
  /** Ce que le film porte réellement, calque par calque (cf. l'en-tête). */
  available: {
    zones: boolean
    placements: { drawable: boolean; unnamed: boolean; dropped: boolean }
    weaponPads: boolean
    flagCarries: boolean
    vipCrown: boolean
  }
  /** La lecture de la carte de chaleur RÉELLEMENT servie, et si les morts sont localisables. */
  heat: { mode: ReplayHeatmapMode; killsAvailable: boolean }
  locale: ReplayLocale
  onClose: () => void
  /** Le bouton qui ouvre le tiroir : exclu du « clic dehors », il reprend le focus. */
  triggerRef: RefObject<HTMLButtonElement | null>
}

/** La lecture de la carte de chaleur, telle que `useReplayHeatmap` la publie. */
type ReplayHeatmapMode = ComponentProps<typeof ReplaySettingsDrawer>['heatmap']['mode']

/** L'objet unique que le canvas repasse au tiroir. */
export type ReplayDrawer = ComponentProps<typeof ReplaySettingsDrawer>

export function useReplayDrawer(o: ReplayDrawerOptions): ReplayDrawer {
  const { settings: s, sound, available, heat, locale, onClose, triggerRef } = o
  // MÉMOÏSÉ SUR SES SOURCES : le tiroir se remonte à chaque rendu du canvas sinon, c'est-à-dire
  // à chaque geste de souris sur la carte (le survol des poses passe par un état React).
  return useMemo(
    () => ({
      locale,
      onClose,
      showAim: s.showAim,
      onToggleAim: s.toggleAim,
      showZones: s.showZones,
      onToggleZones: s.toggleZones,
      showNames: s.showNames,
      onToggleNames: s.toggleNames,
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
      heatmap: {
        show: s.showHeatmap,
        onToggle: s.toggleHeatmap,
        // LA LECTURE SERVIE, pas celle demandée : sans mort localisable, la carte retombe sur
        // « présence » et le tiroir doit montrer CE QU'ON VOIT (cf. useReplayHeatmap).
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
      triggerRef,
    }),
    [s, sound, available, heat, locale, onClose, triggerRef],
  )
}
