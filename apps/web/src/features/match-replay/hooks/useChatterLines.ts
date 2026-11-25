/**
 * useChatterLines — LES RÉPLIQUES DES SPARTANS d'un match, prêtes à afficher : la table versée (chargée une fois, quand le film
 * porte au moins une réplique), les lignes du document, et les filtres des réglages.
 *
 * TROIS LECTEURS, UNE SEULE LECTURE : la carte (bulles et icônes), le fil et la voix lisent chacun ces lignes. Le fichier de
 * la table se télécharge UNE fois par titre (`loadedTables`) ; un film sans réplique ne le demande jamais.
 *
 * UNE REQUÊTE LANCÉE PAR UN EFFET S'ANNULE AU DÉMONTAGE : le signal d'annulation coupe la requête et la réponse arrivée
 * après coup n'écrit plus rien. Un fichier introuvable ou illisible laisse la table vide : les répliques ne s'affichent
 * pas, le reste du rejeu est intact.
 */
import { useEffect, useMemo, useState } from 'react'

import { api } from '@/lib/api/client'
import type { Locale } from '@/lib/i18n/locale'
import { staticAssetURL } from '@/lib/staticAssets'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  buildChatterLines,
  filterChatterLines,
  type ChatterFilter,
  type ChatterLine,
} from '../model/chatterLines'
import { CHATTER_TABLE_STEM, EMPTY_CHATTER_TABLE, parseChatterTable, type ChatterTable } from '../model/chatterTable'
import { chatterFilterOf, useChatterPrefs } from '../settings/useChatterPrefs'

/** Les tables déjà chargées, par titre : un seul téléchargement pour toute la page. */
const loadedTables = new Map<string, ChatterTable>()

/** La table des répliques du titre, vide tant qu'elle n'est pas arrivée (ou si le film n'a aucune réplique à lire). */
export function useChatterTable(titleSlug: string, needed: boolean): ChatterTable {
  const [fetched, setFetched] = useState<{ slug: string; table: ChatterTable } | null>(null)
  useEffect(() => {
    if (!needed || loadedTables.has(titleSlug)) return
    const controller = new AbortController()
    api
      .getStatic<unknown>(staticAssetURL('sound', CHATTER_TABLE_STEM, '.json', titleSlug), { signal: controller.signal })
      .then((raw) => {
        const table = parseChatterTable(raw)
        loadedTables.set(titleSlug, table)
        if (!controller.signal.aborted) setFetched({ slug: titleSlug, table })
      })
      .catch(() => {
        /* Table introuvable ou illisible, ou requête annulée : aucune réplique n'est affichée, le reste du rejeu ne change pas. */
      })
    return () => controller.abort()
  }, [titleSlug, needed])
  if (!needed) return EMPTY_CHATTER_TABLE
  return loadedTables.get(titleSlug) ?? (fetched?.slug === titleSlug ? fetched.table : EMPTY_CHATTER_TABLE)
}

/** Toutes les répliques affichables du match (sans filtre), dans l'ordre du film. */
export function useChatterLines(doc: ReplayDocumentReady | null | undefined, locale: Locale): readonly ChatterLine[] {
  const chatter = doc?.chatter
  const table = useChatterTable(doc?.titleSlug ?? '', (chatter?.length ?? 0) > 0)
  return useMemo(() => (chatter ? buildChatterLines(chatter, table, locale) : []), [chatter, table, locale])
}

/** Les filtres des réglages (familles, fréquence), même objet tant que rien ne change. */
export function useChatterFilter(): ChatterFilter {
  const prefs = useChatterPrefs()
  const { families, frequency } = prefs
  return useMemo(() => chatterFilterOf({ families, frequency }), [families, frequency])
}

/** Les répliques qui passent les filtres des réglages. */
export function useFilteredChatterLines(lines: readonly ChatterLine[], filter: ChatterFilter): readonly ChatterLine[] {
  return useMemo(() => filterChatterLines(lines, filter), [lines, filter])
}

/** Aucune réplique dans le fil : la liste vide partagée. */
const NO_LINES: readonly ChatterLine[] = []

/**
 * Les répliques qui rejoignent le FIL : celles qui passent les familles et la fréquence des réglages, et seulement si le réglage
 * « Répliques dans le fil » est allumé. Le masquage des joueurs s'applique à l'affichage (`useFeedView`), pas ici : le menu du
 * filtre compte les répliques de tout le match, comme il compte les éliminations.
 */
export function useChatterFeedLines(doc: ReplayDocumentReady | null | undefined, locale: Locale): readonly ChatterLine[] {
  const prefs = useChatterPrefs()
  const { showFeed, families, frequency } = prefs
  const all = useChatterLines(doc, locale)
  const filter = useMemo(() => chatterFilterOf({ families, frequency }), [families, frequency])
  return useMemo(() => (showFeed ? filterChatterLines(all, filter) : NO_LINES), [showFeed, all, filter])
}
