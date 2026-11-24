/**
 * useReplayRelief — LE FOND RELIEF, SES TRAITS ET LA COUPE PAR ÉTAGE du rejeu : la vue commune des trois vues 2D
 * (`lib/replay/useReliefView`) plus ce qui n'appartient qu'au rejeu.
 *
 * Le canvas lui prête le fond de base (le rendu ou le plan, tel que la page l'a chargé), l'orientation
 * retenue et les encres ; il rend le fond à poser, les traits à cuire et les commandes du panneau « Vue ».
 * La chaîne (fichiers du relief, calculateur, images, replis, traits, rangs d'étage) est celle de `useReliefView` ;
 * ici restent :
 *  - le suivi du joueur ouvert, qui partage l'état de la coupe avec les choix manuels (`useReplayFloorChoice`) ;
 *  - les anneaux des pastilles d'étage, lien entre la commande et le langage d'étage des marqueurs (`reliefLogic`) ;
 *  - la pose du fond sur le fond de base du canvas (`composeReliefBackground`).
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
 * lue, l'étage de ses pieds. Un choix manuel (« Tout », pastille, curseur) le coupe.
 *
 * LES TRAITS DU RELIEF ont UNE valeur pour les trois fonds (`useReliefLines`), allumés d'office. Leur calque est cuit hors écran avec les autres (`useReplayStaticLayers`).
 */
import { useCallback, useMemo } from 'react'

import { resolveBackgroundVariant, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'
import type { ReliefLinesInput } from '@/lib/replay/relief/reliefLinesLayer'
import { useMatchReliefSource } from '@/lib/replay/reliefQueries'
import { useReliefLines } from '@/lib/replay/useReliefLines'
import { useReliefView, type ReliefViewInks } from '@/lib/replay/useReliefView'

import { composeReliefBackground, floorStateWithRings, type FloorState } from './reliefLogic'
import { useReplayFloorChoice } from './useReplayFloorChoice'
import { useReplayReliefFollow, type ReliefFollowInput } from './useReplayReliefFollow'
import type { ReplayMapBackgroundLayer } from './useReplayView'

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
  ink: ReliefViewInks
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

export function useReplayRelief(o: ReplayReliefOptions): ReplayRelief {
  const { base, playerSlug = '', matchId = '', scopeKey, style, turns, zRange, ink, follow } = o
  const planAvailable = !!base?.planAvailable
  const offered = !!base?.reliefAvailable
  // LE FOND VOULU, avant tout échec de lecture : il décide des traits retenus et de ce qu'il faut charger.
  const wanted = resolveBackgroundVariant(style, planAvailable, { available: offered, failed: false })
  const [linesOn, toggleLines] = useReliefLines()
  const choice = useReplayFloorChoice(scopeKey, follow.player !== null)
  const files = useMatchReliefSource(playerSlug, matchId, offered)
  const view = useReliefView({
    files,
    cut: choice.cut,
    onCut: choice.choose,
    style,
    wanted,
    planAvailable,
    linesOn,
    turns,
    ink,
    baseImage: base?.image ?? null,
    baseIsPlan: base?.variante === 'plan',
  })

  const background = useMemo(() => composeReliefBackground(base, view.surface, view.reliefShown), [base, view.surface, view.reliefShown])
  const viewState = view.floor.state
  const state = useMemo(() => floorStateWithRings(viewState, zRange), [viewState, zRange])
  // Le suivi ne tourne que sur une carte avec des étages ; sa coupe passe par le même état que les choix manuels.
  const followOn = choice.follow && viewState.state === 'ready'
  const followStep = useReplayReliefFollow({ ...follow, on: followOn, data: view.data, onCut: choice.followTo })
  const { setFollow } = choice
  const onToggle = useCallback(() => setFollow(!followOn), [setFollow, followOn])
  const { cut, selectAll, selectRank, setHeight } = view.floor
  return {
    background,
    selected: view.selected,
    options: view.options,
    meshPercent: view.meshPercent,
    lines: { available: view.linesAvailable, show: linesOn, onToggle: toggleLines },
    floor: { state, cut, selectAll, selectRank, setHeight, follow: { available: follow.player !== null, on: followOn, onToggle } },
    linesLayer: view.linesLayer,
    followStep,
  }
}
