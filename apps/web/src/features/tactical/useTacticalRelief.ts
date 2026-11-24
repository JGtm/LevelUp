/**
 * useTacticalRelief — LE FOND RELIEF, SES TRAITS ET LA COUPE PAR ÉTAGE du plan de l'onglet Tactique, en un seul
 * hook que la carte du plan (placement) et celle des réapparitions appellent chacune.
 *
 * Il branche la lecture commune des vues 2D (`useReliefView`) sur ce que la Tactique sait d'une carte : son sidecar
 * (une image plan ? un relief ?), le fond de base que `useTacticalFondVariante` charge (le plan, ou le rendu), et le
 * style de fond du rejeu 2D (`useMapViewPreferences`, une seule clé lue et écrite des deux côtés), les traits du
 * relief (`useReliefLines`, une seule valeur pour tous les fonds, la clé du rejeu).
 *
 * CE QUE LA COUPE FAIT ICI : le fond la suit (relief teinté, ou plan/rendu modulé par son masque), les traits aussi,
 * et deux calques seulement portent une hauteur — les zones nommées de la lecture de placement et celles de la
 * lecture des réapparitions (`zonesInCut`). Tout autre calque reste entier, ce que la légende du plan dit.
 *
 * Le fond de base n'est décodé (`useTacticalBackgroundElement`) que lorsqu'une coupe doit le moduler. La coupe repart
 * de « Tout » pour une autre carte, et à l'ouverture de l'onglet ; elle n'est jamais persistée.
 */
import { useMemo } from 'react'

import { useMapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'
import { resolveBackgroundVariant, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { useReliefInks } from '@/lib/replay/useReliefInks'
import { useReliefLines } from '@/lib/replay/useReliefLines'
import { useReliefView, type ReliefView } from '@/lib/replay/useReliefView'
import { useScopedCut } from '@/lib/replay/useScopedCut'
import { useAppShellStore } from '@/stores/appShellStore'

import { useTacticalMapOffers, tacticalReliefSource, useTacticalBackgroundElement } from './queriesRelief'
import { useTacticalFondVariante } from './useTacticalFondVariante'

/** Ce que les commandes et le plan lisent : la vue du relief, plus le fond choisi, les traits et le fond réellement posé. */
export interface TacticalRelief extends ReliefView {
  /** Change et retient le style de fond (la clé du rejeu 2D). */
  setStyle: (style: MapBackgroundVariant) => void
  /** Les traits du relief sont allumés pour le fond voulu, et leur bascule. */
  linesOn: boolean
  toggleLines: () => void
  /** Le fond RÉELLEMENT posé : le relief quand il est peint, sinon l'image de base chargée (plan ou rendu). */
  drawn: MapBackgroundVariant
}

export function useTacticalRelief(playerSlug: string, mapId: string, turns: QuarterTurns): TacticalRelief {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { backgroundStyle, setBackgroundStyle } = useMapViewPreferences(mapId)
  const { planImage, reliefOffered } = useTacticalMapOffers(playerSlug, mapId)
  const planAvailable = !!planImage
  // Le fond VOULU, avant tout échec de lecture : il décide des traits retenus et de ce qu'il faut charger.
  const wanted = resolveBackgroundVariant(backgroundStyle, planAvailable, { available: reliefOffered, failed: false })
  const [linesOn, toggleLines] = useReliefLines()
  const [cut, onCut] = useScopedCut(mapId)
  const base = useTacticalFondVariante(playerSlug, mapId)
  // Le fond de base se décode pour être modulé, seulement quand une coupe est choisie sur un fond qui n'est pas le relief.
  const element = useTacticalBackgroundElement(mapId, base.variante, base.url, reliefOffered && wanted !== 'relief' && cut !== null)
  const files = useMemo(
    () => (reliefOffered && playerSlug && mapId ? tacticalReliefSource(titleSlug, playerSlug, mapId) : null),
    [reliefOffered, titleSlug, playerSlug, mapId],
  )
  const ink = useReliefInks()
  const view = useReliefView({
    files,
    cut,
    onCut,
    style: backgroundStyle,
    wanted,
    planAvailable,
    linesOn,
    turns,
    ink,
    baseImage: element,
    baseIsPlan: base.variante === 'plan',
  })
  return { ...view, setStyle: setBackgroundStyle, linesOn, toggleLines, drawn: view.reliefShown ? 'relief' : base.variante }
}
