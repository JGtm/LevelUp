/**
 * queries.ts — Hook TanStack Query pour POST /filters/resolve.
 *
 * Le globalFilterStore définit un slot `resolvedContext` (sessions, options
 * cascade) que FilterOmnibar / SessionNavBar / SquadLayout consomment. Sans ce hook,
 * resolvedContext reste null, donc :
 *  - le sélecteur de session affiche "Aucune session disponible"
 *  - les filtres cascade (Playlists, Modes, Cartes, Types) sont absents
 *  - le default-to-latest sur Squad ne se déclenche jamais
 *
 * Ce hook fait le pont : à chaque changement de filterContext, on POST
 * l'endpoint et on dispatche la réponse via setResolvedContext.
 */
import { useEffect } from 'react'
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import type { FilterStore } from '@/stores/createFilterStore'
import type { FilterContextInput, FilterContextResolved } from '@/lib/api/types'

/**
 * Résout le filterContext courant côté backend (sessions disponibles + options
 * cascade) et synchronise le résultat dans le store contextuel passé en arg.
 *
 * À monter une fois par page joueur ; le PlayerLayout l'appelle pour le store
 * solo, SquadLayout pour le store squad. Le hook re-fetch automatiquement
 * quand `filterContextHash` change.
 *
 * Défaut : `useSoloFilterStore` (rétrocompat avec PlayerLayout).
 */
export function useFiltersResolve(playerSlug: string, filterStore: FilterStore = useSoloFilterStore) {
  const filterContext = filterStore((s) => s.filterContext)
  const filterContextHash = filterStore((s) => s.filterContextHash)
  const setResolvedContext = filterStore((s) => s.setResolvedContext)
  // Le titre courant scope la clé (cf. queryKeys.filtersResolve) : au switch de
  // titre la clé change → refetch des options du bon titre, jamais de serve périmé.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)

  const query = useQuery<FilterContextResolved>({
    queryKey: queryKeys.filtersResolve(playerSlug, titleSlug, filterContextHash),
    queryFn: () =>
      api.post<FilterContextResolved>(
        `/players/${playerSlug}/filters/resolve`,
        filterContext satisfies FilterContextInput,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })

  // Synchronise la réponse dans le store + track le latest session_id pour
  // permettre la détection "nouvelles sessions arrivées" (auto-snap dans
  // PlayerLayout sur fin de sync). Note : avec 2 stores contextuels, le snap
  // est désormais centralisé dans PlayerLayout qui détecte la nature solo/squad.
  useEffect(() => {
    if (!query.data) return
    setResolvedContext(query.data)
  }, [query.data, setResolvedContext])

  return query
}

/**
 * useFollowLatestSession — atterrit sur la dernière session tant que l'utilisateur
 * n'a rien épinglé manuellement ("follow-latest"). Piloté par l'ÉTAT
 * (`resolvedContext`), pas par un événement de sync : couvre donc le montage/reload,
 * le refetch post-sync ET la navigation — là où l'ancien trigger (transition
 * activeSyncJobId dans PlayerLayout) ne se déclenchait quasiment jamais.
 *
 * Intention "follow-latest" dérivée (pas de flag dédié) :
 *   - vraie si isAutoSnappingToLatest (on suit déjà la dernière), OU
 *   - vraie en état vierge (aucune période ni session pickée).
 * Dès qu'une session/période est épinglée manuellement, les setters du store
 * repassent isAutoSnappingToLatest=false → followLatest devient faux → la sélection
 * manuelle est préservée (jamais re-snappée).
 *
 * scope='solo' suit la dernière session solo (`!is_squad`), 'squad' la dernière squad.
 *
 * MONTÉ UNE SEULE FOIS, en scope 'solo', par la route joueur
 * (`routes/{-$lang}/t/$titleSlug/players/$playerSlug.tsx`). L'escouade ne le monte
 * PAS : son ancrage de session est piloté par la COMPOSITION (effet de ré-ancrage
 * de SquadLayout), parce que snapper sur la dernière session squad du joueur
 * principal est composition-agnostique — cela ajoutait un coéquipier à une session
 * qu'il n'avait pas jouée. Le scope 'squad' n'a donc plus qu'un appelant : les
 * tests unitaires de ce hook. Ne pas le remonter dans SquadLayout sans rouvrir
 * cette décision.
 */
export function useFollowLatestSession(
  playerSlug: string,
  filterStore: FilterStore,
  scope: 'solo' | 'squad',
) {
  const resolvedContext = filterStore((s) => s.resolvedContext)
  const isAutoSnapping = filterStore((s) => s.isAutoSnappingToLatest)

  useEffect(() => {
    if (!resolvedContext) return
    const all = resolvedContext.session_options?.all_sessions ?? []
    const latest = all.find((s) => (scope === 'squad' ? s.is_squad : !s.is_squad))
    if (!latest) return

    const {
      filterContext,
      lastKnownLatestSessionId,
      setLastKnownLatestSessionId,
      autoSnapToLatestSession,
    } = filterStore.getState()

    const picked = filterContext.sessions?.picked_sessions ?? []
    const hasPeriod = !!(filterContext.period?.start_date || filterContext.period?.end_date)
    const followLatest = isAutoSnapping || (!hasPeriod && picked.length === 0)
    if (!followLatest) return // sélection manuelle épinglée → on la préserve

    // Déjà sur la dernière (label OU session_id legacy hérité) → pas de re-snap.
    // Garde anti-boucle : sans ça, snap → hash change → refetch → resolvedContext
    // change → effet re-déclenché → snap…
    const alreadyOnLatest =
      picked.length === 1 && (picked[0] === latest.label || picked[0] === latest.session_id)
    if (alreadyOnLatest) {
      // Resync de la clé de détection sans toucher au filterContextHash (pas de refetch).
      if (latest.session_id !== lastKnownLatestSessionId) {
        setLastKnownLatestSessionId(latest.session_id)
      }
      return
    }

    autoSnapToLatestSession(latest, true)
    // filterContext/lastKnownLatestSessionId sont lus via getState() (hors closure)
    // pour ne PAS re-déclencher l'effet à chaque frappe de filtre ; les vrais
    // déclencheurs sont resolvedContext, isAutoSnapping et playerSlug (changement
    // de joueur). Le tableau de deps reste donc exhaustif côté react-hooks.
  }, [resolvedContext, isAutoSnapping, playerSlug, scope, filterStore])
}

/**
 * Résout un FilterContextInput arbitraire (état pending) sans écrire dans le store.
 * Utilisé pour le feedback immédiat sur les incompatibilités de filtres dans le
 * dropdown, avant que l'utilisateur clique sur Analyser.
 */
export function useFiltersPreview(playerSlug: string, input: FilterContextInput) {
  // Le titre courant scope la clé (cf. queryKeys.filtersPreview) — même motif que
  // useFiltersResolve : pas de preview périmé d'un autre titre après bascule.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // FNV-1a 32 bits — même algo que computeHash/computePendingHash. Crucial :
  // le hash doit refléter les diffs de cascade pour que toggler une checkbox
  // dans FiltresPill déclenche un refetch et mette à jour available_options.
  const hash = (() => {
    const s = JSON.stringify(input) ?? ''
    let h = 0x811c9dc5
    for (let i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i)
      h = Math.imul(h, 0x01000193) >>> 0
    }
    return h.toString(16).padStart(8, '0')
  })()
  return useQuery<FilterContextResolved>({
    queryKey: queryKeys.filtersPreview(playerSlug, titleSlug, hash),
    queryFn: () =>
      api.post<FilterContextResolved>(`/players/${playerSlug}/filters/resolve`, input),
    enabled: !!playerSlug,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  })
}
