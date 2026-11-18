/**
 * useReplayRelief — LE FOND RELIEF, SES TRAITS ET LA COUPE PAR ÉTAGE du rejeu, assemblés en un seul hook.
 *
 * Le canvas lui prête le fond de base (le rendu ou le plan, tel que la page l'a chargé), l'orientation
 * retenue et les encres ; il rend le fond à poser, les traits à cuire et les commandes du panneau « Vue ».
 * Il porte donc toute la chaîne : sidecar annonce un relief -> fichiers (`useReplayReliefFiles`) ->
 * calculateur (`useReliefRender`) -> images (`layers/reliefBackground`) -> fond composé (`reliefLogic`).
 *
 * LES TROIS FONDS : Plan (défaut), Rendu, Relief (préférence globale `replay-background-style`). Le Relief
 * n'est posé que s'il est lisible : sans relief au sidecar, ou sur une `ReliefError` (version inconnue,
 * image illisible, panne du calculateur — journalisée par la lecture), le PLAN reste posé, sans rien dire à
 * l'écran (le rendu s'il n'y a pas non plus de plan). Pendant la lecture du relief, le fond de base reste posé.
 *
 * LA COUPE vaut pour les trois fonds (D-2) : sur le Relief par son opacité, sur le Plan et le Rendu par un
 * masque d'opacité du relief. Elle est un état de CETTE page (jamais persistée : « Tout » à l'ouverture) et ne
 * masque AUCUN joueur. Sans maillage de navigation, il n'y a pas de coupe : la commande dit pourquoi.
 *
 * « SUIVRE LE JOUEUR OUVERT » pose la coupe à la place de l'utilisateur (`useReplayReliefFollow`) : à chaque image
 * lue, l'étage de ses pieds. Il partage l'état de la coupe (`useReplayFloorChoice`), et un choix manuel le coupe.
 *
 * LES TRAITS DU RELIEF sont retenus PAR FOND (`useReliefLines`) : allumés d'office sur le Relief, éteints sur
 * le Plan et le Rendu. Leur calque est cuit hors écran avec les autres (`useReplayStaticLayers`).
 *
 * L'IMAGE DES HAUTEURS, la plus lourde des deux fichiers, n'est demandée que lorsqu'une commande s'en sert
 * (fond Relief, traits allumés, coupe choisie). Les données (étages, traits) sont lues dès que le sidecar annonce
 * le relief : elles décident de ce que le panneau propose.
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import { resolveBackgroundVariant, reliefUsable, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { meshShareOf, quantizeCut, reliefRamp, rgbToCss, type ReliefData } from '@/lib/replay/relief'
import { useReplayReliefFiles } from '@/lib/replay/reliefQueries'
import { ReliefCanvases } from '@/lib/replay/relief/reliefBackground'
import type { ReliefLinesInk, ReliefLinesInput } from '@/lib/replay/relief/reliefLinesLayer'
import { useReliefRender } from '@/lib/replay/useReliefRender'

import { useReliefLines } from '../settings/useReliefLines'
import {
  backgroundOptions,
  composeReliefBackground,
  floorStateOf,
  reliefRequestOf,
  type FloorState,
  type ReliefImages,
} from './reliefLogic'
import { useReplayFloorChoice } from './useReplayFloorChoice'
import { useReplayReliefFollow, type ReliefFollowInput } from './useReplayReliefFollow'
import type { ReplayMapBackgroundLayer } from './useReplayView'

/** Les encres du relief, résolues par jeton par l'appelant. */
export interface ReliefInks {
  /** `map-relief-base` : la teinte de base de la rampe. */
  base: string
  /** Le filet clair du cerne des pions (`marker-outline-rim`) : le liseré de la zone jouable. */
  outline: string
}

export interface ReplayReliefOptions {
  /** Le fond de base du canvas (le rendu ou le plan, ou `null` : pas de fond). */
  base: ReplayMapBackgroundLayer | null
  /** Le match affiché : désigne les fichiers du relief. Absents = pas de relief. */
  playerSlug?: string
  matchId?: string
  /** La portée de la coupe : une autre carte repart de « Tout ». */
  scopeKey: string
  /** Le fond voulu par l'utilisateur (`useBackgroundStyle`). */
  style: MapBackgroundVariant
  /** L'orientation EFFECTIVE de la carte : elle ne règle que la direction de la lumière. */
  turns: QuarterTurns
  /** L'amplitude verticale du document : elle dit combien d'anneaux porte un pion à chaque étage. */
  zRange: { min: number; max: number }
  ink: ReliefInks
  /** Ce que « Suivre le joueur ouvert » lit : le document, le joueur de la fiche ouverte, la cellule de dessin. */
  follow: ReliefFollowInput
}

/**
 * La commande d'étage et la coupe retenue (`null` = « Tout »). Les trois gestes de choix (« Tout », pastille,
 * curseur) sont MANUELS : chacun coupe le suivi du joueur ouvert.
 */
export interface ReplayFloorState {
  state: FloorState
  cut: number | null
  selectAll: () => void
  /** Retient un étage par sa hauteur exacte (celle d'une pastille). */
  selectRank: (z: number) => void
  /** Retient une hauteur libre, arrondie au quart d'unité (le curseur). */
  setHeight: (z: number) => void
  /** « Suivre le joueur ouvert » : disponible quand une fiche est ouverte, allumé ou non, et son interrupteur. */
  follow: { available: boolean; on: boolean; onToggle: () => void }
}

/** Ce que le hook rend. */
export interface ReplayRelief {
  /** Le fond à poser : le relief teinté, ou le fond de base (modulé par la coupe). */
  background: ReplayMapBackgroundLayer | null
  /** Le fond RETENU après les replis (relief illisible -> plan -> rendu) : ce que le segmenté du panneau montre. */
  selected: MapBackgroundVariant
  /** Les fonds que le panneau propose ; vide quand il n'y a pas de choix. */
  options: readonly MapBackgroundVariant[]
  /** La part du relief tirée du maillage (D-16), en pourcentage entier ; `null` quand elle est inconnue. */
  meshPercent: number | null
  /** L'interrupteur « Traits du relief ». */
  lines: { available: boolean; show: boolean; onToggle: () => void }
  floor: ReplayFloorState
  /** Les traits à cuire dans le calque statique ; `null` = pas de traits. */
  linesLayer: ReliefLinesInput | null
  /** Le pas du suivi de l'étage : à appeler à chaque dessin avec l'image lue ; ne fait rien suivi éteint. */
  followStep: (frame: number) => void
}

const NO_IMAGES: ReliefImages = { relief: null, masked: null }
const NO_FLOOR: FloorState = { state: 'none' }

/** Les encres des traits, tirées de la base de teinte ; `null` quand la base n'est pas une couleur lisible. */
function linesInkOf(ink: ReliefInks): ReliefLinesInk | null {
  const ramp = reliefRamp(ink.base)
  if (!ramp) return null
  return { edge: rgbToCss(ramp.edge), contour: rgbToCss(ramp.contour), wall: rgbToCss(ramp.wall), outline: ink.outline }
}

/** La part du relief tirée du maillage, en pourcentage entier, ou `null` quand le fichier ne la porte pas. */
function meshPercentOf(data: ReliefData | null): number | null {
  return data ? (meshShareOf(data)?.percent ?? null) : null
}

export function useReplayRelief(o: ReplayReliefOptions): ReplayRelief {
  const { base, playerSlug = '', matchId = '', scopeKey, style, turns, zRange, ink, follow } = o
  const planAvailable = !!base?.planAvailable
  const offered = !!base?.reliefAvailable
  // LE FOND VOULU, avant tout échec de lecture : il décide des traits retenus et de ce qu'il faut charger.
  const wanted = resolveBackgroundVariant(style, planAvailable, { available: offered, failed: false })
  const [linesOn, toggleLines] = useReliefLines(wanted)
  const choice = useReplayFloorChoice(scopeKey, follow.player !== null)
  const cut = choice.cut
  const needImage = offered && (wanted === 'relief' || linesOn || cut !== null)
  const files = useReplayReliefFiles(playerSlug, matchId, offered, needImage)
  const request = useMemo(
    () => reliefRequestOf({ needed: needImage, turns, cut, tintBase: wanted === 'relief' ? ink.base : null, lines: linesOn }),
    [needImage, turns, cut, wanted, ink.base, linesOn],
  )
  const render = useReliefRender(files.data, files.blob, request)

  // LE REPLI : une lecture qui échoue (fichier absent, version inconnue, image illisible, calculateur en panne)
  // rend le plan. L'erreur des FICHIERS se journalise ici, celle du calculateur par `useReliefRender`.
  const failed = files.error !== null || render.status === 'failed'
  useEffect(() => {
    if (files.error !== null) console.warn(`[replay-relief] lecture des fichiers du relief impossible, repli sur le plan : ${String(files.error)}`)
  }, [files.error])
  const usable = reliefUsable({ available: offered, failed })
  const selected = resolveBackgroundVariant(style, planAvailable, { available: offered, failed })
  const options = backgroundOptions(planAvailable, usable)

  const [canvases] = useState(() => new ReliefCanvases())
  const frame = render.frame
  const baseImage = base?.image ?? null
  const baseIsPlan = base?.variante === 'plan'
  const images = useMemo<ReliefImages>(() => {
    if (!frame || !usable || !baseImage) return NO_IMAGES
    if (selected === 'relief') return { relief: canvases.reliefImage(frame), masked: null }
    return { relief: null, masked: cut === null ? null : canvases.maskedImage(frame, baseImage, baseIsPlan) }
  }, [frame, usable, selected, cut, baseImage, baseIsPlan, canvases])
  const background = useMemo(() => composeReliefBackground(base, selected, images), [base, selected, images])

  const data = usable ? files.data : null
  const linesInk = useMemo(() => linesInkOf(ink), [ink])
  const linesLayer = useMemo<ReliefLinesInput | null>(() => {
    if (!linesOn || !data || !frame?.lines || !linesInk) return null
    return {
      calibration: data.calibration,
      steps: frame.lines.steps,
      contours: frame.lines.contours,
      walls: frame.lines.walls,
      outline: data.lines.outline,
      ink: linesInk,
    }
  }, [linesOn, data, frame, linesInk])

  const state = useMemo(() => (data ? floorStateOf(data, zRange) : NO_FLOOR), [data, zRange])
  // Le suivi ne tourne que sur une carte avec des étages ; sa coupe passe par le même état que les choix manuels.
  const followOn = choice.follow && state.state === 'ready'
  const followStep = useReplayReliefFollow({ ...follow, on: followOn, data, onCut: choice.followTo })
  const { choose, setFollow } = choice
  const selectAll = useCallback(() => choose(null), [choose])
  const selectRank = useCallback((z: number) => choose(z), [choose])
  const setHeight = useCallback((z: number) => choose(quantizeCut(z)), [choose])
  const onToggle = useCallback(() => setFollow(!followOn), [setFollow, followOn])
  return {
    background,
    selected,
    options,
    meshPercent: meshPercentOf(data),
    lines: { available: data !== null, show: linesOn, onToggle: toggleLines },
    floor: {
      state, cut: state.state === 'ready' ? cut : null, selectAll, selectRank, setHeight,
      follow: { available: follow.player !== null, on: followOn, onToggle },
    },
    linesLayer,
    followStep,
  }
}
