/**
 * usePageScope — primitif générique de persistance d'état de page (filtres,
 * recherche, tri) sur le modèle « hybride URL + localStorage ».
 *
 * Problème résolu : avant ce hook, les filtres d'Explorer vivaient en
 * `useState` local et étaient perdus dès qu'on ouvrait un match puis revenait
 * en arrière (le composant se remontait à zéro). Voir le plan
 * nav-context-unification (Phase 1).
 *
 * Modèle :
 *   - **URL = source de vérité.** Toute mutation écrit dans les query params
 *     via `navigate({ replace: true })`. Le bouton « page précédente » du
 *     navigateur restaure alors l'URL (donc les filtres) sans aucune logique
 *     supplémentaire — c'est ce qui corrige la douleur principale.
 *   - **localStorage = miroir secondaire** pour le cold-start : si on atterrit
 *     sur la page sans aucun param de scope (nouvel onglet, app rouverte), on
 *     réhydrate les derniers filtres connus. `reset()` purge ce miroir pour ne
 *     pas ressusciter des filtres volontairement effacés.
 *
 * Le hook est agnostique de la page : il manipule deux représentations via
 * `encode`/`decode` (cf. explorerScope.ts pour l'instance Explorer) :
 *   - `App` : état riche consommé par le composant (Set<string>, chaînes…)
 *   - `Url` : forme plate sérialisable (toutes clés optionnelles, chaînes)
 *
 * `replace: true` (et non push) : les changements de filtres ne polluent pas
 * l'historique — sinon « page précédente » repasserait par chaque frappe.
 */
import { useCallback, useEffect, useRef } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'
import type { FileRouteTypes } from '@/routeTree.gen'

export interface UsePageScopeOptions<App, Url extends object> {
  /** Cible `navigate` — chemin de route typé du routeur généré (ex:
   *  '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer'). */
  to: FileRouteTypes['to']
  /** Params de route pour `navigate` (ex: { titleSlug, playerSlug }). */
  params: Record<string, string>
  /** Clé localStorage du miroir cold-start (ex: `levelup-explorer-scope:<slug>`). */
  storageKey: string
  /** App → forme plate URL (toutes les clés, `undefined` pour les valeurs vides). */
  encode: (app: App) => Url
  /** Forme plate URL (partielle) → App (remplit les défauts). */
  decode: (url: Partial<Url>) => App
  /** Clés de scope dans l'URL — sert à détecter un atterrissage « vierge » et à reset. */
  urlKeys: readonly (keyof Url)[]
}

export interface PageScopeApi<App> {
  /** État courant décodé depuis l'URL. */
  scope: App
  /** Applique un patch partiel : merge → encode → navigate(replace) + miroir. */
  setScope: (patch: Partial<App>) => void
  /** Efface tous les params de scope de l'URL et purge le miroir localStorage. */
  reset: () => void
}

function hasAnyScope(
  search: Record<string, unknown>,
  urlKeys: readonly (string | number | symbol)[],
): boolean {
  return urlKeys.some((k) => {
    const v = search[k as string]
    return v !== undefined && v !== ''
  })
}

export function usePageScope<App, Url extends object>(
  opts: UsePageScopeOptions<App, Url>,
): PageScopeApi<App> {
  const { to, params, storageKey, encode, decode, urlKeys } = opts
  const navigate = useNavigate()
  // `strict: false` : lecture du search de la route courante sans coupler le
  // hook à un id de route littéral (pattern déjà utilisé dans SessionDetailPage).
  // `as never` sur l'argument : sous le générique, l'inférence du routeur typé
  // ne défaut pas correctement (contrairement à un appel direct en composant).
  const search = useSearch({ strict: false } as never) as Partial<Url>

  const scope = decode(search)

  const writeMirror = useCallback(
    (encoded: Url) => {
      try {
        localStorage.setItem(storageKey, JSON.stringify(encoded))
      } catch {
        // fail-open : mode privé / quota dépassé / storage absent
      }
    },
    [storageKey],
  )

  const setScope = useCallback(
    (patch: Partial<App>) => {
      void navigate({
        to,
        params,
        // Updater fonctionnel : on décode l'URL LIVE (`prev`) plutôt qu'une
        // capture potentiellement périmée, on applique le patch, on ré-encode.
        // Les clés à `undefined` écrasent et retirent le param de l'URL.
        search: ((prev: Record<string, unknown>) => {
          const next = { ...decode(prev as Partial<Url>), ...patch } as App
          const encoded = encode(next)
          writeMirror(encoded)
          return { ...prev, ...encoded }
        }) as never,
        replace: true,
      } as never)
    },
    [navigate, to, params, decode, encode, writeMirror],
  )

  const reset = useCallback(() => {
    try {
      localStorage.removeItem(storageKey)
    } catch {
      // fail-open
    }
    void navigate({
      to,
      params,
      search: ((prev: Record<string, unknown>) => {
        const next = { ...prev }
        for (const k of urlKeys) delete next[k as string]
        return next
      }) as never,
      replace: true,
    } as never)
  }, [navigate, to, params, storageKey, urlKeys])

  // Cold-start : au montage uniquement. Si l'URL ne porte aucun scope mais que
  // le miroir localStorage en a un, on restaure (navigate replace). Si l'URL
  // porte déjà un scope (retour navigateur, F5, lien partagé) → l'URL gagne.
  const restoredRef = useRef(false)
  useEffect(() => {
    if (restoredRef.current) return
    restoredRef.current = true
    if (hasAnyScope(search as Record<string, unknown>, urlKeys)) return
    let saved: Partial<Url> | null = null
    try {
      const raw = localStorage.getItem(storageKey)
      if (raw) saved = JSON.parse(raw) as Partial<Url>
    } catch {
      return
    }
    if (!saved || !hasAnyScope(saved as Record<string, unknown>, urlKeys)) return
    void navigate({
      to,
      params,
      search: ((prev: Record<string, unknown>) => ({ ...prev, ...saved })) as never,
      replace: true,
    } as never)
    // Montage unique : dépendances volontairement figées (cf. ExplorerPage).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { scope, setScope, reset }
}
