/**
 * filterLink — construction de deep-links vers la page Stats Solo (timeseries)
 * avec un contexte de filtre pré-appliqué via le paramètre `?f=`.
 *
 * Source UNIQUE de l'encodage `?f=` (enveloppe v2 `{ t: titleSlug, c: ctx }`,
 * `btoa(encodeURIComponent(JSON.stringify(...)))`) : `createFilterStore.encodeToUrl`
 * consomme le même `encodeFilterContextParam` — pas de duplication de format.
 *
 * NB navigation : `?f=` n'est décodé qu'au rehydrate du store (fresh-load, cf.
 * `createFilterStore.onRehydrateStorage` → `decodeFromUrl`). Un lien construit ici
 * doit donc être suivi par une navigation PLEINE PAGE (`<a href>`), pas un `<Link>`
 * client-side — sinon le store déjà hydraté ne relit pas l'URL.
 */
import type { CascadeInput, FilterContextInput } from '@/lib/api/types'
import { DEFAULT_TITLE_SLUG } from '@/lib/staticAssets'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'

/**
 * Encode l'enveloppe v2 `{ t, c }` en chaîne base64 pour `?f=`. Point unique de
 * vérité du format (partagé avec createFilterStore).
 */
export function encodeFilterContextParam(titleSlug: string, ctx: FilterContextInput): string {
  const payload = { t: titleSlug || DEFAULT_TITLE_SLUG, c: ctx }
  return btoa(encodeURIComponent(JSON.stringify(payload)))
}

export interface SoloFilterLinkInput {
  playerSlug: string
  titleSlug: string
  /** Surcharge partielle de la cascade (modes/maps/playlists/experience_types). */
  cascade?: Partial<CascadeInput>
  /** Fenêtre temporelle bornée (dates UTC `YYYY-MM-DD`). */
  period?: { start: string; end: string }
}

/**
 * Construit le chemin `/players/{slug}/stats/timeseries?f=…` avec le contexte de
 * filtre encodé. La cascade et/ou la période fournies surchargent le contexte
 * vierge ; le reste reste par défaut (mode période, aucune session).
 */
export function buildSoloFilterLink({
  playerSlug,
  titleSlug,
  cascade,
  period,
}: SoloFilterLinkInput): string {
  const ctx: FilterContextInput = {
    // La cascade est orthogonale au couple période/session : on reste en mode
    // 'period' (une période nulle = « tout l'historique »).
    filter_mode: 'period',
    period: period
      ? { start_date: period.start, end_date: period.end }
      : { start_date: null, end_date: null },
    sessions: { picked_sessions: [], gap_minutes: DEFAULT_GAP_MINUTES },
    cascade: {
      experience_types: [],
      playlists: [],
      modes: [],
      maps: [],
      ...cascade,
    },
  }
  const f = encodeFilterContextParam(titleSlug, ctx)
  return `/players/${encodeURIComponent(playerSlug)}/stats/timeseries?f=${f}`
}

/** Fenêtre « la journée du record » (UTC) à partir d'un timestamp ISO. */
export function dayWindowUTC(iso: string): { start: string; end: string } {
  const day = new Date(iso).toISOString().slice(0, 10)
  return { start: day, end: day }
}
