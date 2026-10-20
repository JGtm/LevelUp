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

/** Options de `useFiltersResolve` (lot perf L4a, 2026-09-23). */
export interface FiltersResolveOptions {
  /**
   * `match_context` injecté dans le corps ET porté par la clé (jamais un résolu
   * solo servi à l'escouade, ni l'inverse). L'escouade passe 'squad' : son résolu
   * commité est le repli de l'aperçu, qui ne tourne plus que filtres en attente
   * (D4.3) — il doit donc résoudre la même population que lui. Absent : corps
   * inchangé, tous les matchs (le serveur lit l'absence comme 'all').
   */
  matchContext?: NonNullable<FilterContextInput['match_context']>
  /** Faux : aucune requête (résolu solo hors pages Stats, D4.4). Défaut : vrai. */
  enabled?: boolean
}

/**
 * Résout le filterContext courant côté backend (sessions disponibles + options
 * cascade) et synchronise le résultat dans le store contextuel passé en arg.
 *
 * À monter une fois par page joueur ; le PlayerLayout l'appelle pour le store
 * solo (actif seulement sous la barre solo, D4.4), SquadLayout pour le store
 * squad (en match_context 'squad', D4.3). Le hook re-fetch automatiquement
 * quand `filterContextHash` change.
 *
 * Défaut : `useSoloFilterStore` (rétrocompat avec PlayerLayout).
 */
export function useFiltersResolve(
  playerSlug: string,
  filterStore: FilterStore = useSoloFilterStore,
  { matchContext, enabled = true }: FiltersResolveOptions = {},
) {
  const storedFilterContext = filterStore((s) => s.filterContext)
  const filterContextHash = filterStore((s) => s.filterContextHash)
  const setResolvedContext = filterStore((s) => s.setResolvedContext)
  // Le titre courant scope la clé (cf. queryKeys.filtersResolve) : au switch de
  // titre la clé change → refetch des options du bon titre, jamais de serve périmé.
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Corps envoyé : le filterContext du store, avec `match_context` quand il est
  // demandé (sans lui le corps est celui du store, tel quel).
  const filterContext: FilterContextInput = matchContext
    ? { ...storedFilterContext, match_context: matchContext }
    : storedFilterContext

  const query = useQuery<FilterContextResolved>({
    queryKey: queryKeys.filtersResolve(playerSlug, titleSlug, filterContextHash, matchContext ?? 'all'),
    queryFn: ({ signal }) =>
      api.post<FilterContextResolved>(
        `/players/${playerSlug}/filters/resolve`,
        filterContext satisfies FilterContextInput,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && enabled,
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
  // Faux : aucun snap (hors pages Stats, D4.4 — le résolu n'y est pas rafraîchi,
  // `resolvedContext` peut y dater d'une autre page, voire d'un autre joueur).
  { enabled = true }: { enabled?: boolean } = {},
) {
  const resolvedContext = filterStore((s) => s.resolvedContext)
  const isAutoSnapping = filterStore((s) => s.isAutoSnappingToLatest)

  useEffect(() => {
    if (!enabled || !resolvedContext) return
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
    // déclencheurs sont resolvedContext, isAutoSnapping, playerSlug (changement
    // de joueur) et enabled (retour sur une page Stats). Le tableau de deps reste
    // donc exhaustif côté react-hooks.
  }, [resolvedContext, isAutoSnapping, playerSlug, scope, filterStore, enabled])
}

/**
 * Résout un FilterContextInput arbitraire (état pending) sans écrire dans le store.
 * Utilisé pour le feedback immédiat sur les incompatibilités de filtres dans le
 * dropdown, avant que l'utilisateur clique sur Analyser.
 *
 * `enabled` : l'Escouade ne l'active que si des filtres sont EN ATTENTE (D4.3,
 * 2026-09-23) — sinon le résolu commité dit déjà la même chose. Désactivée, la
 * requête peut encore rendre la donnée d'une clé précédente (placeholder
 * `keepPreviousData`) : l'appelant qui la désactive doit l'ignorer.
 */
export function useFiltersPreview(
  playerSlug: string,
  input: FilterContextInput,
  { enabled = true }: { enabled?: boolean } = {},
) {
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
    queryFn: ({ signal }) =>
      api.post<FilterContextResolved>(
        `/players/${playerSlug}/filters/resolve`,
        input,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  })
}
