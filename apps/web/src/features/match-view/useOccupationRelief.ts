/**
 * useOccupationRelief — LE FOND RELIEF, SES TRAITS ET LA COUPE PAR ÉTAGE d'« Occupation du terrain ».
 *
 * Il branche la lecture commune des vues 2D (`useReliefView`) sur ce que le bloc sait : le sidecar du fond du MATCH
 * (une image plan ? un relief ?), le fond de base que le bloc a chargé (plan ou rendu, déjà décodé), le style de fond
 * et les traits du rejeu 2D (une seule clé chacun, lus et écrits des deux côtés : choisir le relief ici le choisit au
 * rejeu). Le bloc a désormais son PROPRE choix de fond (une pilule), plus seulement la préférence du rejeu en lecture.
 *
 * CE QUE LA COUPE FAIT ICI : le fond la suit, les traits aussi, et les POSITIONS hors de l'étage choisi sont MASQUÉES
 * (`splitPositionsByCut`, par le sol qui les porte) — seule vue où un joueur peut disparaître de l'écran, le bloc dit
 * combien. La coupe repart de « Tout » pour une autre carte et n'est jamais persistée.
 */
import { useMemo } from 'react'

import { useReliefLines } from '@/features/match-replay/settings/useReliefLines'
import type { ReplayMapBackground } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import {
  backgroundImagePath,
  reliefDataPath,
  resolveBackgroundVariant,
  type MapBackgroundImageVariant,
  type MapBackgroundVariant,
} from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'
import type { ReliefFilesSource } from '@/lib/replay/reliefQueries'
import { useReliefInks } from '@/lib/replay/useReliefInks'
import { useReliefView, type ReliefView } from '@/lib/replay/useReliefView'
import { useScopedCut } from '@/lib/replay/useScopedCut'
import { useAppShellStore } from '@/stores/appShellStore'

/** Où lire le relief du fond d'un match (les clés du rejeu : un relief est propre à la carte du match). */
function matchReliefSource(titleSlug: string, playerSlug: string, matchId: string): ReliefFilesSource {
  const imageBase = `/players/${playerSlug}/matches/${matchId}/replay/background.png`
  return {
    dataKey: queryKeys.matchReplayReliefData(playerSlug, titleSlug, matchId),
    dataPath: reliefDataPath(imageBase),
    imageKey: queryKeys.matchReplayReliefImage(playerSlug, titleSlug, matchId),
    imagePath: backgroundImagePath(imageBase, 'relief'),
  }
}

export interface OccupationReliefOptions {
  playerSlug: string
  matchId: string
  /** La carte du match : portée de la coupe ; vide = inconnue. */
  mapId: string
  /** Le sidecar du fond du match ; `undefined` tant qu'il n'est pas lu ou si la carte n'a pas de fond figé. */
  background: ReplayMapBackground | undefined
  style: MapBackgroundVariant
  turns: QuarterTurns
  /** Le fond de base chargé (plan ou rendu), et sa variante RÉELLE. */
  base: { image: CanvasImageSource | null; variante: MapBackgroundImageVariant }
}

/** Ce que le bloc lit : la vue du relief, plus les traits retenus et le fond RÉELLEMENT posé. */
export interface OccupationRelief extends ReliefView {
  linesOn: boolean
  toggleLines: () => void
  /** Le fond RÉELLEMENT posé : le relief quand il est peint, sinon l'image de base chargée (plan ou rendu). */
  drawn: MapBackgroundVariant
}

export function useOccupationRelief(o: OccupationReliefOptions): OccupationRelief {
  const { playerSlug, matchId, mapId, background, style, turns, base } = o
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const planAvailable = !!background?.planImage
  const reliefOffered = !!background?.reliefImage && !!background.reliefData
  // Le fond VOULU, avant tout échec de lecture : il décide des traits retenus et de ce qu'il faut charger.
  const wanted = resolveBackgroundVariant(style, planAvailable, { available: reliefOffered, failed: false })
  const [linesOn, toggleLines] = useReliefLines(wanted)
  const [cut, onCut] = useScopedCut(mapId || matchId)
  const files = useMemo(
    () => (reliefOffered && playerSlug && matchId ? matchReliefSource(titleSlug, playerSlug, matchId) : null),
    [reliefOffered, titleSlug, playerSlug, matchId],
  )
  const ink = useReliefInks()
  const view = useReliefView({
    files,
    cut,
    onCut,
    style,
    wanted,
    planAvailable,
    linesOn,
    turns,
    ink,
    baseImage: base.image,
    baseIsPlan: base.variante === 'plan',
  })
  return { ...view, linesOn, toggleLines, drawn: view.reliefShown ? 'relief' : base.variante }
}
