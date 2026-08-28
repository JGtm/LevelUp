/**
 * replayCanvasConfig.ts — LES RÉGLAGES CONSTANTS DU CANVAS DE REJEU.
 *
 * DIXIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE `ReplayCanvas.tsx` (cf. placementFamily.guard) :
 * trois valeurs qui ne dépendent d'aucun état, d'aucune propriété et d'aucun hook. Elles étaient
 * en tête du composant et le composant n'en avait besoin qu'à la lecture — les sortir ne déplace
 * pas une ligne de logique et rend au canvas la marge qu'un calque de plus lui prend.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import type { CalloutZoneReady } from './calloutsLayer'
import type { ReplayFeedEntry } from './killFeedLogic'
import type { ReplayMediaItem } from './replayTimelineTracksLogic'

/**
 * 8 tokens de série : une teinte par GRANDE ZONE NOMMÉE (cyclés au-delà de 8 via
 * getSeriesColors), et — depuis le 2026-08-24 — la palette des COULEURS DISTINCTES PAR JOUEUR
 * quand l'option du tiroir la choisit. Par défaut un joueur porte la couleur de son ÉQUIPE (D1).
 */
export const SERIES_TOKENS: SemanticToken[] = [
  'chart-series-1', 'chart-series-2', 'chart-series-3', 'chart-series-4',
  'chart-series-5', 'chart-series-6', 'chart-series-7', 'chart-series-8',
]

/** Référence STABLE pour « pas de zones » : un `?? []` inline recuirait le calque à chaque rendu. */
export const EMPTY_ZONES: CalloutZoneReady[] = []

/**
 * Référence STABLE pour « pas de médias » — et, pour l'instant, la SEULE source de la piste
 * Médias : la donnée arrive en phase 2 (endpoint par match, cf. registre des reports). Une
 * référence nommée plutôt qu'un `[]` inline pour la même raison que les zones, et parce que le
 * jour où les médias arrivent, ce nom est exactement l'endroit où la prop se branche.
 */
export const EMPTY_MEDIA: ReplayMediaItem[] = []

/**
 * Référence STABLE pour « pas de fil » : le canvas peut être monté sans que la page ait encore
 * assemblé le fil aligné (la vue du match arrive après l'artefact). Un `?? []` inline
 * reconstruirait les pistes de la frise à chaque rendu, pour un résultat identique.
 */
export const EMPTY_FEED: ReplayFeedEntry[] = []

/**
 * Cadence de publication de l'image courante vers React, en millisecondes.
 *
 * POURQUOI PAS À CHAQUE IMAGE. Le canvas se redessine à la cadence de l'écran ; les fiches
 * joueur, elles, sont du DOM. Les re-rendre 60 fois par seconde coûterait tout le budget
 * d'animation pour un contenu qui change à peine. 150 ms reste bien en deçà de ce que l'œil
 * perçoit comme un retard sur un compteur, et divise le travail de React par dix.
 */
export const FRAME_PUBLISH_MS = 150

/**
 * LE SAUT DES DEUX BOUTONS qui encadrent la lecture, en secondes — et celui des flèches ←/→
 * (cf. `useReplayShortcuts`). Dix secondes est la convention des lecteurs vidéo, et le libellé
 * des boutons PORTE la durée (`skipBackFmt`/`skipForwardFmt`) : la changer ici change les deux
 * commandes, leur nom accessible et le raccourci, sans qu'aucun texte ne mente.
 *
 * ELLE VIT ICI ET NON DANS `ReplayTransport.tsx` (où le design l'avait posée) : un export qui
 * n'est pas un composant, depuis un fichier de composant, déclenche `react-refresh/
 * only-export-components` — la même raison qui a sorti `SlidersIcon` et les hooks du canvas.
 */
export const SKIP_SECONDS = 10
