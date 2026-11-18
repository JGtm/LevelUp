/**
 * replayFrameTypes — CE QUE LA PAGE DU REJEU REMET AUX CADRES. Types seuls.
 *
 * Deux paquets, tenus par la page et jamais reconstruits par les cadres :
 *  - `ReplayFrameChrome` : ce qu'il faut pour dessiner l'en-tête, disponible dans tous les états
 *    de chargement (le titre, le lien vers la fiche, les gestes de cadre) ;
 *  - `ReplayFrameView` : ce qu'il faut pour montrer le rejeu lui-même, disponible une fois le
 *    document lu (la toile, ses surcouches, la colonne des équipes et du fil).
 *
 * Un cadre choisit la DISPOSITION de ces deux paquets ; il n'en lit rien dans un store ni une requête.
 */
import type { ReactNode, Ref } from 'react'

import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { MatchViewHeader } from '@/lib/api/types'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import type { FeedZoneOf } from '../../hooks/useFeedZones'
import type { PlayerPanel } from '../../hooks/usePlayerPanel'
import type { ReplayFrameControl } from '../../hooks/useReplayFrame'
import type { ReplayFullscreen } from '../../hooks/useReplayFullscreen'
import type { ReplayMapBackgroundLayer } from '../../hooks/useReplayView'
import type { ReplayViewpoint } from '../../hooks/useReplayViewpoint'
import type { ReplayLocale } from '../../i18n/i18n'
import type { WeaponIconsByKey } from '../../model/killSourceIcons'
import type { PlaybackStore } from '../../model/playbackStore'
import type { ReplayModel } from '../../model/replayModel'
import type { EndMatchSoundSpec } from '../../sound/endMatchSound'

/** Les paramètres de la route du rejeu, tels que la cible « fiche du match » les attend. */
export interface ReplayRouteParams {
  lang?: string
  titleSlug: string
  playerSlug: string
  matchId: string
}

/** Ce que l'en-tête d'un cadre lit : présent dès le premier rendu, quel que soit l'état de chargement. */
export interface ReplayFrameChrome {
  locale: ReplayLocale
  params: ReplayRouteParams
  /** Le libellé du match (mode et carte) ; vide tant que l'en-tête du match n'est pas lu. */
  matchLabel: string
  /** L'en-tête du match (date, playlist) ; absent tant que le contexte du match est en vol. */
  header: MatchViewHeader | undefined
  /** Le document du rejeu, pour le badge de version de l'administrateur ; absent hors état prêt. */
  doc: ReplayDocumentReady | undefined
  frameControl: ReplayFrameControl
  fullscreen: ReplayFullscreen
}

/** Ce que la toile, ses surcouches et la colonne des équipes et du fil lisent : le rejeu prêt. */
export interface ReplayFrameView {
  doc: ReplayDocumentReady
  locale: ReplayLocale
  titleSlug: string
  playerSlug: string
  matchId: string
  model: ReplayModel
  header: MatchViewHeader | undefined
  viewpoint: ReplayViewpoint
  playerPanel: PlayerPanel
  playbackStore: PlaybackStore
  /** L'image de lecture publiée par la toile (cadence de publication, pas celle de l'écran). */
  playFrame: number
  /** L'instant du film correspondant à `playFrame`, en millisecondes. */
  nowMs: number
  mapBackground: ReplayMapBackgroundLayer | null
  callouts: CalloutZoneReady[]
  endMatchSound: EndMatchSoundSpec | null
  openAtFrame: number | null
  /** Le lien tactique demandait un instant que ce rejeu ne peut pas caler. */
  showMissingOriginNotice: boolean
  feedZoneOf: FeedZoneOf | undefined
  weaponIcons: WeaponIconsByKey
  fullscreen: ReplayFullscreen
  frameControl: ReplayFrameControl
}

/** Le contenu que chaque cadre fournit aux zones du gabarit ; la scène et les avis sont ceux de la page. */
export interface ReplayFrameZoneContent {
  tete: ReactNode
  rangee: ReactNode
  colonne: ReactNode
  /**
   * La colonne est montée mais cachée, et le plan prend sa largeur : le fil complet qu'elle porte reste monté
   * (même nœud, mêmes défilement et filtre) sans être affiché. Absent = la colonne est affichée.
   */
  colonneCachee?: boolean
  /** Mesure du corps du cadre (`useCardRowLayout`) : posée sur le corps du gabarit ; absente = rien n'est mesuré. */
  bodyRef?: Ref<HTMLDivElement>
}
